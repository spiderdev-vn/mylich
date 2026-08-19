package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type actionFinishedMsg struct {
	err error
}

type spinnerModel struct {
	spinner spinner.Model
	title   string
	action  func() error
	err     error
	done    bool
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			err := m.action()
			return actionFinishedMsg{err: err}
		},
	)
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actionFinishedMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m spinnerModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("%s %s", m.spinner.View(), LabelStyle.Render(m.title))
}

// RunWithSpinner executes an action with an animated Charm spinner styled in Pop colors.
func RunWithSpinner(title string, action func() error) error {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)

	m := spinnerModel{
		spinner: s,
		title:   title,
		action:  action,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	if sm, ok := finalModel.(spinnerModel); ok {
		return sm.err
	}
	return nil
}

// NewProgressBar returns a Charm bubbles/progress bar styled with Pop gradients.
func NewProgressBar(width int) progress.Model {
	p := progress.New(
		progress.WithGradient("#FF2A85", "#8075FF"),
		progress.WithWidth(width),
	)
	p.EmptyColor = "#2A2E3F"
	return p
}
