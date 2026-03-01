package data

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const houstonURL = "https://cohweb.houstontx.gov/activeincidents/combined.aspx"

// houstonGeoCache persists geocoded Houston addresses across refreshes.
// Keys are "ADDRESS @ CROSSSTREET" (or just "ADDRESS").
var (
	houstonGeoMu   sync.RWMutex
	houstonGeoData = map[string][2]float64{}
	houstonGeoPath string
)

func init() {
	dir, _ := os.UserCacheDir()
	houstonGeoPath = filepath.Join(dir, "termcity", "addr_geocode.json")
	data, err := os.ReadFile(houstonGeoPath)
	if err == nil {
		houstonGeoMu.Lock()
		json.Unmarshal(data, &houstonGeoData) //nolint:errcheck
		houstonGeoMu.Unlock()
	}
}

func saveHoustonGeoCache() {
	houstonGeoMu.RLock()
	data, err := json.Marshal(houstonGeoData)
	houstonGeoMu.RUnlock()
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(houstonGeoPath), 0755) //nolint:errcheck
	os.WriteFile(houstonGeoPath, data, 0644)         //nolint:errcheck
}

type houstonRow struct {
	agency, address, crossStreet, callTime, incidentType string
}

var (
	reTR  = regexp.MustCompile(`(?si)<tr[^>]*>(.*?)</tr>`)
	reTD  = regexp.MustCompile(`(?si)<t[dh][^>]*>(.*?)</t[dh]>`)
	reTag = regexp.MustCompile(`<[^>]+>`)
)

func parseHoustonHTML(html string) []houstonRow {
	var rows []houstonRow
	for _, tr := range reTR.FindAllStringSubmatch(html, -1) {
		cells := reTD.FindAllStringSubmatch(tr[1], -1)
		if len(cells) < 6 {
			continue
		}
		col := func(i int) string {
			s := reTag.ReplaceAllString(cells[i][1], "")
			s = strings.ReplaceAll(s, "&nbsp;", "")
			return strings.TrimSpace(s)
		}
		agency := col(0)
		if agency != "FD" && agency != "PD" {
			continue
		}
		rows = append(rows, houstonRow{
			agency:       agency,
			address:      col(1),
			crossStreet:  col(2),
			callTime:     col(4),
			incidentType: col(5),
		})
	}
	return rows
}

func houstonCacheKey(row houstonRow) string {
	if row.crossStreet != "" {
		return row.address + " @ " + row.crossStreet
	}
	return row.address
}

func buildHoustonIncident(row houstonRow, lat, lng float64) Incident {
	itype := EMS
	lower := strings.ToLower(row.incidentType)
	switch {
	case row.agency == "PD":
		itype = Police
	case strings.Contains(lower, "fire") || strings.Contains(lower, "structure") ||
		strings.Contains(lower, "brush") || strings.Contains(lower, "alarm"):
		itype = Fire
	}

	addr := row.address
	if row.crossStreet != "" {
		addr = row.address + " @ " + row.crossStreet
	}

	t, _ := time.Parse("01/02/2006 15:04", row.callTime)
	if t.IsZero() {
		t = time.Now()
	}

	return Incident{
		ID:      fmt.Sprintf("hou-%s-%s", strings.ReplaceAll(addr, " ", ""), row.callTime),
		Type:    itype,
		Title:   row.incidentType,
		Address: addr + ", Houston, TX",
		Lat:     lat,
		Lng:     lng,
		Time:    t,
		Source:  "houston",
	}
}

// FetchHoustonIncidents scrapes active HFD/HPD incidents from the City of Houston.
// Incidents without cached geocoordinates are geocoded in the background; they
// will appear on the next refresh once the cache is warm.
func FetchHoustonIncidents() ([]Incident, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", houstonURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "termcity/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("houston fetch: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	rows := parseHoustonHTML(string(body))

	var incidents []Incident
	var uncached []houstonRow

	houstonGeoMu.RLock()
	for _, row := range rows {
		key := houstonCacheKey(row)
		if coords, ok := houstonGeoData[key]; ok {
			incidents = append(incidents, buildHoustonIncident(row, coords[0], coords[1]))
		} else {
			uncached = append(uncached, row)
		}
	}
	houstonGeoMu.RUnlock()

	if len(uncached) > 0 {
		go geocodeHoustonRows(uncached)
	}

	return incidents, nil
}

// geocodeHoustonRows geocodes uncached Houston addresses one at a time,
// respecting the shared Nominatim rate limit, and persists results to disk.
func geocodeHoustonRows(rows []houstonRow) {
	changed := false
	for _, row := range rows {
		key := houstonCacheKey(row)

		houstonGeoMu.RLock()
		_, exists := houstonGeoData[key]
		houstonGeoMu.RUnlock()
		if exists {
			continue
		}

		street := row.address
		if row.crossStreet != "" {
			street = row.address + " & " + row.crossStreet
		}

		lat, lng, err := GeocodeAddress(street, "Houston", "TX")
		if err != nil || (lat == 0 && lng == 0) {
			continue
		}

		houstonGeoMu.Lock()
		houstonGeoData[key] = [2]float64{lat, lng}
		houstonGeoMu.Unlock()
		changed = true
	}
	if changed {
		saveHoustonGeoCache()
	}
}
