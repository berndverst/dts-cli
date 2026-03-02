package views

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/microsoft/durabletask-scheduler/cli/internal/api"
	"github.com/microsoft/durabletask-scheduler/cli/internal/app"
	"github.com/microsoft/durabletask-scheduler/cli/internal/ui/components"
	"github.com/microsoft/durabletask-scheduler/cli/internal/util"
)

// EntitiesView shows the durable entities list.
type EntitiesView struct {
	app   *app.App
	table *components.ResourceTable
	flex  *tview.Flex
	info  *tview.TextView

	data       []api.Entity
	pageIndex  int
	totalCount int
}

// NewEntitiesView creates the entities list view.
func NewEntitiesView(a *app.App) *EntitiesView {
	v := &EntitiesView{
		app:   a,
		table: components.NewResourceTable([]string{"Entity Name", "Entity Key", "Last Modified", "Locked By"}),
		info:  tview.NewTextView().SetDynamicColors(true),
	}
	v.table.SetSelectHandler(func(row int) {
		if row < len(v.data) {
			e := v.data[row]
			v.app.Navigate(NewEntityDetailView(v.app, e.InstanceID))
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
		case 'd':
			v.deleteSelected()
			return nil
		case 'p':
			v.deleteSelected() // purge is same as delete for entities
			return nil
		case '[':
			v.prevPage()
			return nil
		case ']':
			v.nextPage()
			return nil
		}
		return event
	})

	v.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.info, 1, 0, false).
		AddItem(v.table, 0, 1, true)

	return v
}

func (v *EntitiesView) Name() string               { return "entities" }
func (v *EntitiesView) Primitive() tview.Primitive { return v.flex }
func (v *EntitiesView) Crumbs() []string {
	ctx := v.app.Config.CurrentContext
	return []string{ctx, "Entities"}
}
func (v *EntitiesView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "Enter", Description: "Detail"},
		{Key: "d", Description: "Delete"},
		{Key: "Space", Description: "Select"},
		{Key: "[/]", Description: "Page"},
	}
}

func (v *EntitiesView) Init(ctx context.Context) {
	if v.app.Client == nil {
		return
	}

	// Show loading indicator immediately so the UI feels responsive
	v.app.QueueUpdateDraw(func() {
		v.info.SetText(" [gray]Loading entities...[-]")
	})

	req := &api.QueryEntitiesRequest{
		Pagination: &api.Pagination{
			StartIndex: v.pageIndex * v.app.Config.Settings.PageSize,
			Count:      v.app.Config.Settings.PageSize,
		},
		FetchTotalCount: true,
	}

	// Build exclusion filter for internal entities
	var excludes []api.StringFilter
	if v.app.Config.Settings.HideAgentsFromEntities {
		excludes = append(excludes, api.StringFilter{Value: "@agent@"})
	}
	if v.app.Config.Settings.HideSchedulesFromEntities {
		excludes = append(excludes, api.StringFilter{Value: "@schedule@"})
	}
	if len(excludes) > 0 {
		if req.Filter == nil {
			req.Filter = &api.EntityFilter{}
		}
		req.Filter.ExcludeNameStartsWith = excludes
	}

	result, err := v.app.Client.QueryEntities(ctx, req)
	if err != nil {
		v.app.QueueUpdateDraw(func() {
			v.app.FlashError("Failed: " + err.Error())
		})
		return
	}

	v.data = result.Entities
	v.totalCount = result.TotalCount

	v.app.QueueUpdateDraw(func() {
		v.info.SetText(fmt.Sprintf(" [white]Entities[-] [gray](%d shown, %d total)[-]", len(v.data), v.totalCount))
		v.renderTable()
	})
}

func (v *EntitiesView) renderTable() {
	v.table.ClearData()
	local := v.app.Config.UseLocalTime()

	for i, e := range v.data {
		name := api.ParseEntityName(e.InstanceID)
		key := api.ParseEntityKey(e.InstanceID)
		locked := ""
		if e.LockedBy != "" {
			locked = "[yellow]" + e.LockedBy + "[-]"
		}

		v.table.SetDataRow(i,
			name,
			key,
			util.FormatTimestamp(e.LastModifiedTime, local),
			locked,
		)
	}
}

func (v *EntitiesView) getSelectedIDs() []string {
	selected := v.table.GetSelectedRows()
	if len(selected) == 0 {
		row, _ := v.table.GetSelection()
		dataRow := row - 1
		if dataRow >= 0 && dataRow < len(v.data) {
			return []string{v.data[dataRow].InstanceID}
		}
		return nil
	}
	ids := make([]string, 0, len(selected))
	for _, r := range selected {
		if r < len(v.data) {
			ids = append(ids, v.data[r].InstanceID)
		}
	}
	return ids
}

func (v *EntitiesView) deleteSelected() {
	ids := v.getSelectedIDs()
	if len(ids) == 0 {
		v.app.FlashInfo("No entities selected")
		return
	}
	v.app.ShowConfirm("Delete", fmt.Sprintf("Delete %d entity(s)?", len(ids)), func() {
		go func() {
			err := v.app.Client.DeleteEntities(context.Background(), ids)
			v.app.QueueUpdateDraw(func() {
				if err != nil {
					v.app.FlashError("Delete failed: " + err.Error())
				} else {
					v.app.FlashSuccess(fmt.Sprintf("Deleted %d entity(s)", len(ids)))
				}
				v.table.ClearSelection()
			})
			v.Init(context.Background())
		}()
	})
}

func (v *EntitiesView) nextPage() {
	if (v.pageIndex+1)*v.app.Config.Settings.PageSize < v.totalCount {
		v.pageIndex++
		go func() {
			v.Init(context.Background())
		}()
	}
}

func (v *EntitiesView) prevPage() {
	if v.pageIndex > 0 {
		v.pageIndex--
		go func() {
			v.Init(context.Background())
		}()
	}
}
