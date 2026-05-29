package spotify

import (
	"strconv"
	"strings"
)

// spotdlBelow reports whether the given spotdl version string is below floor.
// Both are dotted numeric versions ("4.5.0"). spotdl --version prints a bare
// "4.2.11". Unparseable input is treated as outdated (return true) so a broken
// or pre-release install is upgraded rather than trusted.
func spotdlBelow(version, floor string) bool {
	v, ok := parseVersion(version)
	if !ok {
		return true
	}
	f, ok := parseVersion(floor)
	if !ok {
		return false
	}
	for i := 0; i < 3; i++ {
		if v[i] != f[i] {
			return v[i] < f[i]
		}
	}
	return false
}

// parseVersion turns "4.5.0" / "4.5" into [3]int. Missing components are 0.
func parseVersion(s string) ([3]int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return [3]int{}, false
	}
	parts := strings.Split(s, ".")
	var out [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}
