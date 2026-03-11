package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"termcity/internal/data"
	"termcity/internal/history"
	"time"
)

var (
	flagPort       = flag.Int("port", 8911, "HTTP server port")
	flagForeground = flag.Bool("foreground", false, "Run in foreground (daemon) mode")
	flagLat        = flag.Float64("lat", 0, "Pre-geocoded latitude (internal, set by launcher)")
	flagLng        = flag.Float64("lng", 0, "Pre-geocoded longitude (internal, set by launcher)")
	flagCity       = flag.String("city", "", "Pre-geocoded city name (internal, set by launcher)")
)

// app holds shared state between the polling loop and HTTP handlers.
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

	if *flagForeground {
		runDaemon(zip)
	} else {
		runLauncher(zip)
	}
}

// ── Launcher (user-facing) ────────────────────────────────────────────────────

func runLauncher(zip string) {
	dbPath := historyDBPath()
	logPath := logFilePath()

	fmt.Println()

	// Step 1: Geocode
	spin := startSpinner(1, 3, "Geocoding "+zip)
	loc, err := data.GeocodeZip(zip)
	spin.stop()
	if err != nil {
		stepFail(1, 3, "Geocoding "+zip)
		fmt.Fprintf(os.Stderr, "  error: %v\n\n", err)
		os.Exit(1)
	}
	stepDone(1, 3, "Geocoding "+zip, loc.City+", "+loc.State)

	// Step 2: Fork background daemon
	spin = startSpinner(2, 3, "Starting background server")

	readyR, readyW, err := os.Pipe()
	if err != nil {
		spin.stop()
		stepFail(2, 3, "Starting background server")
		fmt.Fprintf(os.Stderr, "  error: pipe: %v\n\n", err)
		os.Exit(1)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logFile = os.NewFile(uintptr(syscall.Stderr), "stderr")
	}

	self, _ := os.Executable()
	args := []string{
		"-foreground",
		fmt.Sprintf("-port=%d", *flagPort),
		fmt.Sprintf("-lat=%f", loc.Lat),
		fmt.Sprintf("-lng=%f", loc.Lng),
		fmt.Sprintf("-city=%s", loc.City),
		zip,
	}
	cmd := exec.Command(self, args...)
	cmd.Env = append(os.Environ(), "TERMCITY_READY_FD=3")
	cmd.ExtraFiles = []*os.File{readyW} // becomes FD 3 in child
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // detach from terminal

	if err := cmd.Start(); err != nil {
		spin.stop()
		stepFail(2, 3, "Starting background server")
		fmt.Fprintf(os.Stderr, "  error: %v\n\n", err)
		os.Exit(1)
	}
	readyW.Close()
	spin.stop()
	stepDone(2, 3, "Starting background server", fmt.Sprintf("PID %d", cmd.Process.Pid))

	// Step 3: Wait for server ready signal
	spin = startSpinner(3, 3, "Waiting for server")

	type readyResult struct {
		msg string
		err error
	}
	ch := make(chan readyResult, 1)
	go func() {
		buf := make([]byte, 128)
		n, err := readyR.Read(buf)
		ch <- readyResult{strings.TrimSpace(string(buf[:n])), err}
	}()

	select {
	case res := <-ch:
		spin.stop()
		if res.err != nil || !strings.HasPrefix(res.msg, "READY") {
			stepFail(3, 3, "Waiting for server")
			fmt.Fprintf(os.Stderr, "  server failed to start — check logs: %s\n\n", logPath)
			os.Exit(1)
		}
		stepDone(3, 3, "Server ready", fmt.Sprintf("http://localhost:%d", *flagPort))
	case <-time.After(20 * time.Second):
		spin.stop()
		stepFail(3, 3, "Waiting for server")
		fmt.Fprintf(os.Stderr, "  timeout — check logs: %s\n\n", logPath)
		os.Exit(1)
	}

	// Success summary
	fmt.Println()
	fmt.Printf("  TermCity Web — %s, %s (%s)\n", loc.City, loc.State, zip)
	fmt.Printf("  http://localhost:%d\n", *flagPort)
	fmt.Printf("  Database: %s\n", dbPath)
	fmt.Printf("  Logs:     %s\n", logPath)
	fmt.Printf("  Stop:     kill %d   (or: pkill termcity-web)\n", cmd.Process.Pid)
	fmt.Println()

	// Detach from child so it keeps running
	cmd.Process.Release()
}

// ── Daemon (background server process) ───────────────────────────────────────

func runDaemon(zip string) {
	// Redirect log output to file
	logPath := logFilePath()
	if lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		log.SetOutput(lf)
	}

	// Get ready-pipe FD from environment (set by launcher)
	var readyFile *os.File
	if fdStr := os.Getenv("TERMCITY_READY_FD"); fdStr != "" {
		if fd, err := strconv.Atoi(fdStr); err == nil {
			readyFile = os.NewFile(uintptr(fd), "ready")
		}
	}

	fail := func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		log.Print(msg)
		if readyFile != nil {
			fmt.Fprintf(readyFile, "ERROR %s\n", msg)
			readyFile.Close()
		}
		os.Exit(1)
	}

	// Use pre-geocoded coords if provided by launcher, otherwise geocode now.
	var lat, lng float64
	var city string
	if *flagLat != 0 || *flagLng != 0 {
		lat, lng = *flagLat, *flagLng
		city = *flagCity
		log.Printf("Location: %s (%.4f, %.4f)", city, lat, lng)
	} else {
		log.Printf("Geocoding zip %s...", zip)
		loc, err := data.GeocodeZip(zip)
		if err != nil {
			fail("Failed to geocode zip %s: %v", zip, err)
		}
		lat, lng, city = loc.Lat, loc.Lng, loc.City
		log.Printf("Location: %s (%.4f, %.4f)", city, lat, lng)
	}

	dbPath := historyDBPath()
	store, err := history.Open(dbPath)
	if err != nil {
		fail("Failed to open history database: %v", err)
	}
	defer store.Close()

	a := &app{
		store: store,
		zip:   zip,
		lat:   lat,
		lng:   lng,
		city:  city,
	}

	// Bind listener before signalling ready so launcher knows port is available.
	addr := fmt.Sprintf(":%d", *flagPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fail("Failed to bind %s: %v", addr, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/api/incidents", a.handleLiveIncidents)
	mux.HandleFunc("/api/history", a.handleHistory)
	mux.HandleFunc("/api/history/clear", a.handleClearHistory)
	mux.HandleFunc("/api/status", a.handleStatus)
	srv := &http.Server{Handler: mux}

	// Signal ready to launcher; initial poll happens in background.
	if readyFile != nil {
		fmt.Fprintf(readyFile, "READY %d\n", os.Getpid())
		readyFile.Close()
	}

	// First poll in background so server responds immediately.
	go a.poll()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.pollLoop(ctx)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		cancel()
		srv.Shutdown(context.Background())
	}()

	log.Printf("HTTP server listening on %s", addr)
	if err := srv.Serve(listener); err != http.ErrServerClosed {
		log.Printf("HTTP server error: %v", err)
	}
}

// ── Background polling ────────────────────────────────────────────────────────

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

// ── Progress display ──────────────────────────────────────────────────────────

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type spinner struct {
	done chan struct{}
}

func startSpinner(step, total int, label string) *spinner {
	s := &spinner{done: make(chan struct{})}
	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				return
			case <-time.After(80 * time.Millisecond):
				fmt.Printf("\r  [%d/%d] %-42s %s", step, total, label, spinFrames[i%len(spinFrames)])
				i++
			}
		}
	}()
	return s
}

func (s *spinner) stop() {
	close(s.done)
	time.Sleep(10 * time.Millisecond) // let goroutine flush
}

func stepDone(step, total int, label, note string) {
	fmt.Printf("\r  [%d/%d] %-42s ✓  %s\n", step, total, label, note)
}

func stepFail(step, total int, label string) {
	fmt.Printf("\r  [%d/%d] %-42s ✗\n", step, total, label)
}

// ── Paths ─────────────────────────────────────────────────────────────────────

func historyDBPath() string {
	cacheDir, _ := os.UserCacheDir()
	if cacheDir == "" {
		cacheDir = os.TempDir()
	}
	dir := filepath.Join(cacheDir, "termcity")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "history.db")
}

func logFilePath() string {
	cacheDir, _ := os.UserCacheDir()
	if cacheDir == "" {
		cacheDir = os.TempDir()
	}
	dir := filepath.Join(cacheDir, "termcity")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "termcity-web.log")
}
