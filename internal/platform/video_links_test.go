package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

// TestVideoLinkPublication checks administrator/CSRF boundaries, drafts, optimistic updates, deactivation and private-data exclusion.
func TestVideoLinkPublication(t *testing.T) {
	a := testApp(t)
	admin, csrf := authorize(t, a, "admin", "admin")
	putUser(t, a, "customer")
	customer, customerCSRF := authorize(t, a, "customer", "customer")
	link := VideoLink{Slug: "karj-rahat", Title: "कर्ज़ से जुड़ा मार्गदर्शन", Topic: "debt", SignBasis: "moon", RuleIDs: []string{}}
	for _, path := range []string{"/api/admin/video-links", "/api/admin/video-links/karj-rahat"} {
		method := "GET"
		if strings.HasSuffix(path, "karj-rahat") {
			method = "PUT"
		}
		if w := request(t, a, method, path, link, nil, ""); w.Code != 401 {
			t.Fatalf("anonymous admin access: %d", w.Code)
		}
		if w := request(t, a, method, path, link, customer, customerCSRF); w.Code != 401 {
			t.Fatalf("customer admin access: %d", w.Code)
		}
	}
	if w := request(t, a, "PUT", "/api/admin/video-links/karj-rahat", link, admin, "wrong"); w.Code != 403 {
		t.Fatal("missing CSRF accepted")
	}
	if w := request(t, a, "PUT", "/api/admin/video-links/karj-rahat", link, admin, csrf); w.Code != 200 {
		t.Fatalf("draft save: %d %s", w.Code, w.Body.String())
	}
	if w := request(t, a, "GET", "/api/video-links/karj-rahat", nil, nil, ""); w.Code != 404 {
		t.Fatal("draft exposed")
	}
	if w := request(t, a, "GET", "/api/config", nil, nil, ""); strings.Contains(w.Body.String(), "karj-rahat") {
		t.Fatal("draft in public configuration")
	}
	if w := request(t, a, "PUT", "/api/admin/video-links/karj-rahat", link, admin, csrf); w.Code != 409 {
		t.Fatal("duplicate slug replaced existing link")
	}
	link.Version = 1
	link.Enabled = true
	w := request(t, a, "PUT", "/api/admin/video-links/karj-rahat", link, admin, csrf)
	if w.Code != 200 {
		t.Fatalf("publish failed: %s", w.Body.String())
	}
	var saved VideoLink
	if err := json.Unmarshal(w.Body.Bytes(), &saved); err != nil || saved.Version != 2 || saved.Created.IsZero() || saved.Updated.IsZero() {
		t.Fatalf("bad persisted metadata: %s", w.Body.String())
	}
	// Unrelated private data must never be traversed to resolve a public video.
	a.Store.state.OTPs["private"] = &OTP{Phone: "private-marker"}
	public := request(t, a, "GET", "/api/video-links/karj-rahat", nil, nil, "")
	if public.Code != 200 || strings.Contains(public.Body.String(), "private-marker") || public.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("public result: %d %s", public.Code, public.Body.String())
	}
	if w := request(t, a, "PUT", "/api/admin/video-links/karj-rahat", link, admin, csrf); w.Code != 409 {
		t.Fatal("stale editor overwrote published link")
	}
	saved.Enabled = false
	if w := request(t, a, "PUT", "/api/admin/video-links/karj-rahat", saved, admin, csrf); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := request(t, a, "GET", "/api/video-links/karj-rahat", nil, nil, ""); w.Code != 404 {
		t.Fatal("disabled public link active")
	}
	if w := request(t, a, "GET", "/api/admin/video-links", nil, admin, ""); w.Code != 200 || !strings.Contains(w.Body.String(), "karj-rahat") {
		t.Fatal("admin cannot recover inactive link")
	}
	if len(a.Store.state.Audits) < 3 {
		t.Fatal("link changes not audited")
	}
}

// TestVideoLinkValidationAndReadingApproval rejects unsafe input and prevents retired, demo or draft interpretations from becoming campaign choices.
func TestVideoLinkValidationAndReadingApproval(t *testing.T) {
	base := VideoLink{Slug: "mesh-rashi", Title: "Mesh rashi video", Topic: "zodiac", Sign: 1, SignBasis: "moon"}
	changes := []func(*VideoLink){
		func(v *VideoLink) { v.Slug = "../admin" }, func(v *VideoLink) { v.Slug = "UPPER" },
		func(v *VideoLink) { v.Slug = strings.Repeat("a", 65) }, func(v *VideoLink) { v.Title = "  " },
		func(v *VideoLink) { v.Sign = 0 }, func(v *VideoLink) { v.SignBasis = "guess" },
		func(v *VideoLink) { v.Topic = "debt" }, func(v *VideoLink) { v.Version = -1 },
		func(v *VideoLink) { v.RuleIDs = []string{"same", "same"} },
	}
	for i, change := range changes {
		v := base
		change(&v)
		if err := validateVideoLink(v); err == nil {
			t.Fatalf("invalid variant %d accepted", i)
		}
	}
	for _, basis := range []string{"moon", "lagna", "sun"} {
		v := base
		v.SignBasis = basis
		if err := validateVideoLink(v); err != nil {
			t.Fatal(err)
		}
	}
	a := testApp(t)
	admin, csrf := authorize(t, a, "admin", "admin")
	for _, removed := range []string{"english_title", "intro", "english_intro", "video_url"} {
		payload := map[string]any{"slug": "mesh-rashi", "title": "Mesh rashi video", "topic": "zodiac", "sign": 1, "sign_basis": "moon", removed: "old optional value"}
		if w := request(t, a, "PUT", "/api/admin/video-links/mesh-rashi", payload, admin, csrf); w.Code != 400 {
			t.Fatalf("removed field %s still accepted: %d", removed, w.Code)
		}
	}
	now := a.Now()
	a.Store.state.Rules = map[string]Rule{
		"debt:1":     {ID: "debt", Version: 1, Topic: "Debt", Status: "approved", Approver: "Guruji", ApprovedAt: &now},
		"debt:2":     {ID: "debt", Version: 2, Topic: "Next draft", Status: "draft"},
		"old:1":      {ID: "old", Version: 1, Status: "approved", Approver: "Guruji", ApprovedAt: &now},
		"old:2":      {ID: "old", Version: 2, Status: "retired"},
		"demo:1":     {ID: "demo", Version: 1, Status: "approved", Demo: true, Approver: "Guruji", ApprovedAt: &now},
		"unsigned:1": {ID: "unsigned", Version: 1, Status: "approved"},
	}
	choices := latestVideoRules(a.Store.state.Rules)
	if len(choices) != 1 || choices[0].ID != "debt" || choices[0].Version != 1 {
		t.Fatalf("bad eligible choices: %+v", choices)
	}
	for _, id := range []string{"old", "demo", "unsigned", "missing"} {
		v := base
		v.Enabled = true
		v.RuleIDs = []string{id}
		if w := request(t, a, "PUT", "/api/admin/video-links/mesh-rashi", v, admin, csrf); w.Code != 422 {
			t.Fatalf("unapproved choice accepted %s: %d", id, w.Code)
		}
	}
	v := base
	v.Enabled = true
	v.RuleIDs = []string{"debt"}
	if w := request(t, a, "PUT", "/api/admin/video-links/mesh-rashi", v, admin, csrf); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := request(t, a, "PUT", "/api/admin/video-links/different", v, admin, csrf); w.Code != 422 {
		t.Fatal("slug mismatch accepted")
	}
	// Only activated records are public; invalid public identifiers are normal misses.
	for _, path := range []string{"/api/video-links/xx", "/api/video-links/UNKNOWN", "/api/video-links/not-created"} {
		if w := request(t, a, http.MethodGet, path, nil, nil, ""); w.Code != 404 {
			t.Fatalf("accepted %s", path)
		}
	}
}

// TestVideoLinkPersistence ensures the additive record kind round-trips through local storage, selective reads and MongoDB record encoding.
func TestVideoLinkPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	store, err := OpenStore(context.Background(), "", path)
	if err != nil {
		t.Fatal(err)
	}
	link := VideoLink{Slug: "career-video", Title: "Career video", Topic: "promotion", SignBasis: "moon", Version: 1}
	if err := store.Update(context.Background(), func(s *State) error { s.VideoLinks[link.Slug] = link; return nil }); err != nil {
		t.Fatal(err)
	}
	records, err := encodeState(store.state)
	if err != nil || records["video_links:career-video"].Kind != "video_links" {
		t.Fatal("video link not encoded as an independent record", err)
	}
	reopened, err := OpenStore(context.Background(), "", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.ViewKinds(context.Background(), []string{"video_links"}, func(s *State) error {
		if s.VideoLinks[link.Slug].Title != link.Title {
			return fmt.Errorf("video link lost on reopen")
		}
		s.VideoLinks[link.Slug] = VideoLink{}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if reopened.state.VideoLinks[link.Slug].Title != link.Title {
		t.Fatal("partial reader mutated stored link")
	}
}
