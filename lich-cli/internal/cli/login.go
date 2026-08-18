package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"lich-cli/internal/api"
	"lich-cli/internal/config"
)

func RunLogin(args []string) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	serverFlag := fs.String("server", "", "Lich server URL (default from config or http://127.0.0.1:3000)")
	userFlag := fs.String("username", "", "Username")
	passFlag := fs.String("password", "", "Password")
	registerFlag := fs.Bool("register", false, "Register a new user instead of logging in")

	if err := fs.Parse(args); err != nil {
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

	reader := bufio.NewReader(os.Stdin)

	username := *userFlag
	if username == "" {
		fmt.Printf("Username: ")
		input, _ := reader.ReadString('\n')
		username = strings.TrimSpace(input)
	}

	password := *passFlag
	if password == "" {
		fmt.Printf("Password: ")
		input, _ := reader.ReadString('\n')
		password = strings.TrimSpace(input)
	}

	if username == "" || password == "" {
		return fmt.Errorf("username and password cannot be empty")
	}

	client := api.NewClient(serverURL, "")
	ctx := context.Background()

	var authRes *api.AuthResponse
	if *registerFlag {
		fmt.Printf("Registering user '%s' on %s...\n", username, serverURL)
		authRes, err = client.Register(ctx, api.RegisterRequest{
			Username: username,
			Password: password,
		})
	} else {
		fmt.Printf("Logging in as '%s' to %s...\n", username, serverURL)
		authRes, err = client.Login(ctx, api.LoginRequest{
			Username: username,
			Password: password,
		})
	}

	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	cfg.ServerURL = serverURL
	cfg.Token = authRes.Token
	cfg.Username = authRes.User.Username

	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Successfully authenticated as '%s'\n", authRes.User.Username)
	return nil
}
