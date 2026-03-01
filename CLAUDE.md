# TermCity — Claude Code Instructions

## Project Overview
TermCity is a Go TUI application built with bubbletea/lipgloss. It lets users enter a US zip code and displays a full-screen OpenStreetMap tile map with active 911 incidents overlaid as pulsing colored dots, plus a sidebar incident list.

## Build & Run
```bash
go run .                          # Run the app
go build -o termcity .            # Build binary
go test ./internal/tilemap/...    # Unit tests (coordinate math)
go vet ./...                      # Static analysis
```

## Project Structure
```
termcity/
├── main.go                        # Entry point (tea.NewProgram)
├── go.mod / go.sum
├── internal/
│   ├── model/
│   │   ├── app.go                 # Root model, screen state machine
│   │   ├── zipinput.go            # Zip code entry screen
│   │   └── mapview.go             # Main map + sidebar + statusbar
│   ├── data/
│   │   ├── types.go               # Incident struct, IncidentType enum
│   │   ├── geocode.go             # Zip → lat/lng via Nominatim
│   │   ├── pulsepoint.go          # PulsePoint fire/EMS API client
│   │   ├── socrata.go             # City Socrata police API client
│   │   └── aggregator.go          # Merge/deduplicate sources
│   ├── tilemap/
│   │   ├── coords.go              # lat/lng ↔ tile ↔ pixel (Web Mercator)
│   │   ├── fetcher.go             # HTTP tile fetching + rate limiting
│   │   ├── cache.go               # Disk cache at ~/.cache/termcity/tiles/
│   │   └── render.go              # PNG → half-block terminal strings
│   └── ui/
│       ├── styles.go              # Lipgloss styles/colors
│       ├── sidebar.go             # Incident list panel + detail overlay
│       └── statusbar.go           # Bottom status bar + help overlay
```

## Architecture: Bubbletea Model-Message-Command

The app follows strict bubbletea MVU (Model-View-Update) pattern:

- **Model** (`internal/model/`) — immutable value types; NO mutexes (bubbletea is single-goroutine)
- **Messages** — `TileReadyMsg`, `IncidentsFetchedMsg`, `TickMsg`, `RefreshMsg`, `ZipSubmittedMsg`, `GeocodeDoneMsg`
- **Commands** — `fetchTileCmd`, `fetchIncidentsCmd`, `tickCmd`, `refreshCmd`, `geocodeCmd`

**Critical**: Never embed `sync.Mutex`/`sync.RWMutex` in model structs. Bubbletea copies the model by value on every Update call.

## Screen State Machine
```
ScreenZipInput → ScreenLoading → ScreenMap → (help overlay, not a separate screen)
```

## Key Rendering: Half-Block Technique
- OSM tiles: 256×256 px PNG
- Terminal cells: `▀` (U+2580) — top pixel = fg, bottom pixel = bg
- 256px tile → 128 terminal rows
- Incident markers use pulse animation: `pulseChars = []rune{' ', '·', '•', '●', '◉', '●', '•', '·'}`

## Data Sources
- **Geocoding**: Nominatim (1 req/s limit, User-Agent required)
- **Fire/EMS**: PulsePoint unofficial API (PSAP lookup → incident fetch)
- **Police**: Socrata open data portal (city registry in `socrata.go`)
- All sources fetched in parallel via goroutines; results returned as tea.Msg

## Rate Limits / ToS
- OSM tiles: max 2 req/s, `User-Agent: termcity/1.0`
- Nominatim: max 1 req/s, `User-Agent: termcity/1.0`
- PulsePoint: unofficial API — handle errors gracefully, don't abuse

## Tile Cache
Tiles are cached to `~/.cache/termcity/tiles/{z}_{x}_{y}.png`. Cache is checked before every HTTP fetch.

## Color Scheme
- Fire: `#FF4444` (red)
- Police: `#4488FF` (blue)
- EMS: `#EEEEEE` (white)
- Background: `#1a1a2e`, Sidebar: `#16213e`, Border: `#0f3460`

## Key Bindings (ScreenMap)
| Key | Action |
|-----|--------|
| `+` / `-` | Zoom in/out |
| Arrow keys | Pan map |
| `j` / `k` | Navigate incident list |
| `Tab` | Toggle sidebar |
| `Enter` | (reserved for detail view) |
| `r` | Refresh incidents |
| `?` | Toggle help overlay |
| `q` / `Ctrl+C` | Quit |

## Testing
```bash
go test ./internal/tilemap/... -v   # 7 coordinate math tests
go vet ./...                         # Must pass with 0 warnings
```

## Common Gotchas
1. **Tile seams**: stitch by column index, not string concat
2. **Incident off-map**: always bounds-check before writing to grid
3. **PulsePoint format**: unofficial API, format may change; decode errors should be non-fatal
4. **Half-block alignment**: tile height 256px = 128 rows (even, no padding needed)
5. **ANSI cell splitting**: `splitANSICells` scans for `▀` characters to extract individual cells
