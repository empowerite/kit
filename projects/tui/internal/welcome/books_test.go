// The title's - is the workspace menu: it names the workspace kit says, drops down what to switch to on a plain
// click, and a pick asks kittui to switch.
package welcome

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// inW is the repository github.com/empowerite/kit, a member of the group w.
var inW = kit.Workspace{Kind: "repository", Name: "github.com/empowerite/kit", Member: "github.com/empowerite/kit",
	Checkout: "/code/kit", Groups: []kit.WorkspaceGroup{{Name: "w", Members: 2}}}

// click gives a left click at x, y.
func click(x, y int) tea.MouseClickMsg {
	return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft}
}

func TestTheTitleIsTheBooksControl(t *testing.T) {
	m, _ := loaded(t)
	if title := strings.Split(view(m), "\n")[0]; !strings.Contains(title, "kit welcome · - empowerite/kit") {
		t.Errorf("before kit says, the title names origin's owner/repo: %q", title)
	}
	w := kit.Workspace{Kind: "group", Name: "w", Members: []kit.WorkspaceMember{{Repository: "github.com/o/a"},
		{Repository: "github.com/o/b"}}}
	m = drive(m, shell.WorkspaceMsg{Workspace: w})
	if title := strings.Split(view(m), "\n")[0]; !strings.Contains(title, "kit welcome · - w 2") {
		t.Errorf("a group: %q", title)
	}
	if m.Books().Workspace.Name != "w" {
		t.Errorf("the control: %+v", m.Books())
	}
}

// - drops the workspace menu down, as a click on it does, and - again closes it.
func TestMinusDropsTheWorkspaceMenuDown(t *testing.T) {
	m, _ := loaded(t, shell.WorkspaceMsg{Workspace: inW})
	minus := tea.KeyPressMsg{Code: '-', Text: shell.Workspaces}
	if m = drive(m, minus); !m.books.Open {
		t.Fatalf("- did not drop the workspace menu down:\n%s", view(m))
	}
	if m = drive(m, minus); m.books.Open {
		t.Errorf("- again left it open")
	}
}

func TestAClickOnTheBooksDropsItDownAndAPickAsksKittui(t *testing.T) {
	m, _ := loaded(t, shell.WorkspaceMsg{Workspace: inW})
	m = drive(m, click(m.booksAt(), shell.Top))
	if !m.books.Open || keysAt(m) != (shell.Path{shell.MenuKeys()}).String() {
		t.Fatalf("open %v, keys %q", m.books.Open, keysAt(m))
	}
	if v := view(m); !strings.Contains(v, "w 2") || !strings.Contains(v, "Browse…") {
		t.Errorf("the dropdown is not drawn:\n%s", v)
	}
	next, cmd := m.Update(click(m.booksAt()+2, shell.Top+1))
	if next.(Model).books.Open || cmd == nil || cmd() != (shell.PickMsg{Workspace: "w"}) {
		t.Errorf("a click on w: open %v", next.(Model).books.Open)
	}
	// a key on the open dropdown is its own: enter picks the line under the keyboard
	next, cmd = m.Update(enter)
	if cmd == nil || cmd() != (shell.PickMsg{Workspace: "w"}) {
		t.Errorf("enter on w: %v", next.(Model).books.Open)
	}
}

func TestEscClosesTheBooks(t *testing.T) {
	m, _ := loaded(t, shell.WorkspaceMsg{Workspace: inW})
	m = drive(m, click(m.booksAt(), shell.Top), esc)
	if m.books.Open || m.inPane {
		t.Errorf("esc: open %v, in the pane %v", m.books.Open, m.inPane)
	}
	// a click off the dropdown closes it and does nothing else
	m = drive(m, click(m.booksAt(), shell.Top), click(m.booksAt()+60, 20))
	if m.books.Open {
		t.Error("a click off it leaves it open")
	}
}

func TestOpenNamesTheWorkspaceKitSaid(t *testing.T) {
	m := Open(testkit.Runner(t, "welcome"), inW)
	if m.Books().Workspace.Name != inW.Name || m.workspace.Name != inW.Name {
		t.Errorf("opened on: %+v", m.Books())
	}
}
