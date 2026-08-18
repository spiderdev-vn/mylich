package tui

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"lich-cli/internal/api"
	"lich-cli/internal/cache"
	"lich-cli/internal/syncer"
)

type FocusArea int

const (
	FocusCalendar FocusArea = iota
	FocusAgenda
)

type eventsLoadedMsg struct {
	Events map[string][]cache.LocalEvent
}

type syncFinishedMsg struct {
	Pushed int
	Pulled int
	Err    error
}

type crudFinishedMsg struct {
	Err error
}

type errMsg struct {
	err error
}

func (e errMsg) Error() string {
	return e.err.Error()
}

type Model struct {
	Client           *api.Client
	DB               *sql.DB
	CurrentMonth     time.Time
	SelectedDate     time.Time
	SelectedEventIdx int
	Focus            FocusArea
	ViewingEvent     *cache.LocalEvent
	Events           map[string][]cache.LocalEvent
	Loading          bool
	Syncing          bool
	SyncStatus       string
	Err              error
	Location         *time.Location
	Width            int
	Height           int
}

func NewModel(client *api.Client, db *sql.DB) Model {
	now := time.Now()
	loc := now.Location()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)

	return Model{
		Client:           client,
		DB:               db,
		CurrentMonth:     firstOfMonth,
		SelectedDate:     now,
		SelectedEventIdx: 0,
		Focus:            FocusCalendar,
		ViewingEvent:     nil,
		Events:           make(map[string][]cache.LocalEvent),
		Loading:          true,
		Syncing:          false,
		SyncStatus:       "Ready",
		Location:         loc,
		Width:            80,
		Height:           24,
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

func (m Model) getSelectedDayEvents() []cache.LocalEvent {
	dayKey := m.SelectedDate.Format("2006-01-02")
	return m.Events[dayKey]
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
		// Clamp SelectedEventIdx
		dayEvents := m.getSelectedDayEvents()
		if m.SelectedEventIdx >= len(dayEvents) {
			m.SelectedEventIdx = len(dayEvents) - 1
		}
		if m.SelectedEventIdx < 0 {
			m.SelectedEventIdx = 0
		}
		return m, nil

	case crudFinishedMsg:
		// Reload local events after interactive CRUD form closes
		return m, tea.Batch(m.loadLocalEventsCmd(), m.syncCmd())

	case syncFinishedMsg:
		m.Syncing = false
		if msg.Err != nil {
			m.SyncStatus = "Offline / Lỗi kết nối"
		} else {
			m.SyncStatus = "✓ Synced"
		}
		return m, m.loadLocalEventsCmd()

	case errMsg:
		m.Loading = false
		m.Err = msg.err
		return m, nil

	case tea.KeyMsg:
		// Nếu đang mở Modal chi tiết sự kiện
		if m.ViewingEvent != nil {
			switch msg.String() {
			case "esc", "enter", "q", "v", "space":
				m.ViewingEvent = nil
				return m, nil
			}
			return m, nil
		}

		dayEvents := m.getSelectedDayEvents()

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			if m.Focus == FocusCalendar {
				m.Focus = FocusAgenda
				if len(dayEvents) > 0 && m.SelectedEventIdx >= len(dayEvents) {
					m.SelectedEventIdx = 0
				}
			} else {
				m.Focus = FocusCalendar
			}
			return m, nil

		// Phím tắt tạo sự kiện mới (Create)
		case "a", "c", "+":
			exePath, err := os.Executable()
			if err != nil {
				exePath = "lich"
			}
			dateArg := m.SelectedDate.Format("2006-01-02")
			c := exec.Command(exePath, "add", "--date", dateArg)
			return m, tea.ExecProcess(c, func(err error) tea.Msg {
				return crudFinishedMsg{Err: err}
			})

		// Phím tắt chỉnh sửa sự kiện (Update)
		case "e":
			if len(dayEvents) > 0 && m.SelectedEventIdx < len(dayEvents) {
				selectedEv := dayEvents[m.SelectedEventIdx]
				exePath, err := os.Executable()
				if err != nil {
					exePath = "lich"
				}
				c := exec.Command(exePath, "edit", selectedEv.ID)
				return m, tea.ExecProcess(c, func(err error) tea.Msg {
					return crudFinishedMsg{Err: err}
				})
			}

		// Phím tắt xóa sự kiện (Delete)
		case "d", "x":
			if len(dayEvents) > 0 && m.SelectedEventIdx < len(dayEvents) {
				selectedEv := dayEvents[m.SelectedEventIdx]
				exePath, err := os.Executable()
				if err != nil {
					exePath = "lich"
				}
				c := exec.Command(exePath, "delete", selectedEv.ID)
				return m, tea.ExecProcess(c, func(err error) tea.Msg {
					return crudFinishedMsg{Err: err}
				})
			}

		// Xem chi tiết sự kiện (Read Details Modal)
		case "enter", "v":
			if len(dayEvents) > 0 && m.SelectedEventIdx < len(dayEvents) {
				ev := dayEvents[m.SelectedEventIdx]
				m.ViewingEvent = &ev
				return m, nil
			}

		case "left", "h":
			if m.Focus == FocusCalendar {
				m.SelectedDate = m.SelectedDate.AddDate(0, 0, -1)
				m.SelectedEventIdx = 0
				return m.syncMonthAndFetchIfNeeded()
			}

		case "right", "l":
			if m.Focus == FocusCalendar {
				m.SelectedDate = m.SelectedDate.AddDate(0, 0, 1)
				m.SelectedEventIdx = 0
				return m.syncMonthAndFetchIfNeeded()
			}

		case "up", "k":
			if m.Focus == FocusCalendar {
				m.SelectedDate = m.SelectedDate.AddDate(0, 0, -7)
				m.SelectedEventIdx = 0
				return m.syncMonthAndFetchIfNeeded()
			} else {
				if m.SelectedEventIdx > 0 {
					m.SelectedEventIdx--
				}
			}

		case "down", "j":
			if m.Focus == FocusCalendar {
				m.SelectedDate = m.SelectedDate.AddDate(0, 0, 7)
				m.SelectedEventIdx = 0
				return m.syncMonthAndFetchIfNeeded()
			} else {
				if m.SelectedEventIdx < len(dayEvents)-1 {
					m.SelectedEventIdx++
				}
			}

		case "p", "[":
			m.CurrentMonth = m.CurrentMonth.AddDate(0, -1, 0)
			m.SelectedDate = time.Date(m.CurrentMonth.Year(), m.CurrentMonth.Month(), 1, 0, 0, 0, 0, m.Location)
			m.SelectedEventIdx = 0
			return m, m.loadLocalEventsCmd()

		case "n", "]":
			m.CurrentMonth = m.CurrentMonth.AddDate(0, 1, 0)
			m.SelectedDate = time.Date(m.CurrentMonth.Year(), m.CurrentMonth.Month(), 1, 0, 0, 0, 0, m.Location)
			m.SelectedEventIdx = 0
			return m, m.loadLocalEventsCmd()

		case "t":
			now := time.Now().In(m.Location)
			m.SelectedDate = now
			m.SelectedEventIdx = 0
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
	// Nếu đang mở Modal xem chi tiết sự kiện
	if m.ViewingEvent != nil {
		return m.renderEventDetailModal(*m.ViewingEvent)
	}

	var sb strings.Builder

	// Top Title
	appTitle := titleStyle.Render("LICH CALENDAR (MỸ LÍCH)")
	sb.WriteString(appTitle)
	sb.WriteString("\n\n")

	// Left: Calendar Grid
	calView := RenderCalendarGrid(m.CurrentMonth.Year(), m.CurrentMonth.Month(), m.SelectedDate, m.Events, m.Location)
	calBox := calendarBoxStyle.Render(calView)

	// Right: Agenda for Selected Date
	selectedEvents := m.getSelectedDayEvents()
	agendaView := RenderAgenda(m.SelectedDate, selectedEvents, m.Location, m.SelectedEventIdx, m.Focus == FocusAgenda)
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
		"%s di chuyển  •  %s thêm  •  %s sửa  •  %s xóa  •  %s chi tiết  •  %s chuyển  •  %s thoát",
		helpKeyStyle.Render("←↓↑→/hjkl:"),
		helpKeyStyle.Render("a:"),
		helpKeyStyle.Render("e:"),
		helpKeyStyle.Render("d:"),
		helpKeyStyle.Render("Enter:"),
		helpKeyStyle.Render("Tab:"),
		helpKeyStyle.Render("q:"),
	)
	sb.WriteString(helpDescStyle.Render(helpKeys))

	return sb.String()
}

func (m Model) renderEventDetailModal(ev cache.LocalEvent) string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#74C0FC")).
		Padding(1, 2).
		Width(60)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#74C0FC"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7982A9"))
	valStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))

	timeStr := formatEventTime(ev.StartAt, ev.EndAt, m.Location)

	var lines []string
	lines = append(lines, titleStyle.Render("CHI TIẾT SỰ KIỆN"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("Tiêu đề:    "), valStyle.Render(ev.Title)))
	lines = append(lines, fmt.Sprintf("%s %s (%s)", labelStyle.Render("Thời gian:  "), valStyle.Render(timeStr), ev.StartAt[:10]))
	if ev.Location != "" {
		lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("Địa điểm:   "), valStyle.Render(ev.Location)))
	}
	if ev.Description != "" {
		lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("Ghi chú:    "), valStyle.Render(ev.Description)))
	}
	lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("Múi giờ:    "), valStyle.Render(ev.Timezone)))
	lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("Đồng bộ:    "), valStyle.Render(string(ev.SyncState))))
	lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("ID sự kiện: "), labelStyle.Render(ev.ID)))
	lines = append(lines, "")
	lines = append(lines, labelStyle.Render("Nhấn [Esc] hoặc [Enter] để đóng"))

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, modalStyle.Render(strings.Join(lines, "\n")))
}
