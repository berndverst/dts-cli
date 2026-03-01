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

// EntityDetailView shows a single entity's state.
type EntityDetailView struct {
	app      *app.App
	entityID string

	flex      *tview.Flex
	header    *tview.TextView
	stateView *tview.TextView
	entity    *api.Entity
}

// NewEntityDetailView creates the entity detail view.
func NewEntityDetailView(a *app.App, entityID string) *EntityDetailView {
	v := &EntityDetailView{
		app:       a,
		entityID:  entityID,
		header:    tview.NewTextView().SetDynamicColors(true),
		stateView: tview.NewTextView().SetDynamicColors(true).SetScrollable(true),
	}

	v.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.header, 4, 0, false).
		AddItem(v.stateView, 0, 1, true)

	v.flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'd':
			v.deleteEntity()
			return nil
		case 'j':
			v.showState()
			return nil
		}
		return event
	})

	return v
}

func (v *EntityDetailView) Name() string              { return "entity-detail" }
func (v *EntityDetailView) Primitive() tview.Primitive { return v.flex }
func (v *EntityDetailView) Crumbs() []string {
	ctxName := v.app.Config.CurrentContext
	name := api.ParseEntityName(v.entityID)
	key := api.ParseEntityKey(v.entityID)
	return []string{ctxName, "Entities", fmt.Sprintf("%s/%s", name, key)}
}
func (v *EntityDetailView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "d", Description: "Delete"},
		{Key: "j", Description: "View JSON"},
	}
}

func (v *EntityDetailView) Init(ctx context.Context) {
	if v.app.Client == nil {
		return
	}

	entity, err := v.app.Client.GetEntity(ctx, v.entityID)
	if err != nil {
		v.app.QueueUpdateDraw(func() {
			v.app.FlashError("Load failed: " + err.Error())
		})
		return
	}
	v.entity = entity

	state, _ := v.app.Client.GetEntityState(ctx, v.entityID)

	v.app.QueueUpdateDraw(func() {
		v.renderHeader()
		v.renderState(state)
	})
}

func (v *EntityDetailView) renderHeader() {
	if v.entity == nil {
		v.header.SetText(" [gray]Loading...[-]")
		return
	}
	e := v.entity
	local := v.app.Config.UseLocalTime()
	name := api.ParseEntityName(e.InstanceID)
	key := api.ParseEntityKey(e.InstanceID)

	var headerText string
	lockedInfo := ""
	if e.LockedBy != "" {
		lockedInfo = fmt.Sprintf("  [yellow]Locked by: %s[-]", e.LockedBy)
	}
	headerText = fmt.Sprintf(" [white::b]%s[-:-:-] / [white]%s[-]\n ID: [gray]%s[-]\n Last Modified: [white]%s[-]%s",
		name, key, e.InstanceID,
		util.FormatTimestamp(e.LastModifiedTime, local),
		lockedInfo,
	)
	v.header.SetText(headerText)
}

func (v *EntityDetailView) renderState(state string) {
	if state == "" || state == "null" {
		v.stateView.SetText("\n [gray](no state)[-]")
		return
	}
	formatted := util.FormatJSON(state)
	v.stateView.SetText("\n [white::b]State:[-:-:-]\n" + formatted)
}

func (v *EntityDetailView) deleteEntity() {
	v.app.ShowConfirm("Delete", fmt.Sprintf("Delete entity %s?", v.entityID), func() {
		go func() {
			err := v.app.Client.DeleteEntity(context.Background(), v.entityID)
			v.app.QueueUpdateDraw(func() {
				if err != nil {
					v.app.FlashError("Delete failed: " + err.Error())
				} else {
					v.app.FlashSuccess("Entity deleted")
					v.app.Back()
				}
			})
		}()
	})
}

func (v *EntityDetailView) showState() {
	if v.entity == nil {
		return
	}
	state, err := v.app.Client.GetEntityState(context.Background(), v.entityID)
	if err != nil {
		v.app.FlashError("Failed to get state: " + err.Error())
		return
	}
	if state == "" || state == "null" {
		v.app.FlashInfo("State is empty")
		return
	}
	jv := components.JSONViewer("Entity State", util.FormatJSON(state))
	jv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			v.app.Pages().RemovePage("json-viewer")
			return nil
		}
		return event
	})
	v.app.Pages().AddAndSwitchToPage("json-viewer", jv, true)
}
