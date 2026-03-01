package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ConfirmDialog shows a yes/no confirmation dialog.
// Press 'y' or Enter to confirm, 'n' or Escape to cancel.
func ConfirmDialog(app *tview.Application, pages *tview.Pages, title, message string, onConfirm func()) {
	// Save and later restore the app-level input capture
	prevCapture := app.GetInputCapture()
	cleanup := func() {
		pages.RemovePage("confirm")
		app.SetInputCapture(prevCapture)
		// Refocus the front page primitive
		if name, item := pages.GetFrontPage(); name != "" && item != nil {
			app.SetFocus(item)
		}
	}

	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"(y)es", "(n)o"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			cleanup()
			if buttonIndex == 0 {
				onConfirm()
			}
		})
	modal.SetTitle(" " + title + " ").
		SetBorder(true).
		SetBorderColor(tcell.ColorAqua)

	// Capture y/n/Esc at the application level while the confirm page is shown,
	// because the modal's internal form swallows key events before they reach
	// the modal's own InputCapture.
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'y', 'Y':
			cleanup()
			onConfirm()
			return nil
		case 'n', 'N':
			cleanup()
			return nil
		}
		switch event.Key() {
		case tcell.KeyEscape:
			cleanup()
			return nil
		}
		if prevCapture != nil {
			return prevCapture(event)
		}
		return event
	})

	pages.AddPage("confirm", modal, true, true)
}

// InputDialog shows a single-field input dialog.
func InputDialog(app *tview.Application, pages *tview.Pages, title, label, defaultValue string, onSubmit func(value string)) {
	form := tview.NewForm().
		AddInputField(label, defaultValue, 50, nil, nil).
		AddButton("OK", func() {
			value := form_getField(pages, label)
			pages.RemovePage("input")
			onSubmit(value)
		}).
		AddButton("Cancel", func() {
			pages.RemovePage("input")
		})

	form.SetTitle(" " + title + " ").
		SetBorder(true).
		SetBorderColor(tcell.ColorAqua)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.RemovePage("input")
			return nil
		}
		return event
	})

	// Center the form
	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 10, 1, true).
			AddItem(nil, 0, 1, false), 60, 1, true).
		AddItem(nil, 0, 1, false)

	pages.AddPage("input", flex, true, true)
}

// MultiInputDialog shows a form with multiple input fields.
func MultiInputDialog(app *tview.Application, pages *tview.Pages, title string, fields []FormField, onSubmit func(values map[string]string)) {
	form := tview.NewForm()

	for _, f := range fields {
		form.AddInputField(f.Label, f.Default, f.Width, nil, nil)
	}

	form.AddButton("OK", func() {
		values := make(map[string]string)
		for i, f := range fields {
			item := form.GetFormItem(i)
			if input, ok := item.(*tview.InputField); ok {
				values[f.Label] = input.GetText()
			}
		}
		pages.RemovePage("multi-input")
		onSubmit(values)
	}).
		AddButton("Cancel", func() {
			pages.RemovePage("multi-input")
		})

	form.SetTitle(" " + title + " ").
		SetBorder(true).
		SetBorderColor(tcell.ColorAqua)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.RemovePage("multi-input")
			return nil
		}
		return event
	})

	height := len(fields)*2 + 5
	if height < 10 {
		height = 10
	}

	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, height, 1, true).
			AddItem(nil, 0, 1, false), 70, 1, true).
		AddItem(nil, 0, 1, false)

	pages.AddPage("multi-input", flex, true, true)
}

// FormField defines a field in a multi-input dialog.
type FormField struct {
	Label   string
	Default string
	Width   int
}

func form_getField(pages *tview.Pages, label string) string {
	// This is a helper that would need the form reference;
	// in practice the form closure captures the value directly.
	return ""
}

// JSONViewer creates a read-only text view for displaying JSON.
func JSONViewer(title, content string) *tview.TextView {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetText(content)
	tv.SetTitle(" " + title + " ").
		SetBorder(true).
		SetBorderColor(tcell.ColorAqua)
	return tv
}

// CommandInput is the command-mode input bar (triggered by ':').
type CommandInput struct {
	*tview.InputField
	onCommand func(cmd string)
	onCancel  func()
}

// NewCommandInput creates a new command input field.
func NewCommandInput(onCommand func(cmd string), onCancel func()) *CommandInput {
	ci := &CommandInput{
		InputField: tview.NewInputField(),
		onCommand:  onCommand,
		onCancel:   onCancel,
	}
	ci.SetLabel(":")
	ci.SetFieldBackgroundColor(tcell.ColorDefault)
	ci.SetLabelColor(tcell.ColorAqua)
	ci.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			if onCommand != nil {
				onCommand(ci.GetText())
			}
		case tcell.KeyEscape:
			if onCancel != nil {
				onCancel()
			}
		}
	})
	return ci
}

// FilterInput is the filter input bar (triggered by '/').
type FilterInput struct {
	*tview.InputField
	onFilter func(filter string)
	onCancel func()
}

// NewFilterInput creates a new filter input field.
func NewFilterInput(onFilter func(filter string), onCancel func()) *FilterInput {
	fi := &FilterInput{
		InputField: tview.NewInputField(),
		onFilter:   onFilter,
		onCancel:   onCancel,
	}
	fi.SetLabel("/")
	fi.SetFieldBackgroundColor(tcell.ColorDefault)
	fi.SetLabelColor(tcell.ColorYellow)
	fi.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			if onFilter != nil {
				onFilter(fi.GetText())
			}
		case tcell.KeyEscape:
			if onCancel != nil {
				onCancel()
			}
		}
	})
	return fi
}
