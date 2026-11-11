// The menu bar, the first line of every screen, drawn alike on each: the kit menu = and the screen's name; a
// separator and the workspace menu; a separator and what the view stands on, a maximized tab's menu or a verb's form;
// a separator and the crumb, the way back to where a link led from, as crumb.go says; space; and the search field at
// the right end.
package shell

import (
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// MenuBar is what a screen's menu bar shows: the screen's name, after the kit menu's =, and whether that menu is open;
// the workspace menu, as BooksMenu.Label draws it; what the view stands on, a maximized tab's menu or a verb's name;
// the crumb, by the name of the place it goes back to, as CrumbLabel draws it; and the search field, as RecallLine
// draws it. A part left empty is not drawn, nor the separator before it.
type MenuBar struct {
	Screen    string
	Open      bool
	Workspace string
	On        string
	Crumb     string
	Search    string
}

// title gives the menu bar's left part: the kit menu and the screen's name, then each part given after a separator.
func (b MenuBar) title() string {
	title := b.before()
	if b.Crumb != "" {
		title += Between + CrumbLabel(b.Crumb)
	}
	return title
}

// before gives the left part up to the crumb, without it.
func (b MenuBar) before() string {
	title := Title(style.Label(b.Screen), b.Open)
	for _, part := range []string{b.Workspace, b.On} {
		if part != "" {
			title += Between + part
		}
	}
	return title
}

// SearchAt gives the column the search field starts at on a screen so wide, as SearchAt puts every screen's; the
// screen's width where there is no search field.
func (b MenuBar) SearchAt(width int) int {
	if b.Search == "" {
		return width
	}
	return SearchAt(width, b.Search)
}

// TitleWidth gives the cells the menu bar's left part has on a screen so wide: all of it up to two cells before the
// search field.
func (b MenuBar) TitleWidth(width int) int { return max(0, b.SearchAt(width)-2) }

// WorkspaceAt gives the column the workspace menu's glyph starts at.
func (b MenuBar) WorkspaceAt() int { return BooksAt(Title(style.Label(b.Screen), false)) }

// OnSpan gives the columns what the view stands on is drawn across on a screen so wide, from and up to, as much of it
// as the left part shows, and whether it shows whole; -1 and -1 where none of it shows, or there is none.
func (b MenuBar) OnSpan(width int) (from, until int, whole bool) {
	return b.span(b.On, ansi.StringWidth(b.before()), width)
}

// CrumbSpan gives the columns the crumb is drawn across on a screen so wide, as OnSpan gives what the view stands on.
func (b MenuBar) CrumbSpan(width int) (from, until int, whole bool) {
	if b.Crumb == "" {
		return -1, -1, false
	}
	return b.span(CrumbLabel(b.Crumb), ansi.StringWidth(b.title()), width)
}

// span gives the columns a part drawn last up to end is drawn across, as OnSpan says.
func (b MenuBar) span(part string, end, width int) (from, until int, whole bool) {
	from = end - ansi.StringWidth(part)
	if part == "" || from >= b.TitleWidth(width) {
		return -1, -1, false
	}
	return from, min(end, b.TitleWidth(width)), end <= b.TitleWidth(width)
}

// OnCrumb says whether a click at x, y is on the crumb, as much of it as the title shows, on a screen so wide.
func (b MenuBar) OnCrumb(x, y, width int) bool {
	from, until, _ := b.CrumbSpan(width)
	return y == Top && from >= 0 && x >= from && x < until
}

// Line draws the menu bar on a screen so wide: the left part, clipped two cells short of the search field, and the
// search field from SearchAt, clipped a cell short of the edge.
func (b MenuBar) Line(width int) string {
	title := pane.Clip(b.title(), b.TitleWidth(width))
	if b.Search == "" {
		return title
	}
	return pane.Pad(title, b.TitleWidth(width), false) + "  " + pane.Clip(b.Search, width-1-b.SearchAt(width))
}
