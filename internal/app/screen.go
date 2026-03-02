// Package app provides the core TUI application shell for dts-cli.
package app

import (
	"time"

	"github.com/gdamore/tcell/v2"
)

// navCoalescingScreen wraps a tcell.Screen to prevent "momentum scrolling".
//
// Problem: tview calls draw() after EVERY key event (including consumed ones).
// Each draw involves screen.Clear + root.Draw + screen.Show (~10-50ms of I/O).
// During sustained key-hold, the OS buffers 30+ key-repeat events per second.
// If each draw takes longer than the inter-event interval, events queue up and
// keep draining (with visible cursor movement) for seconds after key release.
//
// Solution: intercept PollEvent and coalesce consecutive same-direction
// navigation keys into a single event. Between draws, all buffered nav events
// of the same direction are consumed; tview sees one event, processes one draw.
// After key release, at most one nav event remains buffered — no momentum.
//
// When the nav direction changes mid-batch (e.g., Down→Up), the first direction
// is returned and the new direction is buffered for the next PollEvent call.
type navCoalescingScreen struct {
	tcell.Screen
	ch      chan tcell.Event
	pending *tcell.EventKey // buffered nav event from a direction change
}

// coalesceWindow is how long PollEvent waits for additional same-direction nav
// keys before returning. Must be shorter than one key-repeat interval (~33ms
// at 30Hz) so single taps feel instant, but long enough to catch events that
// arrive during a draw cycle.
const coalesceWindow = 12 * time.Millisecond

// newNavCoalescingScreen wraps screen. Call Init() before use — it starts the
// internal pump goroutine.
func newNavCoalescingScreen(screen tcell.Screen) *navCoalescingScreen {
	return &navCoalescingScreen{
		Screen: screen,
		ch:     make(chan tcell.Event, 512),
	}
}

// Init initializes the underlying screen and starts the event pump goroutine.
func (s *navCoalescingScreen) Init() error {
	if err := s.Screen.Init(); err != nil {
		return err
	}
	go s.pump()
	return nil
}

// pump reads events from the real screen and feeds them into our channel.
func (s *navCoalescingScreen) pump() {
	for {
		ev := s.Screen.PollEvent()
		if ev == nil {
			close(s.ch)
			return
		}
		s.ch <- ev
	}
}

// isNavKey returns true for arrow keys, page up/down, home, and end.
func isNavKey(key tcell.Key) bool {
	switch key {
	case tcell.KeyDown, tcell.KeyUp, tcell.KeyPgDn, tcell.KeyPgUp,
		tcell.KeyHome, tcell.KeyEnd:
		return true
	}
	return false
}

// isNavEvent returns true for navigation events, including vim-style j/k rune
// keys that tview tables and lists interpret as down/up movement. Coalescing
// these prevents momentum scrolling when holding j or k.
func isNavEvent(ev *tcell.EventKey) bool {
	if isNavKey(ev.Key()) {
		return true
	}
	if ev.Key() == tcell.KeyRune {
		switch ev.Rune() {
		case 'j', 'k', 'g', 'G':
			return true
		}
	}
	return false
}

// navDirection returns a comparable value representing the navigation
// direction of a navigation event. Events with the same direction are
// coalesced; direction changes flush the current batch.
func navDirection(ev *tcell.EventKey) interface{} {
	if ev.Key() == tcell.KeyRune {
		return ev.Rune() // j, k, g, G each form their own direction
	}
	return ev.Key()
}

// PollEvent returns the next event, coalescing consecutive same-direction
// navigation keys into a single event. Non-nav events pass through unchanged.
func (s *navCoalescingScreen) PollEvent() tcell.Event {
	// Return a buffered nav event from a previous direction change first.
	if s.pending != nil {
		ev := s.pending
		s.pending = nil
		return s.coalesceFrom(ev)
	}

	ev, ok := <-s.ch
	if !ok {
		return nil
	}

	keyEv, ok := ev.(*tcell.EventKey)
	if !ok || !isNavEvent(keyEv) {
		return ev // non-key or non-nav: pass through immediately
	}

	return s.coalesceFrom(keyEv)
}

// coalesceFrom takes an initial nav key event and absorbs all subsequent
// same-direction nav events currently buffered, returning the last one.
// If a different-direction nav event is encountered, it is saved in s.pending.
func (s *navCoalescingScreen) coalesceFrom(initial *tcell.EventKey) tcell.Event {
	last := initial
	dir := navDirection(initial)

	timer := time.NewTimer(coalesceWindow)
	defer timer.Stop()

	for {
		select {
		case next, ok := <-s.ch:
			if !ok {
				return last
			}
			nextKey, ok := next.(*tcell.EventKey)
			if !ok || !isNavEvent(nextKey) {
				// Non-nav event interrupts the batch.
				// We can't "un-read" it into the channel, so buffer it as
				// a pending event. However, pending is typed *EventKey for
				// nav events only. For non-nav events we need a different
				// approach: re-inject it. Since this is rare (non-nav during
				// rapid nav), we enqueue it back.
				go func() { s.ch <- next }()
				return last
			}
			if navDirection(nextKey) != dir {
				// Direction changed — buffer the new direction, return batch.
				s.pending = nextKey
				return last
			}
			// Same direction — absorb and keep waiting.
			last = nextKey
			timer.Reset(coalesceWindow)

		case <-timer.C:
			return last
		}
	}
}
