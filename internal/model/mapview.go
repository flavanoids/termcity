package model

import (
	"strings"
	"termcity/internal/data"
	"termcity/internal/tilemap"
	"termcity/internal/ui"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// pulseChars is the animation sequence for incident markers.
var pulseChars = []rune{' ', '·', '•', '●', '◉', '●', '•', '·'}

// incidentFetchThreshold is the minimum movement (degrees) before re-fetching incidents on pan.
// ~0.05° ≈ 5.5 km at the equator.
const incidentFetchThreshold = 0.05

// MapViewModel manages the main map + sidebar + statusbar screen.
// NOTE: No sync.Mutex here — bubbletea Update/View run on a single goroutine.
// Tile commands send TileReadyMsg back through the message queue.
type MapViewModel struct {
	// Geography
	zip  string
	lat  float64
	lng  float64
	city string
	zoom int

	// Last position where incidents were fetched (used to detect significant pan).
	lastFetchLat float64
	lastFetchLng float64

	// Terminal dimensions
	width  int
	height int

	// Map state — safe to access without mutex (single-goroutine model)
	tileCache   map[tilemap.TileKey][]string
	incidents   []data.Incident
	warnings    []string
	loading     bool
	nextRefresh time.Time

	// Animation
	frame int

	// UI state
	selectedIncident int
	showSidebar      bool
	showHelp         bool

	// Error state
	err string
}

func NewMapViewModel(zip string, lat, lng float64, city string) MapViewModel {
	return MapViewModel{
		zip:          zip,
		lat:          lat,
		lng:          lng,
		city:         city,
		zoom:         14,
		tileCache:    make(map[tilemap.TileKey][]string),
		loading:      true,
		showSidebar:  true,
		nextRefresh:  time.Now().Add(60 * time.Second),
		lastFetchLat: lat,
		lastFetchLng: lng,
	}
}

// --- Messages ---

// TileReadyMsg signals a tile has been rendered.
type TileReadyMsg struct {
	Key  tilemap.TileKey
	Rows []string
	Err  error
}

// IncidentsFetchedMsg signals incidents have been loaded.
type IncidentsFetchedMsg struct {
	Incidents []data.Incident
	Warnings  []string
}

// RefreshMsg triggers a data refresh.
type RefreshMsg struct{}

// TickMsg drives animation.
type TickMsg time.Time

// --- Commands ---

func tickCmd() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func refreshCmd() tea.Cmd {
	return tea.Tick(60*time.Second, func(t time.Time) tea.Msg {
		return RefreshMsg{}
	})
}

func fetchTileCmd(z, x, y int) tea.Cmd {
	return func() tea.Msg {
		pngData, err := tilemap.FetchTile(z, x, y)
		if err != nil {
			return TileReadyMsg{Key: tilemap.TileKey{Z: z, X: x, Y: y}, Err: err}
		}
		rows, err := tilemap.RenderTile(pngData)
		return TileReadyMsg{
			Key:  tilemap.TileKey{Z: z, X: x, Y: y},
			Rows: rows,
			Err:  err,
		}
	}
}

func fetchIncidentsCmd(lat, lng float64, city string) tea.Cmd {
	return func() tea.Msg {
		incidents, warnings := data.FetchAllIncidents(lat, lng, city)
		return IncidentsFetchedMsg{Incidents: incidents, Warnings: warnings}
	}
}

func (m MapViewModel) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		refreshCmd(),
		fetchIncidentsCmd(m.lat, m.lng, m.city),
	)
}

func (m MapViewModel) Update(msg tea.Msg) (MapViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, m.fetchVisibleTiles()

	case TileReadyMsg:
		if msg.Err == nil {
			m.tileCache[msg.Key] = msg.Rows
		}
		return m, nil

	case IncidentsFetchedMsg:
		m.incidents = msg.Incidents
		m.warnings = msg.Warnings
		m.loading = false
		m.nextRefresh = time.Now().Add(60 * time.Second)
		m.lastFetchLat = m.lat
		m.lastFetchLng = m.lng
		return m, nil

	case RefreshMsg:
		m.loading = true
		return m, tea.Batch(
			fetchIncidentsCmd(m.lat, m.lng, m.city),
			refreshCmd(),
		)

	case TickMsg:
		m.frame = (m.frame + 1) % len(pulseChars)
		return m, tickCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m MapViewModel) handleKey(msg tea.KeyMsg) (MapViewModel, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "+", "=":
		m.zoom = tilemap.ClampZoom(m.zoom + 1)
		m.tileCache = make(map[tilemap.TileKey][]string)
		return m, m.fetchVisibleTiles()

	case "-":
		m.zoom = tilemap.ClampZoom(m.zoom - 1)
		m.tileCache = make(map[tilemap.TileKey][]string)
		return m, m.fetchVisibleTiles()

	case "up":
		m.lat += panDelta(m.zoom)
		return m.afterPan()

	case "down":
		m.lat -= panDelta(m.zoom)
		return m.afterPan()

	case "left":
		m.lng -= panDelta(m.zoom)
		return m.afterPan()

	case "right":
		m.lng += panDelta(m.zoom)
		return m.afterPan()

	case "j":
		// Navigate incident list.
		m.selectedIncident++
		if m.selectedIncident >= len(m.incidents) {
			m.selectedIncident = len(m.incidents) - 1
		}
		return m, nil

	case "k":
		// Navigate incident list.
		m.selectedIncident--
		if m.selectedIncident < 0 {
			m.selectedIncident = 0
		}
		return m, nil

	case "h":
		m.lng -= panDelta(m.zoom)
		return m.afterPan()

	case "l":
		m.lng += panDelta(m.zoom)
		return m.afterPan()

	case "tab":
		m.showSidebar = !m.showSidebar
		return m, m.fetchVisibleTiles()

	case "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "esc":
		m.showHelp = false
		return m, nil

	case "r":
		m.loading = true
		return m, fetchIncidentsCmd(m.lat, m.lng, m.city)
	}

	return m, nil
}

// shouldRefetchIncidents returns true if the map center has moved far enough
// from the last incident fetch position to warrant a new fetch.
func (m MapViewModel) shouldRefetchIncidents() bool {
	dlat := m.lat - m.lastFetchLat
	dlng := m.lng - m.lastFetchLng
	return dlat*dlat+dlng*dlng > incidentFetchThreshold*incidentFetchThreshold
}

// afterPan clears the tile cache, fetches visible tiles, and conditionally
// re-fetches incidents if the map center has moved past the threshold.
func (m MapViewModel) afterPan() (MapViewModel, tea.Cmd) {
	m.tileCache = make(map[tilemap.TileKey][]string)
	cmds := []tea.Cmd{m.fetchVisibleTiles()}
	if m.shouldRefetchIncidents() {
		m.loading = true
		m.lastFetchLat = m.lat
		m.lastFetchLng = m.lng
		cmds = append(cmds, fetchIncidentsCmd(m.lat, m.lng, m.city))
	}
	return m, tea.Batch(cmds...)
}

// fetchVisibleTiles returns commands to fetch all tiles currently in view.
func (m MapViewModel) fetchVisibleTiles() tea.Cmd {
	if m.width == 0 || m.height == 0 {
		return nil
	}

	mapCols, mapRows := m.mapDimensions()
	minTX, minTY, maxTX, maxTY := tilemap.TilesForView(m.lat, m.lng, m.zoom, mapCols, mapRows)

	var cmds []tea.Cmd
	for ty := minTY; ty <= maxTY; ty++ {
		for tx := minTX; tx <= maxTX; tx++ {
			key := tilemap.TileKey{Z: m.zoom, X: tx, Y: ty}
			if _, cached := m.tileCache[key]; !cached {
				cmds = append(cmds, fetchTileCmd(m.zoom, tx, ty))
			}
		}
	}
	return tea.Batch(cmds...)
}

// mapDimensions returns the width/height of the map area in terminal cells.
func (m MapViewModel) mapDimensions() (cols, rows int) {
	sidebarW := 0
	if m.showSidebar {
		sidebarW = ui.SidebarWidth + 1 // +1 for border
	}
	cols = m.width - sidebarW
	rows = m.height - 1 // -1 for status bar
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return cols, rows
}

func (m MapViewModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	mapCols, mapRows := m.mapDimensions()

	// Render map area.
	mapContent := m.renderMap(mapCols, mapRows)

	var view string
	if m.showSidebar {
		sidebar := ui.RenderSidebar(m.incidents, m.selectedIncident, mapRows)
		mapLines := strings.Split(mapContent, "\n")
		sidebarLines := strings.Split(sidebar, "\n")

		// Ensure equal line counts.
		for len(mapLines) < mapRows {
			mapLines = append(mapLines, strings.Repeat(" ", mapCols))
		}
		for len(sidebarLines) < mapRows {
			sidebarLines = append(sidebarLines, strings.Repeat(" ", ui.SidebarWidth))
		}

		var joined strings.Builder
		borderStyle := lipgloss.NewStyle().Foreground(ui.ColorBorder)
		for i := 0; i < mapRows; i++ {
			mapLine := mapLines[i]
			var sbLine string
			if i < len(sidebarLines) {
				sbLine = sidebarLines[i]
			}
			joined.WriteString(mapLine)
			joined.WriteString(borderStyle.Render("│"))
			joined.WriteString(sbLine)
			if i < mapRows-1 {
				joined.WriteString("\n")
			}
		}
		view = joined.String()
	} else {
		view = mapContent
	}

	statusBar := ui.RenderStatusBar(m.zip, m.nextRefresh, m.incidents, m.width, m.loading)
	result := view + "\n" + statusBar

	if m.showHelp {
		result = overlayHelp(result, m.width, m.height)
	}

	return result
}

// renderMap renders the tile map with incident markers overlaid.
func (m MapViewModel) renderMap(cols, rows int) string {
	mapPixelW := cols
	originPX, originPY := tilemap.MapOriginPixel(m.lat, m.lng, m.zoom, cols, rows)

	grid := make([][]string, rows)
	for r := range grid {
		grid[r] = make([]string, cols)
		for c := range grid[r] {
			grid[r][c] = " "
		}
	}

	minTX, minTY, maxTX, maxTY := tilemap.TilesForView(m.lat, m.lng, m.zoom, cols, rows)

	for ty := minTY; ty <= maxTY; ty++ {
		for tx := minTX; tx <= maxTX; tx++ {
			key := tilemap.TileKey{Z: m.zoom, X: tx, Y: ty}
			tileRows, ok := m.tileCache[key]
			if !ok {
				continue
			}

			tilePX := tx * tilemap.TileSize
			tilePY := ty * tilemap.TileSize
			tileOffsetCol := tilePX - originPX
			tileOffsetRow := (tilePY - originPY) / 2

			for tileRow, rowStr := range tileRows {
				gridRow := tileOffsetRow + tileRow
				if gridRow < 0 || gridRow >= len(grid) {
					continue
				}

				cells := splitANSICells(rowStr, tilemap.TileSize)
				for tileCol, cell := range cells {
					gridCol := tileOffsetCol + tileCol
					if gridCol < 0 || gridCol >= mapPixelW {
						continue
					}
					grid[gridRow][gridCol] = cell
				}
			}
		}
	}

	// Overlay incident markers.
	pulseChar := pulseChars[m.frame]
	for _, inc := range m.incidents {
		incPX, incPY := tilemap.LatLngToPixelCoord(inc.Lat, inc.Lng, m.zoom)
		col, row := tilemap.PixelToCell(incPX, incPY, originPX, originPY)
		if col < 0 || col >= cols || row < 0 || row >= rows {
			continue
		}
		grid[row][col] = tilemap.ColoredCell(pulseChar, inc.Type.Color(), 0, 0, 0)
	}

	var sb strings.Builder
	for r, rowCells := range grid {
		for _, cell := range rowCells {
			sb.WriteString(cell)
		}
		sb.WriteString("\x1b[0m")
		if r < len(grid)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// splitANSICells splits an ANSI tile row into per-cell strings by scanning for ▀.
func splitANSICells(row string, expectedCells int) []string {
	cells := make([]string, 0, expectedCells)
	runes := []rune(row)
	start := 0
	for i := 0; i < len(runes); i++ {
		if runes[i] == '▀' {
			cells = append(cells, string(runes[start:i+1]))
			start = i + 1
		}
	}
	for len(cells) < expectedCells {
		cells = append(cells, " ")
	}
	return cells
}

// panDelta returns degrees to pan per keypress at the given zoom level.
func panDelta(zoom int) float64 {
	return 0.15 / float64(zoom)
}

// overlayHelp overlays the help box on the existing rendered output.
func overlayHelp(base string, width, height int) string {
	helpOverlay := ui.RenderHelpOverlay(width, height)
	baseLines := strings.Split(base, "\n")
	helpLines := strings.Split(helpOverlay, "\n")

	out := make([]string, len(baseLines))
	copy(out, baseLines)

	for i, line := range helpLines {
		if i >= len(out) {
			break
		}
		if strings.TrimSpace(line) != "" {
			out[i] = line
		}
	}
	return strings.Join(out, "\n")
}

// LoadingView returns a centered loading message.
func LoadingView(zip, msg string, width, height int) string {
	content := lipgloss.JoinVertical(lipgloss.Center,
		ui.ZipTitleStyle.Render("TermCity"),
		"",
		ui.LoadingStyle.Render("⟳ "+msg),
		"",
		ui.HelpStyle.Render("ZIP: "+zip),
	)

	contentLines := strings.Count(content, "\n") + 1
	topPad := (height - contentLines) / 2
	if topPad < 0 {
		topPad = 0
	}

	return strings.Repeat("\n", topPad) +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, content)
}

// ErrorView returns a centered error message.
func ErrorView(msg string, width, height int) string {
	content := lipgloss.JoinVertical(lipgloss.Center,
		ui.ErrorStyle.Render("Error"),
		"",
		ui.HelpStyle.Render(msg),
		"",
		ui.HelpStyle.Render("Press q to quit"),
	)

	contentLines := strings.Count(content, "\n") + 1
	topPad := (height - contentLines) / 2
	if topPad < 0 {
		topPad = 0
	}

	return strings.Repeat("\n", topPad) +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, content)
}
