// Over a group, Welcome reads the checks once, over the group, as kit judges each member's desk and worktrees itself:
// the setup step draws them as one table, the machine first, then a row per folder of each repository, and a folder's
// fix runs in its folder. The members not cloned are named at the end of the setup step.
package welcome

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// setupStep is the index of the setup step among the known ones.
var setupStep = indexOf(known, setupID)

// group opens the screen at 120×30 over the group w of github.com/o/a and github.com/o/b, whose checks are
// testdata/folders' as kit judges every folder of the group, and github.com/o/c, not cloned. It gives the screen, what
// its fixes ran, and the file the fake writes its welcome calls to.
func group(t *testing.T, msgs ...tea.Msg) (Model, *ran, string) {
	t.Helper()
	m, r := fake(t)
	dir, calls := t.TempDir(), filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_WELCOME_CALLS", calls)
	fixture, err := os.ReadFile(filepath.Join("testdata", "folders", "checks.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "checks.json"), fixture, 0o644); err != nil {
		t.Fatal(err)
	}
	ws := kit.Workspace{Kind: "group", Name: "w", Members: []kit.WorkspaceMember{{Repository: "github.com/o/a",
		Checkout: "/code/a"}, {Repository: "github.com/o/b", Checkout: "/code/b"}, {Repository: "github.com/o/c"}}}
	runner := testkit.Runner(t, "welcome")
	runner.Dir = dir
	g := Open(runner, ws)
	g.exec = m.exec
	return drive(g, append(testkit.Start(g, at120), msgs...)...), r, calls
}

// A group's checks are read once, over the group, which the title names; the setup step ends with a blank line, then
// the members not cloned, and no other step names them.
func TestAGroupsChecksAreReadOnceOverTheGroupWithTheUnclonedNamed(t *testing.T) {
	m, _, path := group(t)
	if got := calls(t, path); len(got) != 1 || got[0] != "--workspace=w welcome --checks --json --progress" {
		t.Errorf("the reads: %q", got)
	}
	if v := view(m); strings.Contains(v, "not cloned") || !strings.Contains(v, "- w 3") {
		t.Errorf("the introduction:\n%s", v)
	}
	if v := view(drive(m, stepKey(chains))); strings.Contains(v, "not cloned") {
		t.Errorf("the toolchains name the members not cloned:\n%s", v)
	}
	// a screen tall enough for every line of the step
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 60}, stepKey(setupStep))
	lines := m.lines()
	if n := len(lines); n < 2 || lines[n-2].text != "" || ansi.Strip(lines[n-1].text) != "   not cloned: o/c" ||
		!strings.Contains(view(m), "\n   not cloned: o/c\n") {
		t.Errorf("setup does not end with the members not cloned:\n%s", view(m))
	}
}

// Where kit check finds more than one problem, setup draws each under the check's line, a line each, and a search for
// what one of them says finds the check.
func TestSetupNamesEachProblemKitCheckFinds(t *testing.T) {
	m, _, _ := group(t, tea.WindowSizeMsg{Width: 120, Height: 60}, stepKey(setupStep))
	// as far in as the check's text, as its fix is
	want := "       ❌ kit check finds 2 problems\n" +
		"          .gitignore is missing; run: cd /code/b && kit update\n" +
		"          .kit/1st.md is missing; run: cd /code/b && kit hydrate\n"
	if v := view(m); !strings.Contains(v, want) {
		t.Errorf("no finding under kit check's line:\n%s", v)
	}
	m = drive(m, typed('/'))
	m = drive(m, testkit.Typed("1st.md")...)
	if v := view(m); !strings.Contains(v, "kit check finds 2 problems") || strings.Contains(v, "shellcheck") {
		t.Errorf("a search for a finding:\n%s", v)
	}
}

func TestTheSetupStepIsOneTableTheMachineFirstThenARowPerFolder(t *testing.T) {
	m, _, _ := group(t, stepKey(setupStep))
	v := view(m)
	testkit.Golden(t, filepath.Join("testdata", "setup-folders.golden"), v)
	// the page opens with its title line, as every step's does: its name, a dash, and what it holds
	if !strings.Contains(v, "   Setup — kit checks this machine, then each folder\n\n") {
		t.Errorf("no title line:\n%s", v)
	}
	machine, a, desk, topic := strings.Index(v, "This machine"), strings.Index(v, "github.com/o/a"),
		strings.Index(v, deskGlyph+" a "), strings.Index(v, worktreeGlyph+" 7-the-topic")
	if machine < 0 || a < machine || desk < a || topic < desk || strings.Index(v, "github.com/o/b") < topic {
		t.Errorf("the machine, then a's desk and worktree, then b:\n%s", v)
	}
	// a folder's failed checks and its offers stand under its row; the ones that hold are its row's ✅ alone
	if !strings.Contains(v, "       "+Offered+" kit is not installed here; it runs on what it infers\n"+
		"          cd /code/a/worktrees/7-the-topic && kit install\n") ||
		strings.Contains(v, "kit check finds nothing wrong") {
		t.Errorf("the folders' checks:\n%s", v)
	}
	if !strings.Contains(v, "7-the-topic") || !strings.Contains(v, "0.3.0") || !strings.Contains(v, "none") {
		t.Errorf("each folder's branch and kit:\n%s", v)
	}
	// the bar counts the failed checks: the login, and four of the folders'; an offer is no failure
	if !strings.Contains(v, "setup  5") {
		t.Errorf("the count:\n%s", v)
	}
}

// onKey moves the cursor, in the pane, down to the check with the key, and says whether it found it.
func onKey(m Model, key string) (Model, bool) {
	for range 12 {
		if m.inPane && m.checks[m.cursor].Key() == key {
			return m, true
		}
		m = drive(m, down)
	}
	return m, false
}

func TestAFoldersFailedCheckOffersItsFixUnderItsRow(t *testing.T) {
	m, _, _ := group(t, stepKey(setupStep))
	m, found := onKey(m, "/code/a hooks-installed")
	if !found {
		t.Fatalf("no hooks-installed at the desk:\n%s", view(m))
	}
	v := view(m)
	testkit.Golden(t, filepath.Join("testdata", "setup-folders-row.golden"), v)
	if !strings.Contains(v, "     ▸ ❌ the git hooks are not installed; lefthook runs none   ⧉   ▶️") ||
		keysAt(m) != choiceKeys {
		t.Errorf("the desk's hooks, offering ⧉ and ▶️:\n%s\n%s", v, keysAt(m))
	}
	// a folder's fix is shown and copied with the cd to its folder, so it runs wherever it is pasted
	if _, msgs := testkit.Drive(m, nil, press('c')); clipboard(msgs) != "cd /code/a && lefthook install" {
		t.Errorf("c copied %q", clipboard(msgs))
	}
}

// A search on the setup step keeps a folder's row where one of its checks holds what is typed, and drops the
// repository none of whose folders it keeps; where it keeps nothing the step says so, or checking… while the checks
// run again.
func TestTheSetupStepsSearchKeepsTheFoldersOfTheChecksItFinds(t *testing.T) {
	m, _, _ := group(t, stepKey(setupStep), press('/'))
	m = drive(m, testkit.Typed("hooks")...)
	v := view(m)
	if !strings.Contains(v, deskGlyph+" a ") || !strings.Contains(v, "the git hooks are not installed") ||
		strings.Contains(v, "7-the-topic") || strings.Contains(v, "github.com/o/b") || strings.Contains(v, "This machine") {
		t.Errorf("searched for hooks:\n%s", v)
	}
	m = drive(m, testkit.Typed("zzz")...)
	if v := view(m); !strings.Contains(v, "no check matches the search") || strings.Contains(v, "github.com/o/a") {
		t.Errorf("searched for what no check holds:\n%s", v)
	}
	m.checking.Again(m.env(), false)
	if v := view(m); !strings.Contains(v, "checking…") || strings.Contains(v, "no check matches the search") {
		t.Errorf("searched for what no check holds, the checks running again:\n%s", v)
	}
}

func TestAFoldersFixRunsInItsFolderAndAMachinesOverTheGroup(t *testing.T) {
	m, r, _ := group(t, stepKey(setupStep))
	m, found := onKey(m, "/code/a/worktrees/7-the-topic kit-installed")
	if !found {
		t.Fatalf("no kit-installed in the worktree:\n%s", view(m))
	}
	m = drive(m, press('r'))
	if len(r.calls) != 1 ||
		strings.Join(r.calls[0], " ") != "--workspace=w welcome --fix kit-installed --folder=/code/a/worktrees/7-the-topic" {
		t.Errorf("the fix: %q", r.calls)
	}
	m, found = onKey(drive(m, press('3'), down), "forge-login-github.com")
	if !found {
		t.Fatalf("no login:\n%s", view(m))
	}
	drive(m, press('r'))
	if len(r.calls) != 2 || strings.Join(r.calls[1], " ") != "--workspace=w welcome --fix forge-login-github.com" {
		t.Errorf("the login's fix: %q", r.calls)
	}
}

func TestAFoldersOfferOffersCopyAndRunAsAFailedCheckDoes(t *testing.T) {
	m, r, _ := group(t, stepKey(setupStep))
	m, found := onKey(m, "/code/a/worktrees/7-the-topic kit-installed")
	if !found {
		t.Fatalf("no kit-installed in the worktree:\n%s", view(m))
	}
	if v := view(m); !strings.Contains(v, "     ▸ "+Offered+" kit is not installed here; it runs on what it infers   ⧉   ▶️") ||
		keysAt(m) != choiceKeys {
		t.Errorf("the worktree's offer, offering ⧉ and ▶️:\n%s\n%s", v, keysAt(m))
	}
	// r runs it, as a failed check's, in its folder
	if drive(m, press('r')); len(r.calls) != 1 ||
		strings.Join(r.calls[0], " ") != "--workspace=w welcome --fix kit-installed --folder=/code/a/worktrees/7-the-topic" {
		t.Errorf("the offer's fix: %q", r.calls)
	}
}

// Over a group, kit's refusal of the checks shows on each step with checks, as over a repository.
func TestAGroupKitRefusesSaysKitsLine(t *testing.T) {
	t.Setenv("FAKE_WELCOME_REFUSE", "1")
	m, _, _ := group(t, stepKey(setupStep))
	if v := view(m); !strings.Contains(v, "kit welcome refuses for the test") {
		t.Errorf("kit refused:\n%s", v)
	}
}

func TestAGroupsChecksEndWithTheScreen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ws := kit.Workspace{Kind: "group", Name: "w", Members: []kit.WorkspaceMember{{Repository: "github.com/o/a",
		Checkout: t.TempDir()}}}
	read := checksOver(testkit.Runner(t, "welcome"), ws)
	if _, err := read(ctx, func(kit.Progress) {}); !errors.Is(err, context.Canceled) {
		t.Errorf("ended: %v", err)
	}
}

// The setup step's search keeps a folder whose branch holds it, with its failed checks, one whose check's line does,
// and one whose path does; a search nothing holds says so.
func TestTheSetupStepsSearchNamesTheFolder(t *testing.T) {
	search := func(text string) Model {
		m, _, _ := group(t, append([]tea.Msg{stepKey(setupStep), tea.KeyPressMsg{Code: '/', Text: "/"}},
			testkit.Typed(text)...)...)
		return m
	}
	if v := view(search("7-the")); !strings.Contains(v, "kit is not installed here; it runs on what it infers") ||
		strings.Contains(v, "several toolchains") {
		t.Errorf("searched for the branch:\n%s", v)
	}
	if v := view(search("lefthook runs none")); !strings.Contains(v, "the git hooks are not installed") ||
		strings.Contains(v, "kit is not installed here") {
		t.Errorf("searched for a check's line:\n%s", v)
	}
	if v := view(search("code/b")); !strings.Contains(v, "several toolchains") || strings.Contains(v, "lefthook runs none") {
		t.Errorf("searched for a folder:\n%s", v)
	}
	if v := view(search("zzz")); !strings.Contains(v, "no check matches the search") {
		t.Errorf("searched for nothing there:\n%s", v)
	}
	// a search nothing holds, while the checks run again, says they run
	m := search("zzz")
	m.checking.Again(m.env(), false)
	if v := view(m); !strings.Contains(v, "checking…") {
		t.Errorf("checking under a search nothing holds:\n%s", v)
	}
}
