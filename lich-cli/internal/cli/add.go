package cli

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"strings"
	"time"

	"lich-cli/internal/api"
	"lich-cli/internal/cache"
	"lich-cli/internal/config"
	"lich-cli/internal/syncer"
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
		Description: *descFlag,
		StartAt:     startRFC,
		EndAt:       endRFC,
		Timezone:    tz,
		Location:    *locationFlag,
		SyncState:   cache.SyncStatePendingCreate,
	}

	if err := cache.UpsertEvent(db, localEvent); err != nil {
		return fmt.Errorf("lỗi lưu sự kiện cục bộ: %w", err)
	}

	// 3. Đưa vào hàng đợi sync_jobs
	_, err = cache.EnqueueSyncJob(db, "event", eventID, cache.SyncOpCreate, syncer.MarshalPayload(localEvent))
	if err != nil {
		return fmt.Errorf("lỗi tạo sync job: %w", err)
	}

	// 4. Kích hoạt đồng bộ hóa ngầm nếu đã đăng nhập
	cfg, err := config.LoadConfig()
	if err == nil && cfg.Token != "" {
		client := api.NewClient(cfg.ServerURL, cfg.Token)
		engine := syncer.NewSyncEngine(db, client)
		engine.SyncInBackground()
	}

	// 5. Trả lời ngay lập tức mà không chờ mạng
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
		fmt.Printf("  ℹ Lưu ý: Sự kiện kéo dài qua đêm, kết thúc lúc %s ngày %s.\n",
			endTime.Format("15:04"),
			endTime.Format("Monday, 02/01/2006"),
		)
	}

	if *locationFlag != "" {
		fmt.Printf("  Địa điểm:  %s\n", *locationFlag)
	}
	if *descFlag != "" {
		fmt.Printf("  Ghi chú:   %s\n", *descFlag)
	}
	fmt.Println("  Đồng bộ:   ↻ Đang đồng bộ ngầm (Sync: pending)")

	return nil
}
