package ui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
)

// DefaultFormKeyMap returns a customized huh.KeyMap that allows using
// Up and Down arrow keys to navigate between form input fields seamlessly,
// alongside Tab / Shift+Tab and Enter.
func DefaultFormKeyMap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()

	// Up / Down navigation on text inputs
	km.Input.Next = key.NewBinding(
		key.WithKeys("tab", "down", "enter"),
		key.WithHelp("↓/tab/enter", "tiếp theo"),
	)
	km.Input.Prev = key.NewBinding(
		key.WithKeys("shift+tab", "up"),
		key.WithHelp("↑/shift+tab", "trước đó"),
	)

	// Left / Right / Up / Down navigation on confirm buttons
	km.Confirm.Next = key.NewBinding(
		key.WithKeys("tab", "right", "down"),
		key.WithHelp("→/↓/tab", "tiếp theo"),
	)
	km.Confirm.Prev = key.NewBinding(
		key.WithKeys("shift+tab", "left", "up"),
		key.WithHelp("←/↑/shift+tab", "trước đó"),
	)

	// Tab / Shift+Tab on select dropdowns (Up/Down moves items inside select)
	km.Select.Next = key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "tiếp theo"),
	)
	km.Select.Prev = key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "trước đó"),
	)

	// Quit / Abort keybindings
	km.Quit = key.NewBinding(
		key.WithKeys("ctrl+c", "esc"),
		key.WithHelp("esc", "hủy bỏ"),
	)

	return km
}
