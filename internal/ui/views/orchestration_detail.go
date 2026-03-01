package views

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/microsoft/durabletask-scheduler/cli/internal/api"
	"github.com/microsoft/durabletask-scheduler/cli/internal/app"
	"github.com/microsoft/durabletask-scheduler/cli/internal/ui/components"
	"github.com/microsoft/durabletask-scheduler/cli/internal/util"
)

// OrchestrationDetailView shows a single orchestration's details, state, and history.
type OrchestrationDetailView struct {
	app         *app.App
	instanceID  string
	executionID string

	flex       *tview.Flex
	header     *tview.TextView
	tabs       *tview.TextView
	stateView  *tview.TextView
	history    *components.ResourceTable
	pages      *tview.Pages

	orch     *api.Orchestration
	payloads *api.OrchestrationPayloads
	events   []api.HistoryEvent
	activeTab int
}

// NewOrchestrationDetailView creates the orchestration detail view.
func NewOrchestrationDetailView(a *app.App, instanceID, executionID string) *OrchestrationDetailView {
	v := &OrchestrationDetailView{
		app:         a,
		instanceID:  instanceID,
		executionID: executionID,
		header:      tview.NewTextView().SetDynamicColors(true),
		tabs:        tview.NewTextView().SetDynamicColors(true),
		stateView:   tview.NewTextView().SetDynamicColors(true).SetScrollable(true),
		history:     components.NewResourceTable([]string{"#", "Timestamp", "Type", "Name", "Status", "Details"}),
		activeTab:   0,
	}

	v.pages = tview.NewPages()
	v.pages.AddPage("state", v.stateView, true, true)
	v.pages.AddPage("history", v.history, true, false)

	v.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.header, 6, 0, false).
		AddItem(v.tabs, 1, 0, false).
		AddItem(v.pages, 0, 1, true)

	v.flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			v.nextTab()
			return nil
		case tcell.KeyBacktab:
			v.prevTab()
			return nil
		case tcell.KeyCtrlK:
			v.forceTerminate()
			return nil
		}
		switch event.Rune() {
		case 's':
			v.doAction("Suspend", func() error {
				return v.app.Client.SuspendOrchestration(context.Background(), v.instanceID, "Suspended via dts-cli")
			})
			return nil
		case 'u':
			v.doAction("Resume", func() error {
				return v.app.Client.ResumeOrchestration(context.Background(), v.instanceID, "Resumed via dts-cli")
			})
			return nil
		case 'k':
			v.doAction("Terminate", func() error {
				return v.app.Client.TerminateOrchestration(context.Background(), v.instanceID, "Terminated via dts-cli")
			})
			return nil
		case 'x':
			v.doRestart()
			return nil
		case 'w':
			v.doAction("Rewind", func() error {
				return v.app.Client.RewindOrchestration(context.Background(), v.instanceID, "Rewound via dts-cli")
			})
			return nil
		case 'p':
			v.doAction("Purge", func() error {
				return v.app.Client.PurgeOrchestration(context.Background(), v.instanceID)
			})
			return nil
		case 'e':
			v.raiseEvent()
			return nil
		case 'i':
			v.showJSON("Input", v.payloads.Input)
			return nil
		case 'o':
			v.showJSON("Output", v.payloads.Output)
			return nil
		case 'c':
			v.showJSON("Custom Status", v.payloads.CustomStatus)
			return nil
		}
		return event
	})

	return v
}

func (v *OrchestrationDetailView) Name() string              { return "orchestration-detail" }
func (v *OrchestrationDetailView) Primitive() tview.Primitive { return v.flex }
func (v *OrchestrationDetailView) Crumbs() []string {
	ctx := v.app.Config.CurrentContext
	return []string{ctx, "Orchestrations", v.instanceID}
}
func (v *OrchestrationDetailView) Hints() []components.KeyHint {
	hints := []components.KeyHint{
		{Key: "Tab", Description: "Switch tab"},
		{Key: "i/o/c", Description: "Input/Output/Custom"},
		{Key: "e", Description: "Raise Event"},
	}
	if v.orch != nil {
		if api.CanSuspend(v.orch.OrchestrationStatus) {
			hints = append(hints, components.KeyHint{Key: "s", Description: "Suspend"})
		}
		if api.CanResume(v.orch.OrchestrationStatus) {
			hints = append(hints, components.KeyHint{Key: "u", Description: "Resume"})
		}
		if api.CanTerminate(v.orch.OrchestrationStatus) {
			hints = append(hints, components.KeyHint{Key: "k", Description: "Terminate"})
		}
		if api.CanPurge(v.orch.OrchestrationStatus) {
			hints = append(hints, components.KeyHint{Key: "p", Description: "Purge"})
		}
	}
	return hints
}

func (v *OrchestrationDetailView) Init(ctx context.Context) {
	if v.app.Client == nil {
		return
	}

	orchCh := make(chan *api.Orchestration)
	payloadsCh := make(chan *api.OrchestrationPayloads)
	historyCh := make(chan []api.HistoryEvent)
	errCh := make(chan error, 3)

	go func() {
		o, err := v.app.Client.GetOrchestration(ctx, v.instanceID)
		if err != nil {
			errCh <- err
			orchCh <- nil
		} else {
			errCh <- nil
			orchCh <- o
		}
	}()

	go func() {
		p, err := v.app.Client.GetOrchestrationPayloads(ctx, v.instanceID)
		if err != nil {
			errCh <- err
			payloadsCh <- nil
		} else {
			errCh <- nil
			payloadsCh <- p
		}
	}()

	go func() {
		h, err := v.app.Client.GetOrchestrationHistory(ctx, v.instanceID, v.executionID)
		if err != nil {
			errCh <- err
			historyCh <- nil
		} else {
			errCh <- nil
			historyCh <- h
		}
	}()

	v.orch = <-orchCh
	v.payloads = <-payloadsCh
	v.events = <-historyCh

	// collect errors
	var firstErr error
	for i := 0; i < 3; i++ {
		if e := <-errCh; e != nil && firstErr == nil {
			firstErr = e
		}
	}

	v.app.QueueUpdateDraw(func() {
		if v.orch == nil && firstErr != nil {
			v.app.FlashError("Load failed: " + firstErr.Error())
			return
		}
		v.renderHeader()
		v.renderTabs()
		v.renderState()
		v.renderHistory()
	})
}

func (v *OrchestrationDetailView) renderHeader() {
	if v.orch == nil {
		v.header.SetText(" [gray]Loading...[-]")
		return
	}
	o := v.orch
	local := v.app.Config.UseLocalTime()
	statusColor := util.StatusColor(o.OrchestrationStatus)
	statusName := util.StatusShortName(o.OrchestrationStatus)

	duration := "-"
	if o.CompletedTimestamp != nil {
		duration = util.FormatDurationBetween(o.CreatedTimestamp, *o.CompletedTimestamp)
	} else {
		duration = util.FormatDurationSince(o.CreatedTimestamp)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, " [white::b]%s[-:-:-]\n", o.InstanceID)
	fmt.Fprintf(&sb, " Name: [white]%s[-]  Version: [white]%s[-]  Status: %s%s[-]\n", o.Name, o.Version, statusColor, statusName)
	fmt.Fprintf(&sb, " Created: [white]%s[-]  Updated: [white]%s[-]  Duration: [white]%s[-]\n",
		util.FormatTimestamp(o.CreatedTimestamp, local),
		util.FormatTimestamp(o.LastUpdatedTimestamp, local),
		duration,
	)
	if o.ParentInstanceID != "" {
		fmt.Fprintf(&sb, " Parent: [aqua]%s[-]\n", o.ParentInstanceID)
	}
	if v.payloads != nil && v.payloads.FailureDetails != nil {
		fmt.Fprintf(&sb, " [red]Error: %s[-]\n", util.Truncate(v.payloads.FailureDetails.ErrorMessage, 100))
	}
	v.header.SetText(sb.String())
}

func (v *OrchestrationDetailView) renderTabs() {
	tabs := []string{"State", "History"}
	var sb strings.Builder
	sb.WriteString(" ")
	for i, tab := range tabs {
		if i == v.activeTab {
			fmt.Fprintf(&sb, "[aqua::b] %s [-:-:-]", tab)
		} else {
			fmt.Fprintf(&sb, "[gray] %s [-]", tab)
		}
		if i < len(tabs)-1 {
			sb.WriteString(" │ ")
		}
	}
	v.tabs.SetText(sb.String())
}

func (v *OrchestrationDetailView) renderState() {
	if v.payloads == nil {
		v.stateView.SetText(" [gray]No payload data[-]")
		return
	}
	var sb strings.Builder
	p := v.payloads

	section := func(title, content string) {
		fmt.Fprintf(&sb, "\n [white::b]%s:[-:-:-]\n", title)
		if content == "" || content == "null" {
			sb.WriteString("   [gray](empty)[-]\n")
		} else {
			formatted := util.FormatJSON(content)
			for _, line := range strings.Split(formatted, "\n") {
				fmt.Fprintf(&sb, "   %s\n", line)
			}
		}
	}

	section("Input", p.Input)
	section("Output", p.Output)
	section("Custom Status", p.CustomStatus)

	if v.payloads.FailureDetails != nil {
		fmt.Fprintf(&sb, "\n [red::b]Failure Details:[-:-:-]\n")
		fmt.Fprintf(&sb, "   Type: [white]%s[-]\n", v.payloads.FailureDetails.ErrorType)
		fmt.Fprintf(&sb, "   Message: [white]%s[-]\n", v.payloads.FailureDetails.ErrorMessage)
		if v.payloads.FailureDetails.StackTrace != "" {
			fmt.Fprintf(&sb, "   Stack Trace:\n")
			for _, line := range strings.Split(v.payloads.FailureDetails.StackTrace, "\n") {
				fmt.Fprintf(&sb, "     [gray]%s[-]\n", line)
			}
		}
	}

	v.stateView.SetText(sb.String())
}

func (v *OrchestrationDetailView) renderHistory() {
	v.history.ClearData()
	if v.events == nil {
		return
	}
	local := v.app.Config.UseLocalTime()

	for i, event := range v.events {
		ts := ""
		if t, ok := event["Timestamp"].(string); ok {
			if parsed, err := time.Parse(time.RFC3339Nano, t); err == nil {
				ts = util.FormatTimestamp(parsed, local)
			}
		}
		eventType := fmt.Sprintf("%v", event["EventType"])
		name := ""
		if n, ok := event["Name"]; ok {
			name = fmt.Sprintf("%v", n)
		}
		status := ""
		if s, ok := event["OrchestrationStatus"]; ok {
			status = fmt.Sprintf("%v", s)
		}
		details := ""
		if d, ok := event["Result"]; ok && d != nil {
			if b, _ := json.Marshal(d); len(b) > 0 {
				details = util.Truncate(string(b), 60)
			}
		}

		v.history.SetDataRow(i,
			fmt.Sprintf("%d", i+1),
			ts,
			eventType,
			name,
			status,
			details,
		)
	}
}

func (v *OrchestrationDetailView) nextTab() {
	v.activeTab = (v.activeTab + 1) % 2
	v.switchTab()
}

func (v *OrchestrationDetailView) prevTab() {
	v.activeTab = (v.activeTab + 1) % 2
	v.switchTab()
}

func (v *OrchestrationDetailView) switchTab() {
	v.renderTabs()
	switch v.activeTab {
	case 0:
		v.pages.SwitchToPage("state")
		v.app.TviewApp().SetFocus(v.stateView)
	case 1:
		v.pages.SwitchToPage("history")
		v.app.TviewApp().SetFocus(v.history)
	}
}

func (v *OrchestrationDetailView) doAction(label string, fn func() error) {
	v.app.ShowConfirm(label, fmt.Sprintf("%s orchestration %s?", label, v.instanceID), func() {
		go func() {
			err := fn()
			v.app.QueueUpdateDraw(func() {
				if err != nil {
					v.app.FlashError(label + " failed: " + err.Error())
				} else {
					v.app.FlashSuccess(label + " successful")
				}
			})
			v.Init(context.Background())
		}()
	})
}

func (v *OrchestrationDetailView) doRestart() {
	v.app.ShowConfirm("Restart", fmt.Sprintf("Restart orchestration %s?", v.instanceID), func() {
		go func() {
			newID, err := v.app.Client.RestartOrchestration(context.Background(), v.instanceID, false)
			v.app.QueueUpdateDraw(func() {
				if err != nil {
					v.app.FlashError("Restart failed: " + err.Error())
				} else {
					v.app.FlashSuccess("Restarted as: " + newID)
					// Navigate to the new instance
					v.instanceID = newID
					v.executionID = ""
				}
			})
			v.Init(context.Background())
		}()
	})
}

func (v *OrchestrationDetailView) forceTerminate() {
	v.app.ShowConfirm("Force Terminate", fmt.Sprintf("Force-terminate %s? This skips graceful shutdown.", v.instanceID), func() {
		go func() {
			_, err := v.app.Client.ForceTerminate(context.Background(), []string{v.instanceID}, "Force-terminated via dts-cli")
			v.app.QueueUpdateDraw(func() {
				if err != nil {
					v.app.FlashError("Force terminate failed: " + err.Error())
				} else {
					v.app.FlashSuccess("Force-terminated")
				}
			})
			v.Init(context.Background())
		}()
	})
}

func (v *OrchestrationDetailView) raiseEvent() {
	fields := []components.FormField{
		{Label: "Event Name", Default: "", Width: 40},
		{Label: "Data (JSON)", Default: "", Width: 50},
	}
	components.MultiInputDialog(v.app.TviewApp(), v.app.Pages(), "Raise Event", fields, func(values map[string]string) {
		name := values["Event Name"]
		if name == "" {
			v.app.FlashError("Event name is required")
			return
		}
		go func() {
			err := v.app.Client.RaiseEvent(context.Background(), v.instanceID, name, values["Data (JSON)"])
			v.app.QueueUpdateDraw(func() {
				if err != nil {
					v.app.FlashError("Raise event failed: " + err.Error())
				} else {
					v.app.FlashSuccess("Event raised: " + name)
				}
			})
			v.Init(context.Background())
		}()
	})
}

func (v *OrchestrationDetailView) showJSON(title, content string) {
	if content == "" || content == "null" {
		v.app.FlashInfo(title + " is empty")
		return
	}
	jv := components.JSONViewer(title, util.FormatJSON(content))
	jv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			v.app.Pages().RemovePage("json-viewer")
			return nil
		}
		return event
	})
	v.app.Pages().AddAndSwitchToPage("json-viewer", jv, true)
}
