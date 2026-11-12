// A range is a filter's value that lets through the numbers in it rather than one value as it is: a bound, <, <=, >,
// >=, ≤, ≥, or =, before a number, as <1h or >=3, or two numbers with a - between them, both in it, as 2-5 or 1m-1h. A
// number is whole or decimal, with a unit of time after it or none: s, m, h, d, or w, in seconds, so a time's raw
// value, its seconds, compares with it.
package grid

import (
	"math"
	"strconv"
	"strings"
)

// units gives a unit of time's seconds, by its letter.
var units = map[byte]float64{'s': 1, 'm': 60, 'h': 3600, 'd': 86400, 'w': 604800}

// Range gives the numbers a filter's value lets through, from lo to hi, both in it, and whether it reads as a range.
// One that does not is a value a row must hold as it is: a name, a plain number, or a bound without a number.
func Range(token string) (lo, hi float64, ok bool) {
	for _, b := range []struct {
		prefix string
		bounds func(n float64) (float64, float64)
	}{
		{"<=", func(n float64) (float64, float64) { return math.Inf(-1), n }},
		{"≤", func(n float64) (float64, float64) { return math.Inf(-1), n }},
		{">=", func(n float64) (float64, float64) { return n, math.Inf(1) }},
		{"≥", func(n float64) (float64, float64) { return n, math.Inf(1) }},
		{"<", func(n float64) (float64, float64) { return math.Inf(-1), math.Nextafter(n, math.Inf(-1)) }},
		{">", func(n float64) (float64, float64) { return math.Nextafter(n, math.Inf(1)), math.Inf(1) }},
		{"=", func(n float64) (float64, float64) { return n, n }},
	} {
		if rest, found := strings.CutPrefix(token, b.prefix); found {
			n, ok := quantity(rest)
			if !ok {
				return 0, 0, false
			}
			lo, hi = b.bounds(n)
			return lo, hi, true
		}
	}
	from, to, found := strings.Cut(token, "-")
	if !found {
		return 0, 0, false
	}
	a, okA := quantity(from)
	z, okZ := quantity(to)
	if !okA || !okZ || a > z {
		return 0, 0, false
	}
	return a, z, true
}

// quantity reads a number, with a unit of time after it or none, in seconds.
func quantity(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	scale := 1.0
	if n := len(s); n > 1 && units[s[n-1]] != 0 {
		scale, s = units[s[n-1]], s[:n-1]
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, false
	}
	return n * scale, true
}

// inRange says whether one of a row's raw values is a number the range from lo to hi holds.
func inRange(raw []string, lo, hi float64) bool {
	for _, r := range raw {
		if n, err := strconv.ParseFloat(r, 64); err == nil && n >= lo && n <= hi {
			return true
		}
	}
	return false
}
