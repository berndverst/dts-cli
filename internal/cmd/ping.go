package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

func newPingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Check connectivity to the DTS backend",
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			if err := client.Ping(context.Background()); err != nil {
				exitError("ping failed", err)
			}
			printJSON(map[string]string{"status": "ok"})
		},
	}
}
