// The workspace menu names the workspace on the title line and drops down the groups and the favorites to switch to.
package shell

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// group gives a group workspace of so many members.
func group(name string, n int) kit.Workspace {
	ws := kit.Workspace{Kind: "group", Name: name}
	for range n {
		ws.Members = append(ws.Members, kit.WorkspaceMember{Repository: "github.com/o/x"})
	}
	return ws
}

// inGroups is the repository github.com/o/a, a member of w and v, whose favorites are w and github.com/o/b, which is
// not cloned.
var inGroups = kit.Workspace{Kind: "repository", Name: "github.com/o/a", Member: "github.com/o/a", Checkout: "/code/a",
	Groups: []kit.WorkspaceGroup{{Name: "w", Members: 2}, {Name: "v", Members: 1}},
	Favorites: []kit.WorkspaceFavorite{{Name: "w", Kind: "group", Members: 2},
		{Name: "github.com/o/b", Kind: "repository"}}}

func TestBooksNamesARepositoryAGroupAndNone(t *testing.T) {
	for _, c := range []struct {
		books BooksMenu
		want  string
	}{
		{BooksMenu{Workspace: kit.Workspace{Kind: "repository", Member: "github.com/empowerite/kit"}},
			"- empowerite/kit"},
		{BooksMenu{Workspace: kit.Workspace{Kind: "repository", Name: "/code/scratch", Checkout: "/code/scratch"}},
			"- scratch"},
		{BooksMenu{Workspace: group("empowerite", 5)}, "- empowerite 5"},
		{BooksMenu{Fallback: "empowerite/kit"}, "- empowerite/kit"},
		{BooksMenu{Workspace: kit.Workspace{Kind: "none"}, Fallback: "empowerite/kit"}, "-"},
		{BooksMenu{}, "-"},
	} {
		if got := ansi.Strip(c.books.Label()); got != c.want {
			t.Errorf("%+v: %q, want %q", c.books.Workspace, got, c.want)
		}
	}
	if label := (BooksMenu{Workspace: group("empowerite", 5)}).Label(); !strings.HasSuffix(label, style.Off("5")) ||
		!strings.Contains(label, style.Repo("empowerite")) {
		t.Errorf("a group's name is the repository's color and its count dim: %q", label)
	}
	if label := (BooksMenu{}).Label(); label != style.Off(Workspaces) {
		t.Errorf("no workspace is - alone, dim: %q", label)
	}
}

// names gives the dropdown's lines' names, and which act.
func names(b BooksMenu) ([]string, []int) {
	var out []string
	var acting []int
	for k, item := range b.Items() {
		out = append(out, item.Name)
		if b.enabled(k) {
			acting = append(acting, k)
		}
	}
	return out, acting
}

func TestBooksDrawsNoLineThatOpensNothing(t *testing.T) {
	got, acting := names(BooksMenu{Workspace: inGroups})
	if strings.Join(got, "|") != "w 2|v 1||w 2|o/b · not cloned||Open…|Manage…" {
		t.Errorf("the lines: %q", got)
	}
	if len(acting) != 5 {
		t.Errorf("the lines that act: %v", acting)
	}
	// with nothing to switch to, Open… and Manage… alone, which act
	got, acting = names(BooksMenu{Workspace: kit.Workspace{Kind: "none"}})
	if strings.Join(got, "|") != "Open…|Manage…" || len(acting) != 2 {
		t.Errorf("with no group: %q, acting %v", got, acting)
	}
	if items := (BooksMenu{}).Items(); items[0] != (Item{Glyph: Opening, Name: "Open…"}) {
		t.Errorf("the line with no group: %+v", items[0])
	}
	if got, _ := names(BooksMenu{Refusal: "x"}); strings.Join(got, "|") != "x||Open…|Manage…" {
		t.Errorf("kit's line alone: %q", got)
	}
	if got, _ := names(BooksMenu{Refusal: "x", Workspace: inGroups}); strings.Join(got, "|") !=
		"x||w 2|v 1||w 2|o/b · not cloned||Open…|Manage…" {
		t.Errorf("kit's line, the groups, and the favorites: %q", got)
	}
}

// Open… and Manage… pick no workspace: they ask kittui for the Open… dialog and the workspaces' screen.
func TestBooksOpenAsksForTheOpenDialog(t *testing.T) {
	b := BooksMenu{Workspace: kit.Workspace{Kind: "none"}}.Opened()
	if !b.Open || b.At != 0 {
		t.Fatalf("opened: %+v", b.Dropdown)
	}
	b2, picked := b.Key(key("enter"))
	if b2.Open || Pick(picked)() != (OpenMsg{}) {
		t.Errorf("enter on Open…: %q, open %v", picked, b2.Open)
	}
	at := BooksAt(Title("kit welcome", false))
	_, left := b.box(at, 120)
	if _, picked := b.Click(left+3, Top+1, at, 120); Pick(picked)() != (OpenMsg{}) {
		t.Errorf("a click on Open…: %q", picked)
	}
	// Manage… asks for the workspaces' screen
	if _, picked := b.Click(left+3, Top+2, at, 120); Pick(picked)() != (SwitchMsg{Manage}) {
		t.Errorf("a click on Manage…: %q", picked)
	}
}

// The repositories opened by their folder follow the groups, picked by their folder; the one shown is not drawn, as
// the title names it, and one that a favorite names already is not drawn twice.
func TestBooksDropsDownTheRecentRepositoriesPickedByTheirFolder(t *testing.T) {
	ws := inGroups
	ws.Recent = []kit.WorkspaceRecent{{Name: "github.com/o/a", Member: "github.com/o/a", Checkout: "/code/a"},
		{Name: "/srv/scratch", Checkout: "/srv/scratch"},
		{Name: "github.com/o/b", Member: "github.com/o/b", Checkout: "/far/b"}}
	b := BooksMenu{Workspace: ws}
	got, acting := names(b)
	if strings.Join(got, "|") != "w 2|v 1||w 2|o/b · not cloned||scratch||Open…|Manage…" {
		t.Errorf("the lines: %q", got)
	}
	if acting[len(acting)-3] != 6 || acting[len(acting)-2] != 8 {
		t.Errorf("the lines that act: %v", acting)
	}
	if items := b.Items(); items[6].Glyph != Opened {
		t.Errorf("a recent one's glyph: %+v", items[6])
	}
	if b.target(6) != "/srv/scratch" {
		t.Errorf("scratch picks %q", b.target(6))
	}
}

// A worktree is no workspace of its own: in a worktree of o/a, o/a's desk among the recent is the repository shown, and
// is not drawn, as the folder shown is not.
func TestBooksTellsTwoFoldersOfOneRepositoryApart(t *testing.T) {
	ws := kit.Workspace{Kind: "repository", Name: "github.com/o/a", Member: "github.com/o/a", Checkout: "/code/a/worktrees/1-x",
		Recent: []kit.WorkspaceRecent{{Name: "github.com/o/a", Member: "github.com/o/a", Checkout: "/code/a/worktrees/1-x"},
			{Name: "github.com/o/a", Member: "github.com/o/a", Checkout: "/code/a"},
			{Name: "github.com/o/c", Member: "github.com/o/c", Checkout: "/code/c"}}}
	got, acting := names(BooksMenu{Workspace: ws})
	for k := range got {
		got[k] = ansi.Strip(got[k])
	}
	if strings.Join(got, "|") != "o/c||Open…|Manage…" || len(acting) != 3 {
		t.Errorf("the lines: %q, acting %v", got, acting)
	}
}

func TestBooksDropsDownTheGroupsThenTheFavorites(t *testing.T) {
	got, acting := names(BooksMenu{Workspace: inGroups})
	want := []string{"w 2", "v 1", "", "w 2", "o/b · not cloned", "", "Open…", "Manage…"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("the lines: %q", got)
	}
	if len(acting) != 5 || acting[0] != 0 || acting[1] != 1 || acting[2] != 3 {
		t.Errorf("the lines that act: %v", acting)
	}
	// over a group, the group itself is dim among the favorites, and no group lines come first
	b := BooksMenu{Workspace: group("w", 2)}
	b.Workspace.Favorites = inGroups.Favorites
	if got, acting := names(b); got[0] != "w 2" || len(acting) != 2 {
		t.Errorf("over w: %q, acting %v", got, acting)
	}
}

// Every group in the file follows the favorites, each a line of its own, but those drawn above already: the groups the
// repository belongs to, and the favorites. However many favorites there are, each has its line.
func TestBooksDropsDownEveryOtherGroupAfterTheFavorites(t *testing.T) {
	ws := inGroups
	ws.Workspaces = []kit.WorkspaceGroup{{Name: "w", Members: 2}, {Name: "x", Members: 3}, {Name: "v", Members: 1},
		{Name: "y", Members: 4}}
	got, acting := names(BooksMenu{Workspace: ws})
	want := []string{"w 2", "v 1", "", "w 2", "o/b · not cloned", "", "x 3", "y 4", "", "Open…", "Manage…"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("the lines: %q", got)
	}
	if len(acting) != 7 || acting[3] != 6 || acting[4] != 7 {
		t.Errorf("the lines that act: %v", acting)
	}
	if items := (BooksMenu{Workspace: ws}).Items(); items[6].Glyph != Books {
		t.Errorf("another group's glyph: %+v", items[6])
	}
	b := BooksMenu{Workspace: ws}.Opened()
	for range 3 {
		b, _ = b.Key(key("down"))
	}
	if _, picked := b.Key(key("enter")); picked != "x" {
		t.Errorf("down to x, enter: %q", picked)
	}
	// over a group, it stands among the others, dim
	over := group("x", 3)
	over.Workspaces = ws.Workspaces
	if got, acting := names(BooksMenu{Workspace: over}); strings.Join(got, "|") != "w 2|x 3|v 1|y 4||Open…|Manage…" ||
		len(acting) != 5 || acting[0] != 0 || acting[1] != 2 {
		t.Errorf("over x: %q, acting %v", got, acting)
	}
	// no cap: every favorite has its line
	many := kit.Workspace{Kind: "none"}
	for k := range 30 {
		name := "g" + strconv.Itoa(k)
		many.Favorites = append(many.Favorites, kit.WorkspaceFavorite{Name: name, Kind: "group", Members: 1})
		many.Workspaces = append(many.Workspaces, kit.WorkspaceGroup{Name: name, Members: 1})
	}
	if got, _ := names(BooksMenu{Workspace: many}); len(got) != 33 {
		t.Errorf("30 favorites: %d lines", len(got))
	}
}

func TestBooksPicksAGroupByClickAndByKey(t *testing.T) {
	b := BooksMenu{Workspace: inGroups}.Opened()
	if !b.Open || b.At != 0 {
		t.Fatalf("opened: %+v", b.Dropdown)
	}
	b, _ = b.Key(key("down"))
	if b, picked := b.Key(key("enter")); picked != "v" || b.Open {
		t.Errorf("down, enter: %q, open %v", picked, b.Open)
	}
	title := Title("kit welcome", false)
	at := BooksAt(title)
	box, left := Box(b.Items(), b.enabled, 0, Books, at, 120)
	x := left + 3
	if b2, picked := b.Opened().Click(x, Top+1, at, 120); picked != "w" || b2.Open {
		t.Errorf("a click on w: %q, open %v", picked, b2.Open)
	}
	if b2, picked := b.Opened().Click(x, Top+1+4, at, 120); picked != "" || !b2.Open {
		t.Errorf("a click on o/b · not cloned: %q, open %v", picked, b2.Open)
	}
	if b2, picked := b.Opened().Click(left+ansi.StringWidth(box[0])+2, Top+1, at, 120); picked != "" || b2.Open {
		t.Errorf("a click off the box: %q, open %v", picked, b2.Open)
	}
	if msg := Pick("w")(); msg != (PickMsg{"w"}) {
		t.Errorf("a pick asks kittui: %#v", msg)
	}
	if Pick("") != nil {
		t.Error("no pick asks kittui to switch")
	}
	if b2, _ := b.Opened().Key(key("esc")); b2.Open {
		t.Error("esc leaves it open")
	}
	lines := b.Opened().Overlay(make([]string, 12), at, 120)
	if !strings.Contains(ansi.Strip(lines[Top]), Workspaces) || !strings.Contains(ansi.Strip(lines[Top+1]), "w 2") {
		t.Errorf("drawn:\n%s", ansi.Strip(strings.Join(lines, "\n")))
	}
	if lines := (BooksMenu{Workspace: inGroups}).Overlay(make([]string, 3), at, 120); lines[0] != "" {
		t.Errorf("closed, it draws: %q", lines[0])
	}
}

func TestBooksSaysKitsLineWhenKitCouldNot(t *testing.T) {
	b := BooksMenu{Fallback: "empowerite/kit"}.Told(WorkspaceMsg{Refusal: "no kit verb 'workspaces'"})
	if ansi.Strip(b.Label()) != "- empowerite/kit" {
		t.Errorf("the label: %q", ansi.Strip(b.Label()))
	}
	if items := b.Items(); items[0].Glyph != "❌" || items[0].Name != "no kit verb 'workspaces'" || b.enabled(0) {
		t.Errorf("the first line: %+v", items[0])
	}
	// kit saying it later takes kit's line away
	b = b.Told(WorkspaceMsg{Workspace: inGroups})
	if b.Refusal != "" || b.Workspace.Name != "github.com/o/a" {
		t.Errorf("told anew: %+v", b)
	}
	// kit refusing later keeps the workspace it said before
	if b = b.Told(WorkspaceMsg{Refusal: "x"}); b.Workspace.Name != "github.com/o/a" || b.Refusal != "x" {
		t.Errorf("refused after: %+v", b)
	}
}

func TestBooksIsClickedOnItsCells(t *testing.T) {
	title := Title("kit welcome", false)
	at := BooksAt(title)
	b := BooksMenu{Workspace: kit.Workspace{Kind: "repository", Member: "github.com/empowerite/kit"}}
	end := at + ansi.StringWidth(b.Label()) - 1
	if !b.On(at, Top, at) || !b.On(end, Top, at) || b.On(end+1, Top, at) || b.On(at-1, Top, at) || b.On(at, Top+1, at) {
		t.Errorf("on its cells %d..%d", at, end)
	}
	if line := title + " · " + b.Label(); ansi.StringWidth(title)+3 != at || !strings.HasPrefix(ansi.Strip(
		from(line, at)), Workspaces) {
		t.Errorf("BooksAt: %d in %q", at, ansi.Strip(line))
	}
}

// from gives the line from the column at on, without its escapes.
func from(line string, at int) string { return ansi.Cut(line, at, ansi.StringWidth(line)) }
