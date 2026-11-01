// Package style is kit tui's look: one accent color, the chosen and dim styles, the glyphs the spec fixes, and the keys
// bar every screen ends with. NO_COLOR
// strips the colors where the program writes, and bold and underline stay, so a chosen name still stands out.
package style

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// The glyphs, as the spec draws them.
const (
	Answered = "✓"       // a field answered, in the transcript
	Focus    = "▸"       // the field that has the keyboard
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
	refusal = lipgloss.NewStyle().Foreground(red)
)

// Chosen draws a chosen name: bold, underlined, and in the accent color, so a terminal without underline still shows it.
func Chosen(s string) string { return chosen.Render(s) }

// Off draws a name not chosen, and any text that steps back: dim.
func Off(s string) string { return off.Render(s) }

// Label draws a field's name.
func Label(s string) string { return label.Render(s) }

// Cursor draws the name under the cursor in a row of toggles: inverse.
func Cursor(s string) string { return cursor.Render(s) }

// Refusal draws the line a verb would print about an answer.
func Refusal(s string) string { return refusal.Render(s) }

// Bar draws the keys bar every screen ends with: one dim line, clipped to the width with …, never wrapped.
func Bar(keys string, width int) string {
	if width < 2 {
		return ""
	}
	return " " + Off(ansi.Truncate(keys, width-1, "…"))
}

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
