package tilemap

import (
	"container/list"
	"sync"
)

// MemCache is a process-lifetime LRU cache for rendered tile cells.
// It lives outside the bubbletea model so a mutex is safe here.
type MemCache struct {
	mu       sync.Mutex
	entries  map[TileKey]*list.Element
	order    *list.List
	capacity int
}

type memEntry struct {
	key  TileKey
	rows [][]string
}

// GlobalMemCache is shared across all fetchTileCmd goroutines.
var GlobalMemCache = NewMemCache(200)

// NewMemCache creates an LRU cache with the given tile capacity.
func NewMemCache(capacity int) *MemCache {
	return &MemCache{
		entries:  make(map[TileKey]*list.Element),
		order:    list.New(),
		capacity: capacity,
	}
}

// Get returns cached pre-split cells for the given tile key, or (nil, false).
func (c *MemCache) Get(key TileKey) ([][]string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(el)
	return el.Value.(*memEntry).rows, true
}

// Put stores pre-split cells for the given tile key, evicting LRU if at capacity.
func (c *MemCache) Put(key TileKey, rows [][]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.entries[key]; ok {
		c.order.MoveToFront(el)
		el.Value.(*memEntry).rows = rows
		return
	}
	if c.order.Len() >= c.capacity {
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			delete(c.entries, oldest.Value.(*memEntry).key)
		}
	}
	el := c.order.PushFront(&memEntry{key: key, rows: rows})
	c.entries[key] = el
}
