package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lich-cli/internal/api"
	"lich-cli/internal/cache"
	"lich-cli/internal/config"
	"lich-cli/internal/ui"
)

const (
	ConfirmationLocalString  = "yes, please, i want to remove all the local data!"
	ConfirmationRemoteString = "yes, please, i want to remove all the remote and local data!"
)

func RunNuke(args []string) error {
	fs := flag.NewFlagSet("nuke-database", flag.ContinueOnError)
	remoteFlag := fs.Bool("remote", false, "Xóa sạch toàn bộ dữ liệu trên máy chủ Lich (server) đồng thời với local")
	fs.BoolVar(remoteFlag, "r", false, "Xóa sạch dữ liệu trên máy chủ (viết tắt)")
	forceFlag := fs.Bool("force", false, "Bỏ qua prompt và nuke ngay lập tức")
	simpleFlag := fs.Bool("simple", false, "Hiển thị dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị dạng văn bản ASCII đơn giản (viết tắt)")

	fs.Usage = func() {
		fmt.Println("Sử dụng: lich nuke-database [flags]")
		fmt.Println()
		fmt.Println("Mô tả:")
		fmt.Println("  XÓA SẠCH toàn bộ cơ sở dữ liệu SQLite cục bộ (sự kiện, lịch, sync queue).")
		fmt.Println("  Nếu truyền cờ --remote, xóa sạch cả dữ liệu người dùng trên máy chủ Lich.")
		fmt.Println()
		fmt.Println("Tùy chọn:")
		fmt.Println("  --remote, -r   Xóa sạch dữ liệu trên cả máy chủ từ xa (remote server)")
		fmt.Println("  --force        Bỏ qua prompt xác nhận")
		fmt.Println("  --simple, -s   Hiển thị dạng văn bản ASCII đơn giản")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	expectedConfirm := ConfirmationLocalString
	actionTargetName := "DATABASE CỤC BỘ (LOCAL)"
	if *remoteFlag {
		expectedConfirm = ConfirmationRemoteString
		actionTargetName = "CẢ MÁY CHỦ (REMOTE) VÀ CỤC BỘ (LOCAL)"
	}

	var client *api.Client
	var cfg *config.Config
	if *remoteFlag {
		var err error
		cfg, err = config.LoadConfig()
		if err != nil || cfg.Token == "" {
			return fmt.Errorf("chưa đăng nhập. Cần đăng nhập để nuke dữ liệu máy chủ từ xa ('lich login')")
		}
		client = api.NewClient(cfg.ServerURL, cfg.Token)
	}

	if !*forceFlag {
		if ui.IsSimpleMode(*simpleFlag) {
			fmt.Printf("CẢNH BÁO: Thao tác này sẽ xóa toàn bộ dữ liệu trên %s!\n", actionTargetName)
			fmt.Printf("Bạn có chắc chắn không? Nhập chính xác \"%s\" để xác nhận:\n> ", expectedConfirm)
		} else {
			warnCard := ui.CardBoxError.Render(fmt.Sprintf(
				"%s\n\n%s\n%s",
				ui.CardTitle.Render(fmt.Sprintf("⚠ NGUY HIỂM: XÓA SẠCH DỮ LIỆU %s", actionTargetName)),
				ui.ValueStyle.Render("Thao tác này KHÔNG thể khôi phục! Toàn bộ sự kiện, lịch và đồng bộ sẽ bị xóa sạch."),
				ui.LabelStyle.Render(fmt.Sprintf("Nhập chính xác: \"%s\"", expectedConfirm)),
			))
			fmt.Println(warnCard)
			fmt.Print("\n> ")
		}

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("không thể đọc phản hồi từ người dùng: %w", err)
		}

		input = strings.TrimRight(input, "\r\n")
		if input != expectedConfirm {
			if ui.IsSimpleMode(*simpleFlag) {
				fmt.Println("Hủy bỏ thao tác. Dữ liệu được giữ nguyên an toàn.")
			} else {
				fmt.Println(ui.CardBox.Render(fmt.Sprintf(
					"%s\n\n%s",
					ui.CardTitle.Render("ĐÃ HỦY THAO TÁC"),
					ui.LabelStyle.Render("Chuỗi xác nhận không khớp. Dữ liệu được giữ nguyên an toàn."),
				)))
			}
			return nil
		}
	}

	// 1. Nuke Remote nếu có cờ --remote
	if *remoteFlag && client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := client.NukeRemote(ctx); err != nil {
			return fmt.Errorf("lỗi xóa dữ liệu trên máy chủ từ xa: %w", err)
		}
	}

	// 2. Nuke Local SQLite cache
	cachePath, err := cache.GetCachePath()
	if err != nil {
		return fmt.Errorf("không thể lấy đường dẫn cache: %w", err)
	}

	// Xóa các file SQLite database và WAL/SHM
	dir := filepath.Dir(cachePath)
	base := filepath.Base(cachePath)

	_ = os.Remove(cachePath)
	_ = os.Remove(filepath.Join(dir, base+"-wal"))
	_ = os.Remove(filepath.Join(dir, base+"-shm"))
	_ = os.Remove(filepath.Join(dir, base+"-journal"))

	// Khởi tạo lại database trắng với schema mới
	db, err := cache.OpenDatabase(cachePath)
	if err != nil {
		return fmt.Errorf("lỗi khởi tạo lại database sạch: %w", err)
	}
	defer db.Close()

	if ui.IsSimpleMode(*simpleFlag) {
		if *remoteFlag {
			fmt.Println("✓ Đã xóa sạch toàn bộ dữ liệu trên MÁY CHỦ và CỤC BỘ thành công!")
		} else {
			fmt.Println("✓ Đã xóa sạch toàn bộ database cục bộ thành công!")
		}
	} else {
		msg := "Đã dọn dẹp sạch toàn bộ cache SQLite cục bộ và khởi tạo lại cấu trúc mới."
		if *remoteFlag {
			msg = "Đã dọn dẹp sạch toàn bộ dữ liệu trên máy chủ (events, integrations, changelog) và SQLite cache cục bộ."
		}
		resultCard := ui.CardBoxSuccess.Render(fmt.Sprintf(
			"%s\n\n%s\n%s %s",
			ui.CardTitle.Render("✓ ĐÃ XÓA SẠCH DỮ LIỆU THÀNH CÔNG"),
			ui.ValueStyle.Render(msg),
			ui.LabelStyle.Render("Trạng thái:"), ui.BadgeSynced,
		))
		fmt.Println(resultCard)
	}

	return nil
}
