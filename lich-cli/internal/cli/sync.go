package cli

import (
	"context"
	"flag"
	"fmt"
	"time"

	"lich-cli/internal/api"
	"lich-cli/internal/cache"
	"lich-cli/internal/config"
	"lich-cli/internal/syncer"
)

func RunSync(args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	waitFlag := fs.Bool("wait", false, "Chờ quá trình đồng bộ hoàn tất")

	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil || cfg.Token == "" {
		return fmt.Errorf("chưa đăng nhập. Vui lòng chạy 'lich login' trước")
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

	client := api.NewClient(cfg.ServerURL, cfg.Token)
	engine := syncer.NewSyncEngine(db, client)

	if !*waitFlag {
		fmt.Println("↻ Bắt đầu đồng bộ hóa ngầm...")
		engine.SyncInBackground()
		fmt.Println("✓ Lệnh đồng bộ đã được phát động trong nền.")
		return nil
	}

	fmt.Printf("↻ Đang đồng bộ hóa với %s...\n", cfg.ServerURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pushed, pulled, err := engine.Sync(ctx)
	if err != nil {
		return fmt.Errorf("đồng bộ hóa thất bại: %w", err)
	}

	fmt.Println("✓ Hoàn tất đồng bộ hóa:")
	fmt.Printf("  - Đã đẩy lên server:   %d thao tác\n", pushed)
	fmt.Printf("  - Nhận mới từ server:  %d thay đổi\n", pulled)
	fmt.Println("  - Trạng thái:          ✓ Synced")

	return nil
}
