package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"lich-cli/internal/api"
	"lich-cli/internal/cache"
	"lich-cli/internal/config"
	"lich-cli/internal/syncer"
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

	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	if len(positionalArgs) == 0 {
		return fmt.Errorf("cần nhập từ khóa tìm kiếm: lich search <từ khóa>")
	}
	keyword := strings.Join(positionalArgs, " ")

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

	// Kích hoạt sync ngầm
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
