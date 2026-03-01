# TermCity — Development Rules

## Non-Negotiable Rules

### 1. Zero Build Errors, Zero Vet Warnings
Every commit must pass:
```bash
go build ./... && go vet ./...
```
No exceptions. A warning is a build failure.

### 2. No Mutex in Model Structs
Bubbletea passes model by value on every `Update()`. Embedding `sync.Mutex` or `sync.RWMutex` directly in a model struct causes data races and `go vet` failures. Use pointer-based external stores if shared mutable state is truly needed (it isn't in this codebase).

### 3. No stdout/stderr in TUI Code
Bubbletea owns the terminal. Any `fmt.Println`, `log.Print`, or `os.Stderr.Write` will corrupt the display. Use `tea.LogToFile()` for debugging.

### 4. Rate Limit All External HTTP
| Endpoint | Limit | Enforced In |
|----------|-------|-------------|
| OSM tiles | 2 req/s (500ms gap) | `tilemap/fetcher.go` |
| Nominatim | 1 req/s (1000ms gap) | `data/geocode.go` |
| PulsePoint | Don't parallelize | `data/pulsepoint.go` |

Violating OSM's tile usage policy risks IP bans that affect all users.

### 5. Graceful API Error Handling
PulsePoint is an unofficial API. Socrata datasets may change schemas. All data fetching must:
- Return `([]Incident, error)` — never panic on bad API responses
- Return `nil, nil` when no data is available (not an error condition)
- Wrap errors with context: `fmt.Errorf("pulsepoint: %w", err)`

### 6. Coordinate Math Must Be Tested
Any changes to `internal/tilemap/coords.go` require corresponding test coverage in `coords_test.go`. The coordinate math is the foundation of correct map rendering.

### 7. Respect User-Agent Requirements
Both OSM and Nominatim ToS require a descriptive `User-Agent` header. Always use:
```
User-Agent: termcity/1.0 (github.com/termcity)
```

### 8. Tile Cache Is Best-Effort
Cache writes (`WriteCachedTile`) must be fire-and-forget — never block the UI on disk I/O. Cache read errors should fall through to a fresh HTTP fetch.

## Architecture Rules

### Bubbletea Model Pattern
```go
// CORRECT: model methods have value receivers, return (Model, tea.Cmd)
func (m MyModel) Update(msg tea.Msg) (MyModel, tea.Cmd) { ... }

// WRONG: pointer receivers break bubbletea's immutable update semantics
func (m *MyModel) Update(msg tea.Msg) (MyModel, tea.Cmd) { ... }
```

### Command Pattern
All I/O (HTTP, disk) happens inside `tea.Cmd` goroutines. Results come back as messages. The model never performs I/O directly.

```go
// CORRECT
func fetchDataCmd() tea.Cmd {
    return func() tea.Msg {
        data, err := http.Get(...)
        return DataFetchedMsg{Data: data, Err: err}
    }
}

// WRONG: I/O in Update()
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
    data, _ := http.Get(...)  // NEVER do this
}
```

### Rendering Rules
- `View()` must be pure — no side effects, no I/O, no randomness
- Animation frame (`m.frame`) is the only stateful element permitted in `View()`
- All lipgloss styles are defined in `internal/ui/styles.go`, not inline

## Security Rules
- No user input is ever passed to shell commands, SQL, or file paths without sanitization
- Zip code input is validated to be exactly 5 ASCII digits before any network call
- All URL construction uses `fmt.Sprintf` or `url.Values` — no string concatenation with user input
- Tile cache paths are derived from integer tile coordinates only — no user-controlled path components

## Dependency Rules
- Only `charmbracelet/bubbletea`, `charmbracelet/lipgloss`, `charmbracelet/bubbles` are permitted as direct dependencies
- All other functionality uses Go stdlib (`net/http`, `image/png`, `math`, `encoding/json`)
- No indirect dependencies should be added manually
