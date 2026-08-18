package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"lich-cli/internal/config"
	"lich-cli/internal/ui"

	"github.com/charmbracelet/huh"
)

func RunConfig(args []string) error {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	jsonFlag := fs.Bool("json", false, "Xuất cấu hình dưới định dạng JSON")
	simpleFlag := fs.Bool("simple", false, "Hiển thị dạng văn bản ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị dạng văn bản ASCII đơn giản (viết tắt)")

	fs.Usage = func() {
		fmt.Println("Sử dụng: lich config [subcommand] [flags]")
		fmt.Println()
		fmt.Println("Lệnh con:")
		fmt.Println("  lich config                    Mở form tương tác cấu hình icon theme và máy chủ")
		fmt.Println("  lich config list               Xem toàn bộ cấu hình hiện tại")
		fmt.Println("  lich config get <key>          Xem giá trị một khóa (icons, server_url, username)")
		fmt.Println("  lich config set <key> <value>  Thay đổi giá trị (ví dụ: lich config set icons nerd)")
		fmt.Println()
		fmt.Println("Các bộ icon theme hỗ trợ:")
		fmt.Println("  unicode   Unicode 1-cell (Mặc định - Chuẩn không bao giờ vỡ khung viền)")
		fmt.Println("  nerd      Nerd Font (Dành cho font lập trình: JetBrainsMono, FiraCode)")
		fmt.Println("  ascii     ASCII thuần túy (Dành cho server/scripts)")
		fmt.Println("  emoji     Emoji màu sắc sống động")
		fmt.Println()
		fmt.Println("Tùy chọn:")
		fmt.Println("  --simple, -s   Hiển thị dạng văn bản ASCII đơn giản")
		fmt.Println("  --json         Xuất cấu hình dưới định dạng JSON")
	}

	// Tách subcommand (list, get, set) và flags
	var subArgs []string
	var commandArgs []string

	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			subArgs = append(subArgs, a)
		} else {
			commandArgs = append(commandArgs, a)
		}
	}

	if err := fs.Parse(subArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = &config.Config{ServerURL: config.DefaultServerURL, Icons: config.DefaultIcons}
	}

	configPath, _ := config.GetConfigPath()

	// 1. Nếu có tham số dòng lệnh con (get / set / list)
	if len(commandArgs) > 0 {
		subCommand := strings.ToLower(commandArgs[0])

		switch subCommand {
		case "list", "show", "ls":
			return printConfigList(cfg, configPath, *jsonFlag, *simpleFlag)

		case "get":
			if len(commandArgs) < 2 {
				return fmt.Errorf("cần chỉ định khóa cấu hình: lich config get <key>")
			}
			key := commandArgs[1]
			val, err := cfg.Get(key)
			if err != nil {
				return err
			}
			fmt.Println(val)
			return nil

		case "set":
			if len(commandArgs) < 3 {
				return fmt.Errorf("cần chỉ định khóa và giá trị: lich config set <key> <value>\nVí dụ: lich config set icons nerd")
			}
			key := commandArgs[1]
			val := commandArgs[2]

			if err := cfg.Set(key, val); err != nil {
				return err
			}

			if err := config.SaveConfig(cfg); err != nil {
				return fmt.Errorf("lỗi lưu cấu hình: %w", err)
			}

			if ui.IsSimpleMode(*simpleFlag) {
				fmt.Printf("✓ Đã cập nhật %s = %s\n", key, val)
			} else {
				fmt.Println(ui.CardBoxSuccess.Render(fmt.Sprintf(
					"%s\n\n%s %s\n%s %s",
					ui.CardTitle.Render("✓ ĐÃ CẬP NHẬT CẤU HÌNH"),
					ui.LabelStyle.Render("Khóa:     "), ui.ValueStyle.Render(key),
					ui.LabelStyle.Render("Giá trị:  "), ui.ValueStyle.Render(val),
				)))
			}
			return nil

		default:
			return fmt.Errorf("lệnh con không hợp lệ '%s'. Sử dụng: list, get <key>, set <key> <val>", subCommand)
		}
	}

	// 2. Nếu ở chế độ Non-Interactive (hoặc dùng --simple / --json), xuất danh sách cấu hình
	if ui.IsSimpleMode(*simpleFlag) || *jsonFlag {
		return printConfigList(cfg, configPath, *jsonFlag, *simpleFlag)
	}

	// 3. Chế độ Interactive: Mở Form Huh để người dùng chọn cấu hình trực quan
	selectedIcons := cfg.Icons
	if selectedIcons == "" {
		selectedIcons = config.DefaultIcons
	}
	serverURL := cfg.ServerURL

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Bộ biểu tượng giao diện (Icon Theme)").
				Description("Chọn phong cách icon phù hợp với font và terminal của bạn").
				Options(
					huh.NewOption("Unicode 1-cell (Mặc định - An toàn 100%, không vỡ khung)", "unicode"),
					huh.NewOption("Nerd Font (Glyphs dành cho font lập trình: JetBrainsMono, FiraCode)", "nerd"),
					huh.NewOption("ASCII thuần túy (Dành cho server/scripts)", "ascii"),
					huh.NewOption("Emoji màu sắc (Hiện đại, sinh động)", "emoji"),
				).
				Value(&selectedIcons),

			huh.NewInput().
				Title("Địa chỉ Máy chủ Lich (Server URL)").
				Value(&serverURL),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	cfg.Icons = selectedIcons
	cfg.ServerURL = strings.TrimSpace(serverURL)

	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("lỗi lưu cấu hình: %w", err)
	}

	fmt.Println(ui.CardBoxSuccess.Render(fmt.Sprintf(
		"%s\n\n%s %s\n%s %s\n%s %s",
		ui.CardTitle.Render("✓ ĐÃ LƯU CẤU HÌNH THÀNH CÔNG"),
		ui.LabelStyle.Render("Bộ Icon:  "), ui.ValueStyle.Render(cfg.Icons),
		ui.LabelStyle.Render("Máy chủ:  "), ui.ValueStyle.Render(cfg.ServerURL),
		ui.LabelStyle.Render("File lưu: "), ui.LabelStyle.Render(configPath),
	)))

	return nil
}

func printConfigList(cfg *config.Config, configPath string, isJSON, isSimple bool) error {
	if isJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"icons":       cfg.Icons,
			"server_url":  cfg.ServerURL,
			"username":    cfg.Username,
			"config_path": configPath,
		})
	}

	if ui.IsSimpleMode(isSimple) {
		fmt.Println("Lich Configuration")
		fmt.Println("==================")
		fmt.Printf("icons:       %s\n", cfg.Icons)
		fmt.Printf("server_url:  %s\n", cfg.ServerURL)
		fmt.Printf("username:    %s\n", cfg.Username)
		fmt.Printf("config_path: %s\n", configPath)
		return nil
	}

	icons := ui.CurrentIcons()
	lines := []string{
		ui.SectionHeaderStyle.Render("CẤU HÌNH HỆ THỐNG LICH"),
		fmt.Sprintf(" • %s %s", ui.LabelStyle.Render("Bộ Icon:   "), ui.ValueStyle.Render(cfg.Icons)),
		fmt.Sprintf(" • %s %s", ui.LabelStyle.Render("Máy chủ:   "), ui.ValueStyle.Render(cfg.ServerURL)),
		fmt.Sprintf(" • %s %s", ui.LabelStyle.Render("Tài khoản: "), ui.ValueStyle.Render(cfg.Username)),
		fmt.Sprintf(" • %s %s", ui.LabelStyle.Render("File lưu:  "), ui.LabelStyle.Render(configPath)),
		"",
		fmt.Sprintf(" %s %s", ui.LabelStyle.Render("Mẫu hiển thị:"), fmt.Sprintf("%s Server | %s Database | %s Sync | %s Lịch", icons.Server, icons.Database, icons.Sync, icons.Calendar)),
	}

	fmt.Println(ui.ContainerCard.Render(strings.Join(lines, "\n")))
	return nil
}
