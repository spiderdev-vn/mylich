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

func RunWeek(args []string) error {
	fs := flag.NewFlagSet("week", flag.ContinueOnError)
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

	weekday := int(now.Weekday())
	daysSinceMonday := (weekday + 6) % 7
	monday := now.AddDate(0, 0, -daysSinceMonday)
	startOfWeek := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, loc)
	sunday := startOfWeek.AddDate(0, 0, 6)
	endOfWeek := time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), loc)

	events, err := cache.GetEventsInRange(db, startOfWeek.UTC().Format(time.RFC3339), endOfWeek.UTC().Format(time.RFC3339), *calendarFlag)
	if err != nil {
		return fmt.Errorf("lỗi đọc sự kiện tuần từ cache: %w", err)
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
		weekHeader := fmt.Sprintf("Tuần từ %s đến %s", startOfWeek.Format("02/01"), endOfWeek.Format("02/01/2006"))
		fmt.Println(weekHeader)
		fmt.Println(strings.Repeat("=", len(weekHeader)))
		fmt.Println()

		for i := 0; i < 7; i++ {
			currentDay := startOfWeek.AddDate(0, 0, i)
			dayKey := currentDay.Format("2006-01-02")
			dayEvents := eventsByDay[dayKey]

			dayHeader := currentDay.Format("Monday, 02/01")
			if currentDay.Year() == now.Year() && currentDay.YearDay() == now.YearDay() {
				dayHeader += " (Hôm nay)"
			}

			fmt.Println(dayHeader)
			fmt.Println(strings.Repeat("-", len(dayHeader)))

			if len(dayEvents) == 0 {
				fmt.Println("  (Không có sự kiện)")
			} else {
				for _, event := range dayEvents {
					timeStr := formatTimeRange(event.StartAt, event.EndAt, loc)
					locStr := ""
					if event.Location != "" {
						locStr = fmt.Sprintf(" (%s)", event.Location)
					}
					syncBadge := ""
					if event.SyncState != cache.SyncStateSynced {
						syncBadge = " [↻ pending]"
					}
					fmt.Printf("  %s  %s%s%s\n", timeStr, event.Title, locStr, syncBadge)
				}
			}
			fmt.Println()
		}
		return nil
	}

	// Lip Gloss Styled Week
	weekHeader := fmt.Sprintf(" 🗓 LỊCH TUẦN: %s — %s ", startOfWeek.Format("02/01"), endOfWeek.Format("02/01/2006"))
	fmt.Println(ui.TitleBanner.Render(weekHeader))

	for i := 0; i < 7; i++ {
		currentDay := startOfWeek.AddDate(0, 0, i)
		dayKey := currentDay.Format("2006-01-02")
		dayEvents := eventsByDay[dayKey]

		dayTitle := currentDay.Format("Monday, 02/01")
		isToday := currentDay.Year() == now.Year() && currentDay.YearDay() == now.YearDay()
		if isToday {
			dayTitle += " ★ HÔM NAY"
			fmt.Println(ui.HeaderDateStyle.Render(" " + dayTitle + " "))
		} else {
			fmt.Println(ui.SubTitleStyle.Render("▶ " + dayTitle))
		}

		if len(dayEvents) == 0 {
			fmt.Println(ui.EventDescStyle.Render("   (Không có sự kiện)"))
		} else {
			for _, event := range dayEvents {
				timeStr := formatTimeRange(event.StartAt, event.EndAt, loc)
				syncBadge := ""
				if event.SyncState != cache.SyncStateSynced {
					syncBadge = " " + ui.BadgePending
				}
				fmt.Printf("   %s  %s%s\n",
					ui.TimePill.Render(timeStr),
					ui.EventTitleStyle.Render(event.Title),
					syncBadge,
				)
				if event.Location != "" {
					fmt.Printf("             %s\n", ui.EventLocationStyle.Render("📍 "+event.Location))
				}
			}
		}
		fmt.Println()
	}

	return nil
}
