package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/api"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/cache"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/config"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/syncer"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/ui"
)

func RunSync(args []string) error {
	var flagArgs []string
	var positionalArgs []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
		} else {
			positionalArgs = append(positionalArgs, arg)
		}
	}

	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	directionFlag := fs.String("direction", "", "Hướng đồng bộ: push, pull, hoặc both (mặc định: both)")
	fs.StringVar(directionFlag, "d", "", "Hướng đồng bộ (viết tắt)")
	waitFlag := fs.Bool("wait", false, "Chờ quá trình đồng bộ hoàn tất và hiển thị chi tiết")
	fs.BoolVar(waitFlag, "w", false, "Chờ quá trình đồng bộ hoàn tất (viết tắt)")
	simpleFlag := fs.Bool("simple", false, "Hiển thị dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị dạng văn bản ASCII đơn giản (viết tắt)")

	fs.Usage = func() {
		fmt.Println("Sử dụng: lich sync [push|pull|both] [flags]")
		fmt.Println()
		fmt.Println("Mô tả:")
		fmt.Println("  Đồng bộ hóa dữ liệu giữa Local SQLite Cache và Máy chủ Lich.")
		fmt.Println("  - 'lich sync' hoặc 'lich sync both': Đồng bộ 2 chiều (Push & Pull).")
		fmt.Println("  - 'lich sync push':                  Chỉ đẩy thay đổi cục bộ lên server.")
		fmt.Println("  - 'lich sync pull':                  Chỉ kéo dữ liệu mới từ server về local.")
		fmt.Println()
		fmt.Println("Tùy chọn:")
		fmt.Println("  --direction, -d <dir>   Hướng đồng bộ: 'push', 'pull', hoặc 'both'")
		fmt.Println("  --wait, -w              Chờ quá trình đồng bộ hoàn tất và hiển thị chi tiết")
		fmt.Println("  --simple, -s            Hiển thị dạng văn bản ASCII đơn giản")
	}

	if err := fs.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// Xác định direction: ưu tiên positional command ("push", "pull", "both"), sau đó đến --direction flag
	direction := "both"
	if len(positionalArgs) > 0 {
		cmd := strings.ToLower(positionalArgs[0])
		if cmd == "push" || cmd == "pull" || cmd == "both" {
			direction = cmd
		} else {
			return fmt.Errorf("hướng đồng bộ không hợp lệ '%s'. Chọn 'push', 'pull', hoặc 'both'", positionalArgs[0])
		}
	} else if *directionFlag != "" {
		cmd := strings.ToLower(*directionFlag)
		if cmd == "push" || cmd == "pull" || cmd == "both" {
			direction = cmd
		} else {
			return fmt.Errorf("hướng đồng bộ '--direction' không hợp lệ '%s'. Chọn 'push', 'pull', hoặc 'both'", *directionFlag)
		}
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

	// Chế độ chạy nhanh (không -w)
	if !*waitFlag {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var pushed, pulled int
		var syncErr error

		if ui.IsSimpleMode(*simpleFlag) {
			switch direction {
			case "push":
				pushed, syncErr = engine.Push(ctx)
			case "pull":
				pulled, syncErr = engine.Pull(ctx)
			default:
				pushed, pulled, syncErr = engine.Sync(ctx)
			}
		} else {
			syncTitle := fmt.Sprintf("ĐỒNG BỘ DỮ LIỆU VỚI MÁY CHỦ [%s]", strings.ToUpper(direction))
			stepTitles := []string{
				"Đẩy các thay đổi cục bộ lên máy chủ",
				"Kéo dữ liệu mới nhất từ máy chủ",
			}
			if direction == "push" {
				stepTitles = []string{"Đẩy các thay đổi cục bộ lên máy chủ"}
			} else if direction == "pull" {
				stepTitles = []string{"Kéo dữ liệu mới nhất từ máy chủ"}
			}

			_ = ui.RunWithTracker(syncTitle, stepTitles, func(reporter *ui.TrackerReporter) error {
				if direction == "push" {
					reporter.SetStepRunning(0, "Đang xử lý hàng đợi đẩy...")
					pushed, syncErr = engine.Push(ctx)
					if syncErr != nil {
						reporter.SetStepFailed(0, syncErr.Error())
						return syncErr
					}
					reporter.SetStepDone(0, fmt.Sprintf("Đã đẩy %d thao tác", pushed))
					return nil
				} else if direction == "pull" {
					reporter.SetStepRunning(0, "Đang tải dữ liệu mới...")
					pulled, syncErr = engine.Pull(ctx)
					if syncErr != nil {
						reporter.SetStepFailed(0, syncErr.Error())
						return syncErr
					}
					reporter.SetStepDone(0, fmt.Sprintf("Đã nhận %d thay đổi", pulled))
					return nil
				}

				// BOTH
				reporter.SetStepRunning(0, "Đang xử lý hàng đợi đẩy...")
				pushed, syncErr = engine.Push(ctx)
				if syncErr != nil {
					reporter.SetStepFailed(0, syncErr.Error())
					return syncErr
				}
				reporter.SetStepDone(0, fmt.Sprintf("Đã đẩy %d thao tác", pushed))

				reporter.SetStepRunning(1, "Đang tải dữ liệu mới...")
				pulled, syncErr = engine.Pull(ctx)
				if syncErr != nil {
					reporter.SetStepFailed(1, syncErr.Error())
					return syncErr
				}
				reporter.SetStepDone(1, fmt.Sprintf("Đã nhận %d thay đổi", pulled))
				return nil
			})
		}

		if ui.IsSimpleMode(*simpleFlag) {
			if syncErr != nil {
				fmt.Printf("⚠ Đồng bộ thất bại: %v\n", syncErr)
			} else {
				switch direction {
				case "push":
					fmt.Printf("✓ Đã đẩy lên máy chủ: %d thao tác\n", pushed)
				case "pull":
					fmt.Printf("✓ Đã nhận từ máy chủ: %d thay đổi\n", pulled)
				default:
					fmt.Printf("✓ Đã đồng bộ 2 chiều (↑%d ↓%d)\n", pushed, pulled)
				}
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

			resultText := ""
			switch direction {
			case "push":
				resultText = fmt.Sprintf("↑%d thao tác đẩy lên server", pushed)
			case "pull":
				resultText = fmt.Sprintf("↓%d thay đổi nhận từ server", pulled)
			default:
				resultText = fmt.Sprintf("↑%d đẩy lên  ↓%d nhận về", pushed, pulled)
			}

			fmt.Println(cardStyle.Render(fmt.Sprintf(
				"%s\n\n%s %s\n%s %s\n%s %s\n%s %s",
				ui.CardTitle.Render(title),
				ui.LabelStyle.Render("Máy chủ:   "), ui.ValueStyle.Render(cfg.ServerURL),
				ui.LabelStyle.Render("Chế độ:    "), ui.ValueStyle.Render(strings.ToUpper(direction)),
				ui.LabelStyle.Render("Kết quả:   "), ui.ValueStyle.Render(resultText),
				ui.LabelStyle.Render("Trạng thái:"), statusLabel,
			)))
		}
		return nil
	}

	// Chế độ chờ hiển thị Live Progress (-w)
	modeTitle := "ĐỒNG BỘ HÓA 2 CHIỀU"
	if direction == "push" {
		modeTitle = "ĐẨY DỮ LIỆU LÊN MÁY CHỦ (PUSH)"
	} else if direction == "pull" {
		modeTitle = "KÉO DỮ LIỆU TỪ MÁY CHỦ (PULL)"
	}

	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Printf("↻ %s với %s...\n", modeTitle, cfg.ServerURL)
	} else {
		fmt.Printf("%s  %s\n\n",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#74C0FC")).Render("↻ "+modeTitle),
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

	progressHandler := func(ev syncer.ProgressEvent) {
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
	}

	var pushed, pulled int
	var syncErr error

	switch direction {
	case "push":
		pushed, syncErr = engine.PushWithProgress(ctx, progressHandler)
	case "pull":
		pulled, syncErr = engine.PullWithProgress(ctx, progressHandler)
	default:
		pushed, pulled, syncErr = engine.SyncWithProgress(ctx, progressHandler)
	}

	if syncErr != nil {
		if ui.IsSimpleMode(*simpleFlag) {
			return fmt.Errorf("đồng bộ hóa thất bại: %w", syncErr)
		}
		fmt.Printf("\n  %s\n", errStyle.Render(fmt.Sprintf("✗ Thất bại: %v", syncErr)))
		return nil
	}

	if !ui.IsSimpleMode(*simpleFlag) {
		fmt.Println()
		var cardText string
		switch direction {
		case "push":
			cardText = fmt.Sprintf("%s    %s", okStyle.Render("✓ Hoàn tất"), pushStyle.Render(fmt.Sprintf("↑ %d đẩy lên", pushed)))
		case "pull":
			cardText = fmt.Sprintf("%s    %s", okStyle.Render("✓ Hoàn tất"), pullStyle.Render(fmt.Sprintf("↓ %d nhận về", pulled)))
		default:
			cardText = fmt.Sprintf("%s    %s    %s",
				okStyle.Render("✓ Hoàn tất"),
				pushStyle.Render(fmt.Sprintf("↑ %d đẩy lên", pushed)),
				pullStyle.Render(fmt.Sprintf("↓ %d nhận về", pulled)),
			)
		}

		summaryCard := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#A6E3A1")).
			Padding(0, 2).
			Render(cardText)
		fmt.Println(summaryCard)
	} else {
		fmt.Println("✓ Hoàn tất đồng bộ hóa:")
		if direction == "push" || direction == "both" {
			fmt.Printf("  - Đã đẩy lên server:   %d thao tác\n", pushed)
		}
		if direction == "pull" || direction == "both" {
			fmt.Printf("  - Nhận mới từ server:  %d thay đổi\n", pulled)
		}
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
