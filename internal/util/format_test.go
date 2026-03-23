package util

import "testing"

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
