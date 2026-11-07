// A dropdown is a menu that drops down from what it hangs from, as each of kittui's menus does. It holds whether it is
// open and the line under the keyboard; its lines, and which of them act, are its owner's, given to each call, since
// they follow what the screen shows. While it is open it has the keyboard: up and down move to the next line that acts,
// a letter to the next that starts with it, enter and space choose the line under the keyboard, and esc, or a key its
// owner names, closes it. A click on a line that acts chooses it, one on a line that does not does nothing, and one off
// the box closes it. A line that cannot act stays, dim, and the keyboard passes it by.
package shell

import (
	"slices"

	tea "charm.land/bubbletea/v2"
)

// Dropdown is a dropdown's own state: whether it is open, and the line under the keyboard, -1 for none.
type Dropdown struct {
	Open bool
	At   int
}

// Opened gives the dropdown open, its first line that acts under the keyboard.
func (d Dropdown) Opened(items []Item, enabled func(int) bool) Dropdown {
	d.Open, d.At = true, Move(items, enabled, -1, 1)
	return d
}

// Key takes a key while the dropdown is open, as the header says, and gives the dropdown and the line chosen, -1 for
// none. Choosing a line closes it, as esc and each of closes do.
func (d Dropdown) Key(k tea.KeyPressMsg, items []Item, enabled func(int) bool, closes ...string) (Dropdown, int) {
	switch key := k.String(); {
	case key == "up":
		d.At = Move(items, enabled, d.At, -1)
	case key == "down":
		d.At = Move(items, enabled, d.At, 1)
	case key == "enter" || key == "space":
		return d.choose(d.At, items, enabled)
	case key == "esc" || slices.Contains(closes, key):
		d.Open = false
	default:
		d.At = NextItem(items, enabled, d.At, k.Text)
	}
	return d, -1
}

// Click takes a left click while the dropdown is open, drawn as a box so many cells wide, its top edge on the screen
// line top and its left edge in the column left, each line under the edge: on a line that acts, it chooses it; on
// another line of the box, it does nothing; off the box, it closes the dropdown. It gives the dropdown and the line
// chosen, -1 for none.
func (d Dropdown) Click(x, y, top, left, width int, items []Item, enabled func(int) bool) (Dropdown, int) {
	k := y - top - 1
	if x > left && x < left+width-1 && k >= 0 && k < len(items) {
		return d.choose(k, items, enabled)
	}
	d.Open = false
	return d, -1
}

// choose gives the dropdown closed and the line k, when it acts; else the dropdown as it was, and -1.
func (d Dropdown) choose(k int, items []Item, enabled func(int) bool) (Dropdown, int) {
	if k < 0 || k >= len(items) || !enabled(k) {
		return d, -1
	}
	d.Open = false
	return d, k
}
