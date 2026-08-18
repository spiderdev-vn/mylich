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
	"lich-cli/internal/ui"
)

func RunSync(args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	waitFlag := fs.Bool("wait", false, "Chờ quá trình đồng bộ hoàn tất")
	fs.BoolVar(waitFlag, "w", false, "Chờ quá trình đồng bộ hoàn tất (viết tắt)")
	simpleFlag := fs.Bool("simple", false, "Hiển thị dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị dạng văn bản ASCII đơn giản (viết tắt)")

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
		engine.SyncInBackground()
		if ui.IsSimpleMode(*simpleFlag) {
			fmt.Println("✓ Lệnh đồng bộ đã được phát động trong nền.")
		} else {
			fmt.Println(ui.CardBoxSuccess.Render(fmt.Sprintf(
				"%s\n\n%s %s\n%s %s",
				ui.CardTitle.Render("↻ ĐỒNG BỘ NỀN"),
				ui.LabelStyle.Render("Máy chủ:   "), ui.ValueStyle.Render(cfg.ServerURL),
				ui.LabelStyle.Render("Trạng thái:"), ui.BadgePending,
			)))
		}
		return nil
	}

	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Printf("↻ Đang đồng bộ hóa với %s...\n", cfg.ServerURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pushed, pulled, err := engine.Sync(ctx)
	if err != nil {
		if ui.IsSimpleMode(*simpleFlag) {
			return fmt.Errorf("đồng bộ hóa thất bại: %w", err)
		}
		errCard := ui.CardBoxError.Render(fmt.Sprintf(
			"%s\n\n%s %s\n%s %v",
			ui.CardTitle.Render("⚠ ĐỒNG BỘ THẤT BẠI"),
			ui.LabelStyle.Render("Máy chủ:"), ui.ValueStyle.Render(cfg.ServerURL),
			ui.LabelStyle.Render("Chi tiết:"), err,
		))
		fmt.Println(errCard)
		return nil
	}

	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Println("✓ Hoàn tất đồng bộ hóa:")
		fmt.Printf("  - Đã đẩy lên server:   %d thao tác\n", pushed)
		fmt.Printf("  - Nhận mới từ server:  %d thay đổi\n", pulled)
		fmt.Println("  - Trạng thái:          ✓ Synced")
		return nil
	}

	syncCard := ui.CardBoxSuccess.Render(fmt.Sprintf(
		"%s\n\n%s %s\n%s %s\n%s %s\n%s %s",
		ui.CardTitle.Render("✓ HOÀN TẤT ĐỒNG BỘ HÓA"),
		ui.LabelStyle.Render("Máy chủ:           "), ui.ValueStyle.Render(cfg.ServerURL),
		ui.LabelStyle.Render("Đã đẩy lên server: "), ui.ValueStyle.Render(fmt.Sprintf("%d thao tác", pushed)),
		ui.LabelStyle.Render("Nhận mới từ server:"), ui.ValueStyle.Render(fmt.Sprintf("%d thay đổi", pulled)),
		ui.LabelStyle.Render("Trạng thái:        "), ui.BadgeSynced,
	))
	fmt.Println(syncCard)

	return nil
}
