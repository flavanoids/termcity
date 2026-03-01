package tilemap

import "math"

// Zoom levels supported by OSM.
const (
	MinZoom = 1
	MaxZoom = 19
)

// TileSize is the pixel dimensions of a single OSM tile.
const TileSize = 256

// LatLngToTileXY converts geographic coordinates to tile indices at given zoom.
func LatLngToTileXY(lat, lng float64, zoom int) (tx, ty int) {
	n := math.Pow(2, float64(zoom))
	tx = int(math.Floor((lng + 180.0) / 360.0 * n))
	latRad := lat * math.Pi / 180.0
	ty = int(math.Floor((1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n))
	return tx, ty
}

// TileXYToLatLng converts tile indices to the lat/lng of the tile's top-left corner.
func TileXYToLatLng(tx, ty, zoom int) (lat, lng float64) {
	n := math.Pow(2, float64(zoom))
	lng = float64(tx)/n*360.0 - 180.0
	latRad := math.Atan(math.Sinh(math.Pi * (1 - 2*float64(ty)/n)))
	lat = latRad * 180.0 / math.Pi
	return lat, lng
}

// LatLngToPixel converts lat/lng to absolute pixel coordinates in the Mercator projection at given zoom.
// Pixel origin is at tile (0,0) top-left corner.
func LatLngToPixel(lat, lng float64, zoom int) (px, py int) {
	n := math.Pow(2, float64(zoom))
	px = int(math.Floor((lng + 180.0) / 360.0 * n * TileSize))
	latRad := lat * math.Pi / 180.0
	py = int(math.Floor((1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n * TileSize))
	return px, py
}

// PixelToCell converts an absolute Mercator pixel to a map-area terminal cell.
// originPx, originPy is the pixel coordinate of the top-left of the map area.
// Returns col, row in terminal cells (each cell = 1 col wide, 2 px tall due to half-block).
func PixelToCell(px, py, originPx, originPy int) (col, row int) {
	col = px - originPx
	row = (py - originPy) / 2
	return col, row
}

// TileOriginPixel returns the absolute pixel coordinate of the top-left corner of tile (tx, ty).
func TileOriginPixel(tx, ty int) (px, py int) {
	return tx * TileSize, ty * TileSize
}

// LatLngToPixelCoord is a convenience wrapper returning (px, py) as a pair.
func LatLngToPixelCoord(lat, lng float64, zoom int) (int, int) {
	return LatLngToPixel(lat, lng, zoom)
}

// ClampZoom clamps zoom to valid OSM range.
func ClampZoom(zoom int) int {
	if zoom < MinZoom {
		return MinZoom
	}
	if zoom > MaxZoom {
		return MaxZoom
	}
	return zoom
}

// TilesForView returns the range of tile indices needed to fill a terminal map area.
// centerLat/Lng is the center of the view.
// mapCols/mapRows is the size of the map area in terminal cells.
// Returns (minTX, minTY, maxTX, maxTY).
func TilesForView(centerLat, centerLng float64, zoom, mapCols, mapRows int) (minTX, minTY, maxTX, maxTY int) {
	// Map rows corresponds to half the pixel rows (half-block encoding: 2 px per row).
	mapPixelW := mapCols
	mapPixelH := mapRows * 2

	centerPX, centerPY := LatLngToPixel(centerLat, centerLng, zoom)

	// Top-left pixel of the map area.
	topLeftPX := centerPX - mapPixelW/2
	topLeftPY := centerPY - mapPixelH/2

	// Bottom-right pixel.
	bottomRightPX := topLeftPX + mapPixelW
	bottomRightPY := topLeftPY + mapPixelH

	minTX = int(math.Floor(float64(topLeftPX) / TileSize))
	minTY = int(math.Floor(float64(topLeftPY) / TileSize))
	maxTX = int(math.Floor(float64(bottomRightPX) / TileSize))
	maxTY = int(math.Floor(float64(bottomRightPY) / TileSize))

	return minTX, minTY, maxTX, maxTY
}

// MapOriginPixel returns the absolute pixel coordinate of the top-left of the visible map area.
func MapOriginPixel(centerLat, centerLng float64, zoom, mapCols, mapRows int) (originPX, originPY int) {
	mapPixelW := mapCols
	mapPixelH := mapRows * 2
	centerPX, centerPY := LatLngToPixel(centerLat, centerLng, zoom)
	originPX = centerPX - mapPixelW/2
	originPY = centerPY - mapPixelH/2
	return originPX, originPY
}
