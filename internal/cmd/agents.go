package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Azure/durabletask-cli/internal/api"
)

func newAgentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agents",
		Aliases: []string{"ag"},
		Short:   "Manage agent sessions",
	}

	cmd.AddCommand(
		newAgentListCmd(),
		newAgentStartCmd(),
		newAgentSendCmd(),
		newAgentStateCmd(),
		newAgentDeleteCmd(),
	)

	return cmd
}

// --- list ---

func newAgentListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List agent sessions",
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			pageSize, _ := c.Flags().GetInt("page-size")
			startIndex, _ := c.Flags().GetInt("start-index")

			result, err := client.ListAgentSessions(context.Background(), pageSize, startIndex)
			if err != nil {
				exitError("list agent sessions failed", err)
			}

			// Convert raw entities to agent entity format
			agents := make([]*api.AgentEntity, 0, len(result.Entities))
			for i := range result.Entities {
				agents = append(agents, api.ParseAgentEntity(&result.Entities[i]))
			}
			printJSON(map[string]interface{}{
				"agents":     agents,
				"totalCount": result.TotalCount,
			})
		},
	}

	cmd.Flags().Int("page-size", 50, "Number of results per page")
	cmd.Flags().Int("start-index", 0, "Pagination start index")

	return cmd
}

// --- start ---

func newAgentStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new agent session",
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}

			name, _ := c.Flags().GetString("name")
			if name == "" {
				exitError("--name is required", nil)
			}
			sessionID, _ := c.Flags().GetString("session-id")
			if sessionID == "" {
				exitError("--session-id is required", nil)
			}
			prompt, _ := c.Flags().GetString("prompt")
			if prompt == "" {
				exitError("--prompt is required", nil)
			}

			instanceID, err := client.StartAgentSession(context.Background(), name, sessionID, prompt)
			if err != nil {
				exitError("start agent session failed", err)
			}
			printJSON(map[string]string{
				"status":     "started",
				"name":       name,
				"sessionId":  sessionID,
				"instanceId": instanceID,
			})
		},
	}

	cmd.Flags().String("name", "", "Agent name (required)")
	cmd.Flags().String("session-id", "", "Session ID (required)")
	cmd.Flags().String("prompt", "", "Initial prompt (required)")

	return cmd
}

// --- send ---

func newAgentSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a prompt to an existing agent session",
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}

			name, _ := c.Flags().GetString("name")
			if name == "" {
				exitError("--name is required", nil)
			}
			sessionID, _ := c.Flags().GetString("session-id")
			if sessionID == "" {
				exitError("--session-id is required", nil)
			}
			prompt, _ := c.Flags().GetString("prompt")
			if prompt == "" {
				exitError("--prompt is required", nil)
			}

			if err := client.SendAgentPrompt(context.Background(), name, sessionID, prompt); err != nil {
				exitError("send prompt failed", err)
			}
			printJSON(map[string]string{
				"status":    "sent",
				"name":      name,
				"sessionId": sessionID,
			})
		},
	}

	cmd.Flags().String("name", "", "Agent name (required)")
	cmd.Flags().String("session-id", "", "Session ID (required)")
	cmd.Flags().String("prompt", "", "Prompt text (required)")

	return cmd
}

// --- state ---

func newAgentStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Get the state of an agent session",
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}

			name, _ := c.Flags().GetString("name")
			if name == "" {
				exitError("--name is required", nil)
			}
			sessionID, _ := c.Flags().GetString("session-id")
			if sessionID == "" {
				exitError("--session-id is required", nil)
			}

			state, err := client.GetAgentState(context.Background(), name, sessionID)
			if err != nil {
				exitError("get agent state failed", err)
			}
			printJSON(state)
		},
	}

	cmd.Flags().String("name", "", "Agent name (required)")
	cmd.Flags().String("session-id", "", "Session ID (required)")

	return cmd
}

// --- delete ---

func newAgentDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <entity-instance-id> [entity-instance-id...]",
		Short: "Delete one or more agent session entities",
		Long: fmt.Sprintf("Delete agent session entities by their full instance ID.\n" +
			"Instance IDs have the format: @agent@<Name>@<SessionId>"),
		Args: cobra.MinimumNArgs(1),
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			if len(args) == 1 {
				if err := client.DeleteAgentSession(context.Background(), args[0]); err != nil {
					exitError("delete agent session failed", err)
				}
				printStatus("deleted", args[0])
			} else {
				if err := client.DeleteAgentSessions(context.Background(), args); err != nil {
					exitError("delete agent sessions failed", err)
				}
				printJSON(map[string]interface{}{
					"status": "deleted",
					"ids":    args,
				})
			}
		},
	}
}
