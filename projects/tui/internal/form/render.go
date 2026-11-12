// The form drawn as text: the purpose, one block per field with its label, description, and answer, and the command
// line last. The live model draws with the same layout, adding focus and color; RenderStatic draws it plain. Widths
// are terminal cells, so a two-cell glyph never pushes a line past the edge.
package form

import (
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/syntax"
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
	gutter   = 3          // the left margin before a field's label; the focus marker sits in it
	gap      = 2          // between the label column and the content
	sep      = "   "      // between the names of a row
	required = "required" // the dim tag after a required field's label until the field has an answer
	oneOf    = "one of"   // the heading over a one-of row's names, one per line
)

// Options say how a form is drawn: its width; the field that has the keyboard (-1 for none) and the option under the
// cursor in a focused row (-1 for none); the styles; Expand, which may draw a field's answer live in place of its
// values (the focused text with its caret, a list with its filter), and gives the line of what it drew that holds a
// list's cursor, -1 for none; the note under the focused field (a list's line about a row, or what kit refused);
// whether the command line has the keyboard, and the refusal under it; and whether the form shows the way back to the
// palette, drawn at the top so it never moves.
type Options struct {
	Width, Focus, Cursor int
	Styles               Styles
	Expand               func(field int, indent string) (string, int)
	Note                 string
	CommandFocus         bool
	CommandRefusal       string
	Back                 bool
}

// RenderStatic draws the form with no focus and plain styles, for golden tests.
func RenderStatic(spec syntax.Spec, fields []Field, width int) string {
	return Render(spec, fields, Options{Width: width, Focus: -1, Cursor: -1, Styles: Plain})
}

// Render draws the form: the way back when there is one, the verb and its purpose, each field's block, and the command
// line.
func Render(spec syntax.Spec, fields []Field, o Options) string {
	content, _, _, _, _ := Layout(spec, fields, o)
	return content
}

// Layout draws the form as Render does, and says where the keyboard is on it: how many lines on top stay in place,
// the way back, the first and last line of the field that has the keyboard, or of the command line, and the line of
// a list's cursor, each -1 for none. A short terminal shows the top and those lines.
func Layout(spec syntax.Spec, fields []Field, o Options) (content string, header, first, last, at int) {
	d := drawer{spec: spec, fields: fields, width: o.Width, focus: o.Focus, cursor: o.Cursor, st: o.Styles, o: o,
		column: labelColumn(fields), first: -1, last: -1, at: -1}
	if o.Back {
		d.b.WriteString(" " + o.Styles.Off("‹ verbs") + "\n\n")
	}
	width := o.Width
	header = d.count()
	for _, line := range pane.Wrap("kit "+spec.Verb+" - "+spec.Purpose, width-1) {
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
	before := d.count()
	for _, line := range pane.Break(strings.TrimSpace("kit "+spec.Verb+" "+Quote(CommandLine(spec, fields))), width-1) {
		if o.CommandFocus {
			line = o.Styles.Label(line)
		}
		d.b.WriteString(" " + line + "\n")
	}
	if o.CommandRefusal != "" {
		d.b.WriteString(" " + o.Styles.Refusal("❌ "+o.CommandRefusal) + "\n")
	}
	if o.CommandFocus {
		d.first, d.last = before, d.count()-1
	}
	return d.b.String(), header, d.first, d.last, d.at
}

// window clips a form to the height: the header stays, and the lines under it are the window that shows the focused
// lines, from their first when they all fit, else from their first still, and the line at, a list's cursor, always.
// A line in dim says how many lines the window hides above it, and one how many it hides below, each in a line's
// place. A height of zero clips nothing.
func window(content string, height, header, first, last, at int, dim func(string) string) string {
	if height <= 0 {
		return content
	}
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	if len(lines) <= height {
		return content
	}
	header = min(header, height-1)
	body, room := lines[header:], height-header
	first, last, at = max(first-header, 0), last-header, at-header
	s := pane.Frame(len(body), room, 0, first, max(first, last))
	if at >= 0 && (at < s.Start || at >= s.End) {
		s = pane.Frame(len(body), room, 0, at, at)
	}
	shown := lines[:header:header]
	if s.Above {
		shown = append(shown, strings.Repeat(" ", gutter)+dim(pane.Above(s.Start)))
	}
	shown = append(shown, body[s.Start:s.End]...)
	if s.Below {
		shown = append(shown, strings.Repeat(" ", gutter)+dim(pane.More(len(body)-s.End)))
	}
	return strings.Join(shown, "\n") + "\n"
}

// drawer holds what every block shares while the form is drawn, the first and last line of the focused field, and the
// line of a list's cursor.
type drawer struct {
	spec          syntax.Spec
	fields        []Field
	width, column int
	focus, cursor int
	first, last   int
	at            int
	st            Styles
	o             Options
	b             strings.Builder
}

// count gives the lines drawn so far.
func (d *drawer) count() int { return strings.Count(d.b.String(), "\n") }

// mark notes the lines drawn since before as the focused field's, when i is that field.
func (d *drawer) mark(i, before int) {
	if i != d.focus {
		return
	}
	if d.first < 0 || before < d.first {
		d.first = before
	}
	d.last = max(d.last, d.count()-1)
}

// labelColumn is where content starts: after the longest label, or the longest name of a row that may go vertical,
// each at its own margin (a field under a row's name sits two columns in), with room kept for the required tag after
// every label that may carry one, so nothing moves when the tag goes. A label counts for 24 columns at most.
func labelColumn(fields []Field) int {
	column := gutter + gap
	for _, f := range fields {
		if unlabeled(fields, f) {
			continue
		}
		margin, names, reserve := gutter, []string{f.Label}, 0
		if f.Group >= 0 {
			margin += 2
		}
		if f.Kind == AnyOf || f.Kind == OneOf {
			names = f.Options
		} else if f.Required && f.Kind != Toggle {
			reserve = 1 + len(required)
		}
		for _, name := range names {
			column = max(column, margin+min(ansi.StringWidth(name)+reserve, 24)+gap)
		}
	}
	return column
}

// unlabeled says whether a field draws no label of its own: the value of a row's option, which the name in the row
// already says, drawn under that name, as --message's lines are under --message.
func unlabeled(fields []Field, f Field) bool {
	return f.Group >= 0 && f.Kind != AnyOf && f.Kind != OneOf && slices.Contains(fields[f.Group].Options, f.Key)
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
	if f.Kind == OneOf {
		for _, alternative := range d.alternatives(row) {
			if len(alternative) > 1 || (len(alternative) == 1 && alternative[0].Kind != "literal" && alternative[0].Kind != "flag") {
				return true
			}
		}
	}
	return gutter+ansi.StringWidth(strings.Join(f.Options, sep))+gap > d.width-20
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

// block draws one field at the margin: its label, tagged while a required field wants an answer, its description
// beside, its answer, and the note under it when it has the keyboard.
func (d *drawer) block(i, margin int) {
	f := d.fields[i]
	before := d.count()
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
		d.beside(lead, name, ansi.StringWidth(f.Label), d.column, f.Description)
		d.note(i, indent)
	case AnyOf, OneOf:
		d.row(i, lead)
		d.note(i, indent)
	default:
		name, width := d.st.Label(f.Label), ansi.StringWidth(f.Label)
		if unanswered(f) {
			name, width = name+" "+d.st.Dim(required), width+1+len(required)
		}
		d.beside(lead, name, width, d.column, f.Description)
		d.values(i, f, indent)
	}
	if f.Refusal != "" {
		d.b.WriteString(indent + d.st.Refusal("❌ "+f.Refusal) + "\n")
	}
	d.mark(i, before)
	d.b.WriteString("\n")
}

// values writes a field's answers, one per line, a pick of many with its boxes, or what Expand draws in their place,
// then the note under the field when it has the keyboard.
func (d *drawer) values(i int, f Field, indent string) {
	before := d.count()
	drawn, at := "", -1
	if d.o.Expand != nil {
		drawn, at = d.o.Expand(i, indent)
	}
	if at >= 0 {
		d.at = before + at
	}
	d.b.WriteString(drawn)
	for _, value := range f.Values {
		if drawn != "" {
			break
		}
		mark := ""
		if f.Kind == PickMany {
			mark = "◼ "
		}
		d.b.WriteString(indent + mark + value + "\n")
	}
	d.note(i, indent)
	d.mark(i, before)
}

// note writes the line under the field that has the keyboard, when there is one: red, with the glyph a refusal
// carries, wrapped to the width left beside the indent.
func (d *drawer) note(i int, indent string) {
	if i != d.focus || d.o.Note == "" {
		return
	}
	for _, line := range pane.Wrap("❌ "+d.o.Note, d.width-ansi.StringWidth(indent)) {
		d.b.WriteString(indent + d.st.Refusal(line) + "\n")
	}
}

// row draws an any-of or one-of row. Horizontal: the names, the chosen marked, with the description of the chosen (or
// the cursor's) beside. Vertical: one name per line with its description, and the chosen alternative's own fields
// under its name. A required one-of row with nothing chosen carries the tag: after its names on one line, or heading
// the names one per line, where no one name may carry it.
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
		if i == d.focus && k == d.cursor {
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
		row, width, reserve := strings.Join(names, sep), ansi.StringWidth(plain), 0
		if f.Required {
			reserve = 1 + len(required) // the room is kept once one is chosen, so the description never moves
		}
		if unanswered(f) {
			row, width = row+" "+d.st.Dim(required), width+reserve
		}
		column := max(d.column, gutter+ansi.StringWidth(plain)+reserve+gap)
		d.beside(lead, row, width, column, d.optionDescription(f.Options[shown]))
		return
	}
	nested := d.nested(i)
	switch {
	case f.Kind == OneOf && unanswered(f):
		d.b.WriteString(strings.Repeat(" ", ansi.StringWidth(lead)) + oneOf + " " + d.st.Dim(required) + "\n")
	case f.Kind == OneOf:
		d.b.WriteString(strings.Repeat(" ", ansi.StringWidth(lead)) + oneOf + "\n")
	}
	room := d.width - ansi.StringWidth(lead) // a name wider than the screen, a whole usage line, is clipped
	for k, option := range f.Options {
		d.beside(lead, pane.Clip(name(k), room), min(ansi.StringWidth(option), room), d.column, d.optionDescription(option))
		lead = strings.Repeat(" ", ansi.StringWidth(lead))
		if !on(k) || f.Kind != OneOf {
			continue
		}
		apart := false // a blank line sets the alternative's own fields apart from its name, and each ends with one
		for _, j := range nested {
			g := d.fields[j]
			if g.Key == option && g.Kind == Toggle {
				continue // the name itself says it: a required flag has no value to show
			}
			if g.Key == option && g.Kind != AnyOf && g.Kind != OneOf {
				d.values(j, g, strings.Repeat(" ", d.column)) // the option's own value: no second label
				continue
			}
			if !apart {
				d.b.WriteString("\n")
				apart = true
			}
			d.block(j, gutter+2)
		}
	}
}

// optionDescription gives the header's text for a row's name, when the name is an argument's key, or the verb's
// purpose for kit and the verb, the alternative that adds no word.
func (d *drawer) optionDescription(option string) string {
	if option == "kit "+d.spec.Verb {
		return d.spec.Purpose
	}
	return d.spec.Arguments[option].Description
}

// beside writes the name at the lead and the description beside it, from the column on, wrapped to the width; a name
// wider than the column pushes the description to the next line.
func (d *drawer) beside(lead, name string, nameWidth, column int, description string) {
	indent := strings.Repeat(" ", column)
	lines := pane.Wrap(description, d.width-column)
	d.b.WriteString(lead + name)
	if len(lines) == 0 {
		d.b.WriteString("\n")
		return
	}
	if ansi.StringWidth(lead)+nameWidth+gap > column {
		d.b.WriteString("\n" + indent + lines[0] + "\n")
	} else {
		d.b.WriteString(strings.Repeat(" ", column-ansi.StringWidth(lead)-nameWidth) + lines[0] + "\n")
	}
	for _, line := range lines[1:] {
		d.b.WriteString(indent + line + "\n")
	}
}
