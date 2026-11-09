package shell

import (
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// The keys bar is made from where the keyboard is, never written per screen: the path from the screen, first, down to
// the control that has the keyboard, last, each control on it a segment of the keys it answers. The bar draws each
// segment's keys between a rule and the next, the screen's leftmost. A control deeper on the path that takes a press
// hides an ancestor's key for it, as a search hides = once = types; a control that types hides every ancestor's key that
// types a character; and one that takes every press, as an open menu, hides every ancestor's key but those it lets
// through. A segment left with no key is not drawn. Too narrow, the bar drops whole segments from the screen's end,
// with … in their place, so the keys that act where the keyboard is are the last to go.

// Binding is one key a control answers, as the keys bar names it: the key as drawn, what it does, and the presses it
// answers, as Bubble Tea names them, by which a control deeper on the path hides it. A binding with no key, as type to
// search, says what the control does with what is typed.
type Binding struct {
	Key, Does string
	Presses   []string
	keys      []string // the keys a run that does the same thing merged into this one, Key among them, as merged draws it
}

// Bind gives a binding: the key as drawn, what it does, and the presses it answers; none says the key as drawn is its
// one press.
func Bind(key, does string, presses ...string) Binding {
	if len(presses) == 0 {
		presses = []string{key}
	}
	return Binding{Key: key, Does: does, Presses: presses}
}

// Segment is one control on the path to the keyboard: the keys the bar names for it; the presses it takes without
// naming them; whether it takes what is typed, every press of a character; whether it takes every press, as an open
// menu does; and the presses it lets through to its ancestors all the same, as a name typed lets = through to the menu.
type Segment struct {
	Bindings []Binding
	Takes    []string
	Types    bool
	Owns     bool
	Passes   []string
}

// With gives the segment with the bindings after its own.
func (s Segment) With(b ...Binding) Segment {
	s.Bindings = append(slices.Clone(s.Bindings), b...)
	return s
}

// Path is the controls from the screen, first, to the one with the keyboard, last.
type Path []Segment

// Shown gives each segment's bindings the bar draws, segment by segment in the path's order: those none of whose
// presses a control deeper on the path takes.
func (p Path) Shown() [][]Binding {
	shown := make([][]Binding, len(p))
	var deeper []func(string) bool
	for i := len(p) - 1; i >= 0; i-- {
		for _, b := range p[i].Bindings {
			if !slices.ContainsFunc(b.Presses, func(press string) bool {
				return slices.ContainsFunc(deeper, func(takes func(string) bool) bool { return takes(press) })
			}) {
				shown[i] = append(shown[i], b)
			}
		}
		deeper = append(deeper, p[i].takes)
	}
	return shown
}

// takes says whether the control takes a press, so its ancestors never see it.
func (s Segment) takes(press string) bool {
	switch {
	case slices.Contains(s.Takes, press) || slices.ContainsFunc(s.Bindings, func(b Binding) bool {
		return slices.Contains(b.Presses, press)
	}):
		return true
	case slices.Contains(s.Passes, press):
		return false
	}
	return s.Owns || s.Types && typing(press)
}

// typing says whether a press types a character: a single one, or the space.
func typing(press string) bool { return press == "space" || utf8.RuneCountInString(press) == 1 }

// The keys bar's separators: a space between the keys of one control, which the keys' color sets apart, and a rule
// between one control and the next. A dialog gives its keys as one line, a dot between them. Keys that do the same
// thing are drawn as one, a dim middle dot between them, never / or another key the bar names.
const (
	gap  = " "
	dot  = " · "
	rule = " │ "
	cut  = "…"
	or   = "·"
)

// Line draws the keys bar for the path on a screen so wide: a cell of gutter, then each segment the bar draws, the one
// the keyboard is in last, its keys brighter than the rest. Too narrow, it keeps the screen's own segment, the first,
// and the last, and drops the segments between them from the second on, … and a rule in their place, so the screen's
// keys stay wherever the keyboard is; what is still too wide is clipped with …, the last segment's end first. Only a
// screen too narrow for the screen's segment alone drops it, as it drops the others, from the first on.
func (p Path) Line(width int) string {
	if width < 2 {
		return ""
	}
	var segments [][]Binding
	shown := p.Shown()
	for _, s := range shown {
		if len(s) > 0 {
			segments = append(segments, merged(s))
		}
	}
	// the screen's segment stays, while it is drawn at all and another follows it
	screen := len(shown) > 0 && len(shown[0]) > 0 && len(segments) > 1
	room := width - 1
	// kept gives the segments drawn, the screen's or none and then those from the from'th on, each part drawn by draw,
	// with … in place of those dropped
	kept := func(keep bool, from int, draw func(i int) string, cutter string) []string {
		var parts []string
		if keep {
			parts = append(parts, draw(0))
		}
		if keep && from > 1 || !keep && from > 0 {
			parts = append(parts, cutter)
		}
		for i := from; i < len(segments); i++ {
			parts = append(parts, draw(i))
		}
		return parts
	}
	plain := func(keep bool, from int) string {
		return strings.Join(kept(keep, from, func(i int) string { return plainSegment(segments[i]) }, cut), rule)
	}
	keep, from := screen, 0
	if keep {
		from = 1
		for from < len(segments)-1 && ansi.StringWidth(plain(true, from)) > room {
			from++
		}
	}
	if keep && ansi.StringWidth(plainSegment(segments[0])+rule+cut+rule) > room {
		keep, from = false, 0
	}
	if !keep {
		for from < len(segments)-1 && ansi.StringWidth(plain(false, from)) > room {
			from++
		}
	}
	parts := kept(keep, from, func(i int) string { return drawSegment(segments[i], i == len(segments)-1) },
		style.Off(cut))
	line := strings.Join(parts, style.BarRule(rule))
	return " " + ansi.Truncate(line, room, cut)
}

// LineWith draws the keys bar as Line does, with what stands at its right end, as each forge's rate limit, against the
// screen's right edge and a cell short of it: the keys keep the bar's room first, so the right end is drawn only where
// it fits after them, two cells apart.
func (p Path) LineWith(width int, right string) string {
	line := p.Line(width)
	used, needs := ansi.StringWidth(line), ansi.StringWidth(right)
	if right == "" || used+2+needs > width-1 {
		return line
	}
	return line + strings.Repeat(" ", width-1-used-needs) + right
}

// merged gives a segment's bindings with each run of keys that do the same thing drawn as one, the keys joined by a
// middle dot, no space either side, as esc and enter both leave the search: esc·⏎ back. A key a deeper control takes
// is gone from the run already.
func merged(s []Binding) []Binding {
	var out []Binding
	for _, b := range s {
		if k := len(out) - 1; k >= 0 && b.Does != "" && b.Key != "" && out[k].Does == b.Does && out[k].Key != "" {
			if len(out[k].keys) == 0 {
				out[k].keys = []string{out[k].Key}
			}
			out[k].keys = append(out[k].keys, b.Key)
			out[k].Key += or + b.Key
			out[k].Presses = append(slices.Clone(out[k].Presses), b.Presses...)
			continue
		}
		out = append(out, b)
	}
	return out
}

// plainSegment gives a segment's keys as the bar draws them, without their styles, to measure.
func plainSegment(s []Binding) string {
	var keys []string
	for _, b := range s {
		keys = append(keys, strings.TrimSpace(b.Key+" "+b.Does))
	}
	return strings.Join(keys, gap)
}

// drawSegment draws a segment's keys, each key bold and what it does after it; here says the keyboard is in its
// control, whose keys are drawn brighter.
func drawSegment(s []Binding, here bool) string {
	var keys []string
	for _, b := range s {
		switch {
		case b.Key == "":
			keys = append(keys, style.BarDoes(b.Does, here))
		case b.Does == "":
			keys = append(keys, drawKeys(b, here))
		default:
			keys = append(keys, drawKeys(b, here)+" "+style.BarDoes(b.Does, here))
		}
	}
	return strings.Join(keys, gap)
}

// drawKeys draws a binding's key, bold, or the keys merged into it, each bold, the middle dot between them dim, so the
// keys stand apart from what joins them.
func drawKeys(b Binding, here bool) string {
	if len(b.keys) == 0 {
		return style.BarKey(b.Key, here)
	}
	var keys []string
	for _, k := range b.keys {
		keys = append(keys, style.BarKey(k, here))
	}
	return strings.Join(keys, style.Off(or))
}

// Digits are the presses that select a tab by its number: 1 to 9, and 0 for the tenth.
var Digits = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"}

// MenuKeys is the segment of an open menu or dropdown, which takes every press: esc closes it, the arrows move, and
// enter chooses.
func MenuKeys() Segment {
	return Segment{Owns: true, Bindings: []Binding{
		Bind("esc", "close"), Bind("↑↓", "move", "up", "down"), Bind("⏎", "choose", "enter"),
	}}
}

// ScreenKeys gives the segment of a screen, the first on every path: = drops the app's menu down and / gives the search
// the keyboard, then the screen's own keys.
func ScreenKeys(own ...Binding) Segment {
	return Segment{Bindings: append([]Binding{Bind("=", "menu"), Bind("/", "search")}, own...)}
}

// SearchKeys gives the segment of the search while it has the keyboard: it types, and ctrl+r recalls a search when
// there are some to recall. What leaves it is the screen's to say, as where the keyboard goes then is.
func SearchKeys(recalls bool) Segment {
	s := Segment{Types: true, Takes: []string{"backspace"}, Bindings: []Binding{Bind("", "type to search")}}
	if recalls {
		s.Bindings = append(s.Bindings, Bind("⌃R", "recent", "ctrl+r"))
	}
	return s
}

// DialogKeys gives the segment of a dialog, which takes every press, from its keys as its bottom edge names them: each
// a key, a space, and what it does, a dot between one and the next.
func DialogKeys(keys string) Segment {
	s := Segment{Owns: true}
	for _, k := range strings.Split(keys, dot) {
		key, does, _ := strings.Cut(k, " ")
		s.Bindings = append(s.Bindings, Bind(key, does))
	}
	return s
}

// String gives the keys bar as plain text, as wide as it takes, its gutter and styles apart.
func (p Path) String() string {
	return strings.TrimPrefix(ansi.Strip(p.Line(math.MaxInt32)), " ")
}
