// A forge's labels, each drawn as a pill, as style draws one: its text on the label's color between half circles, a
// label that is a key and a value, as priority=high, two segments.
package table

import "github.com/empowerite/kit/projects/tui/internal/style"

// Label is one of an issue's or a change's labels, as kit prints it: its name, and its color as #rrggbb, or none where
// the forge gives none.
type Label struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Labelled gives a label value: what the rows filter by, the labels' names, and the labels themselves, which the pills
// draw from.
func Labelled(labels []Label) Value {
	if len(labels) == 0 {
		return Value{}
	}
	names := make([]string, len(labels))
	for k, l := range labels {
		names[k] = l.Name
	}
	return Value{Raw: names, Of: labels}
}

// LabelsIn gives the labels a value holds: its own, or, for a value that holds names alone, a label of each name.
func LabelsIn(v Value) []Label {
	if ls, ok := v.Of.([]Label); ok {
		return ls
	}
	out := make([]Label, len(v.Raw))
	for k, r := range v.Raw {
		out[k] = Label{Name: r}
	}
	return out
}

// Pills draws a value's labels, each a pill.
func Pills(f Field, v Value) []string {
	var out []string
	for _, l := range LabelsIn(v) {
		out = append(out, Pill(l))
	}
	return out
}

// Pill draws a label as style draws a Pill: its name on its forge's color, or the pastel its name picks.
func Pill(l Label) string { return style.Pill(l.Name, l.Color) }
