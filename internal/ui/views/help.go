package views

import (
	"context"

	"github.com/rivo/tview"

	"github.com/microsoft/durabletask-scheduler/cli/internal/app"
	"github.com/microsoft/durabletask-scheduler/cli/internal/ui/components"
)

// HelpView shows available keybindings and commands.
type HelpView struct {
	app      *app.App
	textView *tview.TextView
}

// NewHelpView creates the help/keybinding reference view.
func NewHelpView(a *app.App) *HelpView {
	v := &HelpView{
		app:      a,
		textView: tview.NewTextView().SetDynamicColors(true).SetScrollable(true),
	}
	return v
}

func (v *HelpView) Name() string               { return "help" }
func (v *HelpView) Primitive() tview.Primitive { return v.textView }
func (v *HelpView) Crumbs() []string           { return []string{"Help"} }
func (v *HelpView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "Esc", Description: "Back"},
	}
}

func (v *HelpView) Init(_ context.Context) {
	content := `
 [aqua::b]dts — Durable Task Scheduler CLI[-:-:-]
 [gray]A k9s-style terminal UI for managing DTS orchestrations[-]

 [white::b]Global Keybindings[-:-:-]
 [yellow]Ctrl+C[-]      Quit
 [yellow]Esc[-]         Go back / Close dialog
 [yellow]?[-]           Show this help
 [yellow]:[-]           Command mode
 [yellow]/[-]           Filter mode
 [yellow]r[-]           Refresh current view
 [yellow]q[-]           Quit / Go back

 [white::b]Commands (type : to enter command mode)[-:-:-]
 [yellow]:orch[-]       Go to Orchestrations
 [yellow]:ent[-]        Go to Entities
 [yellow]:sched[-]      Go to Schedules
 [yellow]:work[-]       Go to Workers
 [yellow]:ag[-]         Go to Agents (Preview)
 [yellow]:home[-]       Go to Home / Endpoints
 [yellow]:ctx <name>[-] Switch context
 [yellow]:q[-]          Quit (with confirmation)
 [yellow]:q![-]         Force quit (no confirmation)
 [yellow]:help[-]       Show this help

 [white::b]Home View[-:-:-]
 [yellow]Enter[-]       Connect to selected endpoint
 [yellow]n[-]           Add new endpoint
 [yellow]d[-]           Delete endpoint

 [white::b]Orchestrations List[-:-:-]
 [yellow]Enter[-]       View orchestration details
 [yellow]n[-]           Create new orchestration
 [yellow]o[-]           Cycle sort column
 [yellow]O[-]           Toggle sort direction (asc/desc)
 [yellow]Space[-]       Toggle row selection
 [yellow]Ctrl+A[-]      Select all rows
 [yellow]s[-]           Suspend selected
 [yellow]u[-]           Resume selected
 [yellow]k[-]           Terminate selected
 [yellow]Ctrl+K[-]      Force-terminate selected
 [yellow]x[-]           Restart selected
 [yellow]p[-]           Purge selected
 [yellow]1[-]           Filter: All
 [yellow]2[-]           Filter: Running
 [yellow]3[-]           Filter: Completed
 [yellow]4[-]           Filter: Failed
 [yellow]5[-]           Filter: Pending
 [yellow][ / ][-]       Previous / Next page

 [white::b]Orchestration Detail[-:-:-]
 [yellow]Tab[-]         Switch between State / History tabs
 [yellow]s[-]           Suspend
 [yellow]u[-]           Resume
 [yellow]k[-]           Terminate
 [yellow]Ctrl+K[-]      Force-terminate
 [yellow]x[-]           Restart
 [yellow]w[-]           Rewind
 [yellow]p[-]           Purge
 [yellow]e[-]           Raise event
 [yellow]i[-]           View input JSON
 [yellow]o[-]           View output JSON
 [yellow]c[-]           View custom status JSON

 [white::b]Entities List[-:-:-]
 [yellow]Enter[-]       View entity details
 [yellow]d[-]           Delete selected
 [yellow]Space[-]       Toggle row selection
 [yellow][ / ][-]       Previous / Next page

 [white::b]Entity Detail[-:-:-]
 [yellow]d[-]           Delete entity
 [yellow]j[-]           View state JSON

 [white::b]Schedules List[-:-:-]
 [yellow]Enter[-]       View schedule details
 [yellow]n[-]           Create new schedule
 [yellow]s[-]           Pause selected
 [yellow]u[-]           Resume selected
 [yellow]d[-]           Delete selected

 [white::b]Workers[-:-:-]
 [yellow]r[-]           Refresh

 [white::b]Agents (Preview)[-:-:-]
 [yellow]Enter[-]       Open session
 [yellow]n[-]           Start new session
 [yellow]d[-]           Delete selected
 [yellow]Space[-]       Toggle row selection

 [white::b]Agent Session[-:-:-]
 [yellow]Enter[-]       Send prompt
 [yellow]Tab[-]         Toggle input / messages focus
 [yellow]d[-]           Delete session
`
	v.textView.SetText(content)
}
