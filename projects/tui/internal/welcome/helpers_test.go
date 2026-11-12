// The screen is driven key by key through Update, against testdata/fake-kit, which answers the checks from
// testdata/checks.json and the pages from testdata/release, as Bubble Tea would drive it.
package welcome

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// at120 is the design size.
var at120 = tea.WindowSizeMsg{Width: 120, Height: 30}

func press(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

var (
	up      = press(tea.KeyUp)
	down    = press(tea.KeyDown)
	backTab = tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	tabKey  = press(tea.KeyTab)
	enter   = press(tea.KeyEnter)
	esc     = press(tea.KeyEscape)
	pgdown  = press(tea.KeyPgDown)
	pgup    = press(tea.KeyPgUp)
)

// ran is what a fix handed the terminal ran: its words after the kit, one call each, and the directory each ran in.
type ran struct {
	calls [][]string
	dirs  []string
}

// fake gives the screen over testdata/fake-kit, with its state in a file of its own, so a fixed check stays fixed for
// this test alone. A fix runs at once, its words kept in ran, its output dropped, as the terminal would take it.
func fake(t *testing.T) (Model, *ran) {
	t.Helper()
	t.Setenv("FAKE_WELCOME_STATE", filepath.Join(t.TempDir(), "state"))
	m := New(testkit.Runner(t, "welcome"), "empowerite/kit")
	r := &ran{}
	m.exec = func(cmd *exec.Cmd, done tea.ExecCallback) tea.Cmd {
		r.calls = append(r.calls, cmd.Args[1:])
		r.dirs = append(r.dirs, cmd.Dir)
		return func() tea.Msg { return done(cmd.Run()) }
	}
	return m, r
}

// loaded opens the screen at 120×30, runs Init, then sends the messages.
func loaded(t *testing.T, msgs ...tea.Msg) (Model, *ran) {
	t.Helper()
	m, r := fake(t)
	return drive(m, append(testkit.Start(m, at120), msgs...)...), r
}

// atChains opens the screen as loaded does, then follows the introduction's first failed check, jq, the first under
// the toolchains, as a newcomer would: down to the table's control line, its first group's heading, and its first row,
// then enter. Then it sends the messages.
func atChains(t *testing.T, msgs ...tea.Msg) (Model, *ran) {
	t.Helper()
	return loaded(t, append([]tea.Msg{down, down, down, enter}, msgs...)...)
}

// repoWith makes a git repository with the pages under .kit/welcome, and gives the fake runner in it.
func repoWith(t *testing.T, pages map[string]string) kit.Runner {
	t.Helper()
	dir := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for name, text := range pages {
		path := filepath.Join(dir, pagesDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := testkit.Runner(t, "welcome")
	r.Dir = dir
	return r
}

// house opens the screen at 120×30 in a repository whose .kit is testdata/house's, as a house package composes it:
// the kit package's pages linked, 5-terms.md replaced, and 7-our-team.md added, a plain page long enough to scroll. Then it sends the messages.
func house(t *testing.T, msgs ...tea.Msg) (Model, *ran) {
	t.Helper()
	m, r := fake(t)
	m.runner = repoWith(t, nil)
	fixture, err := filepath.Abs(filepath.Join("testdata", "house", ".kit"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(fixture, filepath.Join(m.runner.Dir, ".kit")); err != nil {
		t.Fatal(err)
	}
	return drive(m, append(testkit.Start(m, at120), msgs...)...), r
}

// drive sends the messages, running every command each update returns and feeding its messages back.
func drive(m Model, msgs ...tea.Msg) Model {
	for _, msg := range msgs {
		if c, ok := msg.(clickMsg); ok {
			msg = c.at(m)
		}
		m, _ = testkit.Drive(m, nil, msg)
	}
	return m
}

// clickMsg stands, in drive, for a click on a button of the row that holds text, read off the screen as drawn when the
// click comes: the button's first cell past the space before it.
type clickMsg struct{ text, button string }

// at gives the click on the screen as m draws it.
func (c clickMsg) at(m Model) tea.Msg {
	for y, l := range strings.Split(view(m), "\n") {
		if !strings.Contains(l, c.text) {
			continue
		}
		if i := strings.LastIndex(l, " "+c.button+" "); i >= 0 {
			return testkit.Click(ansi.StringWidth(l[:i])+1, y)
		}
	}
	panic("no " + c.button + " on the row of " + c.text + ":\n" + view(m))
}

// view gives the screen without its escapes.
func view(m Model) string { return ansi.Strip(m.View().Content) }

// calls reads the welcome calls the fake recorded.
func calls(t *testing.T, path string) []string {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(got)), "\n")
}

// The keys bar in each place the keyboard can be, as the path to it draws it, STEP for the step's name, which esc names
// as it gives the bar the keyboard: on the bar over a step with checks or the introduction with failed checks, and over
// a page; on a check's row that offers no fix, on a row whose fix kit runs, on a row whose fix kit only shows;
// and in a page. jump names the number keys of the known steps.
var (
	jump       = "1-" + strconv.Itoa(len(known)) + " jump"
	stepKeys   = "= menu - workspace / search │ ⇥·⇧⇥ tab ↓ checks " + jump + " : tab menu"
	pageKeys   = "= menu - workspace / search │ ⇥·⇧⇥ tab ↓ page " + jump + " : tab menu"
	paneKeys   = "= menu - workspace / search │ " + jump + " : tab menu │ esc tabs ↑↓ move ⇞⇟ page"
	rowKeys    = paneKeys
	choiceKeys = paneKeys + " c copy r run"
	showKeys   = paneKeys + " c copy"
	scrollKeys = "= menu - workspace / search │ " + jump + " : tab menu │ esc tabs ↑↓ scroll ⇞⇟ page"
)

// stepKey gives the press of a step's number, which selects it from anywhere: 1 for the first.
func stepKey(step int) tea.KeyPressMsg {
	r := rune('1' + step)
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// inHouse gives the keys bar as it reads in testdata/house, whose added page makes a step more to jump to.
func inHouse(keys string) string {
	return strings.Replace(keys, jump, "1-"+strconv.Itoa(len(known)+1)+" jump", 1)
}

// ourTeam is the step of testdata/house's added page, 7-our-team.md, after the kit package's: a plain page, long
// enough to scroll.
var ourTeam = len(known)

// keysAt gives the keys bar where the keyboard is, as the path to it draws it.
func keysAt(m Model) string { return m.keys().String() }

// rowRuns says whether r runs a fix where the keyboard is: on a row of the introduction's table whose fix kit runs.
func rowRuns(m Model) bool {
	i := -1
	if m.introGridded() && m.inPane {
		i = m.checkUnder()
	}
	return i >= 0 && runnable(m.checks[i]) || m.runsHere()
}

// rowCopies says whether c copies a fix where the keyboard is: on a row of the introduction's table with a fix.
func rowCopies(m Model) bool {
	i := -1
	if m.introGridded() && m.inPane {
		i = m.checkUnder()
	}
	return i >= 0 && m.checks[i].Fix != "" || m.copies()
}

// where gives where the keyboard is in the pane: the check under the cursor, and the cursor of the introduction's
// table.
func where(m Model) string {
	at := strconv.Itoa(m.cursor)
	if m.introGridded() {
		at += " " + strconv.Itoa(m.sheet().Table().Cursor) + " " + strconv.FormatBool(m.sheet().InRows())
	}
	return at
}
