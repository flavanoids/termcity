package ui

import "github.com/charmbracelet/lipgloss"

const (
	ColorFire   = lipgloss.Color("#FF4444")
	ColorPolice = lipgloss.Color("#4488FF")
	ColorEMS    = lipgloss.Color("#EEEEEE")

	ColorBg        = lipgloss.Color("#1a1a2e")
	ColorSidebarBg = lipgloss.Color("#16213e")
	ColorBorder    = lipgloss.Color("#0f3460")
	ColorText      = lipgloss.Color("#e0e0e0")
	ColorDim       = lipgloss.Color("#888888")
	ColorHighlight = lipgloss.Color("#e94560")
	ColorSuccess   = lipgloss.Color("#4CAF50")
	ColorWarning   = lipgloss.Color("#FF9800")
)

var (
	SidebarStyle = lipgloss.NewStyle().
			Background(ColorSidebarBg).
			Foreground(ColorText).
			PaddingLeft(1).
			PaddingRight(1)

	SidebarTitleStyle = lipgloss.NewStyle().
				Background(ColorSidebarBg).
				Foreground(ColorHighlight).
				Bold(true).
				PaddingLeft(1)

	SidebarDividerStyle = lipgloss.NewStyle().
				Background(ColorSidebarBg).
				Foreground(ColorBorder)

	StatusBarStyle = lipgloss.NewStyle().
			Background(ColorBorder).
			Foreground(ColorText).
			PaddingLeft(1).
			PaddingRight(1)

	StatusBarKeyStyle = lipgloss.NewStyle().
				Background(ColorBorder).
				Foreground(ColorHighlight).
				Bold(true)

	ZipInputStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorBg).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorHighlight)

	ZipPromptStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true)

	ZipTitleStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Bold(true).
			MarginBottom(1)

	LoadingStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorFire).
			Bold(true)

	IncidentFireStyle = lipgloss.NewStyle().
				Foreground(ColorFire).
				Bold(true)

	IncidentPoliceStyle = lipgloss.NewStyle().
				Foreground(ColorPolice).
				Bold(true)

	IncidentEMSStyle = lipgloss.NewStyle().
				Foreground(ColorEMS)

	SelectedIncidentStyle = lipgloss.NewStyle().
				Background(ColorBorder).
				Foreground(ColorText).
				Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	DetailBoxStyle = lipgloss.NewStyle().
			Background(ColorSidebarBg).
			Foreground(ColorText).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorHighlight)
)
