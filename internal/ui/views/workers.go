package views

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/rivo/tview"

	"github.com/Azure/durabletask-cli/internal/api"
	"github.com/Azure/durabletask-cli/internal/app"
	"github.com/Azure/durabletask-cli/internal/ui/components"
	"github.com/Azure/durabletask-cli/internal/util"
)

// WorkersView shows connected workers with per-category utilization and work item filters.
type WorkersView struct {
	app     *app.App
	table   *components.ResourceTable
	flex    *tview.Flex
	info    *tview.TextView
	detail  *tview.TextView
	workers []api.Worker
}

// NewWorkersView creates the workers list view.
func NewWorkersView(a *app.App) *WorkersView {
	v := &WorkersView{
		app: a,
		table: components.NewResourceTable([]string{
			"Worker ID",
			"Orchestrations",
			"Activities",
			"Entities",
			"Filters",
		}),
		info:   tview.NewTextView().SetDynamicColors(true),
		detail: tview.NewTextView().SetDynamicColors(true).SetWordWrap(true),
	}

	v.detail.SetBorder(true).SetTitle(" Worker Filters ").SetBorderPadding(0, 0, 1, 1)

	// Update the detail panel whenever the selected row changes.
	v.table.SetSelectionChangedFunc(func(row, _ int) {
		if row <= 0 || row-1 >= len(v.workers) {
			return
		}
		v.renderDetail(v.workers[row-1])
	})

	v.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.info, 1, 0, false).
		AddItem(v.table, 0, 1, true).
		AddItem(v.detail, 8, 0, false)

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
			v.info.SetText(" [red]Error: " + tview.Escape(err.Error()) + "[-]")
			v.app.FlashError("Failed: " + err.Error())
		})
		return
	}

	v.app.QueueUpdateDraw(func() {
		v.workers = result.Workers
		v.info.SetText(fmt.Sprintf(" [white]Workers[-] [gray](%d connected)[-]", len(v.workers)))
		v.table.ClearData()

		for i, w := range v.workers {
			v.table.SetDataRow(i,
				util.Truncate(w.WorkerID, 50),
				formatUtilization(w.ActiveOrchestrationsCount, w.MaxOrchestrationsCount),
				formatUtilization(w.ActiveActivitiesCount, w.MaxActivitiesCount),
				formatUtilization(w.ActiveEntitiesCount, w.MaxEntitiesCount),
				filterSummary(w),
			)
		}

		// Pre-select the first row's detail.
		if len(v.workers) > 0 {
			v.renderDetail(v.workers[0])
		} else {
			v.detail.SetText(" [gray]No workers connected[-]")
		}
	})
}

// formatUtilization renders "active/normalized  [bar]" for a single category.
func formatUtilization(active, max int) string {
	norm := util.NormalizeMaximumCount(max)
	return util.SaturationBar(active, norm, 8)
}

// filterSummary returns a compact one-line summary of a worker's filters.
func filterSummary(w api.Worker) string {
	if w.WorkItemFilters == nil {
		return "[gray]n/a[-]"
	}
	o := len(w.WorkItemFilters.Orchestrations)
	a := len(w.WorkItemFilters.Activities)
	e := len(w.WorkItemFilters.Entities)
	if o+a+e == 0 {
		return "[green]All[-]"
	}
	parts := make([]string, 0, 3)
	if o > 0 {
		parts = append(parts, fmt.Sprintf("%d orch", o))
	}
	if a > 0 {
		parts = append(parts, fmt.Sprintf("%d act", a))
	}
	if e > 0 {
		parts = append(parts, fmt.Sprintf("%d ent", e))
	}
	return strings.Join(parts, ", ")
}

// renderDetail populates the inline filter detail panel for a worker.
func (v *WorkersView) renderDetail(w api.Worker) {
	var b strings.Builder

	if w.WorkItemFilters == nil {
		b.WriteString(" [gray]Filter information is not available for this worker[-]\n")
		v.detail.SetText(b.String())
		return
	}

	o := w.WorkItemFilters.Orchestrations
	a := w.WorkItemFilters.Activities
	e := w.WorkItemFilters.Entities

	if len(o)+len(a)+len(e) == 0 {
		b.WriteString(" [green]This worker accepts all work items[-]\n")
		v.detail.SetText(b.String())
		return
	}

	writeSection := func(title string, filters []api.WorkItemFilter, showVersion bool) {
		b.WriteString(fmt.Sprintf(" [aqua]%s:[-] ", title))
		if len(filters) == 0 {
			b.WriteString("[gray](none)[-]\n")
			return
		}
		sorted := make([]api.WorkItemFilter, len(filters))
		copy(sorted, filters)
		sort.Slice(sorted, func(i, j int) bool {
			return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
		})
		names := make([]string, len(sorted))
		for i, f := range sorted {
			if showVersion && f.Version != nil {
				names[i] = fmt.Sprintf("%s (v%s)", f.Name, *f.Version)
			} else {
				names[i] = f.Name
			}
		}
		b.WriteString(strings.Join(names, ", "))
		b.WriteString("\n")
	}

	writeSection("Orchestration Filters", o, true)
	writeSection("Activity Filters", a, true)
	writeSection("Entity Filters", e, false)

	v.detail.SetText(b.String())
}
