// Package form turns a verb's grammar and the answers so far into the fields of its form, and the fields back into the
// command line they build. One field per node of the usage line, in its order: a lone optional flag is a toggle,
// optional flags side by side one any-of row, a group of alternatives one one-of row whose chosen alternative's nodes
// follow it, a value a text line or a pick from the verb's choices, a repeated one lines or a pick of many, and a
// wildcard option lines of --x=y.
package form

import (
	"fmt"
	"slices"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/syntax"
)

// FieldKind is how a field is drawn and answered.
type FieldKind int

// The kinds, as the spec's table of fields names them.
const (
	Toggle   FieldKind = iota // one optional flag: on or off
	AnyOf                     // optional flags side by side: each on or off
	OneOf                     // a group of alternatives: one chosen, or none when the group is optional
	Text                      // one value, typed
	Lines                     // several values, one per line
	Pick                      // one value from the verb's choices, closed or open
	PickMany                  // several values from the verb's choices
	Wildcard                  // any --x=y, one per line
)

var kindNames = [...]string{"Toggle", "AnyOf", "OneOf", "Text", "Lines", "Pick", "PickMany", "Wildcard"}

func (k FieldKind) String() string { return kindNames[k] }

// Field is one field of the form. Key is the argument's key, as kit syntax's arguments object is keyed; Label is what
// the form shows for it; Description the header's text. Options are a row's names (AnyOf, OneOf); Values the answer
// (a toggle's on state is ["1"], a row's the keys on, a one-of's nothing: Chosen is its alternative, -1 for none).
// Choices is closed, open, or empty, for a pick. Required is a part the command line needs; Repeat whether the field
// takes several values. Refusal is the line the verb would print about the answer as it stands. Group is the index
// of the one-of field whose chosen alternative holds this field, -1 at the top.
type Field struct {
	Key, Label, Description string
	Kind                    FieldKind
	Choices                 string
	Options                 []string
	Values                  []string
	Chosen                  int
	Required, Repeat        bool
	Refusal                 string
	Group                   int
}

// Answers is what the person has given so far: the sorted command line, as kit syntax --match gives it, and the
// alternative chosen in each one-of row, by the row's ordinal in the walk, when nothing given decides it.
type Answers struct {
	Literals []string
	Options  map[string][]string
	Values   map[string][]string
	Chosen   map[int]int
}

// FromMatch takes the answers from a judged command line.
func FromMatch(m syntax.Match) Answers {
	return Answers{Literals: m.Literals, Options: m.Options, Values: m.Values}
}

// values gives the answer to a key: an option's or operand's values.
func (a Answers) values(key string) []string {
	if v, ok := a.Options[key]; ok && len(v) > 0 {
		return v
	}
	return a.Values[key]
}

// has says whether the key was answered at all.
func (a Answers) has(key string) bool { return len(a.values(key)) > 0 }

// walker builds the fields along the grammar, consuming the literals given in order and counting one-of rows.
type walker struct {
	spec     syntax.Spec
	answers  Answers
	fields   []Field
	literals int
	groups   int
}

// Fields gives the form's fields for the spec and the answers so far, in the usage line's order.
func Fields(spec syntax.Spec, a Answers) ([]Field, error) {
	if spec.Error != "" {
		return nil, fmt.Errorf("kit %s: usage does not parse: %s", spec.Verb, spec.Error)
	}
	w := &walker{spec: spec, answers: a}
	if len(spec.Syntax) == 1 {
		w.sequence(spec.Syntax[0], -1, true)
	} else {
		w.group(syntax.Node{Kind: "group", Alternatives: spec.Syntax}, -1, true)
	}
	return w.fields, nil
}

// describe gives the header's text for a key.
func (w *walker) describe(key string) (string, string) {
	a := w.spec.Arguments[key]
	return a.Description, a.Choices
}

// sequence walks one sequence, folding runs of optional lone flags into one any-of row. Its nodes are required when
// the command line needs the sequence: at the top, or in an alternative chosen, but not inside [ ].
func (w *walker) sequence(nodes []syntax.Node, group int, required bool) {
	for i := 0; i < len(nodes); i++ {
		n := nodes[i]
		if flag, ok := loneFlag(n); ok {
			j := i
			var flags []syntax.Node
			for ; j < len(nodes); j++ {
				f, ok := loneFlag(nodes[j])
				if !ok {
					break
				}
				flags = append(flags, f)
			}
			if len(flags) == 1 {
				w.toggle(flag, false, group)
			} else {
				w.anyOf(flags, group)
			}
			i = j - 1
			continue
		}
		switch n.Kind {
		case "group":
			w.group(n, group, required)
		case "literal":
			// a literal is fixed by its alternative; at the top it names the operand that follows it, as kit new's file
			// <path>, and in a row's alternative the row already shows it
			if i+1 < len(nodes) && nodes[i+1].Kind == "operand" {
				prefix := n.Name + " "
				if group >= 0 && i == 0 {
					prefix = ""
				}
				w.value(nodes[i+1], prefix, required, group)
				i++
			}
		case "flag":
			w.toggle(n, required, group)
		default:
			w.value(n, "", required, group)
		}
	}
}

// loneFlag says whether a node is an optional group of one flag, the shape of [--all].
func loneFlag(n syntax.Node) (syntax.Node, bool) {
	if n.Kind == "group" && n.Optional && len(n.Alternatives) == 1 && len(n.Alternatives[0]) == 1 && n.Alternatives[0][0].Kind == "flag" {
		return n.Alternatives[0][0], true
	}
	return syntax.Node{}, false
}

func (w *walker) toggle(flag syntax.Node, required bool, group int) {
	key := syntax.Key(flag)
	description, _ := w.describe(key)
	f := Field{Key: key, Label: key, Description: description, Kind: Toggle, Required: required, Chosen: -1, Group: group}
	if w.answers.has(key) || required {
		f.Values = []string{"1"}
	}
	w.fields = append(w.fields, f)
}

func (w *walker) anyOf(flags []syntax.Node, group int) {
	f := Field{Kind: AnyOf, Chosen: -1, Group: group}
	for _, flag := range flags {
		key := syntax.Key(flag)
		f.Options = append(f.Options, key)
		if w.answers.has(key) {
			f.Values = append(f.Values, key)
		}
	}
	f.Key = strings.Join(f.Options, " ")
	f.Label = f.Key
	w.fields = append(w.fields, f)
}

// value adds a field for an option or operand: text or a pick for one value, lines or a pick of many when it repeats,
// and lines of --x=y for a wildcard option.
func (w *walker) value(n syntax.Node, prefix string, required bool, group int) {
	key := syntax.Key(n)
	description, choices := w.describe(key)
	f := Field{Key: key, Label: prefix + key, Description: description, Choices: choices, Required: required,
		Repeat: n.Repeat, Chosen: -1, Group: group, Values: w.answers.values(key)}
	switch {
	case n.Wildcard:
		f.Kind = Wildcard
		f.Label = prefix + "--<" + n.Name + ">=<" + n.Value + ">"
		f.Values = w.wildcardValues()
	case n.Repeat && choices != "":
		f.Kind = PickMany
	case n.Repeat:
		f.Kind = Lines
	case choices != "":
		f.Kind = Pick
	default:
		f.Kind = Text
	}
	if n.Kind == "operand" {
		f.Label = prefix + "<" + n.Name + ">"
		if n.Repeat {
			f.Label += "…"
		}
	}
	w.fields = append(w.fields, f)
}

// wildcardValues gives every --x=y answered that no named argument of the spec owns.
func (w *walker) wildcardValues() []string {
	var out []string
	for key, values := range w.answers.Options {
		if _, named := w.spec.Arguments[key]; named || !strings.HasPrefix(key, "--") {
			continue
		}
		for _, v := range values {
			out = append(out, key+"="+v)
		}
	}
	return out
}

// group adds a one-of row for a group of alternatives, and walks the chosen alternative after it, whose nodes the
// command line then needs. A group of one alternative has no row: its nodes stand on their own, optional inside [ ].
// At the top, an alternative that adds no word, every node of it inside [ ], as kit flow's picture with its options,
// makes the row optional: it is chosen while no other is, and named by the command it leaves, kit and the verb.
func (w *walker) group(n syntax.Node, parent int, required bool) {
	required = required && !n.Optional
	if len(n.Alternatives) == 1 {
		if n.Optional {
			// an optional alternative of one value, [<path>…]: the field is optional
			if len(n.Alternatives[0]) == 1 && n.Alternatives[0][0].Kind != "group" {
				w.value(n.Alternatives[0][0], "", false, parent)
				return
			}
		}
		w.sequence(n.Alternatives[0], parent, required)
		return
	}
	ordinal := w.groups
	w.groups++
	bare := -1
	if parent < 0 {
		bare = bareAlternative(n.Alternatives)
	}
	f := Field{Kind: OneOf, Required: required && bare < 0, Chosen: -1, Group: parent}
	f.Options = labels(n.Alternatives)
	if bare >= 0 {
		f.Options[bare] = "kit " + w.spec.Verb
	}
	f.Key = strings.Join(f.Options, " | ")
	f.Label = f.Key
	if f.Chosen = w.choose(n.Alternatives, ordinal); f.Chosen < 0 {
		f.Chosen = bare
	}
	switch {
	case f.Chosen >= 0 && f.Chosen == bare:
		f.Description = w.spec.Purpose
	case f.Chosen >= 0:
		first := n.Alternatives[f.Chosen][0]
		f.Description, _ = w.describe(syntax.Key(first))
	}
	index := len(w.fields)
	w.fields = append(w.fields, f)
	if f.Chosen >= 0 {
		w.sequence(n.Alternatives[f.Chosen], index, true)
	}
}

// bareAlternative gives the alternative that adds no word to the command line, every node of it inside [ ], or -1.
func bareAlternative(alternatives [][]syntax.Node) int {
	for i, alternative := range alternatives {
		if !slices.ContainsFunc(alternative, func(n syntax.Node) bool { return n.Kind != "group" || !n.Optional }) {
			return i
		}
	}
	return -1
}

// choose picks the alternative the answers select: the one chosen by hand for this row, else the one the next
// literal given opens, else the one that holds an answered key, else -1. A literal that opens the alternative itself
// is consumed here; one that opens it through a group it starts with is left for that group's row to consume.
func (w *walker) choose(alternatives [][]syntax.Node, ordinal int) int {
	if chosen, ok := w.answers.Chosen[ordinal]; ok {
		if chosen >= 0 && chosen < len(alternatives) && len(alternatives[chosen]) > 0 && alternatives[chosen][0].Kind == "literal" {
			w.literals++
		}
		return chosen
	}
	if w.literals < len(w.answers.Literals) {
		next := w.answers.Literals[w.literals]
		for i, alternative := range alternatives {
			if opens(alternative, next) {
				if alternative[0].Kind == "literal" {
					w.literals++
				}
				return i
			}
		}
	}
	for i, alternative := range alternatives {
		if w.holds(alternative) {
			return i
		}
	}
	return -1
}

// opens says whether a literal opens the alternative, as the matcher judges it: it is the alternative's first node,
// or it opens an alternative of the required group the alternative starts with, as add opens
// kit packages (select | add | remove) <package>….
func opens(alternative []syntax.Node, literal string) bool {
	if len(alternative) == 0 {
		return false
	}
	first := alternative[0]
	if first.Kind == "literal" {
		return first.Name == literal
	}
	if first.Kind != "group" || first.Optional {
		return false
	}
	for _, inner := range first.Alternatives {
		if opens(inner, literal) {
			return true
		}
	}
	return false
}

// holds says whether any key in the alternative, groups included, was answered.
func (w *walker) holds(nodes []syntax.Node) bool {
	for _, n := range nodes {
		if n.Kind == "group" {
			for _, alternative := range n.Alternatives {
				if w.holds(alternative) {
					return true
				}
			}
			continue
		}
		if n.Kind == "literal" {
			continue
		}
		if n.Wildcard {
			if len(w.wildcardValues()) > 0 {
				return true
			}
			continue
		}
		if w.answers.has(syntax.Key(n)) {
			return true
		}
	}
	return false
}

// labels names a row's alternatives: each by the key of the node it opens with, or by every key in it when two would
// otherwise read the same, as kit syntax's lines that all take a <verb>.
func labels(alternatives [][]syntax.Node) []string {
	short := make([]string, len(alternatives))
	seen := map[string]int{}
	for i, alternative := range alternatives {
		if len(alternative) > 0 && alternative[0].Kind != "group" {
			short[i] = syntax.Key(alternative[0])
		} else {
			short[i] = label(alternative)
		}
		seen[short[i]]++
	}
	for i, alternative := range alternatives {
		if seen[short[i]] > 1 {
			short[i] = label(alternative)
		}
	}
	return short
}

// label names an alternative in a one-of row: the key of the node it opens with (a literal's word, an operand's or
// flag's key), or, when it opens with a group, the keys of its nodes.
func label(alternative []syntax.Node) string {
	parts := make([]string, 0, len(alternative))
	for _, n := range alternative {
		switch {
		case n.Kind == "group" && len(n.Alternatives) == 1 && len(n.Alternatives[0]) == 1:
			parts = append(parts, syntax.Key(n.Alternatives[0][0]))
		case n.Kind == "group":
			parts = append(parts, syntax.Render([]syntax.Node{n}))
		default:
			parts = append(parts, syntax.Key(n))
		}
	}
	return strings.Join(parts, " ")
}

// unanswered says whether a required field still wants an answer: a one-of row with nothing chosen, or a value with
// none. A toggle or an any-of row never does: a required flag is on, and flags side by side are each optional.
func unanswered(f Field) bool {
	if !f.Required {
		return false
	}
	switch f.Kind {
	case OneOf:
		return f.Chosen < 0
	case Toggle, AnyOf:
		return false
	}
	return len(f.Values) == 0
}

// FirstEmptyRequired gives the index of the first required field with no answer, or -1 when every one has one.
func FirstEmptyRequired(fields []Field) int {
	for i, f := range fields {
		if unanswered(f) {
			return i
		}
	}
	return -1
}
