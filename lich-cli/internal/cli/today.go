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

	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).UTC().Format(time.RFC3339)
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), loc).UTC().Format(time.RFC3339)

	// Đọc tức thì từ SQLite cục bộ (0ms)
	events, err := cache.GetEventsInRange(db, startOfDay, endOfDay, *calendarFlag)
	if err != nil {
		return fmt.Errorf("lỗi đọc sự kiện hôm nay từ cache: %w", err)
	}

	// Kích hoạt đồng bộ hóa ngầm nếu đã cấu hình
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

	if ui.IsSimpleMode(*simpleFlag) {
		dateHeader := now.Format("Monday, January 02, 2006")
		fmt.Println(dateHeader)
		fmt.Println(strings.Repeat("-", len(dateHeader)))

		if len(events) == 0 {
			fmt.Println("Không có sự kiện nào được lên lịch hôm nay.")
			return nil
		}

		for _, event := range events {
			timeStr := formatTimeRange(event.StartAt, event.EndAt, loc)
			locStr := ""
			if event.Location != "" {
				locStr = fmt.Sprintf(" (%s)", event.Location)
			}
			syncBadge := ""
			if event.SyncState != cache.SyncStateSynced {
				syncBadge = " [↻ pending]"
			}
			fmt.Printf("%s  %s%s%s\n", timeStr, event.Title, locStr, syncBadge)
		}
		return nil
	}

	// Lip Gloss Styled Agenda
	headerText := fmt.Sprintf(" 📅 LỊCH TRÌNH HÔM NAY — %s ", now.Format("Monday, 02/01/2006"))
	fmt.Println(ui.HeaderDateStyle.Render(headerText))
	fmt.Println()

	if len(events) == 0 {
		fmt.Println(ui.EventDescStyle.Render("  (Không có sự kiện nào được lên lịch cho hôm nay)"))
		fmt.Println()
		return nil
	}

	for _, event := range events {
		timeStr := formatTimeRange(event.StartAt, event.EndAt, loc)
		syncBadge := ""
		if event.SyncState != cache.SyncStateSynced {
			syncBadge = " " + ui.BadgePending
		}

		fmt.Printf("  %s  %s%s\n",
			ui.TimePill.Render(timeStr),
			ui.EventTitleStyle.Render(event.Title),
			syncBadge,
		)
		if event.Location != "" {
			fmt.Printf("            %s\n", ui.EventLocationStyle.Render("📍 "+event.Location))
		}
		if event.Description != "" {
			fmt.Printf("            %s\n", ui.EventDescStyle.Render(event.Description))
		}
		fmt.Println()
	}

	return nil
}
