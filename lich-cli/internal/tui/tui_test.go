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
