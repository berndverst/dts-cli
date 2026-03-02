// Package cmd implements non-interactive CLI commands for dts-cli.
// All commands output JSON to stdout and are suitable for use in scripts and by AI agents.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/microsoft/durabletask-scheduler/cli/internal/api"
	"github.com/microsoft/durabletask-scheduler/cli/internal/auth"
	"github.com/microsoft/durabletask-scheduler/cli/internal/config"
)

// initClient creates an authenticated API client from persistent flags and config.
func initClient(cmd *cobra.Command) (*api.Client, error) {
	flagURL, _ := cmd.Flags().GetString("url")
	flagTaskHub, _ := cmd.Flags().GetString("taskhub")
	flagAuthMode, _ := cmd.Flags().GetString("auth-mode")
	flagTenantID, _ := cmd.Flags().GetString("tenant-id")

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	if flagAuthMode != "" {
		cfg.Settings.AuthMode = flagAuthMode
	}

	url := flagURL
	taskHub := flagTaskHub
	tenantID := flagTenantID

	if ctx := cfg.CurrentCtx(); ctx != nil {
		if url == "" {
			url = ctx.URL
		}
		if taskHub == "" {
			taskHub = ctx.TaskHub
		}
		if tenantID == "" {
			tenantID = ctx.TenantID
		}
	}

	if url == "" {
		return nil, fmt.Errorf("--url is required (or set a context with a URL)")
	}
	if taskHub == "" {
		return nil, fmt.Errorf("--taskhub is required (or set a context with a task hub)")
	}

	tp, err := auth.NewTokenProvider(cfg.Settings.AuthMode, tenantID)
	if err != nil {
		return nil, fmt.Errorf("auth init failed: %w", err)
	}

	return api.NewClient(url, taskHub, tp), nil
}

// printJSON marshals v as indented JSON and writes it to stdout.
func printJSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		exitError("failed to marshal JSON", err)
		return
	}
	fmt.Fprintln(os.Stdout, string(data))
}

// exitError prints a JSON error object to stderr and exits with code 1.
func exitError(msg string, err error) {
	detail := msg
	if err != nil {
		detail = fmt.Sprintf("%s: %v", msg, err)
	}
	errObj, _ := json.Marshal(map[string]string{"error": detail})
	fmt.Fprintln(os.Stderr, string(errObj))
	os.Exit(1)
}

// printStatus prints a simple JSON status message to stdout.
func printStatus(action, id string) {
	printJSON(map[string]string{
		"status": "ok",
		"action": action,
		"id":     id,
	})
}
