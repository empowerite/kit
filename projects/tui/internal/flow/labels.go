// An issue's or a change's labels, as its forge names and colors them, each drawn as a pill: its ends rounded by the
// half circles a Nerd Font gives, as a powerline prompt rounds its segments, the text on the label's color. A label
// that is a key and a value, as priority=high, kind:fix, or GitLab's scoped area::tui, split at its last separator, is
// two segments: the key on a color of its own, the same for every label with that key, then the value on the label's
// color, the = or : between them dropped for the round edge where the two colors meet.
package flow

import (
	"hash/fnv"
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

// label is one of an issue's or a change's labels, as kit prints it: its name, and its color as #rrggbb, or none where
// the forge gives none.
type label struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// The pill's ends: powerline's half circles, in a Nerd Font's private use area. The right one also stands where a
// key's segment meets its value's.
const (
	capLeft  = ""
	capRight = ""
)

// pastel is the colors a label is drawn on where its forge gives it none, and a key's segment always: deep enough that
// white text reads on each. A name picks one by its hash, so a label, and a key, is the same color on every row.
var pastel = []string{
	"#1d4ed8", "#0f766e", "#7c3aed", "#b45309", "#be123c", "#15803d", "#4338ca", "#a16207", "#0e7490", "#9d174d",
	"#475569", "#c2410c",
}

// labelled gives a label value: what the rows filter by, the labels' names, and the labels themselves, which the pills
// draw from.
func labelled(labels []label) value {
	if len(labels) == 0 {
		return value{}
	}
	names := make([]string, len(labels))
	for k, l := range labels {
		names[k] = l.Name
	}
	return value{Raw: names, Of: labels}
}

// labelsIn gives the labels a value holds: its own, or, for a value that holds names alone, a label of each name.
func labelsIn(v value) []label {
	if ls, ok := v.Of.([]label); ok {
		return ls
	}
	out := make([]label, len(v.Raw))
	for k, r := range v.Raw {
		out[k] = label{Name: r}
	}
	return out
}

// labelNamed gives the label of a name, with its color, as one of the rows the selected stage loaded holds it in the
// field, by id; a name no row holds is a label without a color.
func (m Model) labelNamed(id, name string) label {
	for _, r := range m.records(m.stages[m.stage]) {
		for _, l := range labelsIn(r.values[id]) {
			if l.Name == name {
				return l
			}
		}
	}
	return label{Name: name}
}

// pills draws a value's labels, each a pill.
func pills(f field, v value) []string {
	var out []string
	for _, l := range labelsIn(v) {
		out = append(out, pill(l))
	}
	return out
}

// splitLabel gives a label's key and value, and whether it has both: a name splits at its last :, =, or ::, GitLab's
// scoped label, as GitLab scopes one at its last ::, and the separator is dropped. A name that starts or ends at the
// split, as :wip or todo:, has no key, and stands whole.
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

// pill draws a label: its text on its color, between half circles in that color; a key and a value, as splitLabel
// gives them, each on its own.
func pill(l label) string {
	key, val, split := splitLabel(l.Name)
	back := labelColor(l)
	if !split {
		return end(capLeft, back) + segment(l.Name, back) + end(capRight, back)
	}
	keyed := keyColor(key, back)
	meet := lipgloss.NewStyle().Foreground(lipgloss.Color(keyed)).Background(lipgloss.Color(back)).Render(capRight)
	return end(capLeft, keyed) + segment(key, keyed) + meet + segment(val, back) + end(capRight, back)
}

// end draws one of a pill's half circles in its color, on whatever the line is drawn on.
func end(glyph, back string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(back)).Render(glyph)
}

// segment draws a pill's text on its color, in black or white, whichever reads on it.
func segment(text, back string) string {
	return lipgloss.NewStyle().Background(lipgloss.Color(back)).Foreground(ink(back)).Render(text)
}

// labelColor gives the color a label is drawn on: its forge's, else the pastel its name picks.
func labelColor(l label) string {
	if l.Color != "" {
		return l.Color
	}
	return pastel[picked(l.Name)]
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
