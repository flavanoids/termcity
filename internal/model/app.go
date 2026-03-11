package model

import (
	"termcity/internal/data"
	"termcity/internal/history"

	tea "github.com/charmbracelet/bubbletea"
)

// Screen represents the current UI screen.
type Screen int

const (
	ScreenZipInput Screen = iota
	ScreenLoading
	ScreenMap
	ScreenHistory
)

// GeocodeDoneMsg signals that geocoding has completed.
type GeocodeDoneMsg struct {
	Loc *data.GeoLocation
	Err error
}

// AppModel is the root model that routes between screens.
type AppModel struct {
	screen       Screen
	width        int
	height       int
	zip          string
	loc          *data.GeoLocation
	loadMsg      string
	errMsg       string
	zipInput     ZipInputModel
	mapView      MapViewModel
	historyView  HistoryViewModel
	historyStore *history.Store
}

func NewAppModel(store *history.Store) AppModel {
	return AppModel{
		screen:       ScreenZipInput,
		zipInput:     NewZipInputModel(),
		historyStore: store,
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.zipInput.Init(),
		tea.SetWindowTitle("TermCity"),
	)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Intercept screen-transition messages before routing to sub-models.
	switch msg.(type) {
	case ShowHistoryMsg:
		m.screen = ScreenHistory
		m.historyView = NewHistoryViewModel(m.historyStore, m.width, m.height)
		return m, m.historyView.Init()
	case BackToMapMsg:
		m.screen = ScreenMap
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		switch m.screen {
		case ScreenZipInput:
			m.zipInput, _ = m.zipInput.Update(msg)
		case ScreenMap:
			var cmd tea.Cmd
			m.mapView, cmd = m.mapView.Update(msg)
			return m, cmd
		case ScreenHistory:
			var cmd tea.Cmd
			m.historyView, cmd = m.historyView.Update(msg)
			return m, cmd
		}
		return m, nil

	case ZipSubmittedMsg:
		m.zip = msg.Zip
		m.screen = ScreenLoading
		m.loadMsg = "Geocoding " + msg.Zip + "..."
		return m, geocodeCmd(msg.Zip)

	case GeocodeDoneMsg:
		if msg.Err != nil {
			m.errMsg = "Geocode failed: " + msg.Err.Error()
			m.screen = ScreenZipInput
			return m, nil
		}
		m.loc = msg.Loc
		m.screen = ScreenMap
		m.mapView = NewMapViewModel(m.zip, msg.Loc.Lat, msg.Loc.Lng, msg.Loc.City, m.historyStore)
		m.mapView.width = m.width
		m.mapView.height = m.height
		cmds := []tea.Cmd{m.mapView.Init()}
		if m.width > 0 {
			cmds = append(cmds, m.mapView.fetchVisibleTiles())
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch m.screen {
		case ScreenZipInput:
			var cmd tea.Cmd
			m.zipInput, cmd = m.zipInput.Update(msg)
			return m, cmd
		case ScreenMap:
			var cmd tea.Cmd
			m.mapView, cmd = m.mapView.Update(msg)
			return m, cmd
		case ScreenHistory:
			var cmd tea.Cmd
			m.historyView, cmd = m.historyView.Update(msg)
			return m, cmd
		case ScreenLoading:
			if msg.Type == tea.KeyCtrlC {
				return m, tea.Quit
			}
		}

	default:
		// Route all other messages to the appropriate sub-model.
		switch m.screen {
		case ScreenZipInput:
			var cmd tea.Cmd
			m.zipInput, cmd = m.zipInput.Update(msg)
			return m, cmd
		case ScreenMap:
			var cmd tea.Cmd
			m.mapView, cmd = m.mapView.Update(msg)
			return m, cmd
		case ScreenHistory:
			var cmd tea.Cmd
			m.historyView, cmd = m.historyView.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m AppModel) View() string {
	switch m.screen {
	case ScreenZipInput:
		return m.zipInput.View()
	case ScreenLoading:
		return LoadingView(m.zip, m.loadMsg, m.width, m.height)
	case ScreenMap:
		return m.mapView.View()
	case ScreenHistory:
		return m.historyView.View()
	default:
		return "Unknown screen"
	}
}

// geocodeCmd fetches the lat/lng for a zip code.
func geocodeCmd(zip string) tea.Cmd {
	return func() tea.Msg {
		loc, err := data.GeocodeZip(zip)
		return GeocodeDoneMsg{Loc: loc, Err: err}
	}
}
