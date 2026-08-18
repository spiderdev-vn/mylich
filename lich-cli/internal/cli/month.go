package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"lich-cli/internal/api"
	"lich-cli/internal/cache"
	"lich-cli/internal/config"
	"lich-cli/internal/syncer"
	"lich-cli/internal/ui"
)

func RunMonth(args []string) error {
	fs := flag.NewFlagSet("month", flag.ContinueOnError)
	calendarFlag := fs.String("calendar", "", "Lọc theo calendar ID")
	jsonFlag := fs.Bool("json", false, "Xuất kết quả dưới định dạng JSON")
	simpleFlag := fs.Bool("simple", false, "Hiển thị dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị dạng văn bản ASCII đơn giản (viết tắt)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	cachePath, err := cache.GetCachePath()
	if err != nil {
		return fmt.Errorf("không thể mở thư mục cache: %w", err)
	}

	db, err := cache.OpenDatabase(cachePath)
	if err != nil {
		return fmt.Errorf("lỗi kết nối database cục bộ: %w", err)
	}
	defer db.Close()

	now := time.Now()
	loc := now.Location()

	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	endOfMonth := startOfMonth.AddDate(0, 1, -1)
	endOfMonthDay := time.Date(endOfMonth.Year(), endOfMonth.Month(), endOfMonth.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), loc)

	events, err := cache.GetEventsInRange(db, startOfMonth.UTC().Format(time.RFC3339), endOfMonthDay.UTC().Format(time.RFC3339), *calendarFlag)
	if err != nil {
		return fmt.Errorf("lỗi đọc sự kiện tháng từ cache: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err == nil && cfg.Token != "" {
		client := api.NewClient(cfg.ServerURL, cfg.Token)
		engine := syncer.NewSyncEngine(db, client)
		engine.SyncInBackground()
	}

	if *jsonFlag {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(events)
	}

	eventsByDay := make(map[string][]cache.LocalEvent)
	for _, event := range events {
		t, err := time.Parse(time.RFC3339, event.StartAt)
		if err == nil {
			dayKey := t.In(loc).Format("2006-01-02")
			eventsByDay[dayKey] = append(eventsByDay[dayKey], event)
		}
	}

	if ui.IsSimpleMode(*simpleFlag) {
		monthHeader := fmt.Sprintf("Lịch tháng %s (%d sự kiện)", now.Format("01/2006"), len(events))
		fmt.Println(monthHeader)
		fmt.Println(strings.Repeat("=", len(monthHeader)))
		fmt.Println()

		if len(events) == 0 {
			fmt.Println("Không có sự kiện nào trong tháng này.")
			return nil
		}

		for d := 1; d <= endOfMonth.Day(); d++ {
			currentDate := time.Date(now.Year(), now.Month(), d, 0, 0, 0, 0, loc)
			dayKey := currentDate.Format("2006-01-02")
			dayEvents, exists := eventsByDay[dayKey]
			if !exists || len(dayEvents) == 0 {
				continue
			}

			dayHeader := currentDate.Format("Monday, 02/01/2006")
			if currentDate.Year() == now.Year() && currentDate.YearDay() == now.YearDay() {
				dayHeader += " (Hôm nay)"
			}

			fmt.Println(dayHeader)
			for _, e := range dayEvents {
				timeStr := formatTimeRange(e.StartAt, e.EndAt, loc)
				locStr := ""
				if e.Location != "" {
					locStr = fmt.Sprintf(" (%s)", e.Location)
				}
				fmt.Printf("  • %s: %s%s\n", timeStr, e.Title, locStr)
			}
			fmt.Println()
		}
		return nil
	}

	// Lip Gloss Styled Month
	monthHeader := fmt.Sprintf(" 🗓 LỊCH THÁNG %s — TỔNG CỘNG %d SỰ KIỆN ", now.Format("01/2006"), len(events))
	fmt.Println(ui.TitleBanner.Render(monthHeader))

	if len(events) == 0 {
		fmt.Println(ui.EventDescStyle.Render("  (Không có sự kiện nào trong tháng này)"))
		fmt.Println()
		return nil
	}

	for d := 1; d <= endOfMonth.Day(); d++ {
		currentDate := time.Date(now.Year(), now.Month(), d, 0, 0, 0, 0, loc)
		dayKey := currentDate.Format("2006-01-02")
		dayEvents, exists := eventsByDay[dayKey]
		if !exists || len(dayEvents) == 0 {
			continue
		}

		dayTitle := currentDate.Format("Monday, 02/01/2006")
		isToday := currentDate.Year() == now.Year() && currentDate.YearDay() == now.YearDay()
		if isToday {
			dayTitle += " ★ HÔM NAY"
			fmt.Println(ui.HeaderDateStyle.Render(" " + dayTitle + " "))
		} else {
			fmt.Println(ui.SubTitleStyle.Render("▶ " + dayTitle))
		}

		for _, e := range dayEvents {
			timeStr := formatTimeRange(e.StartAt, e.EndAt, loc)
			syncBadge := ""
			if e.SyncState != cache.SyncStateSynced {
				syncBadge = " " + ui.BadgePending
			}
			fmt.Printf("   %s  %s%s\n",
				ui.TimePill.Render(timeStr),
				ui.EventTitleStyle.Render(e.Title),
				syncBadge,
			)
			if e.Location != "" {
				fmt.Printf("             %s\n", ui.EventLocationStyle.Render("📍 "+e.Location))
			}
		}
		fmt.Println()
	}

	return nil
}
