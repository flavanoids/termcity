package ui

import (
	"fmt"
	"strings"
	"termcity/internal/data"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// RenderStatusBar renders the bottom status bar.
func RenderStatusBar(zip string, nextRefresh time.Time, incidents []data.Incident, width int, loading bool) string {
	// Count by type.
	var fires, police, ems int
	for _, inc := range incidents {
		switch inc.Type {
		case data.Fire:
			fires++
		case data.Police:
			police++
		case data.EMS:
			ems++
		}
	}

	var parts []string

	// ZIP.
	zipPart := StatusBarKeyStyle.Render("ZIP:") + StatusBarStyle.Render(zip)
	parts = append(parts, zipPart)

	// Refresh countdown.
	if loading {
		parts = append(parts, StatusBarStyle.Render("↻ loading..."))
	} else {
		remaining := time.Until(nextRefresh).Round(time.Second)
		if remaining < 0 {
			remaining = 0
		}
		parts = append(parts, StatusBarStyle.Render(fmt.Sprintf("↻ %ds", int(remaining.Seconds()))))
	}

	// Incident counts.
	counts := fmt.Sprintf("%s%d  %s%d  %s%d",
		lipgloss.NewStyle().Foreground(ColorFire).Render("●"), fires,
		lipgloss.NewStyle().Foreground(ColorPolice).Render("●"), police,
		lipgloss.NewStyle().Foreground(ColorEMS).Render("●"), ems,
	)
	parts = append(parts, counts)

	// Help hint.
	helpPart := StatusBarKeyStyle.Render("[?]") + HelpStyle.Render("Help") +
		"  " + StatusBarKeyStyle.Render("[q]") + HelpStyle.Render("Quit")
	parts = append(parts, helpPart)

	// Join with separators.
	content := strings.Join(parts, StatusBarStyle.Render(" | "))

	return StatusBarStyle.Width(width).Render(content)
}

// RenderHelpOverlay renders a help overlay in the center of the screen.
func RenderHelpOverlay(width, height int) string {
	boxWidth := 44
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render("TermCity Help"),
		"",
		fmt.Sprintf("%-12s %s", "+/-", "Zoom in/out"),
		fmt.Sprintf("%-12s %s", "↑↓←→", "Pan map"),
		fmt.Sprintf("%-12s %s", "Tab", "Toggle sidebar"),
		fmt.Sprintf("%-12s %s", "Enter", "View incident detail"),
		fmt.Sprintf("%-12s %s", "j/k", "Navigate incident list"),
		fmt.Sprintf("%-12s %s", "r", "Refresh incidents"),
		fmt.Sprintf("%-12s %s", "?", "Toggle this help"),
		fmt.Sprintf("%-12s %s", "q / Ctrl+C", "Quit"),
		"",
		HelpStyle.Render("Press ? or Esc to close"),
	}

	content := strings.Join(lines, "\n")
	box := DetailBoxStyle.Width(boxWidth).Render(content)

	boxLines := strings.Split(box, "\n")
	paddingTop := (height - len(boxLines)) / 2
	paddingLeft := (width - boxWidth) / 2
	if paddingLeft < 0 {
		paddingLeft = 0
	}
	prefix := strings.Repeat(" ", paddingLeft)

	var out strings.Builder
	for i := 0; i < paddingTop; i++ {
		out.WriteString("\n")
	}
	for _, line := range boxLines {
		out.WriteString(prefix + line + "\n")
	}
	return out.String()
}
