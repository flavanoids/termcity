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
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	DisplayName string  `json:"display_name"`
	City        string  `json:"city"`
	State       string  `json:"state"`
}

// GeocodeAddress resolves a street address to lat/lng via Nominatim structured search.
// Returns 0, 0, nil if no results found (not an error).
func GeocodeAddress(street, city, state string) (float64, float64, error) {
	geocodeRateLimit()

	params := url.Values{}
	params.Set("street", street)
	params.Set("city", city)
	params.Set("state", state)
	params.Set("countrycodes", "us")
	params.Set("format", "json")
	params.Set("limit", "1")

	reqURL := nominatimURL + "?" + params.Encode()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("User-Agent", nominatimUserAgent)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("geocode address: %w", err)
	}
	defer resp.Body.Close()

	var results []nominatimResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return 0, 0, fmt.Errorf("geocode address decode: %w", err)
	}
	if len(results) == 0 {
		return 0, 0, nil
	}

	var lat, lng float64
	fmt.Sscanf(results[0].Lat, "%f", &lat)
	fmt.Sscanf(results[0].Lon, "%f", &lng)
	return lat, lng, nil
}

// GeocodeZip converts a US zip code to lat/lng via Nominatim, with fallback to zippopotam.us.
func GeocodeZip(zip string) (*GeoLocation, error) {
	if loc, err := geocodeZipNominatim(zip); err == nil {
		return loc, nil
	}
	return geocodeZipFallback(zip)
}

func geocodeZipNominatim(zip string) (*GeoLocation, error) {
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nominatim returned HTTP %d", resp.StatusCode)
	}

	var results []struct {
		Lat     string `json:"lat"`
		Lon     string `json:"lon"`
		Display string `json:"display_name"`
		Address struct {
			City     string `json:"city"`
			Town     string `json:"town"`
			Village  string `json:"village"`
			State    string `json:"state"`
			Postcode string `json:"postcode"`
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

// geocodeZipFallback uses api.zippopotam.us when Nominatim is unavailable.
func geocodeZipFallback(zip string) (*GeoLocation, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.zippopotam.us/us/" + url.PathEscape(zip))
	if err != nil {
		return nil, fmt.Errorf("geocode fallback request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geocode fallback returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		PostCode string `json:"post code"`
		Country  string `json:"country"`
		Places   []struct {
			PlaceName string `json:"place name"`
			Longitude string `json:"longitude"`
			Latitude  string `json:"latitude"`
			State     string `json:"state"`
		} `json:"places"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("geocode fallback decode: %w", err)
	}
	if len(result.Places) == 0 {
		return nil, fmt.Errorf("no geocode results for zip %q", zip)
	}

	p := result.Places[0]
	var lat, lng float64
	fmt.Sscanf(p.Latitude, "%f", &lat)
	fmt.Sscanf(p.Longitude, "%f", &lng)

	return &GeoLocation{
		Lat:         lat,
		Lng:         lng,
		DisplayName: fmt.Sprintf("%s, %s %s", p.PlaceName, p.State, zip),
		City:        p.PlaceName,
		State:       p.State,
	}, nil
}
