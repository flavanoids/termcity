package data

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	pulsepointPSAPURL      = "https://web.pulsepoint.org/DB/giba.php?type=giba&lat=%f&lng=%f"
	pulsepointIncidentURL  = "https://web.pulsepoint.org/DB/giba.php?psap_id=%s&_=%d"
	pulsepointUserAgent    = "termcity/1.0"
)

var ppClient = &http.Client{Timeout: 10 * time.Second}

// psapResponse is the response from the PSAP lookup.
type psapResponse struct {
	PSAP []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"psap"`
}

// pulsepointIncidentResponse is the raw incident response.
type pulsepointIncidentResponse struct {
	Incidents struct {
		Active []pulsepointRawIncident `json:"active"`
	} `json:"incidents"`
}

type pulsepointRawIncident struct {
	ID              string `json:"id"`
	CallType        string `json:"call_type"`
	FullDisplayAddr string `json:"full_display_address"`
	Lat             string `json:"lat"`
	Lng             string `json:"lng"`
	PulsePointTime  string `json:"unit_status_transport_timestamp"`
	CallCreated     string `json:"call_created_date_time"`
	Units           []struct {
		UnitID string `json:"unit_id"`
	} `json:"units"`
}

// FetchPulsePointIncidents fetches active incidents near lat/lng.
func FetchPulsePointIncidents(lat, lng float64) ([]Incident, error) {
	psapID, err := lookupPSAP(lat, lng)
	if err != nil {
		return nil, fmt.Errorf("PSAP lookup: %w", err)
	}
	if psapID == "" {
		return nil, nil
	}
	return fetchIncidents(psapID)
}

func lookupPSAP(lat, lng float64) (string, error) {
	url := fmt.Sprintf(pulsepointPSAPURL, lat, lng)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", pulsepointUserAgent)

	resp, err := ppClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("PSAP HTTP %d", resp.StatusCode)
	}

	var result psapResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.PSAP) == 0 {
		return "", nil
	}
	return result.PSAP[0].ID, nil
}

func fetchIncidents(psapID string) ([]Incident, error) {
	ms := time.Now().UnixMilli()
	url := fmt.Sprintf(pulsepointIncidentURL, psapID, ms)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", pulsepointUserAgent)

	resp, err := ppClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("incidents HTTP %d", resp.StatusCode)
	}

	var raw pulsepointIncidentResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding incidents: %w", err)
	}

	incidents := make([]Incident, 0, len(raw.Incidents.Active))
	for _, r := range raw.Incidents.Active {
		inc := parsePulsePointIncident(r)
		incidents = append(incidents, inc)
	}
	return incidents, nil
}

func parsePulsePointIncident(r pulsepointRawIncident) Incident {
	itype := callTypeToIncidentType(r.CallType)

	var lat, lng float64
	fmt.Sscanf(r.Lat, "%f", &lat)
	fmt.Sscanf(r.Lng, "%f", &lng)

	var t time.Time
	if r.CallCreated != "" {
		t, _ = time.Parse("01/02/2006 15:04:05 PM", r.CallCreated)
		if t.IsZero() {
			t, _ = time.Parse("2006-01-02T15:04:05", r.CallCreated)
		}
	}
	if t.IsZero() {
		t = time.Now()
	}

	units := make([]string, 0, len(r.Units))
	for _, u := range r.Units {
		if u.UnitID != "" {
			units = append(units, u.UnitID)
		}
	}

	title := callTypeTitle(r.CallType)

	return Incident{
		ID:      "pp-" + r.ID,
		Type:    itype,
		Title:   title,
		Address: r.FullDisplayAddr,
		Lat:     lat,
		Lng:     lng,
		Time:    t,
		Units:   units,
		Source:  "pulsepoint",
	}
}

// callTypeToIncidentType maps PulsePoint call type codes to IncidentType.
func callTypeToIncidentType(code string) IncidentType {
	switch code {
	case "F", "FS", "FR", "FW", "FA", "FB", "FU":
		return Fire
	case "ME", "MA", "MCI", "MC":
		return EMS
	default:
		// Most other codes (TE, TC, etc.) map to EMS as a default for fire/EMS agency.
		return EMS
	}
}

// callTypeTitle returns a human-readable title for a PulsePoint call type.
func callTypeTitle(code string) string {
	titles := map[string]string{
		"F":   "Structure Fire",
		"FS":  "Fire (Smoke)",
		"FR":  "Fire Reported Out",
		"FW":  "Wildland Fire",
		"FA":  "Fire Alarm",
		"FB":  "Brush Fire",
		"FU":  "Unknown Fire Type",
		"ME":  "Medical Emergency",
		"MA":  "Medical Aid",
		"MCI": "Mass Casualty Incident",
		"MC":  "Medical Call",
		"TE":  "Traffic Emergency",
		"TC":  "Traffic Collision",
		"HZ":  "Hazmat",
		"RS":  "Rescue",
		"WR":  "Water Rescue",
	}
	if t, ok := titles[code]; ok {
		return t
	}
	return "Incident (" + code + ")"
}
