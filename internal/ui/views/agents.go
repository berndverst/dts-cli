package views

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Azure/durabletask-cli/internal/api"
	"github.com/Azure/durabletask-cli/internal/app"
	"github.com/Azure/durabletask-cli/internal/ui/components"
	"github.com/Azure/durabletask-cli/internal/util"
)

// AgentsView shows agent sessions (preview feature).
type AgentsView struct {
	app   *app.App
	table *components.ResourceTable
	flex  *tview.Flex
	info  *tview.TextView

	data []api.AgentEntity
}

// NewAgentsView creates the agents list view.
func NewAgentsView(a *app.App) *AgentsView {
	v := &AgentsView{
		app:   a,
		table: components.NewResourceTable([]string{"Agent Name", "Session ID", "Last Modified"}),
		info:  tview.NewTextView().SetDynamicColors(true),
	}
	v.table.SetSelectHandler(func(row int) {
		if row < len(v.data) {
			agent := v.data[row]
			v.app.Navigate(NewAgentSessionView(v.app, agent.Name, agent.SessionID, agent.EntityID))
		}
	})

	v.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlA:
			v.table.SelectAllRows(true)
			return nil
		}
		switch event.Rune() {
		case ' ':
			row, _ := v.table.GetSelection()
			v.table.ToggleRowSelection(row)
			return nil
		case 'n':
			v.startSession()
			return nil
		case 'd':
			v.deleteSelected()
			return nil
		}
		return event
	})

	v.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.info, 1, 0, false).
		AddItem(v.table, 0, 1, true)

	return v
}

func (v *AgentsView) Name() string               { return "agents" }
func (v *AgentsView) Primitive() tview.Primitive { return v.flex }
func (v *AgentsView) Crumbs() []string {
	ctx := v.app.Config.CurrentContext
	return []string{ctx, "Agents (Preview)"}
}
func (v *AgentsView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "Enter", Description: "Open session"},
		{Key: "n", Description: "New session"},
		{Key: "d", Description: "Delete"},
		{Key: "Space", Description: "Select"},
	}
}

func (v *AgentsView) Init(ctx context.Context) {
	if v.app.Client == nil {
		return
	}

	// Show loading indicator immediately so the UI feels responsive
	v.app.QueueUpdateDraw(func() {
		v.info.SetText(" [gray]Loading agent sessions...[-]")
	})

	result, err := v.app.Client.ListAgentSessions(ctx, v.app.Config.Settings.PageSize, 0)
	if err != nil {
		v.app.QueueUpdateDraw(func() {
			v.info.SetText(" [red]Error: " + tview.Escape(err.Error()) + "[-]")
			v.app.FlashError("Failed: " + err.Error())
		})
		return
	}

	// Convert entities to AgentEntity
	agents := make([]api.AgentEntity, 0, len(result.Entities))
	for _, e := range result.Entities {
		agents = append(agents, *api.ParseAgentEntity(&e))
	}

	v.data = agents

	v.app.QueueUpdateDraw(func() {
		v.info.SetText(fmt.Sprintf(" [white]Agent Sessions[-] [gray::i](Preview)[-:-:-] [gray](%d sessions)[-]", len(v.data)))
		v.renderTable()
	})
}

func (v *AgentsView) renderTable() {
	v.table.ClearData()
	local := v.app.Config.UseLocalTime()

	for i, agent := range v.data {
		v.table.SetDataRow(i,
			agent.Name,
			agent.SessionID,
			util.FormatTimestamp(agent.LastModified, local),
		)
	}
}

func (v *AgentsView) startSession() {
	fields := []components.FormField{
		{Label: "Agent Name", Default: "", Width: 40},
		{Label: "Session ID (optional)", Default: "", Width: 40},
		{Label: "Initial Prompt", Default: "", Width: 60},
	}

	components.MultiInputDialog(v.app.TviewApp(), v.app.Pages(), "Start Agent Session", fields, func(values map[string]string) {
		name := values["Agent Name"]
		if name == "" {
			v.app.FlashError("Agent Name is required")
			return
		}

		go func() {
			sessionID := values["Session ID (optional)"]
			prompt := values["Initial Prompt"]
			sessionID, err := v.app.Client.StartAgentSession(context.Background(), name, sessionID, prompt)
			if err != nil {
				v.app.QueueUpdateDraw(func() {
					v.app.FlashError("Start failed: " + err.Error())
				})
				return
			}

			entityID := fmt.Sprintf("@agent@%s@%s", name, sessionID)
			v.app.QueueUpdateDraw(func() {
				v.app.FlashSuccess("Session started: " + sessionID)
				v.app.Navigate(NewAgentSessionView(v.app, name, sessionID, entityID))
			})
		}()
	})
}

func (v *AgentsView) getSelectedIDs() []string {
	selected := v.table.GetSelectedRows()
	if len(selected) == 0 {
		row, _ := v.table.GetSelection()
		dataRow := row - 1
		if dataRow >= 0 && dataRow < len(v.data) {
			return []string{v.data[dataRow].EntityID}
		}
		return nil
	}
	ids := make([]string, 0, len(selected))
	for _, r := range selected {
		if r < len(v.data) {
			ids = append(ids, v.data[r].EntityID)
		}
	}
	return ids
}

func (v *AgentsView) deleteSelected() {
	ids := v.getSelectedIDs()
	if len(ids) == 0 {
		v.app.FlashInfo("No sessions selected")
		return
	}
	v.app.ShowConfirm("Delete", fmt.Sprintf("Delete %d session(s)?", len(ids)), func() {
		go func() {
			err := v.app.Client.DeleteAgentSessions(context.Background(), ids)
			v.app.QueueUpdateDraw(func() {
				if err != nil {
					v.app.FlashError("Delete failed: " + err.Error())
				} else {
					v.app.FlashSuccess(fmt.Sprintf("Deleted %d session(s)", len(ids)))
				}
				v.table.ClearSelection()
			})
			v.Init(context.Background())
		}()
	})
}
