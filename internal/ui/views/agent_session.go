package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Azure/durabletask-cli/internal/api"
	"github.com/Azure/durabletask-cli/internal/app"
	"github.com/Azure/durabletask-cli/internal/ui/components"
)

// AgentSessionView shows an agent session's messages and allows sending prompts.
type AgentSessionView struct {
	app       *app.App
	agentName string
	sessionID string
	entityID  string

	flex     *tview.Flex
	header   *tview.TextView
	messages *tview.TextView
	input    *tview.InputField

	state *api.AgentState
}

// NewAgentSessionView creates the agent session detail view.
func NewAgentSessionView(a *app.App, agentName, sessionID, entityID string) *AgentSessionView {
	v := &AgentSessionView{
		app:       a,
		agentName: agentName,
		sessionID: sessionID,
		entityID:  entityID,
		header:    tview.NewTextView().SetDynamicColors(true),
		messages:  tview.NewTextView().SetDynamicColors(true).SetScrollable(true),
		input:     tview.NewInputField(),
	}

	v.input.SetLabel("[aqua]> [-]").
		SetFieldBackgroundColor(tcell.ColorDarkSlateGray).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				text := v.input.GetText()
				if text != "" {
					v.input.SetText("")
					v.sendPrompt(text)
				}
			}
		})

	v.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.header, 2, 0, false).
		AddItem(v.messages, 0, 1, false).
		AddItem(v.input, 1, 0, true)

	v.flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'd':
			if v.input.HasFocus() {
				return event // let input handle 'd'
			}
			v.deleteSession()
			return nil
		}
		switch event.Key() {
		case tcell.KeyTab:
			// Toggle focus between messages and input
			if v.input.HasFocus() {
				v.app.TviewApp().SetFocus(v.messages)
			} else {
				v.app.TviewApp().SetFocus(v.input)
			}
			return nil
		}
		return event
	})

	return v
}

func (v *AgentSessionView) Name() string              { return "agent-session" }
func (v *AgentSessionView) Primitive() tview.Primitive { return v.flex }
func (v *AgentSessionView) Crumbs() []string {
	ctx := v.app.Config.CurrentContext
	return []string{ctx, "Agents", v.agentName, v.sessionID}
}
func (v *AgentSessionView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "Enter", Description: "Send"},
		{Key: "Tab", Description: "Toggle focus"},
		{Key: "d", Description: "Delete session"},
		{Key: "r", Description: "Refresh"},
	}
}

func (v *AgentSessionView) Init(ctx context.Context) {
	if v.app.Client == nil {
		return
	}

	state, err := v.app.Client.GetAgentState(ctx, v.agentName, v.sessionID)
	if err != nil {
		v.app.QueueUpdateDraw(func() {
			v.app.FlashError("Load failed: " + err.Error())
		})
		return
	}

	v.state = state

	v.app.QueueUpdateDraw(func() {
		v.renderHeader()
		v.renderMessages()
	})
}

func (v *AgentSessionView) renderHeader() {
	status := "[gray]unknown[-]"
	if v.state != nil {
		switch v.state.Status {
		case "active":
			status = "[green]active[-]"
		case "waiting":
			status = "[yellow]waiting[-]"
		case "completed":
			status = "[blue]completed[-]"
		default:
			status = "[gray]" + v.state.Status + "[-]"
		}
	}
	v.header.SetText(fmt.Sprintf(" [white::b]%s[-:-:-] / [white]%s[-]  Status: %s", v.agentName, v.sessionID, status))
}

func (v *AgentSessionView) renderMessages() {
	if v.state == nil {
		v.messages.SetText("\n [gray]No messages yet. Type a prompt below and press Enter.[-]")
		return
	}

	// Flatten requests into messages
	allMsgs := v.state.Messages
	if len(allMsgs) == 0 {
		for _, req := range v.state.Requests {
			allMsgs = append(allMsgs, req.RequestMessages...)
			allMsgs = append(allMsgs, req.ResponseMessages...)
		}
	}

	if len(allMsgs) == 0 {
		v.messages.SetText("\n [gray]No messages yet. Type a prompt below and press Enter.[-]")
		return
	}

	var sb strings.Builder
	for _, msg := range allMsgs {
		switch msg.Role {
		case "user":
			fmt.Fprintf(&sb, "\n [aqua::b]You:[-:-:-]\n")
		case "assistant":
			fmt.Fprintf(&sb, "\n [green::b]Agent:[-:-:-]\n")
		case "system":
			fmt.Fprintf(&sb, "\n [yellow::b]System:[-:-:-]\n")
		case "tool":
			fmt.Fprintf(&sb, "\n [purple::b]Tool:[-:-:-]\n")
		default:
			fmt.Fprintf(&sb, "\n [gray::b]%s:[-:-:-]\n", msg.Role)
		}
		if msg.Content != "" {
			for _, line := range strings.Split(msg.Content, "\n") {
				fmt.Fprintf(&sb, "   %s\n", line)
			}
		}
		if msg.FunctionCall != nil {
			fmt.Fprintf(&sb, "   [gray]→ %s(%s)[-]\n", msg.FunctionCall.Name, msg.FunctionCall.Arguments)
		}
	}

	v.messages.SetText(sb.String())
	v.messages.ScrollToEnd()
}

func (v *AgentSessionView) sendPrompt(text string) {
	go func() {
		err := v.app.Client.SendAgentPrompt(context.Background(), v.agentName, v.sessionID, text)
		if err != nil {
			v.app.QueueUpdateDraw(func() {
				v.app.FlashError("Send failed: " + err.Error())
			})
			return
		}
		v.Init(context.Background())
	}()
}

func (v *AgentSessionView) deleteSession() {
	v.app.ShowConfirm("Delete", fmt.Sprintf("Delete session %s?", v.sessionID), func() {
		go func() {
			err := v.app.Client.DeleteAgentSession(context.Background(), v.entityID)
			v.app.QueueUpdateDraw(func() {
				if err != nil {
					v.app.FlashError("Delete failed: " + err.Error())
				} else {
					v.app.FlashSuccess("Session deleted")
					v.app.Back()
				}
			})
		}()
	})
}
