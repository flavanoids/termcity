package model

import (
	"strings"
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
	return textinput.Blink
}

func (m ZipInputModel) Update(msg tea.Msg) (ZipInputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

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

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		subtitle,
		"",
		inputBox,
		errLine,
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
