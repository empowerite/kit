// The clones a walk finds stand in the Open… dialog as a table: a tree of repositories, selected, added to a
// workspace, taken out of the one shown, and opened.
package opener

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// group is a workspace shown, the group g, whose one member's checkout is b/c under dir, beside the group h.
func group(dir string) kit.Workspace {
	return kit.Workspace{Kind: "group", Name: "g", Members: []kit.WorkspaceMember{{Repository: "example.com/b/c",
		Checkout: filepath.Join(dir, "b", "c")}}, Workspaces: []kit.WorkspaceGroup{{Name: "g"}, {Name: "h"}}}
}

// found gives the dialog over the workspace, the clones under dir found.
func found(t *testing.T, dir string, ws kit.Workspace) Dialog {
	t.Helper()
	d := New(dir, "/nowhere", ws)
	was := d
	d, _ = d.Key(key("ctrl+f"))
	walk := d.Walk(was)
	if walk == nil {
		t.Fatal("⌃F walks nothing")
	}
	return d.Found(walk().(ClonesMsg))
}

// cursorOn gives the dialog with the table's cursor moved down to the row whose key ends so.
func cursorOn(t *testing.T, d Dialog, suffix string) Dialog {
	t.Helper()
	for range 20 {
		if rows, at := d.table.Rows(), d.table.Table().Cursor; at < len(rows) && strings.HasSuffix(rows[at].Key, suffix) {
			return d
		}
		d, _ = d.Key(key("down"))
	}
	t.Fatalf("no row %q:\n%s", suffix, drawn(d))
	return d
}

// lineAt gives the screen line and column of the first line of the dialog's box drawn over 30 lines, 100 wide, that
// holds the text, where the text starts.
func lineAt(t *testing.T, d Dialog, text string) (x, y int) {
	t.Helper()
	p := d.placed(30, 100)
	for k, l := range p.Lines {
		if i := strings.Index(ansi.Strip(l), text); i >= 0 {
			return p.Left + ansi.StringWidth(ansi.Strip(l)[:i]), p.Top + k
		}
	}
	t.Fatalf("no line %q:\n%s", text, drawn(d))
	return 0, 0
}

// click gives the dialog after a plain click on a cell of a screen 30 lines high and 100 wide, and what it asks.
func click(d Dialog, x, y int) (Dialog, Ask) {
	return d.Click(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft}, 30, 100)
}

// ⌃F walks the folder typed while the dialog says it is finding clones; the clones then stand in place of the folders
// as a table, each repository a heading over its desk and its worktrees, and enter on a clone asks to open it; esc
// goes back to the folders, esc again closes the dialog, and typing goes back to the folders too.
func TestOpenFindsTheClonesUnderTheFolderTypedAsATreeOfRepositories(t *testing.T) {
	dir := cloneTree(t)
	d := newOpen(dir, "/nowhere")
	if v := drawn(d); !strings.Contains(v, Finding+" Find clones under /") {
		t.Fatalf("no line to find clones:\n%s", v)
	}
	was := d
	d, _ = d.Key(key("ctrl+f"))
	walk := d.Walk(was)
	if d.Walk(d) != nil {
		t.Error("a walk asked again")
	}
	if v := drawn(d); !strings.Contains(v, "finding clones…") || strings.Contains(v, Folder+" a/") {
		t.Errorf("while finding:\n%s", v)
	}
	msg := walk().(ClonesMsg)
	if typed := typedIn(d, "a").Found(msg); typed.table != nil {
		t.Errorf("a stale answer is shown:\n%s", drawn(typed))
	}
	d = d.Found(msg)
	v := drawn(d)
	for _, want := range []string{"▾ a 2", "a/worktrees/w1", "▾ b/c 1", "Default", "clone", Adding +
		" Add All to a Workspace…", Removing + " Remove from the Workspace Shown", Opening + " Open"} {
		if !strings.Contains(v, want) {
			t.Errorf("no %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, Folder+" plain/") || strings.Contains(v, "../") {
		t.Errorf("the folders stand still:\n%s", v)
	}
	d = cursorOn(t, d, "/worktrees/w1")
	if !strings.Contains(drawn(d), Opening+" Open a/worktrees/w1") {
		t.Errorf("Open names no clone:\n%s", drawn(d))
	}
	if _, ask := d.Key(key("enter")); ask.Open != filepath.Join(dir, "a", "worktrees", "w1") {
		t.Errorf("enter on w1: %+v", ask)
	}
	back, _ := d.Key(key("esc"))
	if v := drawn(back); !back.Open || !strings.Contains(v, Folder+" plain/") || back.table != nil {
		t.Errorf("esc from the clones:\n%s", v)
	}
	if back, _ = back.Key(key("esc")); back.Open {
		t.Error("esc from the folders leaves the dialog open")
	}
	if typed := typedIn(d, "b"); typed.table != nil || !strings.HasSuffix(typed.Typed, "/b") {
		t.Errorf("typing leaves the clones: %q\n%s", typed.Typed, drawn(typed))
	}
}

// < closes every repository to its heading, and > opens them again; enter on a heading closes and opens it.
func TestTheClonesTreeFoldsALevelAtATime(t *testing.T) {
	d := found(t, cloneTree(t), kit.Workspace{})
	d, _ = d.Key(key("<"))
	if v := drawn(d); !strings.Contains(v, "▸ a 2") || strings.Contains(v, "a/worktrees/w1") {
		t.Errorf("after <:\n%s", v)
	}
	d, _ = d.Key(key(">"))
	if v := drawn(d); !strings.Contains(v, "▾ a 2") || !strings.Contains(v, "a/worktrees/w1") {
		t.Errorf("after >:\n%s", v)
	}
	d = cursorOn(t, d, "\x00a")
	if d, _ = d.Key(key("enter")); !strings.Contains(drawn(d), "▸ a 2") {
		t.Errorf("enter on a's heading:\n%s", drawn(d))
	}
}

// A clone whose repository is a member of the group shown is marked so, and marked again as kit says the group anew.
func TestTheClonesMarkTheMembersOfTheGroupShown(t *testing.T) {
	dir := cloneTree(t)
	d := found(t, dir, group(dir))
	marked := func(d Dialog, clone string) bool {
		for _, l := range strings.Split(drawn(d), "\n") {
			if f := strings.Fields(strings.Trim(l, "│ ")); len(f) > 0 && f[0] == clone {
				return strings.Contains(l, Group)
			}
		}
		t.Fatalf("no row %s:\n%s", clone, drawn(d))
		return false
	}
	if !marked(d, "b/c") || marked(d, "a") {
		t.Errorf("the members:\n%s", drawn(d))
	}
	ws := group(dir)
	ws.Members = append(ws.Members, kit.WorkspaceMember{Checkout: filepath.Join(dir, "a")})
	if d = d.Told(ws); !marked(d, "a") || !marked(d, "a/worktrees/w1") {
		t.Errorf("told a is a member:\n%s", drawn(d))
	}
}

// Space selects a clone, or a heading's every clone; with none selected Add adds every clone found, and with some only
// those: + drops the groups down, and the one chosen is asked of kit, the clones by their folders. kit's line, when it
// is done, stands under what is typed, the selection cleared; Remove takes the selected out of the group shown.
func TestTheClonesSelectedAreAddedToAWorkspaceAndRemovedFromTheOneShown(t *testing.T) {
	dir := cloneTree(t)
	all := []string{filepath.Join(dir, "a"), filepath.Join(dir, "a", "worktrees", "w1"), filepath.Join(dir, "b", "c"),
		filepath.Join(dir, "d", "1", "2", "3", "4", "5")}
	d := found(t, dir, group(dir))
	if d, ask := d.Key(key("-")); ask.Words != nil || d.Busy {
		t.Errorf("- with nothing selected: %+v", ask)
	}
	d, _ = d.Key(key("+"))
	if v := drawn(d); !d.adding.Open || !strings.Contains(strings.Join(d.Overlay(make([]string, 30), 100), "\n"),
		"h") {
		t.Fatalf("+ drops no groups down:\n%s", v)
	}
	d2, ask := d.Key(key("enter"))
	if want := append([]string{"workspaces", "--add=g"}, all...); !slices.Equal(ask.Words, want) || !d2.Busy {
		t.Errorf("Add All: %q", ask.Words)
	}
	d = cursorOn(t, d2.Refused("no"), "\x00a")
	if !strings.Contains(drawn(d), Failed+" no") {
		t.Errorf("kit's refusal:\n%s", drawn(d))
	}
	d, _ = d.Key(key("space"))
	if v := drawn(d); !strings.Contains(v, "2 selected") || !strings.Contains(v, Adding+" Add 2 Clones to a Workspace…") {
		t.Errorf("space on a's heading:\n%s", v)
	}
	d, _ = d.Key(key("+"))
	d, _ = d.Key(key("down"))
	d, ask = d.Key(key("enter"))
	if want := []string{"workspaces", "--add=h", all[0], all[1]}; !slices.Equal(ask.Words, want) {
		t.Errorf("Add 2 to h: %q", ask.Words)
	}
	if d = d.Done("📚 h: added it"); d.Busy || !strings.Contains(drawn(d), "📚 h: added it") || d.table.selected() != 0 {
		t.Errorf("done:\n%s", drawn(d))
	}
	d, _ = d.Key(key("space"))
	if _, ask = d.Key(key("-")); !slices.Equal(ask.Words, []string{"workspaces", "--remove=g", all[0], all[1]}) {
		t.Errorf("Remove: %q", ask.Words)
	}
	d, _ = d.Key(key("ctrl+a"))
	if d.table.selected() != 4 {
		t.Errorf("⌃A selected %d", d.table.selected())
	}
	d = cursorOn(t, d.Done(""), "/a")
	d, _ = d.Key(key("shift+down"))
	if got := table.SelectedRows(d.table); len(got) != 2 || got[1].Key != all[1] {
		t.Errorf("⇧↓ from a selected %d", len(got))
	}
}

// Add with no group to add to, Remove over a repository, and Open on a heading are dim, and ask nothing.
func TestTheClonesActionsAreDimWhileTheyCannotAct(t *testing.T) {
	d := found(t, cloneTree(t), kit.Workspace{Kind: "repository"})
	d, _ = d.Key(key("ctrl+a"))
	body := d.body(90)
	for _, l := range body[len(body)-3:] {
		if !strings.HasPrefix(l.text, style.Off("")[:4]) {
			t.Errorf("not dim: %q", ansi.Strip(l.text))
		}
	}
	if d, _ = d.Key(key("+")); d.adding.Open {
		t.Error("+ with no group drops one down")
	}
	if _, ask := d.Key(key("-")); ask.Words != nil {
		t.Errorf("- over a repository: %q", ask.Words)
	}
	if _, ask := d.open(); ask.Open != "" {
		t.Errorf("Open on no clone: %q", ask.Open)
	}
}

// Every action over the clones is a plain click: on a heading's mark it selects its clones, on Add it drops the groups
// down, and a click on one adds to it, one off the dropdown closing it; on a clone and then Open it asks to open it;
// on Remove it removes; and on the table's preset selector the presets drop down, as they do on any table.
func TestTheClonesAreDoneByPlainClicks(t *testing.T) {
	dir := cloneTree(t)
	d := newOpen(dir, "/nowhere")
	d.ws = group(dir)
	was := d
	d, _ = clickOn(t, d, "Find clones under")
	d = d.Found(d.Walk(was)().(ClonesMsg))
	x, y := lineAt(t, d, "▾ a 2")
	if d, _ = click(d, x-2, y); d.table.selected() != 2 {
		t.Fatalf("a click on a's mark selected %d:\n%s", d.table.selected(), drawn(d))
	}
	if d, _ = clickOn(t, d, "Add 2 Clones"); !d.adding.Open {
		t.Fatal("a click on Add drops nothing down")
	}
	over := d.Overlay(make([]string, 30), 100)
	for k, l := range over {
		if i := strings.Index(ansi.Strip(l), Group+" h"); i >= 0 {
			if _, ask := click(d, ansi.StringWidth(ansi.Strip(l)[:i])+3, k); !slices.Contains(ask.Words, "--add=h") {
				t.Errorf("a click on h: %q", ask.Words)
			}
		}
	}
	if off, _ := click(d, 0, 0); off.adding.Open || !off.Open {
		t.Errorf("a click off the dropdown: open %v, dialog %v", off.adding.Open, off.Open)
	}
	d.adding.Open = false
	if _, ask := clickOn(t, d, "Remove from g"); !slices.Contains(ask.Words, "--remove=g") {
		t.Errorf("a click on Remove: %q", ask.Words)
	}
	x, y = lineAt(t, d, "  b/c")
	d, _ = click(d, x+4, y)
	if _, ask := clickOn(t, d, Opening+" Open"); ask.Open != filepath.Join(dir, "b", "c") {
		t.Errorf("a click on b/c, then on Open: %+v", ask)
	}
	if d, _ = clickOn(t, d, "Default"); !d.table.Layered() {
		t.Errorf("a click on the preset selector drops nothing down:\n%s", drawn(d))
	}
	if v := strings.Join(d.Overlay(make([]string, 30), 100), "\n"); !strings.Contains(ansi.Strip(v), "Flat") {
		t.Errorf("the presets are not drawn:\n%s", ansi.Strip(v))
	}
	if d, _ = d.Key(key("esc")); d.table == nil || d.table.Layered() {
		t.Error("esc in the presets left the clones, or the presets open")
	}
	if off, _ := click(d, 0, 29); off.Open {
		t.Error("a click off the box leaves the dialog open")
	}
}

// The wheel scrolls the clones' table, and the keys bar names its keys: the rows', esc back, a level, add, remove,
// and open; the dropdown's while it is open.
func TestTheClonesScrollAndNameTheirKeys(t *testing.T) {
	dir := t.TempDir()
	for k := range 40 {
		if err := os.MkdirAll(filepath.Join(dir, "r"+string(rune('a'+k%26))+string(rune('a'+k/26)), ".git"),
			0o755); err != nil {
			t.Fatal(err)
		}
	}
	d := found(t, dir, group(dir))
	d = d.Wheel(tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 50, Y: 12}, 30, 100)
	if d.table.Table().Offset == 0 {
		t.Errorf("the wheel scrolled nothing:\n%s", drawn(d))
	}
	got := (shell.Path{d.Keys()}).String()
	for _, want := range []string{"esc back", "␣·⇧↑↓ select", "⌃A all", ">< level", "+ add", "- remove", "⏎ open"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in %q", want, got)
		}
	}
	d, _ = d.Key(key("+"))
	if got := (shell.Path{d.Keys()}).String(); got != "esc close ↑↓ move ⏎ add" {
		t.Errorf("the dropdown's keys: %q", got)
	}
	d, _ = d.Key(key("esc"))
	if d.adding.Open || d.table == nil {
		t.Error("esc closes not the dropdown alone")
	}
	d, _ = d.Key(key("up"))
	d, _ = d.Key(key("up"))
	if got := (shell.Path{d.Keys()}).String(); strings.Contains(got, "+ add") || !d.Keys().Owns {
		t.Errorf("the control line's keys: %q", got)
	}
}

// A linked worktree's desk is the repository whose .git holds it, from where it stands or absolute; a .git file that
// names no such folder, or does not read, leaves the worktree its own desk.
func TestADesksIsTheRepositoryAWorktreeBelongsTo(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, gitdir string) string {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, ".git"), []byte(gitdir+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	stat := func(path string) os.FileInfo {
		info, err := os.Lstat(filepath.Join(path, ".git"))
		if err != nil {
			t.Fatal(err)
		}
		return info
	}
	abs := write("abs", "gitdir: /srv/r/.git/worktrees/abs")
	if got := desk(abs, stat(abs)); got != "/srv/r" {
		t.Errorf("absolute: %q", got)
	}
	odd := write("odd", "gitdir: /srv/elsewhere")
	if got := desk(odd, stat(odd)); got != odd {
		t.Errorf("no .git in it: %q", got)
	}
	gone := write("gone", "")
	info := stat(gone)
	if err := os.Remove(filepath.Join(gone, ".git")); err != nil {
		t.Fatal(err)
	}
	if got := desk(gone, info); got != gone {
		t.Errorf("unread: %q", got)
	}
	c := &clones{found: ClonesMsg{Under: "/srv/w"}, home: "/srv"}
	for desk, want := range map[string]string{"/srv/w": "w", "/srv/w/x": "x", "/srv/o": "~/o", "/opt": "/opt"} {
		if got := c.repository(desk); got != want {
			t.Errorf("the heading of %s: %q", desk, got)
		}
	}
}

// The catalog reads.
func TestTheClonesCatalogReads(t *testing.T) {
	if _, err := table.ReadCatalog(catalogJSON); err != nil || catalog[clonesTable] == nil {
		t.Fatal(err)
	}
}

// clickOn gives the dialog after a plain click where the text starts on the box drawn over 30 lines, 100 wide, and
// what it asks.
func clickOn(t *testing.T, d Dialog, text string) (Dialog, Ask) {
	t.Helper()
	x, y := lineAt(t, d, text)
	return click(d, x, y)
}

// While a walk is on, ⌃F begins no other and esc goes back to the folders, as the keys bar says; a walk that stopped
// at the most it finds says so; one clone selected is counted so; and while kit is asked, Add asks nothing more. A
// click on the head of the box, or in the presets dropped down, is no action's.
func TestTheClonesWalkStopsAndAsksOnce(t *testing.T) {
	dir := cloneTree(t)
	d := newOpen(dir, "/nowhere")
	d, _ = d.Key(key("ctrl+f"))
	if again, _ := d.Key(key("ctrl+f")); again.walks != d.walks {
		t.Error("⌃F while finding walks again")
	}
	if got := (shell.Path{d.Keys()}).String(); !strings.HasPrefix(got, "esc back") {
		t.Errorf("the keys while finding: %q", got)
	}
	if back, _ := d.Key(key("esc")); back.finding || !back.Open {
		t.Error("esc while finding")
	}
	d = d.Found(ClonesMsg{Walk: d.walks, Under: d.Path(), Clones: []string{"a"}, Desks: []string{filepath.Join(dir,
		"a")}, Capped: true})
	if v := drawn(d); !strings.Contains(v, "stopped at the first 1 clone") {
		t.Errorf("capped:\n%s", v)
	}
	d.ws = group(dir)
	d = cursorOn(t, d, "/a")
	d, _ = d.Key(key("space"))
	if v := drawn(d); !strings.Contains(v, "Add 1 Clone to") {
		t.Errorf("one selected:\n%s", v)
	}
	busy, _ := d.add("g")
	if _, ask := busy.add("g"); ask.Words != nil {
		t.Errorf("asked twice: %q", ask.Words)
	}
	if same, ask := clickOn(t, d, "the clones under"); ask.Words != nil || !same.Open {
		t.Errorf("a click on the head: %+v", ask)
	}
	d, _ = clickOn(t, d, "Default")
	if d, _ = click(d, 1, 1); d.table.Layered() || !d.Open {
		t.Error("a click off the presets leaves them open, or closes the dialog")
	}
}
