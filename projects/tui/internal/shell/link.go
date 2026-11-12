package shell

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// IsWeb says whether a link's target is a web address.
func IsWeb(to string) bool {
	return strings.HasPrefix(to, "http://") || strings.HasPrefix(to, "https://")
}

// Link gives the command that asks for where a link on a page leads, its target as internal/markdown gives it after
// kittui:, with the page as its crumb: a screen, by its name, or the screen on one of its tabs, by the screen's name, a
// slash, and the tab's stage, as the app names it, and then a slash and what the tab keeps in view, a row or a term.
// A tutorial is tutorials/ and its id, and a term welcome/terms/ and the term. No target gives none.
func Link(to string) tea.Cmd {
	screen, rest, _ := strings.Cut(to, "/")
	switch {
	case to == "":
		return nil
	case rest == "":
		return func() tea.Msg { return SwitchMsg{Screen: Screen(screen), Crumb: true} }
	case !strings.Contains(rest, "/"):
		rest += "/" // the tab alone, as an app names where it lands: no row
	}
	return func() tea.Msg { return JumpMsg{Screen: Screen(screen), To: rest} }
}
