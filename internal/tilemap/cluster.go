package tilemap

import (
	"fmt"
	"termcity/internal/data"
)

// Cluster represents a group of nearby incidents.
type Cluster struct {
	Lat       float64
	Lng       float64
	Count     int
	Types     map[data.IncidentType]int
	Incidents []data.Incident // original incidents in this cluster
}

// ShouldCluster returns true if clustering should be applied at this zoom level.
func ShouldCluster(zoom int) bool {
	return zoom < 13
}

// ClusterIncidents groups nearby incidents into clusters based on grid cell.
// At low zoom levels, multiple incidents may occupy the same screen cell.
func ClusterIncidents(incidents []data.Incident, lat, lng float64, zoom, mapCols, mapRows int) []Cluster {
	if len(incidents) == 0 {
		return nil
	}

	originPX, originPY := MapOriginPixel(lat, lng, zoom, mapCols, mapRows)

	// Grid cell size in pixels - clusters incidents that would overlap on screen
	// A typical marker is ~3-4 cells wide, so group things within that range
	cellSizePX := 32 // ~1/8 of a tile, creates reasonable clustering

	type cellKey struct{ cx, cy int }
	cells := make(map[cellKey]*Cluster)

	for _, inc := range incidents {
		px, py := LatLngToPixelCoord(inc.Lat, inc.Lng, zoom)
		col := px - originPX
		row := (py - originPY) / 2

		// Map to cluster cell
		cellX := col / cellSizePX
		cellY := row / cellSizePX
		key := cellKey{cellX, cellY}

		if cluster, ok := cells[key]; ok {
			cluster.Count++
			cluster.Types[inc.Type]++
			cluster.Incidents = append(cluster.Incidents, inc)
			// Weighted average position
			cluster.Lat = (cluster.Lat*float64(cluster.Count-1) + inc.Lat) / float64(cluster.Count)
			cluster.Lng = (cluster.Lng*float64(cluster.Count-1) + inc.Lng) / float64(cluster.Count)
		} else {
			cells[key] = &Cluster{
				Lat:       inc.Lat,
				Lng:       inc.Lng,
				Count:     1,
				Types:     map[data.IncidentType]int{inc.Type: 1},
				Incidents: []data.Incident{inc},
			}
		}
	}

	// Convert map to slice
	clusters := make([]Cluster, 0, len(cells))
	for _, c := range cells {
		clusters = append(clusters, *c)
	}
	return clusters
}

// DominantType returns the most common incident type in the cluster.
func (c Cluster) DominantType() data.IncidentType {
	var maxType data.IncidentType
	maxCount := 0
	for t, count := range c.Types {
		if count > maxCount {
			maxCount = count
			maxType = t
		}
	}
	return maxType
}

// ClusterLabel returns the display label for a cluster (number or "99+").
func (c Cluster) ClusterLabel() string {
	if c.Count < 100 {
		return fmt.Sprintf("%d", c.Count)
	}
	return "99+"
}
