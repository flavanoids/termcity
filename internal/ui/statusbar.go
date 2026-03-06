package ui

import (
	"fmt"
	"strings"
	"termcity/internal/data"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// RenderStatusBar renders the bottom status bar.
func RenderStatusBar(zip string, nextRefresh time.Time, incidents []data.Incident, width int, loading bool, mapStyle string, numberBuf string) string {
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

	// Map style.
	parts = append(parts, StatusBarKeyStyle.Render("[m]")+HelpStyle.Render(mapStyle))

	// Number input feedback.
	if numberBuf != "" {
		parts = append(parts, StatusBarKeyStyle.Render("#")+StatusBarStyle.Render(numberBuf+"…"))
	}

	// Help hint.
	helpPart := StatusBarKeyStyle.Render("[Tab]") + HelpStyle.Render("Focus") +
		"  " + StatusBarKeyStyle.Render("[1-9]") + HelpStyle.Render("Go to") +
		"  " + StatusBarKeyStyle.Render("[?]") + HelpStyle.Render("Help") +
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
		lipgloss.NewStyle().Bold(true).Foreground(ColorText).Render("Map (default focus)"),
		fmt.Sprintf("  %-10s %s", "↑↓←→", "Pan map"),
		fmt.Sprintf("  %-10s %s", "+/-", "Zoom in/out"),
		fmt.Sprintf("  %-10s %s", "m", "Cycle map style"),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(ColorText).Render("Events sidebar (Tab to focus)"),
		fmt.Sprintf("  %-10s %s", "↑↓", "Navigate event list"),
		fmt.Sprintf("  %-10s %s", "Enter", "Show event detail"),
		fmt.Sprintf("  %-10s %s", "Esc", "Return to map"),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(ColorText).Render("Global"),
		fmt.Sprintf("  %-10s %s", "1-9", "Jump to event # (detail)"),
		fmt.Sprintf("  %-10s %s", "j/k", "Navigate event list"),
		fmt.Sprintf("  %-10s %s", "r", "Refresh incidents"),
		fmt.Sprintf("  %-10s %s", "Tab", "Switch focus"),
		fmt.Sprintf("  %-10s %s", "?", "Toggle this help"),
		fmt.Sprintf("  %-10s %s", "q / Ctrl+C", "Quit"),
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
