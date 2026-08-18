package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7D56F4") // Purple/Lavender
	secondaryColor = lipgloss.Color("#04B575") // Mint Green
	accentColor    = lipgloss.Color("#FF5F87") // Coral/Pink
	mutedColor     = lipgloss.Color("#626262") // Gray
	todayColor     = lipgloss.Color("#00D7D7") // Cyan
	selectedColor  = lipgloss.Color("#FFAF00") // Amber/Yellow
	bgDarkColor    = lipgloss.Color("#1A1A24")
	bgLightColor   = lipgloss.Color("#282A36")

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(0, 1)

	monthHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(primaryColor).
				Padding(0, 2).
				Align(lipgloss.Center)

	weekdayHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(mutedColor).
				Width(4).
				Align(lipgloss.Center)

	dayCellStyle = lipgloss.NewStyle().
			Width(4).
			Height(1).
			Align(lipgloss.Center)

	dayOtherMonthStyle = lipgloss.NewStyle().
				Width(4).
				Height(1).
				Foreground(mutedColor).
				Align(lipgloss.Center)

	dayTodayStyle = lipgloss.NewStyle().
			Width(4).
			Height(1).
			Bold(true).
			Foreground(todayColor).
			Align(lipgloss.Center)

	daySelectedStyle = lipgloss.NewStyle().
				Width(4).
				Height(1).
				Bold(true).
				Foreground(lipgloss.Color("#000000")).
				Background(selectedColor).
				Align(lipgloss.Center)

	dayWithEventsStyle = lipgloss.NewStyle().
				Width(4).
				Height(1).
				Bold(true).
				Underline(true).
				Foreground(secondaryColor).
				Align(lipgloss.Center)

	calendarBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(1, 2)

	agendaBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#44475A")).
			Padding(1, 2).
			Width(52)

	agendaHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(todayColor).
				MarginBottom(1)

	eventTimeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(secondaryColor)

	eventTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	eventLocStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	helpKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(mutedColor)
)
