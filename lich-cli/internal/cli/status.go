package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"lich-cli/internal/api"
	"lich-cli/internal/cache"
	"lich-cli/internal/config"
	"lich-cli/internal/ui"
)

type StatusReport struct {
	User         string `json:"user"`
	ServerURL    string `json:"server_url"`
	ServerOnline bool   `json:"server_online"`
	CachePath    string `json:"cache_path"`
	TotalEvents  int    `json:"total_events"`
	PendingJobs  int    `json:"pending_jobs"`
	LastSyncTime string `json:"last_sync_time,omitempty"`
	IconStyle    string `json:"icon_style"`
}

func RunStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	simpleFlag := fs.Bool("simple", false, "Hiển thị dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị dạng văn bản ASCII đơn giản (viết tắt)")
	jsonFlag := fs.Bool("json", false, "Xuất kết quả dưới định dạng JSON")

	fs.Usage = func() {
		fmt.Println("Sử dụng: lich status [flags]")
		fmt.Println()
		fmt.Println("Mô tả:")
		fmt.Println("  Hiển thị bảng trạng thái máy chủ, bộ nhớ đệm cục bộ và hàng đợi đồng bộ.")
		fmt.Println()
		fmt.Println("Tùy chọn:")
		fmt.Println("  --simple, -s   Hiển thị dạng văn bản ASCII đơn giản (phù hợp scripts/CI)")
		fmt.Println("  --json         Xuất kết quả dưới định dạng JSON")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	icons := ui.CurrentIcons()
	if *simpleFlag {
		icons = ui.IconASCII
	}

	report := StatusReport{
		User:      "(Chưa đăng nhập)",
		ServerURL: "-",
		IconStyle: icons.Name,
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

	// Giao diện dạng Hàng đơn (Single Column Card) — Thích ứng động theo Icon Set
	serverStatusBadge := ui.RenderBadgeOffline(icons.Failed)
	if report.ServerOnline {
		serverStatusBadge = ui.RenderBadgeOnline(icons.Dot)
	}

	syncBadge := ui.RenderBadgeSynced(icons.Check)
	if report.PendingJobs > 0 {
		syncBadge = ui.RenderBadgePending(icons.Pending)
	}

	syncTimeStr := report.LastSyncTime
	if syncTimeStr == "" {
		syncTimeStr = "Chưa thực hiện"
	}

	var sb strings.Builder

	// Header banner
	sb.WriteString(ui.TitleBanner.Render(fmt.Sprintf("MỸ LÍCH — TRẠNG THÁI HỆ THỐNG [Theme: %s]", icons.Name)) + "\n\n")

	// Section 1: Server & Auth
	var lines []string
	lines = append(lines, ui.SectionHeaderStyle.Render(fmt.Sprintf("%s MÁY CHỦ & TÀI KHOẢN", icons.Server)))
	lines = append(lines, fmt.Sprintf("  %s %s %s", icons.Bullet, ui.LabelStyle.Render("Tài khoản: "), ui.ValueStyle.Render(report.User)))
	lines = append(lines, fmt.Sprintf("  %s %s %s", icons.Bullet, ui.LabelStyle.Render("Máy chủ:   "), ui.ValueStyle.Render(report.ServerURL)))
	lines = append(lines, fmt.Sprintf("  %s %s %s", icons.Bullet, ui.LabelStyle.Render("Trạng thái:"), serverStatusBadge))
	lines = append(lines, "")

	// Section 2: Local Cache
	lines = append(lines, ui.SectionHeaderStyle.Render(fmt.Sprintf("%s BỘ NHỚ ĐỆM CỤC BỘ (SQLITE)", icons.Database)))
	lines = append(lines, fmt.Sprintf("  %s %s %s", icons.Bullet, ui.LabelStyle.Render("Vị trí:    "), ui.LabelStyle.Render(report.CachePath)))
	lines = append(lines, fmt.Sprintf("  %s %s %s", icons.Bullet, ui.LabelStyle.Render("Sự kiện:   "), ui.ValueStyle.Render(fmt.Sprintf("%d sự kiện", report.TotalEvents))))
	lines = append(lines, "")

	// Section 3: Synchronization
	lines = append(lines, ui.SectionHeaderStyle.Render(fmt.Sprintf("%s ĐỒNG BỘ HÓA 2 CHIỀU", icons.Sync)))
	lines = append(lines, fmt.Sprintf("  %s %s %s", icons.Bullet, ui.LabelStyle.Render("Hàng đợi:  "), ui.ValueStyle.Render(fmt.Sprintf("%d thao tác chờ", report.PendingJobs))))
	lines = append(lines, fmt.Sprintf("  %s %s %s", icons.Bullet, ui.LabelStyle.Render("Sync gần nhất:"), ui.ValueStyle.Render(syncTimeStr)))
	lines = append(lines, fmt.Sprintf("  %s %s %s", icons.Bullet, ui.LabelStyle.Render("Trạng thái:   "), syncBadge))

	cardContent := strings.Join(lines, "\n")
	sb.WriteString(ui.ContainerCard.Render(cardContent))

	fmt.Println(sb.String())
	return nil
}
