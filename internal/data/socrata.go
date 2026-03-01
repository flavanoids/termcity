package data

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SocrataDataset describes a city's Socrata police incident dataset.
type SocrataDataset struct {
	Domain    string
	DatasetID string
	DateField string // name of the timestamp field
	LatField  string
	LngField  string
	TypeField string
	AddrField string
}

// socrataRegistry maps lowercase city names to their dataset info.
var socrataRegistry = map[string]SocrataDataset{
	"new york": {
		Domain:    "data.cityofnewyork.us",
		DatasetID: "qgea-i56i",
		DateField: "cmplnt_fr_dt",
		LatField:  "latitude",
		LngField:  "longitude",
		TypeField: "ofns_desc",
		AddrField: "addr_pct_cd",
	},
	"chicago": {
		Domain:    "data.cityofchicago.org",
		DatasetID: "ijzp-q8t2",
		DateField: "date",
		LatField:  "latitude",
		LngField:  "longitude",
		TypeField: "primary_type",
		AddrField: "block",
	},
	"los angeles": {
		Domain:    "data.lacity.org",
		DatasetID: "2nrs-mtv8",
		DateField: "date_occ",
		LatField:  "lat",
		LngField:  "lon",
		TypeField: "crm_cd_desc",
		AddrField: "location_1",
	},
	"san francisco": {
		Domain:    "data.sfgov.org",
		DatasetID: "wg3w-h783",
		DateField: "incident_datetime",
		LatField:  "latitude",
		LngField:  "longitude",
		TypeField: "incident_category",
		AddrField: "intersection",
	},
	"seattle": {
		Domain:    "data.seattle.gov",
		DatasetID: "tazs-3rd5",
		DateField: "event_clearance_date",
		LatField:  "latitude",
		LngField:  "longitude",
		TypeField: "event_clearance_description",
		AddrField: "hundred_block_location",
	},
	"denver": {
		Domain:    "www.denvergov.org",
		DatasetID: "d9t3-3gj6",
		DateField: "first_occurrence_date",
		LatField:  "geo_lat",
		LngField:  "geo_lon",
		TypeField: "offense_type_id",
		AddrField: "incident_address",
	},
	"austin": {
		Domain:    "data.austintexas.gov",
		DatasetID: "r3af-2r8x",
		DateField: "occ_date_time",
		LatField:  "latitude",
		LngField:  "longitude",
		TypeField: "highest_offense_desc",
		AddrField: "location_type",
	},
	"portland": {
		Domain:    "www.portlandoregon.gov",
		DatasetID: "h5ff-oykk",
		DateField: "report_date",
		LatField:  "lat",
		LngField:  "long",
		TypeField: "offense_type",
		AddrField: "neighborhood",
	},
}

var socrataClient = &http.Client{Timeout: 15 * time.Second}

// FetchSocrataIncidents fetches police incidents from Socrata for a city.
// Returns nil, nil if the city is not in the registry.
func FetchSocrataIncidents(city string) ([]Incident, error) {
	key := strings.ToLower(strings.TrimSpace(city))
	ds, ok := socrataRegistry[key]
	if !ok {
		// Try partial match.
		for k, v := range socrataRegistry {
			if strings.Contains(key, k) || strings.Contains(k, key) {
				ds = v
				ok = true
				break
			}
		}
		if !ok {
			return nil, nil // No data available for this city.
		}
	}
	return fetchSocrataData(ds)
}

func fetchSocrataData(ds SocrataDataset) ([]Incident, error) {
	since := time.Now().Add(-6 * time.Hour).Format("2006-01-02T15:04:05")
	where := fmt.Sprintf("%s > '%s'", ds.DateField, since)

	params := url.Values{}
	params.Set("$where", where)
	params.Set("$limit", "200")
	params.Set("$order", ds.DateField+" DESC")

	reqURL := fmt.Sprintf("https://%s/resource/%s.json?%s", ds.Domain, ds.DatasetID, params.Encode())
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "termcity/1.0")

	resp, err := socrataClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Socrata fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Socrata HTTP %d", resp.StatusCode)
	}

	var rows []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("Socrata decode: %w", err)
	}

	incidents := make([]Incident, 0, len(rows))
	for i, row := range rows {
		inc := parseSocrataRow(row, ds, i)
		if inc.Lat == 0 && inc.Lng == 0 {
			continue // Skip rows without location data.
		}
		incidents = append(incidents, inc)
	}
	return incidents, nil
}

func parseSocrataRow(row map[string]interface{}, ds SocrataDataset, idx int) Incident {
	getString := func(key string) string {
		if v, ok := row[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}
	getFloat := func(key string) float64 {
		s := getString(key)
		var f float64
		fmt.Sscanf(s, "%f", &f)
		return f
	}

	id := getString("unique_key")
	if id == "" {
		id = fmt.Sprintf("soc-%d", idx)
	} else {
		id = "soc-" + id
	}

	title := getString(ds.TypeField)
	if title == "" {
		title = "Police Incident"
	}

	addr := getString(ds.AddrField)

	var t time.Time
	dateStr := getString(ds.DateField)
	if dateStr != "" {
		formats := []string{
			"2006-01-02T15:04:05.000",
			"2006-01-02T15:04:05",
			"01/02/2006 03:04:05 PM",
			"2006-01-02",
		}
		for _, f := range formats {
			if parsed, err := time.Parse(f, dateStr); err == nil {
				t = parsed
				break
			}
		}
	}
	if t.IsZero() {
		t = time.Now()
	}

	// Handle nested location objects (some Socrata datasets use this).
	lat := getFloat(ds.LatField)
	lng := getFloat(ds.LngField)

	if lat == 0 {
		if loc, ok := row["location"].(map[string]interface{}); ok {
			latStr := fmt.Sprintf("%v", loc["latitude"])
			lngStr := fmt.Sprintf("%v", loc["longitude"])
			fmt.Sscanf(latStr, "%f", &lat)
			fmt.Sscanf(lngStr, "%f", &lng)
		}
	}

	return Incident{
		ID:      id,
		Type:    Police,
		Title:   title,
		Address: addr,
		Lat:     lat,
		Lng:     lng,
		Time:    t,
		Source:  "socrata",
	}
}
