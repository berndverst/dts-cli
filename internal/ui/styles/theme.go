// Package styles defines color themes for the dts-cli TUI.
package styles

import "github.com/gdamore/tcell/v2"

// Theme defines the color palette for the TUI.
type Theme struct {
	// Base colors
	Background   tcell.Color
	Foreground   tcell.Color
	Border       tcell.Color
	Title        tcell.Color
	Subtitle     tcell.Color
	Highlight    tcell.Color
	HighlightFg  tcell.Color

	// Status colors
	StatusRunning   tcell.Color
	StatusCompleted tcell.Color
	StatusFailed    tcell.Color
	StatusPending   tcell.Color
	StatusSuspended tcell.Color
	StatusTerminated tcell.Color

	// UI element colors
	TableHeader    tcell.Color
	TableSelected  tcell.Color
	TableSelectedFg tcell.Color
	CrumbActive    tcell.Color
	CrumbInactive  tcell.Color
	StatusBar      tcell.Color
	StatusBarFg    tcell.Color
	FilterBar      tcell.Color
	FlashSuccess   tcell.Color
	FlashError     tcell.Color
	FlashInfo      tcell.Color

	// JSON viewer colors
	JSONKey     tcell.Color
	JSONString  tcell.Color
	JSONNumber  tcell.Color
	JSONBool    tcell.Color
	JSONNull    tcell.Color
}

// DarkTheme returns the default dark color theme.
func DarkTheme() *Theme {
	return &Theme{
		Background:   tcell.ColorDefault,
		Foreground:   tcell.ColorWhite,
		Border:       tcell.ColorDarkCyan,
		Title:        tcell.ColorAqua,
		Subtitle:     tcell.ColorGray,
		Highlight:    tcell.ColorDarkCyan,
		HighlightFg:  tcell.ColorWhite,

		StatusRunning:    tcell.ColorDodgerBlue,
		StatusCompleted:  tcell.ColorGreen,
		StatusFailed:     tcell.ColorRed,
		StatusPending:    tcell.ColorYellow,
		StatusSuspended:  tcell.ColorOrange,
		StatusTerminated: tcell.ColorGray,

		TableHeader:     tcell.ColorAqua,
		TableSelected:   tcell.ColorDarkCyan,
		TableSelectedFg: tcell.ColorWhite,
		CrumbActive:     tcell.ColorAqua,
		CrumbInactive:   tcell.ColorDarkGray,
		StatusBar:       tcell.ColorDarkSlateGray,
		StatusBarFg:     tcell.ColorWhite,
		FilterBar:       tcell.ColorDarkBlue,
		FlashSuccess:    tcell.ColorGreen,
		FlashError:      tcell.ColorRed,
		FlashInfo:       tcell.ColorDodgerBlue,

		JSONKey:    tcell.ColorAqua,
		JSONString: tcell.ColorGreen,
		JSONNumber: tcell.ColorYellow,
		JSONBool:   tcell.ColorFuchsia,
		JSONNull:   tcell.ColorGray,
	}
}

// LightTheme returns a light color theme.
func LightTheme() *Theme {
	return &Theme{
		Background:   tcell.ColorWhite,
		Foreground:   tcell.ColorBlack,
		Border:       tcell.ColorDarkBlue,
		Title:        tcell.ColorDarkBlue,
		Subtitle:     tcell.ColorDarkGray,
		Highlight:    tcell.ColorLightBlue,
		HighlightFg:  tcell.ColorBlack,

		StatusRunning:    tcell.ColorBlue,
		StatusCompleted:  tcell.ColorDarkGreen,
		StatusFailed:     tcell.ColorDarkRed,
		StatusPending:    tcell.ColorOlive,
		StatusSuspended:  tcell.ColorOrangeRed,
		StatusTerminated: tcell.ColorDarkGray,

		TableHeader:     tcell.ColorDarkBlue,
		TableSelected:   tcell.ColorLightBlue,
		TableSelectedFg: tcell.ColorBlack,
		CrumbActive:     tcell.ColorDarkBlue,
		CrumbInactive:   tcell.ColorGray,
		StatusBar:       tcell.ColorLightGray,
		StatusBarFg:     tcell.ColorBlack,
		FilterBar:       tcell.ColorLightCyan,
		FlashSuccess:    tcell.ColorDarkGreen,
		FlashError:      tcell.ColorDarkRed,
		FlashInfo:       tcell.ColorDarkBlue,

		JSONKey:    tcell.ColorDarkBlue,
		JSONString: tcell.ColorDarkGreen,
		JSONNumber: tcell.ColorDarkRed,
		JSONBool:   tcell.ColorDarkMagenta,
		JSONNull:   tcell.ColorGray,
	}
}

// GetTheme returns a theme by name.
func GetTheme(name string) *Theme {
	switch name {
	case "light":
		return LightTheme()
	default:
		return DarkTheme()
	}
}
