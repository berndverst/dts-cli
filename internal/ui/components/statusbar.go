package components

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// StatusBar is the bottom status bar showing context info and keybinding hints.
//
// Uses a dirty-flag pattern (inspired by k9s): property setters mark the bar
// dirty instead of immediately re-rendering. The actual render is deferred to
// the next Draw() call via tview's Draw override. When multiple properties
// change in the same event-loop tick (common during navigation), only a single
// render executes instead of one per setter.
type StatusBar struct {
	*tview.TextView
	context   string
	resource  string
	hints     []KeyHint
	filter    string
	message   string
	msgColor  tcell.Color
	countdown int
	dirty     bool
}

// KeyHint represents a keybinding hint displayed in the status bar.
type KeyHint struct {
	Key         string
	Description string
}

// NewStatusBar creates a new status bar.
func NewStatusBar() *StatusBar {
	sb := &StatusBar{
		TextView: tview.NewTextView(),
		dirty:    true, // ensure first Draw() renders content
	}
	sb.SetDynamicColors(true)
	sb.SetTextAlign(tview.AlignLeft)
	sb.SetBackgroundColor(tcell.ColorDarkSlateGray)
	return sb
}

// SetContext updates the displayed context name.
func (sb *StatusBar) SetContext(name string) {
	sb.context = name
	sb.dirty = true
}

// SetResource updates the displayed resource type.
func (sb *StatusBar) SetResource(name string) {
	sb.resource = name
	sb.dirty = true
}

// SetHints updates the displayed keybinding hints.
func (sb *StatusBar) SetHints(hints []KeyHint) {
	sb.hints = hints
	sb.dirty = true
}

// SetFilter updates the active filter display.
func (sb *StatusBar) SetFilter(filter string) {
	sb.filter = filter
	sb.dirty = true
}

// Flash shows a temporary message in the status bar.
func (sb *StatusBar) Flash(msg string, color tcell.Color) {
	sb.message = msg
	sb.msgColor = color
	sb.dirty = true
}

// ClearFlash removes the flash message.
func (sb *StatusBar) ClearFlash() {
	sb.message = ""
	sb.dirty = true
}

// SetCountdown updates the refresh countdown display.
func (sb *StatusBar) SetCountdown(seconds int) {
	sb.countdown = seconds
	sb.dirty = true
}

// Draw renders the status bar, re-building the text only when dirty.
// This override lets tview's draw cycle drive rendering instead of
// eagerly rebuilding on every property change.
func (sb *StatusBar) Draw(screen tcell.Screen) {
	if sb.dirty {
		sb.dirty = false
		sb.render()
	}
	sb.TextView.Draw(screen)
}

func (sb *StatusBar) render() {
	var parts []string

	if sb.context != "" {
		parts = append(parts, fmt.Sprintf("[aqua]⎈ %s[-]", sb.context))
	}
	if sb.resource != "" {
		parts = append(parts, fmt.Sprintf("[white]%s[-]", sb.resource))
	}
	if sb.filter != "" {
		parts = append(parts, fmt.Sprintf("[yellow]🔍 %s[-]", sb.filter))
	}

	left := strings.Join(parts, " [gray]│[-] ")

	// Countdown to next refresh
	if sb.countdown > 0 {
		left += fmt.Sprintf(" [gray]│[-] [white]%ds[-]", sb.countdown)
	}

	var hintParts []string
	for _, h := range sb.hints {
		hintParts = append(hintParts, fmt.Sprintf("[aqua]<%s>[white] %s", h.Key, h.Description))
	}
	right := strings.Join(hintParts, " ")

	if sb.message != "" {
		var colorTag string
		switch sb.msgColor {
		case tcell.ColorGreen:
			colorTag = "[green]"
		case tcell.ColorRed:
			colorTag = "[red]"
		default:
			colorTag = "[yellow]"
		}
		sb.SetText(fmt.Sprintf(" %s  %s%s[-]", left, colorTag, sb.message))
	} else {
		sb.SetText(fmt.Sprintf(" %s  %s", left, right))
	}
}

// Crumbs is the top breadcrumb navigation bar.
type Crumbs struct {
	*tview.TextView
	items []string
}

// NewCrumbs creates a new breadcrumb bar.
func NewCrumbs() *Crumbs {
	c := &Crumbs{
		TextView: tview.NewTextView(),
	}
	c.SetDynamicColors(true)
	c.SetTextAlign(tview.AlignLeft)
	c.SetBackgroundColor(tcell.ColorDefault)
	c.render()
	return c
}

// SetCrumbs updates the breadcrumb trail.
func (c *Crumbs) SetCrumbs(items ...string) {
	c.items = items
	c.render()
}

func (c *Crumbs) render() {
	if len(c.items) == 0 {
		c.SetText(" [aqua]dts-cli[-]")
		return
	}
	var parts []string
	for i, item := range c.items {
		if i == len(c.items)-1 {
			parts = append(parts, fmt.Sprintf("[aqua::b]%s[-:-:-]", item))
		} else {
			parts = append(parts, fmt.Sprintf("[gray]%s[-]", item))
		}
	}
	c.SetText(" " + strings.Join(parts, " [white]>[-] "))
}

// TitleBar is the top-most bar showing the DTS endpoint and task hub.
type TitleBar struct {
	*tview.TextView
	endpoint string
	taskHub  string
}

// NewTitleBar creates a new title bar.
func NewTitleBar() *TitleBar {
	tb := &TitleBar{
		TextView: tview.NewTextView(),
	}
	tb.SetDynamicColors(true)
	tb.SetTextAlign(tview.AlignLeft)
	tb.SetBackgroundColor(tcell.ColorDefault)
	tb.SetBorder(true)
	tb.SetBorderColor(tcell.ColorAqua)
	tb.render()
	return tb
}

// SetEndpoint updates the displayed endpoint URL.
func (tb *TitleBar) SetEndpoint(url string) {
	tb.endpoint = url
	tb.render()
}

// SetTaskHub updates the displayed task hub name.
func (tb *TitleBar) SetTaskHub(name string) {
	tb.taskHub = name
	tb.render()
}

// SetContext updates both endpoint and task hub from a context.
func (tb *TitleBar) SetContext(url, taskHub string) {
	tb.endpoint = url
	tb.taskHub = taskHub
	tb.render()
}

func (tb *TitleBar) render() {
	if tb.endpoint == "" && tb.taskHub == "" {
		tb.SetText(" [white::b]Durable Task Scheduler[-:-:-]")
		return
	}

	taskHubPart := ""
	if tb.taskHub != "" {
		taskHubPart = fmt.Sprintf(" [gray]│[-] [white]Task Hub:[aqua::b] %s[-:-:-]", tb.taskHub)
	}

	// Calculate how much space is available for the endpoint.
	_, _, width, _ := tb.GetInnerRect()
	if width <= 0 {
		width = 120 // sensible default before first draw
	}

	prefix := "Durable Task Scheduler: "
	taskHubPlain := ""
	if tb.taskHub != "" {
		taskHubPlain = " | Task Hub: " + tb.taskHub
	}

	// Available space for the endpoint: total width minus prefix, suffix, and padding.
	available := width - len(prefix) - len(taskHubPlain) - 2
	endpoint := tb.endpoint
	if available > 0 && len(endpoint) > available {
		if available > 3 {
			endpoint = endpoint[:available-3] + "..."
		} else {
			endpoint = "..."
		}
	}

	tb.SetText(fmt.Sprintf(" [white::b]%s[aqua]%s[-:-:-]%s", prefix, endpoint, taskHubPart))
}
