package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Palette: Soft, elegant Catppuccin/Nord inspired colors
	ColorPrimary   = lipgloss.Color("#8075FF") // Soft Indigo/Purple
	ColorSecondary = lipgloss.Color("#74C0FC") // Soft Blue/Cyan
	ColorSuccess   = lipgloss.Color("#63E6BE") // Soft Mint/Green
	ColorWarning   = lipgloss.Color("#FCD34D") // Soft Amber
	ColorError     = lipgloss.Color("#FFA8A8") // Soft Rose/Red
	ColorMuted     = lipgloss.Color("#7982A9") // Soft Muted Slate
	ColorBorder    = lipgloss.Color("#4B526D") // Muted Border
	ColorHighlight = lipgloss.Color("#F8F9FA") // Crisp White

	// Typography & Styles
	TitleBanner = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 1)

	SectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary)

	CardTitle = SectionHeaderStyle
	SubTitleStyle = SectionHeaderStyle
	CardBox = ContainerCard

	HeaderDateStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHighlight).
			Background(lipgloss.Color("#2A2E3F")).
			Padding(0, 1)

	// Single unified container box that adapts cleanly to terminal width
	ContainerCard = lipgloss.NewStyle().
			Width(72).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	CardBoxSuccess = lipgloss.NewStyle().
			Width(72).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSuccess).
			Padding(0, 1)

	CardBoxError = lipgloss.NewStyle().
			Width(72).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorError).
			Padding(0, 1)

	LabelStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	ValueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHighlight)

	TimePill = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			Background(lipgloss.Color("#24273A")).
			Padding(0, 1)

	EventTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHighlight)

	EventLocationStyle = lipgloss.NewStyle().
				Italic(true).
				Foreground(lipgloss.Color("#939AB5"))

	EventDescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	DividerStyle = lipgloss.NewStyle().
			Foreground(ColorBorder)

	RequiredAsterisk = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FF6B6B")).
				Render("*")

	// Status Badges
	BadgeOnline = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Render("● Trực tuyến (Online)")

	BadgeOffline = lipgloss.NewStyle().
			Foreground(ColorError).
			Render("○ Ngoại tuyến (Offline)")

	BadgeSynced = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Render("✓ Đã đồng bộ")

	BadgePending = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Render("↻ Đang chờ đồng bộ")

	BadgeFailed = lipgloss.NewStyle().
			Foreground(ColorError).
			Render("⚠ Đồng bộ thất bại")
)

func RenderBadgeOnline(icon string) string {
	return lipgloss.NewStyle().Foreground(ColorSuccess).Render(icon + " Trực tuyến (Online)")
}

func RenderBadgeOffline(icon string) string {
	return lipgloss.NewStyle().Foreground(ColorError).Render(icon + " Ngoại tuyến (Offline)")
}

func RenderBadgeSynced(icon string) string {
	return lipgloss.NewStyle().Foreground(ColorSuccess).Render(icon + " Đã đồng bộ")
}

func RenderBadgePending(icon string) string {
	return lipgloss.NewStyle().Foreground(ColorWarning).Render(icon + " Đang chờ đồng bộ")
}

func RenderBadgeFailed(icon string) string {
	return lipgloss.NewStyle().Foreground(ColorError).Render(icon + " Đồng bộ thất bại")
}

// IsSimpleMode checks if the user requested plain ASCII output or if terminal is explicitly dumb
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
	return false
}
