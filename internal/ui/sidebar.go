package ui

import (
	"fmt"
	"strings"
	"termcity/internal/data"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const SidebarWidth = 26

// RenderSidebar renders the incident list sidebar.
// height is the available height (excluding status bar).
// selected is the index of the currently highlighted incident.
func RenderSidebar(incidents []data.Incident, selected int, height int, focused bool) string {
	var sb strings.Builder

	titleText := "ACTIVE INCIDENTS"
	if focused {
		titleText = "▸ ACTIVE INCIDENTS"
	}
	titleStyle := SidebarTitleStyle
	if focused {
		titleStyle = titleStyle.Background(ColorBorder)
	}
	title := titleStyle.Width(SidebarWidth).Render(titleText)
	sb.WriteString(title)
	sb.WriteString("\n")

	divider := SidebarDividerStyle.Width(SidebarWidth).Render(strings.Repeat("─", SidebarWidth))
	sb.WriteString(divider)
	sb.WriteString("\n")

	// Each incident takes 2 lines: symbol+title, address.
	linesUsed := 2
	maxIncidents := (height - linesUsed) / 2
	if maxIncidents < 1 {
		maxIncidents = 1
	}

	if len(incidents) == 0 {
		msg := SidebarStyle.Width(SidebarWidth).Render("No active incidents")
		sb.WriteString(msg)
		// Pad remaining lines.
		remaining := height - linesUsed - 1
		for i := 0; i < remaining; i++ {
			sb.WriteString(SidebarStyle.Width(SidebarWidth).Render(""))
			sb.WriteString("\n")
		}
		return sb.String()
	}

	// Determine scroll offset so selected item is visible.
	scrollOffset := 0
	if selected >= maxIncidents {
		scrollOffset = selected - maxIncidents + 1
	}

	renderedLines := 2 // title + divider already written

	for i := scrollOffset; i < len(incidents) && renderedLines < height; i++ {
		inc := incidents[i]

		numStr := fmt.Sprintf("%d.", i+1)
		symbol := incidentSymbol(inc.Type)
		maxTitle := SidebarWidth - 4 - len(numStr) - 1
		if maxTitle < 4 {
			maxTitle = 4
		}
		title := truncate(inc.Title, maxTitle)
		line1 := fmt.Sprintf("%s %s %-*s", symbol, numStr, maxTitle, title)

		addr := truncate(inc.Address, SidebarWidth-2)
		line2 := fmt.Sprintf("  %-*s", SidebarWidth-2, addr)

		ago := timeAgo(inc.Time)
		if len(ago) > 0 && len(line2)+len(ago)+1 < SidebarWidth {
			// Fits: append ago time.
		} else {
			ago = ""
		}

		var style1, style2 lipgloss.Style
		if i == selected {
			style1 = SelectedIncidentStyle.Width(SidebarWidth)
			style2 = SelectedIncidentStyle.Width(SidebarWidth).Foreground(ColorDim)
		} else {
			style1 = incidentStyle(inc.Type).Width(SidebarWidth)
			style2 = SidebarStyle.Width(SidebarWidth).Foreground(ColorDim)
		}

		sb.WriteString(style1.Render(line1))
		sb.WriteString("\n")
		renderedLines++

		if renderedLines < height {
			if ago != "" {
				line2 = fmt.Sprintf("  %-*s%s", SidebarWidth-2-len(ago)-1, addr, ago)
			}
			sb.WriteString(style2.Render(line2))
			sb.WriteString("\n")
			renderedLines++
		}
	}

	// Fill remaining lines with empty styled rows.
	for renderedLines < height {
		sb.WriteString(SidebarStyle.Width(SidebarWidth).Render(""))
		sb.WriteString("\n")
		renderedLines++
	}

	return sb.String()
}

func incidentSymbol(t data.IncidentType) string {
	switch t {
	case data.Fire:
		return IncidentFireStyle.Render("●")
	case data.Police:
		return IncidentPoliceStyle.Render("●")
	case data.EMS:
		return IncidentEMSStyle.Render("●")
	}
	return "●"
}

func incidentStyle(t data.IncidentType) lipgloss.Style {
	switch t {
	case data.Fire:
		return IncidentFireStyle
	case data.Police:
		return IncidentPoliceStyle
	case data.EMS:
		return IncidentEMSStyle
	}
	return SidebarStyle
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

// RenderDetailOverlay renders a full incident detail box.
func RenderDetailOverlay(inc data.Incident, width, height int) string {
	boxWidth := min(60, width-4)

	var sb strings.Builder
	sb.WriteString(DetailBoxStyle.Width(boxWidth).Render(
		formatDetail(inc, boxWidth-4),
	))

	// Center in terminal.
	content := sb.String()
	lines := strings.Split(content, "\n")
	paddingTop := (height - len(lines)) / 2

	var out strings.Builder
	for i := 0; i < paddingTop; i++ {
		out.WriteString("\n")
	}
	paddingLeft := (width - boxWidth) / 2
	prefix := strings.Repeat(" ", paddingLeft)
	for _, line := range lines {
		out.WriteString(prefix + line + "\n")
	}
	return out.String()
}

func formatDetail(inc data.Incident, width int) string {
	symbol := inc.Type.Symbol()
	title := fmt.Sprintf("%s %s", symbol, inc.Title)

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(incidentColor(inc.Type)).Render(title))
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("Address: %s\n", inc.Address))
	sb.WriteString(fmt.Sprintf("Time:    %s (%s ago)\n", inc.Time.Format("15:04:05"), timeAgo(inc.Time)))
	sb.WriteString(fmt.Sprintf("Source:  %s\n", inc.Source))
	if len(inc.Units) > 0 {
		sb.WriteString(fmt.Sprintf("Units:   %s\n", strings.Join(inc.Units, ", ")))
	}
	sb.WriteString("\n")
	sb.WriteString(HelpStyle.Render("Press Esc or Enter to close"))
	return sb.String()
}

func incidentColor(t data.IncidentType) lipgloss.Color {
	switch t {
	case data.Fire:
		return ColorFire
	case data.Police:
		return ColorPolice
	case data.EMS:
		return ColorEMS
	}
	return ColorText
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
