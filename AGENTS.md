# TermCity — Agent Guidelines

This document describes how AI agents should work with the TermCity codebase.

## Project Identity
TermCity is a terminal-based 911 incident viewer. It fetches real-time emergency dispatch data and renders it on an OpenStreetMap tile map directly in the terminal using Unicode half-block characters and 24-bit ANSI color.

## Agent Workflow

### Before Making Changes
1. Run `go build ./...` — must succeed with zero errors
2. Run `go vet ./...` — must pass with zero warnings
3. Run `go test ./internal/tilemap/...` — all 7 tests must pass
4. Read `CLAUDE.md` for architecture constraints

### Package Responsibilities
| Package | Responsibility | Key Constraint |
|---------|---------------|----------------|
| `internal/data` | API clients + types | Handle all network errors gracefully; never panic on bad JSON |
| `internal/tilemap` | Map math + tile I/O | Pure functions where possible; coordinate math is unit-tested |
| `internal/ui` | Lipgloss rendering | Functions only (no state); return strings, don't print |
| `internal/model` | Bubbletea MVU | No mutexes in model structs; all state changes via Update() |

### Code Style
- Go standard formatting (`gofmt`)
- No third-party dependencies beyond the declared ones in `go.mod`
- Error handling: wrap errors with `fmt.Errorf("context: %w", err)`
- Prefer explicit returns over early returns with side effects

### Adding a New Data Source
1. Create `internal/data/{source}.go` with a `Fetch{Source}Incidents(lat, lng float64) ([]Incident, error)` function
2. Add it to `aggregator.go`'s goroutine fan-out pattern
3. Use the existing `Incident` struct — do not add new fields without updating all consumers

### Adding a New Screen
1. Add a new `Screen` constant in `internal/model/app.go`
2. Create `internal/model/{screen}.go` with `Model`, `Init()`, `Update()`, `View()` methods
3. Add routing in `AppModel.Update()` and `AppModel.View()`
4. Define the new message type in the new file, not in `app.go`

### Tile Rendering Changes
- The `RenderTile()` function in `render.go` must always return exactly `h/2` strings (128 for standard OSM tiles)
- The `splitANSICells()` function in `mapview.go` must always return exactly `expectedCells` entries
- Half-block encoding: `▀` fg=top pixel, bg=bottom pixel — do not change this convention

### API Rate Limiting
All HTTP clients must respect these limits:
- OSM tiles: 500ms between requests (`fetcher.go`)
- Nominatim geocoding: 1000ms between requests (`geocode.go`)
- PulsePoint: no explicit rate limit enforced; don't parallelize

### Testing New Coordinate Math
Add test cases to `internal/tilemap/coords_test.go`. Verify against known OSM tile coordinates for landmark locations (e.g., Manhattan = tile 4823,6160 at zoom 14).

## What Agents Should NOT Do
- Do not add dependencies to `go.mod` without justification
- Do not embed `sync.Mutex` or `sync.RWMutex` in any model struct
- Do not use `fmt.Println` / `log.Print` in TUI code (bubbletea captures stdout)
- Do not make unbounded concurrent HTTP requests (all tiles are fetched serially per rate limit)
- Do not modify `go.sum` manually
- Do not create new files unless necessary — prefer editing existing ones

## Debugging Tips
- Use `tea.LogToFile("debug.log", "")` in `main.go` to capture bubbletea logs without corrupting TUI
- The tile cache at `~/.cache/termcity/tiles/` can be cleared to force fresh tile fetches
- PulsePoint returns empty `active` array (not an error) when no incidents are active
