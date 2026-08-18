package cli

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"lich-cli/internal/api"
	"lich-cli/internal/config"
)

func RunAdd(args []string) error {
	// Reorder args so flags are parsed even if title is specified first
	var flagArgs []string
	var positionalArgs []string

	flagsWithValues := map[string]bool{
		"--date": true, "-date": true,
		"--at": true, "-at": true,
		"--duration": true, "-duration": true,
		"--calendar": true, "-calendar": true,
		"--desc": true, "-desc": true,
		"--location": true, "-location": true,
		"--timezone": true, "-timezone": true,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagName := arg
			if eqIdx := strings.Index(arg, "="); eqIdx != -1 {
				flagName = arg[:eqIdx]
			}
			flagArgs = append(flagArgs, arg)
			if flagsWithValues[flagName] && !strings.Contains(arg, "=") && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		} else {
			positionalArgs = append(positionalArgs, arg)
		}
	}

	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	dateFlag := fs.String("date", "", "Event date (YYYY-MM-DD, default today)")
	atFlag := fs.String("at", "", "Event start time (HH:MM or HH:MM:SS, default next hour)")
	durationFlag := fs.String("duration", "1h", "Event duration (e.g. 30m, 1h, 2h30m)")
	calendarFlag := fs.String("calendar", "", "Calendar ID (optional, defaults to primary)")
	descFlag := fs.String("desc", "", "Event description")
	locationFlag := fs.String("location", "", "Event location")
	timezoneFlag := fs.String("timezone", "", "Event timezone (e.g. Asia/Ho_Chi_Minh, default system)")

	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	if len(positionalArgs) == 0 {
		return fmt.Errorf("event title is required: lich add \"<title>\" [options]")
	}
	title := strings.Join(positionalArgs, " ")

	cfg, err := config.LoadConfig()
	if err != nil || cfg.Token == "" {
		return fmt.Errorf("not logged in. Please run 'lich login' first")
	}

	client := api.NewClient(cfg.ServerURL, cfg.Token)
	ctx := context.Background()

	now := time.Now()
	loc := now.Location()
	if *timezoneFlag != "" {
		parsedLoc, err := time.LoadLocation(*timezoneFlag)
		if err != nil {
			return fmt.Errorf("invalid timezone '%s': %w", *timezoneFlag, err)
		}
		loc = parsedLoc
	}

	// 1. Determine Date
	targetDate := now.In(loc)
	if *dateFlag != "" {
		parsedDate, err := time.ParseInLocation("2006-01-02", *dateFlag, loc)
		if err != nil {
			return fmt.Errorf("invalid date format '%s' (expected YYYY-MM-DD): %w", *dateFlag, err)
		}
		targetDate = parsedDate
	}

	// 2. Determine Time
	hour := (now.Hour() + 1) % 24
	minute := 0
	if *atFlag != "" {
		var h, m int
		_, err := fmt.Sscanf(*atFlag, "%d:%d", &h, &m)
		if err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
			return fmt.Errorf("invalid time format '%s' (expected HH:MM)", *atFlag)
		}
		hour = h
		minute = m
	}

	// 3. Determine Duration
	dur, err := time.ParseDuration(*durationFlag)
	if err != nil || dur <= 0 {
		return fmt.Errorf("invalid duration '%s' (e.g. 30m, 1h): %w", *durationFlag, err)
	}

	startTime := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), hour, minute, 0, 0, loc)
	endTime := startTime.Add(dur)

	event, err := client.CreateEvent(ctx, api.CreateEventRequest{
		Title:       title,
		CalendarID:  *calendarFlag,
		Description: *descFlag,
		StartAt:     startTime.Format(time.RFC3339),
		EndAt:       endTime.Format(time.RFC3339),
		Timezone:    *timezoneFlag,
		Location:    *locationFlag,
	})
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	fmt.Println("✓ Event created")
	fmt.Printf("  ID:       %s\n", event.ID)
	fmt.Printf("  Title:    %s\n", event.Title)
	fmt.Printf("  Time:     %s (%s)\n", formatTimeRange(event.StartAt, event.EndAt, loc), startTime.Format("Mon, 02 Jan 2006"))
	if event.Location != "" {
		fmt.Printf("  Location: %s\n", event.Location)
	}
	if event.Description != "" {
		fmt.Printf("  Notes:    %s\n", event.Description)
	}

	return nil
}
