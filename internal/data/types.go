package data

import (
	"encoding/json"
	"strings"
	"time"
)

// IncidentType categorizes an incident.
type IncidentType int

const (
	Fire   IncidentType = iota
	Police IncidentType = iota
	EMS    IncidentType = iota
)

// MarshalJSON emits "fire", "police", "ems" for API consumers.
func (t IncidentType) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(t.String()))
}

func (t IncidentType) String() string {
	switch t {
	case Fire:
		return "Fire"
	case Police:
		return "Police"
	case EMS:
		return "EMS"
	default:
		return "Unknown"
	}
}

func (t IncidentType) Symbol() string {
	switch t {
	case Fire:
		return "🔥"
	case Police:
		return "👮"
	case EMS:
		return "🚑"
	default:
		return "⚪"
	}
}

func (t IncidentType) Color() string {
	switch t {
	case Fire:
		return "#FF4444"
	case Police:
		return "#4488FF"
	case EMS:
		return "#EEEEEE"
	default:
		return "#AAAAAA"
	}
}

// Incident represents a single 911 incident.
type Incident struct {
	ID      string       `json:"id"`
	Type    IncidentType `json:"type"`
	Title   string       `json:"title"`
	Address string       `json:"address"`
	Lat     float64      `json:"lat"`
	Lng     float64      `json:"lng"`
	Time    time.Time    `json:"time"`
	Units   []string     `json:"units"`
	Source  string       `json:"source"` // "pulsepoint" | "socrata" | "houston"
}
