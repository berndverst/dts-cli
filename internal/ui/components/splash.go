package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// SplashScreen is a full-screen splash displayed on startup.
type SplashScreen struct {
	*tview.Flex
}

// NewSplashScreen creates a splash screen with centered branding text.
func NewSplashScreen() *SplashScreen {
	art := tview.NewTextView()
	art.SetDynamicColors(true)
	art.SetTextAlign(tview.AlignCenter)
	art.SetBackgroundColor(tcell.ColorDefault)

	art.SetText(
		"\n\n\n\n" +
			"[aqua::b]Durable Task Scheduler CLI[-:-:-]\n" +
			"\n" +
			"[white]By Bernd Verst, Azure Durable Team[-]\n" +
			"[gray]Based on the Browser Dashboard by Phillip Hoff[-]\n" +
			"\n\n" +
			"[darkgray::i]Press any key to continue[-:-:-]",
	)

	// Center the text view horizontally and vertically.
	inner := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(art, 0, 2, true).
		AddItem(nil, 0, 1, false)

	outer := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(inner, 0, 1, true).
		AddItem(nil, 0, 1, false)

	return &SplashScreen{Flex: outer}
}
