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

	"lich-cli/internal/api"
	"lich-cli/internal/config"
	"lich-cli/internal/ui"
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
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	switch action {
	case "connect", "login":
		return runGoogleConnect(ctx, client, subArgs)
	case "status":
		return runGoogleStatus(ctx, client, subArgs)
	case "calendars", "list":
		return runGoogleCalendars(ctx, client, subArgs)
	case "map":
		return runGoogleMap(ctx, client, subArgs)
	case "create-calendar", "create-cal", "new-calendar":
		return runGoogleCreateCalendar(ctx, client, subArgs)
	case "sync":
		return runGoogleSync(ctx, client, subArgs)
	case "disconnect", "logout":
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
  lich google connect         Mở trình duyệt để xác thực và liên kết tài khoản Google
  lich google status          Kiểm tra trạng thái kết nối và các lịch đã ánh xạ
  lich google calendars       Xem danh sách lịch Google Calendar có thể map
  lich google map <cal> <ext> Ánh xạ lịch Lich với lịch Google
  lich google sync [flags]    Đồng bộ hóa 2 chiều hoặc 1 chiều với Google Calendar
  lich google disconnect      Hủy liên kết và thu hồi quyền truy cập Google

Tùy chọn cho 'sync':
  --direction, -d <push|pull|both>  Hướng đồng bộ (mặc định: both)
  --calendar <id>                   Chỉ định ID lịch cần đồng bộ
  --simple, -s                      Xuất văn bản ASCII đơn giản
`)
		return
	}

	banner := ui.TitleBanner.Render(" TÍCH HỢP GOOGLE CALENDAR (PHASE 3) ")
	fmt.Println(banner)

	helpContent := fmt.Sprintf(`%s
  %s    %s
  %s     %s
  %s  %s
  %s        %s
  %s       %s
  %s %s

%s
  %s      %s
  %s           %s`,
		ui.CardTitle.Render("CÁC LỆNH GOOGLE:"),
		ui.ValueStyle.Render("lich google connect"), ui.LabelStyle.Render("Mở trình duyệt để xác thực OAuth với Google Calendar"),
		ui.ValueStyle.Render("lich google status"), ui.LabelStyle.Render("Kiểm tra tài khoản, trạng thái sync và ánh xạ lịch"),
		ui.ValueStyle.Render("lich google calendars"), ui.LabelStyle.Render("Danh sách các lịch Google Calendar của bạn"),
		ui.ValueStyle.Render("lich google map"), ui.LabelStyle.Render("Ánh xạ lịch Lich -> Google: lich google map <cal_id> <ext_cal_id>"),
		ui.ValueStyle.Render("lich google sync"), ui.LabelStyle.Render("Đồng bộ tức thì với Google: --direction push|pull|both"),
		ui.ValueStyle.Render("lich google disconnect"), ui.LabelStyle.Render("Hủy liên kết và ngắt kết nối Google Calendar"),
		ui.CardTitle.Render("TÙY CHỌN CHUNG:"),
		ui.ValueStyle.Render("--simple, -s"), ui.LabelStyle.Render("Xuất văn bản ASCII đơn giản"),
		ui.ValueStyle.Render("--direction, -d"), ui.LabelStyle.Render("Hướng đồng bộ ('push', 'pull', 'both')"),
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
		fmt.Printf("Trạng thái Google: %s\n", status.Status)
		fmt.Printf("Đã kết nối:        %v\n", status.Connected)
		fmt.Printf("Số lịch đã map:    %d\n", len(status.MappedCalendars))
		for _, m := range status.MappedCalendars {
			fmt.Printf("  - %s -> %s (%s)\n", m.CalendarName, m.ExternalCalendarID, m.SyncDirection)
		}
		return nil
	}

	statusBadge := ui.BadgeFailed
	if status.Connected {
		statusBadge = ui.BadgeSynced
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

func runGoogleSync(ctx context.Context, client *api.Client, args []string) error {
	fs := flag.NewFlagSet("google sync", flag.ContinueOnError)
	directionFlag := fs.String("direction", "both", "Hướng đồng bộ (push, pull, both)")
	fs.StringVar(directionFlag, "d", "both", "Hướng đồng bộ (viết tắt)")
	calFlag := fs.String("calendar", "", "ID lịch cụ thể")
	simpleFlag := fs.Bool("simple", false, "Hiển thị ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị ASCII đơn giản (viết tắt)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Printf("↻ Đang đồng bộ với Google Calendar (%s)...\n", *directionFlag)
	}

	res, err := client.SyncGoogle(ctx, *calFlag, *directionFlag)
	if err != nil {
		return fmt.Errorf("lỗi đồng bộ Google Calendar: %w", err)
	}

	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Println("✓ Hoàn tất đồng bộ Google Calendar:")
		fmt.Printf("  - Đẩy lên Google: %d sự kiện\n", res.Pushed)
		fmt.Printf("  - Nhận từ Google: %d sự kiện\n", res.Pulled)
		return nil
	}

	fmt.Println(ui.CardBoxSuccess.Render(fmt.Sprintf(
		"%s\n\n%s %s\n%s %s\n%s %s\n%s %s",
		ui.CardTitle.Render("✓ HOÀN TẤT ĐỒNG BỘ GOOGLE CALENDAR"),
		ui.LabelStyle.Render("Chế độ:         "), ui.ValueStyle.Render(strings.ToUpper(*directionFlag)),
		ui.LabelStyle.Render("Đẩy lên Google: "), ui.ValueStyle.Render(fmt.Sprintf("%d sự kiện", res.Pushed)),
		ui.LabelStyle.Render("Nhận từ Google: "), ui.ValueStyle.Render(fmt.Sprintf("%d sự kiện", res.Pulled)),
		ui.LabelStyle.Render("Trạng thái:     "), ui.BadgeSynced,
	)))
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

