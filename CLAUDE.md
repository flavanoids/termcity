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
├── main.go                        # TUI entry point (tea.NewProgram)
├── cmd/termcity-web/              # Web server binary
│   ├── main.go                    # Web server entry
│   ├── server.go                  # HTTP handlers
│   ├── daemon_unix.go             # Unix session detachment
│   ├── daemon_windows.go          # Windows service support
│   ├── index.html                 # Embedded web UI
│   └── static/                    # Static assets
├── internal/
│   ├── model/
│   │   ├── app.go                 # Root model, screen state machine
│   │   ├── zipinput.go            # Zip code entry screen (+ recent zips)
│   │   └── mapview.go             # Main map + sidebar + statusbar
│   ├── data/
│   │   ├── types.go               # Incident struct, IncidentType enum
│   │   ├── geocode.go             # Zip → lat/lng via Nominatim
│   │   ├── pulsepoint.go          # PulsePoint fire/EMS API client
│   │   ├── socrata.go             # City Socrata police API client
│   │   ├── aggregator.go          # Merge/deduplicate sources
│   │   ├── validation.go          # Freshness classification
│   │   └── enrich.go              # Timezone normalization
│   ├── tilemap/
│   │   ├── coords.go              # lat/lng ↔ tile ↔ pixel (Web Mercator)
│   │   ├── fetcher.go             # HTTP tile fetching + rate limiting
│   │   ├── cache.go               # Disk cache at ~/.cache/termcity/tiles/
│   │   ├── memcache.go            # In-memory tile cache
│   │   ├── render.go              # PNG → half-block terminal strings
│   │   └── cluster.go             # Incident clustering for low zoom
│   ├── ui/
│   │   ├── styles.go              # Lipgloss styles/colors
│   │   ├── sidebar.go             # Incident list panel + detail overlay
│   │   └── statusbar.go           # Bottom status bar + help overlay
│   └── history/
│       └── store.go               # SQLite persistence for incident history
└── packaging/                     # .deb packaging files
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

## Caching & Persistence

### Tile Cache
Tiles are cached to `~/.cache/termcity/tiles/{z}_{x}_{y}.png`. Cache is checked before every HTTP fetch.

### Incident History (Web)
The web version stores incidents to `~/.cache/termcity/history.db` (SQLite with WAL mode). History is retained for 7 days and accessible via `/api/history` endpoints.

## Color Scheme
- Fire: `#FF4444` (red)
- Police: `#4488FF` (blue)
- EMS: `#EEEEEE` (white)
- Background: `#1a1a2e`, Sidebar: `#16213e`, Border: `#0f3460`
- Highlight: `#e94560` (accent)

## Key Bindings (ScreenMap)
| Key | Action |
|-----|--------|
| `+` / `-` | Zoom in/out |
| Mouse wheel | Zoom in/out |
| Arrow keys | Pan map (map focus) / Navigate list (sidebar focus) |
| `h` / `l` | Pan west/east (vim-style) |
| `j` / `k` | Navigate incident list (any focus) |
| `Tab` | Switch focus between map and sidebar |
| `Enter` | Show incident detail / select from recent zips |
| `1`–`9` | Jump to incident # and show detail |
| `f` | Toggle Fire incidents filter |
| `p` | Toggle Police incidents filter |
| `e` | Toggle EMS incidents filter |
| `m` | Cycle map style (OSM / Dark / Light) |
| `Esc` | Return focus to map / close overlay |
| `r` | Refresh incidents |
| `?` | Toggle help overlay |
| `q` / `Ctrl+C` | Quit |

## Visual Features

### Marker Clustering
At zoom levels < 13, nearby incidents cluster into numbered markers showing the count. Click to zoom in and see individual incidents.

### Time-Based Fading
Incident brightness reflects age:
- **New** (< 15 min): Full brightness with pulse
- **Recent** (15–60 min): 50–90% brightness  
- **Stale** (1–4 hours): 30–60% brightness
- **Old** (> 4 hours): 15–30% brightness (barely visible)

### Incident Type Filters
Toggle visibility per type with `f`/`p`/`e`. Status bar shows `●` (visible) or `○` (filtered) for each type. Applies to both map and sidebar.

### Recent Zips
On the zip entry screen, shows last 5 locations. Press `1`–`5` to quickly reselect.

## Web Version Deployment

### Building
```bash
go build -o termcity-web ./cmd/termcity-web   # Build binary
./termcity-web -port 8911 -foreground 10001   # Run in foreground
```

### Systemd Service
Service file: `/etc/systemd/system/termcity-web.service`

```ini
[Unit]
Description=TermCity Web — 911 incident map viewer
After=network-online.target

[Service]
Type=simple
User=termcity
Group=termcity
WorkingDirectory=/var/lib/termcity
Environment="TERMCITY_ZIP=10001"
Environment="TERMCITY_PORT=8911"
ExecStart=/usr/local/bin/termcity-web -port ${TERMCITY_PORT} ${TERMCITY_ZIP}
Restart=on-failure
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=/var/lib/termcity

[Install]
WantedBy=multi-user.target
```

### Commands
```bash
sudo systemctl enable termcity-web   # Enable on boot
sudo systemctl start termcity-web    # Start service
sudo systemctl status termcity-web   # Check status
sudo journalctl -u termcity-web -f    # View logs
```

### API Endpoints
- `GET /` — Web UI
- `GET /api/incidents` — Live incidents
- `GET /api/history?days=N` — Historical (1, 3, or 7 days)
- `POST /api/history/clear` — Clear history
- `GET /api/status` — Server status

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
