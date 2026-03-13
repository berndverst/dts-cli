package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/microsoft/durabletask-scheduler/cli/internal/api"
)

func newSchedulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schedules",
		Aliases: []string{"sched"},
		Short:   "Manage schedules",
	}

	cmd.AddCommand(
		newScheduleListCmd(),
		newScheduleCreateCmd(),
		newScheduleDeleteCmd(),
		newSchedulePauseCmd(),
		newScheduleResumeCmd(),
	)

	return cmd
}

// --- list ---

func newScheduleListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List schedules",
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			token, _ := c.Flags().GetString("continuation-token")
			result, err := client.ListSchedules(context.Background(), token)
			if err != nil {
				exitError("list schedules failed", err)
			}
			printJSON(result)
		},
	}
	cmd.Flags().String("continuation-token", "", "Continuation token for pagination")
	return cmd
}

// --- create ---

func newScheduleCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new schedule",
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}

			scheduleID, _ := c.Flags().GetString("schedule-id")
			if scheduleID == "" {
				exitError("--schedule-id is required", nil)
			}
			orchName, _ := c.Flags().GetString("orchestration-name")
			if orchName == "" {
				exitError("--orchestration-name is required", nil)
			}
			interval, _ := c.Flags().GetString("interval")
			if interval == "" {
				exitError("--interval is required", nil)
			}

			req := &api.CreateScheduleRequest{
				ScheduleID:        scheduleID,
				OrchestrationName: orchName,
				Interval:          interval,
			}

			if v, _ := c.Flags().GetString("input"); v != "" {
				req.OrchestrationInput = v
			}
			if v, _ := c.Flags().GetString("instance-id"); v != "" {
				req.OrchestrationInstanceID = v
			}
			if v, _ := c.Flags().GetString("start-at"); v != "" {
				t, err := time.Parse(time.RFC3339, v)
				if err != nil {
					exitError("invalid --start-at format (use RFC3339)", err)
				}
				req.StartAt = &t
			}
			if v, _ := c.Flags().GetString("end-at"); v != "" {
				t, err := time.Parse(time.RFC3339, v)
				if err != nil {
					exitError("invalid --end-at format (use RFC3339)", err)
				}
				req.EndAt = &t
			}
			if v, _ := c.Flags().GetBool("start-immediately-if-late"); v {
				req.StartImmediatelyIfLate = true
			}

			if err := client.CreateSchedule(context.Background(), req); err != nil {
				exitError("create schedule failed", err)
			}
			printJSON(map[string]string{
				"status":     "created",
				"scheduleId": scheduleID,
			})
		},
	}

	cmd.Flags().String("schedule-id", "", "Schedule ID (required)")
	cmd.Flags().String("orchestration-name", "", "Orchestration name to schedule (required)")
	cmd.Flags().String("interval", "", "Schedule interval, e.g. PT1H, PT30M (ISO 8601 duration, required)")
	cmd.Flags().String("input", "", "Orchestration input (JSON string)")
	cmd.Flags().String("instance-id", "", "Custom orchestration instance ID")
	cmd.Flags().String("start-at", "", "Schedule start time (RFC3339)")
	cmd.Flags().String("end-at", "", "Schedule end time (RFC3339)")
	cmd.Flags().Bool("start-immediately-if-late", false, "Start immediately if the schedule is late")

	return cmd
}

// --- delete ---

func newScheduleDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <schedule-id>",
		Short: "Delete a schedule",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			if err := client.DeleteSchedule(context.Background(), args[0]); err != nil {
				exitError("delete schedule failed", err)
			}
			printStatus("deleted", args[0])
		},
	}
}

// --- pause ---

func newSchedulePauseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pause <schedule-id>",
		Short: "Pause a schedule",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			if err := client.PauseSchedule(context.Background(), args[0]); err != nil {
				exitError("pause schedule failed", err)
			}
			printStatus("paused", args[0])
		},
	}
}

// --- resume ---

func newScheduleResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <schedule-id>",
		Short: "Resume a paused schedule",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			if err := client.ResumeSchedule(context.Background(), args[0]); err != nil {
				exitError("resume schedule failed", err)
			}
			printStatus("resumed", args[0])
		},
	}
}
