package model

import (
	"fmt"
	"strconv"
	"strings"
	"termcity/internal/data"
	"termcity/internal/tilemap"
	"termcity/internal/ui"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FocusArea indicates which panel has keyboard focus.
type FocusArea int

const (
	FocusMap FocusArea = iota
	FocusSidebar
)

// pulseChars is the animation sequence for incident markers.
var pulseChars = []rune{' ', '·', '•', '●', '◉', '●', '•', '·'}

// incidentFetchThreshold is the minimum movement (degrees) before re-fetching incidents on pan.
// ~0.05° ≈ 5.5 km at the equator.
const incidentFetchThreshold = 0.05

// numberBufTimeout is how long to wait after the last digit key before
// auto-opening the detail overlay for the typed incident number.
const numberBufTimeout = 800 * time.Millisecond

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

	// Map style
	tileSource tilemap.TileSource

	// Map state — safe to access without mutex (single-goroutine model)
	tileCache map[tilemap.TileKey][][]string
	incidents []data.Incident
	// validation holds derived validation/enrichment info per incident index.
	validation  []ui.IncidentValidation
	warnings    []string
	loading     bool
	nextRefresh time.Time

	// Animation
	frame int

	// UI state
	focus            FocusArea
	selectedIncident int
	showHelp         bool
	showDetail       bool

	// Number-key quick-select buffer (e.g. press "1" "2" → incident #12).
	numberBuf   string
	numberBufAt time.Time

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
		tileSource:   tilemap.SourceDark,
		tileCache:    make(map[tilemap.TileKey][][]string),
		loading:      true,
		focus:        FocusMap,
		nextRefresh:  time.Now().Add(60 * time.Second),
		lastFetchLat: lat,
		lastFetchLng: lng,
	}
}

// --- Messages ---

// TileReadyMsg signals a tile has been rendered.
type TileReadyMsg struct {
	Key  tilemap.TileKey
	Rows [][]string
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

func fetchTileCmd(src tilemap.TileSource, z, x, y int) tea.Cmd {
	return func() tea.Msg {
		key := tilemap.TileKey{Z: z, X: x, Y: y, Source: src}
		// Check in-memory rendered cache first (no disk I/O, no PNG decode).
		if rows, ok := tilemap.GlobalMemCache.Get(key); ok {
			return TileReadyMsg{Key: key, Rows: rows}
		}
		pngData, err := tilemap.FetchTile(src, z, x, y)
		if err != nil {
			return TileReadyMsg{Key: key, Err: err}
		}
		rows, err := tilemap.RenderTile(pngData)
		if err == nil {
			tilemap.GlobalMemCache.Put(key, rows)
		}
		return TileReadyMsg{Key: key, Rows: rows, Err: err}
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
		m.validation = computeIncidentValidation(msg.Incidents, m.lat, m.lng, m.zoom, m.width, m.height)
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
		// Auto-open detail after number-key timeout.
		if m.numberBuf != "" && time.Since(m.numberBufAt) > numberBufTimeout {
			num, err := strconv.Atoi(m.numberBuf)
			if err == nil && num >= 1 && num <= len(m.incidents) {
				m.selectedIncident = num - 1
				m.showDetail = true
			}
			m.numberBuf = ""
		}
		return m, tickCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m MapViewModel) handleKey(msg tea.KeyMsg) (MapViewModel, tea.Cmd) {
	// When detail overlay is visible, only allow closing it.
	if m.showDetail {
		switch msg.String() {
		case "esc", "enter":
			m.showDetail = false
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}

	key := msg.String()

	// Digit keys: quick-select an incident by its displayed number.
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
		return m.handleDigit(key)
	}

	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "tab":
		// Toggle focus between map and sidebar.
		if m.focus == FocusMap {
			m.focus = FocusSidebar
		} else {
			m.focus = FocusMap
		}
		return m, nil

	case "enter":
		m.numberBuf = ""
		if len(m.incidents) > 0 {
			m.showDetail = true
		}
		return m, nil

	case "+", "=":
		m.zoom = tilemap.ClampZoom(m.zoom + 1)
		m.tileCache = make(map[tilemap.TileKey][][]string)
		return m, m.fetchVisibleTiles()

	case "-":
		m.zoom = tilemap.ClampZoom(m.zoom - 1)
		m.tileCache = make(map[tilemap.TileKey][][]string)
		return m, m.fetchVisibleTiles()

	case "up":
		if m.focus == FocusSidebar {
			m.selectedIncident--
			if m.selectedIncident < 0 {
				m.selectedIncident = 0
			}
			return m, nil
		}
		m.lat += panDelta(m.zoom)
		return m.afterPan()

	case "down":
		if m.focus == FocusSidebar {
			m.selectedIncident++
			if m.selectedIncident >= len(m.incidents) {
				m.selectedIncident = len(m.incidents) - 1
			}
			return m, nil
		}
		m.lat -= panDelta(m.zoom)
		return m.afterPan()

	case "left":
		if m.focus == FocusSidebar {
			return m, nil
		}
		m.lng -= panDelta(m.zoom)
		return m.afterPan()

	case "right":
		if m.focus == FocusSidebar {
			return m, nil
		}
		m.lng += panDelta(m.zoom)
		return m.afterPan()

	case "j":
		m.selectedIncident++
		if m.selectedIncident >= len(m.incidents) {
			m.selectedIncident = len(m.incidents) - 1
		}
		return m, nil

	case "k":
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

	case "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "esc":
		if m.focus == FocusSidebar {
			m.focus = FocusMap
			return m, nil
		}
		m.showHelp = false
		return m, nil

	case "m":
		m.tileSource = m.tileSource.Next()
		m.tileCache = make(map[tilemap.TileKey][][]string)
		return m, m.fetchVisibleTiles()

	case "r":
		m.loading = true
		return m, fetchIncidentsCmd(m.lat, m.lng, m.city)
	}

	return m, nil
}

// handleDigit accumulates digit presses into numberBuf. After a short
// timeout (checked in TickMsg) the corresponding incident detail is shown.
func (m MapViewModel) handleDigit(digit string) (MapViewModel, tea.Cmd) {
	// Reset buffer if too much time has passed since last digit.
	if time.Since(m.numberBufAt) > 1*time.Second {
		m.numberBuf = ""
	}
	// Don't start a number with 0.
	if digit == "0" && m.numberBuf == "" {
		return m, nil
	}
	m.numberBuf += digit
	m.numberBufAt = time.Now()

	// Update selection immediately so the sidebar highlights the target.
	num, err := strconv.Atoi(m.numberBuf)
	if err == nil && num >= 1 && num <= len(m.incidents) {
		m.selectedIncident = num - 1
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

// afterPan fetches visible tiles and conditionally re-fetches incidents if the
// map center has moved past the threshold. Tile cache is NOT wiped — cached tiles
// from before the pan remain valid and reusable.
func (m MapViewModel) afterPan() (MapViewModel, tea.Cmd) {
	cmds := []tea.Cmd{m.fetchVisibleTiles()}
	if m.shouldRefetchIncidents() {
		m.loading = true
		m.lastFetchLat = m.lat
		m.lastFetchLng = m.lng
		cmds = append(cmds, fetchIncidentsCmd(m.lat, m.lng, m.city))
	}
	return m, tea.Batch(cmds...)
}

// fetchVisibleTiles returns commands to fetch all tiles currently in view,
// plus a 1-tile prefetch ring around the viewport.
func (m MapViewModel) fetchVisibleTiles() tea.Cmd {
	if m.width == 0 || m.height == 0 {
		return nil
	}

	mapCols, mapRows := m.mapDimensions()
	minTX, minTY, maxTX, maxTY := tilemap.TilesForView(m.lat, m.lng, m.zoom, mapCols, mapRows)
	maxTile := (1 << uint(m.zoom)) - 1

	var cmds []tea.Cmd

	// Visible tiles first.
	for ty := minTY; ty <= maxTY; ty++ {
		for tx := minTX; tx <= maxTX; tx++ {
			key := tilemap.TileKey{Z: m.zoom, X: tx, Y: ty, Source: m.tileSource}
			if _, cached := m.tileCache[key]; !cached {
				cmds = append(cmds, fetchTileCmd(m.tileSource, m.zoom, tx, ty))
			}
		}
	}

	// Prefetch 1-tile border ring outside the visible area.
	for ty := minTY - 1; ty <= maxTY+1; ty++ {
		for tx := minTX - 1; tx <= maxTX+1; tx++ {
			if tx < 0 || ty < 0 || tx > maxTile || ty > maxTile {
				continue
			}
			if ty >= minTY && ty <= maxTY && tx >= minTX && tx <= maxTX {
				continue // already queued above
			}
			key := tilemap.TileKey{Z: m.zoom, X: tx, Y: ty, Source: m.tileSource}
			if _, cached := m.tileCache[key]; cached {
				continue
			}
			if _, ok := tilemap.GlobalMemCache.Get(key); ok {
				continue // already rendered in memory; will be a fast cmd when needed
			}
			cmds = append(cmds, fetchTileCmd(m.tileSource, m.zoom, tx, ty))
		}
	}

	return tea.Batch(cmds...)
}

// mapDimensions returns the width/height of the map area in terminal cells.
// The sidebar is always visible, so its width is always subtracted.
func (m MapViewModel) mapDimensions() (cols, rows int) {
	sidebarW := ui.SidebarWidth + 1 // +1 for border
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

	// Recompute validation when dimensions change so OffMap is accurate.
	if len(m.incidents) > 0 {
		m.validation = computeIncidentValidation(m.incidents, m.lat, m.lng, m.zoom, m.width, m.height)
	}

	// Render map area.
	mapContent := m.renderMap(mapCols, mapRows)

	sidebar := ui.RenderSidebarWithValidation(m.incidents, m.validation, m.selectedIncident, mapRows, m.focus == FocusSidebar)
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
	if m.focus == FocusSidebar {
		borderStyle = borderStyle.Foreground(ui.ColorHighlight)
	}
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
	view := joined.String()

	statusBar := ui.RenderStatusBarWithValidation(m.zip, m.nextRefresh, m.incidents, m.validation, m.width, m.loading, m.tileSource.Name(), m.numberBuf)
	result := view + "\n" + statusBar

	if m.showDetail && len(m.incidents) > 0 && m.selectedIncident < len(m.incidents) {
		result = overlayDetail(result, m.incidents[m.selectedIncident], m.width, m.height)
	}

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
			key := tilemap.TileKey{Z: m.zoom, X: tx, Y: ty, Source: m.tileSource}
			tileRows, ok := m.tileCache[key]
			if !ok {
				continue
			}

			tilePX := tx * tilemap.TileSize
			tilePY := ty * tilemap.TileSize
			tileOffsetCol := tilePX - originPX
			tileOffsetRow := (tilePY - originPY) / 2

			for tileRow, cells := range tileRows {
				gridRow := tileOffsetRow + tileRow
				if gridRow < 0 || gridRow >= len(grid) {
					continue
				}
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

	// Overlay numbered incident markers (3+ cells wide, 2 rows tall).
	for i, inc := range m.incidents {
		// Skip incidents that are off-map according to current validation snapshot.
		if i < len(m.validation) && m.validation[i].OffMap {
			continue
		}
		incPX, incPY := tilemap.LatLngToPixelCoord(inc.Lat, inc.Lng, m.zoom)
		col, row := tilemap.PixelToCell(incPX, incPY, originPX, originPY)

		numStr := strconv.Itoa(i + 1)
		markerW := len(numStr) + 2 // padding on each side
		startCol := col - markerW/2
		colorHex := inc.Type.Color()

		// Top row: solid colored background.
		if row-1 >= 0 && row-1 < rows {
			for dx := 0; dx < markerW; dx++ {
				gc := startCol + dx
				if gc >= 0 && gc < cols {
					grid[row-1][gc] = tilemap.SolidBgCell(colorHex)
				}
			}
		}

		// Main row: padding + number digits + padding.
		if row >= 0 && row < rows {
			for dx := 0; dx < markerW; dx++ {
				gc := startCol + dx
				if gc < 0 || gc >= cols {
					continue
				}
				digitIdx := dx - 1
				if digitIdx >= 0 && digitIdx < len(numStr) {
					grid[row][gc] = tilemap.NumberCell(rune(numStr[digitIdx]), colorHex)
				} else {
					grid[row][gc] = tilemap.SolidBgCell(colorHex)
				}
			}
		}
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

// computeIncidentValidation derives validation flags for each incident based on
// current position/zoom and simple heuristics. It never mutates the incidents.
func computeIncidentValidation(incidents []data.Incident, lat, lng float64, zoom, width, height int) []ui.IncidentValidation {
	out := make([]ui.IncidentValidation, len(incidents))
	if len(incidents) == 0 {
		return out
	}

	now := time.Now()

	// Pre-compute duplicate signatures.
	sigs := make(map[string]int, len(incidents))
	for _, inc := range incidents {
		key := duplicateSignature(inc)
		if key == "" {
			continue
		}
		sigs[key]++
	}

	// Map bounds for off-map detection.
	mapCols := 1
	mapRows := 1
	if width > 0 && height > 0 {
		// Reuse mapDimensions calculation: sidebar width + status bar.
		mapCols = width - ui.SidebarWidth - 1
		if mapCols < 1 {
			mapCols = 1
		}
		mapRows = height - 1
		if mapRows < 1 {
			mapRows = 1
		}
	}
	originPX, originPY := tilemap.MapOriginPixel(lat, lng, zoom, mapCols, mapRows)

	for i, inc := range incidents {
		var v ui.IncidentValidation
		v.Freshness = data.ClassifyFreshness(inc.Time, now)

		// Duplicate inference.
		if key := duplicateSignature(inc); key != "" && sigs[key] > 1 {
			v.LikelyDuplicate = true
		}

		// Off-map detection using current viewport.
		px, py := tilemap.LatLngToPixelCoord(inc.Lat, inc.Lng, zoom)
		col, row := tilemap.PixelToCell(px, py, originPX, originPY)
		if col < 0 || col >= mapCols || row < 0 || row >= mapRows {
			v.OffMap = true
		}

		out[i] = v
	}
	return out
}

// duplicateSignature collapses an incident into a coarse key used to spot
// likely duplicates from multiple sources.
func duplicateSignature(inc data.Incident) string {
	addr := strings.TrimSpace(strings.ToLower(inc.Address))
	title := strings.TrimSpace(strings.ToLower(inc.Title))
	if addr == "" && title == "" {
		return ""
	}
	// Round to 5-minute buckets to avoid minor clock skew differences.
	trunc := inc.Time.Truncate(5 * time.Minute).UTC().Format(time.RFC3339)
	return fmt.Sprintf("%s|%s|%s|%d", addr, title, trunc, inc.Type)
}

// panDelta returns degrees to pan per keypress at the given zoom level.
func panDelta(zoom int) float64 {
	return 0.15 / float64(zoom)
}

// overlayDetail overlays the incident detail box on the existing rendered output.
func overlayDetail(base string, inc data.Incident, width, height int) string {
	detailOverlay := ui.RenderDetailOverlay(inc, width, height)
	baseLines := strings.Split(base, "\n")
	detailLines := strings.Split(detailOverlay, "\n")

	out := make([]string, len(baseLines))
	copy(out, baseLines)

	for i, line := range detailLines {
		if i >= len(out) {
			break
		}
		if strings.TrimSpace(line) != "" {
			out[i] = line
		}
	}
	return strings.Join(out, "\n")
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
