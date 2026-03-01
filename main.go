// dts-cli is a k9s-style terminal UI for Durable Task Scheduler.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/microsoft/durabletask-scheduler/cli/internal/api"
	"github.com/microsoft/durabletask-scheduler/cli/internal/app"
	"github.com/microsoft/durabletask-scheduler/cli/internal/auth"
	"github.com/microsoft/durabletask-scheduler/cli/internal/config"
	"github.com/microsoft/durabletask-scheduler/cli/internal/ui/views"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	var (
		flagURL      string
		flagTaskHub  string
		flagAuthMode string
		flagTenantID string
		flagConfig   string
		flagNoSplash bool
	)

	rootCmd := &cobra.Command{
		Use:   "dts-cli",
		Short: "Terminal UI for Durable Task Scheduler",
		Long: `dts-cli is a k9s-style terminal user interface for managing
Durable Task Scheduler orchestrations, entities, schedules, workers, and agents.`,
		Version: fmt.Sprintf("%s (commit %s)", version, commit),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			var cfg *config.Config
			var err error

			if flagConfig != "" {
				// TODO: support custom config path
				cfg, err = config.Load()
			} else {
				cfg, err = config.Load()
			}
			if err != nil {
				cfg = config.DefaultConfig()
			}

			// CLI flags override config settings
			if flagAuthMode != "" {
				cfg.Settings.AuthMode = flagAuthMode
			}

			// Determine connection details
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

			// Create API client (may be nil if no URL configured yet)
			var client *api.Client
			if url != "" {
				tp, tpErr := auth.NewTokenProvider(cfg.Settings.AuthMode, tenantID)
				if tpErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: auth init failed: %v\n", tpErr)
					// Continue anyway - home view allows adding contexts
				}
				client = api.NewClient(url, taskHub, tp)
			}

			// Create and configure application
			a := app.New(cfg, client)
			a.ViewFactory = &views.Factory{}

			// Set title bar from resolved connection details
			if url != "" || taskHub != "" {
				a.SetTitleContext(url, taskHub)
			}

			// Navigate to starting view (with optional splash screen)
			navigate := func() {
				if url != "" && taskHub != "" {
					a.NavigateToResource("orchestrations")
				} else {
					a.NavigateToResource("home")
				}
			}

			if flagNoSplash {
				navigate()
			} else {
				a.ShowSplash(navigate)
			}

			// Run the TUI
			return a.Run()
		},
	}

	rootCmd.Flags().StringVar(&flagURL, "url", "", "DTS endpoint URL (overrides current context)")
	rootCmd.Flags().StringVar(&flagTaskHub, "taskhub", "", "Task hub name (overrides current context)")
	rootCmd.Flags().StringVar(&flagAuthMode, "auth-mode", "", "Authentication mode: default, browser, cli, device")
	rootCmd.Flags().StringVar(&flagTenantID, "tenant-id", "", "Azure AD tenant ID")
	rootCmd.Flags().StringVar(&flagConfig, "config", "", "Path to config file")
	rootCmd.Flags().BoolVar(&flagNoSplash, "no-splash", false, "Skip the splash screen")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
