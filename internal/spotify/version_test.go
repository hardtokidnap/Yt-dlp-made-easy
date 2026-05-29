package spotify

import "testing"

func TestSpotdlBelow(t *testing.T) {
	cases := []struct {
		version string
		floor   string
		want    bool
	}{
		{"4.2.11", "4.5.0", true},
		{"4.5.0", "4.5.0", false},
		{"4.6.0", "4.5.0", false},
		{"4.5.1", "4.5.0", false},
		{"4.4.4", "4.5.0", true},
		{"3.9.6", "4.5.0", true},
		{"", "4.5.0", true},        // unparseable -> treat as outdated
		{"garbage", "4.5.0", true}, // unparseable -> treat as outdated
		{"4.5", "4.5.0", false},    // missing patch -> 4.5.0
	}
	for _, c := range cases {
		if got := spotdlBelow(c.version, c.floor); got != c.want {
			t.Errorf("spotdlBelow(%q, %q) = %v, want %v", c.version, c.floor, got, c.want)
		}
	}
}
