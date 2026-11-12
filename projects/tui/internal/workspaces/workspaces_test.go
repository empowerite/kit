// The workspaces' screen lists the groups kit keeps, a row each, and its actions run kit workspaces, each from the app's
// menu by a plain click or by its key, the ones that want an answer through a dialog that says kit's line when it
// refuses.
package workspaces

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// listJSON is what the tests' kit lists: w, a favorite of two members, and solo, of one.
const listJSON = `{"file":"/c/kit/workspaces.yml","roots":["/code"],"favorites":["w"],"workspaces":[
	{"name":"w","members":2,"repos":["github.com/o/a","github.com/o/b"],"favorite":true},
	{"name":"solo","members":1,"repos":["github.com/me/solo"],"favorite":false}]}`

// opened gives the screen over the kit package's fake kit, listing list, at 120×30, and the file kit's calls are
// recorded in.
func opened(t *testing.T, list string) (Model, string) {
	t.Helper()
	fades = time.Millisecond
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir, record := t.TempDir(), filepath.Join(t.TempDir(), "record")
	if err := os.WriteFile(filepath.Join(dir, "list.json"), []byte(list), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_WORKSPACES", dir)
	t.Setenv("FAKE_KIT_RECORD", record)
	m := Open(testkit.Runner(t, "kit"))
	m, _ = testkit.Drive(m, nil, testkit.Start(m, tea.WindowSizeMsg{Width: 120, Height: 30})...)
	return m, record
}

// view gives the screen drawn, without its escapes.
func view(m Model) string { return ansi.Strip(m.View().Content) }

// asks says whether a message is one the screen asks kittui for, which the tests catch rather than feed back.
func asks(msg tea.Msg) bool {
	switch msg.(type) {
	case shell.ChangedMsg, shell.PickMsg, shell.SwitchMsg:
		return true
	}
	return false
}

// drive sends the messages, and gives the screen and what it asked kittui for; what kit did is left standing, its fade
// caught too.
func drive(m Model, msgs ...tea.Msg) (Model, []tea.Msg) {
	m, out := testkit.Drive(m, func(msg tea.Msg) bool {
		_, faded := msg.(shell.FadedMsg)
		return faded || asks(msg)
	}, msgs...)
	return m, slices.DeleteFunc(out, func(msg tea.Msg) bool { return !asks(msg) })
}

// recorded gives the kit calls the fake recorded.
func recorded(t *testing.T, record string) string {
	t.Helper()
	got, _ := os.ReadFile(record)
	return string(got)
}

// lineAt gives the screen line that holds the text, and its column.
func lineAt(t *testing.T, m Model, text string) (x, y int) {
	t.Helper()
	for k, l := range strings.Split(view(m), "\n") {
		if at := strings.Index(l, text); at >= 0 {
			return ansi.StringWidth(l[:at]), k
		}
	}
	t.Fatalf("no line holds %q:\n%s", text, view(m))
	return 0, 0
}

// left gives a plain click at x, y.
func left(x, y int) tea.MouseClickMsg { return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft} }

func TestTheCatalogReads(t *testing.T) {
	if catalog[groupsTable] == nil || len(catalog[groupsTable].Fields) != 4 {
		t.Fatalf("the catalog: %+v", catalog)
	}
}

func TestTheScreenListsEveryGroupARowEach(t *testing.T) {
	m, _ := opened(t, listJSON)
	v := view(m)
	for _, want := range []string{"kit workspaces", "w", "solo", "github.com/o/a", "github.com/me/solo", Star} {
		if !strings.Contains(v, want) {
			t.Errorf("no %q:\n%s", want, v)
		}
	}
	if g, ok := m.current(); !ok || g.Name != "w" {
		t.Errorf("the row under the cursor: %+v, %v", g, ok)
	}
}

func TestWithNoGroupOnlyNewActs(t *testing.T) {
	m, _ := opened(t, `{"file":"/f","roots":[],"favorites":[],"workspaces":[]}`)
	if v := view(m); !strings.Contains(v, "no group yet") {
		t.Errorf("no group:\n%s", v)
	}
	menu := m.menu()
	if slices.Contains(menu.Dim, New) || !slices.Contains(menu.Dim, Rename) || !slices.Contains(menu.Dim, DeleteOne) {
		t.Errorf("dim: %q", menu.Dim)
	}
	// the app's menu drops down whole over the few lines the screen draws, its bottom edge too, the keys bar last
	m, _ = drive(m, tea.KeyPressMsg{Code: '=', Text: "="})
	v := view(m)
	lines := strings.Split(v, "\n")
	if !strings.Contains(v, Workspace) || !strings.Contains(v, "╰") || len(lines) != 30 {
		t.Errorf("the menu over no group, %d lines:\n%s", len(lines), v)
	}
}

func TestNewTypesANameAndItsRepositoriesAndKitMakesTheGroup(t *testing.T) {
	m, record := opened(t, listJSON)
	m, _ = drive(m, testkit.Press("n"))
	if !m.prompt.open || !strings.Contains(view(m), "New workspace") {
		t.Fatalf("n opened no dialog:\n%s", view(m))
	}
	m, asked := drive(m, testkit.Typed("tools github.com/o/*-tool")...)
	m, asked = drive(m, testkit.Press("enter"))
	if got := recorded(t, record); !strings.Contains(got, "workspaces --new=tools github.com/o/*-tool") {
		t.Errorf("kit was not asked: %q", got)
	}
	if m.prompt.open || !strings.Contains(m.note.Text, "the fake did it") {
		t.Errorf("after kit did it: open %v, %q", m.prompt.open, m.note.Text)
	}
	if len(asked) != 1 || asked[0] != (shell.ChangedMsg{}) {
		t.Errorf("kittui was not told: %#v", asked)
	}
}

// Rename… is done by plain clicks alone: the = drops the menu down, a click on Rename… opens its dialog, its name
// typed already, and a click on its last line asks kit.
func TestRenameIsDoneByPlainClicksAndFollowsTheGroup(t *testing.T) {
	m, record := opened(t, listJSON)
	m, _ = drive(m, left(2, 0))
	x, y := lineAt(t, m, Folder+" "+Workspace)
	m, _ = drive(m, left(x, y))
	x, y = lineAt(t, m, Rename)
	m, _ = drive(m, left(x, y))
	if !m.prompt.open || m.prompt.typed != "w" || m.appMenu.Open || m.group.Open {
		t.Fatalf("Rename… opened %+v", m.prompt)
	}
	m, _ = drive(m, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	m, _ = drive(m, testkit.Typed("core")...)
	x, y = lineAt(t, m, Do+" Rename")
	m, asked := drive(m, left(x, y))
	if got := recorded(t, record); !strings.Contains(got, "workspaces --rename=w core") {
		t.Errorf("kit was not asked: %q", got)
	}
	if len(asked) != 1 || asked[0] != (shell.ChangedMsg{From: "w", To: "core"}) {
		t.Errorf("kittui was not told: %#v", asked)
	}
}

func TestARefusalStandsInTheDialog(t *testing.T) {
	m, _ := opened(t, listJSON)
	t.Setenv("FAKE_KIT_REFUSE", "workspaces-edit")
	m, _ = drive(m, testkit.Press("a"))
	m, _ = drive(m, testkit.Typed("github.com/o/c")...)
	m, asked := drive(m, testkit.Press("enter"))
	if !m.prompt.open || !strings.Contains(view(m), "refuses the edit") || len(asked) != 0 {
		t.Errorf("the refusal: open %v, asked %v\n%s", m.prompt.open, asked, view(m))
	}
	if m, _ = drive(m, testkit.Press("esc")); m.prompt.open {
		t.Error("esc leaves the dialog open")
	}
}

func TestAnOwnersRepositoriesAreListedByGhAndAPatternKeepsSome(t *testing.T) {
	m, record := opened(t, listJSON)
	m, _ = drive(m, testkit.Press("o"))
	m, _ = drive(m, testkit.Typed("empowerite homebrew-*")...)
	m, _ = drive(m, testkit.Press("enter"))
	want := "workspaces --add=w --from=gh repo list empowerite --limit 1000 --json url --jq '.[].url' " +
		"github.com/empowerite/homebrew-*"
	if got := recorded(t, record); !strings.Contains(got, want) {
		t.Errorf("kit was asked %q", got)
	}
	// an owner that is no name makes no command
	p := m.prompt
	m, _ = drive(m, testkit.Press("o"))
	m, _ = drive(m, testkit.Typed("x;rm")...)
	if _, ok := m.prompt.makes(); ok {
		t.Errorf("a bad owner makes a command; was %+v", p)
	}
}

func TestRemoveFromACommandFavoriteDeleteAndSwitch(t *testing.T) {
	m, record := opened(t, listJSON)
	m, _ = drive(m, testkit.Press("c"))
	m, _ = drive(m, testkit.Typed("cat repos.txt")...)
	m, _ = drive(m, testkit.Press("enter"))
	m, _ = drive(m, testkit.Press("x"))
	m, _ = drive(m, testkit.Typed("github.com/o/b")...)
	m, _ = drive(m, testkit.Press("enter"))
	m, _ = drive(m, testkit.Press("f"))
	m, _ = drive(m, testkit.Press("d"))
	if !strings.Contains(view(m), "its checkouts stay as they are") {
		t.Errorf("delete asks first:\n%s", view(m))
	}
	m, asked := drive(m, testkit.Press("enter"))
	got := recorded(t, record)
	for _, want := range []string{"workspaces --add=w --from=cat repos.txt", "workspaces --remove=w github.com/o/b",
		"workspaces --unfavorite=w", "workspaces --delete=w"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in %q", want, got)
		}
	}
	if len(asked) != 1 || asked[0] != (shell.ChangedMsg{From: "w"}) {
		t.Errorf("a delete told kittui %#v", asked)
	}
	_, asked = drive(m, testkit.Press("enter"))
	if len(asked) != 1 || asked[0] != (shell.PickMsg{Workspace: "w"}) {
		t.Errorf("enter switches: %#v", asked)
	}
}

func TestTheSearchNarrowsTheRows(t *testing.T) {
	m, _ := opened(t, listJSON)
	m, _ = drive(m, testkit.Press("/"))
	m, _ = drive(m, testkit.Typed("solo")...)
	if v := view(m); strings.Contains(v, "github.com/o/a") || !strings.Contains(v, "github.com/me/solo") {
		t.Errorf("the search did not narrow:\n%s", v)
	}
	m, _ = drive(m, testkit.Press("enter"))
	if g, ok := m.current(); m.search.Focused || !ok || g.Name != "solo" {
		t.Errorf("after the search: focused %v, %+v", m.search.Focused, g)
	}
}

// bothJSON is a kit that lists w and solo, both favorites, so kittui's Auto-group folds the ⭐ every row shares into
// a heading.
const bothJSON = `{"file":"/f","roots":[],"favorites":["w","solo"],"workspaces":[
	{"name":"w","members":2,"repos":["github.com/o/a","github.com/o/b"],"favorite":true},
	{"name":"solo","members":1,"repos":["github.com/me/solo"],"favorite":true}]}`

// quits says whether the messages hold the program's quitting.
func quits(out []tea.Msg) bool {
	return slices.ContainsFunc(out, func(msg tea.Msg) bool { _, ok := msg.(tea.QuitMsg); return ok })
}

func TestCtrlCQuitsInterruptedCtrlQAndTheMenusQuitDoNot(t *testing.T) {
	m, _ := opened(t, listJSON)
	n, out := testkit.Drive(m, nil, testkit.Press("ctrl+c"))
	if !quits(out) || !n.Interrupted() || n.ctx.Err() == nil {
		t.Errorf("ctrl-c: quit %v, interrupted %v, kit stopped %v", quits(out), n.Interrupted(), n.ctx.Err())
	}
	m, _ = opened(t, listJSON)
	if n, out := testkit.Drive(m, nil, testkit.Press("ctrl+q")); !quits(out) || n.Interrupted() {
		t.Errorf("ctrl-q: quit %v, interrupted %v", quits(out), n.Interrupted())
	}
	m, _ = opened(t, listJSON)
	m, _ = drive(m, testkit.Press("="))
	x, y := lineAt(t, m, "Quit")
	if n, out := testkit.Drive(m, nil, left(x, y)); !quits(out) || n.Interrupted() {
		t.Errorf("the menu's Quit: quit %v, interrupted %v", quits(out), n.Interrupted())
	}
}

func TestCloseStopsWhatKitStillRuns(t *testing.T) {
	m, _ := opened(t, listJSON)
	m.Close()
	if got := m.read()().(listedMsg); !errors.Is(got.err, context.Canceled) {
		t.Errorf("a read after Close: %v", got.err)
	}
}

// The app's menu drops down on = and takes the keyboard: esc closes it doing nothing, enter on its first line asks
// kittui for the welcome, an action's key does the action, and a click on another screen asks kittui for it.
func TestTheAppsMenuTakesItsKeysAndSwitchesScreens(t *testing.T) {
	m, _ := opened(t, listJSON)
	m, _ = drive(m, testkit.Press("="))
	if !m.appMenu.Open || !strings.Contains(view(m), "esc close") {
		t.Fatalf("= opened no menu:\n%s", view(m))
	}
	if m, _ = drive(m, testkit.Press("esc")); m.appMenu.Open || m.prompt.open {
		t.Errorf("esc: menu open %v, dialog open %v", m.appMenu.Open, m.prompt.open)
	}
	m, asked := drive(m, testkit.Press("="), testkit.Press("enter"))
	if len(asked) != 1 || asked[0] != (shell.SwitchMsg{Screen: shell.Welcome}) {
		t.Errorf("enter on the menu's first line asked %#v", asked)
	}
	m, _ = drive(m, testkit.Press("="), testkit.Press("n"))
	if m.appMenu.Open || !m.prompt.open || !strings.Contains(view(m), "New workspace") {
		t.Errorf("n in the menu:\n%s", view(m))
	}
	m, _ = drive(m, testkit.Press("esc"), left(2, 0))
	x, y := lineAt(t, m, "Help")
	if _, asked := drive(m, left(x, y)); len(asked) != 1 || asked[0] != (shell.SwitchMsg{Screen: shell.Help}) {
		t.Errorf("a click on Help asked %#v", asked)
	}
}

// The workspace menu, after the screen's name, drops down on - or a plain click, and picks a workspace as on every
// screen.
func TestTheWorkspaceMenuPicksAGroupByKeyAndByClick(t *testing.T) {
	m, _ := opened(t, listJSON)
	here := shell.WorkspaceMsg{Workspace: kit.Workspace{Kind: "repository", Name: "github.com/o/a",
		Member: "github.com/o/a", Groups: []kit.WorkspaceGroup{{Name: "w", Members: 2}}}}
	m, _ = drive(m, here, testkit.Press("-"))
	if !m.books.Open || !strings.Contains(view(m), shell.Books+" w 2") {
		t.Fatalf("- dropped down no workspace menu:\n%s", view(m))
	}
	if _, asked := drive(m, testkit.Press("enter")); len(asked) != 1 || asked[0] != (shell.PickMsg{Workspace: "w"}) {
		t.Errorf("enter on w asked %#v", asked)
	}
	m, _ = drive(m, testkit.Press("esc"))
	x, y := lineAt(t, m, "· -")
	if m, _ = drive(m, left(x+2, y)); !m.books.Open {
		t.Fatalf("a click on - dropped down nothing:\n%s", view(m))
	}
	x, y = lineAt(t, m, shell.Books+" w 2")
	if _, asked := drive(m, left(x, y)); len(asked) != 1 || asked[0] != (shell.PickMsg{Workspace: "w"}) {
		t.Errorf("a click on w asked %#v", asked)
	}
}

func TestKitsLineStandsWhenItListsNoGroup(t *testing.T) {
	m, _ := opened(t, listJSON)
	if err := os.Remove(filepath.Join(os.Getenv("FAKE_WORKSPACES"), "list.json")); err != nil {
		t.Fatal(err)
	}
	m, _ = drive(m, m.read()())
	if v := view(m); !strings.Contains(v, Failed+" cat: ") || !strings.Contains(v, "list.json") {
		t.Errorf("kit's line:\n%s", v)
	}
	// a kit that fails and says nothing is said to have failed
	silent, err := exec.LookPath("false")
	if err != nil {
		t.Skip("no false here")
	}
	m = Open(kit.Runner{Kit: silent, Dir: t.TempDir()})
	m, _ = drive(m, testkit.Start(m)...)
	if v := view(m); !strings.Contains(v, Failed+" kit workspaces failed") {
		t.Errorf("a silent kit:\n%s", v)
	}
}

func TestTheScreenSaysItReadsTheGroupsUntilKitAnswers(t *testing.T) {
	if v := view(Open(testkit.Runner(t, "kit"))); !strings.Contains(v, "reading the groups…") {
		t.Errorf("before kit answers:\n%s", v)
	}
}

// After an action, the groups are read again; an answer to an earlier read is dropped.
func TestAnEarlierReadsAnswerIsDropped(t *testing.T) {
	m, _ := opened(t, listJSON)
	earlier := m.read()
	m, _ = drive(m, testkit.Press("f"))
	none := `{"file":"/f","roots":[],"favorites":[],"workspaces":[]}`
	if err := os.WriteFile(filepath.Join(os.Getenv("FAKE_WORKSPACES"), "list.json"), []byte(none), 0o644); err != nil {
		t.Fatal(err)
	}
	if m, _ = drive(m, earlier()); strings.Contains(view(m), "no group yet") || !strings.Contains(view(m), "solo") {
		t.Errorf("an earlier read landed:\n%s", view(m))
	}
}

// Favorite asks no question: kit's line, when it refuses, stands in place of the keys until a key; what kit did, when
// it does, stands there until it fades.
func TestFavoriteSaysKitsLineInPlaceOfTheKeys(t *testing.T) {
	m, _ := opened(t, listJSON)
	t.Setenv("FAKE_KIT_REFUSE", "workspaces-edit")
	m, _ = drive(m, testkit.Press("f"))
	last := func(m Model) string { lines := strings.Split(view(m), "\n"); return lines[len(lines)-1] }
	if !m.Noticing() || !strings.Contains(last(m), Failed+" kit workspaces refuses the edit") {
		t.Errorf("the refusal: noticing %v, %q", m.Noticing(), last(m))
	}
	if slices.Contains(m.menu().Dim, Favorite) {
		t.Error("the actions stay dim after kit refused")
	}
	if m, _ = drive(m, testkit.Press("down")); m.Noticing() || !strings.Contains(last(m), "= menu") {
		t.Errorf("a key leaves %q", last(m))
	}
	t.Setenv("FAKE_KIT_REFUSE", "")
	m, out := testkit.Drive(m, func(msg tea.Msg) bool { _, faded := msg.(shell.FadedMsg); return faded || asks(msg) },
		testkit.Press("f"))
	if !strings.Contains(last(m), "the fake did it: --favorite=solo") {
		t.Errorf("what kit did: %q", last(m))
	}
	k := slices.IndexFunc(out, func(msg tea.Msg) bool { _, faded := msg.(shell.FadedMsg); return faded })
	if k < 0 {
		t.Fatalf("no fade: %#v", out)
	}
	if m, _ = drive(m, out[k]); m.Noticing() {
		t.Errorf("faded, it stands: %q", last(m))
	}
}

// A dialog asks kit nothing while what is typed makes no command, and nothing more while kit runs what it asked.
func TestADialogAsksOnceAndOnlyForACommand(t *testing.T) {
	m, record := opened(t, listJSON)
	m, _ = drive(m, testkit.Press("n"), testkit.Press("enter"))
	if !m.prompt.open || m.prompt.busy || recorded(t, record) != "" {
		t.Errorf("enter with nothing typed: open %v, busy %v, %q", m.prompt.open, m.prompt.busy, recorded(t, record))
	}
	m, _ = drive(m, testkit.Typed("tools")...)
	next, cmd := m.Update(testkit.Press("enter"))
	if m = next.(Model); cmd == nil || !m.prompt.busy {
		t.Fatalf("enter: busy %v", m.prompt.busy)
	}
	if v := view(m); !strings.Contains(v, "New workspace") || !slices.Contains(m.menu().Dim, New) {
		t.Errorf("while kit runs, the dialog stays and the actions are dim:\n%s", v)
	}
	if _, again := m.Update(testkit.Press("enter")); again != nil {
		t.Error("a second enter while kit runs asked again")
	}
	// an owner takes a pattern at most
	m, _ = opened(t, listJSON)
	m, _ = drive(m, testkit.Press("o"))
	m, _ = drive(m, testkit.Typed("empowerite homebrew-* more")...)
	if _, ok := m.prompt.makes(); ok {
		t.Error("an owner and two patterns make a command")
	}
}

// A dialog takes plain clicks: one off the box closes it, one on its edge or its words does nothing, and a right
// click nothing either.
func TestADialogClosesOnAClickOffItsBox(t *testing.T) {
	m, _ := opened(t, listJSON)
	m, _ = drive(m, testkit.Press("r"))
	x, y := lineAt(t, m, "Rename 📚 w")
	if m, _ = drive(m, left(x, y)); !m.prompt.open {
		t.Error("a click on the edge closed the dialog")
	}
	x, y = lineAt(t, m, "its new name")
	if m, _ = drive(m, left(x, y), tea.MouseClickMsg{X: 0, Y: 25, Button: tea.MouseRight}); !m.prompt.open {
		t.Error("a click on its words, or a right click, closed the dialog")
	}
	if m, _ = drive(m, left(0, 25)); m.prompt.open {
		t.Error("a click off the box leaves the dialog open")
	}
	// a right click on the screen does nothing either
	if m, _ = drive(m, tea.MouseClickMsg{X: 2, Y: 0, Button: tea.MouseRight}); m.appMenu.Open {
		t.Error("a right click on = dropped the menu down")
	}
}

// The rows take the keys the screen does not: down moves the cursor, a click on a row puts it there, and the actions
// then act on that row; once esc took the keyboard from the rows, down gives it back.
func TestTheRowsTakeTheArrowsAndAClick(t *testing.T) {
	m, record := opened(t, listJSON)
	if m, _ = drive(m, testkit.Press("down")); func() bool { g, _ := m.current(); return g.Name != "solo" }() {
		t.Fatalf("down: cursor %d", m.sheet.Table().Cursor)
	}
	m, _ = drive(m, testkit.Press("up"))
	x, y := lineAt(t, m, "github.com/me/solo")
	m, _ = drive(m, left(x, y), testkit.Press("f"))
	if got := recorded(t, record); !strings.Contains(got, "workspaces --favorite=solo") {
		t.Errorf("f after a click on solo asked %q", got)
	}
	m, _ = drive(m, testkit.Press("esc"))
	if m.sheet.Focused() {
		t.Fatal("esc leaves the rows the keyboard")
	}
	if m, _ = drive(m, testkit.Press("down")); !m.sheet.Focused() {
		t.Error("down did not give the rows the keyboard again")
	}
}

func TestAClickOnTheSearchGivesItTheKeyboard(t *testing.T) {
	m, _ := opened(t, listJSON)
	x, y := lineAt(t, m, "/search")
	m, _ = drive(m, left(x, y))
	m, _ = drive(m, testkit.Typed("solo")...)
	if v := view(m); !m.search.Focused || strings.Contains(v, "github.com/o/a") {
		t.Errorf("the search: focused %v\n%s", m.search.Focused, v)
	}
}

// With kittui's Auto-group on, the ⭐ every group shares is a heading, where the cursor starts: no group is under it,
// so the actions on one are dim, and r does nothing, until down puts a group under it.
func TestOnAHeadingOnlyNewActs(t *testing.T) {
	m, _ := opened(t, bothJSON)
	m, _ = drive(m, shell.SettingsMsg{Settings: shell.Settings{KeysBar: true, FoldShared: true}})
	if _, on := m.current(); on || !slices.Contains(m.menu().Dim, Rename) || slices.Contains(m.menu().Dim, New) {
		t.Fatalf("on the heading: on a group %v, dim %q\n%s", on, m.menu().Dim, view(m))
	}
	if m, _ = drive(m, testkit.Press("r")); m.prompt.open {
		t.Error("r on a heading opened a dialog")
	}
	if strings.Contains(m.keys().Line(120), "rename") {
		t.Errorf("the keys bar offers rename on a heading: %q", m.keys().Line(120))
	}
	m, _ = drive(m, testkit.Press("down"))
	if g, on := m.current(); !on || g.Name != "w" || !strings.Contains(m.keys().Line(120), "rename") {
		t.Errorf("down: %+v, %v, %q", g, on, m.keys().Line(120))
	}
}

// A click on a column's name drops its menu down, which then takes the keys, and the keys bar names its own alone; esc
// closes it.
func TestAColumnsMenuTakesTheKeysWhileOpen(t *testing.T) {
	m, _ := opened(t, listJSON)
	x, y := lineAt(t, m, "workspace  #")
	m, _ = drive(m, left(x+1, y))
	if !m.sheet.Layered() || strings.Contains(m.keys().Line(120), "n new") {
		t.Fatalf("a click on workspace: layered %v, %q\n%s", m.sheet.Layered(), m.keys().Line(120), view(m))
	}
	if m, _ = drive(m, testkit.Press("n")); m.prompt.open {
		t.Error("n reached the screen past the column's menu")
	}
	if m, _ = drive(m, testkit.Press("esc")); m.sheet.Layered() {
		t.Error("esc leaves the column's menu open")
	}
}

// The mouse wheel scrolls the rows, the cursor left where it is; a turn on the menu bar, or while a menu or a dialog is
// open, scrolls nothing.
func TestTheWheelScrollsTheRowsAndLeavesTheCursor(t *testing.T) {
	var groups []string
	for i := range 40 {
		groups = append(groups, fmt.Sprintf(`{"name":"g%02d","members":1,"repos":["github.com/o/r%02d"],"favorite":false}`, i, i))
	}
	m, _ := opened(t, `{"file":"/c/kit/workspaces.yml","roots":["/code"],"favorites":[],"workspaces":[`+
		strings.Join(groups, ",")+`]}`)
	m, _ = drive(m, tea.WindowSizeMsg{Width: 120, Height: 14})
	_, y := lineAt(t, m, "g02")
	turn := func(y int, down bool) tea.MouseWheelMsg {
		if down {
			return tea.MouseWheelMsg{X: 5, Y: y, Button: tea.MouseWheelDown}
		}
		return tea.MouseWheelMsg{X: 5, Y: y, Button: tea.MouseWheelUp}
	}
	cursor := m.sheet.Table().Cursor
	n, _ := drive(m, turn(y, true))
	if n.sheet.Table().Offset == 0 || n.sheet.Table().Cursor != cursor {
		t.Fatalf("a turn down: offset %d, cursor %d, want %d\n%s", n.sheet.Table().Offset, n.sheet.Table().Cursor, cursor,
			view(n))
	}
	if n, _ = drive(n, turn(y, false)); n.sheet.Table().Offset != 0 {
		t.Errorf("a turn back up: offset %d", n.sheet.Table().Offset)
	}
	for name, before := range map[string][]tea.Msg{
		"on the menu bar":   nil,
		"the app menu open": {testkit.Typed("=")[0]},
		"a dialog open":     {testkit.Press("n")},
	} {
		at := y
		if before == nil {
			at = 0
		}
		if n, _ := drive(m, append(before, turn(at, true))...); n.sheet.Table().Offset != 0 {
			t.Errorf("%s: a turn scrolled to %d", name, n.sheet.Table().Offset)
		}
	}
}

// The app's menu keeps New Workspace…, Switch to It, and Favorite; Workspace… drops the group's menu down from its own
// line with the rest. Esc goes back to the app's menu, = closes both, a key chooses, and a click off the group's menu
// lands on the app's menu around it.
func TestWorkspaceDropsTheGroupsMenuDownFromItsLine(t *testing.T) {
	m, _ := opened(t, listJSON)
	m, _ = drive(m, testkit.Press("="))
	for _, name := range []string{Rename, Add, Owner, Command, Remove, DeleteOne} {
		if strings.Contains(view(m), name) {
			t.Errorf("the app's menu lists %q:\n%s", name, view(m))
		}
	}
	x, y := lineAt(t, m, Folder+" "+Workspace)
	m, _ = drive(m, left(x, y))
	if !m.group.Open || !m.appMenu.Open {
		t.Fatalf("Workspace… opened group %v, app %v", m.group.Open, m.appMenu.Open)
	}
	// its top edge is drawn on Workspace…'s line, its 📂 where that line's was, its first line under it
	if gx, gy := lineAt(t, m, Folder); gx != x || gy != y {
		t.Errorf("the group's menu hangs at %d,%d, Workspace… at %d,%d:\n%s", gx, gy, x, y, view(m))
	}
	if _, ry := lineAt(t, m, Rename); ry != y+1 {
		t.Errorf("Rename… on line %d:\n%s", ry, view(m))
	}
	if m, _ = drive(m, testkit.Press("esc")); m.group.Open || !m.appMenu.Open {
		t.Errorf("esc: group %v, app %v", m.group.Open, m.appMenu.Open)
	}
	m, _ = drive(m, left(x, y), testkit.Press("="))
	if m.group.Open || m.appMenu.Open {
		t.Errorf("=: group %v, app %v", m.group.Open, m.appMenu.Open)
	}
	m, _ = drive(m, testkit.Press("="), left(x, y), testkit.Press("x"))
	if m.group.Open || m.appMenu.Open || !strings.Contains(view(m), "Remove from") {
		t.Errorf("x in the group's menu:\n%s", view(m))
	}
	m, _ = drive(m, testkit.Press("esc"), testkit.Press("="), left(x, y))
	hx, hy := lineAt(t, m, "Help")
	if _, asked := drive(m, left(hx, hy)); len(asked) != 1 || asked[0] != (shell.SwitchMsg{Screen: shell.Help}) {
		t.Errorf("a click on Help past the group's menu asked %#v", asked)
	}
}
