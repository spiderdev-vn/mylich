package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spiderdev-vn/mylich/lich-cli/internal/api"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/cache"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/config"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/syncer"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/ui"
)

func formatTimeRange(startStr, endStr string, localLoc *time.Location) string {
	tStart, err1 := time.Parse(time.RFC3339, startStr)
	tEnd, err2 := time.Parse(time.RFC3339, endStr)
	if err1 != nil || err2 != nil {
		return fmt.Sprintf("%s - %s", startStr, endStr)
	}

	startLocal := tStart.In(localLoc)
	endLocal := tEnd.In(localLoc)

	startDay := startLocal.Truncate(24 * time.Hour)
	endDay := endLocal.Truncate(24 * time.Hour)

	if endDay.Equal(startDay) {
		return fmt.Sprintf("%02d:%02d - %02d:%02d", startLocal.Hour(), startLocal.Minute(), endLocal.Hour(), endLocal.Minute())
	}
	return fmt.Sprintf("%02d:%02d %02d/%02d - %02d:%02d %02d/%02d",
		startLocal.Hour(), startLocal.Minute(), startLocal.Day(), int(startLocal.Month()),
		endLocal.Hour(), endLocal.Minute(), endLocal.Day(), int(endLocal.Month()))
}

func RunToday(args []string) error {
	fs := flag.NewFlagSet("today", flag.ContinueOnError)
	calendarFlag := fs.String("calendar", "", "Lọc theo calendar ID")
	jsonFlag := fs.Bool("json", false, "Xuất kết quả dưới định dạng JSON")
	simpleFlag := fs.Bool("simple", false, "Hiển thị dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị dạng văn bản ASCII đơn giản (viết tắt)")

	fs.Usage = func() {
		fmt.Println("Sử dụng: lich today [flags]")
		fmt.Println()
		fmt.Println("Tùy chọn:")
		fmt.Println("  --calendar <id>  Lọc sự kiện theo ID lịch")
		fmt.Println("  --simple, -s     Hiển thị dạng văn bản ASCII đơn giản")
		fmt.Println("  --json           Xuất kết quả dưới định dạng JSON")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
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

	icons := ui.CurrentIcons()
	if *simpleFlag {
		icons = ui.IconASCII
	}

	// Lip Gloss Styled Agenda
	headerText := fmt.Sprintf(" %s LỊCH TRÌNH HÔM NAY — %s ", icons.Calendar, now.Format("Monday, 02/01/2006"))
	fmt.Println(ui.HeaderDateStyle.Render(headerText))

	if len(events) == 0 {
		fmt.Println(ui.EventDescStyle.Render("  (Không có sự kiện nào được lên lịch cho hôm nay)"))
		fmt.Println()
		return nil
	}

	for _, event := range events {
		timeStr := formatTimeRange(event.StartAt, event.EndAt, loc)
		syncBadge := ""
		if event.SyncState != cache.SyncStateSynced {
			syncBadge = " " + ui.RenderBadgePending(icons.Pending)
		}

		fmt.Printf("  %s  %s%s\n",
			ui.TimePill.Render(timeStr),
			ui.EventTitleStyle.Render(event.Title),
			syncBadge,
		)
		if event.Location != "" {
			fmt.Printf("            %s %s\n", ui.LabelStyle.Render(icons.Location), ui.EventLocationStyle.Render(event.Location))
		}
		if event.Description != "" {
			fmt.Printf("            %s\n", ui.EventDescStyle.Render(event.Description))
		}
		fmt.Println()
	}

	return nil
}
