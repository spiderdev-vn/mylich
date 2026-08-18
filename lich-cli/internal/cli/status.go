package cli

import (
	"context"
	"fmt"
	"time"

	"lich-cli/internal/api"
	"lich-cli/internal/cache"
	"lich-cli/internal/config"
)

func RunStatus(_ []string) error {
	fmt.Println("Lich System Status")
	fmt.Println("==================")

	// 1. Kiểm tra cấu hình & Auth
	cfg, err := config.LoadConfig()
	if err != nil || cfg.Token == "" {
		fmt.Println("Tài khoản:     ⚠ Chưa đăng nhập (chạy 'lich login')")
		fmt.Println("Máy chủ:       -")
	} else {
		fmt.Printf("Tài khoản:     ✓ %s\n", cfg.Username)
		fmt.Printf("Máy chủ:       %s\n", cfg.ServerURL)

		// 2. Ping Server Health
		client := api.NewClient(cfg.ServerURL, cfg.Token)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := client.Health(ctx); err != nil {
			fmt.Printf("Kết nối:       ⚠ Không thể kết nối tới server (%v)\n", err)
		} else {
			fmt.Println("Kết nối:       ✓ Trực tuyến (Online)")
		}
	}

	// 3. Kiểm tra Local Cache
	cachePath, err := cache.GetCachePath()
	if err != nil {
		fmt.Printf("Cache cục bộ:  ⚠ Lỗi đường dẫn: %v\n", err)
		return nil
	}

	db, err := cache.OpenDatabase(cachePath)
	if err != nil {
		fmt.Printf("Cache cục bộ:  ⚠ Lỗi database: %v\n", err)
		return nil
	}
	defer db.Close()

	fmt.Printf("Cache file:    %s\n", cachePath)

	// Đếm tổng số sự kiện trong cache
	events, _ := cache.GetEventsInRange(db, "", "", "")
	fmt.Printf("Tổng sự kiện:  %d sự kiện cục bộ\n", len(events))

	// 4. Kiểm tra hàng đợi đồng bộ
	pendingCount, _ := cache.GetPendingJobCount(db)
	if pendingCount > 0 {
		fmt.Printf("Hàng đợi sync: ↻ %d thao tác đang chờ đẩy lên server\n", pendingCount)
	} else {
		fmt.Println("Hàng đợi sync: ✓ Đã đồng bộ toàn bộ (0 pending)")
	}

	// 5. Thời điểm đồng bộ gần nhất
	lastSync, _ := cache.GetLastSyncTime(db)
	if lastSync != nil {
		fmt.Printf("Sync gần nhất: %s\n", lastSync.In(time.Local).Format("15:04:05 02/01/2006"))
	} else {
		fmt.Println("Sync gần nhất: Chưa thực hiện đồng bộ")
	}

	return nil
}
