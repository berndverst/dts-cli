package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/microsoft/durabletask-scheduler/cli/internal/api"
	"github.com/microsoft/durabletask-scheduler/cli/internal/app"
	"github.com/microsoft/durabletask-scheduler/cli/internal/ui/components"
	"github.com/microsoft/durabletask-scheduler/cli/internal/util"
)

// OrchestrationsView shows the orchestrations list with filtering, sorting, and actions.
type OrchestrationsView struct {
	app     *app.App
	table   *components.ResourceTable
	flex    *tview.Flex
	summary *tview.TextView

	// State
	data         []api.Orchestration
	trivia       *api.OrchestrationTrivia
	pageIndex    int
	pageSize     int
	filter       *api.OrchestrationFilter
	sortCol      string
	sortDir      string
	statusFilter string // Quick filter: "" = all
}

// NewOrchestrationsView creates the orchestrations list view.
func NewOrchestrationsView(a *app.App) *OrchestrationsView {
	v := &OrchestrationsView{
		app:      a,
		table:    components.NewResourceTable([]string{"Instance ID", "Name", "Version", "Created", "Last Updated", "Duration", "Status", "Tags"}),
		summary:  tview.NewTextView().SetDynamicColors(true),
		pageSize: a.Config.Settings.PageSize,
		sortCol:  api.SortByLastUpdatedAt,
		sortDir:  api.SortDescending,
	}
	v.table.SetSelectHandler(func(row int) {
		if row < len(v.data) {
			orch := v.data[row]
			v.app.Navigate(NewOrchestrationDetailView(v.app, orch.InstanceID, orch.ExecutionID))
		}
	})

	v.table.SetSortHandler(func(col int, asc bool) {
		columns := []string{api.SortByOrchestrationID, api.SortByName, api.SortByVersion, api.SortByCreatedAt, api.SortByLastUpdatedAt, "", api.SortByStatus, api.SortByTags}
		if col < len(columns) && columns[col] != "" {
			v.sortCol = columns[col]
			if asc {
				v.sortDir = api.SortAscending
			} else {
				v.sortDir = api.SortDescending
			}
			v.pageIndex = 0
			go func() {
				v.Init(context.Background())
			}()
		}
	})

	v.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlA:
			v.table.SelectAllRows(true)
			return nil
		case tcell.KeyCtrlK:
			v.bulkForceTerminate()
			return nil
		}
		switch event.Rune() {
		case ' ':
			row, _ := v.table.GetSelection()
			v.table.ToggleRowSelection(row)
			return nil
		case 'n':
			v.createOrchestration()
			return nil
		case 's':
			v.bulkAction("Suspend", v.bulkSuspend)
			return nil
		case 'u':
			v.bulkAction("Resume", v.bulkResume)
			return nil
		case 'k':
			v.bulkAction("Terminate", v.bulkTerminate)
			return nil
		case 'x':
			v.bulkAction("Restart", v.bulkRestart)
			return nil
		case 'p':
			v.bulkAction("Purge", v.bulkPurge)
			return nil
		case 'd':
			v.describeSelected()
			return nil
		case '1':
			v.setQuickFilter("")
			return nil
		case '2':
			v.setQuickFilter(api.StatusRunning)
			return nil
		case '3':
			v.setQuickFilter(api.StatusCompleted)
			return nil
		case '4':
			v.setQuickFilter(api.StatusFailed)
			return nil
		case '5':
			v.setQuickFilter(api.StatusPending)
			return nil
		case '[':
			v.prevPage()
			return nil
		case ']':
			v.nextPage()
			return nil
		case 'o':
			// Columns: 0=ID, 1=Name, 2=Version, 3=Created, 4=Updated, 5=Duration, 6=Status, 7=Tags
			// Duration (5) is computed client-side, not sortable server-side
			v.table.NextSortableColumn(map[int]bool{5: true})
			return nil
		case 'O':
			// Toggle sort direction
			if v.sortDir == api.SortAscending {
				v.sortDir = api.SortDescending
			} else {
				v.sortDir = api.SortAscending
			}
			v.table.SetSortDirection(v.sortDir == api.SortAscending)
			v.pageIndex = 0
			go func() {
				v.Init(context.Background())
			}()
			return nil
		}
		return event
	})

	v.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.summary, 2, 0, false).
		AddItem(v.table, 0, 1, true)

	return v
}

func (v *OrchestrationsView) Name() string               { return "orchestrations" }
func (v *OrchestrationsView) Primitive() tview.Primitive { return v.flex }
func (v *OrchestrationsView) Crumbs() []string {
	ctx := v.app.Config.CurrentContext
	return []string{ctx, "Orchestrations"}
}
func (v *OrchestrationsView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "Enter", Description: "Detail"},
		{Key: "n", Description: "New"},
		{Key: "s", Description: "Suspend"},
		{Key: "u", Description: "Resume"},
		{Key: "k", Description: "Terminate"},
		{Key: "p", Description: "Purge"},
		{Key: "o", Description: "Sort"},
		{Key: "O", Description: "Asc/Desc"},
		{Key: "1-5", Description: "Filter"},
		{Key: "[/]", Description: "Page"},
	}
}

func (v *OrchestrationsView) Init(ctx context.Context) {
	if v.app.Client == nil {
		return
	}

	// Show loading indicator immediately so the UI feels responsive
	v.app.QueueUpdateDraw(func() {
		v.summary.SetText(" [gray]Loading orchestrations...[-]")
	})

	req := &api.QueryOrchestrationsRequest{
		Pagination: &api.Pagination{
			StartIndex: v.pageIndex * v.pageSize,
			Count:      v.pageSize,
		},
		Sort: []api.SortOption{
			{Column: v.sortCol, Direction: v.sortDir},
		},
		Fields: api.DefaultOrchestrationFields,
	}

	// Apply status quick filter
	if v.statusFilter != "" {
		if req.Filter == nil {
			req.Filter = &api.OrchestrationFilter{}
		}
		req.Filter.OrchestrationStatus = &api.StatusFilter{
			Status: []string{v.statusFilter},
		}
	}

	// Apply text filter
	if v.filter != nil {
		if req.Filter == nil {
			req.Filter = &api.OrchestrationFilter{}
		}
		if v.filter.Name != nil {
			req.Filter.Name = v.filter.Name
		}
		if v.filter.OrchestrationID != nil {
			req.Filter.OrchestrationID = v.filter.OrchestrationID
		}
	}

	result, err := v.app.Client.QueryOrchestrations(ctx, req)
	if err != nil {
		v.app.QueueUpdateDraw(func() {
			v.app.FlashError("Failed: " + err.Error())
		})
		return
	}

	v.data = result.Orchestrations
	v.trivia = result.Trivia

	v.app.QueueUpdateDraw(func() {
		v.renderSummary()
		v.renderTable()
	})
}

func (v *OrchestrationsView) renderSummary() {
	if v.trivia == nil {
		v.summary.SetText(" [gray]Loading...[-]")
		return
	}
	t := v.trivia
	active := func(label string, count int, status string) string {
		if v.statusFilter == status {
			return fmt.Sprintf("[aqua::b]%s:%d[-:-:-]", label, count)
		}
		return fmt.Sprintf("[white]%s:[gray]%d[-]", label, count)
	}
	text := fmt.Sprintf(" %s │ %s │ %s │ %s │ %s  [gray]Page %d (showing %d of %d)[-]",
		active("All", t.TotalCount, ""),
		active("Running", t.RunningCount, api.StatusRunning),
		active("Completed", t.CompletedCount, api.StatusCompleted),
		active("Failed", t.FailedCount, api.StatusFailed),
		active("Pending", t.PendingCount, api.StatusPending),
		v.pageIndex+1, len(v.data), t.TotalCount,
	)
	v.summary.SetText(text)
}

func (v *OrchestrationsView) renderTable() {
	v.table.ClearData()
	local := v.app.Config.UseLocalTime()

	for i, orch := range v.data {
		duration := "-"
		if orch.CompletedTimestamp != nil {
			duration = util.FormatDurationBetween(orch.CreatedTimestamp, *orch.CompletedTimestamp)
		}

		statusColor := util.StatusColor(orch.OrchestrationStatus)
		statusName := util.StatusShortName(orch.OrchestrationStatus)

		tags := ""
		if len(orch.Tags) > 0 {
			parts := make([]string, 0, len(orch.Tags))
			for k, val := range orch.Tags {
				parts = append(parts, k+"="+val)
			}
			tags = strings.Join(parts, ", ")
		}

		v.table.SetDataRow(i,
			orch.InstanceID,
			orch.Name,
			orch.Version,
			util.FormatTimestamp(orch.CreatedTimestamp, local),
			util.FormatTimestamp(orch.LastUpdatedTimestamp, local),
			duration,
			statusColor+statusName+"[-]",
			tags,
		)
	}
}

func (v *OrchestrationsView) setQuickFilter(status string) {
	v.statusFilter = status
	v.pageIndex = 0
	go func() {
		v.Init(context.Background())
	}()
}

func (v *OrchestrationsView) nextPage() {
	if v.trivia != nil && (v.pageIndex+1)*v.pageSize < v.trivia.TotalCount {
		v.pageIndex++
		go func() {
			v.Init(context.Background())
		}()
	}
}

func (v *OrchestrationsView) prevPage() {
	if v.pageIndex > 0 {
		v.pageIndex--
		go func() {
			v.Init(context.Background())
		}()
	}
}

func (v *OrchestrationsView) describeSelected() {
	row, _ := v.table.GetSelection()
	dataRow := row - 1
	if dataRow >= 0 && dataRow < len(v.data) {
		orch := v.data[dataRow]
		v.app.Navigate(NewOrchestrationDetailView(v.app, orch.InstanceID, orch.ExecutionID))
	}
}

func (v *OrchestrationsView) getSelectedInstanceIDs() []string {
	selected := v.table.GetSelectedRows()
	if len(selected) == 0 {
		// Use current row
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

func (v *OrchestrationsView) bulkAction(action string, fn func(ids []string)) {
	ids := v.getSelectedInstanceIDs()
	if len(ids) == 0 {
		v.app.FlashInfo("No orchestrations selected")
		return
	}
	v.app.ShowConfirm(action, fmt.Sprintf("%s %d orchestration(s)?", action, len(ids)), func() {
		fn(ids)
	})
}

func (v *OrchestrationsView) bulkSuspend(ids []string) {
	go func() {
		unsuccessful, err := v.app.Client.BatchSuspend(context.Background(), ids, "Suspended via dts-cli")
		v.app.QueueUpdateDraw(func() {
			if err != nil {
				v.app.FlashError("Suspend failed: " + err.Error())
			} else if len(unsuccessful) > 0 {
				v.app.FlashError(fmt.Sprintf("Suspend: %d failed", len(unsuccessful)))
			} else {
				v.app.FlashSuccess(fmt.Sprintf("Suspended %d orchestration(s)", len(ids)))
			}
			v.table.ClearSelection()
		})
		v.Init(context.Background())
	}()
}

func (v *OrchestrationsView) bulkResume(ids []string) {
	go func() {
		unsuccessful, err := v.app.Client.BatchResume(context.Background(), ids, "Resumed via dts-cli")
		v.app.QueueUpdateDraw(func() {
			if err != nil {
				v.app.FlashError("Resume failed: " + err.Error())
			} else if len(unsuccessful) > 0 {
				v.app.FlashError(fmt.Sprintf("Resume: %d failed", len(unsuccessful)))
			} else {
				v.app.FlashSuccess(fmt.Sprintf("Resumed %d orchestration(s)", len(ids)))
			}
			v.table.ClearSelection()
		})
		v.Init(context.Background())
	}()
}

func (v *OrchestrationsView) bulkTerminate(ids []string) {
	go func() {
		unsuccessful, err := v.app.Client.BatchTerminate(context.Background(), ids, "Terminated via dts-cli")
		v.app.QueueUpdateDraw(func() {
			if err != nil {
				v.app.FlashError("Terminate failed: " + err.Error())
			} else if len(unsuccessful) > 0 {
				v.app.FlashError(fmt.Sprintf("Terminate: %d failed", len(unsuccessful)))
			} else {
				v.app.FlashSuccess(fmt.Sprintf("Terminated %d orchestration(s)", len(ids)))
			}
			v.table.ClearSelection()
		})
		v.Init(context.Background())
	}()
}

func (v *OrchestrationsView) bulkForceTerminate() {
	ids := v.getSelectedInstanceIDs()
	if len(ids) == 0 {
		v.app.FlashInfo("No orchestrations selected")
		return
	}
	v.app.ShowConfirm("Force Terminate", fmt.Sprintf("Force-terminate %d orchestration(s)? This skips graceful shutdown.", len(ids)), func() {
		go func() {
			unsuccessful, err := v.app.Client.ForceTerminate(context.Background(), ids, "Force-terminated via dts-cli")
			v.app.QueueUpdateDraw(func() {
				if err != nil {
					v.app.FlashError("Force terminate failed: " + err.Error())
				} else if len(unsuccessful) > 0 {
					v.app.FlashError(fmt.Sprintf("Force terminate: %d failed", len(unsuccessful)))
				} else {
					v.app.FlashSuccess(fmt.Sprintf("Force-terminated %d orchestration(s)", len(ids)))
				}
				v.table.ClearSelection()
			})
			v.Init(context.Background())
		}()
	})
}

func (v *OrchestrationsView) bulkRestart(ids []string) {
	go func() {
		var failed int
		for _, id := range ids {
			if _, err := v.app.Client.RestartOrchestration(context.Background(), id, false); err != nil {
				failed++
			}
		}
		v.app.QueueUpdateDraw(func() {
			if failed > 0 {
				v.app.FlashError(fmt.Sprintf("Restart: %d of %d failed", failed, len(ids)))
			} else {
				v.app.FlashSuccess(fmt.Sprintf("Restarted %d orchestration(s)", len(ids)))
			}
			v.table.ClearSelection()
		})
		v.Init(context.Background())
	}()
}

func (v *OrchestrationsView) bulkPurge(ids []string) {
	go func() {
		err := v.app.Client.PurgeOrchestrations(context.Background(), ids)
		v.app.QueueUpdateDraw(func() {
			if err != nil {
				v.app.FlashError("Purge failed: " + err.Error())
			} else {
				v.app.FlashSuccess(fmt.Sprintf("Purged %d orchestration(s)", len(ids)))
			}
			v.table.ClearSelection()
		})
		v.Init(context.Background())
	}()
}

func (v *OrchestrationsView) createOrchestration() {
	fields := []components.FormField{
		{Label: "Name", Default: "", Width: 40},
		{Label: "Instance ID", Default: "", Width: 40},
		{Label: "Version", Default: "", Width: 20},
		{Label: "Input (JSON)", Default: "", Width: 50},
	}

	components.MultiInputDialog(v.app.TviewApp(), v.app.Pages(), "Start New Orchestration", fields, func(values map[string]string) {
		name := values["Name"]
		if name == "" {
			v.app.FlashError("Name is required")
			return
		}

		go func() {
			req := &api.CreateOrchestrationRequest{
				Name:       name,
				InstanceID: values["Instance ID"],
				Version:    values["Version"],
				Input:      values["Input (JSON)"],
			}
			instanceID, err := v.app.Client.CreateOrchestration(context.Background(), req)
			v.app.QueueUpdateDraw(func() {
				if err != nil {
					v.app.FlashError("Create failed: " + err.Error())
				} else {
					v.app.FlashSuccess("Created: " + instanceID)
				}
			})
			v.Init(context.Background())
		}()
	})
}
