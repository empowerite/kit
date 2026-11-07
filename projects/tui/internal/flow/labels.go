// An issue's or a change's labels, as its forge names and colors them, each drawn as a pill, as style draws one: its
// text on the label's color between half circles, a label that is a key and a value, as priority=high, two segments.
package flow

import "github.com/empowerite/kit/projects/tui/internal/style"

// label is one of an issue's or a change's labels, as kit prints it: its name, and its color as #rrggbb, or none where
// the forge gives none.
type label struct {
	Name  string `json:"name"`
	Color string `json:"color"`
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

// pill draws a label as style draws a pill: its name on its forge's color, or the pastel its name picks.
func pill(l label) string { return style.Pill(l.Name, l.Color) }
