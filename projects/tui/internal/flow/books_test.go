// The picture's title carries the workspace menu, as every screen's does: it names the workspace kit says, drops down
// what to switch to on a plain click, and a pick asks kittui to switch.
package flow

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// inW is the repository github.com/empowerite/kit, a member of the group w.
var inW = kit.Workspace{Kind: "repository", Name: "github.com/empowerite/kit", Member: "github.com/empowerite/kit",
	Checkout: "/code/kit", Groups: []kit.WorkspaceGroup{{Name: "w", Members: 2}}}

// leftAt gives a left click at x, y.
func leftAt(x, y int) tea.MouseClickMsg { return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft} }

func TestThePicturesTitleIsTheBooksControl(t *testing.T) {
	m := loaded(t, options())
	if title := lines(m)[0]; !strings.Contains(title, "kit flow · - empowerite/kit") {
		t.Errorf("before kit says, origin's owner/repo: %q", title)
	}
	m = drive(m, shell.WorkspaceMsg{Workspace: kit.Workspace{Kind: "group", Name: "w",
		Members: []kit.WorkspaceMember{{Repository: "github.com/o/a"}, {Repository: "github.com/o/b"}}}})
	if title := lines(m)[0]; !strings.Contains(title, "kit flow · - w 2") {
		t.Errorf("a group: %q", title)
	}
}

// - drops the workspace menu down, as a click on it does, and - again closes it.
func TestMinusDropsTheWorkspaceMenuDown(t *testing.T) {
	m := loaded(t, options(), shell.WorkspaceMsg{Workspace: inW})
	if m = drive(m, typed(shell.Workspaces)...); !m.books.Open {
		t.Fatalf("- did not drop the workspace menu down:\n%s", view(m))
	}
	if m = drive(m, typed(shell.Workspaces)...); m.books.Open {
		t.Errorf("- again left it open")
	}
}

func TestAClickOnThePicturesBooksDropsItDownAndAPickAsksKittui(t *testing.T) {
	m := loaded(t, options(), shell.WorkspaceMsg{Workspace: inW})
	m = drive(m, leftAt(m.booksAt(), 0))
	if !m.books.Open || m.keys().String() != (shell.Path{shell.MenuKeys()}).String() {
		t.Fatalf("open %v, keys %q", m.books.Open, m.keys().String())
	}
	if v := view(m); !strings.Contains(v, "w 2") || !strings.Contains(v, "Manage workspaces…") {
		t.Errorf("the dropdown is not drawn:\n%s", v)
	}
	next, cmd := m.Update(leftAt(m.booksAt()+2, 1))
	if next.(Model).books.Open || cmd == nil || cmd() != (shell.PickMsg{Workspace: "w"}) {
		t.Errorf("a click on w: open %v", next.(Model).books.Open)
	}
	next, cmd = m.Update(press(tea.KeyEnter))
	if cmd == nil || cmd() != (shell.PickMsg{Workspace: "w"}) {
		t.Errorf("enter on w: open %v", next.(Model).books.Open)
	}
	// esc closes it; a click that is not a left one leaves it to the screen
	if m2 := drive(m, press(tea.KeyEscape)); m2.books.Open {
		t.Error("esc leaves it open")
	}
	if n, _ := m.Update(tea.MouseClickMsg{X: m.booksAt() + 2, Y: 1, Button: tea.MouseRight}); !n.(Model).books.Open {
		t.Error("a right click closed it")
	}
	// a key or a click that picks nothing asks nothing
	if _, cmd := m.Update(press(tea.KeyDown)); cmd != nil {
		t.Error("down asked kittui")
	}
}
