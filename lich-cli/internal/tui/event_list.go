package tui

import (
	"fmt"
	"strings"
	"time"

	"lich-cli/internal/cache"

	"github.com/charmbracelet/lipgloss"
)

var (
	selectedEventStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#45475A")).
				Padding(0, 1)

	cursorArrowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#74C0FC"))

	crudHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Italic(true)
)

func formatEventTime(startStr, endStr string, loc *time.Location) string {
	tStart, err1 := time.Parse(time.RFC3339, startStr)
	tEnd, err2 := time.Parse(time.RFC3339, endStr)
	if err1 != nil || err2 != nil {
		return fmt.Sprintf("%s - %s", startStr, endStr)
	}

	startLocal := tStart.In(loc)
	endLocal := tEnd.In(loc)

	return fmt.Sprintf("%02d:%02d - %02d:%02d", startLocal.Hour(), startLocal.Minute(), endLocal.Hour(), endLocal.Minute())
}

func RenderAgenda(selectedDate time.Time, events []cache.LocalEvent, loc *time.Location, selectedIdx int, isFocused bool) string {
	var sb strings.Builder

	header := fmt.Sprintf("Agenda: %s (%d sự kiện)", selectedDate.Format("Mon, 02/01/2006"), len(events))
	if isFocused {
		header += " [TIÊU ĐIỂM]"
	}
	sb.WriteString(agendaHeaderStyle.Render(header))
	sb.WriteString("\n\n")

	if len(events) == 0 {
		sb.WriteString(eventLocStyle.Render("Không có sự kiện nào cho ngày này.\n\nNhấn 'a' để tạo sự kiện mới."))
		return sb.String()
	}

	for i, ev := range events {
		isSelected := isFocused && i == selectedIdx
		cursor := "  "
		if isSelected {
			cursor = cursorArrowStyle.Render("▶ ")
		}

		timeStr := formatEventTime(ev.StartAt, ev.EndAt, loc)
		syncBadge := ""
		if ev.SyncState != cache.SyncStateSynced {
			syncBadge = " [↻]"
		}

		titleRender := eventTitleStyle.Render(ev.Title)
		if isSelected {
			titleRender = selectedEventStyle.Render(ev.Title)
		}

		sb.WriteString(fmt.Sprintf("%s%s  %s%s\n", cursor, eventTimeStyle.Render(timeStr), titleRender, syncBadge))
		if ev.Location != "" {
			sb.WriteString(fmt.Sprintf("         %s\n", eventLocStyle.Render(ev.Location)))
		}
		if ev.Description != "" {
			sb.WriteString(fmt.Sprintf("         %s\n", eventLocStyle.Render(ev.Description)))
		}
		if i < len(events)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
