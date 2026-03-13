// Package views provides all TUI views for dts-cli.
package views

import (
	"context"
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/microsoft/durabletask-scheduler/cli/internal/api"
	"github.com/microsoft/durabletask-scheduler/cli/internal/app"
	"github.com/microsoft/durabletask-scheduler/cli/internal/config"
	"github.com/microsoft/durabletask-scheduler/cli/internal/ui/components"
)

// HomeView shows the list of configured DTS endpoints (contexts).
type HomeView struct {
	app          *app.App
	table        *components.ResourceTable
	flex         *tview.Flex
	contextNames []string // stable sorted slice of context names, kept in sync with table rows
}

// NewHomeView creates the home/endpoint selector view.
func NewHomeView(a *app.App) *HomeView {
	v := &HomeView{
		app:   a,
		table: components.NewResourceTable([]string{"Name", "URL", "Task Hub", "Scheduler", "Description"}),
	}
	v.table.SetSelectHandler(func(row int) {
		v.selectContext(row)
	})

	v.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'n':
			v.addEndpoint()
			return nil
		case 'd':
			v.deleteEndpoint()
			return nil
		case ' ':
			row, _ := v.table.GetSelection()
			v.table.ToggleRowSelection(row)
			return nil
		}
		return event
	})

	header := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[aqua::b] Durable Task Scheduler CLI [-:-:-]\n [gray]Select a task hub to connect to, or press [aqua]<n>[-] to add a new endpoint[-]")

	v.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 3, 0, false).
		AddItem(v.table, 0, 1, true)

	return v
}

func (v *HomeView) Name() string               { return "home" }
func (v *HomeView) Primitive() tview.Primitive { return v.flex }
func (v *HomeView) Crumbs() []string           { return []string{"Home"} }
func (v *HomeView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "Enter", Description: "Connect"},
		{Key: "n", Description: "Add endpoint"},
		{Key: "d", Description: "Delete"},
		{Key: "?", Description: "Help"},
	}
}

func (v *HomeView) Init(ctx context.Context) {
	v.app.QueueUpdateDraw(func() {
		v.table.ClearData()
		cfg := v.app.Config

		// Build a sorted slice of context names so that table row indices
		// map deterministically to contexts across Init, selectContext and deleteEndpoint.
		names := make([]string, 0, len(cfg.Contexts))
		for name := range cfg.Contexts {
			names = append(names, name)
		}
		sort.Strings(names)
		v.contextNames = names

		for row, name := range v.contextNames {
			c := cfg.Contexts[name]
			displayName := name
			if name == cfg.CurrentContext {
				displayName = "● " + name
			}
			v.table.SetDataRow(row, displayName, c.URL, c.TaskHub, c.Scheduler, c.Description)
		}

		if len(v.contextNames) == 0 {
			// Show empty state
			v.table.SetDataRow(0, "(no endpoints configured)", "", "", "", "Press 'n' to add one")
		}
	})
}

func (v *HomeView) selectContext(row int) {
	if row >= len(v.contextNames) {
		return
	}

	name := v.contextNames[row]
	cfg := v.app.Config
	cfg.CurrentContext = name
	_ = cfg.Save()

	ctx := cfg.Contexts[name]
	v.app.Client = createClient(ctx, v.app)
	v.app.FlashSuccess("Connected to " + name)
	v.app.NavigateToResource("orchestrations")
}

func (v *HomeView) addEndpoint() {
	fields := []components.FormField{
		{Label: "Name", Default: "", Width: 30},
		{Label: "URL", Default: "https://", Width: 50},
		{Label: "Task Hub", Default: "default", Width: 30},
		{Label: "Subscription", Default: "", Width: 40},
		{Label: "Scheduler", Default: "", Width: 30},
		{Label: "Tenant ID", Default: "", Width: 40},
	}

	components.MultiInputDialog(v.app.TviewApp(), v.app.Pages(), "Add Endpoint", fields, func(values map[string]string) {
		name := values["Name"]
		if name == "" {
			v.app.FlashError("Name is required")
			return
		}

		ctx := &config.Context{
			URL:          values["URL"],
			TaskHub:      values["Task Hub"],
			Subscription: values["Subscription"],
			Scheduler:    values["Scheduler"],
			TenantID:     values["Tenant ID"],
		}
		v.app.Config.AddContext(name, ctx)
		if err := v.app.Config.Save(); err != nil {
			v.app.FlashError("Failed to save: " + err.Error())
			return
		}
		v.app.FlashSuccess("Added endpoint: " + name)
		v.Init(context.Background())
	})
}

func (v *HomeView) deleteEndpoint() {
	row, _ := v.table.GetSelection()
	if row <= 0 {
		return
	}

	dataRow := row - 1
	if dataRow >= len(v.contextNames) {
		return
	}
	name := v.contextNames[dataRow]

	cfg := v.app.Config
	v.app.ShowConfirm("Delete Endpoint", fmt.Sprintf("Delete endpoint '%s'?", name), func() {
		cfg.RemoveContext(name)
		_ = cfg.Save()
		v.app.FlashSuccess("Deleted endpoint: " + name)
		v.Init(context.Background())
	})
}

func createClient(ctx *config.Context, a *app.App) *api.Client {
	// The client is re-created with the current auth; in practice
	// the token provider is shared and persisted across client instances.
	// For now, we create a client without re-initializing auth.
	if a.Client != nil {
		return a.Client
	}
	return nil
}
