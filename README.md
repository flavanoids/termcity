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

## Data Sources

| Source | Data | Notes |
|--------|------|-------|
| [OpenStreetMap](https://www.openstreetmap.org) | Map tiles | Max 2 req/s per ToS |
| [Nominatim](https://nominatim.org) | Zip → lat/lng geocoding | Max 1 req/s per ToS |
| [PulsePoint](https://www.pulsepoint.org) | Fire & EMS incidents | Unofficial API |
| [Socrata Open Data](https://dev.socrata.com) | Police incidents | City-specific datasets |

Incident data is sourced from public safety APIs and open government data portals. Coverage depends on your city's data sharing agreements.

## Architecture

Built in Go with the [Bubbletea](https://github.com/charmbracelet/bubbletea) TUI framework (MVU pattern).

```
termcity/
├── main.go
└── internal/
    ├── model/       # Bubbletea screen models (zip input → loading → map view)
    ├── data/        # API clients: geocoding, PulsePoint, Socrata, aggregator
    ├── tilemap/     # Tile fetching, disk cache, Web Mercator math, PNG → half-block render
    └── ui/          # Lipgloss styles, sidebar, status bar
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
