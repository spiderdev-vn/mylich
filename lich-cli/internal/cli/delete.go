package cli

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/api"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/cache"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/config"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/syncer"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/ui"
)

func RunDelete(args []string) error {
	var flagArgs []string
	var positionalArgs []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
		} else {
			positionalArgs = append(positionalArgs, arg)
		}
	}

	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	yesFlag := fs.Bool("yes", false, "Bỏ qua bước xác nhận xóa")
	fs.BoolVar(yesFlag, "y", false, "Bỏ qua bước xác nhận xóa (viết tắt)")
	simpleFlag := fs.Bool("simple", false, "Hiển thị dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị dạng văn bản ASCII đơn giản (viết tắt)")

	fs.Usage = func() {
		fmt.Println("Sử dụng: lich delete [id] [flags]")
		fmt.Println()
		fmt.Println("Mô tả:")
		fmt.Println("  Xóa sự kiện. Nếu không truyền ID, sẽ mở danh sách chọn và hỏi xác nhận.")
		fmt.Println()
		fmt.Println("Tùy chọn:")
		fmt.Println("  --yes, -y      Bỏ qua bước xác nhận xóa")
		fmt.Println("  --simple, -s   Hiển thị dạng văn bản ASCII đơn giản")
	}

	if err := fs.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
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

	var eventID string
	if len(positionalArgs) > 0 {
		eventID = positionalArgs[0]
	}

	// Nếu không truyền ID và ở chế độ interactive -> Mở danh sách chọn bằng Huh Select
	if eventID == "" && !ui.IsSimpleMode(*simpleFlag) {
		events, err := cache.GetEventsInRange(db, "", "", "")
		if err != nil || len(events) == 0 {
			fmt.Println("Không có sự kiện nào trong bộ nhớ đệm để xóa.")
			return nil
		}

		var options []huh.Option[string]
		loc := time.Now().Location()
		for _, e := range events {
			tStart, _ := time.Parse(time.RFC3339, e.StartAt)
			dateStr := tStart.In(loc).Format("02/01 15:04")
			label := fmt.Sprintf("[%s] %s (%s)", dateStr, e.Title, e.ID[:8])
			options = append(options, huh.NewOption(label, e.ID))
		}

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Chọn sự kiện cần xóa").
					Options(options...).
					Value(&eventID),
			),
		).WithKeyMap(ui.DefaultFormKeyMap())

		if err := form.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil
			}
			return err
		}
	}

	if eventID == "" {
		return fmt.Errorf("cần chỉ định ID sự kiện: lich delete <id>")
	}

	// Resolve short ID prefix và báo conflict nếu trùng
	targetEvent, err := cache.ResolveEventByPrefix(db, eventID)
	if err != nil {
		return err
	}
	eventID = targetEvent.ID

	// Hỏi xác nhận xóa nếu chưa có cờ -y và không ở simple mode
	if !*yesFlag && !ui.IsSimpleMode(*simpleFlag) {
		confirm := false
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Bạn có chắc chắn muốn xóa sự kiện '%s' [%s] không?", targetEvent.Title, targetEvent.ID[:8])).
					Affirmative("Có, xóa ngay").
					Negative("Hủy bỏ").
					Value(&confirm),
			),
		).WithKeyMap(ui.DefaultFormKeyMap())

		if err := confirmForm.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil
			}
			return err
		}

		if !confirm {
			fmt.Println("Đã hủy thao tác xóa sự kiện.")
			return nil
		}
	}

	// 1. Đánh dấu xóa trong SQLite cục bộ
	if err := cache.MarkEventPendingDelete(db, eventID); err != nil {
		return fmt.Errorf("lỗi cập nhật sự kiện cục bộ: %w", err)
	}

	// 2. Tạo sync job
	_, _ = cache.EnqueueSyncJob(db, "event", eventID, cache.SyncOpDelete, "")

	// 3. Kích hoạt sync ngầm
	cfg, err := config.LoadConfig()
	if err == nil && cfg.Token != "" {
		client := api.NewClient(cfg.ServerURL, cfg.Token)
		engine := syncer.NewSyncEngine(db, client)
		engine.SyncInBackground()
	}

	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Printf("✓ Đã xóa sự kiện %s (Sync: pending)\n", eventID)
	} else {
		card := ui.CardBoxSuccess.Render(fmt.Sprintf(
			"%s\n\n%s %s\n%s %s",
			ui.CardTitle.Render("✓ ĐÃ XÓA SỰ KIỆN"),
			ui.LabelStyle.Render("ID sự kiện:"), ui.ValueStyle.Render(eventID),
			ui.LabelStyle.Render("Đồng bộ:   "), ui.BadgePending,
		))
		fmt.Println(card)
	}

	return nil
}
