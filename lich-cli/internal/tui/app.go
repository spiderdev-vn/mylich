package tui

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"lich-cli/internal/api"
	"lich-cli/internal/cache"
	"lich-cli/internal/syncer"
)

type eventsLoadedMsg struct {
	Events map[string][]cache.LocalEvent
}

type syncFinishedMsg struct {
	Pushed int
	Pulled int
	Err    error
}

type errMsg struct {
	err error
}

func (e errMsg) Error() string {
	return e.err.Error()
}

type Model struct {
	Client       *api.Client
	DB           *sql.DB
	CurrentMonth time.Time
	SelectedDate time.Time
	Events       map[string][]cache.LocalEvent
	Loading      bool
	Syncing      bool
	SyncStatus   string
	Err          error
	Location     *time.Location
	Width        int
	Height       int
}

func NewModel(client *api.Client, db *sql.DB) Model {
	now := time.Now()
	loc := now.Location()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)

	return Model{
		Client:       client,
		DB:           db,
		CurrentMonth: firstOfMonth,
		SelectedDate: now,
		Events:       make(map[string][]cache.LocalEvent),
		Loading:      true,
		Syncing:      false,
		SyncStatus:   "Ready",
		Location:     loc,
		Width:        80,
		Height:       24,
	}
}

func (m Model) loadLocalEventsCmd() tea.Cmd {
	return func() tea.Msg {
		if m.DB == nil {
			return eventsLoadedMsg{Events: make(map[string][]cache.LocalEvent)}
		}

		from := m.CurrentMonth.AddDate(0, -1, 0)
		to := m.CurrentMonth.AddDate(0, 2, 0)

		events, err := cache.GetEventsInRange(
			m.DB,
			from.UTC().Format(time.RFC3339),
			to.UTC().Format(time.RFC3339),
			"",
		)
		if err != nil {
			return errMsg{err: err}
		}

		eventsMap := make(map[string][]cache.LocalEvent)
		for _, ev := range events {
			t, err := time.Parse(time.RFC3339, ev.StartAt)
			if err == nil {
				key := t.In(m.Location).Format("2006-01-02")
				eventsMap[key] = append(eventsMap[key], ev)
			}
		}

		return eventsLoadedMsg{Events: eventsMap}
	}
}

func (m Model) syncCmd() tea.Cmd {
	return func() tea.Msg {
		if m.DB == nil || m.Client == nil {
			return syncFinishedMsg{Pushed: 0, Pulled: 0, Err: nil}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		engine := syncer.NewSyncEngine(m.DB, m.Client)
		pushed, pulled, err := engine.Sync(ctx)

		return syncFinishedMsg{Pushed: pushed, Pulled: pulled, Err: err}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadLocalEventsCmd(),
		m.syncCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case eventsLoadedMsg:
		m.Loading = false
		m.Events = msg.Events
		m.Err = nil
		return m, nil

	case syncFinishedMsg:
		m.Syncing = false
		if msg.Err != nil {
			m.SyncStatus = "Offline / Lỗi kết nối"
		} else {
			m.SyncStatus = "✓ Synced"
		}
		// Reload local events after sync finishes to reflect any pulled changes
		return m, m.loadLocalEventsCmd()

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
			return m, m.loadLocalEventsCmd()

		case "n", "]":
			m.CurrentMonth = m.CurrentMonth.AddDate(0, 1, 0)
			m.SelectedDate = time.Date(m.CurrentMonth.Year(), m.CurrentMonth.Month(), 1, 0, 0, 0, 0, m.Location)
			return m, m.loadLocalEventsCmd()

		case "t":
			now := time.Now().In(m.Location)
			m.SelectedDate = now
			m.CurrentMonth = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, m.Location)
			return m, m.loadLocalEventsCmd()

		case "r":
			m.Syncing = true
			m.SyncStatus = "● Đang đồng bộ..."
			return m, tea.Batch(m.loadLocalEventsCmd(), m.syncCmd())
		}
	}

	return m, nil
}

func (m Model) syncMonthAndFetchIfNeeded() (Model, tea.Cmd) {
	if m.SelectedDate.Month() != m.CurrentMonth.Month() || m.SelectedDate.Year() != m.CurrentMonth.Year() {
		m.CurrentMonth = time.Date(m.SelectedDate.Year(), m.SelectedDate.Month(), 1, 0, 0, 0, 0, m.Location)
		return m, m.loadLocalEventsCmd()
	}
	return m, nil
}

func (m Model) View() string {
	var sb strings.Builder

	// Top Title
	appTitle := titleStyle.Render("⚡ LICH CALENDAR (MỸ LÍCH)")
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
	statusText := m.SyncStatus
	if m.Loading {
		statusText = "Đang tải dữ liệu local..."
	} else if m.Syncing {
		statusText = "● Đang đồng bộ ngầm..."
	} else if m.Err != nil {
		statusText = fmt.Sprintf("Lỗi: %v", m.Err)
	}
	sb.WriteString(statusBarStyle.Render(fmt.Sprintf("Trạng thái: %s", statusText)))
	sb.WriteString("\n")

	// Footer Help
	helpKeys := fmt.Sprintf(
		"%s di chuyển  •  %s chuyển tháng  •  %s hôm nay  •  %s đồng bộ (refresh)  •  %s thoát",
		helpKeyStyle.Render("←↓↑→ / hjkl:"),
		helpKeyStyle.Render("p/n:"),
		helpKeyStyle.Render("t:"),
		helpKeyStyle.Render("r:"),
		helpKeyStyle.Render("q:"),
	)
	sb.WriteString(helpDescStyle.Render(helpKeys))

	return sb.String()
}
