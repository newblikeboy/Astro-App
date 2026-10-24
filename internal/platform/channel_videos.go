package platform

import (
	"context"
	_ "embed"
	"encoding/json"
	"slices"
	"time"
)

type ChannelVideo struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	EnglishTitle string    `json:"english_title"`
	Views        int64     `json:"views"`
	Duration     string    `json:"duration"`
	PublishedAt  time.Time `json:"published_at"`
}
type VideoCatalog struct {
	ChannelID   string         `json:"channel_id"`
	ChannelName string         `json:"channel_name"`
	ChannelURL  string         `json:"channel_url"`
	SourceURL   string         `json:"source_url"`
	CheckedAt   time.Time      `json:"checked_at"`
	Selection   string         `json:"selection"`
	Videos      []ChannelVideo `json:"videos"`
}

//go:embed data/channel-videos.json
var channelVideoData []byte
var selectedChannelVideos = func() VideoCatalog {
	var catalog VideoCatalog
	if err := json.Unmarshal(channelVideoData, &catalog); err != nil {
		panic("invalid bundled channel video catalog")
	}
	return catalog
}()

// FeaturedVideoCatalog returns the bundled, dated catalogue of the channel's selected popular videos.
func FeaturedVideoCatalog() VideoCatalog {
	catalog := selectedChannelVideos
	catalog.Videos = slices.Clone(catalog.Videos)
	return catalog
}

// FeaturedVideoIDs returns the featured catalogue's YouTube IDs in display order.
func FeaturedVideoIDs() []string {
	ids := make([]string, len(selectedChannelVideos.Videos))
	for i, video := range selectedChannelVideos.Videos {
		ids[i] = video.ID
	}
	return ids
}

// SelectFeaturedVideos changes only the curated video IDs. Content/policy
// approval, prices, availability and all customer state are preserved.
func SelectFeaturedVideos(ctx context.Context, store *Store) error {
	return store.Update(ctx, func(s *State) error {
		ids := FeaturedVideoIDs()
		if slices.Equal(s.Settings.Content.Videos, ids) {
			return nil
		}
		s.Settings.Content.Videos = ids
		audit(s, "local-operator", "content.channel-videos-selected", selectedChannelVideos.ChannelID)
		return nil
	})
}
