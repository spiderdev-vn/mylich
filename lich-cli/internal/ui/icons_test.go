package ui

import (
	"testing"
)

func TestIcons_GetIconSet(t *testing.T) {
	// Unicode preset
	u := GetIconSet("unicode")
	if u.Name != "unicode" || u.Check != "✓" || u.Bullet != "•" {
		t.Errorf("unexpected unicode preset: %+v", u)
	}

	// Nerd font preset
	n := GetIconSet("nerd")
	if n.Name != "nerd" || n.Server != "󰒋" || n.Calendar != "󰃭" {
		t.Errorf("unexpected nerd preset: %+v", n)
	}

	// ASCII preset
	a := GetIconSet("ascii")
	if a.Name != "ascii" || a.Check != "[v]" || a.Server != "[SERVER]" {
		t.Errorf("unexpected ascii preset: %+v", a)
	}

	// Emoji preset
	e := GetIconSet("emoji")
	if e.Name != "emoji" || e.Calendar != "📅" || e.Location != "📍" {
		t.Errorf("unexpected emoji preset: %+v", e)
	}

	// Fallback to unicode
	fb := GetIconSet("non_existent_preset")
	if fb.Name != "unicode" {
		t.Errorf("expected fallback to unicode, got %s", fb.Name)
	}
}

func TestIcons_CurrentIcons(t *testing.T) {
	icons := CurrentIcons()
	if icons.Name == "" {
		t.Errorf("CurrentIcons returned empty icon set")
	}
}
