package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"lich-cli/internal/api"
	"lich-cli/internal/cache"
	"lich-cli/internal/config"
	"lich-cli/internal/tui"
	"lich-cli/internal/ui"
)

const Version = "v0.2.0"

func printHelp(simple bool) {
	if ui.IsSimpleMode(simple) {
		fmt.Printf(`Lich — Personal Calendar System (%s)

Sử dụng:
  lich                          Mở giao diện Terminal tương tác (TUI)
  lich login [options]          Đăng nhập hoặc đăng ký tài khoản (Interactive)
  lich status [options]         Kiểm tra máy chủ, cache và sync queue (--simple)
  lich sync [options]           Đồng bộ hóa 2 chiều với máy chủ (--wait)
  lich today [options]          Xem lịch trình hôm nay
  lich week [options]           Xem lịch trình cả tuần
  lich month [options]          Xem lịch trình cả tháng
  lich search <keyword>         Tìm kiếm sự kiện theo từ khóa
  lich add <title> [options]    Tạo sự kiện mới (Form tương tác nếu không truyền tham số)
  lich edit <id> [options]      Chỉnh sửa sự kiện
  lich delete <id> [options]    Xóa sự kiện (Hỏi xác nhận hoặc chọn từ danh sách)
  lich google <action>          Tích hợp Google Calendar (connect, status, sync, calendars)
  lich nuke-database            Xóa sạch toàn bộ SQLite cache cục bộ
  lich version                  Xem phiên bản hiện tại
  lich help                     Hiển thị trợ giúp này

Các cờ chung:
  --simple, -s                  Xuất văn bản ASCII đơn giản (phù hợp với scripts/CI)
  --json                        Xuất kết quả dưới định dạng JSON

Tùy chọn cho 'add':
  --date <date>                 Ngày sự kiện (YYYY-MM-DD, today, tomorrow)
  --at <time>                   Giờ bắt đầu (10:00, 23:30, 11:30pm, 10am)
  --to <time>                   Giờ kết thúc (22:33, 03:00, 3am)
  --duration <duration>         Thời lượng sự kiện (30m, 1h, 2h30m)
  --calendar <id>               ID lịch đích
  --desc <text>                 Ghi chú mô tả
  --location <text>             Địa điểm
  --timezone <tz>               Múi giờ (ví dụ: Asia/Ho_Chi_Minh)
`, Version)
		return
	}

	banner := ui.TitleBanner.Render(fmt.Sprintf(" MỸ LÍCH — HỆ THỐNG LỊCH CÁ NHÂN (%s) ", Version))
	fmt.Println(banner)

	commandsHelp := fmt.Sprintf(`%s
  %s    %s
  %s    %s
  %s   %s
  %s   %s
  %s     %s
  %s    %s
  %s     %s
  %s    %s
  %s   %s
  %s      %s
  %s   %s
  %s %s
  %s  %s
  %s  %s
  %s  %s

%s
  %s     %s
  %s           %s`,
		ui.CardTitle.Render("CÁC LỆNH SỬ DỤNG:"),
		ui.ValueStyle.Render("lich"), ui.LabelStyle.Render("Mở giao diện Terminal tương tác (TUI)"),
		ui.ValueStyle.Render("lich login"), ui.LabelStyle.Render("Đăng nhập hoặc đăng ký tài khoản (Interactive form)"),
		ui.ValueStyle.Render("lich status"), ui.LabelStyle.Render("Bảng điều khiển trạng thái hệ thống, cache & sync"),
		ui.ValueStyle.Render("lich config"), ui.LabelStyle.Render("Quản lý cấu hình (icon theme: unicode/nerd/ascii/emoji)"),
		ui.ValueStyle.Render("lich calendar"), ui.LabelStyle.Render("Quản lý danh sách lịch (list, add, delete, --sync-google)"),
		ui.ValueStyle.Render("lich sync"), ui.LabelStyle.Render("Đồng bộ hóa 2 chiều với máy chủ (--wait)"),
		ui.ValueStyle.Render("lich today"), ui.LabelStyle.Render("Xem lịch trình hôm nay"),
		ui.ValueStyle.Render("lich week"), ui.LabelStyle.Render("Xem lịch trình cả tuần"),
		ui.ValueStyle.Render("lich month"), ui.LabelStyle.Render("Xem lịch trình cả tháng"),
		ui.ValueStyle.Render("lich search"), ui.LabelStyle.Render("Tìm kiếm sự kiện theo từ khóa"),
		ui.ValueStyle.Render("lich add"), ui.LabelStyle.Render("Tạo sự kiện mới (Interactive form nếu không truyền cờ)"),
		ui.ValueStyle.Render("lich edit"), ui.LabelStyle.Render("Chỉnh sửa sự kiện (Interactive form hoặc flags)"),
		ui.ValueStyle.Render("lich google"), ui.LabelStyle.Render("Tích hợp Google Calendar (connect, status, sync, calendars)"),
		ui.ValueStyle.Render("lich delete"), ui.LabelStyle.Render("Xóa sự kiện (Interactive select & confirm)"),
		ui.ValueStyle.Render("lich help"), ui.LabelStyle.Render("Hiển thị bảng hướng dẫn này"),
		ui.CardTitle.Render("CỜ TOÀN CỤC:"),
		ui.ValueStyle.Render("--simple, -s"), ui.LabelStyle.Render("Xuất văn bản ASCII đơn giản (cho scripts/CI)"),
		ui.ValueStyle.Render("--json"), ui.LabelStyle.Render("Xuất kết quả dưới định dạng JSON"),
	)

	fmt.Println(ui.CardBox.Width(78).Render(commandsHelp))
}

func Execute(args []string) int {
	if len(args) == 0 {
		return runTUI()
	}

	command := args[0]
	subArgs := args[1:]

	var err error
	switch command {
	case "login":
		err = RunLogin(subArgs)
	case "status":
		err = RunStatus(subArgs)
	case "config":
		err = RunConfig(subArgs)
	case "calendar", "calendars", "cal":
		err = RunCalendar(subArgs)
	case "sync":
		err = RunSync(subArgs)
	case "today":
		err = RunToday(subArgs)
	case "week":
		err = RunWeek(subArgs)
	case "month":
		err = RunMonth(subArgs)
	case "search":
		err = RunSearch(subArgs)
	case "add":
		err = RunAdd(subArgs)
	case "edit":
		err = RunEdit(subArgs)
	case "delete":
		err = RunDelete(subArgs)
	case "google":
		err = RunGoogle(subArgs)
	case "nuke-database", "nuke":
		err = RunNuke(subArgs)
	case "version", "-v", "--version":
		fmt.Printf("Lich %s\n", Version)
		return 0
	case "help", "-h", "--help":
		simple := false
		for _, a := range subArgs {
			if a == "--simple" || a == "-s" {
				simple = true
			}
		}
		printHelp(simple)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Lệnh không hợp lệ '%s'. Gõ 'lich help' để xem danh sách lệnh.\n", command)
		return 1
	}

	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		if ui.IsSimpleMode(false) {
			fmt.Fprintf(os.Stderr, "Lỗi: %v\n", err)
		} else {
			errCard := ui.CardBoxError.Render(fmt.Sprintf(
				"%s\n\n%s",
				ui.CardTitle.Render("⚠ ĐÃ XẢY RA LỖI"),
				err.Error(),
			))
			fmt.Fprintln(os.Stderr, errCard)
		}
		return 1
	}

	return 0
}

func runTUI() int {
	cfg, err := config.LoadConfig()
	var client *api.Client
	if err == nil && cfg.Token != "" {
		client = api.NewClient(cfg.ServerURL, cfg.Token)
	}

	cachePath, _ := cache.GetCachePath()
	db, _ := cache.OpenDatabase(cachePath)

	p := tea.NewProgram(tui.NewModel(client, db), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Lỗi chạy giao diện TUI: %v\n", err)
		return 1
	}
	return 0
}
