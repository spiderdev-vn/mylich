package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spiderdev-vn/mylich/lich-cli/internal/api"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/cache"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/config"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/syncer"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/ui"
)

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func RunGoogle(args []string) error {
	if len(args) == 0 {
		printGoogleHelp(false)
		return nil
	}

	action := strings.ToLower(args[0])
	subArgs := args[1:]

	if action == "help" || action == "-h" || action == "--help" {
		printGoogleHelp(false)
		return nil
	}

	cfg, err := config.LoadConfig()
	if err != nil || cfg.Token == "" {
		return fmt.Errorf("chưa đăng nhập. Vui lòng chạy 'lich login' trước")
	}

	client := api.NewClient(cfg.ServerURL, cfg.Token)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	switch action {
	case "connect", "conn", "login", "c":
		return runGoogleConnect(ctx, client, subArgs)
	case "status", "stat", "st", "info":
		return runGoogleStatus(ctx, client, subArgs)
	case "calendars", "calendar", "list", "cals", "cal", "ls", "l":
		return runGoogleCalendars(ctx, client, subArgs)
	case "map", "m":
		return runGoogleMap(ctx, client, subArgs)
	case "create-calendar", "create-cal", "new-calendar", "create", "new":
		return runGoogleCreateCalendar(ctx, client, subArgs)
	case "sync", "sy", "s":
		return runGoogleSync(ctx, client, subArgs)
	case "push":
		return runGoogleSync(ctx, client, append([]string{"push"}, subArgs...))
	case "pull":
		return runGoogleSync(ctx, client, append([]string{"pull"}, subArgs...))
	case "disconnect", "disc", "logout", "dc":
		return runGoogleDisconnect(ctx, client, subArgs)
	default:
		return fmt.Errorf("hành động không hợp lệ '%s'. Gõ 'lich google help' để xem hướng dẫn", action)
	}
}

func printGoogleHelp(simple bool) {
	if ui.IsSimpleMode(simple) {
		fmt.Print(`Lich Google Calendar Integration
================================
Sử dụng:
  lich google connect                     Mở trình duyệt để xác thực và liên kết tài khoản Google
  lich google status                      Kiểm tra trạng thái kết nối và các lịch đã ánh xạ
  lich google calendars                   Xem danh sách lịch Google Calendar có thể map
  lich google map <cal> <ext>             Ánh xạ lịch Lich với lịch Google
  lich google sync [push|pull|both] [opt] Đồng bộ hóa với Google Calendar (2 chiều hoặc 1 chiều)
  lich google disconnect                  Hủy liên kết và thu hồi quyền truy cập Google

Tùy chọn cho 'sync':
  lich google sync [push|pull|both]       Hướng đồng bộ (mặc định: both)
  lich google sync <event_id>             Đồng bộ nhanh 1 sự kiện theo ID
  lich google sync today|tomorrow         Đồng bộ các sự kiện của Hôm nay / Ngày mai
  lich google sync week|month             Đồng bộ các sự kiện trong Tuần / Tháng
  --event, -e <id>                        ID sự kiện cụ thể
  --date <YYYY-MM-DD|keyword>             Ngày cụ thể hoặc từ khóa
  --from <date> --to <date>               Khoảng thời gian bắt đầu và kết thúc
  --direction, -d <push|pull|both>        Hướng đồng bộ (mặc định: both)
  --calendar <id>                         Chỉ định ID lịch cần đồng bộ
  --simple, -s                            Xuất văn bản ASCII đơn giản
`)
		return
	}

	banner := ui.TitleBanner.Render(" TÍCH HỢP GOOGLE CALENDAR ")
	fmt.Println(banner)

	helpContent := fmt.Sprintf(`%s
  %s    %s
  %s     %s
  %s  %s
  %s        %s
  %s       %s
  %s %s

%s
  %s  %s
  %s    %s
  %s  %s
  %s    %s
  %s      %s
  %s           %s`,
		ui.CardTitle.Render("CÁC LỆNH GOOGLE:"),
		ui.ValueStyle.Render("lich google connect"), ui.LabelStyle.Render("Mở trình duyệt để xác thực OAuth với Google Calendar"),
		ui.ValueStyle.Render("lich google status"), ui.LabelStyle.Render("Kiểm tra tài khoản, trạng thái sync và ánh xạ lịch"),
		ui.ValueStyle.Render("lich google calendars"), ui.LabelStyle.Render("Danh sách các lịch Google Calendar của bạn"),
		ui.ValueStyle.Render("lich google map"), ui.LabelStyle.Render("Ánh xạ lịch: lich google map <cal_id> <ext_cal_id>"),
		ui.ValueStyle.Render("lich google sync"), ui.LabelStyle.Render("Đồng bộ dữ liệu: lich google sync [push|pull|both] [flags]"),
		ui.ValueStyle.Render("lich google disconnect"), ui.LabelStyle.Render("Hủy liên kết và ngắt kết nối Google Calendar"),
		ui.CardTitle.Render("TÙY CHỌN ĐỒNG BỘ (SYNC):"),
		ui.ValueStyle.Render("[push|pull|both]"), ui.LabelStyle.Render("Hướng đồng bộ: 2 chiều hoặc 1 chiều (mặc định: both)"),
		ui.ValueStyle.Render("<id> / --event, -e"), ui.LabelStyle.Render("Đồng bộ nhanh 1 sự kiện theo Event ID (<300ms)"),
		ui.ValueStyle.Render("today | tomorrow"), ui.LabelStyle.Render("Đồng bộ sự kiện Hôm nay / Ngày mai (--today / --tomorrow)"),
		ui.ValueStyle.Render("week | month"), ui.LabelStyle.Render("Đồng bộ sự kiện Tuần này / Tháng này (--week / --month)"),
		ui.ValueStyle.Render("--from / --to"), ui.LabelStyle.Render("Khoảng thời gian tùy chỉnh (VD: --from 2026-08-19 --to 2026-08-25)"),
		ui.ValueStyle.Render("--simple, -s"), ui.LabelStyle.Render("Xuất văn bản ASCII đơn giản"),
	)
	fmt.Println(ui.CardBox.Width(78).Render(helpContent))
}

func runGoogleConnect(ctx context.Context, client *api.Client, args []string) error {
	fs := flag.NewFlagSet("google connect", flag.ContinueOnError)
	simpleFlag := fs.Bool("simple", false, "Hiển thị ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị ASCII đơn giản (viết tắt)")
	noBrowser := fs.Bool("no-browser", false, "Không tự động mở trình duyệt")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	authURL, err := client.GetGoogleAuthURL(ctx)
	if err != nil {
		return fmt.Errorf("lỗi lấy URL xác thực từ máy chủ: %w", err)
	}

	if !*noBrowser {
		_ = openBrowser(authURL)
	}

	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Println("Vui lòng hoàn tất xác thực Google trên trình duyệt:")
		fmt.Println(authURL)
	} else {
		card := ui.CardBoxSuccess.Render(fmt.Sprintf(
			"%s\n\n%s\n\n%s\n%s\n\n%s",
			ui.CardTitle.Render("🔗 KẾT NỐI GOOGLE CALENDAR"),
			ui.ValueStyle.Render("Đang mở trình duyệt web để bạn đăng nhập Google và cấp quyền lịch..."),
			ui.LabelStyle.Render("Nếu trình duyệt không tự mở, hãy truy cập link sau:"),
			ui.ValueStyle.Render(authURL),
			ui.LabelStyle.Render("Sau khi cấp quyền thành công, gõ 'lich google status' để kiểm tra."),
		))
		fmt.Println(card)
	}

	return nil
}

func runGoogleStatus(ctx context.Context, client *api.Client, args []string) error {
	fs := flag.NewFlagSet("google status", flag.ContinueOnError)
	simpleFlag := fs.Bool("simple", false, "Hiển thị ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị ASCII đơn giản (viết tắt)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	status, err := client.GetGoogleStatus(ctx)
	if err != nil {
		return fmt.Errorf("lỗi kiểm tra trạng thái Google: %w", err)
	}

	if ui.IsSimpleMode(*simpleFlag) {
		statusStr := "Chưa kết nối (Disconnected)"
		if status.Connected {
			statusStr = "Đã kết nối (Connected)"
		}
		fmt.Printf("Trạng thái Google: %s\n", statusStr)
		fmt.Printf("Provider:          %s\n", status.Provider)
		fmt.Printf("Số lịch đã map:    %d\n", len(status.MappedCalendars))
		for _, m := range status.MappedCalendars {
			fmt.Printf("  - %s -> %s (%s)\n", m.CalendarName, m.ExternalCalendarID, m.SyncDirection)
		}
		return nil
	}

	statusBadge := ui.BadgeDisconnected
	if status.Connected {
		statusBadge = ui.BadgeConnected
	}

	content := fmt.Sprintf(
		"%s\n\n%s %s\n%s %s\n%s %d lịch",
		ui.CardTitle.Render("GOOGLE CALENDAR STATUS"),
		ui.LabelStyle.Render("Trạng thái: "), statusBadge,
		ui.LabelStyle.Render("Provider:   "), ui.ValueStyle.Render(status.Provider),
		ui.LabelStyle.Render("Lịch ánh xạ:"), len(status.MappedCalendars),
	)

	if len(status.MappedCalendars) > 0 {
		content += "\n\n" + ui.CardTitle.Render("DANH SÁCH ÁNH XẠ:")
		for _, m := range status.MappedCalendars {
			content += fmt.Sprintf("\n  • %s %s %s %s",
				ui.ValueStyle.Render(m.CalendarName),
				ui.LabelStyle.Render("➔"),
				ui.ValueStyle.Render(m.ExternalCalendarID),
				ui.TimePill.Render(strings.ToUpper(m.SyncDirection)),
			)
		}
	} else if !status.Connected {
		content += "\n\n" + ui.LabelStyle.Render("Gợi ý: Gõ 'lich google connect' để liên kết tài khoản Google Calendar.")
	}

	fmt.Println(ui.CardBox.Render(content))
	return nil
}

func runGoogleCalendars(ctx context.Context, client *api.Client, args []string) error {
	fs := flag.NewFlagSet("google calendars", flag.ContinueOnError)
	simpleFlag := fs.Bool("simple", false, "Hiển thị ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị ASCII đơn giản (viết tắt)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	cals, err := client.ListGoogleCalendars(ctx)
	if err != nil {
		return fmt.Errorf("lỗi lấy danh sách lịch Google: %w", err)
	}

	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Println("Google Calendars:")
		for _, c := range cals {
			primaryTag := ""
			if c.IsPrimary {
				primaryTag = " [Primary]"
			}
			fmt.Printf("  • %s (%s)%s\n", c.Name, c.ID, primaryTag)
		}
		return nil
	}

	content := fmt.Sprintf("%s\n", ui.CardTitle.Render("DANH SÁCH LỊCH GOOGLE CALENDAR:"))
	for _, c := range cals {
		primaryBadge := ""
		if c.IsPrimary {
			primaryBadge = " " + ui.BadgeSynced
		}
		content += fmt.Sprintf("\n  • %s %s%s\n    %s",
			ui.ValueStyle.Render(c.Name),
			ui.LabelStyle.Render(fmt.Sprintf("(%s)", c.ID)),
			primaryBadge,
			ui.EventDescStyle.Render(fmt.Sprintf("Múi giờ: %s | Quyền: %s", c.TimeZone, c.AccessRole)),
		)
	}

	fmt.Println(ui.CardBox.Render(content))
	return nil
}

func runGoogleMap(ctx context.Context, client *api.Client, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("cần truyền đủ tham số: lich google map <lich_calendar_id> <google_calendar_id> [sync_direction]")
	}

	calID := args[0]
	extCalID := args[1]
	direction := "bidirectional"
	if len(args) >= 3 {
		direction = args[2]
	}

	err := client.MapGoogleCalendar(ctx, calID, extCalID, direction)
	if err != nil {
		return fmt.Errorf("lỗi thiết lập ánh xạ lịch: %w", err)
	}

	fmt.Println(ui.CardBoxSuccess.Render(fmt.Sprintf(
		"%s\n\n%s %s\n%s %s\n%s %s",
		ui.CardTitle.Render("✓ ĐÃ ÁNH XẠ LỊCH THÀNH CÔNG"),
		ui.LabelStyle.Render("Lich Calendar:  "), ui.ValueStyle.Render(calID),
		ui.LabelStyle.Render("Google Calendar:"), ui.ValueStyle.Render(extCalID),
		ui.LabelStyle.Render("Hướng đồng bộ:  "), ui.TimePill.Render(strings.ToUpper(direction)),
	)))
	return nil
}

func resolveTimeRange(rangeKeyword, dateStr, fromStr, toStr string, now time.Time, loc *time.Location) (string, string, string) {
	kw := strings.ToLower(strings.TrimSpace(rangeKeyword))
	if kw == "" {
		kw = strings.ToLower(strings.TrimSpace(dateStr))
	}

	switch kw {
	case "today", "hom-nay", "hn":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		end := start.Add(24*time.Hour - time.Nanosecond)
		return start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339), "Hôm nay (Today)"
	case "tomorrow", "ngay-mai", "mai":
		start := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
		end := start.Add(24*time.Hour - time.Nanosecond)
		return start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339), "Ngày mai (Tomorrow)"
	case "yesterday", "hom-qua", "qua":
		start := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, loc)
		end := start.Add(24*time.Hour - time.Nanosecond)
		return start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339), "Hôm qua (Yesterday)"
	case "week", "this-week", "tuan":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day()-(weekday-1), 0, 0, 0, 0, loc)
		end := start.Add(7*24*time.Hour - time.Nanosecond)
		return start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339), "Tuần này (This Week)"
	case "month", "this-month", "thang":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
		return start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339), "Tháng này (This Month)"
	}

	var fromRFC, toRFC string
	var label string
	if fromStr != "" {
		if t, err := parseFlexibleDate(fromStr, loc); err == nil {
			fromRFC = t.UTC().Format(time.RFC3339)
			label = fmt.Sprintf("Từ %s", fromStr)
		}
	}
	if toStr != "" {
		if t, err := parseFlexibleDate(toStr, loc); err == nil {
			if t.Hour() == 0 && t.Minute() == 0 {
				t = t.Add(24*time.Hour - time.Nanosecond)
			}
			toRFC = t.UTC().Format(time.RFC3339)
			if label != "" {
				label += fmt.Sprintf(" đến %s", toStr)
			} else {
				label = fmt.Sprintf("Đến %s", toStr)
			}
		}
	}

	if fromRFC == "" && toRFC == "" && kw != "" {
		// Thử parse dạng ngày cụ thể như 2026-08-20
		if t, err := parseFlexibleDate(kw, loc); err == nil {
			start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
			end := start.Add(24*time.Hour - time.Nanosecond)
			return start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339), fmt.Sprintf("Ngày %s", kw)
		}
	}

	return fromRFC, toRFC, label
}

func runGoogleSync(ctx context.Context, client *api.Client, args []string) error {
	fs := flag.NewFlagSet("google sync", flag.ContinueOnError)
	directionFlag := fs.String("direction", "", "Hướng đồng bộ: push, pull, hoặc both (mặc định: both)")
	fs.StringVar(directionFlag, "d", "", "Hướng đồng bộ (viết tắt)")
	eventFlag := fs.String("event", "", "ID sự kiện cụ thể để đồng bộ tức thì")
	fs.StringVar(eventFlag, "e", "", "ID sự kiện cụ thể (viết tắt)")
	calFlag := fs.String("calendar", "", "ID lịch cụ thể")
	dateFlag := fs.String("date", "", "Khoảng thời gian (today, tomorrow, week, month, YYYY-MM-DD)")
	fromFlag := fs.String("from", "", "Thời gian bắt đầu (VD: 2026-08-19 hoặc 2026-08-19T00:00:00)")
	toFlag := fs.String("to", "", "Thời gian kết thúc (VD: 2026-08-25)")
	todayFlag := fs.Bool("today", false, "Chỉ đồng bộ các sự kiện của Hôm nay")
	tomorrowFlag := fs.Bool("tomorrow", false, "Chỉ đồng bộ các sự kiện của Ngày mai")
	weekFlag := fs.Bool("week", false, "Chỉ đồng bộ các sự kiện trong Tuần này")
	monthFlag := fs.Bool("month", false, "Chỉ đồng bộ các sự kiện trong Tháng này")

	verboseFlag := fs.Bool("verbose", false, "Hiển thị chi tiết các bước đồng bộ")
	fs.BoolVar(verboseFlag, "v", false, "Hiển thị chi tiết các bước đồng bộ (viết tắt)")
	fs.BoolVar(verboseFlag, "w", false, "Hiển thị chi tiết các bước đồng bộ (viết tắt)")
	simpleFlag := fs.Bool("simple", false, "Hiển thị ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị ASCII đơn giản (viết tắt)")

	fs.Usage = func() {
		if ui.IsSimpleMode(*simpleFlag) {
			fmt.Println("Sử dụng: lich google sync [push|pull|both] [flags]")
			fmt.Println()
			fmt.Println("Mô tả:")
			fmt.Println("  Đồng bộ hóa dữ liệu với Google Calendar (Source of Truth: Lich).")
			fmt.Println("  - 'lich google sync' hoặc 'lich google sync both': Đồng bộ 2 chiều (Push & Pull).")
			fmt.Println("  - 'lich google sync push':                         Chỉ đẩy sự kiện lên Google Calendar.")
			fmt.Println("  - 'lich google sync pull':                         Chỉ kéo sự kiện từ Google Calendar về.")
			fmt.Println()
			fmt.Println("Tùy chọn:")
			fmt.Println("  --event, -e <id>        Đồng bộ nhanh 1 sự kiện theo ID (<300ms)")
			fmt.Println("  --today / --tomorrow    Đồng bộ sự kiện Hôm nay / Ngày mai")
			fmt.Println("  --week / --month        Đồng bộ sự kiện Tuần này / Tháng này")
			fmt.Println("  --from <date>           Thời gian bắt đầu (VD: 2026-08-19)")
			fmt.Println("  --to <date>             Thời gian kết thúc (VD: 2026-08-25)")
			fmt.Println("  --direction, -d <dir>   Hướng đồng bộ: 'push', 'pull', hoặc 'both'")
			fmt.Println("  --calendar <id>         Lọc theo ID lịch cụ thể")
			fmt.Println("  --simple, -s            Hiển thị dạng văn bản ASCII đơn giản")
			fmt.Println("  --verbose, -v           Hiển thị chi tiết các bước đồng bộ")
			return
		}

		banner := ui.TitleBanner.Render(" ĐỒNG BỘ GOOGLE CALENDAR (LICH GOOGLE SYNC) ")
		fmt.Println(banner)

		helpContent := fmt.Sprintf(`%s
  %s    %s
  %s   %s
  %s   %s

%s
  %s  %s
  %s    %s
  %s  %s
  %s    %s
  %s      %s
  %s   %s
  %s   %s
  %s           %s`,
			ui.CardTitle.Render("CÚ PHÁP ĐỒNG BỘ:"),
			ui.ValueStyle.Render("lich google sync [push|pull|both]"), ui.LabelStyle.Render("Đồng bộ 2 chiều hoặc 1 chiều (mặc định: both)"),
			ui.ValueStyle.Render("lich google sync <short_id>"), ui.LabelStyle.Render("Đồng bộ nhanh 1 sự kiện theo ID (<300ms)"),
			ui.ValueStyle.Render("lich google sync today|week"), ui.LabelStyle.Render("Đồng bộ theo khoảng thời gian (today, tomorrow, week, month)"),
			ui.CardTitle.Render("TÙY CHỌN & CỜ:"),
			ui.ValueStyle.Render("--event, -e <id>"), ui.LabelStyle.Render("ID sự kiện cụ thể cần đồng bộ"),
			ui.ValueStyle.Render("--today / --tomorrow"), ui.LabelStyle.Render("Đồng bộ phạm vi Hôm nay / Ngày mai"),
			ui.ValueStyle.Render("--week / --month"), ui.LabelStyle.Render("Đồng bộ phạm vi Tuần này / Tháng này"),
			ui.ValueStyle.Render("--from <d> --to <d>"), ui.LabelStyle.Render("Khoảng ngày tùy chỉnh (VD: --from 2026-08-19 --to 2026-08-25)"),
			ui.ValueStyle.Render("--direction, -d <dir>"), ui.LabelStyle.Render("Hướng đồng bộ ('push', 'pull', 'both')"),
			ui.ValueStyle.Render("--calendar <id>"), ui.LabelStyle.Render("Lọc theo ID lịch cụ thể"),
			ui.ValueStyle.Render("--verbose, -v"), ui.LabelStyle.Render("Hiển thị chi tiết các bước đồng bộ"),
			ui.ValueStyle.Render("--simple, -s"), ui.LabelStyle.Render("Xuất văn bản ASCII đơn giản"),
		)
		fmt.Println(ui.CardBox.Width(78).Render(helpContent))
	}

	flagsTakingValue := map[string]bool{
		"-direction": true, "--direction": true, "-d": true,
		"-event": true, "--event": true, "-e": true,
		"-calendar": true, "--calendar": true,
		"-date": true, "--date": true,
		"-from": true, "--from": true,
		"-to": true, "--to": true,
	}

	var cleanArgs []string
	var positionalTokens []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			cleanArgs = append(cleanArgs, a)
			eqIdx := strings.Index(a, "=")
			flagName := a
			if eqIdx != -1 {
				flagName = a[:eqIdx]
			}
			if flagsTakingValue[flagName] && eqIdx == -1 && i+1 < len(args) {
				i++
				cleanArgs = append(cleanArgs, args[i])
			}
		} else {
			positionalTokens = append(positionalTokens, a)
		}
	}

	if err := fs.Parse(cleanArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	positionalTokens = append(positionalTokens, fs.Args()...)

	loc := time.Local
	now := time.Now().In(loc)

	direction := "both"
	rangeKeyword := *dateFlag
	if *todayFlag {
		rangeKeyword = "today"
	} else if *tomorrowFlag {
		rangeKeyword = "tomorrow"
	} else if *weekFlag {
		rangeKeyword = "week"
	} else if *monthFlag {
		rangeKeyword = "month"
	}

	// Xử lý positional args: "push", "pull", "both", "today", "tomorrow", "week", "month", hoặc <event-id> / <date>
	for _, p := range positionalTokens {
		lower := strings.ToLower(strings.TrimSpace(p))
		if lower == "push" || lower == "pull" || lower == "both" {
			direction = lower
		} else if lower == "today" || lower == "tomorrow" || lower == "week" || lower == "month" || lower == "yesterday" {
			rangeKeyword = lower
		} else if strings.Count(p, "-") == 2 || strings.Count(p, "/") == 2 {
			// Date like 2026-08-20 or 20/08/2026
			rangeKeyword = p
		} else if *eventFlag == "" {
			*eventFlag = p
		}
	}

	if *directionFlag != "" {
		d := strings.ToLower(strings.TrimSpace(*directionFlag))
		if d == "push" || d == "pull" || d == "both" {
			direction = d
		}
	}

	// Resolve short ID prefix và báo conflict nếu trùng
	if *eventFlag != "" {
		if cachePath, cErr := cache.GetCachePath(); cErr == nil {
			if db, dbErr := cache.OpenDatabase(cachePath); dbErr == nil {
				if ev, rErr := cache.ResolveEventByPrefix(db, *eventFlag); rErr == nil && ev != nil {
					*eventFlag = ev.ID
				} else if rErr != nil {
					db.Close()
					return rErr
				}
				db.Close()
			}
		}
	}

	fromRFC, toRFC, rangeLabel := resolveTimeRange(rangeKeyword, *dateFlag, *fromFlag, *toFlag, now, loc)

	var res *api.GoogleSyncResponse
	var err error

	stepTitles := []string{
		"Đẩy dữ liệu cục bộ",
		"Đồng bộ Google Calendar",
		"Cập nhật SQLite máy",
	}

	performSyncWithReporter := func(reporter *ui.TrackerReporter) error {
		// Step 1: Push pending local mutations from SQLite cache to server
		if reporter != nil {
			reporter.SetStepRunning(0, "")
			reporter.SetSubDetail("Kiểm tra hàng đợi thay đổi...")
		}

		var localPushed int
		if direction == "push" || direction == "both" {
			if cachePath, cErr := cache.GetCachePath(); cErr == nil {
				if db, dbErr := cache.OpenDatabase(cachePath); dbErr == nil {
					engine := syncer.NewSyncEngine(db, client)
					pCount, pErr := engine.Push(ctx)
					if pErr == nil {
						localPushed = pCount
					}
					db.Close()
				}
			}
		}

		if reporter != nil {
			if localPushed > 0 {
				reporter.SetStepDone(0, fmt.Sprintf("↑%d sự kiện", localPushed))
			} else {
				reporter.SetStepDone(0, "đã khớp")
			}
		}

		// Step 2: Server <-> Google Calendar sync
		if reporter != nil {
			googleDetail := strings.ToUpper(direction)
			if rangeLabel != "" {
				googleDetail += fmt.Sprintf(" (%s)", rangeLabel)
			}
			reporter.SetStepRunning(1, googleDetail)
			reporter.SetSubDetail("Đang so khớp sự kiện...")
		}

		var syncErr error
		res, syncErr = client.SyncGoogle(ctx, *calFlag, direction, *eventFlag, fromRFC, toRFC)
		if syncErr != nil {
			if reporter != nil {
				reporter.SetStepFailed(1, syncErr.Error())
			}
			return syncErr
		}

		if reporter != nil {
			reporter.SetStepDone(1, fmt.Sprintf("↑%d ↓%d", res.Pushed, res.Pulled))
		}

		// Step 3: Pull fresh events from server back into local SQLite cache
		if reporter != nil {
			reporter.SetStepRunning(2, "")
			reporter.SetSubDetail("Ghi vào bộ nhớ đệm...")
		}

		var localPulled int
		if direction == "pull" || direction == "both" {
			if cachePath, cErr := cache.GetCachePath(); cErr == nil {
				if db, dbErr := cache.OpenDatabase(cachePath); dbErr == nil {
					engine := syncer.NewSyncEngine(db, client)
					pCount, pErr := engine.Pull(ctx)
					if pErr == nil {
						localPulled = pCount
					}
					db.Close()
				}
			}
		}

		if reporter != nil {
			if localPulled > 0 {
				reporter.SetStepDone(2, fmt.Sprintf("↓%d sự kiện", localPulled))
			} else {
				reporter.SetStepDone(2, "đã khớp")
			}
		}

		return nil
	}

	trackerTitle := "ĐỒNG BỘ GOOGLE CALENDAR"
	if *eventFlag != "" {
		trackerTitle = fmt.Sprintf("ĐẨY SỰ KIỆN [%s] LÊN GOOGLE", *eventFlag)
	} else if rangeLabel != "" {
		trackerTitle = fmt.Sprintf("ĐỒNG BỘ GOOGLE CALENDAR [%s]", strings.ToUpper(rangeLabel))
	}

	if ui.IsSimpleMode(*simpleFlag) || *verboseFlag {
		if *verboseFlag {
			if ui.IsSimpleMode(*simpleFlag) {
				fmt.Println("[1/3] Đang đồng bộ thay đổi cục bộ với máy chủ...")
				if rangeLabel != "" {
					fmt.Printf("[2/3] Đang đồng bộ với Google Calendar phạm vi %s (hướng: %s)...\n", rangeLabel, direction)
				} else {
					fmt.Printf("[2/3] Đang đồng bộ với Google Calendar (hướng: %s, Last-Write-Wins)...\n", direction)
				}
			} else {
				fmt.Println(ui.LabelStyle.Render("↻ [1/3] Đang đồng bộ thay đổi cục bộ với máy chủ..."))
				if rangeLabel != "" {
					fmt.Println(ui.LabelStyle.Render(fmt.Sprintf("↻ [2/3] Đang đồng bộ với Google Calendar phạm vi %s (hướng: %s)...", rangeLabel, direction)))
				} else {
					fmt.Println(ui.LabelStyle.Render(fmt.Sprintf("↻ [2/3] Đang đồng bộ với Google Calendar (hướng: %s, Last-Write-Wins)...", direction)))
				}
			}
		}
		err = performSyncWithReporter(nil)
	} else {
		err = ui.RunWithTracker(trackerTitle, stepTitles, performSyncWithReporter)
	}

	if err != nil {
		if errors.Is(err, ui.ErrAborted) || err.Error() == "user aborted" {
			return nil
		}
		errMsg := err.Error()
		if strings.Contains(errMsg, "rateLimitExceeded") || strings.Contains(errMsg, "RATE_LIMIT_EXCEEDED") || strings.Contains(errMsg, "Quota exceeded") {
			return fmt.Errorf("Google Calendar API tạm thời bị giới hạn tần suất yêu cầu (Rate Limit: 600 req/phút). Vui lòng đợi khoảng 30 giây rồi thử lại")
		}
		return fmt.Errorf("lỗi đồng bộ Google Calendar: %w", err)
	}

	if res == nil {
		return nil
	}

	if *verboseFlag {
		if ui.IsSimpleMode(*simpleFlag) {
			fmt.Printf("[3/3] Hoàn tất: Đẩy lên %d sự kiện, Kéo về %d sự kiện\n", res.Pushed, res.Pulled)
		} else {
			fmt.Println(ui.LabelStyle.Render(fmt.Sprintf("✓ [3/3] Hoàn tất: Đẩy lên %d sự kiện, Kéo về %d sự kiện", res.Pushed, res.Pulled)))
		}
	}

	if ui.IsSimpleMode(*simpleFlag) {
		if *eventFlag != "" {
			fmt.Printf("✓ Đã đồng bộ sự kiện [%s] lên Google Calendar\n", *eventFlag)
		} else if rangeLabel != "" {
			fmt.Printf("✓ Đã đồng bộ Google Calendar (%s): %d đẩy lên, %d kéo về\n", rangeLabel, res.Pushed, res.Pulled)
		} else {
			fmt.Printf("✓ Đã đồng bộ Google Calendar: %d đẩy lên, %d kéo về\n", res.Pushed, res.Pulled)
		}
		return nil
	}

	if *eventFlag != "" {
		fmt.Println(ui.CardBoxSuccess.Render(fmt.Sprintf(
			"%s\n\n%s %s\n%s %s\n%s %s",
			ui.CardTitle.Render("✓ ĐỒNG BỘ SỰ KIỆN LÊN GOOGLE THÀNH CÔNG"),
			ui.LabelStyle.Render("ID Sự kiện:     "), ui.ValueStyle.Render(*eventFlag),
			ui.LabelStyle.Render("Kết quả:        "), ui.ValueStyle.Render("Đã cập nhật trên Google Calendar"),
			ui.LabelStyle.Render("Trạng thái:     "), ui.BadgeSynced,
		)))
		return nil
	}

	cardContent := fmt.Sprintf(
		"%s\n\n%s %s\n%s %s\n%s %s\n%s %s",
		ui.CardTitle.Render("✓ ĐỒNG BỘ GOOGLE CALENDAR THÀNH CÔNG"),
		ui.LabelStyle.Render("Chế độ:         "), ui.ValueStyle.Render(strings.ToUpper(direction)),
		ui.LabelStyle.Render("Đẩy lên Google: "), ui.ValueStyle.Render(fmt.Sprintf("%d sự kiện", res.Pushed)),
		ui.LabelStyle.Render("Kéo về Lich:    "), ui.ValueStyle.Render(fmt.Sprintf("%d sự kiện", res.Pulled)),
		ui.LabelStyle.Render("Trạng thái:     "), ui.BadgeSynced,
	)

	if rangeLabel != "" {
		cardContent = fmt.Sprintf(
			"%s\n\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s",
			ui.CardTitle.Render("✓ ĐỒNG BỘ GOOGLE CALENDAR THÀNH CÔNG"),
			ui.LabelStyle.Render("Phạm vi:        "), ui.TimePill.Render(rangeLabel),
			ui.LabelStyle.Render("Chế độ:         "), ui.ValueStyle.Render(strings.ToUpper(direction)),
			ui.LabelStyle.Render("Đẩy lên Google: "), ui.ValueStyle.Render(fmt.Sprintf("%d sự kiện", res.Pushed)),
			ui.LabelStyle.Render("Kéo về Lich:    "), ui.ValueStyle.Render(fmt.Sprintf("%d sự kiện", res.Pulled)),
			ui.LabelStyle.Render("Trạng thái:     "), ui.BadgeSynced,
		)
	}

	fmt.Println(ui.CardBoxSuccess.Render(cardContent))
	return nil
}

func runGoogleDisconnect(ctx context.Context, client *api.Client, args []string) error {
	err := client.DisconnectGoogle(ctx)
	if err != nil {
		return fmt.Errorf("lỗi hủy kết nối Google: %w", err)
	}

	fmt.Println(ui.CardBox.Render(fmt.Sprintf(
		"%s\n\n%s",
		ui.CardTitle.Render("✓ ĐÃ HỦY LIÊN KẾT GOOGLE CALENDAR"),
		ui.LabelStyle.Render("Đã thu hồi quyền truy cập và xóa credentials của Google trên máy chủ."),
	)))
	return nil
}

func runGoogleCreateCalendar(ctx context.Context, client *api.Client, args []string) error {
	fs := flag.NewFlagSet("google create-calendar", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Tên lịch hiển thị trên Google Calendar (mặc định lấy theo lịch Lich)")
	dirFlag := fs.String("direction", "bidirectional", "Hướng đồng bộ (bidirectional, push, pull)")
	simpleFlag := fs.Bool("simple", false, "Xuất dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Xuất dạng văn bản ASCII đơn giản (viết tắt)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	calID := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if calID == "" {
		return fmt.Errorf("vui lòng cung cấp ID lịch Lich: lich google create-calendar <lich_calendar_id> [flags]")
	}

	extCal, err := client.CreateGoogleCalendar(ctx, calID, *nameFlag, *dirFlag)
	if err != nil {
		return fmt.Errorf("lỗi tạo lịch trên Google Calendar: %w", err)
	}

	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Printf("✓ Đã tạo và liên kết lịch Google thành công: %s (%s)\n", extCal.Name, extCal.ID)
		return nil
	}

	fmt.Println(ui.CardBoxSuccess.Render(fmt.Sprintf(
		"%s\n\n%s %s\n%s %s\n%s %s (%s)\n%s %s",
		ui.CardTitle.Render("✓ ĐÃ TẠO VÀ LIÊN KẾT LỊCH GOOGLE THÀNH CÔNG"),
		ui.LabelStyle.Render("Lịch Lich ID:   "), ui.ValueStyle.Render(calID),
		ui.LabelStyle.Render("Tên Google Cal: "), ui.ValueStyle.Render(extCal.Name),
		ui.LabelStyle.Render("Google Cal ID:  "), ui.ValueStyle.Render(extCal.ID), *dirFlag,
		ui.LabelStyle.Render("Trạng thái:     "), ui.BadgeSynced,
	)))
	return nil
}

