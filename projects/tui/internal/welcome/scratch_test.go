// The setup table's folders named so they tell apart, a scratch worktree saying it is one, and its buttons pressed by a
// click, as the rows of any step's checks are.
package welcome

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/fix"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// rowOfFolder gives a folder's row of checks, its path.
func rowOfFolder(path string) folderRow { return folderRow{head: kit.Check{Folder: path}} }

// Folders whose names are alike take their parents' names before them, as many as tell them apart; the rest keep
// their own.
func TestFolderNamesTellCollidingBasenamesApart(t *testing.T) {
	got := folderNames([]folderRow{rowOfFolder("/code/a"), rowOfFolder("/tmp/claude-1/x/scratchpad/tree"),
		rowOfFolder("/tmp/claude-2/y/scratchpad/tree"), rowOfFolder("/tmp/claude-1/z/r"),
		rowOfFolder("/tmp/claude-1/z/base-tree"), rowOfFolder("/code/a/worktrees/tree")})
	want := map[string]string{"/code/a": "a", "/tmp/claude-1/x/scratchpad/tree": "x/scratchpad/tree",
		"/tmp/claude-2/y/scratchpad/tree": "y/scratchpad/tree", "/tmp/claude-1/z/r": "r",
		"/tmp/claude-1/z/base-tree": "base-tree", "/code/a/worktrees/tree": "worktrees/tree"}
	for folder, name := range want {
		if got[folder] != name {
			t.Errorf("%s is named %q, not %q", folder, got[folder], name)
		}
	}
	// two folders that are the same path to the root name each other; the names stop at the whole path
	if same := folderNames([]folderRow{rowOfFolder("/a"), rowOfFolder("a")}); same["/a"] != "a" {
		t.Errorf("the same path twice: %v", same)
	}
}

// scratchTree adds to the group's checks a worktree of a's, in a temporary folder, with the check that it is one.
func scratchTree(m Model, folder string) Model {
	for _, c := range []kit.Check{
		{ID: "repo-git", Step: "setup", OK: true, Line: "this folder is a git repository", Runs: true, Folder: folder,
			Repository: "github.com/o/a", Branch: "scratch"},
		{ID: scratchID, Step: "setup", Line: "this worktree is in a temporary folder, outside /code/a: a scratch worktree, left behind",
			Fix: "git -C /code/a worktree remove " + folder, Runs: true, Folder: folder, Repository: "github.com/o/a",
			Branch: "scratch"},
	} {
		m.checks = append(m.checks, c)
	}
	m.followSheet()
	return m
}

// A worktree in a temporary folder outside its desk's tree is drawn with 🧹, the check that says so is in its git
// column, ❌, and the fix that removes it stands under its row; two of them under different parents are named by them.
func TestAScratchWorktreeSaysSoAndIsNamedByItsParents(t *testing.T) {
	m, _, _ := group(t, tea.WindowSizeMsg{Width: 120, Height: 60}, stepKey(setupStep))
	m = scratchTree(scratchTree(m, "/tmp/claude-1/x/scratchpad/tree"), "/tmp/claude-2/y/scratchpad/tree")
	v := view(m)
	for _, want := range []string{"🧹 x/scratchpad/tree", "🧹 y/scratchpad/tree", "a scratch worktree, left behind",
		"git -C /code/a worktree remove /tmp/claude-1/x/scratchpad/tree"} {
		if !strings.Contains(v, want) {
			t.Errorf("no %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "🌳 tree") {
		t.Errorf("a scratch worktree named as a checkout is:\n%s", v)
	}
}

// A click on ⧉ copies the fix of the row under the cursor, one on ▶️ runs it, and one on the row, off the buttons, only
// selects the row.
func TestAClickOnAFoldersButtonsCopiesAndRunsItsFix(t *testing.T) {
	m, r, _ := group(t, stepKey(setupStep))
	m, found := onKey(m, "/code/a hooks-installed")
	if !found {
		t.Fatalf("no hooks check:\n%s", view(m))
	}
	copied, msgs := testkit.Drive(m, nil, clickMsg{"lefthook install", fix.CopyName}.at(m))
	if clipboard(msgs) != "cd /code/a && lefthook install" || !strings.Contains(view(copied), "copied") {
		t.Errorf("the click on ⧉: %q\n%s", clipboard(msgs), view(copied))
	}
	testkit.Drive(m, nil, clickMsg{"lefthook install", fix.RunName}.at(m))
	if len(r.calls) != 1 || strings.Join(r.calls[0], " ") != "--workspace=w welcome --fix hooks-installed --folder=/code/a" {
		t.Errorf("the click on ▶️: %q", r.calls)
	}
	// a click on the row's name, off the buttons, selects the row and runs nothing
	for y, l := range strings.Split(view(m), "\n") {
		if strings.Contains(l, "7-the-topic  ") {
			if _, msgs := testkit.Drive(m, nil, testkit.Click(20, y)); clipboard(msgs) != "" || len(r.calls) != 1 {
				t.Errorf("a click on a row pressed a button")
			}
		}
	}
}

// A click on keep, on a terminal's failed finding in the table, keeps it, as the row's key does.
func TestAClickOnKeepKeepsATerminalFindingInTheTable(t *testing.T) {
	m, _, path := group(t, stepKey(setupStep))
	m.checks = append(m.checks, kit.Check{ID: "terminal-iterm-option", Step: "setup", Line: "iTerm sends ⌥ as a plain key",
		Runs: true})
	m.followSheet()
	m, found := onKey(m, "terminal-iterm-option")
	if !found || !m.keeping() || !strings.Contains(keysAt(m), "k keep") {
		t.Fatalf("no terminal finding to keep:\n%s\n%s", view(m), keysAt(m))
	}
	testkit.Drive(m, nil, clickMsg{"iTerm sends", keepName}.at(m))
	if got := strings.Join(calls(t, path), "\n"); !strings.Contains(got, "welcome --keep=terminal-iterm-option") {
		t.Errorf("calls %q", got)
	}
}

// When kit names no folder, the table is gone and the step is drawn as any step with checks.
func TestTheFoldersTableGoesWhereKitNamesNoFolder(t *testing.T) {
	m, _, _ := group(t, stepKey(setupStep))
	if !m.gridded() {
		t.Fatalf("no table over folders:\n%s", view(m))
	}
	m.checks = slices.DeleteFunc(m.checks, func(c kit.Check) bool { return c.Folder != "" })
	m.followSheet()
	if m.gridded() || !strings.Contains(view(m), "gh is not logged in") {
		t.Errorf("a table with no folder:\n%s", view(m))
	}
}

// A check of a kit that names no folder, and says what it found, draws each finding under its line, as before.
func TestAChecksFindingsAreDrawnUnderItsLineWhereKitNamesNoFolder(t *testing.T) {
	m, _ := loaded(t, stepKey(setupStep))
	m.checks = append(m.checks, kit.Check{ID: "kit-check", Step: "setup", Line: "kit check finds 1 problem",
		Findings: []string{".kit/1st.md is missing"}, Runs: true})
	if v := view(m); !strings.Contains(v, "kit check finds 1 problem") || !strings.Contains(v, ".kit/1st.md is missing") {
		t.Errorf("no finding under the check:\n%s", v)
	}
}

// A click on the cursor's row, off its buttons, presses none; a machine check that holds and offers a fix is an offer.
func TestAClickOnTheCursorsRowPressesNoButtonAndAMachineOfferIsOne(t *testing.T) {
	m, r, _ := group(t, stepKey(setupStep))
	m.checks = append(m.checks, kit.Check{ID: "forge-cli-x", Step: "setup", OK: true, Line: "x is on PATH", Fix: "x update",
		Runs: true})
	m.followSheet()
	m, found := onKey(m, "/code/a hooks-installed")
	if !found {
		t.Fatalf("no hooks check:\n%s", view(m))
	}
	if !strings.Contains(view(m), Offered+" x is on PATH") {
		t.Errorf("a machine offer is no offer:\n%s", view(m))
	}
	for y, l := range strings.Split(view(m), "\n") {
		if strings.Contains(l, deskGlyph+" a ") {
			if _, msgs := testkit.Drive(m, nil, testkit.Click(10, y)); clipboard(msgs) != "" || len(r.calls) != 0 {
				t.Errorf("a click on the cursor's row pressed a button")
			}
		}
	}
}

// A folder's check shown where no row of its folder is drawn above it names its repository and folder, the desk's or a
// worktree's glyph first.
func TestAFoldersCheckOutsideTheTableNamesItsFolder(t *testing.T) {
	m, _ := loaded(t, stepKey(chains))
	m.checks = append(m.checks,
		kit.Check{ID: "tool-x", Step: "toolchains", Line: "x is missing", Folder: "/code/a", Repository: "github.com/o/a", Desk: true},
		kit.Check{ID: "tool-x", Step: "toolchains", Line: "x is missing", Folder: "/code/a/w", Repository: "github.com/o/a"})
	if v := view(m); !strings.Contains(v, "a "+deskGlyph+" a") || !strings.Contains(v, "a "+worktreeGlyph+" w") {
		t.Errorf("no folder named after the checks:\n%s", v)
	}
}
