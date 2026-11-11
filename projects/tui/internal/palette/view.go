// The palette's view: the menu bar, kit's line under it when a verb did not open, the table's pane, its layers and the
// menus over it, and the keys bar last, or what the table did, or why not, in its place.
package palette

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// menuBar gives the menu bar, as internal/shell draws every screen's: the kit menu's = before kit help; the workspace
// menu, which names the workspace; and the search field.
func (m Model) menuBar() shell.MenuBar {
	return shell.MenuBar{Screen: appLabel, Open: m.appMenu.Open, Workspace: m.books.Label(), Search: m.searchLine()}
}

// searchAt gives the column the search starts at on the menu bar, as internal/shell puts every screen's.
func (m Model) searchAt() int { return m.menuBar().SearchAt(m.width) }

// booksAt gives the column the menu bar's workspace menu starts at.
func (m Model) booksAt() int { return m.menuBar().WorkspaceAt() }

// recallAt gives the screen column of the search's Recall glyph.
func (m Model) recallAt() int { return shell.RecallAt(m.searchAt()) }

// searchLine gives the search as the title line draws it, its cursor showing except while the app's menu has the
// keyboard.
func (m Model) searchLine() string {
	return shell.RecallLine(m.search.Query, m.search.Focused && !m.appMenu.Open, len(m.recent.Items) > 0)
}

// top gives the lines over the table: the menu bar; kit's line under it, in red, when a verb did not open; and a gap.
func (m Model) top() []string {
	top := []string{m.menuBar().Line(m.width)}
	if m.refusal != "" {
		for _, line := range page(style.Excluded+" "+m.refusal, m.width-2) {
			top = append(top, " "+style.Refusal(line))
		}
	}
	return append(top, "")
}

// View draws the lines over the table, then the table's pane, each line clipped at the screen's edge; the table's
// layers, the searches remembered, and the menus over them; and the keys bar last, one line clipped to the width, so
// neither searching nor a refusal moves it, or what the table did, or why not, in its place while it says so.
func (m Model) View() tea.View {
	var drawn []string
	for _, l := range append(m.top(), m.sheet.View()...) {
		drawn = append(drawn, pane.Clip(l, m.width))
	}
	drawn = m.sheet.Over(append(drawn, ""))
	drawn = m.recent.Overlay(drawn, m.recallAt(), m.width)
	drawn = m.appMenu.Overlay(drawn, m.width)
	drawn = m.books.Overlay(drawn, m.booksAt(), m.width)
	bottom := m.keys().Line(m.width)
	switch n := m.sheet.Notice(); {
	case n.Text == "":
	case n.Done:
		bottom = " " + pane.Clip(n.Text, m.width-1)
	default:
		bottom = " " + style.Refusal(pane.Clip(n.Text, m.width-1))
	}
	return shell.View(drawn, m.height, bottom)
}

// keys gives the path to the keyboard, which the keys bar is drawn from: the screen; then the search, as every
// screen's is, whose ⏎ gives the rows the keyboard, while it has focus; or the part of the table that has the
// keyboard, its esc giving the search the keyboard, and on a verb, ⏎ opening its form. An open menu, or a layer of the
// table, takes every press, and is the path alone.
func (m Model) keys() shell.Path {
	if m.appMenu.Open || m.recent.Open || m.books.Open {
		return shell.Path{shell.MenuKeys()}
	}
	if seg, layer := m.sheet.Keys(); layer {
		return shell.Path{seg}
	}
	if m.search.Focused {
		search := shell.SearchKeys("verbs", m.search.Query, len(m.recent.Items) > 0)
		search.Passes = append(search.Passes, "=")
		return shell.Path{shell.WorkspaceScreenKeys(), search}
	}
	if !m.sheet.Focused() {
		return shell.Path{shell.WorkspaceScreenKeys()}
	}
	seg, _ := m.sheet.Keys()
	seg.Bindings = append([]shell.Binding(nil), seg.Bindings...)
	for k, b := range seg.Bindings {
		if b.Does == table.TabsHint {
			seg.Bindings[k].Does = "search"
		}
	}
	if _, ok := m.highlighted(); ok && m.sheet.InRows() {
		seg = seg.With(shell.Bind("⏎", "open", "enter"))
	}
	return shell.Path{shell.WorkspaceScreenKeys(), seg}
}

// page wraps a text to the width, keeping each line's indent, and gives its lines. A width counts what shows, so the
// escapes that make a heading bold take none.
func page(text string, width int) []string {
	var out []string
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
		words := strings.Fields(line)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		current := indent + words[0]
		for _, word := range words[1:] {
			if ansi.StringWidth(current)+1+ansi.StringWidth(word) > width {
				out = append(out, current)
				current = indent + word
				continue
			}
			current += " " + word
		}
		out = append(out, current)
	}
	return out
}
