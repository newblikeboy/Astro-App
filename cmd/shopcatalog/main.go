// shopcatalog adds editable reference-priced sample gemstone listings in development.
package main

import (
	"astroapp/backend/internal/platform"
	"context"
	"log"
	"path/filepath"
	"time"
)

// main loads development storage and adds missing sample products without changing merchant settings.
func main() {
	c, err := platform.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	if c.Production {
		log.Fatal("use the authenticated administrator editor in production")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	store, err := platform.OpenStore(ctx, c.MongoURI, filepath.Join(c.DataDir, "development-state.json"), c.MongoDatabase)
	if err != nil {
		log.Fatal("could not open app storage; configuration details redacted")
	}
	defer store.Close()
	count, err := platform.SeedGemstoneCatalog(ctx, store)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Added %d editable sample products; merchant settings and existing products preserved", count)
}
