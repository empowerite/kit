// The dialog an action opens before kit runs: a line to type, as a group's new name or the members to add, or what the
// action will do, to confirm; and the line that does it, which enter does too. Kit's line, when it refuses, stands in
// the dialog, which stays open; esc, or a click off the box, closes it.
package workspaces

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Do is the glyph of the dialog's line that does what it asks.
const Do = "✅"

// prompt is an action's dialog: whether it is open, which then has the keyboard; its title; whether a line is typed,
// and what is typed so far; what to type, or what the action will do, dim; the name of its line that does it; whether
// kit runs it now; kit's line, when it refused; and the kit command what is typed makes, ok false while it makes none.
type prompt struct {
	open    bool
	title   string
	typing  bool
	typed   string
	says    string
	does    string
	busy    bool
	refusal string
	words   func(typed string) (words []string, ok bool)
}

// makes gives the kit command what is typed makes, and whether it makes one.
func (p prompt) makes() ([]string, bool) {
	return p.words(strings.TrimSpace(p.typed))
}

// ask gives the prompt asking kit to run what is typed, and the words; none while kit runs, or while what is typed
// makes no command.
func (p prompt) ask() (prompt, []string) {
	if p.busy {
		return p, nil
	}
	words, ok := p.makes()
	if !ok {
		return p, nil
	}
	p.busy, p.refusal = true, ""
	return p, words
}

// refused gives the prompt told kit's line: open still, the line under what is typed.
func (p prompt) refused(line string) prompt {
	p.busy, p.refusal = false, line
	return p
}

// key takes a key while the prompt is open: while a line is typed, a printable key types, backspace takes one back,
// and ⌃U clears; enter asks kit, and esc closes the prompt. It gives the prompt and the words kit runs, none for none.
func (p prompt) key(k tea.KeyPressMsg) (prompt, []string) {
	switch k.String() {
	case "esc":
		p.open = false
		return p, nil
	case "enter":
		return p.ask()
	case "ctrl+u":
		if p.typing {
			p.typed, p.refusal = "", ""
		}
		return p, nil
	}
	if typed, ok := shell.Type(p.typed, k); ok && p.typing {
		p.typed, p.refusal = typed, ""
	}
	return p, nil
}

// keys gives the prompt's segment of the keys bar: it takes every press.
func (p prompt) keys() shell.Segment {
	s := shell.DialogKeys("esc close · ⏎ " + strings.ToLower(p.does))
	s.Bindings[1].Presses = []string{"enter"}
	return s
}

// body gives the prompt's lines under its top edge: what is typed, a cursor after it, while a line is typed; kit's line,
// or what to type; and the line that does it, dim while kit runs or what is typed makes no command.
func (p prompt) body() []string {
	var lines []string
	if p.typing {
		lines = append(lines, "› "+p.typed+"▏")
	}
	if p.refusal != "" {
		lines = append(lines, style.Refusal(Failed+" "+p.refusal))
	} else {
		lines = append(lines, style.Off(p.says))
	}
	does := Do + " " + p.does
	if _, ok := p.makes(); p.busy || !ok {
		does = style.Off(does)
	}
	return append(lines, "", does)
}

// placed gives the prompt's box where it stands over a screen of so many lines, so wide: near its top, centered.
func (p prompt) placed(height, width int) shell.Placed {
	// a line of spaces, taken out again, keeps the box wide enough for what is typed
	box := shell.Dialog(p.title, append(p.body(), strings.Repeat(" ", min(64, max(0, width-6)))), width-2)
	box = append(box[:len(box)-2], box[len(box)-1])
	return shell.Placed{Lines: box, Top: min(2, max(0, height-len(box))),
		Left: max(0, (width-ansi.StringWidth(box[0]))/2)}
}

// overlay draws the open prompt over the screen's lines, so wide.
func (p prompt) overlay(lines []string, width int) []string {
	if !p.open {
		return lines
	}
	return p.placed(len(lines), width).Draw(lines)
}

// click takes a left click while the prompt is open, over a screen of so many lines, so wide: one on the line that does
// it asks kit, as enter does, and one off the box closes the prompt. It gives the prompt and the words kit runs.
func (p prompt) click(c tea.MouseClickMsg, height, width int) (prompt, []string) {
	k, on := p.placed(height, width).Hit(c.X, c.Y)
	switch {
	case !on:
		p.open = false
	case k == len(p.body())-1:
		return p.ask()
	}
	return p, nil
}
