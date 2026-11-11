// A crumb is the way back from where a link led: a link in one app's table to a row of another app, or the workspace
// menu's Manage…, leaves the place it was followed from as the crumb of the place it leads to. The title line draws the
// crumb last, after the workspace menu and a maximized tab, a dot between, as ↩ and the place's name; a plain click on
// it, or ⌫, which the keys bar names while it shows, goes back there, the view as it was left, with that place's own
// crumb, so crumbs chain, each one deep. Only the app's menu, =, shows a screen with no crumb. kittui's root keeps the
// chain, for the session; each screen draws the crumb it is told, and says when a click or a key is the way back.
package shell

import "github.com/empowerite/kit/projects/tui/internal/style"

// Back is the crumb's glyph, before the name of the place it goes back to; one cell, as the title's = and - are.
const Back = "↩"

// BackPress is the press that goes back to the crumb, ⌫, which no screen takes while its rows or its bar have the
// keyboard: a search, a name being typed, or an open layer takes it first.
const BackPress = "backspace"

// BackKeys gives the keys bar's binding of the press that goes back, named while a crumb shows.
func BackKeys() Binding { return Bind("⌫", "back", BackPress) }

// CrumbLabel gives the crumb as the title line draws it: Back, tinted, as what is clicked, and the place's name.
func CrumbLabel(name string) string { return style.Tint(Back) + " " + name }

// CrumbOf gives the screen segment's bindings with the press that goes back after them, while a crumb is named.
func CrumbOf(crumb string, own ...Binding) []Binding {
	if crumb == "" {
		return own
	}
	return append(own, BackKeys())
}
