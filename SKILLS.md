# TermCity — Skills & Capabilities Reference

This document catalogs the specific technical skills demonstrated in this codebase, useful for onboarding contributors and guiding AI agents.

## Skill: Web Mercator Coordinate Math (`internal/tilemap/coords.go`)

### What It Does
Converts between geographic coordinates (lat/lng), OSM tile indices (tx, ty, zoom), absolute Mercator pixels, and terminal cell positions.

### Key Formulas
```go
// Tile from lat/lng
tx = floor((lng + 180) / 360 * 2^zoom)
ty = floor((1 - log(tan(lat_rad) + sec(lat_rad)) / π) / 2 * 2^zoom)

// Absolute pixel from lat/lng (same formula, multiplied by TileSize=256)
px = floor((lng + 180) / 360 * 2^zoom * 256)
py = floor((1 - log(tan(lat_rad) + sec(lat_rad)) / π) / 2 * 2^zoom * 256)

// Terminal cell from pixel (half-block: 2 pixels per row)
col = px - originPX
row = (py - originPY) / 2
```

### When to Apply
- Adding a new map layer or marker type
- Implementing viewport pan/zoom
- Converting incident lat/lng to screen position

---

## Skill: PNG Half-Block Terminal Rendering (`internal/tilemap/render.go`)

### What It Does
Converts a 256×256 PNG tile into 128 terminal rows where each cell uses `▀` (U+2580) with 24-bit ANSI foreground/background colors to encode 2 vertical pixels.

### ANSI Sequence per Cell
```
\x1b[38;2;{R};{G};{B}m    ← foreground (top pixel)
\x1b[48;2;{R};{G};{B}m    ← background (bottom pixel)
▀                          ← upper half block
```
Terminal row ends with `\x1b[0m` to reset attributes.

### When to Apply
- Changing zoom rendering resolution
- Adding overlay layers (buildings, traffic)
- Implementing tile blending or transparency

---

## Skill: Bubbletea Command Fan-Out (`internal/data/aggregator.go`)

### What It Does
Launches multiple data fetches concurrently (PulsePoint + Socrata) via goroutines, collects results into a channel, then returns a single `IncidentsFetchedMsg`.

### Pattern
```go
ch := make(chan result, N)
go func() { ch <- fetch1() }()
go func() { ch <- fetch2() }()
for i := 0; i < N; i++ {
    r := <-ch
    // aggregate
}
```

### When to Apply
- Adding a new data source (add another goroutine + channel slot)
- Implementing timeout on data fetches
- Adding retry logic per source

---

## Skill: Disk Tile Cache (`internal/tilemap/cache.go`)

### What It Does
Reads/writes OSM tile PNG files to `~/.cache/termcity/tiles/{z}_{x}_{y}.png`. Cache misses fall through to HTTP fetch. Cache writes are best-effort.

### File Path Convention
```
~/.cache/termcity/tiles/{zoom}_{tileX}_{tileY}.png
```

### When to Apply
- Implementing cache eviction (add LRU or mtime check)
- Adding a "refresh tiles" command
- Porting to a new platform with different cache paths

---

## Skill: OSM Tile HTTP Fetch with Rate Limiting (`internal/tilemap/fetcher.go`)

### What It Does
Fetches tiles from `https://tile.openstreetmap.org/{z}/{x}/{y}.png` with:
- Global mutex-protected rate limiter (500ms gap = 2 req/s max)
- Correct `User-Agent` header (OSM ToS requirement)
- 10-second timeout

### When to Apply
- Adding alternative tile servers (e.g., Stadia, Carto)
- Implementing parallel tile fetching (increase rate limit carefully)
- Adding retry on HTTP 429/503

---

## Skill: Nominatim Geocoding (`internal/data/geocode.go`)

### What It Does
Converts a US zip code string to `(lat, lng, city, state)` via the Nominatim OSM geocoding API.

### API
```
GET https://nominatim.openstreetmap.org/search
  ?q={zip}&format=json&countrycodes=us&limit=1&addressdetails=1
```

### When to Apply
- Supporting city name input instead of zip
- Adding reverse geocoding (pixel → address)
- Caching geocode results

---

## Skill: PulsePoint Incident Parsing (`internal/data/pulsepoint.go`)

### What It Does
Two-step API call: (1) PSAP lookup by lat/lng → (2) active incidents by PSAP ID. Maps call type codes (`ME`, `F`, `TE`, etc.) to `IncidentType`.

### Call Type Mapping
| Code | Meaning | IncidentType |
|------|---------|-------------|
| `F`, `FS`, `FR`, `FW`, `FA`, `FB` | Fire variants | Fire |
| `ME`, `MA`, `MCI`, `MC` | Medical variants | EMS |
| Others (`TE`, `TC`, `HZ`, `RS`) | Traffic/Hazmat/Rescue | EMS |

### When to Apply
- Adding new call type codes
- Parsing unit status timestamps
- Supporting multiple PSAPs per area

---

## Skill: Socrata Police Data (`internal/data/socrata.go`)

### What It Does
Queries city-specific Socrata open data portals for recent police incidents. Maintains a registry of supported cities with their dataset IDs and field name mappings.

### Adding a City
1. Look up the dataset on `data.cityname.gov`
2. Identify field names for: date, lat, lng, type, address
3. Add an entry to `socrataRegistry` in `socrata.go`

### When to Apply
- Adding support for a new city
- Handling Socrata API version changes
- Adding OAuth for premium datasets

---

## Skill: Lipgloss Sidebar Rendering (`internal/ui/sidebar.go`)

### What It Does
Renders a fixed-width (26 cols) incident list with:
- Colored bullet symbols per incident type
- Scrolling to keep selected item visible
- Time-ago formatting
- Detail overlay for selected incident

### When to Apply
- Adding incident filtering by type
- Implementing search within incident list
- Adding keyboard-driven selection with `j`/`k`

---

## Skill: ANSI Pulse Animation (`internal/model/mapview.go`)

### What It Does
Uses a `TickMsg` (300ms interval) to increment `m.frame`, cycling through `pulseChars` to animate incident markers.

```go
var pulseChars = []rune{' ', '·', '•', '●', '◉', '●', '•', '·'}
// frame 0→1→2→3→4→5→6→7→0 ...
```

### When to Apply
- Adding new animation styles per incident type
- Implementing blinking (alternate char with space)
- Adjusting animation speed
