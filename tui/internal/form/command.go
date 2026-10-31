// The command line a form builds: the fields' answers gathered, then emitted along the grammar in the usage line's
// order, and quoted as a shell would need.
package form

import (
	"regexp"
	"sort"
	"strings"

	"github.com/empowerite/kit/tui/internal/syntax"
)

// Collect gathers the answers the fields hold, as Fields takes them: a toggle on and each flag of an any-of row as
// ["1"], a one-of row's choice by its ordinal, a wildcard's --x=y lines as options, and every value by its key.
func Collect(fields []Field) Answers {
	a := Answers{Options: map[string][]string{}, Values: map[string][]string{}, Chosen: map[int]int{}}
	ordinal := 0
	for _, f := range fields {
		switch f.Kind {
		case Toggle:
			if len(f.Values) > 0 {
				a.Options[f.Key] = []string{"1"}
			}
		case AnyOf:
			for _, key := range f.Values {
				a.Options[key] = []string{"1"}
			}
		case OneOf:
			a.Chosen[ordinal] = f.Chosen
			ordinal++
		case Wildcard:
			for _, line := range f.Values {
				key, value, _ := strings.Cut(line, "=")
				a.Options[key] = append(a.Options[key], value)
			}
		default:
			if len(f.Values) == 0 {
				continue
			}
			if strings.HasPrefix(f.Key, "--") {
				a.Options[f.Key] = f.Values
			} else {
				a.Values[f.Key] = f.Values
			}
		}
	}
	return a
}

// CommandLine gives the words after kit <verb> that the fields' answers build, in the usage line's order.
func CommandLine(spec syntax.Spec, fields []Field) []string {
	e := &emitter{spec: spec, answers: Collect(fields)}
	if len(spec.Syntax) == 1 {
		e.sequence(spec.Syntax[0])
	} else {
		e.group(syntax.Node{Kind: "group", Alternatives: spec.Syntax})
	}
	return e.words
}

// emitter walks the grammar with the answers, writing the words of the command line.
type emitter struct {
	spec    syntax.Spec
	answers Answers
	words   []string
	groups  int
}

func (e *emitter) sequence(nodes []syntax.Node) {
	for _, n := range nodes {
		switch n.Kind {
		case "group":
			e.group(n)
		case "literal":
			e.words = append(e.words, n.Name)
		case "flag":
			if e.answers.has(syntax.Key(n)) {
				e.words = append(e.words, "--"+n.Name)
			}
		case "option":
			if n.Wildcard {
				e.wildcard()
				continue
			}
			for _, value := range e.answers.values(syntax.Key(n)) {
				if n.Inline {
					e.words = append(e.words, "--"+n.Name+"="+value)
				} else {
					e.words = append(e.words, "--"+n.Name, value)
				}
			}
		case "operand":
			e.words = append(e.words, e.answers.values(syntax.Key(n))...)
		}
	}
}

// group walks the alternative chosen for the group, as Collect recorded it by the group's ordinal; a group of one
// alternative is walked as it stands.
func (e *emitter) group(n syntax.Node) {
	if len(n.Alternatives) == 1 {
		e.sequence(n.Alternatives[0])
		return
	}
	chosen, ok := e.answers.Chosen[e.groups]
	e.groups++
	if ok && chosen >= 0 && chosen < len(n.Alternatives) {
		e.sequence(n.Alternatives[chosen])
	}
}

// wildcard writes every --x=y answered that no named argument owns, in key order.
func (e *emitter) wildcard() {
	keys := make([]string, 0, len(e.answers.Options))
	for key := range e.answers.Options {
		if _, named := e.spec.Arguments[key]; !named && strings.HasPrefix(key, "--") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		for _, value := range e.answers.Options[key] {
			e.words = append(e.words, key+"="+value)
		}
	}
}

var bare = regexp.MustCompile(`^[A-Za-z0-9_./=@:+-]+$`)

// Quote joins the words as a shell line: a word of letters, digits, and _ . / = @ : + - stands bare, any other is
// single-quoted, with a quote inside written as '\”.
func Quote(words []string) string {
	quoted := make([]string, 0, len(words))
	for _, word := range words {
		if bare.MatchString(word) {
			quoted = append(quoted, word)
			continue
		}
		// an option with its value inline keeps its name bare and quotes the value alone: --title='kit tui'
		if name, value, ok := strings.Cut(word, "="); ok && strings.HasPrefix(word, "--") && bare.MatchString(name) {
			quoted = append(quoted, name+"="+quote(value))
			continue
		}
		quoted = append(quoted, quote(word))
	}
	return strings.Join(quoted, " ")
}

// quote single-quotes one word, a quote inside written as '\”.
func quote(word string) string {
	if bare.MatchString(word) {
		return word
	}
	return "'" + strings.ReplaceAll(word, "'", `'\''`) + "'"
}
