package tui

import (
	"fmt"
	"strings"
	"time"

	"lich-cli/internal/api"
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

func RenderAgenda(selectedDate time.Time, events []api.Event, loc *time.Location) string {
	var sb strings.Builder

	header := fmt.Sprintf("Agenda: %s", selectedDate.Format("Mon, Jan 02, 2006"))
	sb.WriteString(agendaHeaderStyle.Render(header))
	sb.WriteString("\n\n")

	if len(events) == 0 {
		sb.WriteString(eventLocStyle.Render("No events scheduled for this day."))
		return sb.String()
	}

	for i, ev := range events {
		timeStr := formatEventTime(ev.StartAt, ev.EndAt, loc)
		sb.WriteString(fmt.Sprintf("%s  %s\n", eventTimeStyle.Render(timeStr), eventTitleStyle.Render(ev.Title)))
		if ev.Location != "" {
			sb.WriteString(fmt.Sprintf("       %s\n", eventLocStyle.Render("📍 "+ev.Location)))
		}
		if ev.Description != "" {
			sb.WriteString(fmt.Sprintf("       %s\n", eventLocStyle.Render(ev.Description)))
		}
		if i < len(events)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
