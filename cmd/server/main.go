package main

import (
	"astroapp/backend/internal/platform"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// main loads configuration, opens storage, starts the HTTP server and worker, and handles shutdown.
func main() {
	c, e := platform.LoadConfig()
	if e != nil {
		log.Fatal(e)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if c.MongoURI == "" {
			log.Fatal("MONGODB_URI required")
		}
		if e = platform.Migrate(ctx, c.MongoURI, c.MongoDatabase); e != nil {
			log.Fatal("migration failed; inspect database connectivity")
		}
		log.Print("migrations applied")
		return
	}
	s, e := platform.OpenStore(ctx, c.MongoURI, filepath.Join(c.DataDir, "development-state.json"), c.MongoDatabase)
	if e != nil {
		log.Fatal("database could not be opened; inspect configuration")
	}
	defer s.Close()
	app := platform.NewApp(s, c)
	go app.RunWorker(ctx)
	server := &http.Server{Addr: c.Addr, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 90 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 16}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	log.Printf("Mishra 4 You listening on http://%s (production=%v, payments=%s)", c.Addr, c.Production, c.PaymentMode)
	if e = server.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		log.Fatal("HTTP server failed")
	}
}
