package tilemap

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	osmTileURL = "https://tile.openstreetmap.org/%d/%d/%d.png"
	userAgent  = "termcity/1.0 (github.com/termcity)"
)

// TileSource identifies a map tile provider/style.
type TileSource int

const (
	SourceOSM   TileSource = iota // OpenStreetMap default
	SourceDark                    // CartoDB Dark Matter
	SourceLight                   // CartoDB Positron
	numSources
)

// Name returns a short display label for the source.
func (s TileSource) Name() string {
	switch s {
	case SourceDark:
		return "Dark"
	case SourceLight:
		return "Light"
	default:
		return "OSM"
	}
}

// Next cycles to the next tile source.
func (s TileSource) Next() TileSource {
	return (s + 1) % numSources
}

var cartoSubdomains = []string{"a", "b", "c", "d"}

func tileURL(src TileSource, z, x, y int) string {
	switch src {
	case SourceDark:
		sub := cartoSubdomains[(x+y)%4]
		return fmt.Sprintf("https://%s.basemaps.cartocdn.com/dark_all/%d/%d/%d.png", sub, z, x, y)
	case SourceLight:
		sub := cartoSubdomains[(x+y)%4]
		return fmt.Sprintf("https://%s.basemaps.cartocdn.com/light_all/%d/%d/%d.png", sub, z, x, y)
	default:
		return fmt.Sprintf(osmTileURL, z, x, y)
	}
}

// OSM rate limiter: max 2 req/s per ToS.
var (
	osmRateMu      sync.Mutex
	osmLastRequest time.Time
	osmMinInterval = 500 * time.Millisecond
)

func osmRateLimit() {
	osmRateMu.Lock()
	defer osmRateMu.Unlock()
	if elapsed := time.Since(osmLastRequest); elapsed < osmMinInterval {
		time.Sleep(osmMinInterval - elapsed)
	}
	osmLastRequest = time.Now()
}

// CartoDB semaphore: allow up to 3 concurrent requests (separate servers from OSM).
var cartoSem = make(chan struct{}, 3)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// FetchTile fetches a tile PNG from cache or the given source. Returns raw PNG bytes.
func FetchTile(src TileSource, z, x, y int) ([]byte, error) {
	// Try disk cache first.
	data, err := ReadCachedTile(src, z, x, y)
	if err == nil && data != nil {
		return data, nil
	}

	// Apply per-source rate limiting / concurrency control before network request.
	switch src {
	case SourceOSM:
		osmRateLimit()
	default: // CartoDB Dark / Light
		cartoSem <- struct{}{}
		defer func() { <-cartoSem }()
	}

	url := tileURL(src, z, x, y)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching tile %d/%d/%d: %w", z, x, y, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tile %d/%d/%d: HTTP %d", z, x, y, resp.StatusCode)
	}

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading tile body: %w", err)
	}

	// Cache to disk (best-effort).
	_ = WriteCachedTile(src, z, x, y, data)

	return data, nil
}

// TileKey uniquely identifies a tile including its source style.
type TileKey struct {
	Z, X, Y int
	Source  TileSource
}
