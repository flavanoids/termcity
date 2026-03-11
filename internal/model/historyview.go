package model

import (
	"fmt"
	"termcity/internal/data"
	"termcity/internal/history"
	"termcity/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

// BackToMapMsg signals the history view wants to return to the map screen.
type BackToMapMsg struct{}

// HistoryFetchedMsg carries loaded history incidents.
type HistoryFetchedMsg struct {
	Days      int
	Incidents []data.Incident
	Err       error
}

// HistoryClearedMsg signals ClearHistory completed.
type HistoryClearedMsg struct{ Err error }

// HistoryViewModel is the full-screen incident history sub-model.
// No mutexes — all I/O goes through commands per bubbletea MVU rules.
type HistoryViewModel struct {
	store        *history.Store
	width        int
	height       int
	days         int
	incidents    []data.Incident
	selected     int
	loading      bool
	err          string
	showDetail   bool
	confirmClear bool
}

func NewHistoryViewModel(store *history.Store, w, h int) HistoryViewModel {
	return HistoryViewModel{
		store:   store,
		width:   w,
		height:  h,
		days:    7,
		loading: true,
	}
}

func (m HistoryViewModel) Init() tea.Cmd {
	return fetchHistoryCmd(m.store, m.days)
}

func (m HistoryViewModel) Update(msg tea.Msg) (HistoryViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case HistoryFetchedMsg:
		if msg.Days == m.days {
			m.incidents = msg.Incidents
			m.loading = false
			if msg.Err != nil {
				m.err = msg.Err.Error()
			}
		}
		return m, nil

	case HistoryClearedMsg:
		m.confirmClear = false
		if msg.Err != nil {
			m.err = msg.Err.Error()
		} else {
			m.incidents = nil
			m.selected = 0
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m HistoryViewModel) handleKey(msg tea.KeyMsg) (HistoryViewModel, tea.Cmd) {
	// When detail overlay is visible, only close it.
	if m.showDetail {
		switch msg.String() {
		case "esc", "enter":
			m.showDetail = false
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}

	// Any key other than X resets the clear confirmation.
	if msg.String() != "X" {
		m.confirmClear = false
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		return m, func() tea.Msg { return BackToMapMsg{} }

	case "1":
		m.days = 1
		m.loading = true
		m.selected = 0
		return m, fetchHistoryCmd(m.store, 1)

	case "3":
		m.days = 3
		m.loading = true
		m.selected = 0
		return m, fetchHistoryCmd(m.store, 3)

	case "7":
		m.days = 7
		m.loading = true
		m.selected = 0
		return m, fetchHistoryCmd(m.store, 7)

	case "j", "down":
		m.selected++
		if m.selected >= len(m.incidents) {
			m.selected = len(m.incidents) - 1
		}
		if m.selected < 0 {
			m.selected = 0
		}
		return m, nil

	case "k", "up":
		m.selected--
		if m.selected < 0 {
			m.selected = 0
		}
		return m, nil

	case "enter":
		if len(m.incidents) > 0 && m.selected < len(m.incidents) {
			m.showDetail = true
		}
		return m, nil

	case "r":
		m.loading = true
		return m, fetchHistoryCmd(m.store, m.days)

	case "X":
		if m.confirmClear {
			m.confirmClear = false
			return m, clearHistoryCmd(m.store)
		}
		m.confirmClear = true
		return m, nil
	}

	return m, nil
}

func (m HistoryViewModel) View() string {
	base := ui.RenderHistoryView(m.incidents, m.selected, m.days, m.width, m.height, m.loading, m.confirmClear)
	if m.showDetail && len(m.incidents) > 0 && m.selected < len(m.incidents) {
		base = overlayDetail(base, m.incidents[m.selected], m.width, m.height)
	}
	return base
}

func fetchHistoryCmd(store *history.Store, days int) tea.Cmd {
	return func() tea.Msg {
		if store == nil {
			return HistoryFetchedMsg{Days: days, Err: fmt.Errorf("history unavailable")}
		}
		incs, err := store.QueryHistory(days)
		return HistoryFetchedMsg{Days: days, Incidents: incs, Err: err}
	}
}

func clearHistoryCmd(store *history.Store) tea.Cmd {
	return func() tea.Msg {
		if store == nil {
			return HistoryClearedMsg{}
		}
		return HistoryClearedMsg{Err: store.ClearHistory()}
	}
}
