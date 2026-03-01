package data

import "time"

// IncidentType categorizes an incident.
type IncidentType int

const (
	Fire   IncidentType = iota
	Police IncidentType = iota
	EMS    IncidentType = iota
)

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
		return "🔴"
	case Police:
		return "🔵"
	case EMS:
		return "⬜"
	default:
		return "⬜"
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
	ID      string
	Type    IncidentType
	Title   string
	Address string
	Lat     float64
	Lng     float64
	Time    time.Time
	Units   []string
	Source  string // "pulsepoint" | "socrata"
}
