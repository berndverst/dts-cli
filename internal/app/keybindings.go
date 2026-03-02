package app

import "github.com/gdamore/tcell/v2"

// globalKeyHandler handles application-wide key events.
func (a *App) globalKeyHandler(event *tcell.EventKey) *tcell.EventKey {
	// Don't intercept keys when a dialog/modal is showing
	if name, _ := a.pages.GetFrontPage(); name == "confirm" || name == "input" || name == "multi-input" || name == "json-viewer" {
		return event
	}

	// When command or filter input is active, only handle Escape (to dismiss)
	// and Ctrl+C (to quit). All other keys must pass through to the input widget.
	if a.cmdVisible || a.filterVisible {
		switch event.Key() {
		case tcell.KeyCtrlC:
			a.Stop()
			return nil
		case tcell.KeyEscape:
			if a.cmdVisible {
				a.hideCommandInput()
			} else {
				a.hideFilterInput()
			}
			return nil
		}
		return event
	}

	switch event.Key() {
	case tcell.KeyCtrlC:
		a.Stop()
		return nil

	case tcell.KeyEscape:
		a.Back()
		return nil

	case tcell.KeyRune:
		switch event.Rune() {
		case ':':
			a.showCommandInput()
			return nil
		case '/':
			a.showFilterInput()
			return nil
		case '?':
			a.NavigateToResource("help")
			return nil
		case 'q':
			if len(a.viewStack) <= 1 {
				a.ShowConfirm("Quit", "Are you sure you want to quit?", func() {
					a.Stop()
				})
			} else {
				a.Back()
			}
			return nil
		case 'r':
			a.Refresh()
			return nil
		}
	}

	return event
}
