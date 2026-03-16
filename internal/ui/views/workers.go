package views

import (
	"context"
	"fmt"

	"github.com/rivo/tview"

	"github.com/Azure/durabletask-cli/internal/app"
	"github.com/Azure/durabletask-cli/internal/ui/components"
	"github.com/Azure/durabletask-cli/internal/util"
)

// WorkersView shows connected workers.
type WorkersView struct {
	app   *app.App
	table *components.ResourceTable
	flex  *tview.Flex
	info  *tview.TextView
}

// NewWorkersView creates the workers list view.
func NewWorkersView(a *app.App) *WorkersView {
	v := &WorkersView{
		app:   a,
		table: components.NewResourceTable([]string{"Worker ID", "Orchestrations", "Activities", "Entities", "Saturation"}),
		info:  tview.NewTextView().SetDynamicColors(true),
	}
	v.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.info, 1, 0, false).
		AddItem(v.table, 0, 1, true)

	return v
}

func (v *WorkersView) Name() string               { return "workers" }
func (v *WorkersView) Primitive() tview.Primitive { return v.flex }
func (v *WorkersView) Crumbs() []string {
	ctx := v.app.Config.CurrentContext
	return []string{ctx, "Workers"}
}
func (v *WorkersView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "r", Description: "Refresh"},
	}
}

func (v *WorkersView) Init(ctx context.Context) {
	if v.app.Client == nil {
		return
	}

	// Show loading indicator immediately so the UI feels responsive
	v.app.QueueUpdateDraw(func() {
		v.info.SetText(" [gray]Loading workers...[-]")
	})

	result, err := v.app.Client.ListWorkers(ctx)
	if err != nil {
		v.app.QueueUpdateDraw(func() {
			v.app.FlashError("Failed: " + err.Error())
		})
		return
	}

	v.app.QueueUpdateDraw(func() {
		v.info.SetText(fmt.Sprintf(" [white]Workers[-] [gray](%d connected)[-]", len(result.Workers)))
		v.table.ClearData()

		for i, w := range result.Workers {
			v.table.SetDataRow(i,
				util.Truncate(w.WorkerID, 50),
				fmt.Sprintf("%d / %d", w.ActiveOrchestrationsCount, w.MaxOrchestrationsCount),
				fmt.Sprintf("%d / %d", w.ActiveActivitiesCount, w.MaxActivitiesCount),
				fmt.Sprintf("%d / %d", w.ActiveEntitiesCount, w.MaxEntitiesCount),
				util.SaturationBar(w.ActiveActivitiesCount, w.MaxActivitiesCount, 10),
			)
		}
	})
}
