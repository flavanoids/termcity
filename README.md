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
- **Time-based fading** — Older incidents fade automatically (new → recent → stale → old)
- **Marker clustering** — When zoomed out, nearby incidents group into numbered clusters
- **Incident type filters** — Toggle Fire (`f`), Police (`p`), EMS (`e`) on/off
- **Incident sidebar** — Scrollable list with address, type, and timestamp
- **Detail overlay** — Select any incident for responding unit details
- **Recent zips** — Quick access to last 5 locations on startup
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

A browser-based server with the same functionality (geocode by ZIP, OSM map, incidents, sidebar, detail modal). It uses the same data layer and is optimized for desktop and mobile. Includes SQLite-backed incident history (7-day retention).

### Quick Start

```bash
go build -o termcity-web ./cmd/termcity-web
./termcity-web -port 8911 -foreground 10001
```

Then open http://localhost:8911 in your browser.

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `-port N` | 8911 | HTTP server port |
| `-foreground` | false | Run in foreground (logs to stdout) |

### Production Deployment (systemd)

**1. Install the binary:**

```bash
go build -o termcity-web ./cmd/termcity-web
sudo cp termcity-web /usr/local/bin/
sudo chmod +x /usr/local/bin/termcity-web
```

**2. Create system user:**

```bash
sudo useradd --system --home /var/lib/termcity --create-home --shell /usr/sbin/nologin termcity
```

**3. Create systemd service:** `/etc/systemd/system/termcity-web.service`

```ini
[Unit]
Description=TermCity Web — 911 incident map viewer
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=termcity
Group=termcity
WorkingDirectory=/var/lib/termcity
Environment="TERMCITY_ZIP=10001"
Environment="TERMCITY_PORT=8911"
ExecStart=/usr/local/bin/termcity-web -port ${TERMCITY_PORT} ${TERMCITY_ZIP}
Restart=on-failure
RestartSec=10
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/termcity
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

**4. Enable and start:**

```bash
sudo systemctl daemon-reload
sudo systemctl enable termcity-web
sudo systemctl start termcity-web
```

**Management:**

```bash
sudo systemctl status termcity-web    # Check status
sudo journalctl -u termcity-web -f    # View logs
sudo systemctl restart termcity-web   # Restart
```

Change the zip code by editing `Environment="TERMCITY_ZIP=..."` in the service file, then `systemctl restart termcity-web`.

### Build .deb Package

```bash
./scripts/build-deb-web.sh
sudo dpkg -i termcity-web_1.0.0_amd64.deb
sudo systemctl enable --now termcity-web
```

### API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /` | Web UI (HTML) |
| `GET /api/incidents` | Live incidents with metadata |
| `GET /api/history?days=N` | Historical incidents (1, 3, or 7 days) |
| `POST /api/history/clear` | Clear history database |
| `GET /api/status` | Server status and history counts |

## Key Bindings

| Key | Action |
|-----|--------|
| `+` / `-` | Zoom in / out |
| Mouse wheel | Zoom in / out |
| Arrow keys | Pan map |
| `j` / `k` | Scroll incident list |
| `Tab` | Toggle sidebar focus |
| `Enter` | Show incident detail / select |
| `1`–`9` | Jump to incident # and show detail |
| `f` | Toggle Fire incidents filter |
| `p` | Toggle Police incidents filter |
| `e` | Toggle EMS incidents filter |
| `m` | Cycle map style (OSM / Dark / Light) |
| `r` | Refresh incidents |
| `?` | Toggle help overlay |
| `Esc` | Return focus to map / close overlay |
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
