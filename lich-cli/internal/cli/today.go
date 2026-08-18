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

func formatTimeRange(startStr, endStr string, localLoc *time.Location) string {
	tStart, err1 := time.Parse(time.RFC3339, startStr)
	tEnd, err2 := time.Parse(time.RFC3339, endStr)
	if err1 != nil || err2 != nil {
		return fmt.Sprintf("%s - %s", startStr, endStr)
	}

	startLocal := tStart.In(localLoc)
	endLocal := tEnd.In(localLoc)

	return fmt.Sprintf("%02d:%02d - %02d:%02d", startLocal.Hour(), startLocal.Minute(), endLocal.Hour(), endLocal.Minute())
}

func RunToday(args []string) error {
	fs := flag.NewFlagSet("today", flag.ContinueOnError)
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

	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), loc)

	events, err := client.ListEvents(ctx, api.EventFilter{
		CalendarID: *calendarFlag,
		From:       startOfDay.Format(time.RFC3339),
		To:         endOfDay.Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("failed to fetch events: %w", err)
	}

	if *jsonFlag {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(events)
	}

	dateHeader := now.Format("Monday, January 02, 2006")
	fmt.Println(dateHeader)
	fmt.Println(stringsRepeat("-", len(dateHeader)))

	if len(events) == 0 {
		fmt.Println("No events scheduled for today.")
		return nil
	}

	for _, event := range events {
		timeStr := formatTimeRange(event.StartAt, event.EndAt, loc)
		locStr := ""
		if event.Location != "" {
			locStr = fmt.Sprintf(" (%s)", event.Location)
		}
		fmt.Printf("%s  %s%s\n", timeStr, event.Title, locStr)
	}

	return nil
}

func stringsRepeat(s string, count int) string {
	var result string
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
