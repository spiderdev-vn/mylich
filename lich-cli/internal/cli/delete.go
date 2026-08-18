package cli

import (
	"context"
	"fmt"

	"lich-cli/internal/api"
	"lich-cli/internal/config"
)

func RunDelete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("event ID required: lich delete <id>")
	}
	eventID := args[0]

	cfg, err := config.LoadConfig()
	if err != nil || cfg.Token == "" {
		return fmt.Errorf("not logged in. Please run 'lich login' first")
	}

	client := api.NewClient(cfg.ServerURL, cfg.Token)
	ctx := context.Background()

	if err := client.DeleteEvent(ctx, eventID); err != nil {
		return fmt.Errorf("failed to delete event '%s': %w", eventID, err)
	}

	fmt.Printf("✓ Event %s deleted\n", eventID)
	return nil
}
