package converter

import "testing"

func TestParseFrameRate(t *testing.T) {
	cases := []struct {
		input string
		want  float64
	}{
		{"30/1", 30.0},
		{"30000/1001", 29.97002997002997},
		{"60/1", 60.0},
		{"24000/1001", 23.976023976023978},
		{"", 0},
		{"0/0", 0},
		{"garbage", 0},
		{"30", 0}, // missing denominator
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := parseFrameRate(tc.input)
			if !floatNearlyEqual(got, tc.want) {
				t.Errorf("parseFrameRate(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func floatNearlyEqual(a, b float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < 0.001
}
