package ui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StepStatus int

const (
	StepPending StepStatus = iota
	StepRunning
	StepCompleted
	StepFailed
)

var ErrAborted = errors.New("user aborted")

type StepItem struct {
	Title     string
	Detail    string
	Status    StepStatus
	StartTime time.Time
	Duration  time.Duration
}

type actionFinishedMsg struct {
	err error
}

type updateStepMsg struct {
	index  int
	status StepStatus
	detail string
}

type updateDetailMsg struct {
	detail string
}

type updateProgressMsg struct {
	percent float64
}

type timerTickMsg time.Time

type trackerModel struct {
	title       string
	steps       []StepItem
	currentStep int
	subDetail   string
	spinner     spinner.Model
	progress    progress.Model
	action      func(reporter *TrackerReporter) error
	err         error
	done        bool
	startTime   time.Time
}

type TrackerReporter struct {
	program *tea.Program
}

func (r *TrackerReporter) SetStepRunning(index int, detail string) {
	if r.program != nil {
		r.program.Send(updateStepMsg{index: index, status: StepRunning, detail: detail})
	}
}

func (r *TrackerReporter) SetStepDone(index int, detail string) {
	if r.program != nil {
		r.program.Send(updateStepMsg{index: index, status: StepCompleted, detail: detail})
	}
}

func (r *TrackerReporter) SetStepFailed(index int, detail string) {
	if r.program != nil {
		r.program.Send(updateStepMsg{index: index, status: StepFailed, detail: detail})
	}
}

func (r *TrackerReporter) SetSubDetail(detail string) {
	if r.program != nil {
		r.program.Send(updateDetailMsg{detail: detail})
	}
}

func (r *TrackerReporter) SetProgress(percent float64) {
	if r.program != nil {
		r.program.Send(updateProgressMsg{percent: percent})
	}
}

func tickTimer() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return timerTickMsg(t)
	})
}

func (m trackerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tickTimer(),
	)
}

func (m trackerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actionFinishedMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit

	case updateStepMsg:
		if msg.index >= 0 && msg.index < len(m.steps) {
			m.steps[msg.index].Status = msg.status
			if msg.detail != "" {
				m.steps[msg.index].Detail = msg.detail
			}
			if msg.status == StepRunning {
				m.currentStep = msg.index
				m.steps[msg.index].StartTime = time.Now()
			} else if msg.status == StepCompleted || msg.status == StepFailed {
				if !m.steps[msg.index].StartTime.IsZero() {
					m.steps[msg.index].Duration = time.Since(m.steps[msg.index].StartTime)
				}
				// Auto update progress percentage based on completed steps
				completed := 0
				for _, s := range m.steps {
					if s.Status == StepCompleted {
						completed++
					}
				}
				if len(m.steps) > 0 {
					m.progress.SetPercent(float64(completed) / float64(len(m.steps)))
				}
			}
		}
		return m, nil

	case updateDetailMsg:
		m.subDetail = msg.detail
		return m, nil

	case updateProgressMsg:
		m.progress.SetPercent(msg.percent)
		return m, nil

	case timerTickMsg:
		if !m.done {
			return m, tickTimer()
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "esc" {
			m.done = true
			m.err = ErrAborted
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m trackerModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	mutedStyle := lipgloss.NewStyle().Foreground(ColorMuted)
	doneStyle := lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)
	activeStyle := lipgloss.NewStyle().Foreground(ColorHighlight).Bold(true)
	pendingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6370"))
	subDetailStyle := lipgloss.NewStyle().Foreground(ColorSecondary).Italic(true)
	elapsedStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	b.WriteString(fmt.Sprintf("%s\n\n", titleStyle.Render("⚡ "+m.title)))

	for i, step := range m.steps {
		numTag := fmt.Sprintf("[%d/%d]", i+1, len(m.steps))
		switch step.Status {
		case StepCompleted:
			info := ""
			if step.Detail != "" {
				info = fmt.Sprintf(" (%s)", step.Detail)
			}
			durStr := ""
			if step.Duration > 0 {
				durStr = fmt.Sprintf(" [%0.1fs]", step.Duration.Seconds())
			}
			b.WriteString(fmt.Sprintf("  %s %s %s%s%s\n",
				doneStyle.Render("✓"),
				mutedStyle.Render(numTag),
				doneStyle.Render(step.Title),
				elapsedStyle.Render(info),
				elapsedStyle.Render(durStr),
			))

		case StepRunning:
			elapsed := ""
			if !step.StartTime.IsZero() {
				elapsed = fmt.Sprintf(" [%0.1fs]", time.Since(step.StartTime).Seconds())
			}
			info := ""
			if step.Detail != "" {
				info = fmt.Sprintf(" (%s)", step.Detail)
			}
			b.WriteString(fmt.Sprintf("  %s %s %s%s%s\n",
				m.spinner.View(),
				titleStyle.Render(numTag),
				activeStyle.Render(step.Title),
				elapsedStyle.Render(info),
				elapsedStyle.Render(elapsed),
			))
			if m.subDetail != "" {
				b.WriteString(fmt.Sprintf("      %s %s\n",
					lipgloss.NewStyle().Foreground(ColorSecondary).Render("↳"),
					subDetailStyle.Render(m.subDetail),
				))
			}

		case StepFailed:
			b.WriteString(fmt.Sprintf("  %s %s %s - %s\n",
				lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render("✗"),
				mutedStyle.Render(numTag),
				lipgloss.NewStyle().Foreground(ColorError).Render(step.Title),
				lipgloss.NewStyle().Foreground(ColorError).Render(step.Detail),
			))

		default:
			b.WriteString(fmt.Sprintf("  %s %s %s\n",
				pendingStyle.Render("○"),
				pendingStyle.Render(numTag),
				pendingStyle.Render(step.Title),
			))
		}
	}

	b.WriteString("\n  " + m.progress.View() + "\n")

	return ContainerCard.Render(b.String())
}

// RunWithTracker runs a multi-step workflow with live verbose progress tracking.
func RunWithTracker(title string, stepTitles []string, action func(reporter *TrackerReporter) error) error {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)

	prog := progress.New(
		progress.WithGradient("#FF2A85", "#8075FF"),
		progress.WithWidth(42),
	)
	prog.EmptyColor = "#2A2E3F"

	steps := make([]StepItem, len(stepTitles))
	for i, t := range stepTitles {
		steps[i] = StepItem{
			Title:  t,
			Status: StepPending,
		}
	}
	if len(steps) > 0 {
		steps[0].Status = StepRunning
		steps[0].StartTime = time.Now()
	}

	m := trackerModel{
		title:       title,
		steps:       steps,
		currentStep: 0,
		spinner:     s,
		progress:    prog,
		action:      action,
		startTime:   time.Now(),
	}

	p := tea.NewProgram(m)
	reporter := &TrackerReporter{program: p}

	go func() {
		err := action(reporter)
		p.Send(actionFinishedMsg{err: err})
	}()

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	if tm, ok := finalModel.(trackerModel); ok {
		return tm.err
	}
	return nil
}

// RunWithSpinner executes an action with an animated Charm spinner styled in Pop colors.
func RunWithSpinner(title string, action func() error) error {
	return RunWithTracker(title, []string{title}, func(reporter *TrackerReporter) error {
		return action()
	})
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
