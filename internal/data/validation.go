package data

import "time"

// FreshnessBucket describes a coarse age bucket for an incident.
type FreshnessBucket int

const (
	FreshnessUnknown FreshnessBucket = iota
	FreshnessNew                     // < 15 minutes
	FreshnessRecent                  // 15–60 minutes
	FreshnessStale                   // 1–4 hours
	FreshnessOld                     // > 4 hours
	FreshnessFuture                  // timestamp is in the future beyond allowed skew
)

// ClassifyFreshness returns a bucket describing how old the incident is.
// now is passed in so callers can supply a consistent snapshot time.
func ClassifyFreshness(t, now time.Time) FreshnessBucket {
	if t.IsZero() || now.IsZero() {
		return FreshnessUnknown
	}

	// Treat anything more than 5 minutes in the future as clock skew / bad data.
	if t.After(now.Add(5 * time.Minute)) {
		return FreshnessFuture
	}

	d := now.Sub(t)
	if d < 0 {
		// Small negative deltas are treated as "new".
		return FreshnessNew
	}
	switch {
	case d < 15*time.Minute:
		return FreshnessNew
	case d < time.Hour:
		return FreshnessRecent
	case d < 4*time.Hour:
		return FreshnessStale
	default:
		return FreshnessOld
	}
}

// FreshnessLabel returns a short string label for the freshness bucket.
func FreshnessLabel(b FreshnessBucket) string {
	switch b {
	case FreshnessNew:
		return "NEW"
	case FreshnessRecent:
		return "RECENT"
	case FreshnessStale:
		return "STALE"
	case FreshnessOld:
		return "OLD"
	case FreshnessFuture:
		return "FUTURE?"
	default:
		return ""
	}
}
