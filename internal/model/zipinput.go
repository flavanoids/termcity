package model

import (
	"os"
	"path/filepath"
	"strings"
	"termcity/internal/history"
	"termcity/internal/ui"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ZipInputModel is the zip code entry screen.
type ZipInputModel struct {
	textInput textinput.Model
	err       string
	width     int
	height    int

	recentZips []history.RecentZip
}

func NewZipInputModel() ZipInputModel {
	ti := textinput.New()
	ti.Placeholder = "Enter zip code (e.g. 10001)"
	ti.Focus()
	ti.CharLimit = 5
	ti.Width = 30

	return ZipInputModel{
		textInput: ti,
	}
}

func (m ZipInputModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		func() tea.Msg {
			return fetchRecentZipsCmd()()
		},
	)
}

type RecentZipsMsg struct {
	Zips []history.RecentZip
}

func fetchRecentZipsCmd() tea.Cmd {
	return func() tea.Msg {
		store, err := openHistoryStore()
		if err != nil {
			return RecentZipsMsg{Zips: nil}
		}
		defer store.Close()
		zips, _ := store.GetRecentZips()
		return RecentZipsMsg{Zips: zips}
	}
}

func (m ZipInputModel) Update(msg tea.Msg) (ZipInputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case RecentZipsMsg:
		m.recentZips = msg.Zips

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			zip := strings.TrimSpace(m.textInput.Value())
			if len(zip) != 5 {
				m.err = "Please enter a valid 5-digit zip code"
				return m, nil
			}
			for _, ch := range zip {
				if ch < '0' || ch > '9' {
					m.err = "Zip code must contain only digits"
					return m, nil
				}
			}
			// Signal that zip is ready — parent model handles the transition.
			return m, func() tea.Msg {
				return ZipSubmittedMsg{Zip: zip}
			}
		case tea.KeyCtrlC:
			return m, tea.Quit
		}

		// Handle number keys for quick selection of recent zips
		key := msg.String()
		if len(key) == 1 && key[0] >= '1' && key[0] <= '5' {
			idx := int(key[0] - '1')
			if idx < len(m.recentZips) {
				return m, func() tea.Msg {
					return ZipSubmittedMsg{Zip: m.recentZips[idx].Zip}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m ZipInputModel) View() string {
	var sb strings.Builder

	title := ui.ZipTitleStyle.Render("TermCity — Emergency Incident Map")
	subtitle := ui.HelpStyle.Render("Real-time 911 incidents overlaid on an OSM map")

	inputBox := ui.ZipInputStyle.Render(
		ui.ZipPromptStyle.Render("Enter ZIP Code") + "\n\n" +
			m.textInput.View() + "\n\n" +
			ui.HelpStyle.Render("Press Enter to load map"),
	)

	var errLine string
	if m.err != "" {
		errLine = "\n" + ui.ErrorStyle.Render("⚠ "+m.err)
	}

	var recentSection string
	if len(m.recentZips) > 0 {
		var recentLines []string
		recentLines = append(recentLines, ui.SidebarDividerStyle.Render(strings.Repeat("─", 30)))
		recentLines = append(recentLines, ui.SidebarTitleStyle.Render("Recent Locations"))
		for i, z := range m.recentZips {
			label := ui.HelpStyle.Render(string('1'+i)) + " " + z.Zip
			if z.City != "" {
				label += ui.HelpStyle.Render(" — "+z.City)
			}
			recentLines = append(recentLines, label)
		}
		recentSection = lipgloss.JoinVertical(lipgloss.Left, recentLines...)
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		subtitle,
		"",
		inputBox,
		errLine,
		recentSection,
	)

	// Center vertically.
	contentHeight := strings.Count(content, "\n") + 1
	topPad := (m.height - contentHeight) / 2
	if topPad < 0 {
		topPad = 0
	}

	sb.WriteString(strings.Repeat("\n", topPad))
	sb.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, content))

	return sb.String()
}

// ZipSubmittedMsg signals that the user has entered a valid zip code.
type ZipSubmittedMsg struct {
	Zip string
}

func openHistoryStore() (*history.Store, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	dir := filepath.Join(cacheDir, "termcity")
	os.MkdirAll(dir, 0755)
	dbPath := filepath.Join(dir, "history.db")
	return history.Open(dbPath)
}
