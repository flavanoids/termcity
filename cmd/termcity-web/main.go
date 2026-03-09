package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"termcity/internal/data"
	"termcity/internal/history"
	"time"
)

var (
	flagPort       = flag.Int("port", 8911, "HTTP server port")
	flagForeground = flag.Bool("foreground", false, "Run in foreground mode")
)

// app holds the shared state between the polling loop and HTTP handlers.
type app struct {
	store *history.Store

	mu       sync.RWMutex
	live     []data.Incident
	warnings []string
	lastPoll time.Time

	zip  string
	lat  float64
	lng  float64
	city string
}

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: termcity-web [flags] <zipcode>\n")
		os.Exit(1)
	}
	zip := flag.Arg(0)

	log.Printf("Geocoding zip %s...", zip)
	loc, err := data.GeocodeZip(zip)
	if err != nil {
		log.Fatalf("Failed to geocode zip %s: %v", zip, err)
	}
	log.Printf("Location: %s (%.4f, %.4f)", loc.City, loc.Lat, loc.Lng)

	dbPath := historyDBPath()
	store, err := history.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to open history database: %v", err)
	}
	defer store.Close()

	a := &app{
		store: store,
		zip:   zip,
		lat:   loc.Lat,
		lng:   loc.Lng,
		city:  loc.City,
	}

	a.poll()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.pollLoop(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/api/incidents", a.handleLiveIncidents)
	mux.HandleFunc("/api/history", a.handleHistory)
	mux.HandleFunc("/api/history/clear", a.handleClearHistory)
	mux.HandleFunc("/api/status", a.handleStatus)

	addr := fmt.Sprintf(":%d", *flagPort)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		cancel()
		srv.Shutdown(context.Background())
	}()

	fmt.Printf("TermCity Web — %s (%s)\n", zip, loc.City)
	fmt.Printf("http://localhost%s\n", addr)
	fmt.Printf("Database: %s\n", dbPath)

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}

func (a *app) poll() {
	incidents, warnings := data.FetchAllIncidents(a.lat, a.lng, a.city)

	a.mu.Lock()
	a.live = incidents
	a.warnings = warnings
	a.lastPoll = time.Now()
	a.mu.Unlock()

	if len(incidents) > 0 {
		n, err := a.store.LogIncidents(incidents)
		if err != nil {
			log.Printf("Error logging incidents: %v", err)
		} else if n > 0 {
			log.Printf("Logged %d new incidents (%d total fetched)", n, len(incidents))
		}
	}

	if err := a.store.Prune(); err != nil {
		log.Printf("Error pruning history: %v", err)
	}
}

func (a *app) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.poll()
		}
	}
}

func historyDBPath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	dir := filepath.Join(cacheDir, "termcity")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "history.db")
}
