package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"lich-cli/internal/api"
	"lich-cli/internal/config"
	"lich-cli/internal/tui"
)

const Version = "v0.1.0"

func printHelp() {
	fmt.Printf(`Lich — Personal Calendar System (%s)

Usage:
  lich                          Launch interactive calendar TUI
  lich login [options]          Authenticate with Lich server
  lich today [options]          Show today's agenda
  lich week [options]           Show this week's agenda
  lich add <title> [options]    Create a new event
  lich delete <id>              Delete an event
  lich version                  Show Lich version
  lich help                     Show this help message

Options for 'add':
  --date <YYYY-MM-DD>           Event date (default: today)
  --at <HH:MM>                  Event start time (default: next hour)
  --duration <duration>         Event duration (default: 1h, e.g. 30m, 2h)
  --calendar <id>               Target calendar ID
  --desc <text>                 Event description
  --location <text>             Event location
  --timezone <tz>               Event timezone (e.g. Asia/Ho_Chi_Minh)

Options for 'today' / 'week':
  --calendar <id>               Filter by calendar ID
  --json                        Output JSON format

Options for 'login':
  --server <url>                Server URL (default: http://127.0.0.1:3000)
  --username <user>             Username
  --password <pass>             Password
  --register                    Register new account
`, Version)
}

func Execute(args []string) int {
	if len(args) == 0 {
		return runTUI()
	}

	command := args[0]
	subArgs := args[1:]

	var err error
	switch command {
	case "login":
		err = RunLogin(subArgs)
	case "today":
		err = RunToday(subArgs)
	case "week":
		err = RunWeek(subArgs)
	case "add":
		err = RunAdd(subArgs)
	case "delete":
		err = RunDelete(subArgs)
	case "version", "-v", "--version":
		fmt.Printf("Lich %s\n", Version)
		return 0
	case "help", "-h", "--help":
		printHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown command '%s'. Run 'lich help' for available commands.\n", command)
		return 1
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}

func runTUI() int {
	cfg, err := config.LoadConfig()
	var client *api.Client
	if err == nil && cfg.Token != "" {
		client = api.NewClient(cfg.ServerURL, cfg.Token)
	}

	p := tea.NewProgram(tui.NewModel(client), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		return 1
	}
	return 0
}
