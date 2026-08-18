package tui

import (
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

func TestTUI_RenderView(t *testing.T) {
	model := NewModel(nil, nil)
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
