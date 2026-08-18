package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"lich-cli/internal/api"
	"lich-cli/internal/config"
)

func RunWeek(args []string) error {
	fs := flag.NewFlagSet("week", flag.ContinueOnError)
	calendarFlag := fs.String("calendar", "", "Filter by calendar ID")
	jsonFlag := fs.Bool("json", false, "Output results as JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil || cfg.Token == "" {
		return fmt.Errorf("not logged in. Please run 'lich login' first")
	}

	client := api.NewClient(cfg.ServerURL, cfg.Token)
	ctx := context.Background()

	now := time.Now()
	loc := now.Location()

	// Compute Monday of current week
	weekday := int(now.Weekday())
	daysSinceMonday := (weekday + 6) % 7
	monday := now.AddDate(0, 0, -daysSinceMonday)
	startOfWeek := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, loc)
	sunday := startOfWeek.AddDate(0, 0, 6)
	endOfWeek := time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), loc)

	events, err := client.ListEvents(ctx, api.EventFilter{
		CalendarID: *calendarFlag,
		From:       startOfWeek.Format(time.RFC3339),
		To:         endOfWeek.Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("failed to fetch events: %w", err)
	}

	if *jsonFlag {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(events)
	}

	// Map events by date string (YYYY-MM-DD)
	eventsByDay := make(map[string][]api.Event)
	for _, event := range events {
		t, err := time.Parse(time.RFC3339, event.StartAt)
		if err == nil {
			dayKey := t.In(loc).Format("2006-01-02")
			eventsByDay[dayKey] = append(eventsByDay[dayKey], event)
		}
	}

	weekHeader := fmt.Sprintf("Week of %s - %s", startOfWeek.Format("Jan 02"), endOfWeek.Format("Jan 02, 2006"))
	fmt.Println(weekHeader)
	fmt.Println(stringsRepeat("=", len(weekHeader)))
	fmt.Println()

	for i := 0; i < 7; i++ {
		currentDay := startOfWeek.AddDate(0, 0, i)
		dayKey := currentDay.Format("2006-01-02")
		dayEvents := eventsByDay[dayKey]

		dayHeader := currentDay.Format("Monday, Jan 02")
		if currentDay.Year() == now.Year() && currentDay.YearDay() == now.YearDay() {
			dayHeader += " (Today)"
		}

		fmt.Println(dayHeader)
		fmt.Println(stringsRepeat("-", len(dayHeader)))

		if len(dayEvents) == 0 {
			fmt.Println("  No events")
		} else {
			for _, event := range dayEvents {
				timeStr := formatTimeRange(event.StartAt, event.EndAt, loc)
				locStr := ""
				if event.Location != "" {
					locStr = fmt.Sprintf(" (%s)", event.Location)
				}
				fmt.Printf("  %s  %s%s\n", timeStr, event.Title, locStr)
			}
		}
		fmt.Println()
	}

	return nil
}
