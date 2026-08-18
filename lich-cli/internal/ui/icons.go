package ui

import (
	"strings"

	"lich-cli/internal/config"
)

type IconSet struct {
	Name     string
	Server   string
	Database string
	Sync     string
	Calendar string
	Clock    string
	User     string
	Location string
	Check    string
	Pending  string
	Failed   string
	Dot      string
	Bullet   string
	Arrow    string
	TagToday string
	Star     string
}

var (
	IconUnicode = IconSet{
		Name:     "unicode",
		Server:   "●",
		Database: "■",
		Sync:     "↻",
		Calendar: "■",
		Clock:    "•",
		User:     "•",
		Location: "»",
		Check:    "✓",
		Pending:  "↻",
		Failed:   "⚠",
		Dot:      "●",
		Bullet:   "•",
		Arrow:    "▶",
		TagToday: "[HÔM NAY]",
		Star:     "★",
	}

	IconNerd = IconSet{
		Name:     "nerd",
		Server:   "󰒋",
		Database: "",
		Sync:     "󰑓",
		Calendar: "󰃭",
		Clock:    "󰥔",
		User:     "",
		Location: "",
		Check:    "✔",
		Pending:  "󰑓",
		Failed:   "✖",
		Dot:      "●",
		Bullet:   "",
		Arrow:    "❯",
		TagToday: "[HÔM NAY]",
		Star:     "󰓎",
	}

	IconASCII = IconSet{
		Name:     "ascii",
		Server:   "[SERVER]",
		Database: "[DB]",
		Sync:     "[SYNC]",
		Calendar: "[CAL]",
		Clock:    "[TIME]",
		User:     "[USER]",
		Location: "[LOC]",
		Check:    "[v]",
		Pending:  "[~]",
		Failed:   "[!]",
		Dot:      "[*]",
		Bullet:   "-",
		Arrow:    ">",
		TagToday: "[TODAY]",
		Star:     "*",
	}

	IconEmoji = IconSet{
		Name:     "emoji",
		Server:   "🌐",
		Database: "💾",
		Sync:     "🔄",
		Calendar: "📅",
		Clock:    "⏰",
		User:     "👤",
		Location: "📍",
		Check:    "✓",
		Pending:  "↻",
		Failed:   "❌",
		Dot:      "🟢",
		Bullet:   "•",
		Arrow:    "▶",
		TagToday: "⭐ [HÔM NAY]",
		Star:     "⭐",
	}
)

func GetIconSet(style string) IconSet {
	switch strings.ToLower(strings.TrimSpace(style)) {
	case "nerd", "nerdfont", "nf":
		return IconNerd
	case "ascii", "plain", "simple":
		return IconASCII
	case "emoji":
		return IconEmoji
	case "unicode":
		return IconUnicode
	default:
		return IconUnicode
	}
}

// CurrentIcons returns the active icon set based on user config and terminal environment
func CurrentIcons() IconSet {
	if IsSimpleMode(false) {
		return IconASCII
	}

	cfg, err := config.LoadConfig()
	if err != nil || cfg.Icons == "" {
		return IconUnicode
	}

	return GetIconSet(cfg.Icons)
}
