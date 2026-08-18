package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"lich-cli/internal/api"
	"lich-cli/internal/cache"
	"lich-cli/internal/config"
	"lich-cli/internal/ui"
)

type StatusReport struct {
	User          string `json:"user"`
	ServerURL     string `json:"server_url"`
	ServerOnline  bool   `json:"server_online"`
	CachePath     string `json:"cache_path"`
	TotalEvents   int    `json:"total_events"`
	PendingJobs   int    `json:"pending_jobs"`
	LastSyncTime  string `json:"last_sync_time,omitempty"`
}

func RunStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	simpleFlag := fs.Bool("simple", false, "Hiển thị dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị dạng văn bản ASCII đơn giản (viết tắt)")
	jsonFlag := fs.Bool("json", false, "Xuất kết quả dưới định dạng JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}

	report := StatusReport{
		User:      "(Chưa đăng nhập)",
		ServerURL: "-",
	}

	// 1. Kiểm tra cấu hình & Auth
	cfg, err := config.LoadConfig()
	if err == nil && cfg.Token != "" {
		report.User = cfg.Username
		report.ServerURL = cfg.ServerURL

		client := api.NewClient(cfg.ServerURL, cfg.Token)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := client.Health(ctx); err == nil {
			report.ServerOnline = true
		}
	}

	// 2. Kiểm tra Local Cache
	cachePath, err := cache.GetCachePath()
	if err == nil {
		report.CachePath = cachePath
		if db, err := cache.OpenDatabase(cachePath); err == nil {
			events, _ := cache.GetEventsInRange(db, "", "", "")
			report.TotalEvents = len(events)

			report.PendingJobs, _ = cache.GetPendingJobCount(db)

			if lastSync, _ := cache.GetLastSyncTime(db); lastSync != nil {
				report.LastSyncTime = lastSync.In(time.Local).Format("15:04:05 02/01/2006")
			}
			db.Close()
		}
	}

	// Xuất JSON nếu được yêu cầu
	if *jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	// Xuất ASCII đơn giản nếu dùng cờ --simple
	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Println("Lich System Status")
		fmt.Println("==================")
		fmt.Printf("Tài khoản:     %s\n", report.User)
		fmt.Printf("Máy chủ:       %s\n", report.ServerURL)
		if report.ServerOnline {
			fmt.Println("Kết nối:       [ONLINE] Trực tuyến")
		} else {
			fmt.Println("Kết nối:       [OFFLINE] Ngoại tuyến")
		}
		fmt.Printf("Cache file:    %s\n", report.CachePath)
		fmt.Printf("Tổng sự kiện:  %d sự kiện cục bộ\n", report.TotalEvents)
		fmt.Printf("Hàng đợi sync: %d thao tác đang chờ\n", report.PendingJobs)
		if report.LastSyncTime != "" {
			fmt.Printf("Sync gần nhất: %s\n", report.LastSyncTime)
		} else {
			fmt.Println("Sync gần nhất: Chưa đồng bộ")
		}
		return nil
	}

	// Giao diện Dashboard Lip Gloss đẹp mắt
	banner := ui.TitleBanner.Render(" ⚡ MỸ LÍCH — TRẠNG THÁI HỆ THỐNG ")

	// Card 1: Server & Auth
	serverStatusBadge := ui.BadgeOffline
	if report.ServerOnline {
		serverStatusBadge = ui.BadgeOnline
	}
	serverCardContent := fmt.Sprintf(
		"%s\n\n%s %s\n%s %s\n%s %s",
		ui.CardTitle.Render("🌐 MÁY CHỦ & TÀI KHOẢN"),
		ui.LabelStyle.Render("Tài khoản: "), ui.ValueStyle.Render(report.User),
		ui.LabelStyle.Render("Máy chủ:   "), ui.ValueStyle.Render(report.ServerURL),
		ui.LabelStyle.Render("Trạng thái:"), serverStatusBadge,
	)
	serverCard := ui.CardBox.Width(38).Render(serverCardContent)

	// Card 2: Cache & Storage
	cacheCardContent := fmt.Sprintf(
		"%s\n\n%s %s\n%s %s\n%s %s",
		ui.CardTitle.Render("💾 BỘ NHỚ ĐỆM CỤC BỘ"),
		ui.LabelStyle.Render("Trạng thái:"), ui.BadgeOnline,
		ui.LabelStyle.Render("Sự kiện:   "), ui.ValueStyle.Render(fmt.Sprintf("%d sự kiện", report.TotalEvents)),
		ui.LabelStyle.Render("Vị trí:    "), ui.ValueStyle.Render("cache.db (SQLite)"),
	)
	cacheCard := ui.CardBoxSecondary.Width(38).Render(cacheCardContent)

	// Card 3: Synchronization Queue
	syncBadge := ui.BadgeSynced
	if report.PendingJobs > 0 {
		syncBadge = ui.BadgePending
	}
	syncTimeStr := report.LastSyncTime
	if syncTimeStr == "" {
		syncTimeStr = "Chưa thực hiện"
	}
	syncCardContent := fmt.Sprintf(
		"%s\n\n%s %s\n%s %s\n%s %s",
		ui.CardTitle.Render("🔄 ĐỒNG BỘ HÓA BẤT ĐỒNG BỘ"),
		ui.LabelStyle.Render("Hàng đợi:  "), ui.ValueStyle.Render(fmt.Sprintf("%d thao tác chờ", report.PendingJobs)),
		ui.LabelStyle.Render("Sync gần nhất:"), ui.ValueStyle.Render(syncTimeStr),
		ui.LabelStyle.Render("Trạng thái:   "), syncBadge,
	)
	syncCard := ui.CardBoxSuccess.Width(78).Render(syncCardContent)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, serverCard, cacheCard)
	fullDashboard := lipgloss.JoinVertical(lipgloss.Left, banner, topRow, syncCard)

	fmt.Println(fullDashboard)
	return nil
}
