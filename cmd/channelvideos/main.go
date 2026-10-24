// channelvideos applies the user's selected channel videos to local app content.
package main

import (
	"astroapp/backend/internal/platform"
	"context"
	"log"
	"path/filepath"
	"time"
)

// main loads the featured-video selection into development storage without replacing other settings.
func main() {
	c, err := platform.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	if c.Production {
		log.Fatal("this local content command does not operate in production; use the authenticated admin editor")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	store, err := platform.OpenStore(ctx, c.MongoURI, filepath.Join(c.DataDir, "development-state.json"), c.MongoDatabase)
	if err != nil {
		log.Fatal("could not open app storage; configuration details redacted")
	}
	defer store.Close()
	if err = platform.SelectFeaturedVideos(ctx, store); err != nil {
		log.Fatal("could not save the selected videos")
	}
	log.Printf("Saved %d channel videos in view order; other content and approvals preserved", len(platform.FeaturedVideoIDs()))
}
