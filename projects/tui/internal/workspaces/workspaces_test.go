// The workspaces' screen lists the groups kit keeps, a row each, and its actions run kit workspaces, each from the app's
// menu by a plain click or by its key, the ones that want an answer through a dialog that says kit's line when it
// refuses.
package workspaces

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

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
	if !strings.Contains(v, Rename) || !strings.Contains(v, "╰") || len(lines) != 30 {
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
	x, y := lineAt(t, m, Rename)
	m, _ = drive(m, left(x, y))
	if !m.prompt.open || m.prompt.typed != "w" {
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
