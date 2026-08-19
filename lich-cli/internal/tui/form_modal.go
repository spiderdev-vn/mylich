package tui

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spiderdev-vn/mylich/lich-cli/internal/api"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/cache"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/config"
	"github.com/spiderdev-vn/mylich/lich-cli/internal/syncer"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FormMode int

const (
	FormModeNone FormMode = iota
	FormModeAdd
	FormModeEdit
	FormModeDelete
	FormModeView
)

type EventFormModal struct {
	Mode        FormMode
	EventID     string
	CalendarID  string
	Inputs      []textinput.Model
	FocusIndex  int
	ErrorMsg    string
	DeleteEvent *cache.LocalEvent
}

func newTextInput(placeholder string, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Width = width
	ti.Prompt = "  "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#74C0FC"))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))
	return ti
}

func NewAddFormModal(selectedDate time.Time) EventFormModal {
	inputs := make([]textinput.Model, 7)

	inputs[0] = newTextInput("Tiêu đề sự kiện *", 40)
	inputs[0].Focus()
	inputs[0].Prompt = "▶ "

	inputs[1] = newTextInput("dd/mm/yyyy, dd-mm-yy, today...", 40)
	inputs[1].SetValue(selectedDate.Format("02/01/2006"))

	inputs[2] = newTextInput("10:00, 10am, 22:30...", 40)
	inputs[2].SetValue("10:00")

	inputs[3] = newTextInput("11:00, 11am, 23:30...", 40)
	inputs[3].SetValue("11:00")

	// [4] Ngày kết thúc — mặc định cùng ngày bắt đầu
	inputs[4] = newTextInput("Mặc định: cùng ngày bắt đầu", 40)
	inputs[4].SetValue(selectedDate.Format("02/01/2006"))

	inputs[5] = newTextInput("Địa điểm (tùy chọn)", 40)
	inputs[6] = newTextInput("Ghi chú mô tả (tùy chọn)", 40)

	return EventFormModal{
		Mode:       FormModeAdd,
		Inputs:     inputs,
		FocusIndex: 0,
	}
}

func NewEditFormModal(ev cache.LocalEvent, loc *time.Location) EventFormModal {
	inputs := make([]textinput.Model, 7)

	tStart, _ := time.Parse(time.RFC3339, ev.StartAt)
	tEnd, _ := time.Parse(time.RFC3339, ev.EndAt)
	startLocal := tStart.In(loc)
	endLocal := tEnd.In(loc)

	inputs[0] = newTextInput("Tiêu đề sự kiện *", 40)
	inputs[0].SetValue(ev.Title)
	inputs[0].Focus()
	inputs[0].Prompt = "▶ "

	inputs[1] = newTextInput("dd/mm/yyyy, dd-mm-yy, today...", 40)
	inputs[1].SetValue(startLocal.Format("02/01/2006"))

	inputs[2] = newTextInput("10:00, 10am, 22:30...", 40)
	inputs[2].SetValue(startLocal.Format("15:04"))

	inputs[3] = newTextInput("11:00, 11am, 23:30...", 40)
	inputs[3].SetValue(endLocal.Format("15:04"))

	// [4] Ngày kết thúc — mặc định cùng ngày bắt đầu
	inputs[4] = newTextInput("Mặc định: cùng ngày bắt đầu", 40)
	inputs[4].SetValue(endLocal.Format("02/01/2006"))

	inputs[5] = newTextInput("Địa điểm (tùy chọn)", 40)
	inputs[5].SetValue(ev.Location)

	inputs[6] = newTextInput("Ghi chú mô tả (tùy chọn)", 40)
	inputs[6].SetValue(ev.Description)

	return EventFormModal{
		Mode:       FormModeEdit,
		EventID:    ev.ID,
		CalendarID: ev.CalendarID,
		Inputs:     inputs,
		FocusIndex: 0,
	}
}

func NewDeleteConfirmModal(ev cache.LocalEvent) EventFormModal {
	return EventFormModal{
		Mode:        FormModeDelete,
		EventID:     ev.ID,
		DeleteEvent: &ev,
	}
}

func (f *EventFormModal) Update(msg tea.Msg) (*EventFormModal, bool, error) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			// Cancel form immediately
			return nil, true, nil

		case "tab", "down":
			f.nextFocus()
			return f, false, nil

		case "shift+tab", "up":
			f.prevFocus()
			return f, false, nil

		case "enter":
			if f.FocusIndex == len(f.Inputs) { // Submit button
				return f, true, nil
			}
			f.nextFocus()
			return f, false, nil
		}
	}

	// Update the focused input
	if f.FocusIndex >= 0 && f.FocusIndex < len(f.Inputs) {
		var cmd tea.Cmd
		f.Inputs[f.FocusIndex], cmd = f.Inputs[f.FocusIndex].Update(msg)
		_ = cmd
	}

	return f, false, nil
}

func (f *EventFormModal) nextFocus() {
	if f.FocusIndex < len(f.Inputs) {
		f.Inputs[f.FocusIndex].Blur()
		f.Inputs[f.FocusIndex].Prompt = "  "
	}
	f.FocusIndex++
	if f.FocusIndex > len(f.Inputs) {
		f.FocusIndex = 0
	}
	if f.FocusIndex < len(f.Inputs) {
		f.Inputs[f.FocusIndex].Focus()
		f.Inputs[f.FocusIndex].Prompt = "▶ "
	}
}

func (f *EventFormModal) prevFocus() {
	if f.FocusIndex < len(f.Inputs) {
		f.Inputs[f.FocusIndex].Blur()
		f.Inputs[f.FocusIndex].Prompt = "  "
	}
	f.FocusIndex--
	if f.FocusIndex < 0 {
		f.FocusIndex = len(f.Inputs)
	}
	if f.FocusIndex < len(f.Inputs) {
		f.Inputs[f.FocusIndex].Focus()
		f.Inputs[f.FocusIndex].Prompt = "▶ "
	}
}

func parseDateFlexibleInTUI(input string, loc *time.Location) (time.Time, error) {
	now := time.Now().In(loc)
	clean := strings.TrimSpace(strings.ToLower(input))

	switch clean {
	case "", "today", "hom nay", "hôm nay":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc), nil
	case "tomorrow", "ngay mai", "ngày mai":
		tomorrow := now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, loc), nil
	case "yesterday", "hom qua", "hôm qua":
		yesterday := now.AddDate(0, 0, -1)
		return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, loc), nil
	}

	var separator string
	if strings.Contains(input, "/") {
		separator = "/"
	} else if strings.Contains(input, "-") {
		separator = "-"
	} else if strings.Contains(input, ".") {
		separator = "."
	} else {
		return time.Time{}, fmt.Errorf("định dạng ngày không hợp lệ (hỗ trợ dd/mm, dd-mm, dd/mm/yyyy)")
	}

	parts := strings.Split(input, separator)
	var day, month, year int

	if len(parts) == 3 {
		p0, err0 := strconv.Atoi(strings.TrimSpace(parts[0]))
		p1, err1 := strconv.Atoi(strings.TrimSpace(parts[1]))
		p2, err2 := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err0 != nil || err1 != nil || err2 != nil {
			return time.Time{}, fmt.Errorf("ngày không hợp lệ")
		}

		if p0 >= 1000 {
			year = p0
			month = p1
			day = p2
		} else {
			day = p0
			month = p1
			year = p2
			if year < 100 {
				year += 2000
			}
		}
	} else if len(parts) == 2 {
		p0, err0 := strconv.Atoi(strings.TrimSpace(parts[0]))
		p1, err1 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err0 != nil || err1 != nil {
			return time.Time{}, fmt.Errorf("ngày không hợp lệ")
		}
		day = p0
		month = p1
		year = now.Year()
	} else {
		return time.Time{}, fmt.Errorf("định dạng ngày không hợp lệ")
	}

	if month < 1 || month > 12 || day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("ngày tháng ngoài khoảng cho phép")
	}

	res := time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
	if res.Day() != day || int(res.Month()) != month || res.Year() != year {
		return time.Time{}, fmt.Errorf("ngày không tồn tại trên lịch")
	}

	return res, nil
}

func parseTimeFlexibleInTUI(input string) (int, int, error) {
	clean := strings.TrimSpace(strings.ToLower(input))
	if clean == "" {
		return 0, 0, fmt.Errorf("thời gian trống")
	}

	// 12h format
	if strings.HasSuffix(clean, "am") || strings.HasSuffix(clean, "pm") {
		isPM := strings.HasSuffix(clean, "pm")
		timePart := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(clean, "pm"), "am"))
		parts := strings.Split(timePart, ":")
		h, err := strconv.Atoi(parts[0])
		if err != nil || h < 1 || h > 12 {
			return 0, 0, fmt.Errorf("giờ không hợp lệ")
		}
		m := 0
		if len(parts) > 1 {
			m, err = strconv.Atoi(parts[1])
			if err != nil || m < 0 || m > 59 {
				return 0, 0, fmt.Errorf("phút không hợp lệ")
			}
		}
		if isPM && h < 12 {
			h += 12
		} else if !isPM && h == 12 {
			h = 0
		}
		return h, m, nil
	}

	// 24h format
	parts := strings.Split(clean, ":")
	if len(parts) >= 2 {
		h, err1 := strconv.Atoi(parts[0])
		m, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
			return 0, 0, fmt.Errorf("giờ 24h không hợp lệ")
		}
		return h, m, nil
	} else if len(parts) == 1 {
		h, err := strconv.Atoi(parts[0])
		if err == nil && h >= 0 && h <= 23 {
			return h, 0, nil
		}
	}

	return 0, 0, fmt.Errorf("định dạng giờ không hợp lệ (ví dụ: 10:00 hoặc 10am)")
}

func (f *EventFormModal) SubmitToDB(db *sql.DB, loc *time.Location) error {
	title := strings.TrimSpace(f.Inputs[0].Value())
	if title == "" {
		return fmt.Errorf("tiêu đề không được để trống")
	}

	startDate, err := parseDateFlexibleInTUI(f.Inputs[1].Value(), loc)
	if err != nil {
		return err
	}

	startH, startM, err := parseTimeFlexibleInTUI(f.Inputs[2].Value())
	if err != nil {
		return fmt.Errorf("giờ bắt đầu: %w", err)
	}

	endH, endM, err := parseTimeFlexibleInTUI(f.Inputs[3].Value())
	if err != nil {
		return fmt.Errorf("giờ kết thúc: %w", err)
	}

	// [4] Ngày kết thúc — mặc định cùng ngày bắt đầu nếu trống
	endDate := startDate
	if endDateStr := strings.TrimSpace(f.Inputs[4].Value()); endDateStr != "" {
		if parsed, err2 := parseDateFlexibleInTUI(endDateStr, loc); err2 == nil {
			endDate = parsed
		}
	}

	startTime := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startH, startM, 0, 0, loc)
	endTime := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), endH, endM, 0, 0, loc)
	// Nếu cùng ngày mà giờ kết thúc <= giờ bắt đầu → qua đêm
	if endDate.Equal(startDate) && !endTime.After(startTime) {
		endTime = endTime.AddDate(0, 0, 1)
	}

	startRFC := startTime.UTC().Format(time.RFC3339)
	endRFC := endTime.UTC().Format(time.RFC3339)
	locationVal := strings.TrimSpace(f.Inputs[5].Value())
	descVal := strings.TrimSpace(f.Inputs[6].Value())

	eventID := f.EventID
	calID := f.CalendarID
	if calID == "" {
		calID = "default"
	}

	if f.Mode == FormModeAdd {
		if eventID == "" {
			b := make([]byte, 16)
			_, _ = rand.Read(b)
			eventID = hex.EncodeToString(b)
		}
		newEvent := cache.LocalEvent{
			ID:          eventID,
			CalendarID:  calID,
			Title:       title,
			Description: descVal,
			StartAt:     startRFC,
			EndAt:       endRFC,
			Timezone:    loc.String(),
			Location:    locationVal,
			SyncState:   cache.SyncStatePendingCreate,
		}
		if err := cache.UpsertEvent(db, newEvent); err != nil {
			return err
		}
		_, _ = cache.EnqueueSyncJob(db, "event", eventID, cache.SyncOpCreate, syncer.MarshalPayload(newEvent))
	} else if f.Mode == FormModeEdit {
		updatedEvent := cache.LocalEvent{
			ID:          eventID,
			CalendarID:  calID,
			Title:       title,
			Description: descVal,
			StartAt:     startRFC,
			EndAt:       endRFC,
			Timezone:    loc.String(),
			Location:    locationVal,
			SyncState:   cache.SyncStatePendingUpdate,
		}
		if err := cache.UpsertEvent(db, updatedEvent); err != nil {
			return err
		}
		_, _ = cache.EnqueueSyncJob(db, "event", eventID, cache.SyncOpUpdate, syncer.MarshalPayload(updatedEvent))
	}

	// Trigger background sync
	cfg, err := config.LoadConfig()
	if err == nil && cfg.Token != "" {
		client := api.NewClient(cfg.ServerURL, cfg.Token)
		engine := syncer.NewSyncEngine(db, client)
		engine.SyncInBackground()
	}

	return nil
}

func (f *EventFormModal) DeleteFromDB(db *sql.DB) error {
	if f.EventID == "" {
		return fmt.Errorf("không có ID sự kiện để xóa")
	}

	if err := cache.MarkEventPendingDelete(db, f.EventID); err != nil {
		return err
	}

	_, _ = cache.EnqueueSyncJob(db, "event", f.EventID, cache.SyncOpDelete, "")

	cfg, err := config.LoadConfig()
	if err == nil && cfg.Token != "" {
		client := api.NewClient(cfg.ServerURL, cfg.Token)
		engine := syncer.NewSyncEngine(db, client)
		engine.SyncInBackground()
	}

	return nil
}

func (f EventFormModal) Render(width, height int) string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#74C0FC")).
		Padding(1, 2).
		Width(56)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#74C0FC"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7982A9"))
	btnStyle := lipgloss.NewStyle().Padding(0, 2).Bold(true)
	activeBtnStyle := btnStyle.Background(lipgloss.Color("#74C0FC")).Foreground(lipgloss.Color("#000000"))
	inactiveBtnStyle := btnStyle.Background(lipgloss.Color("#313244")).Foreground(lipgloss.Color("#CDD6F4"))

	if f.Mode == FormModeDelete {
		deleteTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFA8A8")).Render("XÁC NHẬN XÓA SỰ KIỆN")
		evTitle := "(Không có tiêu đề)"
		if f.DeleteEvent != nil && f.DeleteEvent.Title != "" {
			evTitle = f.DeleteEvent.Title
		}

		lines := []string{
			deleteTitle,
			"",
			"Bạn có chắc chắn muốn xóa sự kiện:",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Render("  \"" + evTitle + "\""),
			"",
			fmt.Sprintf("%s    %s",
				activeBtnStyle.Render("[y] Xóa ngay"),
				inactiveBtnStyle.Render("[Esc/Ctrl+C] Hủy bỏ"),
			),
		}
		card := modalStyle.BorderForeground(lipgloss.Color("#FFA8A8")).Render(strings.Join(lines, "\n"))
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, card)
	}

	var modalTitle string
	if f.Mode == FormModeAdd {
		modalTitle = titleStyle.Render("✚ TẠO SỰ KIỆN MỚI")
	} else {
		modalTitle = titleStyle.Render("✎ CHỈNH SỬA SỰ KIỆN")
	}

	// Plain-text labels (không có ANSI) để pad width chính xác
	redAsterisk := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Bold(true).Render("*")
	reqStar := redAsterisk
	labelWidth := 16 // độ rộng plain text tối đa
	plainLabels := []string{
		"Tiêu đề",
		"Ngày bắt đầu",
		"Bắt đầu",
		"Kết thúc",
		"Ngày kết thúc",
		"Địa điểm",
		"Ghi chú",
	}
	required := []bool{true, true, true, true, false, false, false}

	var lines []string
	lines = append(lines, modalTitle)
	lines = append(lines, "")

	if f.ErrorMsg != "" {
		errBox := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA8A8")).Render("⚠ " + f.ErrorMsg)
		lines = append(lines, errBox, "")
	}

	for i := 0; i < len(f.Inputs); i++ {
		// Pad plain text đến labelWidth để colon thẳng hàng
		padded := fmt.Sprintf("%-*s", labelWidth, plainLabels[i])
		// Dành slot cố định cho asterisk: " * " (3 visible chars) hoặc "   " (3 spaces)
		// => tất cả input bắt đầu tại cùng cột
		var prefix string
		if required[i] {
			prefix = labelStyle.Render(padded+":") + " " + reqStar + " "
		} else {
			prefix = labelStyle.Render(padded+":") + "   "
		}
		inputView := f.Inputs[i].View()
		lines = append(lines, prefix+inputView)
	}

	lines = append(lines, "")

	saveBtn := inactiveBtnStyle.Render("[ Lưu (Enter) ]")
	if f.FocusIndex == len(f.Inputs) {
		saveBtn = activeBtnStyle.Render("[ Lưu (Enter) ]")
	}
	cancelHint := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")).Italic(true).Render("Hủy [Esc] - [Ctrl+C]")

	lines = append(lines, fmt.Sprintf("%s    %s", saveBtn, cancelHint))

	card := modalStyle.Render(strings.Join(lines, "\n"))
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, card)
}
