package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
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

	fs.Usage = func() {
		fmt.Println("Sử dụng: lich sync [flags]")
		fmt.Println()
		fmt.Println("Mô tả:")
		fmt.Println("  Đồng bộ hóa 2 chiều (Push & Pull) giữa Local SQLite Cache và Máy chủ Lich.")
		fmt.Println()
		fmt.Println("Tùy chọn:")
		fmt.Println("  --wait, -w     Chờ quá trình đồng bộ hoàn tất và hiển thị chi tiết")
		fmt.Println("  --simple, -s   Hiển thị dạng văn bản ASCII đơn giản")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
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
		// Chạy sync nhanh (timeout 10s), không chờ kết quả chi tiết
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pushed, pulled, syncErr := engine.Sync(ctx)
		if ui.IsSimpleMode(*simpleFlag) {
			if syncErr != nil {
				fmt.Printf("⚠ Đồng bộ thất bại: %v\n", syncErr)
			} else {
				fmt.Printf("✓ Đã đồng bộ (↑%d ↓%d)\n", pushed, pulled)
			}
		} else {
			statusLabel := ui.BadgeSynced
			cardStyle := ui.CardBoxSuccess
			title := "✓ ĐỒNG BỘ HOÀN TẤT"
			if syncErr != nil {
				statusLabel = ui.BadgeFailed
				cardStyle = ui.CardBoxError
				title = "⚠ ĐỒNG BỘ THẤT BẠI"
			}
			fmt.Println(cardStyle.Render(fmt.Sprintf(
				"%s\n\n%s %s\n%s %s\n%s %s",
				ui.CardTitle.Render(title),
				ui.LabelStyle.Render("Máy chủ:   "), ui.ValueStyle.Render(cfg.ServerURL),
				ui.LabelStyle.Render("Kết quả:   "), ui.ValueStyle.Render(fmt.Sprintf("↑%d đẩy lên  ↓%d nhận về", pushed, pulled)),
				ui.LabelStyle.Render("Trạng thái:"), statusLabel,
			)))
		}
		return nil
	}

	// -w: chạy sync với live progress
	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Printf("↻ Đang đồng bộ hóa với %s...\n", cfg.ServerURL)
	} else {
		fmt.Printf("%s  %s\n\n",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#74C0FC")).Render("↻ ĐỒNG BỘ HÓA"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")).Render(cfg.ServerURL),
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	iconStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")).Bold(true)
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8")).Bold(true)
	pushStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89DCEB"))
	pullStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CBA6F7"))

	pushed, pulled, syncErr := engine.SyncWithProgress(ctx, func(ev syncer.ProgressEvent) {
		if ui.IsSimpleMode(*simpleFlag) {
			fmt.Printf("  %s\n", ev.Message)
			return
		}
		switch ev.Kind {
		case syncer.ProgressStart:
			if ev.Total == 0 {
				fmt.Printf("  %s  %s\n", dimStyle.Render("◦"), dimStyle.Render("Không có thao tác nào chờ đẩy lên"))
			} else {
				fmt.Printf("  %s  %s\n", iconStyle.Render("↑"), pushStyle.Render(fmt.Sprintf("Đẩy lên %d thao tác...", ev.Total)))
			}
		case syncer.ProgressPush:
			bar := renderProgressBar(ev.Current, ev.Total, 16)
			fmt.Printf("    %s  %s\n", bar, dimStyle.Render(ev.Message))
		case syncer.ProgressSkip:
			fmt.Printf("    %s  %s\n", dimStyle.Render("—"), dimStyle.Render(ev.Message))
		case syncer.ProgressPull:
			fmt.Printf("  %s  %s\n", iconStyle.Render("↓"), pullStyle.Render(ev.Message))
		case syncer.ProgressError:
			fmt.Printf("    %s  %s\n", errStyle.Render("✗"), errStyle.Render(ev.Message))
		case syncer.ProgressDone:
			fmt.Printf("\n  %s\n", okStyle.Render(ev.Message))
		}
	})

	if syncErr != nil {
		if ui.IsSimpleMode(*simpleFlag) {
			return fmt.Errorf("đồng bộ hóa thất bại: %w", syncErr)
		}
		fmt.Printf("\n  %s\n", errStyle.Render(fmt.Sprintf("✗ Thất bại: %v", syncErr)))
		return nil
	}

	if !ui.IsSimpleMode(*simpleFlag) {
		fmt.Println()
		summaryCard := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#A6E3A1")).
			Padding(0, 2).
			Render(fmt.Sprintf("%s    %s    %s",
				okStyle.Render("✓ Hoàn tất"),
				pushStyle.Render(fmt.Sprintf("↑ %d đẩy lên", pushed)),
				pullStyle.Render(fmt.Sprintf("↓ %d nhận về", pulled)),
			))
		fmt.Println(summaryCard)
	} else {
		fmt.Println("✓ Hoàn tất đồng bộ hóa:")
		fmt.Printf("  - Đã đẩy lên server:   %d thao tác\n", pushed)
		fmt.Printf("  - Nhận mới từ server:  %d thay đổi\n", pulled)
		fmt.Println("  - Trạng thái:          ✓ Synced")
	}

	return nil
}

// renderProgressBar tạo progress bar dạng ASCII cho N/Total
func renderProgressBar(current, total, width int) string {
	if total == 0 {
		return strings.Repeat("─", width)
	}
	filled := (current * width) / total
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#89DCEB")).Render(bar)
}
