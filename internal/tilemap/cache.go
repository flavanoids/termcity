package tilemap

import (
	"fmt"
	"os"
	"path/filepath"
)

// cacheDir returns the directory for tile caching.
func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".cache", "termcity", "tiles")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func sourceCachePrefix(src TileSource) string {
	switch src {
	case SourceDark:
		return "dark"
	case SourceLight:
		return "light"
	default:
		return "osm"
	}
}

// tileCachePath returns the file path for a cached tile.
func tileCachePath(src TileSource, z, x, y int) (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s_%d_%d_%d.png", sourceCachePrefix(src), z, x, y)
	return filepath.Join(dir, filename), nil
}

// ReadCachedTile reads a tile from disk cache. Returns nil, nil if not cached.
func ReadCachedTile(src TileSource, z, x, y int) ([]byte, error) {
	path, err := tileCachePath(src, z, x, y)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

// WriteCachedTile writes tile data to disk cache.
func WriteCachedTile(src TileSource, z, x, y int, data []byte) error {
	path, err := tileCachePath(src, z, x, y)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
