package cli

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/spiderdev-vn/mylich/lich-cli/internal/api"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/cache"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/config"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/syncer"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/ui"

	"github.com/charmbracelet/huh"
)

func RunEdit(args []string) error {
	var flagArgs []string
	var positionalArgs []string
	hasDurationFlag := false

	flagsWithValues := map[string]bool{
		"--title": true, "-title": true,
		"--date": true, "-date": true,
		"--end-date": true, "-end-date": true,
		"--to-date": true, "-to-date": true,
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

	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	titleFlag := fs.String("title", "", "Tiêu đề sự kiện mới")
	dateFlag := fs.String("date", "", "Ngày diễn ra (YYYY-MM-DD, today, tomorrow)")
	endDateFlag := fs.String("end-date", "", "Ngày kết thúc sự kiện (Mặc định: cùng ngày diễn ra)")
	fs.StringVar(endDateFlag, "to-date", "", "Ngày kết thúc sự kiện (viết tắt)")
	atFlag := fs.String("at", "", "Giờ bắt đầu (10:00, 23:30, 11:30pm)")
	toFlag := fs.String("to", "", "Giờ kết thúc (22:33, 03:00, 3am)")
	durationFlag := fs.String("duration", "", "Thời lượng sự kiện (ví dụ: 30m, 1h, 2h30m)")
	calendarFlag := fs.String("calendar", "", "ID lịch đích")
	descFlag := fs.String("desc", "", "Ghi chú mô tả")
	locationFlag := fs.String("location", "", "Địa điểm diễn ra")
	timezoneFlag := fs.String("timezone", "", "Múi giờ sự kiện (ví dụ: Asia/Ho_Chi_Minh)")
	simpleFlag := fs.Bool("simple", false, "Hiển thị dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị dạng văn bản ASCII đơn giản (viết tắt)")

	fs.Usage = func() {
		fmt.Println("Sử dụng: lich edit <id> [flags]")
		fmt.Println()
		fmt.Println("Mô tả:")
		fmt.Println("  Chỉnh sửa sự kiện đã có. Nếu không truyền cờ, sẽ mở Form tương tác Huh với dữ liệu hiện tại.")
		fmt.Println()
		fmt.Println("Tùy chọn:")
		fmt.Println("  --title <text>        Tiêu đề sự kiện mới")
		fmt.Println("  --date <date>         Ngày diễn ra (YYYY-MM-DD, today, tomorrow)")
		fmt.Println("  --end-date <date>     Ngày kết thúc (Mặc định: cùng ngày với --date)")
		fmt.Println("  --at <time>           Giờ bắt đầu (10:00, 23:30, 11:30pm)")
		fmt.Println("  --to <time>           Giờ kết thúc (22:33, 03:00, 3am)")
		fmt.Println("  --duration <duration> Thời lượng sự kiện (ví dụ: 30m, 1h, 2h30m)")
		fmt.Println("  --calendar <id>       ID lịch đích")
		fmt.Println("  --location <text>     Địa điểm")
		fmt.Println("  --desc <text>         Ghi chú mô tả")
		fmt.Println("  --timezone <tz>       Múi giờ (ví dụ: Asia/Ho_Chi_Minh)")
		fmt.Println("  --simple, -s          Hiển thị dạng văn bản ASCII đơn giản")
	}

	if err := fs.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if len(positionalArgs) == 0 {
		return fmt.Errorf("cần chỉ định ID sự kiện: lich edit <id> [tùy chọn]")
	}
	eventID := positionalArgs[0]

	// 1. Mở cache database cục bộ
	cachePath, err := cache.GetCachePath()
	if err != nil {
		return fmt.Errorf("không thể mở thư mục cache: %w", err)
	}

	db, err := cache.OpenDatabase(cachePath)
	if err != nil {
		return fmt.Errorf("lỗi khởi động database cục bộ: %w", err)
	}
	defer db.Close()

	// 2. Lấy sự kiện hiện tại
	existingEvent, err := cache.GetEvent(db, eventID)
	if err != nil || existingEvent == nil {
		return fmt.Errorf("không tìm thấy sự kiện có ID '%s'", eventID)
	}

	loc := time.Now().Location()
	if existingEvent.Timezone != "" && existingEvent.Timezone != "UTC" && existingEvent.Timezone != "Local" {
		if l, err := time.LoadLocation(existingEvent.Timezone); err == nil {
			loc = l
		}
	}

	curStart, _ := time.Parse(time.RFC3339, existingEvent.StartAt)
	curEnd, _ := time.Parse(time.RFC3339, existingEvent.EndAt)
	curStartLocal := curStart.In(loc)
	curEndLocal := curEnd.In(loc)

	titleVal := existingEvent.Title
	if *titleFlag != "" {
		titleVal = *titleFlag
	}

	dateVal := curStartLocal.Format("2006-01-02")
	if *dateFlag != "" {
		dateVal = *dateFlag
	}

	atVal := curStartLocal.Format("15:04")
	if *atFlag != "" {
		atVal = *atFlag
	}

	toVal := curEndLocal.Format("15:04")
	if *toFlag != "" {
		toVal = *toFlag
	}

	locVal := existingEvent.Location
	if *locationFlag != "" {
		locVal = *locationFlag
	}

	descVal := existingEvent.Description
	if *descFlag != "" {
		descVal = *descFlag
	}

	calVal := existingEvent.CalendarID
	if *calendarFlag != "" {
		calVal = *calendarFlag
	}

	tzVal := existingEvent.Timezone
	if *timezoneFlag != "" {
		tzVal = *timezoneFlag
	}

	// 3. Nếu không có cờ nào được truyền qua CLI và ở chế độ interactive, mở Form Huh điền sẵn
	hasAnyFlag := *titleFlag != "" || *dateFlag != "" || *endDateFlag != "" || *atFlag != "" || *toFlag != "" || *durationFlag != "" || *locationFlag != "" || *descFlag != "" || *calendarFlag != "" || *timezoneFlag != ""
	if !hasAnyFlag && !ui.IsSimpleMode(*simpleFlag) {
		reqStar := ui.RequiredAsterisk

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fmt.Sprintf("Tiêu đề sự kiện %s", reqStar)).
					Value(&titleVal).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("tiêu đề không được để trống")
						}
						return nil
					}),

				huh.NewInput().
					Title(fmt.Sprintf("Ngày %s", reqStar)).
					Value(&dateVal),

				huh.NewInput().
					Title(fmt.Sprintf("Bắt đầu (--at) %s", reqStar)).
					Value(&atVal),

				huh.NewInput().
					Title(fmt.Sprintf("Kết thúc (--to) %s", reqStar)).
					Value(&toVal),

				huh.NewInput().
					Title("Địa điểm (Tùy chọn)").
					Value(&locVal),

				huh.NewInput().
					Title("Ghi chú (Tùy chọn)").
					Value(&descVal),
			),
		).WithKeyMap(ui.DefaultFormKeyMap())

		if err := form.Run(); err != nil {
			return err
		}
	}

	// 4. Tính toán thời gian mới
	startTime, endTime, _, err := parseFlexibleTimeRangeWithEndDate(
		dateVal,
		*endDateFlag,
		atVal,
		toVal,
		*durationFlag,
		hasDurationFlag,
		loc,
	)
	if err != nil {
		return err
	}

	startRFC := startTime.UTC().Format(time.RFC3339)
	endRFC := endTime.UTC().Format(time.RFC3339)

	updatedEvent := cache.LocalEvent{
		ID:          eventID,
		CalendarID:  calVal,
		Title:       titleVal,
		Description: descVal,
		StartAt:     startRFC,
		EndAt:       endRFC,
		Timezone:    tzVal,
		Location:    locVal,
		SyncState:   cache.SyncStatePendingUpdate,
	}

	if err := cache.UpsertEvent(db, updatedEvent); err != nil {
		return fmt.Errorf("lỗi cập nhật sự kiện cục bộ: %w", err)
	}

	// 5. Đưa job UPDATE vào hàng đợi
	_, _ = cache.EnqueueSyncJob(db, "event", eventID, cache.SyncOpUpdate, syncer.MarshalPayload(updatedEvent))

	// 6. Kích hoạt sync ngầm
	cfg, err := config.LoadConfig()
	if err == nil && cfg.Token != "" {
		client := api.NewClient(cfg.ServerURL, cfg.Token)
		engine := syncer.NewSyncEngine(db, client)
		engine.SyncInBackground()
	}

	// 7. Hiển thị thông báo
	icons := ui.CurrentIcons()
	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Printf("✓ Đã cập nhật sự kiện %s\n", eventID)
		fmt.Printf("  Tiêu đề:   %s\n", titleVal)
		fmt.Printf("  Thời gian: %s - %s\n", startTime.Format("15:04"), endTime.Format("15:04 02/01/2006"))
		fmt.Println("  Đồng bộ:   [PENDING] Đang đồng bộ ngầm")
		return nil
	}

	timeStr := formatTimeRange(startRFC, endRFC, loc)
	cardContent := fmt.Sprintf(
		"%s\n\n%s %s\n%s %s\n%s %s (%s)\n%s %s",
		ui.CardTitle.Render("✓ ĐÃ CẬP NHẬT SỰ KIỆN"),
		ui.LabelStyle.Render("Tiêu đề:  "), ui.ValueStyle.Render(titleVal),
		ui.LabelStyle.Render("ID:       "), ui.LabelStyle.Render(eventID),
		ui.LabelStyle.Render("Thời gian:"), ui.TimePill.Render(timeStr), startTime.Format("Monday, 02/01/2006"),
		ui.LabelStyle.Render("Đồng bộ:  "), ui.RenderBadgePending(icons.Pending),
	)

	if locVal != "" {
		cardContent += fmt.Sprintf("\n%s %s", ui.LabelStyle.Render("Địa điểm: "), ui.ValueStyle.Render(locVal))
	}
	if descVal != "" {
		cardContent += fmt.Sprintf("\n%s %s", ui.LabelStyle.Render("Ghi chú:  "), ui.EventDescStyle.Render(descVal))
	}

	fmt.Println(ui.CardBoxSuccess.Render(cardContent))
	return nil
}
