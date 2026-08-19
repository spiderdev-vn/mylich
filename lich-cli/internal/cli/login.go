package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/api"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/config"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/ui"
)

func RunLogin(args []string) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	serverFlag := fs.String("server", "", fmt.Sprintf("URL máy chủ Lich (mặc định: %s)", config.DefaultServerURL))
	userFlag := fs.String("username", "", "Tên đăng nhập")
	fs.StringVar(userFlag, "u", "", "Tên đăng nhập (viết tắt)")
	passFlag := fs.String("password", "", "Mật khẩu")
	fs.StringVar(passFlag, "p", "", "Mật khẩu (viết tắt)")
	registerFlag := fs.Bool("register", false, "Đăng ký tài khoản mới thay vì đăng nhập")
	simpleFlag := fs.Bool("simple", false, "Hiển thị dạng text ASCII đơn giản")
	fs.BoolVar(simpleFlag, "s", false, "Hiển thị dạng text ASCII đơn giản (viết tắt)")

	fs.Usage = func() {
		if ui.IsSimpleMode(*simpleFlag) {
			fmt.Println("Sử dụng: lich login [flags]")
			fmt.Println()
			fmt.Println("Mô tả:")
			fmt.Println("  Đăng nhập hoặc đăng ký tài khoản mới. Nếu để trống cờ, sẽ mở Form tương tác Huh.")
			fmt.Println()
			fmt.Println("Tùy chọn:")
			fmt.Printf("  --server <url>        URL máy chủ (mặc định: %s)\n", config.DefaultServerURL)
			fmt.Println("  --username, -u <user> Tên đăng nhập")
			fmt.Println("  --password, -p <pass> Mật khẩu")
			fmt.Println("  --register            Đăng ký tài khoản mới")
			fmt.Println("  --simple, -s          Hiển thị dạng text ASCII đơn giản")
			return
		}

		banner := ui.TitleBanner.Render(" ĐĂNG NHẬP / ĐĂNG KÝ (LICH LOGIN) ")
		fmt.Println(banner)

		helpContent := fmt.Sprintf(`%s
  %s             %s
  %s %s
  %s    %s

%s
  %s     %s
  %s       %s
  %s       %s
  %s         %s
  %s        %s`,
			ui.CardTitle.Render("CÚ PHÁP:"),
			ui.ValueStyle.Render("lich login"), ui.LabelStyle.Render("Mở Form đăng nhập/đăng ký tương tác trực quan"),
			ui.ValueStyle.Render("lich login -u alice -p 123"), ui.LabelStyle.Render("Đăng nhập nhanh qua dòng lệnh"),
			ui.ValueStyle.Render("lich login --register"), ui.LabelStyle.Render("Đăng ký tài khoản người dùng mới"),
			ui.CardTitle.Render("TÙY CHỌN & CỜ:"),
			ui.ValueStyle.Render("--server <url>"), ui.LabelStyle.Render("URL máy chủ (mặc định: "+config.DefaultServerURL+")"),
			ui.ValueStyle.Render("--username, -u"), ui.LabelStyle.Render("Tên tài khoản người dùng"),
			ui.ValueStyle.Render("--password, -p"), ui.LabelStyle.Render("Mật khẩu tài khoản"),
			ui.ValueStyle.Render("--register"), ui.LabelStyle.Render("Chế độ đăng ký tài khoản mới"),
			ui.ValueStyle.Render("--simple, -s"), ui.LabelStyle.Render("Hiển thị dạng text ASCII đơn giản"),
		)
		fmt.Println(ui.CardBox.Width(78).Render(helpContent))
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = &config.Config{ServerURL: config.DefaultServerURL}
	}

	serverURL := *serverFlag
	if serverURL == "" {
		serverURL = cfg.ServerURL
	}
	if serverURL == "" {
		serverURL = config.DefaultServerURL
	}

	username := *userFlag
	password := *passFlag
	isRegister := *registerFlag

	// Nếu chưa truyền username/password qua CLI và ở chế độ Interactive, mở form Huh
	if (username == "" || password == "") && !ui.IsSimpleMode(*simpleFlag) {
		action := "login"
		if isRegister {
			action = "register"
		}

		reqStar := ui.RequiredAsterisk

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Chọn thao tác xác thực").
					Options(
						huh.NewOption("Đăng nhập tài khoản có sẵn", "login"),
						huh.NewOption("Đăng ký tài khoản mới", "register"),
					).
					Value(&action),

				huh.NewInput().
					Title("Địa chỉ Máy chủ (Server URL)").
					Value(&serverURL),

				huh.NewInput().
					Title(fmt.Sprintf("Tên đăng nhập (Username) %s", reqStar)).
					Placeholder("alice").
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("tên đăng nhập không được để trống")
						}
						return nil
					}).
					Value(&username),

				huh.NewInput().
					Title(fmt.Sprintf("Mật khẩu (Password) %s", reqStar)).
					EchoMode(huh.EchoModePassword).
					Validate(func(s string) error {
						if len(s) < 6 {
							return fmt.Errorf("mật khẩu phải có ít nhất 6 ký tự")
						}
						return nil
					}).
					Value(&password),
			),
		).WithKeyMap(ui.DefaultFormKeyMap())

		if err := form.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil
			}
			return err
		}

		isRegister = (action == "register")
	}

	if username == "" || password == "" {
		return fmt.Errorf("tên đăng nhập và mật khẩu không được để trống")
	}

	client := api.NewClient(serverURL, "")
	ctx := context.Background()

	var authRes *api.AuthResponse
	if isRegister {
		authRes, err = client.Register(ctx, api.RegisterRequest{
			Username: username,
			Password: password,
		})
	} else {
		authRes, err = client.Login(ctx, api.LoginRequest{
			Username: username,
			Password: password,
		})
	}

	if err != nil {
		return fmt.Errorf("xác thực thất bại: %w", err)
	}

	cfg.ServerURL = serverURL
	cfg.Token = authRes.Token
	cfg.Username = authRes.User.Username

	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("lỗi lưu file cấu hình: %w", err)
	}

	if ui.IsSimpleMode(*simpleFlag) {
		fmt.Printf("✓ Đăng nhập thành công với tài khoản '%s' trên máy chủ %s\n", authRes.User.Username, serverURL)
	} else {
		successCard := ui.CardBoxSuccess.Render(fmt.Sprintf(
			"%s\n\n%s %s\n%s %s\n%s %s",
			ui.CardTitle.Render("✓ XÁC THỰC THÀNH CÔNG"),
			ui.LabelStyle.Render("Tài khoản:"), ui.ValueStyle.Render(authRes.User.Username),
			ui.LabelStyle.Render("Máy chủ:  "), ui.ValueStyle.Render(serverURL),
			ui.LabelStyle.Render("Trạng thái:"), ui.BadgeOnline,
		))
		fmt.Println(successCard)
	}

	return nil
}
