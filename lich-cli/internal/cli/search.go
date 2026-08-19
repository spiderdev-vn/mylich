package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/api"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/cache"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/config"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/syncer"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/ui"
)

func RunSearch(args []string) error {
	var flagArgs []string
	var positionalArgs []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
		} else {
			positionalArgs = append(positionalArgs, arg)
		}
	}

	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	jsonFlag := fs.Bool("json", false, "Xuất kết quả dưới định dạng JSON")
	simpleFlag := fs.Bool("simple", false, "Hiển thị dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị dạng văn bản ASCII đơn giản (viết tắt)")

	fs.Usage = func() {
		fmt.Println("Sử dụng: lich search <từ khóa> [flags]")
		fmt.Println()
		fmt.Println("Tùy chọn:")
		fmt.Println("  --simple, -s   Hiển thị dạng văn bản ASCII đơn giản")
		fmt.Println("  --json         Xuất kết quả dưới định dạng JSON")
	}

	if err := fs.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	keyword := strings.Join(positionalArgs, " ")

	// Nếu không truyền từ khóa và ở chế độ interactive, mở form Huh Input
	if strings.TrimSpace(keyword) == "" && !ui.IsSimpleMode(*simpleFlag) {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Nhập từ khóa tìm kiếm").
					Placeholder("Họp / Khám răng / Sinh nhật").
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("từ khóa tìm kiếm không được để trống")
						}
						return nil
					}).
					Value(&keyword),
			),
		).WithKeyMap(ui.DefaultFormKeyMap())

		if err := form.Run(); err != nil {
			return err
		}
	}

	if strings.TrimSpace(keyword) == "" {
		return fmt.Errorf("cần nhập từ khóa tìm kiếm: lich search <từ khóa>")
	}

	cachePath, err := cache.GetCachePath()
	if err != nil {
		return fmt.Errorf("không thể mở thư mục cache: %w", err)
	}

	db, err := cache.OpenDatabase(cachePath)
	if err != nil {
		return fmt.Errorf("lỗi kết nối database cục bộ: %w", err)
	}
	defer db.Close()

	events, err := cache.SearchEvents(db, keyword)
	if err != nil {
		return fmt.Errorf("lỗi tìm kiếm sự kiện: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err == nil && cfg.Token != "" {
		client := api.NewClient(cfg.ServerURL, cfg.Token)
		engine := syncer.NewSyncEngine(db, client)
		engine.SyncInBackground()
	}

	if *jsonFlag {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(events)
	}

	loc := time.Now().Location()

	if ui.IsSimpleMode(*simpleFlag) {
		searchHeader := fmt.Sprintf("Kết quả tìm kiếm cho '%s' (%d kết quả)", keyword, len(events))
		fmt.Println(searchHeader)
		fmt.Println(strings.Repeat("-", len(searchHeader)))

		if len(events) == 0 {
			fmt.Println("Không tìm thấy sự kiện nào phù hợp.")
			return nil
		}

		for _, event := range events {
			tStart, _ := time.Parse(time.RFC3339, event.StartAt)
			dateStr := tStart.In(loc).Format("02/01/2006")
			timeStr := formatTimeRange(event.StartAt, event.EndAt, loc)
			locStr := ""
			if event.Location != "" {
				locStr = fmt.Sprintf(" - Địa điểm: %s", event.Location)
			}
			fmt.Printf("• [%s %s] %s (ID: %s)%s\n", dateStr, timeStr, event.Title, event.ID, locStr)
		}
		return nil
	}

	icons := ui.CurrentIcons()
	if *simpleFlag {
		icons = ui.IconASCII
	}

	// Lip Gloss Search Results
	headerText := fmt.Sprintf(" %s KẾT QUẢ TÌM KIẾM CHO: '%s' — %d KẾT QUẢ ", icons.Calendar, keyword, len(events))
	fmt.Println(ui.TitleBanner.Render(headerText))

	if len(events) == 0 {
		fmt.Println(ui.EventDescStyle.Render("  (Không tìm thấy sự kiện nào phù hợp)"))
		fmt.Println()
		return nil
	}

	for _, event := range events {
		tStart, _ := time.Parse(time.RFC3339, event.StartAt)
		dateStr := tStart.In(loc).Format("02/01/2006")
		timeStr := formatTimeRange(event.StartAt, event.EndAt, loc)
		syncBadge := ""
		if event.SyncState != cache.SyncStateSynced {
			syncBadge = " " + ui.RenderBadgePending(icons.Pending)
		}

		fmt.Printf("  %s %s  %s%s\n",
			ui.HeaderDateStyle.Render(" "+dateStr+" "),
			ui.TimePill.Render(timeStr),
			ui.EventTitleStyle.Render(event.Title),
			syncBadge,
		)
		fmt.Printf("            %s %s\n", ui.LabelStyle.Render("ID:"), ui.LabelStyle.Render(event.ID))
		if event.Location != "" {
			fmt.Printf("            %s %s\n", ui.LabelStyle.Render(icons.Location), ui.EventLocationStyle.Render(event.Location))
		}
		if event.Description != "" {
			fmt.Printf("            %s\n", ui.EventDescStyle.Render(event.Description))
		}
		fmt.Println()
	}

	return nil
}
