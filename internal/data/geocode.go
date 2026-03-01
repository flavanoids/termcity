package data

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const nominatimURL = "https://nominatim.openstreetmap.org/search"
const nominatimUserAgent = "termcity/1.0 (github.com/termcity)"

// Nominatim rate limit: 1 req/s.
var (
	geocodeMu       sync.Mutex
	lastGeocodeTime time.Time
)

func geocodeRateLimit() {
	geocodeMu.Lock()
	defer geocodeMu.Unlock()
	elapsed := time.Since(lastGeocodeTime)
	if elapsed < time.Second {
		time.Sleep(time.Second - elapsed)
	}
	lastGeocodeTime = time.Now()
}

type nominatimResult struct {
	Lat     string `json:"lat"`
	Lon     string `json:"lon"`
	Display string `json:"display_name"`
}

// GeoLocation holds a geocoded lat/lng with a display name.
type GeoLocation struct {
	Lat, Lng    float64
	DisplayName string
	City        string
	State       string
}

// GeocodeZip converts a US zip code to lat/lng via Nominatim.
func GeocodeZip(zip string) (*GeoLocation, error) {
	geocodeRateLimit()

	params := url.Values{}
	params.Set("q", zip)
	params.Set("format", "json")
	params.Set("countrycodes", "us")
	params.Set("limit", "1")
	params.Set("addressdetails", "1")

	reqURL := nominatimURL + "?" + params.Encode()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", nominatimUserAgent)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("geocode request: %w", err)
	}
	defer resp.Body.Close()

	var results []struct {
		Lat     string `json:"lat"`
		Lon     string `json:"lon"`
		Display string `json:"display_name"`
		Address struct {
			City       string `json:"city"`
			Town       string `json:"town"`
			Village    string `json:"village"`
			State      string `json:"state"`
			Postcode   string `json:"postcode"`
		} `json:"address"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("geocode decode: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no geocode results for zip %q", zip)
	}

	r := results[0]
	var lat, lng float64
	fmt.Sscanf(r.Lat, "%f", &lat)
	fmt.Sscanf(r.Lon, "%f", &lng)

	city := r.Address.City
	if city == "" {
		city = r.Address.Town
	}
	if city == "" {
		city = r.Address.Village
	}

	return &GeoLocation{
		Lat:         lat,
		Lng:         lng,
		DisplayName: r.Display,
		City:        city,
		State:       r.Address.State,
	}, nil
}
