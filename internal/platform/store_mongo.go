package platform

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed migrations/*.json
var migrations embed.FS

type Store struct {
	mu     sync.Mutex
	path   string
	client *mongo.Client
	db     *mongo.Database
	state  *State
}
type record struct {
	ID   string   `bson:"_id"`
	Kind string   `bson:"kind"`
	Key  string   `bson:"key"`
	Data bson.Raw `bson:"data"`
}

// OpenStore uses Atlas transactions when a URI is provided. The local file store
// is development-only and cannot be selected by production configuration.
func OpenStore(ctx context.Context, uri, path string, database ...string) (*Store, error) {
	s := &Store{path: path, state: seed()}
	if uri != "" {
		if !strings.HasPrefix(uri, "mongodb://") && !strings.HasPrefix(uri, "mongodb+srv://") {
			return nil, fmt.Errorf("MONGODB_URI must use mongodb:// or mongodb+srv://")
		}
		c, e := mongo.Connect(options.Client().ApplyURI(uri).SetServerSelectionTimeout(10 * time.Second).SetMaxPoolSize(12))
		if e != nil {
			return nil, fmt.Errorf("MongoDB connection configuration is invalid (URI redacted)")
		}
		s.client = c
		startup, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		if e = c.Ping(startup, nil); e != nil {
			c.Disconnect(context.Background())
			return nil, fmt.Errorf("MongoDB connection failed; check Atlas network access and database user (URI redacted)")
		}
		var hello bson.M
		if e = c.Database("admin").RunCommand(startup, bson.D{{Key: "hello", Value: 1}}).Decode(&hello); e != nil {
			c.Disconnect(context.Background())
			return nil, fmt.Errorf("MongoDB topology check failed")
		}
		if hello["setName"] == nil && hello["msg"] != "isdbgrid" {
			c.Disconnect(context.Background())
			return nil, fmt.Errorf("MongoDB replica set required for booking/payment transactions; Atlas is supported")
		}
		name := "mishra4you_development"
		if len(database) > 0 && database[0] != "" {
			name = database[0]
		}
		if strings.ContainsAny(name, " /\\.\"$\x00") {
			c.Disconnect(context.Background())
			return nil, fmt.Errorf("invalid MONGODB_DATABASE")
		}
		s.db = c.Database(name)
		if e = s.migrate(startup); e != nil {
			s.Close()
			return nil, fmt.Errorf("MongoDB migrations failed; inspect database permissions")
		}
		if e = s.Update(startup, func(*State) error { return nil }); e != nil {
			s.Close()
			return nil, fmt.Errorf("MongoDB transaction setup failed")
		}
		return s, nil
	}
	if path != "" {
		b, e := os.ReadFile(path)
		if e == nil {
			if e = json.Unmarshal(b, s.state); e != nil {
				return nil, fmt.Errorf("development store is invalid")
			}
		} else if !errors.Is(e, os.ErrNotExist) {
			return nil, e
		}
	}
	return s, nil
}

// Close disconnects the MongoDB client with a bounded shutdown timeout.
func (s *Store) Close() {
	if s.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.client.Disconnect(ctx)
	}
}

// migrate creates declared indexes and migration records, then initializes the transaction revision guard.
func (s *Store) migrate(ctx context.Context) error {
	entries, e := migrations.ReadDir("migrations")
	if e != nil {
		return e
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		var migration struct {
			Version int `json:"version"`
			Indexes []struct {
				Collection string   `json:"collection"`
				Name       string   `json:"name"`
				Fields     []string `json:"fields"`
				Unique     bool     `json:"unique"`
			} `json:"indexes"`
		}
		b, e := migrations.ReadFile("migrations/" + entry.Name())
		if e != nil {
			return e
		}
		if e = json.Unmarshal(b, &migration); e != nil {
			return e
		}
		for _, index := range migration.Indexes {
			keys := bson.D{}
			for _, field := range index.Fields {
				keys = append(keys, bson.E{Key: field, Value: 1})
			}
			_, e = s.db.Collection(index.Collection).Indexes().CreateOne(ctx, mongo.IndexModel{Keys: keys, Options: options.Index().SetName(index.Name).SetUnique(index.Unique)})
			if e != nil {
				return e
			}
		}
		_, e = s.db.Collection("schema_migrations").UpdateOne(ctx, bson.M{"_id": migration.Version}, bson.M{"$setOnInsert": bson.M{"applied_at": time.Now().UTC(), "file": entry.Name()}}, options.UpdateOne().SetUpsert(true))
		if e != nil {
			return e
		}
	}
	_, e = s.db.Collection("transaction_guard").UpdateOne(ctx, bson.M{"_id": "state"}, bson.M{"$setOnInsert": bson.M{"revision": int64(0)}}, options.UpdateOne().SetUpsert(true))
	return e
}

// Each entity is a separate BSON document, avoiding a growing 16MB state document.
// The shared revision write serializes cross-entity invariants inside a MongoDB
// transaction. This conservative implementation targets a small business.
func encodeState(s *State) (map[string]record, error) {
	b, e := json.Marshal(s)
	if e != nil {
		return nil, e
	}
	var fields map[string]json.RawMessage
	if e = json.Unmarshal(b, &fields); e != nil {
		return nil, e
	}
	out := map[string]record{}
	// add wraps one state value as a BSON record under a stable entity kind/key.
	add := func(kind, key string, value json.RawMessage) error {
		var raw bson.Raw
		if e := bson.UnmarshalExtJSON(append(append([]byte(`{"value":`), value...), '}'), false, &raw); e != nil {
			return e
		}
		rid := kind + ":" + key
		out[rid] = record{rid, kind, key, raw}
		return nil
	}
	for kind, raw := range fields {
		if kind == "settings" {
			if e = add(kind, "root", raw); e != nil {
				return nil, e
			}
			continue
		}
		if kind == "events" || kind == "audits" {
			var items []json.RawMessage
			if e = json.Unmarshal(raw, &items); e != nil {
				return nil, e
			}
			for i, item := range items {
				if e = add(kind, fmt.Sprintf("%012d", i), item); e != nil {
					return nil, e
				}
			}
			continue
		}
		var items map[string]json.RawMessage
		if e = json.Unmarshal(raw, &items); e != nil {
			return nil, e
		}
		for key, item := range items {
			if e = add(kind, key, item); e != nil {
				return nil, e
			}
		}
	}
	return out, nil
}

// readMongo reconstructs application state and its document snapshots from MongoDB records.
func (s *Store) readMongo(ctx context.Context) (*State, map[string]record, error) {
	return s.readMongoRecords(ctx, bson.D{})
}

// readMongoRecords reconstructs a snapshot from a filtered set of entity records;
// omitted kinds retain seed defaults and must not be used for transaction writes.
func (s *Store) readMongoRecords(ctx context.Context, filter bson.D, projection ...bson.D) (*State, map[string]record, error) {
	// Avoid a small first batch followed by repeated network round trips when
	// reconstructing a full snapshot; MongoDB retains its byte limit per batch.
	find := options.Find().SetBatchSize(1000)
	if len(projection) > 0 && len(projection[0]) > 0 {
		find.SetProjection(projection[0])
	}
	cursor, e := s.db.Collection("records").Find(ctx, filter, find)
	if e != nil {
		return nil, nil, e
	}
	defer cursor.Close(ctx)
	records := map[string]record{}
	groups := map[string]map[string]json.RawMessage{}
	for cursor.Next(ctx) {
		var r record
		if e = cursor.Decode(&r); e != nil {
			return nil, nil, e
		}
		records[r.ID] = r
		b, e := bson.MarshalExtJSON(r.Data, false, false)
		if e != nil {
			return nil, nil, e
		}
		var wrapped struct {
			Value json.RawMessage `json:"value"`
		}
		if e = json.Unmarshal(b, &wrapped); e != nil {
			return nil, nil, e
		}
		if groups[r.Kind] == nil {
			groups[r.Kind] = map[string]json.RawMessage{}
		}
		groups[r.Kind][r.Key] = wrapped.Value
	}
	if e = cursor.Err(); e != nil {
		return nil, nil, e
	}
	fields := map[string]json.RawMessage{}
	for kind, items := range groups {
		if kind == "settings" {
			fields[kind] = items["root"]
		} else if kind == "events" || kind == "audits" {
			keys := []string{}
			for key := range items {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			values := []json.RawMessage{}
			for _, key := range keys {
				values = append(values, items[key])
			}
			fields[kind], e = json.Marshal(values)
		} else {
			fields[kind], e = json.Marshal(items)
		}
		if e != nil {
			return nil, nil, e
		}
	}
	b, e := json.Marshal(fields)
	if e != nil {
		return nil, nil, e
	}
	state := seed()
	if e = json.Unmarshal(b, state); e != nil {
		return nil, nil, e
	}
	return state, records, nil
}

// viewMongo executes an independent read-only snapshot transaction. MongoDB
// snapshot isolation provides consistency without blocking unrelated readers
// behind the process-local write mutex or a slow response callback.
func (s *Store) viewMongo(ctx context.Context, filter, projection bson.D, f func(*State) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	session, e := s.client.StartSession()
	if e != nil {
		return e
	}
	defer session.EndSession(ctx)
	var snapshot *State
	_, e = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		v, _, err := s.readMongoRecords(tx, filter, projection)
		snapshot = v
		return nil, err
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()))
	if e != nil {
		return fmt.Errorf("MongoDB read unavailable")
	}
	return f(snapshot)
}

// View runs a callback against an independent database snapshot or a cloned
// local-development state. Database reads do not acquire the local write mutex.
func (s *Store) View(ctx context.Context, f func(*State) error) error {
	if s.client != nil {
		return s.viewMongo(ctx, bson.D{}, nil, f)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	b, _ := json.Marshal(s.state)
	v := seed()
	if e := json.Unmarshal(b, v); e != nil {
		return e
	}
	return f(v)
}

// stateSubset selects explicitly named entity kinds without serialising unrelated
// reports or private records. Callers must hold the local store mutex while using
// its returned references. Unknown or empty selections fail instead of reading all.
func stateSubset(s *State, kinds []string) (map[string]any, error) {
	all := map[string]any{
		"video_links": s.VideoLinks,
		"settings":    s.Settings, "gemstones": s.Gemstones, "users": s.Users,
		"sessions": s.Sessions, "otps": s.OTPs, "reports": s.Reports,
		"orders": s.Orders, "bookings": s.Bookings, "meetings": s.Meetings,
		"rules": s.Rules, "jobs": s.Jobs, "webhooks": s.Webhooks,
		"expenses": s.Expenses, "distributions": s.Distributions,
		"reconciliations": s.Reconciliations, "events": s.Events, "audits": s.Audits,
	}
	if len(kinds) == 0 {
		return nil, fmt.Errorf("choose entity kinds for a partial read")
	}
	selected := make(map[string]any, len(kinds))
	for _, kind := range kinds {
		value, ok := all[kind]
		if !ok {
			return nil, fmt.Errorf("unsupported entity kind")
		}
		selected[kind] = value
	}
	return selected, nil
}

// ViewKinds reads only requested entity kinds in one consistent snapshot. Other
// fields contain defaults, so callers must name every kind their callback needs.
// This is read-only and never supplies a partial snapshot to Update.
func (s *Store) ViewKinds(ctx context.Context, kinds []string, f func(*State) error) error {
	return s.viewKinds(ctx, kinds, false, f)
}

// ViewWithoutOrderSnapshots reads selected kinds without the purchased report
// snapshot embedded in orders. Catalog availability and worker preflight need
// order status, holds and line quantities, never immutable report payloads.
// Returned orders are incomplete read-only summaries and must never be saved.
func (s *Store) ViewWithoutOrderSnapshots(ctx context.Context, kinds []string, f func(*State) error) error {
	return s.viewKinds(ctx, kinds, true, f)
}

// viewKinds applies the same kind selection and optional order-snapshot omission
// to MongoDB projections and cloned local state; stored orders are never mutated.
func (s *Store) viewKinds(ctx context.Context, kinds []string, omitOrderSnapshots bool, f func(*State) error) error {
	if s.client != nil {
		if _, err := stateSubset(&State{}, kinds); err != nil {
			return err
		}
		var projection bson.D
		if omitOrderSnapshots {
			projection = bson.D{{Key: "data.value.snapshot", Value: 0}}
		}
		return s.viewMongo(ctx, bson.D{{Key: "kind", Value: bson.D{{Key: "$in", Value: kinds}}}}, projection, f)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	selected, err := stateSubset(s.state, kinds)
	if err != nil {
		return err
	}
	if omitOrderSnapshots {
		if orders, ok := selected["orders"].(map[string]*Order); ok {
			summaries := make(map[string]*Order, len(orders))
			for key, order := range orders {
				if order == nil {
					summaries[key] = nil
					continue
				}
				copy := *order
				copy.Snapshot = nil
				summaries[key] = &copy
			}
			selected["orders"] = summaries
		}
	}
	data, err := json.Marshal(selected)
	if err != nil {
		return err
	}
	snapshot := seed()
	if err := json.Unmarshal(data, snapshot); err != nil {
		return err
	}
	return f(snapshot)
}

// Ping checks database connectivity without loading business records or waiting
// behind application transactions. Local development uses its open in-memory store.
func (s *Store) Ping(ctx context.Context) error {
	if s.client != nil {
		return s.client.Ping(ctx, nil)
	}
	return ctx.Err()
}

// Update commits state changes atomically, preserving cross-record invariants and local rollback on failure.
func (s *Store) Update(ctx context.Context, f func(*State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil {
		session, e := s.client.StartSession()
		if e != nil {
			return e
		}
		defer session.EndSession(ctx)
		_, e = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
			if _, e := s.db.Collection("transaction_guard").UpdateOne(tx, bson.M{"_id": "state"}, bson.M{"$inc": bson.M{"revision": int64(1)}}); e != nil {
				return nil, e
			}
			v, before, e := s.readMongo(tx)
			if e != nil {
				return nil, e
			}
			if e = f(v); e != nil {
				return nil, e
			}
			after, e := encodeState(v)
			if e != nil {
				return nil, e
			}
			models := []mongo.WriteModel{}
			for key, r := range after {
				if previous, ok := before[key]; !ok || !bytes.Equal(previous.Data, r.Data) {
					models = append(models, mongo.NewReplaceOneModel().SetFilter(bson.M{"_id": key}).SetReplacement(r).SetUpsert(true))
				}
			}
			for key := range before {
				if _, ok := after[key]; !ok {
					models = append(models, mongo.NewDeleteOneModel().SetFilter(bson.M{"_id": key}))
				}
			}
			if len(models) > 0 {
				if _, e = s.db.Collection("records").BulkWrite(tx, models); e != nil {
					return nil, e
				}
			}
			return nil, nil
		}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
		return e
	}
	b, _ := json.Marshal(s.state)
	v := seed()
	if e := json.Unmarshal(b, v); e != nil {
		return e
	}
	if e := f(v); e != nil {
		return e
	}
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	if s.path != "" {
		if e = os.MkdirAll(filepath.Dir(s.path), 0700); e != nil {
			return e
		}
		tmp := s.path + ".tmp"
		if e = os.WriteFile(tmp, b, 0600); e != nil {
			return e
		}
		if e = os.Rename(tmp, s.path); e != nil {
			return e
		}
	}
	s.state = v
	return nil
}
