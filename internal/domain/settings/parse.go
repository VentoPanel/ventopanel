package settings

import (
	"strconv"
	"strings"
)

// ParseBool parses common truthy strings; empty string uses def.
func ParseBool(s string, def bool) bool {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return def
	}
}

// ParseIntBounded parses a decimal integer and clamps it to [min, max].
// On empty string or parse failure, def is used before clamping.
func ParseIntBounded(s string, def, min, max int) int {
	s = strings.TrimSpace(s)
	n := def
	if s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			n = v
		}
	}
	return ClampInt(n, min, max)
}

// ClampInt returns n constrained to [min, max].
func ClampInt(n, min, max int) int {
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
