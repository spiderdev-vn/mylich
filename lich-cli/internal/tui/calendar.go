package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"lich-cli/internal/api"
)

type DayInfo struct {
	Date        time.Time
	InMonth     bool
	IsToday     bool
	IsSelected  bool
	HasEvents   bool
	EventsCount int
}

func GetMonthDays(year int, month time.Month, selectedDate time.Time, events map[string][]api.Event, loc *time.Location) [][]DayInfo {
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	now := time.Now().In(loc)

	// In Go: Sunday=0, Monday=1, ...
	weekday := int(firstOfMonth.Weekday())
	daysBefore := (weekday + 6) % 7 // Monday-based: Monday=0, Sunday=6

	startDate := firstOfMonth.AddDate(0, 0, -daysBefore)

	var weeks [][]DayInfo
	current := startDate

	for w := 0; w < 6; w++ {
		var week []DayInfo
		for d := 0; d < 7; d++ {
			inMonth := current.Month() == month
			isToday := current.Year() == now.Year() && current.Month() == now.Month() && current.Day() == now.Day()
			isSelected := current.Year() == selectedDate.Year() && current.Month() == selectedDate.Month() && current.Day() == selectedDate.Day()

			dayKey := current.Format("2006-01-02")
			evs := events[dayKey]
			hasEvents := len(evs) > 0

			week = append(week, DayInfo{
				Date:        current,
				InMonth:     inMonth,
				IsToday:     isToday,
				IsSelected:  isSelected,
				HasEvents:   hasEvents,
				EventsCount: len(evs),
			})

			current = current.AddDate(0, 0, 1)
		}
		weeks = append(weeks, week)
	}

	return weeks
}

func RenderCalendarGrid(year int, month time.Month, selectedDate time.Time, events map[string][]api.Event, loc *time.Location) string {
	var sb strings.Builder

	// Header: Month Year
	monthName := fmt.Sprintf("%s %d", month.String(), year)
	header := monthHeaderStyle.Render(monthName)
	sb.WriteString(lipgloss.NewStyle().Width(28).Align(lipgloss.Center).Render(header))
	sb.WriteString("\n\n")

	// Weekdays: Mo Tu We Th Fr Sa Su
	weekdays := []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
	var wdHeaders []string
	for _, wd := range weekdays {
		wdHeaders = append(wdHeaders, weekdayHeaderStyle.Render(wd))
	}
	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, wdHeaders...))
	sb.WriteString("\n")

	// Weeks
	weeks := GetMonthDays(year, month, selectedDate, events, loc)
	for _, week := range weeks {
		var dayCells []string
		for _, day := range week {
			dayNum := fmt.Sprintf("%2d", day.Date.Day())
			if day.HasEvents && !day.IsSelected {
				dayNum = fmt.Sprintf("%d•", day.Date.Day())
				if day.Date.Day() < 10 {
					dayNum = fmt.Sprintf(" %d•", day.Date.Day())
				}
			}

			var cell string
			switch {
			case day.IsSelected:
				cell = daySelectedStyle.Render(dayNum)
			case day.IsToday:
				cell = dayTodayStyle.Render(dayNum)
			case !day.InMonth:
				cell = dayOtherMonthStyle.Render(dayNum)
			case day.HasEvents:
				cell = dayWithEventsStyle.Render(dayNum)
			default:
				cell = dayCellStyle.Render(dayNum)
			}
			dayCells = append(dayCells, cell)
		}
		sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, dayCells...))
		sb.WriteString("\n")
	}

	return sb.String()
}
