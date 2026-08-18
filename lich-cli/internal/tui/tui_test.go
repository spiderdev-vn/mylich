package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"lich-cli/internal/cache"
)

func TestTUI_Navigation(t *testing.T) {
	model := NewModel(nil, nil)
	baseDate := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
	model.SelectedDate = baseDate
	model.CurrentMonth = time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	model.Location = time.UTC

	// 1. Move right (next day)
	m1, _ := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	newModel := m1.(Model)
	if newModel.SelectedDate.Day() != 19 {
		t.Errorf("expected day 19 after KeyRight, got %d", newModel.SelectedDate.Day())
	}

	// 2. Move left (prev day)
	m2, _ := newModel.Update(tea.KeyMsg{Type: tea.KeyLeft})
	newModel = m2.(Model)
	if newModel.SelectedDate.Day() != 18 {
		t.Errorf("expected day 18 after KeyLeft, got %d", newModel.SelectedDate.Day())
	}

	// 3. Move down (next week +7 days)
	m3, _ := newModel.Update(tea.KeyMsg{Type: tea.KeyDown})
	newModel = m3.(Model)
	if newModel.SelectedDate.Day() != 25 {
		t.Errorf("expected day 25 after KeyDown, got %d", newModel.SelectedDate.Day())
	}

	// 4. Move up (prev week -7 days)
	m4, _ := newModel.Update(tea.KeyMsg{Type: tea.KeyUp})
	newModel = m4.(Model)
	if newModel.SelectedDate.Day() != 18 {
		t.Errorf("expected day 18 after KeyUp, got %d", newModel.SelectedDate.Day())
	}
}

func TestTUI_MonthChange(t *testing.T) {
	model := NewModel(nil, nil)
	model.CurrentMonth = time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	model.SelectedDate = time.Date(2026, time.August, 18, 0, 0, 0, 0, time.UTC)
	model.Location = time.UTC

	// Next month ('n')
	m1, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	newModel := m1.(Model)
	if newModel.CurrentMonth.Month() != time.September {
		t.Errorf("expected September after 'n', got %s", newModel.CurrentMonth.Month())
	}

	// Prev month ('p')
	m2, _ := newModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	newModel = m2.(Model)
	if newModel.CurrentMonth.Month() != time.August {
		t.Errorf("expected August after 'p', got %s", newModel.CurrentMonth.Month())
	}
}

func TestTUI_CRUDInteractions(t *testing.T) {
	model := NewModel(nil, nil)
	model.Width = 100
	model.Height = 30
	model.SelectedDate = time.Date(2026, time.August, 18, 0, 0, 0, 0, time.UTC)
	model.CurrentMonth = time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	model.AgendaMode = "list" // Ensure deterministic start for cycle test
	model.Events["2026-08-18"] = []cache.LocalEvent{
		{
			ID:        "ev-1",
			Title:     "Sprint Planning",
			StartAt:   "2026-08-18T10:00:00Z",
			EndAt:     "2026-08-18T11:00:00Z",
			SyncState: cache.SyncStateSynced,
		},
		{
			ID:        "ev-2",
			Title:     "Team Lunch",
			StartAt:   "2026-08-18T12:00:00Z",
			EndAt:     "2026-08-18T13:00:00Z",
			SyncState: cache.SyncStateSynced,
		},
	}

	// 1. Tab switches focus to Agenda
	m1, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = m1.(Model)
	if model.Focus != FocusAgenda {
		t.Errorf("expected FocusAgenda after Tab, got %v", model.Focus)
	}

	// 2. Down in Agenda moves to next event
	m2, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = m2.(Model)
	if model.SelectedEventIdx != 1 {
		t.Errorf("expected SelectedEventIdx 1 after KeyDown, got %d", model.SelectedEventIdx)
	}

	// 3. Up in Agenda moves back to first event
	m3, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = m3.(Model)
	if model.SelectedEventIdx != 0 {
		t.Errorf("expected SelectedEventIdx 0 after KeyUp, got %d", model.SelectedEventIdx)
	}

	// 4. Enter opens detail modal
	m4, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = m4.(Model)
	if model.ViewingEvent == nil || model.ViewingEvent.ID != "ev-1" {
		t.Errorf("expected ViewingEvent to be ev-1, got %v", model.ViewingEvent)
	}

	// View should render modal
	modalView := model.View()
	if modalView == "" {
		t.Errorf("modal view is empty")
	}

	// 5. Esc closes modal
	m5, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = m5.(Model)
	if model.ViewingEvent != nil {
		t.Errorf("expected ViewingEvent to be nil after Esc, got %v", model.ViewingEvent)
	}

	// 6. Press 'a' opens Native Add Modal
	m6, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = m6.(Model)
	if model.Modal == nil || model.Modal.Mode != FormModeAdd {
		t.Errorf("expected FormModeAdd modal, got %v", model.Modal)
	}

	// Ctrl+C cancels Add Modal
	m7, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model = m7.(Model)
	if model.Modal != nil {
		t.Errorf("expected Modal to be nil after Ctrl+C, got %v", model.Modal)
	}

	// 7. Press 'e' opens Native Edit Modal
	m8, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = m8.(Model)
	if model.Modal == nil || model.Modal.Mode != FormModeEdit || model.Modal.EventID != "ev-1" {
		t.Errorf("expected FormModeEdit modal for ev-1, got %v", model.Modal)
	}

	// Esc cancels Edit Modal
	m9, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = m9.(Model)
	if model.Modal != nil {
		t.Errorf("expected Modal to be nil after Esc, got %v", model.Modal)
	}

	// 8. Press 'd' opens Native Delete Confirm Modal
	m10, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = m10.(Model)
	if model.Modal == nil || model.Modal.Mode != FormModeDelete || model.Modal.EventID != "ev-1" {
		t.Errorf("expected FormModeDelete modal for ev-1, got %v", model.Modal)
	}

	// 'n' cancels Delete Modal
	m11, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model = m11.(Model)
	if model.Modal != nil {
		t.Errorf("expected Modal to be nil after 'n', got %v", model.Modal)
	}

	// 9. 'm' key cycles Agenda modes: list -> timeline -> gantt -> ascii -> list
	m12, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	model = m12.(Model)
	if model.AgendaMode != "timeline" {
		t.Errorf("expected AgendaMode timeline, got %s", model.AgendaMode)
	}

	m13, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	model = m13.(Model)
	if model.AgendaMode != "gantt" {
		t.Errorf("expected AgendaMode gantt, got %s", model.AgendaMode)
	}

	m14, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	model = m14.(Model)
	if model.AgendaMode != "ascii" {
		t.Errorf("expected AgendaMode ascii, got %s", model.AgendaMode)
	}

	m15, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	model = m15.(Model)
	if model.AgendaMode != "list" {
		t.Errorf("expected AgendaMode list, got %s", model.AgendaMode)
	}
}

func TestTUI_OverlappingEvents(t *testing.T) {
	loc := time.UTC
	selectedDate := time.Date(2026, time.August, 18, 0, 0, 0, 0, loc)

	events := []cache.LocalEvent{
		{
			ID:        "e1",
			Title:     "di lam viec",
			StartAt:   "2026-08-18T12:00:00Z",
			EndAt:     "2026-08-18T18:00:00Z",
			SyncState: cache.SyncStateSynced,
		},
		{
			ID:        "e2",
			Title:     "Team Retrospective",
			StartAt:   "2026-08-18T14:00:00Z",
			EndAt:     "2026-08-18T15:00:00Z",
			Location:  "Meeting Room 3",
			SyncState: cache.SyncStateSynced,
		},
	}

	// 1. Conflict detection
	conflicts := detectConflicts(events, loc)
	if !conflicts[0] || !conflicts[1] {
		t.Errorf("expected e1 and e2 to be in conflict")
	}

	// 2. Test RenderAgendaList
	listView := RenderAgendaList(selectedDate, events, loc, 0, true)
	if !strings.Contains(listView, "Trùng giờ") {
		t.Errorf("expected 'Trùng giờ' badge in list view")
	}

	// 3. Test RenderAgendaTimeline (Parallel columns for overlapping events)
	timelineView := RenderAgendaTimeline(selectedDate, events, loc, 0, true, 60)
	if !strings.Contains(timelineView, "di lam viec") || !strings.Contains(timelineView, "Team Retrospective") {
		t.Errorf("expected events in Timeline view, got: %s", timelineView)
	}
	if !strings.Contains(timelineView, "Cột 1") || !strings.Contains(timelineView, "Cột 2") {
		t.Errorf("expected multi-column parallel tracks in timeline view, got: %s", timelineView)
	}

	// 4. Test RenderAgendaGantt (Horizontal duration bars)
	ganttView := RenderAgendaGantt(selectedDate, events, loc, 0, true, 60)
	if !strings.Contains(ganttView, "di lam viec") || !strings.Contains(ganttView, "Team Retro") {
		t.Errorf("expected events in Gantt view, got: %s", ganttView)
	}
	if !strings.Contains(ganttView, "█") {
		t.Errorf("expected visual horizontal bars in gantt view, got: %s", ganttView)
	}

	// 5. Test RenderAgendaASCII
	asciiView := RenderAgendaASCII(selectedDate, events, loc, 0, true)
	if !strings.Contains(asciiView, "[CONFLICT]") {
		t.Errorf("expected '[CONFLICT]' tag in ascii view")
	}
}

func TestTUI_TerminalTooSmall(t *testing.T) {
	model := NewModel(nil, nil)
	model.Width = 40 // Too narrow (< 60)
	model.Height = 20

	view := model.View()
	if !strings.Contains(view, "CỬA SỔ TERMINAL QUÁ NHỎ") {
		t.Errorf("expected warning about small terminal, got: %s", view)
	}
}

func TestTUI_RenderView(t *testing.T) {
	model := NewModel(nil, nil)
	model.Width = 100
	model.Height = 30
	model.SelectedDate = time.Date(2026, time.August, 18, 0, 0, 0, 0, time.UTC)
	model.CurrentMonth = time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	model.Events["2026-08-18"] = []cache.LocalEvent{
		{
			ID:        "test-1",
			Title:     "Sprint Review",
			StartAt:   "2026-08-18T10:00:00Z",
			EndAt:     "2026-08-18T11:00:00Z",
			SyncState: cache.SyncStateSynced,
		},
	}

	view := model.View()
	if view == "" {
		t.Fatalf("rendered view is empty")
	}
}

func TestTUI_MultiDayEventFormatAndDetail(t *testing.T) {
	loc := time.UTC

	// 1. Same-day event
	f1 := formatEventTime("2026-08-18T10:00:00Z", "2026-08-18T11:30:00Z", loc)
	if f1 != "10:00 - 11:30" {
		t.Errorf("expected '10:00 - 11:30', got '%s'", f1)
	}

	// 2. Overnight cross midnight (20/08 22:00 -> 21/08 03:00)
	f2 := formatEventTime("2026-08-20T22:00:00Z", "2026-08-21T03:00:00Z", loc)
	if f2 != "22:00 20/08 - 03:00 21/08" {
		t.Errorf("expected '22:00 20/08 - 03:00 21/08', got '%s'", f2)
	}

	// 3. Multi-day 3-day spanning (20/08 08:00 -> 23/08 17:00)
	f3 := formatEventTime("2026-08-20T08:00:00Z", "2026-08-23T17:00:00Z", loc)
	if f3 != "08:00 20/08 - 17:00 23/08" {
		t.Errorf("expected '08:00 20/08 - 17:00 23/08', got '%s'", f3)
	}
}
