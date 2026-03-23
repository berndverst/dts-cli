package util

import (
	"strings"
	"testing"
)

func TestNormalizeMaximumCount(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  int
	}{
		{"zero returns 1 (div-by-zero guard)", 0, 1},
		{"one returns 2 (min of 2*1=2, 1+100=101)", 1, 2},
		{"ten returns 20 (min of 2*10=20, 10+100=110)", 10, 20},
		{"fifty returns 100 (min of 2*50=100, 50+100=150)", 50, 100},
		{"hundred returns 200 (min of 2*100=200, 100+100=200)", 100, 200},
		{"hundred-one returns 201 (min of 2*101=202, 101+100=201)", 101, 201},
		{"five-hundred returns 600 (min of 2*500=1000, 500+100=600)", 500, 600},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeMaximumCount(tc.count)
			if got != tc.want {
				t.Errorf("NormalizeMaximumCount(%d) = %d, want %d", tc.count, got, tc.want)
			}
		})
	}
}

func TestSaturationBarThresholds(t *testing.T) {
	// Verify the bar does not panic with zero max.
	bar := SaturationBar(0, 0, 10)
	if bar == "" {
		t.Fatal("SaturationBar(0,0,10) returned empty string")
	}

	// Verify the bar handles large active > max without panic.
	bar = SaturationBar(200, 100, 10)
	if bar == "" {
		t.Fatal("SaturationBar(200,100,10) returned empty string")
	}
}

func TestSaturationBarColorBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		active    int
		max       int
		wantColor string // expected tview color tag in the bar
	}{
		{"0% green", 0, 100, "[green]"},
		{"64% green", 64, 100, "[green]"},
		{"65% yellow", 65, 100, "[yellow]"},
		{"84% yellow", 84, 100, "[yellow]"},
		{"85% red", 85, 100, "[red]"},
		{"100% red", 100, 100, "[red]"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bar := SaturationBar(tc.active, tc.max, 10)
			if !containsString(bar, tc.wantColor) {
				t.Errorf("SaturationBar(%d, %d, 10) = %q, want color %s", tc.active, tc.max, bar, tc.wantColor)
			}
		})
	}
}

func TestNormalizedSaturationBarDisplaysOriginalMax(t *testing.T) {
	// NormalizedSaturationBar should display the original max (20), not the
	// normalized one (40), while using normalized max for fill/color.
	bar := NormalizedSaturationBar(4, 20, 10)
	if !containsString(bar, "4/20") {
		t.Errorf("NormalizedSaturationBar(4,20,10) = %q, want label 4/20", bar)
	}

	// Zero max should show "?".
	bar = NormalizedSaturationBar(0, 0, 10)
	if !containsString(bar, "0/?") {
		t.Errorf("NormalizedSaturationBar(0,0,10) = %q, want label 0/?", bar)
	}
}

func TestSaturationBarFallbackWidth(t *testing.T) {
	// When max is 0, the empty bar should respect the width parameter.
	bar8 := SaturationBar(0, 0, 8)
	bar12 := SaturationBar(0, 0, 12)
	if containsString(bar8, "░░░░░░░░░░░░") {
		t.Errorf("SaturationBar(0,0,8) shouldn't have 12 empty chars: %q", bar8)
	}
	if !containsString(bar12, "░░░░░░░░░░░░") {
		t.Errorf("SaturationBar(0,0,12) should have 12 empty chars: %q", bar12)
	}
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && strings.Contains(s, substr)
}
