package views

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
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

	flex         *tview.Flex
	header       *tview.TextView
	tabs         *tview.TextView
	stateView    *tview.TextView
	history      *components.ResourceTable
	timelineView *tview.TextView
	pages        *tview.Pages

	orch              *api.Orchestration
	payloads          *api.OrchestrationPayloads
	events            []api.HistoryEvent
	activeTab         int
	lastTimelineWidth int
}

// NewOrchestrationDetailView creates the orchestration detail view.
func NewOrchestrationDetailView(a *app.App, instanceID, executionID string) *OrchestrationDetailView {
	v := &OrchestrationDetailView{
		app:          a,
		instanceID:   instanceID,
		executionID:  executionID,
		header:       tview.NewTextView().SetDynamicColors(true),
		tabs:         tview.NewTextView().SetDynamicColors(true),
		stateView:    tview.NewTextView().SetDynamicColors(true).SetScrollable(true),
		history:      components.NewResourceTable([]string{"#", "Timestamp", "Type", "Event ID", "Name", "Tags"}),
		timelineView: tview.NewTextView().SetDynamicColors(true).SetScrollable(true),
		activeTab:    0,
	}

	// Show event details popup on Enter
	v.history.SetSelectHandler(func(row int) {
		v.showHistoryEventDetail(row)
	})

	// Re-render timeline when the view is resized
	v.timelineView.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
		if width != v.lastTimelineWidth && v.events != nil {
			v.lastTimelineWidth = width
			v.renderTimeline()
		}
		return x, y, width, height
	})

	v.pages = tview.NewPages()
	v.pages.AddPage("timeline", v.timelineView, true, true)
	v.pages.AddPage("history", v.history, true, false)
	v.pages.AddPage("state", v.stateView, true, false)

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

func (v *OrchestrationDetailView) Name() string               { return "orchestration-detail" }
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

	// Show loading indicator immediately so the UI feels responsive
	v.app.QueueUpdateDraw(func() {
		v.header.SetText(" [gray]Loading orchestration details...[-]")
	})

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
		v.renderTimeline()
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
	tabs := []string{"Timeline", "History", "State"}
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

	row := 0
	for i := range v.events {
		event := &v.events[i]
		eventType := event.Type()
		if eventType == "Unknown" {
			continue
		}
		ts := ""
		timestamp := event.ParseTimestamp()
		if !timestamp.IsZero() {
			ts = util.FormatTimestamp(timestamp, local)
		}
		name := event.EventName()
		eventID := fmt.Sprintf("%d", event.EventID)
		tags := event.FormatTags()

		v.history.SetDataRow(row,
			fmt.Sprintf("%d", row+1),
			ts,
			eventType,
			eventID,
			name,
			tags,
		)
		row++
	}
}

func (v *OrchestrationDetailView) nextTab() {
	v.activeTab = (v.activeTab + 1) % 3
	v.switchTab()
}

func (v *OrchestrationDetailView) prevTab() {
	v.activeTab = (v.activeTab + 2) % 3
	v.switchTab()
}

func (v *OrchestrationDetailView) switchTab() {
	v.renderTabs()
	switch v.activeTab {
	case 0:
		v.pages.SwitchToPage("timeline")
		v.app.TviewApp().SetFocus(v.timelineView)
	case 1:
		v.pages.SwitchToPage("history")
		v.app.TviewApp().SetFocus(v.history)
	case 2:
		v.pages.SwitchToPage("state")
		v.app.TviewApp().SetFocus(v.stateView)
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

// --- Timeline rendering ---

// timelineEntry represents a single event/activity in the timeline visualization.
type timelineEntry struct {
	name      string
	category  string // "Orchestration", "Activity", "SubOrchestration", "Timer", "Event"
	startTime time.Time
	endTime   *time.Time // nil if still running
	failed    bool
}

func (v *OrchestrationDetailView) renderTimeline() {
	entries := v.parseTimelineEntries()
	if len(entries) == 0 {
		v.timelineView.SetText(" [gray]No timeline data[-]")
		return
	}

	// Determine overall time range
	minTime := entries[0].startTime
	maxTime := minTime
	now := time.Now()

	for _, e := range entries {
		if e.startTime.Before(minTime) {
			minTime = e.startTime
		}
		end := now
		if e.endTime != nil {
			end = *e.endTime
		}
		if end.After(maxTime) {
			maxTime = end
		}
	}

	totalDuration := maxTime.Sub(minTime)
	if totalDuration <= 0 {
		totalDuration = time.Second
	}

	// Compute bar width dynamically from the view width.
	// Layout: " I name │ bar │ duration\n"
	// Fixed overhead: 1 (space) + 1 (icon) + 1 (space) + nameWidth + 1 (space) + 1 (│) + 1 (space) + 1 (space) + 1 (│) + 1 (space) + 9 (duration) = nameWidth + 18
	const nameWidth = 24
	const fixedOverhead = nameWidth + 18
	_, _, viewWidth, _ := v.timelineView.GetInnerRect()
	barWidth := viewWidth - fixedOverhead
	if barWidth < 20 {
		barWidth = 20
	}

	var sb strings.Builder
	sb.WriteString("\n")

	// Time axis header
	startLabel := util.FormatRelativeTime(minTime, minTime)
	midLabel := util.FormatRelativeTime(minTime, minTime.Add(totalDuration/2))
	endLabel := util.FormatRelativeTime(minTime, maxTime)

	fmt.Fprintf(&sb, " [aqua::b]%-*s[-:-:-] │ [aqua::b]%-*s[-:-:-] │ [aqua::b]Duration[-:-:-]\n",
		nameWidth+2, "Event", barWidth, "Timeline")
	sb.WriteString(" " + strings.Repeat("─", nameWidth+2) + "─┼─")
	sb.WriteString(strings.Repeat("─", barWidth))
	sb.WriteString("─┼─────────\n")

	// Time tick labels
	midPos := barWidth / 2
	axisBuf := make([]byte, barWidth)
	for i := range axisBuf {
		axisBuf[i] = ' '
	}
	// Place start, mid, end labels
	placeLabel(axisBuf, 0, startLabel)
	placeLabel(axisBuf, midPos-len(midLabel)/2, midLabel)
	placeLabel(axisBuf, barWidth-len(endLabel), endLabel)

	fmt.Fprintf(&sb, " %-*s │ [gray]%s[-] │\n",
		nameWidth+2, "", string(axisBuf))

	// Render each entry
	for _, e := range entries {
		icon := categoryIcon(e.category)
		displayName := util.Truncate(e.name, nameWidth)
		displayName = util.PadRight(displayName, nameWidth)

		// Calculate bar positions
		startFrac := float64(e.startTime.Sub(minTime)) / float64(totalDuration)
		var endFrac float64
		if e.endTime != nil {
			endFrac = float64(e.endTime.Sub(minTime)) / float64(totalDuration)
		} else {
			endFrac = float64(now.Sub(minTime)) / float64(totalDuration)
		}

		startPos := int(math.Round(startFrac * float64(barWidth)))
		endPos := int(math.Round(endFrac * float64(barWidth)))
		if endPos <= startPos {
			endPos = startPos + 1
		}
		if startPos < 0 {
			startPos = 0
		}
		if endPos > barWidth {
			endPos = barWidth
		}

		// Build the colored bar
		barColor := barColorForEntry(e)
		prefixDots := strings.Repeat("·", startPos)
		block := strings.Repeat("█", endPos-startPos)
		suffixDots := strings.Repeat("·", barWidth-endPos)

		barStr := fmt.Sprintf("[gray]%s[-]%s%s[-][gray]%s[-]", prefixDots, barColor, block, suffixDots)

		// Duration label
		var duration string
		if e.endTime != nil {
			duration = util.FormatDuration(e.endTime.Sub(e.startTime))
		} else {
			duration = util.FormatDurationSince(e.startTime) + "…"
		}

		fmt.Fprintf(&sb, " %s %s │ %s │ %s\n", icon, displayName, barStr, duration)
	}

	// Category legend
	sb.WriteString(" " + strings.Repeat("─", nameWidth+2) + "─┼─")
	sb.WriteString(strings.Repeat("─", barWidth))
	sb.WriteString("─┼─────────\n")
	fmt.Fprintf(&sb, " [gray]%-*s[-]\n",
		nameWidth+barWidth,
		"═ Orch  ▲ Activity  ◈ Sub  ◷ Timer  ▸ Event")

	v.timelineView.SetText(sb.String())
}

// placeLabel writes a label into a byte buffer at the given position without overflowing.
func placeLabel(buf []byte, pos int, label string) {
	for i := 0; i < len(label) && pos+i < len(buf); i++ {
		if pos+i >= 0 {
			buf[pos+i] = label[i]
		}
	}
}

func categoryIcon(category string) string {
	switch category {
	case "Orchestration":
		return "[blue]═[-]"
	case "Activity":
		return "[green]▲[-]"
	case "SubOrchestration":
		return "[purple]◈[-]"
	case "Timer":
		return "[yellow]◷[-]"
	case "Event":
		return "[aqua]▸[-]"
	default:
		return "[gray]·[-]"
	}
}

func barColorForEntry(e timelineEntry) string {
	if e.failed {
		return "[red]"
	}
	switch e.category {
	case "Orchestration":
		return "[blue]"
	case "Activity":
		return "[green]"
	case "SubOrchestration":
		return "[purple]"
	case "Timer":
		return "[yellow]"
	case "Event":
		return "[aqua]"
	default:
		return "[white]"
	}
}

// parseTimelineEntries converts raw history events into timeline entries for visualization.
func (v *OrchestrationDetailView) parseTimelineEntries() []timelineEntry {
	if v.events == nil {
		return nil
	}

	type pendingEvent struct {
		name      string
		category  string
		startTime time.Time
	}

	pending := make(map[int]*pendingEvent)
	var entries []timelineEntry

	for i := range v.events {
		event := &v.events[i]
		eventType := event.Type()
		timestamp := event.ParseTimestamp()
		if timestamp.IsZero() {
			continue
		}

		switch eventType {
		case "ExecutionStarted":
			name := event.EventName()
			if name == "" {
				name = v.instanceID
			}
			entries = append(entries, timelineEntry{
				name:      name,
				category:  "Orchestration",
				startTime: timestamp,
			})

		case "ExecutionCompleted":
			for i := range entries {
				if entries[i].category == "Orchestration" && entries[i].endTime == nil {
					t := timestamp
					entries[i].endTime = &t
				}
			}

		case "ExecutionFailed", "ExecutionTerminated":
			for i := range entries {
				if entries[i].category == "Orchestration" && entries[i].endTime == nil {
					t := timestamp
					entries[i].endTime = &t
					if eventType == "ExecutionFailed" {
						entries[i].failed = true
					}
				}
			}

		case "TaskScheduled":
			id := event.EventID
			name := event.EventName()
			if name == "" {
				name = "Activity"
			}
			pending[id] = &pendingEvent{name: name, category: "Activity", startTime: timestamp}

		case "TaskCompleted":
			scheduledID := event.ScheduledID()
			if p, ok := pending[scheduledID]; ok {
				t := timestamp
				entries = append(entries, timelineEntry{
					name: p.name, category: p.category,
					startTime: p.startTime, endTime: &t,
				})
				delete(pending, scheduledID)
			}

		case "TaskFailed":
			scheduledID := event.ScheduledID()
			if p, ok := pending[scheduledID]; ok {
				t := timestamp
				entries = append(entries, timelineEntry{
					name: p.name, category: p.category,
					startTime: p.startTime, endTime: &t, failed: true,
				})
				delete(pending, scheduledID)
			}

		case "SubOrchestrationInstanceCreated":
			id := event.EventID
			name := event.EventName()
			if name == "" {
				name = "SubOrchestration"
			}
			pending[id] = &pendingEvent{name: name, category: "SubOrchestration", startTime: timestamp}

		case "SubOrchestrationInstanceCompleted":
			scheduledID := event.ScheduledID()
			if p, ok := pending[scheduledID]; ok {
				t := timestamp
				entries = append(entries, timelineEntry{
					name: p.name, category: p.category,
					startTime: p.startTime, endTime: &t,
				})
				delete(pending, scheduledID)
			}

		case "SubOrchestrationInstanceFailed":
			scheduledID := event.ScheduledID()
			if p, ok := pending[scheduledID]; ok {
				t := timestamp
				entries = append(entries, timelineEntry{
					name: p.name, category: p.category,
					startTime: p.startTime, endTime: &t, failed: true,
				})
				delete(pending, scheduledID)
			}

		case "TimerCreated":
			id := event.EventID
			pending[id] = &pendingEvent{name: "Timer", category: "Timer", startTime: timestamp}

		case "TimerFired":
			timerID := event.FiredTimerID()
			if p, ok := pending[timerID]; ok {
				t := timestamp
				entries = append(entries, timelineEntry{
					name: p.name, category: p.category,
					startTime: p.startTime, endTime: &t,
				})
				delete(pending, timerID)
			}

		case "EventRaised":
			name := event.EventName()
			if name == "" {
				name = "Event"
			}
			entries = append(entries, timelineEntry{
				name: name, category: "Event",
				startTime: timestamp, endTime: &timestamp,
			})

		case "EventSent":
			name := event.EventName()
			if name == "" {
				name = "Sent Event"
			}
			entries = append(entries, timelineEntry{
				name: name, category: "Event",
				startTime: timestamp, endTime: &timestamp,
			})
		}
	}

	// Add any remaining pending events (still running)
	for _, p := range pending {
		entries = append(entries, timelineEntry{
			name:      p.name,
			category:  p.category,
			startTime: p.startTime,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].startTime.Before(entries[j].startTime)
	})

	return entries
}

// showHistoryEventDetail shows a popup with the full JSON of a history event.
func (v *OrchestrationDetailView) showHistoryEventDetail(row int) {
	if v.events == nil {
		return
	}
	// Map display row back to the event, skipping Unknown types
	idx := 0
	for i := range v.events {
		event := &v.events[i]
		eventType := event.Type()
		if eventType == "Unknown" {
			continue
		}
		if idx == row {
			b, err := json.MarshalIndent(event, "", "  ")
			if err != nil {
				v.app.FlashError("Failed to marshal event: " + err.Error())
				return
			}
			title := fmt.Sprintf("Event #%d - %s", row+1, eventType)
			v.showJSON(title, string(b))
			return
		}
		idx++
	}
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
			v.app.TviewApp().SetFocus(v.flex)
			return nil
		}
		return event
	})
	v.app.Pages().AddAndSwitchToPage("json-viewer", jv, true)
}
