// A pill: a name drawn as a forge draws a label, its ends rounded by the half circles a Nerd Font gives, as a powerline
// prompt rounds its segments, the text on its color. A name that is a key and a value, as priority=high, kind:fix, or
// GitLab's scoped area::tui, split at its last separator, is two segments: the key on a color of its own, the same for
// every name with that key, then the value on the name's color, the = or : between them dropped for the round edge
// where the two colors meet.
package style

import (
	"hash/fnv"
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

// The pill's ends: powerline's half circles, in a Nerd Font's private use area. The right one also stands where a
// key's segment meets its value's.
const (
	PillLeft  = "\ue0b6"
	PillRight = "\ue0b4"
)

// pastel is the colors a pill is drawn on where it is given none, and a key's segment always: deep enough that white
// text reads on each. A name picks one by its hash, so a name, and a key, is the same color wherever it is drawn.
var pastel = []string{
	"#1d4ed8", "#0f766e", "#7c3aed", "#b45309", "#be123c", "#15803d", "#4338ca", "#a16207", "#0e7490", "#9d174d",
	"#475569", "#c2410c",
}

// Pill draws a name: its text on its color, #rrggbb or a 256-color number, between half circles in that color; a key
// and a value, as splitLabel gives them, each on its own. A name without a color is drawn on the pastel it picks.
func Pill(text, color string) string {
	key, val, split := splitLabel(text)
	back := pillColor(text, color)
	if !split {
		return PillEnd(PillLeft, back) + segment(text, back) + PillEnd(PillRight, back)
	}
	keyed := keyColor(key, back)
	meet := lipgloss.NewStyle().Foreground(lipgloss.Color(keyed)).Background(lipgloss.Color(back)).Render(PillRight)
	return PillEnd(PillLeft, keyed) + segment(key, keyed) + meet + segment(val, back) + PillEnd(PillRight, back)
}

// PillEnd draws one of a pill's half circles, PillLeft or PillRight, in its color, on whatever the line is drawn on.
func PillEnd(glyph, color string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(glyph)
}

// segment draws a pill's text on its color, in black or white, whichever reads on it.
func segment(text, back string) string {
	return lipgloss.NewStyle().Background(lipgloss.Color(back)).Foreground(ink(back)).Render(text)
}

// splitLabel gives a name's key and value, and whether it has both: a name splits at its last :, =, or ::, as GitLab
// scopes a label at its last ::, and the separator is dropped. A name that starts or ends at the split, as :wip or
// todo:, has no key, and stands whole.
func splitLabel(name string) (key, val string, ok bool) {
	k := strings.LastIndexAny(name, ":=")
	start := k
	if k > 0 && name[k] == ':' && name[k-1] == ':' {
		start = k - 1
	}
	if start <= 0 || k == len(name)-1 {
		return "", name, false
	}
	return name[:start], name[k+1:], true
}

// pillColor gives the color a name is drawn on: the one given, else the pastel the name picks.
func pillColor(name, color string) string {
	if color != "" {
		return color
	}
	return pastel[picked(name)]
}

// keyColor gives the color a key is drawn on: the pastel its name picks, or the next, where that is its value's, so the
// two segments stand apart.
func keyColor(key, val string) string {
	k := picked(key)
	if strings.EqualFold(pastel[k], val) {
		k = (k + 1) % len(pastel)
	}
	return pastel[k]
}

// picked gives the index of the pastel a name picks, by its hash.
func picked(name string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return int(h.Sum32() % uint32(len(pastel)))
}

// ink gives the text color that reads on a color: black on a light one, white on a dark one, by its luma; white on a
// color that is not #rrggbb.
func ink(back string) color.Color {
	white, black := lipgloss.Color("#ffffff"), lipgloss.Color("#000000")
	hex := strings.TrimPrefix(back, "#")
	if len(hex) != 6 {
		return white
	}
	rgb, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return white
	}
	r, g, b := float64(rgb>>16&0xff), float64(rgb>>8&0xff), float64(rgb&0xff)
	if 0.299*r+0.587*g+0.114*b > 150 {
		return black
	}
	return white
}
