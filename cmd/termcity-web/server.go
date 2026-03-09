package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"termcity/internal/data"
	"time"
)

//go:embed index.html
var indexHTML []byte

type incidentJSON struct {
	Title   string   `json:"title"`
	Address string   `json:"address"`
	Lat     float64  `json:"lat"`
	Lng     float64  `json:"lng"`
	Type    string   `json:"type"`
	Color   string   `json:"color"`
	Source  string   `json:"source"`
	Units   []string `json:"units,omitempty"`
	Time    string   `json:"time"`
	Ago     string   `json:"ago"`
}

func toJSON(inc data.Incident) incidentJSON {
	return incidentJSON{
		Title:   inc.Title,
		Address: inc.Address,
		Lat:     inc.Lat,
		Lng:     inc.Lng,
		Type:    inc.Type.String(),
		Color:   inc.Type.Color(),
		Source:  inc.Source,
		Units:   inc.Units,
		Time:    inc.Time.Format("2006-01-02 15:04"),
		Ago:     timeAgo(inc.Time),
	}
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

func (a *app) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func (a *app) handleLiveIncidents(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	incs := make([]incidentJSON, len(a.live))
	for i, inc := range a.live {
		incs[i] = toJSON(inc)
	}
	resp := map[string]any{
		"incidents": incs,
		"warnings":  a.warnings,
		"zip":       a.zip,
		"city":      a.city,
		"lat":       a.lat,
		"lng":       a.lng,
		"last_poll": a.lastPoll,
	}
	a.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *app) handleHistory(w http.ResponseWriter, r *http.Request) {
	days := 1
	switch r.URL.Query().Get("days") {
	case "3":
		days = 3
	case "7":
		days = 7
	}

	incidents, err := a.store.QueryHistory(days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	incs := make([]incidentJSON, len(incidents))
	for i, inc := range incidents {
		incs[i] = toJSON(inc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"days":      days,
		"incidents": incs,
		"count":     len(incs),
	})
}

func (a *app) handleClearHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if err := a.store.ClearHistory(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})
}

func (a *app) handleStatus(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	lastPoll := a.lastPoll
	liveCount := len(a.live)
	a.mu.RUnlock()

	d1, d3, d7, _ := a.store.Stats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"zip":        a.zip,
		"city":       a.city,
		"live_count": liveCount,
		"last_poll":  lastPoll,
		"history_1d": d1,
		"history_3d": d3,
		"history_7d": d7,
	})
}
