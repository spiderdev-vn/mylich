package cli

import (
	"fmt"

	"lich-cli/internal/api"
	"lich-cli/internal/cache"
	"lich-cli/internal/config"
	"lich-cli/internal/syncer"
)

func RunDelete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("cần chỉ định ID sự kiện: lich delete <id>")
	}
	eventID := args[0]

	cachePath, err := cache.GetCachePath()
	if err != nil {
		return fmt.Errorf("không thể mở thư mục cache: %w", err)
	}

	db, err := cache.OpenDatabase(cachePath)
	if err != nil {
		return fmt.Errorf("lỗi kết nối database cục bộ: %w", err)
	}
	defer db.Close()

	// 1. Đánh dấu xóa trong SQLite cục bộ (người dùng sẽ không thấy sự kiện này nữa)
	if err := cache.MarkEventPendingDelete(db, eventID); err != nil {
		return fmt.Errorf("lỗi cập nhật sự kiện cục bộ: %w", err)
	}

	// 2. Tạo sync job cho thao tác DELETE
	_, _ = cache.EnqueueSyncJob(db, "event", eventID, cache.SyncOpDelete, "")

	// 3. Kích hoạt đồng bộ ngầm
	cfg, err := config.LoadConfig()
	if err == nil && cfg.Token != "" {
		client := api.NewClient(cfg.ServerURL, cfg.Token)
		engine := syncer.NewSyncEngine(db, client)
		engine.SyncInBackground()
	}

	fmt.Printf("✓ Đã xóa sự kiện %s (Sync: pending)\n", eventID)
	return nil
}
