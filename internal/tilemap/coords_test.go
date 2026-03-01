package tilemap

import (
	"math"
	"testing"
)

func TestLatLngToTileXY_NYC(t *testing.T) {
	// Manhattan, zoom 14.
	tx, ty := LatLngToTileXY(40.7128, -74.0059, 14)
	// Expected tile (from OSM slippy map math).
	if tx != 4823 {
		t.Errorf("tx = %d, want 4823", tx)
	}
	if ty != 6160 {
		t.Errorf("ty = %d, want 6160", ty)
	}
}

func TestTileXYToLatLng_Roundtrip(t *testing.T) {
	zoom := 12
	origTX, origTY := 1205, 1539
	lat, lng := TileXYToLatLng(origTX, origTY, zoom)
	tx, ty := LatLngToTileXY(lat, lng, zoom)
	if tx != origTX {
		t.Errorf("roundtrip tx = %d, want %d", tx, origTX)
	}
	if ty != origTY {
		t.Errorf("roundtrip ty = %d, want %d", ty, origTY)
	}
}

func TestLatLngToPixel_Origin(t *testing.T) {
	// At zoom 0, the entire world is one 256x256 tile.
	// (0, 0) lat/lng should be near pixel (128, 128).
	px, py := LatLngToPixel(0, 0, 0)
	if px != 128 {
		t.Errorf("px = %d, want 128", px)
	}
	// Mercator y for 0 lat at zoom 0 should be 128.
	if math.Abs(float64(py-128)) > 2 {
		t.Errorf("py = %d, want ~128", py)
	}
}

func TestPixelToCell(t *testing.T) {
	// Origin at (100, 200), incident at (110, 210).
	// col = 110-100 = 10, row = (210-200)/2 = 5.
	col, row := PixelToCell(110, 210, 100, 200)
	if col != 10 {
		t.Errorf("col = %d, want 10", col)
	}
	if row != 5 {
		t.Errorf("row = %d, want 5", row)
	}
}

func TestTilesForView_Positive(t *testing.T) {
	minTX, minTY, maxTX, maxTY := TilesForView(40.7128, -74.0059, 14, 100, 50)
	if minTX > maxTX {
		t.Errorf("minTX %d > maxTX %d", minTX, maxTX)
	}
	if minTY > maxTY {
		t.Errorf("minTY %d > maxTY %d", minTY, maxTY)
	}
}

func TestClampZoom(t *testing.T) {
	if ClampZoom(0) != MinZoom {
		t.Errorf("ClampZoom(0) = %d, want %d", ClampZoom(0), MinZoom)
	}
	if ClampZoom(100) != MaxZoom {
		t.Errorf("ClampZoom(100) = %d, want %d", ClampZoom(100), MaxZoom)
	}
	if ClampZoom(14) != 14 {
		t.Errorf("ClampZoom(14) = %d, want 14", ClampZoom(14))
	}
}

func TestMapOriginPixel(t *testing.T) {
	// Center pixel should be centerPX - mapW/2.
	lat, lng := 40.7128, -74.0059
	zoom := 14
	mapCols, mapRows := 100, 50
	originPX, originPY := MapOriginPixel(lat, lng, zoom, mapCols, mapRows)
	centerPX, centerPY := LatLngToPixel(lat, lng, zoom)
	if originPX != centerPX-mapCols/2 {
		t.Errorf("originPX = %d, want %d", originPX, centerPX-mapCols/2)
	}
	if originPY != centerPY-mapRows {
		t.Errorf("originPY = %d, want %d", originPY, centerPY-mapRows)
	}
}
