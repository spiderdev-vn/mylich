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

func RunWeek(args []string) error {
	fs := flag.NewFlagSet("week", flag.ContinueOnError)
	calendarFlag := fs.String("calendar", "", "Lọc theo calendar ID")
	detailFlag := fs.Bool("detail", false, "Hiển thị chi tiết bao gồm ID sự kiện")
	fs.BoolVar(detailFlag, "d", false, "Hiển thị chi tiết bao gồm ID sự kiện (viết tắt)")
	idFlag := fs.Bool("id", false, "Hiển thị ID sự kiện")
	jsonFlag := fs.Bool("json", false, "Xuất kết quả dưới định dạng JSON")
	simpleFlag := fs.Bool("simple", false, "Hiển thị dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị dạng văn bản ASCII đơn giản (viết tắt)")

	fs.Usage = func() {
		if ui.IsSimpleMode(*simpleFlag) {
			fmt.Println("Sử dụng: lich week [flags]")
			fmt.Println()
			fmt.Println("Tùy chọn:")
			fmt.Println("  --detail, -d     Hiển thị chi tiết bao gồm ID sự kiện")
			fmt.Println("  --id             Hiển thị ID sự kiện")
			fmt.Println("  --calendar <id>  Lọc sự kiện theo ID lịch")
			fmt.Println("  --simple, -s     Hiển thị dạng văn bản ASCII đơn giản")
			fmt.Println("  --json           Xuất kết quả dưới định dạng JSON")
			return
		}

		banner := ui.TitleBanner.Render(" LỊCH TRÌNH TUẦN (LICH WEEK) ")
		fmt.Println(banner)

		helpContent := fmt.Sprintf(`%s
  %s     %s
  %s  %s

%s
  %s      %s
  %s            %s
  %s %s
  %s          %s
  %s     %s`,
			ui.CardTitle.Render("CÚ PHÁP:"),
			ui.ValueStyle.Render("lich week"), ui.LabelStyle.Render("Xem toàn bộ lịch trình 7 ngày trong tuần"),
			ui.ValueStyle.Render("lich week -d"), ui.LabelStyle.Render("Hiện chi tiết kèm Event ID từng sự kiện"),
			ui.CardTitle.Render("TÙY CHỌN & CỜ:"),
			ui.ValueStyle.Render("--detail, -d"), ui.LabelStyle.Render("Hiển thị chi tiết bao gồm ID sự kiện"),
			ui.ValueStyle.Render("--id"), ui.LabelStyle.Render("Hiển thị ID sự kiện"),
			ui.ValueStyle.Render("--calendar <id>"), ui.LabelStyle.Render("Lọc sự kiện theo ID lịch cụ thể"),
			ui.ValueStyle.Render("--json"), ui.LabelStyle.Render("Xuất kết quả dưới định dạng JSON"),
			ui.ValueStyle.Render("--simple, -s"), ui.LabelStyle.Render("Hiển thị dạng văn bản ASCII đơn giản"),
		)
		fmt.Println(ui.CardBox.Width(78).Render(helpContent))
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
		tStart, err1 := time.Parse(time.RFC3339, event.StartAt)
		tEnd, err2 := time.Parse(time.RFC3339, event.EndAt)
		if err1 != nil {
			continue
		}
		if err2 != nil {
			tEnd = tStart
		}
		startLocal := tStart.In(loc)
		endLocal := tEnd.In(loc)

		cur := time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day(), 0, 0, 0, 0, loc)
		lastDay := endLocal
		if lastDay.Hour() == 0 && lastDay.Minute() == 0 && lastDay.Second() == 0 && lastDay.After(startLocal) {
			lastDay = lastDay.Add(-time.Second)
		}
		endDay := time.Date(lastDay.Year(), lastDay.Month(), lastDay.Day(), 0, 0, 0, 0, loc)

		for !cur.After(endDay) {
			dayKey := cur.Format("2006-01-02")
			eventsByDay[dayKey] = append(eventsByDay[dayKey], event)
			cur = cur.AddDate(0, 0, 1)
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
					idSuffix := ""
					if *detailFlag || *idFlag {
						idSuffix = fmt.Sprintf(" [id: %s]", event.ID)
					}
					fmt.Printf("  %s  %s%s%s%s\n", timeStr, event.Title, locStr, syncBadge, idSuffix)
				}
			}
			fmt.Println()
		}
		return nil
	}

	icons := ui.CurrentIcons()
	if *simpleFlag {
		icons = ui.IconASCII
	}

	// Lip Gloss Styled Week
	weekHeader := fmt.Sprintf(" %s LỊCH TUẦN: %s — %s ", icons.Calendar, startOfWeek.Format("02/01"), endOfWeek.Format("02/01/2006"))
	fmt.Println(ui.TitleBanner.Render(weekHeader))

	for i := 0; i < 7; i++ {
		currentDay := startOfWeek.AddDate(0, 0, i)
		dayKey := currentDay.Format("2006-01-02")
		dayEvents := eventsByDay[dayKey]

		dayTitle := currentDay.Format("Monday, 02/01")
		isToday := currentDay.Year() == now.Year() && currentDay.YearDay() == now.YearDay()
		if isToday {
			dayTitle += " " + icons.TagToday
			fmt.Println(ui.HeaderDateStyle.Render(" " + dayTitle + " "))
		} else {
			fmt.Println(ui.SubTitleStyle.Render(fmt.Sprintf("%s %s", icons.Arrow, dayTitle)))
		}

		if len(dayEvents) == 0 {
			fmt.Println(ui.EventDescStyle.Render("   (Không có sự kiện)"))
		} else {
			for _, event := range dayEvents {
				timeStr := formatTimeRange(event.StartAt, event.EndAt, loc)
				syncBadge := ""
				if event.SyncState != cache.SyncStateSynced {
					syncBadge = " " + ui.RenderBadgePending(icons.Pending)
				}
				fmt.Printf("   %s  %s%s\n",
					ui.TimePill.Render(timeStr),
					ui.EventTitleStyle.Render(event.Title),
					syncBadge,
				)
				if event.Location != "" {
					fmt.Printf("             %s %s\n", ui.LabelStyle.Render(icons.Location), ui.EventLocationStyle.Render(event.Location))
				}
				if *detailFlag || *idFlag {
					fmt.Printf("             %s %s\n", ui.LabelStyle.Render("ID:"), ui.EventDescStyle.Render(event.ID))
				}
			}
		}
		fmt.Println()
	}

	return nil
}
