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

// SchedulesView shows the schedules list.
type SchedulesView struct {
	app   *app.App
	table *components.ResourceTable
	flex  *tview.Flex
	info  *tview.TextView

	data           []api.Schedule
	nextPageToken  string
	prevPageTokens []string
	currentToken   string
}

// NewSchedulesView creates the schedules list view.
func NewSchedulesView(a *app.App) *SchedulesView {
	v := &SchedulesView{
		app:   a,
		table: components.NewResourceTable([]string{"Schedule ID", "Orchestration", "Status", "Interval", "Next Run", "Last Run", "Created"}),
		info:  tview.NewTextView().SetDynamicColors(true),
	}

	v.table.SetSelectHandler(func(row int) {
		if row < len(v.data) {
			s := v.data[row]
			v.showScheduleDetail(s)
		}
	})

	v.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'n':
			v.createSchedule()
			return nil
		case 's':
			v.pauseSelected()
			return nil
		case 'u':
			v.resumeSelected()
			return nil
		case 'd':
			v.deleteSelected()
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

func (v *SchedulesView) Name() string              { return "schedules" }
func (v *SchedulesView) Primitive() tview.Primitive { return v.flex }
func (v *SchedulesView) Crumbs() []string {
	ctx := v.app.Config.CurrentContext
	return []string{ctx, "Schedules"}
}
func (v *SchedulesView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "Enter", Description: "Detail"},
		{Key: "n", Description: "New"},
		{Key: "s", Description: "Pause"},
		{Key: "u", Description: "Resume"},
		{Key: "d", Description: "Delete"},
		{Key: "[/]", Description: "Page"},
	}
}

func (v *SchedulesView) Init(ctx context.Context) {
	if v.app.Client == nil {
		return
	}

	result, err := v.app.Client.ListSchedules(ctx, v.currentToken)
	if err != nil {
		v.app.QueueUpdateDraw(func() {
			v.app.FlashError("Failed: " + err.Error())
		})
		return
	}

	v.data = result.Entities
	v.nextPageToken = result.ContinuationToken

	v.app.QueueUpdateDraw(func() {
		v.info.SetText(fmt.Sprintf(" [white]Schedules[-] [gray](%d shown)[-]", len(v.data)))
		v.renderTable()
	})
}

func (v *SchedulesView) renderTable() {
	v.table.ClearData()
	local := v.app.Config.UseLocalTime()

	for i, s := range v.data {
		status := "[green]Active[-]"
		if s.Status != 0 {
			status = "[yellow]Paused[-]"
		}

		interval := s.ScheduleConfiguration.Interval

		lastRun := "-"
		if s.LastRunAt != nil {
			lastRun = util.FormatTimestamp(*s.LastRunAt, local)
		}
		createdAt := "-"
		if s.ScheduleCreatedAt != nil {
			createdAt = util.FormatTimestamp(*s.ScheduleCreatedAt, local)
		}

		v.table.SetDataRow(i,
			s.ScheduleConfiguration.ScheduleID,
			s.ScheduleConfiguration.OrchestrationName,
			status,
			interval,
			"-", // Next run not available from API directly
			lastRun,
			createdAt,
		)
	}
}

func (v *SchedulesView) showScheduleDetail(s api.Schedule) {
	formatted := util.FormatJSON(util.MustMarshal(s))
	jv := components.JSONViewer("Schedule: "+s.ScheduleConfiguration.ScheduleID, formatted)
	jv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			v.app.Pages().RemovePage("json-viewer")
			return nil
		}
		return event
	})
	v.app.Pages().AddAndSwitchToPage("json-viewer", jv, true)
}

func (v *SchedulesView) getCurrentSchedule() *api.Schedule {
	row, _ := v.table.GetSelection()
	dataRow := row - 1
	if dataRow >= 0 && dataRow < len(v.data) {
		return &v.data[dataRow]
	}
	return nil
}

func (v *SchedulesView) pauseSelected() {
	s := v.getCurrentSchedule()
	if s == nil {
		return
	}
	v.app.ShowConfirm("Pause", fmt.Sprintf("Pause schedule %s?", s.ScheduleConfiguration.ScheduleID), func() {
		go func() {
			err := v.app.Client.PauseSchedule(context.Background(), s.ScheduleConfiguration.ScheduleID)
			v.app.QueueUpdateDraw(func() {
				if err != nil {
					v.app.FlashError("Pause failed: " + err.Error())
				} else {
					v.app.FlashSuccess("Schedule paused")
				}
			})
			v.Init(context.Background())
		}()
	})
}

func (v *SchedulesView) resumeSelected() {
	s := v.getCurrentSchedule()
	if s == nil {
		return
	}
	v.app.ShowConfirm("Resume", fmt.Sprintf("Resume schedule %s?", s.ScheduleConfiguration.ScheduleID), func() {
		go func() {
			err := v.app.Client.ResumeSchedule(context.Background(), s.ScheduleConfiguration.ScheduleID)
			v.app.QueueUpdateDraw(func() {
				if err != nil {
					v.app.FlashError("Resume failed: " + err.Error())
				} else {
					v.app.FlashSuccess("Schedule resumed")
				}
			})
			v.Init(context.Background())
		}()
	})
}

func (v *SchedulesView) deleteSelected() {
	s := v.getCurrentSchedule()
	if s == nil {
		return
	}
	v.app.ShowConfirm("Delete", fmt.Sprintf("Delete schedule %s?", s.ScheduleConfiguration.ScheduleID), func() {
		go func() {
			err := v.app.Client.DeleteSchedule(context.Background(), s.ScheduleConfiguration.ScheduleID)
			v.app.QueueUpdateDraw(func() {
				if err != nil {
					v.app.FlashError("Delete failed: " + err.Error())
				} else {
					v.app.FlashSuccess("Schedule deleted")
				}
			})
			v.Init(context.Background())
		}()
	})
}

func (v *SchedulesView) createSchedule() {
	fields := []components.FormField{
		{Label: "Schedule ID", Default: "", Width: 40},
		{Label: "Orchestration Name", Default: "", Width: 40},
		{Label: "Interval (e.g. PT1H)", Default: "", Width: 20},
		{Label: "Input (JSON)", Default: "", Width: 50},
	}

	components.MultiInputDialog(v.app.TviewApp(), v.app.Pages(), "Create Schedule", fields, func(values map[string]string) {
		scheduleID := values["Schedule ID"]
		orchName := values["Orchestration Name"]
		if scheduleID == "" || orchName == "" {
			v.app.FlashError("Schedule ID and Orchestration Name are required")
			return
		}

		go func() {
			req := &api.CreateScheduleRequest{
				ScheduleID:        scheduleID,
				OrchestrationName: orchName,
				Interval:          values["Interval (e.g. PT1H)"],
				OrchestrationInput: values["Input (JSON)"],
			}
			err := v.app.Client.CreateSchedule(context.Background(), req)
			v.app.QueueUpdateDraw(func() {
				if err != nil {
					v.app.FlashError("Create failed: " + err.Error())
				} else {
					v.app.FlashSuccess("Schedule created: " + scheduleID)
				}
			})
			v.Init(context.Background())
		}()
	})
}

func (v *SchedulesView) nextPage() {
	if v.nextPageToken != "" {
		v.prevPageTokens = append(v.prevPageTokens, v.currentToken)
		v.currentToken = v.nextPageToken
		go func() {
			v.Init(context.Background())
		}()
	}
}

func (v *SchedulesView) prevPage() {
	if len(v.prevPageTokens) > 0 {
		v.currentToken = v.prevPageTokens[len(v.prevPageTokens)-1]
		v.prevPageTokens = v.prevPageTokens[:len(v.prevPageTokens)-1]
		go func() {
			v.Init(context.Background())
		}()
	}
}
