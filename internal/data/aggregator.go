package data

import (
	"sort"
	"strings"
	"time"
)

// FetchAllIncidents fetches from all available sources and merges/deduplicates.
func FetchAllIncidents(lat, lng float64, city string) ([]Incident, []string) {
	type result struct {
		incidents []Incident
		source    string
		err       error
	}

	isHouston := strings.EqualFold(strings.TrimSpace(city), "houston")
	sources := 2
	if isHouston {
		sources = 3
	}

	ch := make(chan result, sources)

	// Fetch PulsePoint (fire/EMS).
	go func() {
		incidents, err := FetchPulsePointIncidents(lat, lng)
		ch <- result{incidents, "pulsepoint", err}
	}()

	// Fetch Socrata (police).
	go func() {
		incidents, err := FetchSocrataIncidents(city)
		ch <- result{incidents, "socrata", err}
	}()

	// Houston combined FD/PD feed (replaces both for Houston).
	if isHouston {
		go func() {
			incidents, err := FetchHoustonIncidents()
			ch <- result{incidents, "houston", err}
		}()
	}

	var all []Incident
	var warnings []string

	for i := 0; i < sources; i++ {
		r := <-ch
		if r.err != nil {
			warnings = append(warnings, r.source+": "+r.err.Error())
		} else {
			all = append(all, r.incidents...)
		}
	}

	// Deduplicate by ID.
	seen := make(map[string]bool)
	deduped := make([]Incident, 0, len(all))
	for _, inc := range all {
		if !seen[inc.ID] {
			seen[inc.ID] = true
			deduped = append(deduped, inc)
		}
	}

	// Sort by time descending (newest first).
	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].Time.After(deduped[j].Time)
	})

	// Filter out incidents older than 2 hours.
	cutoff := time.Now().Add(-2 * time.Hour)
	filtered := make([]Incident, 0, len(deduped))
	for _, inc := range deduped {
		if inc.Time.After(cutoff) {
			filtered = append(filtered, inc)
		}
	}

	return filtered, warnings
}
