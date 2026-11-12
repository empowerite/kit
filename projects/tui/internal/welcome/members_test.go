// Over a group, Welcome reads the checks once, over the group, as kit judges each member's desk and worktrees itself:
// the setup step draws them as one table, the machine first, then a row per folder of each repository, and a folder's
// fix runs in its folder. The members not cloned are named in the introduction.
package welcome

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

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

func TestAGroupsChecksAreReadOnceOverTheGroupWithTheUnclonedNamed(t *testing.T) {
	m, _, path := group(t)
	if got := calls(t, path); len(got) != 1 || got[0] != "--workspace=w welcome --checks --json --progress" {
		t.Errorf("the reads: %q", got)
	}
	if v := view(m); !strings.Contains(v, "not cloned: o/c") || !strings.Contains(v, "- w 3") {
		t.Errorf("the introduction:\n%s", v)
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
	// a folder's failed checks stand under its row; the ones that hold are its row's ✅ alone
	if !strings.Contains(v, "kit.yml declares no kit version") || strings.Contains(v, "kit check finds nothing wrong") {
		t.Errorf("the folders' checks:\n%s", v)
	}
	if !strings.Contains(v, "7-the-topic") || !strings.Contains(v, "0.3.0") || !strings.Contains(v, "none") {
		t.Errorf("each folder's branch and kit:\n%s", v)
	}
	// the bar counts the failed checks: the login, and five of the folders'
	if !strings.Contains(v, "setup  6") {
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
	if !strings.Contains(v, "     ▸ ❌ the git hooks are not installed; lefthook runs none    now   later") ||
		keysAt(m) != choiceKeys {
		t.Errorf("the desk's hooks, offering now and later:\n%s\n%s", v, keysAt(m))
	}
}

func TestAFoldersFixRunsInItsFolderAndAMachinesOverTheGroup(t *testing.T) {
	m, r, _ := group(t, stepKey(setupStep))
	m, found := onKey(m, "/code/a/worktrees/7-the-topic kit-installed")
	if !found {
		t.Fatalf("no kit-installed in the worktree:\n%s", view(m))
	}
	m = drive(m, press('n'))
	if len(r.calls) != 1 ||
		strings.Join(r.calls[0], " ") != "--workspace=w welcome --fix kit-installed --folder=/code/a/worktrees/7-the-topic" {
		t.Errorf("the fix: %q", r.calls)
	}
	m, found = onKey(drive(m, press('3'), down), "forge-login-github.com")
	if !found {
		t.Fatalf("no login:\n%s", view(m))
	}
	drive(m, press('n'))
	if len(r.calls) != 2 || strings.Join(r.calls[1], " ") != "--workspace=w welcome --fix forge-login-github.com" {
		t.Errorf("the login's fix: %q", r.calls)
	}
}

func TestTheIntroductionNamesEachFolderAboveItsFailedChecks(t *testing.T) {
	m, _, _ := group(t)
	v := view(m)
	testkit.Golden(t, filepath.Join("testdata", "introduction-folders.golden"), v)
	head := strings.Index(v, "github.com/o/a · "+worktreeGlyph+" 7-the-topic on 7-the-topic")
	if head < 0 || !strings.Contains(v[head:], "\n     ❌ kit.yml declares no kit version") {
		t.Errorf("the worktree's failed check under its line:\n%s", v)
	}
	// a search for a folder's branch keeps that folder's checks
	m = drive(m, press('/'))
	m = drive(m, testkit.Typed("7-the")...)
	if v := view(m); !strings.Contains(v, "kit.yml declares no kit version") || strings.Contains(v, "several toolchains") {
		t.Errorf("searched for the branch:\n%s", v)
	}
}

func TestAGroupKitRefusesSaysKitsLine(t *testing.T) {
	t.Setenv("FAKE_WELCOME_REFUSE", "1")
	m, _, _ := group(t)
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
