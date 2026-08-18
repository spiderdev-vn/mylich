package cli

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"lich-cli/internal/api"
	"lich-cli/internal/cache"
	"lich-cli/internal/config"
	"lich-cli/internal/syncer"
	"lich-cli/internal/ui"
)

func generateEventID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func RunAdd(args []string) error {
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
	dateFlag := fs.String("date", "", "Ngày diễn ra (YYYY-MM-DD, today, tomorrow)")
	atFlag := fs.String("at", "", "Giờ bắt đầu (ví dụ: 10:00, 23:30, 11:30pm)")
	toFlag := fs.String("to", "", "Giờ kết thúc (ví dụ: 22:33, 3am, 03:00)")
	durationFlag := fs.String("duration", "1h", "Thời lượng sự kiện (ví dụ: 30m, 1h, 2h30m)")
	calendarFlag := fs.String("calendar", "", "ID lịch đích")
	descFlag := fs.String("desc", "", "Ghi chú mô tả")
	locationFlag := fs.String("location", "", "Địa điểm diễn ra")
	timezoneFlag := fs.String("timezone", "", "Múi giờ sự kiện (ví dụ: Asia/Ho_Chi_Minh)")
	simpleFlag := fs.Bool("simple", false, "Hiển thị dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị dạng văn bản ASCII đơn giản (viết tắt)")

	fs.Usage = func() {
		fmt.Println("Sử dụng: lich add [tiêu đề] [flags]")
		fmt.Println()
		fmt.Println("Mô tả:")
		fmt.Println("  Tạo sự kiện mới. Nếu không truyền tham số, sẽ mở Form tương tác Huh để điền thông tin.")
		fmt.Println()
		fmt.Println("Tùy chọn:")
		fmt.Println("  --date <date>         Ngày sự kiện (YYYY-MM-DD, today, tomorrow)")
		fmt.Println("  --at <time>           Giờ bắt đầu (10:00, 23:30, 11:30pm, 10am)")
		fmt.Println("  --to <time>           Giờ kết thúc (22:33, 03:00, 3am)")
		fmt.Println("  --duration <duration> Thời lượng sự kiện (30m, 1h, 2h30m, mặc định: 1h)")
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

	title := strings.Join(positionalArgs, " ")
	dateVal := *dateFlag
	atVal := *atFlag
	toVal := *toFlag
	locVal := *locationFlag
	descVal := *descFlag

	// Nếu không truyền tiêu đề qua dòng lệnh và không ở chế độ simple -> Mở Form tương tác Huh
	if strings.TrimSpace(title) == "" && !ui.IsSimpleMode(*simpleFlag) {
		if dateVal == "" {
			dateVal = "today"
		}
		if atVal == "" {
			atVal = "10:00"
		}
		if toVal == "" {
			toVal = "11:00"
		}

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Tiêu đề sự kiện").
					Placeholder("Họp nhóm Sprint / Đi chơi").
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("tiêu đề không được để trống")
						}
						return nil
					}).
					Value(&title),

				huh.NewInput().
					Title("Ngày diễn ra").
					Placeholder("today / tomorrow / 2026-08-20").
					Value(&dateVal),

				huh.NewInput().
					Title("Giờ bắt đầu (--at)").
					Placeholder("10:00 / 11:30pm / 9am").
					Value(&atVal),

				huh.NewInput().
					Title("Giờ kết thúc (--to)").
					Placeholder("11:30 / 3am / 22:33").
					Value(&toVal),

				huh.NewInput().
					Title("Địa điểm (Tùy chọn)").
					Placeholder("Phòng họp 101 / Quán cafe").
					Value(&locVal),

				huh.NewInput().
					Title("Ghi chú (Tùy chọn)").
					Placeholder("Nội dung chi tiết...").
					Value(&descVal),
			),
		)

		if err := form.Run(); err != nil {
			return err
		}
	}

	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("tiêu đề sự kiện là bắt buộc: lich add \"<tiêu đề>\" [tùy chọn]")
	}

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
		dateVal,
		atVal,
		toVal,
		*durationFlag,
		hasDurationFlag,
		loc,
	)
	if err != nil {
		return err
	}

	// 2. Tạo sự kiện cục bộ (Local-First)
	eventID := generateEventID()
	startRFC := startTime.UTC().Format(time.RFC3339)
	endRFC := endTime.UTC().Format(time.RFC3339)

	tz := *timezoneFlag
	if tz == "" || tz == "Local" {
		tz = "UTC"
	}

	localEvent := cache.LocalEvent{
		ID:          eventID,
		CalendarID:  *calendarFlag,
		Title:       title,
		Description: descVal,
		StartAt:     startRFC,
		EndAt:       endRFC,
		Timezone:    tz,
		Location:    locVal,
		SyncState:   cache.SyncStatePendingCreate,
	}

	if err := cache.UpsertEvent(db, localEvent); err != nil {
		return fmt.Errorf("lỗi lưu sự kiện cục bộ: %w", err)
	}

	// 3. Đưa vào hàng đợi sync_jobs
	_, _ = cache.EnqueueSyncJob(db, "event", eventID, cache.SyncOpCreate, syncer.MarshalPayload(localEvent))

	// 4. Kích hoạt đồng bộ hóa ngầm nếu đã đăng nhập
	cfg, err := config.LoadConfig()
	if err == nil && cfg.Token != "" {
		client := api.NewClient(cfg.ServerURL, cfg.Token)
		engine := syncer.NewSyncEngine(db, client)
		engine.SyncInBackground()
	}

	// 5. Trả lời ngay lập tức
	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Println("✓ Đã tạo sự kiện")
		fmt.Printf("  ID:        %s\n", eventID)
		fmt.Printf("  Tiêu đề:   %s\n", title)
		if startTime.Format("2006-01-02") == endTime.Format("2006-01-02") {
			fmt.Printf("  Thời gian: %s (%s)\n", formatTimeRange(startRFC, endRFC, loc), startTime.Format("Mon, 02 Jan 2006"))
		} else {
			fmt.Printf("  Thời gian: %s %s -> %s %s\n",
				startTime.Format("15:04"), startTime.Format("02/01/2006"),
				endTime.Format("15:04"), endTime.Format("02/01/2006"),
			)
		}
		if isOvernight {
			fmt.Printf("  Lưu ý:     Sự kiện kéo dài qua đêm, kết thúc lúc %s ngày %s.\n",
				endTime.Format("15:04"),
				endTime.Format("Monday, 02/01/2006"),
			)
		}
		if locVal != "" {
			fmt.Printf("  Địa điểm:  %s\n", locVal)
		}
		if descVal != "" {
			fmt.Printf("  Ghi chú:   %s\n", descVal)
		}
		fmt.Println("  Đồng bộ:   [PENDING] Đang đồng bộ ngầm")
		return nil
	}

	// Lip Gloss Success Card
	timeStr := formatTimeRange(startRFC, endRFC, loc)
	if startTime.Format("2006-01-02") != endTime.Format("2006-01-02") {
		timeStr = fmt.Sprintf("%s (%s) -> %s (%s)",
			startTime.Format("15:04"), startTime.Format("02/01"),
			endTime.Format("15:04"), endTime.Format("02/01/2006"),
		)
	}

	cardContent := fmt.Sprintf(
		"%s\n\n%s %s\n%s %s\n%s %s (%s)\n%s %s",
		ui.CardTitle.Render("✓ ĐÃ TẠO SỰ KIỆN THÀNH CÔNG"),
		ui.LabelStyle.Render("Tiêu đề:  "), ui.ValueStyle.Render(title),
		ui.LabelStyle.Render("ID:       "), ui.LabelStyle.Render(eventID),
		ui.LabelStyle.Render("Thời gian:"), ui.TimePill.Render(timeStr), startTime.Format("Monday, 02/01/2006"),
		ui.LabelStyle.Render("Đồng bộ:  "), ui.BadgePending,
	)

	if isOvernight {
		cardContent += fmt.Sprintf("\n%s %s",
			ui.LabelStyle.Render("Chú ý:    "),
			ui.LabelStyle.Render(fmt.Sprintf("Kéo dài qua đêm, kết thúc %s ngày %s", endTime.Format("15:04"), endTime.Format("Monday, 02/01"))),
		)
	}
	if locVal != "" {
		cardContent += fmt.Sprintf("\n%s %s", ui.LabelStyle.Render("Địa điểm: "), ui.ValueStyle.Render(locVal))
	}
	if descVal != "" {
		cardContent += fmt.Sprintf("\n%s %s", ui.LabelStyle.Render("Ghi chú:  "), ui.EventDescStyle.Render(descVal))
	}

	fmt.Println(ui.CardBoxSuccess.Render(cardContent))
	return nil
}
