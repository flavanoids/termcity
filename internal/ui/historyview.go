package ui

import (
	"fmt"
	"strings"
	"termcity/internal/data"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// RenderHistoryView renders the full-screen incident history view.
func RenderHistoryView(incidents []data.Incident, selected, days, width, height int, loading bool, confirmClear bool) string {
	if width == 0 || height == 0 {
		return "Initializing..."
	}

	var sb strings.Builder

	// Header line: title + day tabs + count
	countStr := fmt.Sprintf("%d incidents", len(incidents))
	if loading {
		countStr = "↻ loading..."
	}
	tab := func(d int) string {
		label := fmt.Sprintf("[%dd]", d)
		if d == days {
			return HistoryTabActive.Render(label)
		}
		return HistoryTabInactive.Render(label)
	}
	headerContent := fmt.Sprintf("INCIDENT HISTORY  %s %s %s   %s",
		tab(1), tab(3), tab(7), countStr)
	sb.WriteString(HistoryTitleStyle.Width(width).Render(headerContent))
	sb.WriteString("\n")

	// Column widths
	numW := 4
	typeW := 2
	timeW := 14
	remaining := width - numW - typeW - timeW - 4
	if remaining < 20 {
		remaining = 20
	}
	titleW := remaining * 55 / 100
	addrW := remaining - titleW

	// Column header
	hdrContent := fmt.Sprintf("%-*s %-*s Time",
		numW+typeW+1+titleW, "  # Type  Title",
		addrW, "Address")
	sb.WriteString(HistoryHeaderStyle.Width(width).Render(hdrContent))
	sb.WriteString("\n")

	// Divider
	divider := SidebarDividerStyle.Width(width).Render(strings.Repeat("─", width))
	sb.WriteString(divider)
	sb.WriteString("\n")

	// Rows area: height minus header(1) + col-hdr(1) + divider(1) + status(1) = 4
	rowsAvail := height - 4
	if rowsAvail < 0 {
		rowsAvail = 0
	}

	// Scroll to keep selected visible.
	scrollOffset := 0
	if selected >= rowsAvail {
		scrollOffset = selected - rowsAvail + 1
	}

	renderedRows := 0
	for i := scrollOffset; i < len(incidents) && renderedRows < rowsAvail; i++ {
		inc := incidents[i]
		numStr := fmt.Sprintf("%*d", numW, i+1)
		dot := historyIncidentDot(inc.Type)
		title := truncate(inc.Title, titleW)
		addr := truncate(inc.Address, addrW)
		timeStr := formatHistoryTime(inc.Time)

		row := fmt.Sprintf("%s %s %-*s %-*s %s",
			numStr, dot,
			titleW, title,
			addrW, addr,
			timeStr)

		var style lipgloss.Style
		if i == selected {
			style = SelectedIncidentStyle.Width(width)
		} else {
			style = HistoryRowStyle.Width(width)
		}
		sb.WriteString(style.Render(row))
		sb.WriteString("\n")
		renderedRows++
	}

	// Fill remaining rows.
	for renderedRows < rowsAvail {
		sb.WriteString(HistoryRowStyle.Width(width).Render(""))
		sb.WriteString("\n")
		renderedRows++
	}

	// Status bar
	sb.WriteString(renderHistoryStatusBar(width, confirmClear))

	return sb.String()
}

func renderHistoryStatusBar(width int, confirmClear bool) string {
	var parts []string
	parts = append(parts, StatusBarKeyStyle.Render("[1/3/7]")+HelpStyle.Render("Window"))
	parts = append(parts, StatusBarKeyStyle.Render("[j/k]")+HelpStyle.Render("Navigate"))
	parts = append(parts, StatusBarKeyStyle.Render("[Enter]")+HelpStyle.Render("Detail"))
	if confirmClear {
		parts = append(parts, StatusBarKeyStyle.Render("[X]")+
			lipgloss.NewStyle().Foreground(ColorWarning).Render("Press X again to confirm clear"))
	} else {
		parts = append(parts, StatusBarKeyStyle.Render("[X]")+HelpStyle.Render("Clear all"))
	}
	parts = append(parts, StatusBarKeyStyle.Render("[r]")+HelpStyle.Render("Reload"))
	parts = append(parts, StatusBarKeyStyle.Render("[Esc]")+HelpStyle.Render("Back"))
	parts = append(parts, StatusBarKeyStyle.Render("[q]")+HelpStyle.Render("Quit"))

	content := strings.Join(parts, StatusBarStyle.Render(" | "))
	return StatusBarStyle.Width(width).Render(content)
}

func historyIncidentDot(t data.IncidentType) string {
	switch t {
	case data.Fire:
		return lipgloss.NewStyle().Foreground(ColorFire).Render("●")
	case data.Police:
		return lipgloss.NewStyle().Foreground(ColorPolice).Render("●")
	case data.EMS:
		return lipgloss.NewStyle().Foreground(ColorEMS).Render("●")
	}
	return "●"
}

func formatHistoryTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s %s", t.Format("01/02 15:04"), timeAgo(t))
}
