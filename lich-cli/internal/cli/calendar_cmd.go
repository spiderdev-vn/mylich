package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"lich-cli/internal/api"
	"lich-cli/internal/config"
	"lich-cli/internal/ui"
)

func RunCalendar(args []string) error {
	if len(args) == 0 {
		return runCalendarList(nil)
	}

	action := strings.ToLower(args[0])
	subArgs := args[1:]

	switch action {
	case "list", "ls":
		return runCalendarList(subArgs)
	case "add", "create", "new":
		return runCalendarAdd(subArgs)
	case "delete", "remove", "rm":
		return runCalendarDelete(subArgs)
	case "help", "-h", "--help":
		printCalendarHelp(false)
		return nil
	default:
		// If first arg is not a recognized action, assume listing or help
		if strings.HasPrefix(action, "-") {
			return runCalendarList(args)
		}
		return fmt.Errorf("hành động không hợp lệ '%s'. Gõ 'lich calendar help' để xem trợ giúp", action)
	}
}

func printCalendarHelp(simple bool) {
	if ui.IsSimpleMode(simple) {
		fmt.Print(`Quản lý Lịch (Calendars)
========================
Sử dụng:
  lich calendar list                 Xem danh sách các lịch
  lich calendar add <name> [flags]   Tạo lịch mới (tùy chọn tự động tạo trên Google Calendar)
  lich calendar delete <id> [flags]  Xóa lịch

Tùy chọn cho 'add':
  --timezone <tz>    Múi giờ (ví dụ: Asia/Ho_Chi_Minh, UTC)
  --color <hex>      Mã màu hiển thị (ví dụ: #4285F4)
  --desc <text>      Mô tả lịch
  --sync-google      Tự động tạo lịch tương ứng trên Google Calendar và map 2 chiều
  --simple, -s       Xuất dạng văn bản ASCII đơn giản
`)
		return
	}

	helpCard := ui.CardBox.Render(fmt.Sprintf(
		"%s\n\n  %s  %s\n  %s  %s\n  %s  %s\n\n%s\n  %s %s\n  %s %s\n  %s %s",
		ui.CardTitle.Render("QUẢN LÝ LỊCH (CALENDARS):"),
		ui.ValueStyle.Render("lich calendar list"), ui.LabelStyle.Render("Liệt kê tất cả các lịch trong hệ thống"),
		ui.ValueStyle.Render("lich calendar add <tên>"), ui.LabelStyle.Render("Tạo lịch mới (hỗ trợ --sync-google tạo trên Google)"),
		ui.ValueStyle.Render("lich calendar delete <id>"), ui.LabelStyle.Render("Xóa lịch khỏi hệ thống"),
		ui.CardTitle.Render("TÙY CHỌN 'add':"),
		ui.ValueStyle.Render("--sync-google"), ui.LabelStyle.Render("Tự động tạo lịch mới trên Google Calendar và liên kết 2 chiều"),
		ui.ValueStyle.Render("--timezone <tz>"), ui.LabelStyle.Render("Chỉ định múi giờ cho lịch mới"),
		ui.ValueStyle.Render("--simple, -s"), ui.LabelStyle.Render("Xuất văn bản ASCII đơn giản"),
	))
	fmt.Println(helpCard)
}

func runCalendarList(args []string) error {
	fs := flag.NewFlagSet("calendar list", flag.ContinueOnError)
	simpleFlag := fs.Bool("simple", false, "Xuất dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Xuất dạng văn bản ASCII đơn giản (viết tắt)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil || cfg.Token == "" {
		return fmt.Errorf("chưa đăng nhập. Vui lòng chạy 'lich login' trước")
	}

	client := api.NewClient(cfg.ServerURL, cfg.Token)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	calendars, err := client.ListCalendars(ctx)
	if err != nil {
		return fmt.Errorf("lỗi lấy danh sách lịch: %w", err)
	}

	// Lấy trạng thái google mapping nếu có
	googleStatus, _ := client.GetGoogleStatus(ctx)

	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Println("Danh Sách Lịch:")
		for _, cal := range calendars {
			defTag := ""
			if cal.IsDefault {
				defTag = " [Mặc định]"
			}
			gTag := ""
			if googleStatus != nil && googleStatus.Connected {
				for _, m := range googleStatus.MappedCalendars {
					if m.CalendarID == cal.ID {
						gTag = fmt.Sprintf(" (Google: %s, %s)", m.ExternalCalendarID, m.SyncDirection)
						break
					}
				}
			}
			fmt.Printf("  • %s (ID: %s)%s%s\n    Múi giờ: %s\n", cal.Name, cal.ID, defTag, gTag, cal.Timezone)
		}
		return nil
	}

	var sb strings.Builder
	sb.WriteString(ui.CardTitle.Render("DANH SÁCH LỊCH TRONG MỸ LÍCH"))
	sb.WriteString("\n\n")

	for i, cal := range calendars {
		defBadge := ""
		if cal.IsDefault {
			defBadge = " " + ui.BadgeSynced
		}
		gBadge := ""
		if googleStatus != nil && googleStatus.Connected {
			for _, m := range googleStatus.MappedCalendars {
				if m.CalendarID == cal.ID {
					gBadge = fmt.Sprintf("\n    %s %s (%s)", ui.LabelStyle.Render("Google:"), ui.ValueStyle.Render(m.ExternalCalendarID), m.SyncDirection)
					break
				}
			}
		}

		sb.WriteString(fmt.Sprintf(
			"%s %s%s\n    %s %s  |  %s %s%s\n",
			ui.CardTitle.Render(fmt.Sprintf("[%d]", i+1)),
			ui.ValueStyle.Render(cal.Name),
			defBadge,
			ui.LabelStyle.Render("ID:"), cal.ID,
			ui.LabelStyle.Render("Múi giờ:"), cal.Timezone,
			gBadge,
		))
	}

	fmt.Println(ui.CardBox.Render(sb.String()))
	return nil
}

func runCalendarAdd(args []string) error {
	fs := flag.NewFlagSet("calendar add", flag.ContinueOnError)
	tzFlag := fs.String("timezone", "Asia/Ho_Chi_Minh", "Múi giờ của lịch")
	descFlag := fs.String("desc", "", "Mô tả lịch")
	syncGoogleFlag := fs.Bool("sync-google", false, "Tự động tạo lịch tương ứng trên Google Calendar và map 2 chiều")
	simpleFlag := fs.Bool("simple", false, "Xuất dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Xuất dạng văn bản ASCII đơn giản (viết tắt)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	name := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if name == "" {
		return fmt.Errorf("vui lòng cung cấp tên lịch: lich calendar add <tên_lịch> [tùy_chọn]")
	}

	cfg, err := config.LoadConfig()
	if err != nil || cfg.Token == "" {
		return fmt.Errorf("chưa đăng nhập. Vui lòng chạy 'lich login' trước")
	}

	client := api.NewClient(cfg.ServerURL, cfg.Token)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// 1. Tạo lịch trên lich-server
	cal, err := client.CreateCalendar(ctx, api.CreateCalendarRequest{
		Name:        name,
		Timezone:    *tzFlag,
		Description: *descFlag,
	})
	if err != nil {
		return fmt.Errorf("lỗi tạo lịch trên máy chủ: %w", err)
	}

	var extCal *api.GoogleExternalCalendar
	var googleErr error
	if *syncGoogleFlag {
		extCal, googleErr = client.CreateGoogleCalendar(ctx, cal.ID, cal.Name, "bidirectional")
	}

	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Printf("✓ Đã tạo lịch thành công: %s (ID: %s)\n", cal.Name, cal.ID)
		if *syncGoogleFlag {
			if googleErr != nil {
				fmt.Printf("  ⚠ Không thể tạo trên Google Calendar: %v\n", googleErr)
			} else if extCal != nil {
				fmt.Printf("  ✓ Đã tự động tạo trên Google Calendar: %s (Google ID: %s)\n", extCal.Name, extCal.ID)
			}
		}
		return nil
	}

	var sb strings.Builder
	sb.WriteString(ui.CardTitle.Render("✓ ĐÃ TẠO LỊCH THÀNH CÔNG"))
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("%s %s\n", ui.LabelStyle.Render("Tên lịch:"), ui.ValueStyle.Render(cal.Name)))
	sb.WriteString(fmt.Sprintf("%s %s\n", ui.LabelStyle.Render("ID lịch:"), cal.ID))
	sb.WriteString(fmt.Sprintf("%s %s\n", ui.LabelStyle.Render("Múi giờ:"), cal.Timezone))

	if *syncGoogleFlag {
		if googleErr != nil {
			sb.WriteString(fmt.Sprintf("\n%s %v", ui.LabelStyle.Render("⚠ Lỗi Google Sync:"), googleErr))
		} else if extCal != nil {
			sb.WriteString(fmt.Sprintf("\n%s %s\n", ui.LabelStyle.Render("Google Calendar:"), ui.ValueStyle.Render(extCal.Name)))
			sb.WriteString(fmt.Sprintf("%s %s (bidirectional)\n", ui.LabelStyle.Render("Google ID:"), extCal.ID))
		}
	}

	fmt.Println(ui.CardBoxSuccess.Render(sb.String()))
	return nil
}

func runCalendarDelete(args []string) error {
	fs := flag.NewFlagSet("calendar delete", flag.ContinueOnError)
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
		return fmt.Errorf("vui lòng cung cấp ID lịch cần xóa: lich calendar delete <calendar_id>")
	}

	cfg, err := config.LoadConfig()
	if err != nil || cfg.Token == "" {
		return fmt.Errorf("chưa đăng nhập. Vui lòng chạy 'lich login' trước")
	}

	client := api.NewClient(cfg.ServerURL, cfg.Token)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.DeleteCalendar(ctx, calID); err != nil {
		return fmt.Errorf("lỗi xóa lịch: %w", err)
	}

	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Printf("✓ Đã xóa lịch '%s' thành công.\n", calID)
	} else {
		fmt.Println(ui.CardBoxSuccess.Render(fmt.Sprintf(
			"%s\n\n%s %s",
			ui.CardTitle.Render("✓ ĐÃ XÓA LỊCH THÀNH CÔNG"),
			ui.LabelStyle.Render("ID lịch đã xóa:"), calID,
		)))
	}
	return nil
}
