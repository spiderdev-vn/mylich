package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"lich-cli/internal/api"
)

type eventsFetchedMsg struct {
	Events map[string][]api.Event
}

type errMsg struct {
	err error
}

func (e errMsg) Error() string {
	return e.err.Error()
}

type Model struct {
	Client       *api.Client
	CurrentMonth time.Time
	SelectedDate time.Time
	Events       map[string][]api.Event
	Loading      bool
	Err          error
	Location     *time.Location
	Width        int
	Height       int
}

func NewModel(client *api.Client) Model {
	now := time.Now()
	loc := now.Location()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)

	return Model{
		Client:       client,
		CurrentMonth: firstOfMonth,
		SelectedDate: now,
		Events:       make(map[string][]api.Event),
		Loading:      true,
		Location:     loc,
		Width:        80,
		Height:       24,
	}
}

func (m Model) fetchEventsCmd() tea.Cmd {
	return func() tea.Msg {
		if m.Client == nil {
			return eventsFetchedMsg{Events: make(map[string][]api.Event)}
		}

		// Fetch for current month range (-1 month to +2 months for smooth browsing)
		from := m.CurrentMonth.AddDate(0, -1, 0)
		to := m.CurrentMonth.AddDate(0, 2, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		events, err := m.Client.ListEvents(ctx, api.EventFilter{
			From: from.Format(time.RFC3339),
			To:   to.Format(time.RFC3339),
		})
		if err != nil {
			return errMsg{err: err}
		}

		eventsMap := make(map[string][]api.Event)
		for _, ev := range events {
			t, err := time.Parse(time.RFC3339, ev.StartAt)
			if err == nil {
				key := t.In(m.Location).Format("2006-01-02")
				eventsMap[key] = append(eventsMap[key], ev)
			}
		}

		return eventsFetchedMsg{Events: eventsMap}
	}
}

func (m Model) Init() tea.Cmd {
	return m.fetchEventsCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case eventsFetchedMsg:
		m.Loading = false
		m.Events = msg.Events
		m.Err = nil
		return m, nil

	case errMsg:
		m.Loading = false
		m.Err = msg.err
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "left", "h":
			m.SelectedDate = m.SelectedDate.AddDate(0, 0, -1)
			return m.syncMonthAndFetchIfNeeded()

		case "right", "l":
			m.SelectedDate = m.SelectedDate.AddDate(0, 0, 1)
			return m.syncMonthAndFetchIfNeeded()

		case "up", "k":
			m.SelectedDate = m.SelectedDate.AddDate(0, 0, -7)
			return m.syncMonthAndFetchIfNeeded()

		case "down", "j":
			m.SelectedDate = m.SelectedDate.AddDate(0, 0, 7)
			return m.syncMonthAndFetchIfNeeded()

		case "p", "[":
			m.CurrentMonth = m.CurrentMonth.AddDate(0, -1, 0)
			m.SelectedDate = time.Date(m.CurrentMonth.Year(), m.CurrentMonth.Month(), 1, 0, 0, 0, 0, m.Location)
			m.Loading = true
			return m, m.fetchEventsCmd()

		case "n", "]":
			m.CurrentMonth = m.CurrentMonth.AddDate(0, 1, 0)
			m.SelectedDate = time.Date(m.CurrentMonth.Year(), m.CurrentMonth.Month(), 1, 0, 0, 0, 0, m.Location)
			m.Loading = true
			return m, m.fetchEventsCmd()

		case "t":
			now := time.Now().In(m.Location)
			m.SelectedDate = now
			m.CurrentMonth = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, m.Location)
			m.Loading = true
			return m, m.fetchEventsCmd()

		case "r":
			m.Loading = true
			return m, m.fetchEventsCmd()
		}
	}

	return m, nil
}

func (m Model) syncMonthAndFetchIfNeeded() (Model, tea.Cmd) {
	if m.SelectedDate.Month() != m.CurrentMonth.Month() || m.SelectedDate.Year() != m.CurrentMonth.Year() {
		m.CurrentMonth = time.Date(m.SelectedDate.Year(), m.SelectedDate.Month(), 1, 0, 0, 0, 0, m.Location)
		m.Loading = true
		return m, m.fetchEventsCmd()
	}
	return m, nil
}

func (m Model) View() string {
	var sb strings.Builder

	// Top Title
	appTitle := titleStyle.Render("⚡ LICH CALENDAR")
	sb.WriteString(appTitle)
	sb.WriteString("\n\n")

	// Left: Calendar Grid
	calView := RenderCalendarGrid(m.CurrentMonth.Year(), m.CurrentMonth.Month(), m.SelectedDate, m.Events, m.Location)
	calBox := calendarBoxStyle.Render(calView)

	// Right: Agenda for Selected Date
	selectedDayKey := m.SelectedDate.Format("2006-01-02")
	selectedEvents := m.Events[selectedDayKey]
	agendaView := RenderAgenda(m.SelectedDate, selectedEvents, m.Location)
	agendaBox := agendaBoxStyle.Render(agendaView)

	// Join side-by-side or stacked based on width
	var mainLayout string
	if m.Width > 75 {
		mainLayout = lipgloss.JoinHorizontal(lipgloss.Top, calBox, "  ", agendaBox)
	} else {
		mainLayout = lipgloss.JoinVertical(lipgloss.Left, calBox, "\n", agendaBox)
	}
	sb.WriteString(mainLayout)
	sb.WriteString("\n")

	// Status line
	statusText := "Ready"
	if m.Loading {
		statusText = "Syncing events..."
	} else if m.Err != nil {
		statusText = fmt.Sprintf("Error: %v", m.Err)
	}
	sb.WriteString(statusBarStyle.Render(fmt.Sprintf("Status: %s", statusText)))
	sb.WriteString("\n")

	// Footer Help
	helpKeys := fmt.Sprintf(
		"%s navigate  •  %s month  •  %s today  •  %s refresh  •  %s quit",
		helpKeyStyle.Render("←↓↑→ / hjkl:"),
		helpKeyStyle.Render("p/n:"),
		helpKeyStyle.Render("t:"),
		helpKeyStyle.Render("r:"),
		helpKeyStyle.Render("q:"),
	)
	sb.WriteString(helpDescStyle.Render(helpKeys))

	return sb.String()
}
