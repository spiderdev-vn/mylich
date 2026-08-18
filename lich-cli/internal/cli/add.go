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
	hasDurationFlag := false

	flagsWithValues := map[string]bool{
		"--date": true, "-date": true,
		"--at": true, "-at": true,
		"--to": true, "-to": true,
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
			if flagName == "--duration" || flagName == "-duration" {
				hasDurationFlag = true
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
	dateFlag := fs.String("date", "", "Event date (YYYY-MM-DD, today, tomorrow)")
	atFlag := fs.String("at", "", "Event start time (e.g. 10:00, 23:30, 11:30pm)")
	toFlag := fs.String("to", "", "Event end time (e.g. 22:33, 3am, 03:00)")
	durationFlag := fs.String("duration", "1h", "Event duration (e.g. 30m, 1h, 2h30m)")
	calendarFlag := fs.String("calendar", "", "Target calendar ID")
	descFlag := fs.String("desc", "", "Event description")
	locationFlag := fs.String("location", "", "Event location")
	timezoneFlag := fs.String("timezone", "", "Event timezone (e.g. Asia/Ho_Chi_Minh)")

	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	if len(positionalArgs) == 0 {
		return fmt.Errorf("tiêu đề sự kiện là bắt buộc: lich add \"<tiêu đề>\" [tùy chọn]")
	}
	title := strings.Join(positionalArgs, " ")

	cfg, err := config.LoadConfig()
	if err != nil || cfg.Token == "" {
		return fmt.Errorf("chưa đăng nhập. Vui lòng chạy 'lich login' trước")
	}

	client := api.NewClient(cfg.ServerURL, cfg.Token)
	ctx := context.Background()

	now := time.Now()
	loc := now.Location()
	if *timezoneFlag != "" {
		parsedLoc, err := time.LoadLocation(*timezoneFlag)
		if err != nil {
			return fmt.Errorf("múi giờ không hợp lệ '%s': %w", *timezoneFlag, err)
		}
		loc = parsedLoc
	}

	startTime, endTime, isOvernight, err := parseFlexibleTimeRange(
		*dateFlag,
		*atFlag,
		*toFlag,
		*durationFlag,
		hasDurationFlag,
		loc,
	)
	if err != nil {
		return err
	}

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
		return fmt.Errorf("tạo sự kiện thất bại: %w", err)
	}

	fmt.Println("✓ Đã tạo sự kiện")
	fmt.Printf("  ID:       %s\n", event.ID)
	fmt.Printf("  Tiêu đề:  %s\n", event.Title)

	if startTime.Format("2006-01-02") == endTime.Format("2006-01-02") {
		fmt.Printf("  Thời gian: %s (%s)\n", formatTimeRange(event.StartAt, event.EndAt, loc), startTime.Format("Mon, 02 Jan 2006"))
	} else {
		fmt.Printf("  Thời gian: %s %s -> %s %s\n",
			startTime.Format("15:04"), startTime.Format("02/01/2006"),
			endTime.Format("15:04"), endTime.Format("02/01/2006"),
		)
	}

	if isOvernight {
		fmt.Printf("  ℹ Lưu ý: Sự kiện kéo dài qua đêm, kết thúc lúc %s ngày %s.\n",
			endTime.Format("15:04"),
			endTime.Format("Monday, 02/01/2006"),
		)
	}

	if event.Location != "" {
		fmt.Printf("  Địa điểm: %s\n", event.Location)
	}
	if event.Description != "" {
		fmt.Printf("  Ghi chú:  %s\n", event.Description)
	}

	return nil
}
