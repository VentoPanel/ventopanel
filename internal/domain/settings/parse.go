package settings

import (
	"strconv"
	"strings"
)

// ParseBool parses common truthy strings; empty uses def.
func ParseBool(s string, def bool) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return def
	}
	switch s {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return def
	}
}

// ParseIntBounded parses decimal int; empty or invalid uses def, then clamps to [min,max].
func ParseIntBounded(s string, def, min, max int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return clampInt(def, min, max)
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return clampInt(def, min, max)
	}
	return clampInt(n, min, max)
}

func clampInt(n, min, max int) int {
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

// ClampInt returns n constrained to [min, max].
func ClampInt(n, min, max int) int {
	return clampInt(n, min, max)
}
