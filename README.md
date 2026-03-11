# TermCity

A terminal-based 911 incident viewer. Enter a US zip code and see live fire, EMS, and police incidents overlaid on an OpenStreetMap tile map — rendered entirely in your terminal using Unicode half-block characters and 24-bit color.

```
╔══════════════════════════════════════════════════════════╗
║  [map tiles rendered with ▀ half-block characters]       ║  Incidents (12)
║                                                          ║  ─────────────
║        ●  Fire - Structure Fire                          ║  🔴 Structure Fire
║                                                          ║  🔵 Theft Report
║             ◉  EMS - Medical Emergency                   ║  ⚪ Medical Emergency
║                                                          ║  ...
║  ●  Police - Traffic Stop                                ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝
  Downtown Austin, TX  |  Zoom 14  |  12 incidents   [?] help
```

## Why

For people who live near emergency activity and want to know what's happening around them — without a browser, ads, or a smartphone. TermCity gives you a calm, accurate picture of nearby emergency activity. No sensationalism, no surveillance.

## Features

- **Live map** — OpenStreetMap tiles rendered with half-block Unicode, pan and zoom freely
- **Real-time incidents** — Fire, EMS, and police incidents fetched from public APIs
- **Pulsing markers** — Animated incident dots with color-coded type (red/white/blue)
- **Incident sidebar** — Scrollable list with address, type, and timestamp
- **Detail overlay** — Select any incident for responding unit details
- **Disk tile cache** — Tiles cached at `~/.cache/termcity/tiles/` to reduce network load
- **Rate-limited** — Respects OSM and Nominatim ToS out of the box

## Requirements

- Go 1.24+
- A terminal with 24-bit color and Unicode support (iTerm2, kitty, Alacritty, Windows Terminal, etc.)

## Install & Run

```bash
git clone https://github.com/your-username/termcity
cd termcity
go run .
```

Or build a binary:

```bash
go build -o termcity .
./termcity
```

On launch, enter any US zip code to center the map on that location.

## Web version

A separate browser-based binary with the same functionality (geocode by ZIP, OSM map, incidents, sidebar, detail modal). It uses the same data layer and is optimized for desktop and mobile.

**Run (separate binary, background by default):**

```bash
go build -o termcity-web ./cmd/termcity-web
./termcity-web
```

The server starts in the background and prints the URL and port, for example:

```
TermCity web server is running in the background.
Open in your browser: http://localhost:8080
Port: 8080 (stop with: kill <pid>)
```

Then open the printed URL in your browser. Enter a US zip code to load the map and incidents. Use the map style dropdown (OSM / Dark / Light), the Refresh button or <kbd>r</kbd>, and click markers or list items for details. Keyboard: <kbd>1</kbd>–<kbd>9</kbd> jump to incident and show detail; <kbd>?</kbd> opens help.

**Options:** `-port 8080` (default; overridden by `PORT` env), `-foreground` to run in the foreground (e.g. under systemd or to see logs).

**Install via .deb (Linux amd64):**

```bash
./scripts/build-deb-web.sh
sudo dpkg -i termcity-web_1.0.0_amd64.deb
```

Then run `termcity-web`; it will start in the background and print the URL and port.

## Key Bindings

| Key | Action |
|-----|--------|
| `+` / `-` | Zoom in / out |
| Arrow keys | Pan map |
| `j` / `k` | Scroll incident list |
| `Tab` | Toggle sidebar |
| `r` | Refresh incidents |
| `?` | Toggle help overlay |
| `q` / `Ctrl+C` | Quit |

## Coverage

> **Currently focused on Houston, TX.** Incident data is actively tested and validated against Houston's live feed. Other US cities may work partially via PulsePoint (fire/EMS) and Socrata (police) where those cities publish open data, but coverage is not guaranteed.

## Data Sources

| Source | Data | Coverage |
|--------|------|----------|
| [OpenStreetMap](https://www.openstreetmap.org) | Map tiles | Worldwide |
| [Nominatim](https://nominatim.org) | Zip → lat/lng geocoding | US zip codes |
| [City of Houston](https://cohweb.houstontx.gov/ActiveIncidents/Combined.aspx) | HFD & HPD active incidents | Houston, TX only |
| [PulsePoint](https://www.pulsepoint.org) | Fire & EMS incidents | Select US cities (unofficial API) |
| [Socrata Open Data](https://dev.socrata.com) | Police incidents | Select US cities |

Incident data is sourced from public safety APIs and open government data portals. Rate limits are respected per each source's ToS.

## Architecture

Built in Go with the [Bubbletea](https://github.com/charmbracelet/bubbletea) TUI framework (MVU pattern).

```
termcity/
├── main.go              # TUI entry point
├── cmd/termcity-web/    # Web server: API (geocode, incidents) + embedded static frontend
│   ├── main.go
│   └── static/          # HTML, CSS, JS (Leaflet map, Bootstrap UI)
└── internal/
    ├── model/       # Bubbletea screen models (zip input → loading → map view)
    ├── data/        # API clients: geocoding, PulsePoint, Socrata, aggregator (shared by TUI and web)
    ├── tilemap/     # Tile fetching, disk cache, Web Mercator math, PNG → half-block render (TUI only)
    └── ui/          # Lipgloss styles, sidebar, status bar (TUI only)
```

Map tiles (256×256 PNG) are decoded pixel-by-pixel and converted to terminal strings using `▀` (U+2580): each cell encodes two vertical pixels via foreground and background 24-bit colors. Incident data is fetched in parallel from all sources and merged before display.

## Development

```bash
go test ./internal/tilemap/...   # Coordinate math unit tests
go vet ./...                     # Static analysis (must pass clean)
```

## Disclaimer

TermCity displays public safety data from publicly available APIs. It is intended for informational awareness only — not for emergency response coordination. Always call 911 in an emergency.

PulsePoint integration uses an unofficial API and may break without notice. Police incident coverage varies by city.
