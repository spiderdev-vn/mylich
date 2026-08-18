package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultServerURL  = "http://127.0.0.1:3000"
	DefaultIcons      = "unicode"
	DefaultAgendaMode = "list"
)

type Config struct {
	ServerURL  string `json:"server_url"`
	Token      string `json:"token"`
	Username   string `json:"username,omitempty"`
	Icons      string `json:"icons,omitempty"`       // unicode | nerd | ascii | emoji
	AgendaMode string `json:"agenda_mode,omitempty"` // list | timeline | gantt | ascii
}

var ValidIconStyles = map[string]bool{
	"unicode": true,
	"nerd":    true,
	"ascii":   true,
	"emoji":   true,
}

var ValidAgendaModes = map[string]bool{
	"list":     true,
	"timeline": true,
	"gantt":    true,
	"ascii":    true,
}

func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("không thể lấy thư mục config người dùng: %w", err)
	}
	return filepath.Join(configDir, "lich", "config.json"), nil
}

func LoadConfig() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return &Config{ServerURL: DefaultServerURL, Icons: DefaultIcons, AgendaMode: DefaultAgendaMode}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{ServerURL: DefaultServerURL, Icons: DefaultIcons, AgendaMode: DefaultAgendaMode}, nil
		}
		return nil, fmt.Errorf("lỗi đọc file cấu hình: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("lỗi phân tích file cấu hình JSON: %w", err)
	}

	if cfg.ServerURL == "" {
		cfg.ServerURL = DefaultServerURL
	}
	if cfg.Icons == "" {
		cfg.Icons = DefaultIcons
	}
	if cfg.AgendaMode == "" || !ValidAgendaModes[cfg.AgendaMode] {
		cfg.AgendaMode = DefaultAgendaMode
	}

	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("không thể tạo thư mục config: %w", err)
	}

	if cfg.Icons == "" {
		cfg.Icons = DefaultIcons
	}
	if cfg.AgendaMode == "" || !ValidAgendaModes[cfg.AgendaMode] {
		cfg.AgendaMode = DefaultAgendaMode
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("lỗi mã hóa cấu hình JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("lỗi ghi file cấu hình: %w", err)
	}

	return nil
}

func (c *Config) Get(key string) (string, error) {
	switch strings.ToLower(key) {
	case "server_url", "server", "url":
		return c.ServerURL, nil
	case "token":
		return c.Token, nil
	case "username", "user":
		return c.Username, nil
	case "icons", "icon", "theme":
		if c.Icons == "" {
			return DefaultIcons, nil
		}
		return c.Icons, nil
	case "agenda_mode", "agenda", "mode":
		if c.AgendaMode == "" {
			return DefaultAgendaMode, nil
		}
		return c.AgendaMode, nil
	default:
		return "", fmt.Errorf("khóa cấu hình không tồn tại '%s'", key)
	}
}

func (c *Config) Set(key, value string) error {
	switch strings.ToLower(key) {
	case "server_url", "server", "url":
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("server_url không được để trống")
		}
		c.ServerURL = strings.TrimSpace(value)
	case "token":
		c.Token = strings.TrimSpace(value)
	case "username", "user":
		c.Username = strings.TrimSpace(value)
	case "icons", "icon", "theme":
		val := strings.ToLower(strings.TrimSpace(value))
		if !ValidIconStyles[val] {
			return fmt.Errorf("bộ icon '%s' không hợp lệ. Các tùy chọn: unicode, nerd, ascii, emoji", value)
		}
		c.Icons = val
	case "agenda_mode", "agenda", "mode":
		val := strings.ToLower(strings.TrimSpace(value))
		if !ValidAgendaModes[val] {
			return fmt.Errorf("chế độ agenda '%s' không hợp lệ. Các tùy chọn: list, gantt, ascii", value)
		}
		c.AgendaMode = val
	default:
		return fmt.Errorf("khóa cấu hình không hợp lệ '%s'. Các khóa hỗ trợ: icons, agenda_mode, server_url, username", key)
	}
	return nil
}
