// Package style is kit tui's look: one accent color, the chosen and dim styles, the glyphs the spec fixes, and the keys
// bar every screen ends with. NO_COLOR
// strips the colors where the program writes, and bold and underline stay, so a chosen name still stands out.
package style

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// The glyphs, as the spec draws them.
const (
	Answered = "✓"       // a field answered, in the transcript
	Focus    = "▸"       // the field that has the keyboard
	Opened   = "▾"       // the row that has the keyboard, open to what it holds
	Picked   = "◼"       // a value picked in a list of many
	Unpicked = "◻"       // a value not picked
	Filter   = "›"       // the filter line of a list
	Active   = "✅"       // a choice active now, and staying
	Excluded = "❌"       // a choice excluded now, and a refusal
	Add      = "➕"       // a choice to be added
	Remove   = "➖"       // a choice to be removed
	Back     = "‹ verbs" // the way back to the palette
)

var (
	accent  = lipgloss.Color("12")
	red     = lipgloss.Color("9")
	chosen  = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(accent)
	off     = lipgloss.NewStyle().Faint(true)
	label   = lipgloss.NewStyle().Bold(true)
	cursor  = lipgloss.NewStyle().Reverse(true)
	italic  = lipgloss.NewStyle().Italic(true)
	refusal = lipgloss.NewStyle().Foreground(red)
	mark    = lipgloss.NewStyle().Bold(true).Foreground(accent)
	tint    = lipgloss.NewStyle().Foreground(accent)
	menu    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	warn    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	gain    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	loss    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1"))
	repo    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13"))
	band    = lipgloss.NewStyle().Background(lipgloss.Color("236"))
	// on the menu's own bright cyan, black: 16 is the 256-color cube's black, which no theme remaps as it may 0
	selected = lipgloss.NewStyle().Background(lipgloss.Color("14")).Foreground(lipgloss.Color("16"))
)

// Mark draws what is selected on a line of stages: bold, in the accent color.
func Mark(s string) string { return mark.Render(s) }

// Menu draws an open menu, its box and the glyph it hangs from: bold, in bright cyan, a color nothing else on the screen
// has, so the menu stands out over what it covers.
func Menu(s string) string { return menu.Render(s) }

// In gives Mark and Tint in a color of their own, a 256-color number, for what belongs to one group, as a tab of the
// stages' bar does; no color gives Mark and Tint themselves.
func In(color string) (marked, tinted func(string) string) {
	if color == "" {
		return Mark, Tint
	}
	m, t := mark.Foreground(lipgloss.Color(color)), tint.Foreground(lipgloss.Color(color))
	return func(s string) string { return m.Render(s) }, func(s string) string { return t.Render(s) }
}

// Tint draws what belongs to the selected tab but is not chosen: in the accent color, not bold.
func Tint(s string) string { return tint.Render(s) }

// Gain draws a count that adds: commits ahead, lines added. Bold green, as kit flow --text draws it.
func Gain(s string) string { return gain.Render(s) }

// Loss draws a count that takes away: commits behind, lines removed. Bold red, as kit flow --text draws it.
func Loss(s string) string { return loss.Render(s) }

// Warn draws what wants attention but is not wrong, such as uncommitted paths: yellow, as git status draws them.
func Warn(s string) string { return warn.Render(s) }

// Repo draws a repository's owner/name: bold, in bright magenta, a color of its own.
func Repo(s string) string { return repo.Render(s) }

// Chosen draws a chosen name: bold, underlined, and in the accent color, so a terminal without underline still
// shows it.
func Chosen(s string) string { return chosen.Render(s) }

// Off draws a name not chosen, and any text that steps back: dim.
func Off(s string) string { return off.Render(s) }

// Label draws a field's name.
func Label(s string) string { return label.Render(s) }

// Selected draws the line under the keyboard in a menu or a dialog: on the color of the box and its title, Menu's, in
// black, which reads on it, the line's own styles dropped, so none of them breaks the bar.
func Selected(s string) string { return selected.Render(ansi.Strip(s)) }

// Cursor draws the name under the cursor in a row of toggles: inverse.
func Cursor(s string) string { return cursor.Render(s) }

// Italic draws a name that no longer quite holds, as the preset a table has changed from: italic, and inverse too
// while it is under the cursor.
func Italic(s string, under bool) string {
	if under {
		return italic.Reverse(true).Render(s)
	}
	return italic.Render(s)
}

// Refusal draws the line a verb would print about an answer.
func Refusal(s string) string { return refusal.Render(s) }

// Band draws a line on a faint background, as the row under the cursor is drawn, to the width: the line's own styles
// stay, and the band is opened again after each of them ends, so it runs the whole line.
func Band(line string, width int) string {
	open, end, _ := strings.Cut(band.Render("x"), "x")
	if open == "" {
		return line
	}
	line += strings.Repeat(" ", max(0, width-ansi.StringWidth(line)))
	return open + strings.ReplaceAll(line, "\x1b[m", "\x1b[m"+open) + end
}

// BarKey draws a key on the keys bar: bold, and in the accent color too in the control the keyboard is in, so the
// keys that act where it is are the first the eye finds.
func BarKey(s string, here bool) string {
	if here {
		return mark.Render(s)
	}
	return label.Render(s)
}

// BarDoes draws what a key on the keys bar does: plain in the control the keyboard is in, dim in the controls around
// it.
func BarDoes(s string, here bool) string {
	if here {
		return s
	}
	return off.Render(s)
}

// BarRule draws the rule on the keys bar between the keys of one control and the next: in the accent color, so the
// controls stand apart where the dots between their keys, dim, do not.
func BarRule(s string) string { return tint.Render(s) }

// Bottom puts the bar on the screen's last line: the lines above it cut or padded to the height less the bar. A
// height of zero, not yet known, draws the lines, a gap, and the bar.
func Bottom(lines []string, height int, bar string) []string {
	if height <= 0 {
		return append(lines, "", bar)
	}
	lines = lines[:min(len(lines), height-1)]
	for len(lines) < height-1 {
		lines = append(lines, "")
	}
	return append(lines, bar)
}
