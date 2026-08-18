package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

var (
	// Colors
	ColorPrimary   = lipgloss.Color("#7D56F4") // Charm Purple
	ColorSecondary = lipgloss.Color("#00D2FF") // Neon Cyan
	ColorSuccess   = lipgloss.Color("#04B575") // Emerald Green
	ColorWarning   = lipgloss.Color("#FFAF00") // Amber
	ColorError     = lipgloss.Color("#FF5F87") // Coral Red
	ColorMuted     = lipgloss.Color("#6C7086") // Slate Gray
	ColorHighlight = lipgloss.Color("#FFFFFF") // Bright White

	// Typography & Styles
	TitleBanner = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 1).
			MarginBottom(1)

	SubTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

	HeaderDateStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#313244")).
			Padding(0, 1)

	CardBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Margin(0, 1, 1, 0)

	CardBoxSecondary = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorSecondary).
				Padding(0, 1).
				Margin(0, 1, 1, 0)

	CardBoxSuccess = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSuccess).
			Padding(0, 1).
			Margin(0, 0, 1, 0)

	CardBoxError = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorError).
			Padding(0, 1).
			Margin(0, 0, 1, 0)

	CardTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			MarginBottom(0)

	LabelStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	ValueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHighlight)

	TimePill = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			Background(lipgloss.Color("#1E1E2E")).
			Padding(0, 1)

	EventTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHighlight)

	EventLocationStyle = lipgloss.NewStyle().
				Italic(true).
				Foreground(lipgloss.Color("#A6ADC8"))

	EventDescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Status Badges
	BadgeOnline = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSuccess).
			Render("● TRỰC TUYẾN (ONLINE)")

	BadgeOffline = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorError).
			Render("○ NGOẠI TUYẾN (OFFLINE)")

	BadgeSynced = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSuccess).
			Render("✓ ĐÃ ĐỒNG BỘ")

	BadgePending = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWarning).
			Render("↻ ĐANG CHỜ ĐỒNG BỘ")

	BadgeFailed = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorError).
			Render("⚠ ĐỒNG BỘ THẤT BẠI")
)

// IsSimpleMode checks if the user requested plain ASCII output or if terminal doesn't support color/TTY
func IsSimpleMode(simpleFlag bool) bool {
	if simpleFlag {
		return true
	}
	if os.Getenv("NO_COLOR") != "" {
		return true
	}
	if os.Getenv("TERM") == "dumb" {
		return true
	}
	// If stdout is redirected / piped and not a terminal, default to simple
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return true
	}
	return false
}
