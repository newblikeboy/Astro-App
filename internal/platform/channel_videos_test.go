package platform

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

// TestFeaturedVideoCatalogAndSelection verifies the bundled video catalogue and the chosen featured selection.
func TestFeaturedVideoCatalogAndSelection(t *testing.T) {
	catalog := FeaturedVideoCatalog()
	if catalog.ChannelID != "UC9-N4M0ZcIVefwMx4_Ep7TA" || len(catalog.Videos) != 10 || catalog.CheckedAt.IsZero() {
		t.Fatal("channel selection is incomplete")
	}
	ids := FeaturedVideoIDs()
	for i, v := range catalog.Videos {
		if !videoID.MatchString(v.ID) || v.Title == "" || v.EnglishTitle == "" || v.Views <= 0 || v.Duration == "" || v.PublishedAt.IsZero() {
			t.Fatalf("incomplete video: %+v", v)
		}
		if i > 0 && v.Views > catalog.Videos[i-1].Views {
			t.Fatal("videos are not ordered by views")
		}
	}
	catalog.Videos[0].Title = "modified copy"
	if FeaturedVideoCatalog().Videos[0].Title == "modified copy" {
		t.Fatal("catalog mutated by caller")
	}
	a := testApp(t)
	if err := a.Store.Update(context.Background(), func(s *State) error {
		s.Settings.Content.Videos = []string{}
		s.Settings.Content.Intro = "Keep this introduction"
		s.Settings.PDFPrice = 12300
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := SelectFeaturedVideos(context.Background(), a.Store); err != nil {
		t.Fatal(err)
	}
	if err := a.Store.View(context.Background(), func(s *State) error {
		if !reflect.DeepEqual(s.Settings.Content.Videos, ids) || s.Settings.Content.Intro != "Keep this introduction" || s.Settings.PDFPrice != 12300 || s.Settings.Content.Approved {
			t.Fatal("selection changed unrelated settings or approvals")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	response := request(t, a, "GET", "/api/config", nil, nil, "")
	var config struct {
		VideoCatalog VideoCatalog `json:"video_catalog"`
		Settings     Settings     `json:"settings"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &config); err != nil || response.Code != 200 || len(config.VideoCatalog.Videos) != 10 || !reflect.DeepEqual(config.Settings.Content.Videos, ids) {
		t.Fatal("public configuration lost channel metadata")
	}
}

// TestVideoSelectionLimits rejects featured-video selections outside the supported limits.
func TestVideoSelectionLimits(t *testing.T) {
	s := seed().Settings
	if err := validateSettings(s); err != nil {
		t.Fatal("ten valid videos rejected:", err)
	}
	s.Content.Videos = append(FeaturedVideoIDs(), "abcdefghijk")
	if validateSettings(s) == nil {
		t.Fatal("eleven videos accepted")
	}
	s.Content.Videos = []string{"lU25JP299BI", "lU25JP299BI"}
	if validateSettings(s) == nil {
		t.Fatal("duplicate videos accepted")
	}
	s.Content.Videos = []string{"https://www.youtube.com/watch?v=lU25JP299BI"}
	if validateSettings(s) == nil {
		t.Fatal("URL accepted as a video ID")
	}
}
