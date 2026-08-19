package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// StartSpinner displays an animated spinner with the given message until the returned stop function is called.
func StartSpinner(w io.Writer, message string) func() {
	if w == nil {
		w = os.Stdout
	}

	// If stdout is not a TTY or simple mode is active, don't show dynamic animation
	if f, ok := w.(*os.File); ok {
		if !isatty.IsTerminal(f.Fd()) && !isatty.IsCygwinTerminal(f.Fd()) {
			return func() {}
		}
	}

	stopChan := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		idx := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				// Clear the line
				fmt.Fprintf(w, "\r\033[K")
				return
			case <-ticker.C:
				frame := spinnerFrames[idx%len(spinnerFrames)]
				styledFrame := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(frame)
				fmt.Fprintf(w, "\r%s %s", styledFrame, LabelStyle.Render(message))
				idx++
			}
		}
	}()

	return func() {
		close(stopChan)
		wg.Wait()
	}
}
