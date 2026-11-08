// A host for the controls' tests: a table of its own fields and rows, its panel kept as the controls set it.
package table

import "github.com/empowerite/kit/projects/tui/internal/grid"

// host is a screen of one table: its fields, its panel, the values its rows hold, by field, its caller, the renderings
// drawn, and its size.
type host struct {
	fields []Field
	panel  grid.Panel
	values map[string][]Value
	me     string
	drawn  map[string]string
	w, h   int
	sets   int // how many times the panel was set
}

func (h *host) Fields() []Field          { return h.fields }
func (h *host) Panel() grid.Panel        { return h.panel }
func (h *host) SetPanel(p grid.Panel)    { h.panel, h.sets = p, h.sets+1 }
func (h *host) Values(id string) []Value { return h.values[id] }
func (h *host) Me() string               { return h.me }
func (h *host) Drawn(id string) string   { return h.drawn[id] }
func (h *host) Size() (int, int)         { return h.w, h.h }
func (h *host) Name() string             { return "things" }
func (h *host) Pane() (int, int)         { return 4, h.w - 1 }
func (h *host) Label(id, name string) Label {
	for _, v := range h.values[id] {
		for _, l := range LabelsIn(v) {
			if l.Name == name {
				return l
			}
		}
	}
	return Label{Name: name}
}

// who is a person field whose filter offers @me.
var who = Field{ID: "who", Label: "who", Type: "person", Renderings: []string{"text"}, Filter: true}

// tags is a field of labels.
var tags = Field{ID: "tags", Label: "tags", Type: "label", Renderings: []string{"pills"}, Filter: true}

// screen gives a host of state, who, tags, and title, the rows holding a few values of each.
func screen() *host {
	return &host{fields: []Field{state, who, tags, title}, w: 80, h: 24,
		values: map[string][]Value{
			"state": {One("open"), One("merged")},
			"who":   {One("bob"), One("ada"), {}},
			"tags":  {Labelled([]Label{{"bug", "#ff0000"}})},
		},
		drawn: map[string]string{"state": "glyph+text"}}
}
