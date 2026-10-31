// The form drawn as text: the purpose, one block per field with its label, description, and answer, and the command
// line last. The live model draws with the same layout, adding focus and color; RenderStatic draws it plain.
package form

import (
	"strings"

	"github.com/empowerite/kit/tui/internal/syntax"
)

// Styles are the ways the layout marks text: a chosen option, an option off, a label, dim text, a refusal. Each takes
// the text and gives it back dressed; the plain styles for a static render mark a chosen option with asterisks.
type Styles struct {
	Chosen, Off, Label, Dim, Refusal, Cursor func(string) string
}

// Plain marks only what a plain text must: *chosen*.
var Plain = Styles{
	Chosen:  func(s string) string { return "*" + s + "*" },
	Off:     func(s string) string { return s },
	Label:   func(s string) string { return s },
	Dim:     func(s string) string { return s },
	Refusal: func(s string) string { return s },
	Cursor:  func(s string) string { return s },
}

const (
	gutter = 3     // the left margin before a field's label; the focus marker sits in it
	gap    = 2     // between the label column and the content
	sep    = "   " // between the names of a row
)

// Options say how a form is drawn: its width; the field that has the keyboard (-1 for none) and the option under the
// cursor in a focused row (-1 for none); the styles; Expand, which may draw a field's answer live in place of its
// values (the focused text with its caret, a list with its filter); whether the command line has the keyboard, and
// the refusal under it; and whether the form shows the way back to the palette.
type Options struct {
	Width, Focus, Cursor int
	Styles               Styles
	Expand               func(field int, indent string) string
	CommandFocus         bool
	CommandRefusal       string
	Back                 bool
}

// RenderStatic draws the form with no focus and plain styles, for golden tests.
func RenderStatic(spec syntax.Spec, fields []Field, width int) string {
	return Render(spec, fields, Options{Width: width, Focus: -1, Cursor: -1, Styles: Plain})
}

// Render draws the form: the way back when there is one, the purpose, each field's block, and the command line.
func Render(spec syntax.Spec, fields []Field, o Options) string {
	d := drawer{spec: spec, fields: fields, width: o.Width, focus: o.Focus, cursor: o.Cursor, st: o.Styles, o: o,
		column: labelColumn(fields)}
	if o.Back {
		d.b.WriteString(" " + o.Styles.Off("‹ verbs") + "\n\n")
	}
	width := o.Width
	for _, line := range wrap(spec.Purpose, width-1) {
		d.b.WriteString(" " + line + "\n")
	}
	d.b.WriteString("\n")
	for i := range fields {
		if fields[i].Group >= 0 && d.vertical(fields[i].Group) {
			continue // drawn under its option in the row
		}
		d.block(i, gutter)
	}
	d.b.WriteString("\n")
	for _, line := range wrap(strings.TrimSpace("kit "+spec.Verb+" "+Quote(CommandLine(spec, fields))), width-1) {
		if o.CommandFocus {
			line = o.Styles.Label(line)
		}
		d.b.WriteString(" " + line + "\n")
	}
	if o.CommandRefusal != "" {
		d.b.WriteString(" " + o.Styles.Refusal("❌ "+o.CommandRefusal) + "\n")
	}
	return d.b.String()
}

// drawer holds what every block shares while the form is drawn.
type drawer struct {
	spec          syntax.Spec
	fields        []Field
	width, column int
	focus, cursor int
	st            Styles
	o             Options
	b             strings.Builder
}

// labelColumn is where content starts: after the longest label, or the longest name of a row that may go vertical,
// within bounds.
func labelColumn(fields []Field) int {
	widest := 0
	for _, f := range fields {
		names := []string{f.Label}
		if f.Kind == AnyOf || f.Kind == OneOf {
			names = f.Options
		}
		for _, name := range names {
			if n := len([]rune(name)); n > widest {
				widest = n
			}
		}
	}
	if widest > 24 {
		widest = 24
	}
	return gutter + widest + gap
}

// nested gives the fields that belong to a one-of row's chosen alternative.
func (d *drawer) nested(row int) []int {
	var out []int
	for i, f := range d.fields {
		if f.Group == row {
			out = append(out, i)
		}
	}
	return out
}

// vertical says whether a row is drawn one name per line: when its chosen alternative has fields of its own, which
// nest under the name, or when the names side by side would not fit the width.
func (d *drawer) vertical(row int) bool {
	f := d.fields[row]
	if f.Kind != OneOf && f.Kind != AnyOf {
		return false
	}
	if f.Kind == OneOf {
		for _, alternative := range d.alternatives(row) {
			if len(alternative) > 1 || (len(alternative) == 1 && alternative[0].Kind != "literal" && alternative[0].Kind != "flag") {
				return true
			}
		}
	}
	return gutter+len([]rune(strings.Join(f.Options, sep)))+gap > d.width-20
}

// alternatives finds the group node a one-of row came from, by walking the grammar in the same order Fields did.
func (d *drawer) alternatives(row int) [][]syntax.Node {
	ordinal := 0
	for i := 0; i < row; i++ {
		if d.fields[i].Kind == OneOf {
			ordinal++
		}
	}
	found := [][]syntax.Node(nil)
	count := 0
	var walk func(nodes []syntax.Node)
	walk = func(nodes []syntax.Node) {
		for _, n := range nodes {
			if n.Kind != "group" || found != nil {
				continue
			}
			if len(n.Alternatives) > 1 {
				if count == ordinal {
					found = n.Alternatives
					return
				}
				count++
			}
			for _, alternative := range n.Alternatives {
				walk(alternative)
			}
		}
	}
	if len(d.spec.Syntax) > 1 {
		if ordinal == 0 {
			return d.spec.Syntax
		}
		count = 1
	}
	for _, alternative := range d.spec.Syntax {
		walk(alternative)
	}
	return found
}

// block draws one field at the margin.
func (d *drawer) block(i, margin int) {
	f := d.fields[i]
	lead := strings.Repeat(" ", margin)
	if i == d.focus {
		lead = lead[:margin-2] + "▸ "
	}
	indent := strings.Repeat(" ", d.column)
	switch f.Kind {
	case Toggle:
		name := d.st.Off(f.Label)
		if len(f.Values) > 0 {
			name = d.st.Chosen(f.Label)
		}
		d.beside(lead, name, len([]rune(f.Label)), d.column, f.Description)
	case AnyOf, OneOf:
		d.row(i, lead)
	default:
		d.beside(lead, d.st.Label(f.Label), len([]rune(f.Label)), d.column, f.Description)
		d.values(i, f, indent)
	}
	if f.Refusal != "" {
		d.b.WriteString(indent + d.st.Refusal("❌ "+f.Refusal) + "\n")
	}
	d.b.WriteString("\n")
}

// values writes a field's answers, one per line, a pick of many with its boxes, or what Expand draws in their place.
func (d *drawer) values(i int, f Field, indent string) {
	if d.o.Expand != nil {
		if s := d.o.Expand(i, indent); s != "" {
			d.b.WriteString(s)
			return
		}
	}
	for _, value := range f.Values {
		mark := ""
		if f.Kind == PickMany {
			mark = "◼ "
		}
		d.b.WriteString(indent + mark + value + "\n")
	}
}

// row draws an any-of or one-of row. Horizontal: the names, the chosen marked, with the description of the chosen (or
// the cursor's) beside. Vertical: one name per line with its description, and the chosen alternative's own fields
// under its name.
func (d *drawer) row(i int, lead string) {
	f := d.fields[i]
	on := func(k int) bool {
		if f.Kind == OneOf {
			return k == f.Chosen
		}
		for _, v := range f.Values {
			if v == f.Options[k] {
				return true
			}
		}
		return false
	}
	name := func(k int) string {
		text := d.st.Off(f.Options[k])
		if on(k) {
			text = d.st.Chosen(f.Options[k])
		}
		if i == d.focus && k == d.cursor && f.Kind == AnyOf {
			text = d.st.Cursor(text)
		}
		return text
	}
	if !d.vertical(i) {
		shown := f.Chosen
		if i == d.focus && d.cursor >= 0 {
			shown = d.cursor
		}
		if shown < 0 {
			shown = 0
		}
		names := make([]string, len(f.Options))
		for k := range f.Options {
			names[k] = name(k)
		}
		plain := strings.Join(f.Options, sep)
		column := d.column
		if gutter+len([]rune(plain))+gap > column {
			column = gutter + len([]rune(plain)) + gap
		}
		d.beside(lead, strings.Join(names, sep), len([]rune(plain)), column, d.optionDescription(f.Options[shown]))
		return
	}
	nested := d.nested(i)
	for k, option := range f.Options {
		d.beside(lead, name(k), len([]rune(option)), d.column, d.optionDescription(option))
		lead = strings.Repeat(" ", len([]rune(lead)))
		if !on(k) || f.Kind != OneOf {
			continue
		}
		for _, j := range nested {
			g := d.fields[j]
			if g.Key == option && g.Kind != AnyOf && g.Kind != OneOf {
				d.values(j, g, strings.Repeat(" ", d.column)) // the option's own value: no second label
				continue
			}
			d.b.WriteString("\n")
			d.block(j, gutter+2)
		}
	}
}

// optionDescription gives the header's text for a row's name, when the name is an argument's key.
func (d *drawer) optionDescription(option string) string {
	return d.spec.Arguments[option].Description
}

// beside writes the name at the lead and the description beside it, from the column on, wrapped to the width; a name
// wider than the column pushes the description to the next line.
func (d *drawer) beside(lead, name string, nameWidth, column int, description string) {
	indent := strings.Repeat(" ", column)
	lines := wrap(description, d.width-column)
	d.b.WriteString(lead + name)
	if len(lines) == 0 {
		d.b.WriteString("\n")
		return
	}
	if len([]rune(lead))+nameWidth+gap > column {
		d.b.WriteString("\n" + indent + lines[0] + "\n")
	} else {
		d.b.WriteString(strings.Repeat(" ", column-len([]rune(lead))-nameWidth) + lines[0] + "\n")
	}
	for _, line := range lines[1:] {
		d.b.WriteString(indent + line + "\n")
	}
}

// wrap breaks text at spaces into lines no wider than width, in runes; a word wider than the width stands alone.
func wrap(text string, width int) []string {
	if text == "" {
		return nil
	}
	if width < 10 {
		width = 10
	}
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		switch {
		case line == "":
			line = word
		case len([]rune(line))+1+len([]rune(word)) <= width:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	return append(lines, line)
}
