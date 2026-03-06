package data

import "time"

// EnrichmentOptions configures optional external enrichment of incidents.
// All fields are best-effort; failures are always non-fatal.
type EnrichmentOptions struct {
	// EnableTimezoneNormalization controls whether incident timestamps should
	// be normalized into a specific location's time zone when available.
	EnableTimezoneNormalization bool
	// TargetLocation is a best-effort identifier for the location used when
	// normalizing times (e.g. city name). For now this is only used for
	// future extension points.
	TargetLocation string
}

// EnrichIncidents applies lightweight, best-effort enrichment to a slice of
// incidents. It returns a new slice; the input is never mutated.
//
// For now, this function simply normalizes timestamps to the local system
// time zone when EnableTimezoneNormalization is true. It is intentionally
// conservative and can be extended later to call free, unauthenticated
// APIs (timezone or reverse geocoding) while keeping failures non-fatal.
func EnrichIncidents(incidents []Incident, opts EnrichmentOptions) []Incident {
	if !opts.EnableTimezoneNormalization || len(incidents) == 0 {
		// Return a shallow copy so callers can safely assume ownership.
		out := make([]Incident, len(incidents))
		copy(out, incidents)
		return out
	}

	loc := time.Local

	out := make([]Incident, len(incidents))
	for i, inc := range incidents {
		incCopy := inc
		if !incCopy.Time.IsZero() {
			incCopy.Time = incCopy.Time.In(loc)
		}
		out[i] = incCopy
	}
	return out
}
