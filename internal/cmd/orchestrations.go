package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/microsoft/durabletask-scheduler/cli/internal/api"
)

func newOrchestationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "orchestrations",
		Aliases: []string{"orch"},
		Short:   "Manage orchestrations",
	}

	cmd.AddCommand(
		newOrchListCmd(),
		newOrchGetCmd(),
		newOrchPayloadsCmd(),
		newOrchHistoryCmd(),
		newOrchCreateCmd(),
		newOrchSuspendCmd(),
		newOrchResumeCmd(),
		newOrchTerminateCmd(),
		newOrchForceTerminateCmd(),
		newOrchRestartCmd(),
		newOrchRewindCmd(),
		newOrchPurgeCmd(),
		newOrchRaiseEventCmd(),
	)

	return cmd
}

// --- list ---

func newOrchListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List orchestrations with optional filters",
		Run: func(cmd *cobra.Command, args []string) {
			client, err := initClient(cmd)
			if err != nil {
				exitError("init failed", err)
			}

			req := buildOrchQueryRequest(cmd)

			result, err := client.QueryOrchestrations(context.Background(), req)
			if err != nil {
				exitError("query orchestrations failed", err)
			}
			printJSON(result)
		},
	}

	cmd.Flags().StringSlice("status", nil, "Filter by status: Running, Completed, Failed, Pending, Suspended, Terminated, Canceled, ContinuedAsNew")
	cmd.Flags().String("name", "", "Filter by orchestration name (substring match)")
	cmd.Flags().String("instance-id", "", "Filter by instance ID (substring match)")
	cmd.Flags().String("created-after", "", "Filter by created timestamp (RFC3339, e.g. 2024-01-01T00:00:00Z)")
	cmd.Flags().String("created-before", "", "Filter by created timestamp (RFC3339)")
	cmd.Flags().Int("page-size", 50, "Number of results per page")
	cmd.Flags().Int("start-index", 0, "Pagination start index")
	cmd.Flags().String("sort-by", "CREATED_AT", "Sort column: ORCHESTRATION_ID, NAME, VERSION, CREATED_AT, LAST_UPDATED_AT, START_AT, COMPLETED_AT, ORCHESTRATION_STATUS")
	cmd.Flags().String("sort-dir", "DESCENDING_SORT", "Sort direction: ASCENDING_SORT, DESCENDING_SORT")

	return cmd
}

func buildOrchQueryRequest(cmd *cobra.Command) *api.QueryOrchestrationsRequest {
	req := &api.QueryOrchestrationsRequest{
		Fields: api.DefaultOrchestrationFields,
	}

	// Status filter
	statuses, _ := cmd.Flags().GetStringSlice("status")
	if len(statuses) > 0 {
		var statusValues []string
		for _, s := range statuses {
			statusValues = append(statusValues, normalizeStatus(s))
		}
		if req.Filter == nil {
			req.Filter = &api.OrchestrationFilter{}
		}
		req.Filter.OrchestrationStatus = &api.StatusFilter{Status: statusValues}
	}

	// Name filter
	name, _ := cmd.Flags().GetString("name")
	if name != "" {
		if req.Filter == nil {
			req.Filter = &api.OrchestrationFilter{}
		}
		req.Filter.Name = &api.StringFilter{Value: name}
	}

	// Instance ID filter
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if instanceID != "" {
		if req.Filter == nil {
			req.Filter = &api.OrchestrationFilter{}
		}
		req.Filter.OrchestrationID = &api.StringFilter{Value: instanceID}
	}

	// Date filters
	createdAfter, _ := cmd.Flags().GetString("created-after")
	createdBefore, _ := cmd.Flags().GetString("created-before")
	if createdAfter != "" || createdBefore != "" {
		if req.Filter == nil {
			req.Filter = &api.OrchestrationFilter{}
		}
		dtFilter := &api.DateTimeFilter{}
		if createdAfter != "" {
			t, err := time.Parse(time.RFC3339, createdAfter)
			if err != nil {
				exitError("invalid --created-after format (use RFC3339)", err)
			}
			dtFilter.Start = &t
		}
		if createdBefore != "" {
			t, err := time.Parse(time.RFC3339, createdBefore)
			if err != nil {
				exitError("invalid --created-before format (use RFC3339)", err)
			}
			dtFilter.End = &t
		}
		req.Filter.CreatedAt = dtFilter
	}

	// Pagination
	pageSize, _ := cmd.Flags().GetInt("page-size")
	startIndex, _ := cmd.Flags().GetInt("start-index")
	req.Pagination = &api.Pagination{
		StartIndex: startIndex,
		Count:      pageSize,
	}

	// Sort
	sortBy, _ := cmd.Flags().GetString("sort-by")
	sortDir, _ := cmd.Flags().GetString("sort-dir")
	req.Sort = []api.SortOption{{Column: sortBy, Direction: sortDir}}

	return req
}

func normalizeStatus(s string) string {
	upper := strings.ToUpper(strings.TrimSpace(s))
	if !strings.HasPrefix(upper, "ORCHESTRATION_STATUS_") {
		return "ORCHESTRATION_STATUS_" + upper
	}
	return upper
}

// --- get ---

func newOrchGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <instance-id>",
		Short: "Get orchestration metadata",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			client, err := initClient(cmd)
			if err != nil {
				exitError("init failed", err)
			}
			result, err := client.GetOrchestration(context.Background(), args[0])
			if err != nil {
				exitError("get orchestration failed", err)
			}
			printJSON(result)
		},
	}
}

// --- payloads ---

func newOrchPayloadsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "payloads <instance-id>",
		Short: "Get orchestration input/output/failure details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			client, err := initClient(cmd)
			if err != nil {
				exitError("init failed", err)
			}
			result, err := client.GetOrchestrationPayloads(context.Background(), args[0])
			if err != nil {
				exitError("get payloads failed", err)
			}
			printJSON(result)
		},
	}
}

// --- history ---

func newOrchHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history <instance-id>",
		Short: "Get orchestration execution history",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}

			instanceID := args[0]
			executionID, _ := c.Flags().GetString("execution-id")

			// If no execution ID specified, fetch orchestration to get current executionID
			if executionID == "" {
				orch, err := client.GetOrchestration(context.Background(), instanceID)
				if err != nil {
					exitError("get orchestration failed (needed for execution ID)", err)
				}
				executionID = orch.ExecutionID
			}

			result, err := client.GetOrchestrationHistory(context.Background(), instanceID, executionID)
			if err != nil {
				exitError("get history failed", err)
			}
			printJSON(result)
		},
	}

	cmd.Flags().String("execution-id", "", "Execution ID (auto-detected from current orchestration if omitted)")

	return cmd
}

// --- create ---

func newOrchCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new orchestration instance",
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}

			name, _ := c.Flags().GetString("name")
			if name == "" {
				exitError("--name is required", nil)
			}

			req := &api.CreateOrchestrationRequest{Name: name}

			if v, _ := c.Flags().GetString("instance-id"); v != "" {
				req.InstanceID = v
			}
			if v, _ := c.Flags().GetString("input"); v != "" {
				req.Input = v
			}
			if v, _ := c.Flags().GetString("version"); v != "" {
				req.Version = v
			}
			if v, _ := c.Flags().GetString("scheduled-start"); v != "" {
				t, err := time.Parse(time.RFC3339, v)
				if err != nil {
					exitError("invalid --scheduled-start format (use RFC3339)", err)
				}
				req.ScheduledStartTimestamp = &t
			}
			if v, _ := c.Flags().GetStringSlice("tags"); len(v) > 0 {
				tags := make(map[string]string)
				for _, tag := range v {
					parts := strings.SplitN(tag, "=", 2)
					if len(parts) == 2 {
						tags[parts[0]] = parts[1]
					} else {
						tags[parts[0]] = ""
					}
				}
				req.Tags = tags
			}

			instanceID, err := client.CreateOrchestration(context.Background(), req)
			if err != nil {
				exitError("create orchestration failed", err)
			}
			printJSON(map[string]string{
				"status":     "created",
				"instanceId": instanceID,
			})
		},
	}

	cmd.Flags().String("name", "", "Orchestration name (required)")
	cmd.Flags().String("instance-id", "", "Custom instance ID (auto-generated if omitted)")
	cmd.Flags().String("input", "", "Orchestration input (JSON string)")
	cmd.Flags().String("version", "", "Orchestration version")
	cmd.Flags().String("scheduled-start", "", "Scheduled start time (RFC3339)")
	cmd.Flags().StringSlice("tags", nil, "Tags as key=value pairs")

	return cmd
}

// --- suspend ---

func newOrchSuspendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "suspend <instance-id>",
		Short: "Suspend a running orchestration",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			reason, _ := c.Flags().GetString("reason")
			if err := client.SuspendOrchestration(context.Background(), args[0], reason); err != nil {
				exitError("suspend failed", err)
			}
			printStatus("suspended", args[0])
		},
	}
	cmd.Flags().String("reason", "", "Reason for suspending")
	return cmd
}

// --- resume ---

func newOrchResumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume <instance-id>",
		Short: "Resume a suspended orchestration",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			reason, _ := c.Flags().GetString("reason")
			if err := client.ResumeOrchestration(context.Background(), args[0], reason); err != nil {
				exitError("resume failed", err)
			}
			printStatus("resumed", args[0])
		},
	}
	cmd.Flags().String("reason", "", "Reason for resuming")
	return cmd
}

// --- terminate ---

func newOrchTerminateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "terminate <instance-id>",
		Short: "Terminate an orchestration",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			reason, _ := c.Flags().GetString("reason")
			if err := client.TerminateOrchestration(context.Background(), args[0], reason); err != nil {
				exitError("terminate failed", err)
			}
			printStatus("terminated", args[0])
		},
	}
	cmd.Flags().String("reason", "", "Reason for terminating")
	return cmd
}

// --- force-terminate ---

func newOrchForceTerminateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "force-terminate",
		Short: "Force-terminate one or more orchestrations",
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			ids, _ := c.Flags().GetStringSlice("ids")
			if len(ids) == 0 {
				exitError("--ids is required", nil)
			}
			reason, _ := c.Flags().GetString("reason")
			unsuccessful, err := client.ForceTerminate(context.Background(), ids, reason)
			if err != nil {
				exitError("force-terminate failed", err)
			}
			printJSON(map[string]interface{}{
				"status":       "ok",
				"action":       "force-terminated",
				"requested":    ids,
				"unsuccessful": unsuccessful,
			})
		},
	}
	cmd.Flags().StringSlice("ids", nil, "Instance IDs to force-terminate (comma-separated)")
	cmd.Flags().String("reason", "", "Reason for force-terminating")
	return cmd
}

// --- restart ---

func newOrchRestartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart <instance-id>",
		Short: "Restart an orchestration",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			newID, _ := c.Flags().GetBool("new-id")
			result, err := client.RestartOrchestration(context.Background(), args[0], newID)
			if err != nil {
				exitError("restart failed", err)
			}
			// Try to parse as JSON first
			var parsed interface{}
			if json.Unmarshal([]byte(result), &parsed) == nil {
				printJSON(map[string]interface{}{
					"status": "restarted",
					"id":     args[0],
					"result": parsed,
				})
			} else {
				printJSON(map[string]string{
					"status": "restarted",
					"id":     args[0],
					"result": result,
				})
			}
		},
	}
	cmd.Flags().Bool("new-id", false, "Restart with a new instance ID")
	return cmd
}

// --- rewind ---

func newOrchRewindCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rewind <instance-id>",
		Short: "Rewind a failed orchestration",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			reason, _ := c.Flags().GetString("reason")
			if err := client.RewindOrchestration(context.Background(), args[0], reason); err != nil {
				exitError("rewind failed", err)
			}
			printStatus("rewound", args[0])
		},
	}
	cmd.Flags().String("reason", "", "Reason for rewinding")
	return cmd
}

// --- purge ---

func newOrchPurgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "purge <instance-id> [instance-id...]",
		Short: "Purge (delete) one or more orchestrations",
		Args:  cobra.MinimumNArgs(1),
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			if len(args) == 1 {
				if err := client.PurgeOrchestration(context.Background(), args[0]); err != nil {
					exitError("purge failed", err)
				}
				printStatus("purged", args[0])
			} else {
				if err := client.PurgeOrchestrations(context.Background(), args); err != nil {
					exitError("purge failed", err)
				}
				printJSON(map[string]interface{}{
					"status": "purged",
					"ids":    args,
				})
			}
		},
	}
	return cmd
}

// --- raise-event ---

func newOrchRaiseEventCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "raise-event <instance-id>",
		Short: "Send a named event to an orchestration",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			eventName, _ := c.Flags().GetString("event-name")
			if eventName == "" {
				exitError("--event-name is required", nil)
			}
			data, _ := c.Flags().GetString("data")
			if err := client.RaiseEvent(context.Background(), args[0], eventName, data); err != nil {
				exitError("raise event failed", err)
			}
			printJSON(map[string]string{
				"status":    "ok",
				"action":    "event-raised",
				"id":        args[0],
				"eventName": eventName,
			})
		},
	}
	cmd.Flags().String("event-name", "", "Name of the event to raise (required)")
	cmd.Flags().String("data", "", "Event data (JSON string)")
	return cmd
}
