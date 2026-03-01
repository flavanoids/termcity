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

// rateLimiter enforces max 2 requests per second for OSM tiles.
var (
	rateMu      sync.Mutex
	lastRequest time.Time
	minInterval = 500 * time.Millisecond // 2 req/s
)

func rateLimit() {
	rateMu.Lock()
	defer rateMu.Unlock()
	now := time.Now()
	elapsed := now.Sub(lastRequest)
	if elapsed < minInterval {
		time.Sleep(minInterval - elapsed)
	}
	lastRequest = time.Now()
}

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// FetchTile fetches a tile PNG from cache or OSM. Returns raw PNG bytes.
func FetchTile(z, x, y int) ([]byte, error) {
	// Try cache first.
	data, err := ReadCachedTile(z, x, y)
	if err == nil && data != nil {
		return data, nil
	}

	// Rate limit before network request.
	rateLimit()

	url := fmt.Sprintf(osmTileURL, z, x, y)
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
	_ = WriteCachedTile(z, x, y, data)

	return data, nil
}

// TileKey uniquely identifies a tile.
type TileKey struct {
	Z, X, Y int
}
