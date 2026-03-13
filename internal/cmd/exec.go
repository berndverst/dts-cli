package cmd

import "github.com/spf13/cobra"

// NewExecCmd creates the top-level "exec" command for non-interactive operations.
func NewExecCmd() *cobra.Command {
	execCmd := &cobra.Command{
		Use:   "exec",
		Short: "Run non-interactive commands (JSON output)",
		Long: `Execute one-off operations against Durable Task Scheduler.
All output is JSON, making these commands suitable for scripts, automation,
and AI agents. Use global flags --url, --taskhub, --auth-mode, and --tenant-id
for authentication.`,
	}

	execCmd.AddCommand(
		newOrchestationsCmd(),
		newEntitiesCmd(),
		newSchedulesCmd(),
		newWorkersCmd(),
		newAgentsCmd(),
		newPingCmd(),
	)

	return execCmd
}
