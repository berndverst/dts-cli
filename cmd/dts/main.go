// dts is a k9s-style terminal UI for Durable Task Scheduler.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Azure/durabletask-cli/internal/api"
	"github.com/Azure/durabletask-cli/internal/app"
	"github.com/Azure/durabletask-cli/internal/auth"
	"github.com/Azure/durabletask-cli/internal/cmd"
	"github.com/Azure/durabletask-cli/internal/config"
	"github.com/Azure/durabletask-cli/internal/ui/views"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	var flagNoSplash bool

	rootCmd := &cobra.Command{
		Use:   "dts",
		Short: "Terminal UI for Durable Task Scheduler",
		Long: `dts is a k9s-style terminal user interface for managing
Durable Task Scheduler orchestrations, entities, schedules, workers, and agents.

Use 'dts exec' for non-interactive, JSON-output commands suitable for
scripts and AI agents.`,
		Version: fmt.Sprintf("%s (commit %s)", version, commit),
		RunE: func(c *cobra.Command, args []string) error {
			flagURL, _ := c.Flags().GetString("url")
			flagTaskHub, _ := c.Flags().GetString("taskhub")
			flagAuthMode, _ := c.Flags().GetString("auth-mode")
			flagTenantID, _ := c.Flags().GetString("tenant-id")

			// Load config
			cfg, err := config.Load()
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
			err = a.Run()

			// Reset terminal state after tview exits.
			// tview/tcell don't always fully restore the terminal on Windows,
			// which leaves formatting (colors, cursor, line wrapping) broken.
			fmt.Print("\033[?25h") // show cursor
			fmt.Print("\033[0m")   // reset attributes
			fmt.Print("\033c")     // full terminal reset (RIS)

			return err
		},
	}

	// Global persistent flags — shared by TUI and all exec subcommands
	rootCmd.PersistentFlags().String("url", "", "DTS endpoint URL (overrides current context)")
	rootCmd.PersistentFlags().String("taskhub", "", "Task hub name (overrides current context)")
	rootCmd.PersistentFlags().String("auth-mode", "", "Authentication mode: default, browser, cli, device, none")
	rootCmd.PersistentFlags().String("tenant-id", "", "Azure AD tenant ID")
	rootCmd.PersistentFlags().String("config", "", "Path to config file")

	// TUI-only local flags
	rootCmd.Flags().BoolVar(&flagNoSplash, "no-splash", false, "Skip the splash screen")

	// Register the exec command family for non-interactive use
	rootCmd.AddCommand(cmd.NewExecCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
