// Package util provides formatting and string utility functions for the dts.
package util

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// FormatTimestamp formats a time as a short human-readable string.
// If local is true, converts to local timezone; otherwise uses UTC.
func FormatTimestamp(t time.Time, local bool) string {
	if t.IsZero() {
		return "-"
	}
	if local {
		t = t.Local()
	} else {
		t = t.UTC()
	}
	return t.Format("2006-01-02 15:04:05")
}

// FormatDuration formats a duration as a compact human-readable string.
func FormatDuration(d time.Duration) string {
	if d < 0 {
		return "-"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}

// FormatDurationBetween formats the duration between two timestamps.
func FormatDurationBetween(start, end time.Time) string {
	if start.IsZero() || end.IsZero() {
		return "-"
	}
	return FormatDuration(end.Sub(start))
}

// FormatDurationSince formats the duration from a timestamp to now.
func FormatDurationSince(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return FormatDuration(time.Since(t))
}

// FormatRelativeTime formats a timestamp as relative offsets from
// a reference time (e.g., "+00:03" meaning 3 minutes after reference).
func FormatRelativeTime(ref, t time.Time) string {
	if ref.IsZero() || t.IsZero() {
		return "??:??"
	}
	d := t.Sub(ref)
	if d < 0 {
		d = 0
	}
	totalSec := int(d.Seconds())
	m := totalSec / 60
	s := totalSec % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

// Truncate shortens a string to maxLen, adding "…" if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return s[:maxLen-1] + "…"
}

// PadRight pads a string to the given width with spaces.
func PadRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// StatusColor returns a tview color tag for the given orchestration status.
func StatusColor(status string) string {
	switch strings.ToUpper(status) {
	case "ORCHESTRATION_STATUS_RUNNING":
		return "[blue]"
	case "ORCHESTRATION_STATUS_COMPLETED":
		return "[green]"
	case "ORCHESTRATION_STATUS_FAILED":
		return "[red]"
	case "ORCHESTRATION_STATUS_PENDING":
		return "[yellow]"
	case "ORCHESTRATION_STATUS_SUSPENDED":
		return "[orange]"
	case "ORCHESTRATION_STATUS_TERMINATED":
		return "[gray]"
	case "ORCHESTRATION_STATUS_CANCELED":
		return "[gray]"
	case "ORCHESTRATION_STATUS_CONTINUED_AS_NEW":
		return "[purple]"
	default:
		return "[white]"
	}
}

// StatusShortName converts a wire status enum to a short display name.
func StatusShortName(status string) string {
	switch strings.ToUpper(status) {
	case "ORCHESTRATION_STATUS_RUNNING":
		return "Running"
	case "ORCHESTRATION_STATUS_COMPLETED":
		return "Completed"
	case "ORCHESTRATION_STATUS_FAILED":
		return "Failed"
	case "ORCHESTRATION_STATUS_PENDING":
		return "Pending"
	case "ORCHESTRATION_STATUS_SUSPENDED":
		return "Suspended"
	case "ORCHESTRATION_STATUS_TERMINATED":
		return "Terminated"
	case "ORCHESTRATION_STATUS_CANCELED":
		return "Canceled"
	case "ORCHESTRATION_STATUS_CONTINUED_AS_NEW":
		return "ContinuedAsNew"
	default:
		return status
	}
}

// ScheduleStatusName converts a schedule status integer to a display name.
func ScheduleStatusName(status int) string {
	switch status {
	case 0:
		return "Active"
	case 1:
		return "Paused"
	default:
		return fmt.Sprintf("Unknown(%d)", status)
	}
}

// ScheduleStatusColor returns a tview color tag for schedule status.
func ScheduleStatusColor(status int) string {
	switch status {
	case 0:
		return "[green]"
	case 1:
		return "[yellow]"
	default:
		return "[white]"
	}
}

// NormalizeMaximumCount adjusts a worker's max concurrency for display purposes,
// matching the Durable Task Scheduler Dashboard normalization logic.
// It prevents overly tight gauges by allowing headroom. If count is 0 the
// result is clamped to 1 to avoid division-by-zero in saturation calculations.
func NormalizeMaximumCount(count int) int {
	n := count + 100
	if d := 2 * count; d < n {
		n = d
	}
	if n < 1 {
		n = 1
	}
	return n
}

// SaturationBar renders an ASCII progress bar like [████░░░░░░] 4/10.
// Thresholds are aligned with the Durable Task Scheduler Dashboard (65% warn, 85% error).
func SaturationBar(active, max int, width int) string {
	if max <= 0 {
		emptyBar := strings.Repeat("░", width)
		return fmt.Sprintf("[%s] %d/?", emptyBar, active)
	}
	ratio := float64(active) / float64(max)
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))
	empty := width - filled

	bar := saturationColor(ratio) + strings.Repeat("█", filled) + "[gray]" + strings.Repeat("░", empty) + "[white]"
	return fmt.Sprintf("[%s] %d/%d", bar, active, max)
}

// NormalizedSaturationBar renders a saturation bar where the fill and color
// are based on the normalized maximum (providing dashboard-style headroom)
// while the label displays the original max count. When max is 0 the bar
// shows "?" to indicate unknown capacity.
func NormalizedSaturationBar(active, max, width int) string {
	if max <= 0 {
		emptyBar := strings.Repeat("░", width)
		return fmt.Sprintf("[%s] %d/?", emptyBar, active)
	}
	norm := NormalizeMaximumCount(max)
	ratio := float64(active) / float64(norm)
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))
	empty := width - filled

	bar := saturationColor(ratio) + strings.Repeat("█", filled) + "[gray]" + strings.Repeat("░", empty) + "[white]"
	return fmt.Sprintf("[%s] %d/%d", bar, active, max)
}

// saturationColor returns the tview color tag for a saturation ratio.
func saturationColor(ratio float64) string {
	switch {
	case ratio >= 0.85:
		return "[red]"
	case ratio >= 0.65:
		return "[yellow]"
	default:
		return "[green]"
	}
}

// MustMarshal marshals a value to a JSON string, returning "" on error.
func MustMarshal(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

// FormatJSON does basic pretty-printing of JSON with tview color tags.
func FormatJSON(raw string) string {
	if raw == "" {
		return "[gray](empty)[-]"
	}
	var result strings.Builder
	indent := 0
	inString := false
	escaped := false

	for i := 0; i < len(raw); i++ {
		c := raw[i]

		if escaped {
			result.WriteByte(c)
			escaped = false
			continue
		}

		if c == '\\' && inString {
			result.WriteByte(c)
			escaped = true
			continue
		}

		if c == '"' {
			inString = !inString
			if inString {
				result.WriteString("[green]\"")
			} else {
				result.WriteString("\"[-]")
			}
			continue
		}

		if inString {
			result.WriteByte(c)
			continue
		}

		switch c {
		case '{', '[':
			result.WriteString("[white]")
			result.WriteByte(c)
			result.WriteString("[-]")
			indent++
			result.WriteByte('\n')
			result.WriteString(strings.Repeat("  ", indent))
		case '}', ']':
			indent--
			result.WriteByte('\n')
			result.WriteString(strings.Repeat("  ", indent))
			result.WriteString("[white]")
			result.WriteByte(c)
			result.WriteString("[-]")
		case ',':
			result.WriteByte(c)
			result.WriteByte('\n')
			result.WriteString(strings.Repeat("  ", indent))
		case ':':
			result.WriteString(": ")
		case ' ', '\t', '\n', '\r':
			// skip existing whitespace
		default:
			// numbers, booleans, null
			result.WriteString("[yellow]")
			result.WriteByte(c)
			result.WriteString("[-]")
		}
	}
	return result.String()
}
