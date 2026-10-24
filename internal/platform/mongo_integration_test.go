package platform

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestMongoTransactionsAcrossConnections checks transaction behavior across separate connections to the explicitly configured test database.
func TestMongoTransactionsAcrossConnections(t *testing.T) {
	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("set MONGODB_TEST_URI to an isolated test replica set; never uses the application's Atlas URI")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if os.Getenv("MONGODB_TEST_INIT") == "true" {
		if !strings.HasPrefix(uri, "mongodb://127.0.0.1:27028/") {
			t.Fatal("test replica-set initialization only allowed on dedicated localhost port 27028")
		}
		client, e := mongo.Connect(options.Client().ApplyURI("mongodb://127.0.0.1:27028/?directConnection=true").SetServerSelectionTimeout(10 * time.Second))
		if e != nil {
			t.Fatal("test Mongo client failed")
		}
		var hello bson.M
		_ = client.Database("admin").RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&hello)
		if hello["setName"] == nil {
			e = client.Database("admin").RunCommand(ctx, bson.D{{Key: "replSetInitiate", Value: bson.M{"_id": "m4ytest", "members": bson.A{bson.M{"_id": 0, "host": "127.0.0.1:27028"}}}}}).Err()
			if e != nil {
				t.Fatal("could not initiate isolated test replica set")
			}
		}
		_ = client.Disconnect(ctx)
	}
	name := "m4y_test_" + strings.TrimPrefix(id("db"), "db_")
	first, e := OpenStore(ctx, uri, "", name)
	if e != nil {
		t.Fatal(e)
	}
	defer first.Close()
	defer first.db.Drop(context.Background())
	second, e := OpenStore(ctx, uri, "", name)
	if e != nil {
		t.Fatal(e)
	}
	defer second.Close()
	if e = first.Update(ctx, func(s *State) error {
		s.Users["fixture-user"] = &User{ID: "fixture-user", Phone: "+919876543210", Created: time.Now().UTC()}
		s.Orders["snapshot-fixture"] = &Order{ID: "snapshot-fixture", Snapshot: &Report{Birth: Birth{Name: "Immutable fixture"}}}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	_ = second.View(ctx, func(s *State) error {
		if s.Users["fixture-user"] == nil {
			t.Fatal("BSON state did not round-trip across connections")
		}
		return nil
	})
	if e = second.ViewKinds(ctx, []string{"users", "settings"}, func(s *State) error {
		if s.Users["fixture-user"] == nil || len(s.Reports) != 0 || len(s.Orders) != 0 {
			t.Fatal("partial Mongo snapshot omitted a requested kind or included unrelated records")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if e = second.ViewWithoutOrderSnapshots(ctx, []string{"orders"}, func(s *State) error {
		if s.Orders["snapshot-fixture"] == nil || s.Orders["snapshot-fixture"].Snapshot != nil {
			t.Fatal("Mongo order summary retained the report payload")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	// A blocked writer in this process must not stall independent Mongo reads.
	first.mu.Lock()
	read := make(chan error, 1)
	go func() {
		if err := first.Ping(ctx); err != nil {
			read <- err
			return
		}
		if err := first.ViewKinds(ctx, []string{"settings"}, func(*State) error { return nil }); err != nil {
			read <- err
			return
		}
		read <- first.View(ctx, func(*State) error { return nil })
	}()
	select {
	case err := <-read:
		first.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		first.mu.Unlock()
		t.Fatal("Mongo reads waited for the application write mutex")
	}
	config := Config{PaymentMode: "simulation", ZoomMode: "simulation", OTPMode: "simulation"}
	apps := []*App{NewApp(first, config), NewApp(second, config)}
	now := time.Now().UTC()
	for _, a := range apps {
		a.Now = func() time.Time { return now }
	}
	var slot string
	_ = first.View(ctx, func(s *State) error { slot = slots(s, now, false)[0].Format(time.RFC3339); return nil })
	var wg sync.WaitGroup
	var successes atomic.Int32
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, e := apps[i%2].Checkout(ctx, "fixture-user", CheckoutRequest{Product: "consultation", Slot: slot, Key: fmt.Sprintf("mongo-concurrent-%d", i)})
			if e == nil {
				successes.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("transactional cross-connection double booking: successes=%d", successes.Load())
	}
	if e = first.Update(ctx, func(s *State) error {
		s.Users["must-rollback"] = &User{ID: "must-rollback"}
		return fmt.Errorf("intentional rollback")
	}); e == nil {
		t.Fatal("transaction did not abort")
	}
	_ = second.View(ctx, func(s *State) error {
		if s.Orders["snapshot-fixture"].Snapshot == nil || s.Orders["snapshot-fixture"].Snapshot.Birth.Name != "Immutable fixture" {
			t.Fatal("a summary read changed a purchased report snapshot")
		}
		if s.Users["must-rollback"] != nil {
			t.Fatal("aborted write persisted")
		}
		if len(s.Bookings) != 1 {
			t.Fatalf("wanted one booking, got %d", len(s.Bookings))
		}
		return nil
	})
}
