package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lich-cli/internal/cache"
	"lich-cli/internal/ui"
)

const ConfirmationString = "yes, please, i want to remove all the local data!"

func RunNuke(args []string) error {
	fs := flag.NewFlagSet("nuke-database", flag.ContinueOnError)
	forceFlag := fs.Bool("force", false, "Bỏ qua prompt và nuke ngay lập tức")
	simpleFlag := fs.Bool("simple", false, "Hiển thị dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị dạng văn bản ASCII đơn giản (viết tắt)")

	fs.Usage = func() {
		fmt.Println("Sử dụng: lich nuke-database [flags]")
		fmt.Println()
		fmt.Println("Mô tả:")
		fmt.Println("  XÓA SẠCH toàn bộ cơ sở dữ liệu SQLite cục bộ (sự kiện, lịch, sync queue).")
		fmt.Println()
		fmt.Println("Tùy chọn:")
		fmt.Println("  --force        Bỏ qua prompt xác nhận")
		fmt.Println("  --simple, -s   Hiển thị dạng văn bản ASCII đơn giản")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if !*forceFlag {
		if ui.IsSimpleMode(*simpleFlag) {
			fmt.Println("CẢNH BÁO: Thao tác này sẽ xóa toàn bộ dữ liệu lịch và hàng đợi đồng bộ cục bộ!")
			fmt.Printf("Bạn có chắc chắn không? Nhập chính xác \"%s\" để xác nhận:\n> ", ConfirmationString)
		} else {
			warnCard := ui.CardBoxError.Render(fmt.Sprintf(
				"%s\n\n%s\n%s",
				ui.CardTitle.Render("⚠ NGUY HIỂM: XÓA SẠCH DATABASE CỤC BỘ"),
				ui.ValueStyle.Render("Thao tác này sẽ xóa toàn bộ sự kiện, cache và sync queue trên máy này!"),
				ui.LabelStyle.Render(fmt.Sprintf("Nhập chính xác: \"%s\"", ConfirmationString)),
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
		if input != ConfirmationString {
			if ui.IsSimpleMode(*simpleFlag) {
				fmt.Println("Hủy bỏ thao tác. Database cục bộ được giữ nguyên.")
			} else {
				fmt.Println(ui.CardBox.Render(fmt.Sprintf(
					"%s\n\n%s",
					ui.CardTitle.Render("ĐÃ HỦY THAO TÁC"),
					ui.LabelStyle.Render("Chuỗi xác nhận không khớp. Database cục bộ được giữ nguyên an toàn."),
				)))
			}
			return nil
		}
	}

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
		fmt.Println("✓ Đã xóa sạch toàn bộ database cục bộ thành công!")
	} else {
		successCard := ui.CardBoxSuccess.Render(fmt.Sprintf(
			"%s\n\n%s %s\n%s %s",
			ui.CardTitle.Render("✓ ĐÃ XÓA SẠCH DATABASE CỤC BỘ"),
			ui.LabelStyle.Render("Vị trí cache:"), ui.ValueStyle.Render(cachePath),
			ui.LabelStyle.Render("Trạng thái:  "), ui.BadgeSynced,
		))
		fmt.Println(successCard)
	}

	return nil
}
