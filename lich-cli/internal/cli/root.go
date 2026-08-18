package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"lich-cli/internal/api"
	"lich-cli/internal/cache"
	"lich-cli/internal/config"
	"lich-cli/internal/tui"
)

const Version = "v0.2.0"

func printHelp() {
	fmt.Printf(`Lich — Personal Calendar System (%s)

Sử dụng:
  lich                          Mở giao diện Terminal tương tác (TUI)
  lich login [options]          Đăng nhập hoặc đăng ký tài khoản
  lich status                   Kiểm tra trạng thái máy chủ, cache và hàng đợi sync
  lich sync [options]           Đồng bộ hóa 2 chiều với máy chủ (--wait)
  lich today [options]          Xem lịch trình hôm nay
  lich week [options]           Xem lịch trình cả tuần
  lich month [options]          Xem lịch trình cả tháng
  lich search <keyword>         Tìm kiếm sự kiện theo từ khóa
  lich add <title> [options]    Tạo sự kiện mới (Local-First tức thì)
  lich delete <id>              Xóa sự kiện
  lich version                  Xem phiên bản hiện tại
  lich help                     Hiển thị trợ giúp này

Tùy chọn cho 'add':
  --date <date>                 Ngày sự kiện (YYYY-MM-DD, today, tomorrow, default: today)
  --at <time>                   Giờ bắt đầu (ví dụ: 10:00, 23:30, 11:30pm, 10am)
  --to <time>                   Giờ kết thúc (ví dụ: 22:33, 03:00, 3am)
  --duration <duration>         Thời lượng (ví dụ: 30m, 1h, 2h30m, default: 1h)
  --calendar <id>               ID lịch đích
  --desc <text>                 Ghi chú chi tiết
  --location <text>             Địa điểm
  --timezone <tz>               Múi giờ (ví dụ: Asia/Ho_Chi_Minh)

Tùy chọn cho 'today' / 'week' / 'month' / 'search':
  --calendar <id>               Lọc theo ID lịch
  --json                        Xuất kết quả định dạng JSON

Tùy chọn cho 'login':
  --server <url>                URL máy chủ (mặc định: http://127.0.0.1:3000)
  --username <user>             Tên đăng nhập
  --password <pass>             Mật khẩu
  --register                    Đăng ký tài khoản mới
`, Version)
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
	case "delete":
		err = RunDelete(subArgs)
	case "version", "-v", "--version":
		fmt.Printf("Lich %s\n", Version)
		return 0
	case "help", "-h", "--help":
		printHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Lệnh không hợp lệ '%s'. Gõ 'lich help' để xem danh sách lệnh.\n", command)
		return 1
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Lỗi: %v\n", err)
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
