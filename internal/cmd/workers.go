package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

func newWorkersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workers",
		Aliases: []string{"work"},
		Short:   "View connected workers",
	}

	cmd.AddCommand(newWorkerListCmd())

	return cmd
}

// --- list ---

func newWorkerListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List connected workers",
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			result, err := client.ListWorkers(context.Background())
			if err != nil {
				exitError("list workers failed", err)
			}
			printJSON(result)
		},
	}
}
