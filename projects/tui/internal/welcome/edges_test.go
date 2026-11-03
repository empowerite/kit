package welcome

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
)

// A page that cannot be read gives the path it was looked for at, and no markdown.
func TestAPageThatIsNotThereGivesThePathItWasLookedFor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "4-setup.md")
	if page, missing := readPage(path); page != "" || missing != path {
		t.Errorf("readPage = %q, %q", page, missing)
	}
}

// The screen opens with origin's owner/repo in its title, read from git where the command was typed, and without
// it in a repository with no origin.
func TestOpenReadsTheRepositoryFromOrigin(t *testing.T) {
	r := repoWith(t, nil)
	if m := Open(r); m.repo != "" {
		t.Errorf("no origin, and the repo is %q", m.repo)
	}
	if out, err := exec.Command("git", "-C", r.Dir, "remote", "add", "origin",
		"https://github.com/empowerite/kit.git").CombinedOutput(); err != nil {
		t.Fatalf("git remote add: %v\n%s", err, out)
	}
	if m := Open(r); m.repo != "empowerite/kit" || !strings.Contains(view(drive(m, at120)), "empowerite/kit") {
		t.Errorf("origin's repo is %q:\n%s", m.repo, view(drive(m, at120)))
	}
}

// A check whose step has no checks, or no page, shows under the last step that has checks.
func TestACheckOfAStepWithoutChecksShowsUnderTheLastStepThatHasThem(t *testing.T) {
	m, _ := fake(t)
	for _, step := range []string{"intro", "nonesuch"} {
		if got := m.stepOf(kit.Check{ID: "stray", Step: step}); got != setup {
			t.Errorf("a check of %q shows under step %d, not setup", step, got)
		}
	}
}

// Pages that make no step change nothing; pages without the selected step's select the first step, on its line.
func TestNewPagesWithoutTheSelectedStepSelectTheFirst(t *testing.T) {
	m, _ := loaded(t, esc, right, down)
	before := m.tabs
	if m.setTabs(nil); len(m.tabs) != len(before) || m.step != chains || !m.inRows {
		t.Fatalf("no pages changed the steps: %d steps, step %d", len(m.tabs), m.step)
	}
	m.setTabs([]tab{known[intro], known[terms]})
	if m.step != 0 || m.inRows || len(m.tabs) != 2 {
		t.Errorf("without the selected step: step %d, in rows %v, %d steps", m.step, m.inRows, len(m.tabs))
	}
}

// A refusal with nothing on stderr says how kit ended: with its code, or that it failed when there is none.
func TestARefusalWithNothingOnStderrSaysHowKitEnded(t *testing.T) {
	for _, c := range []struct {
		err  error
		want string
	}{
		{&kit.ExitError{Code: 3}, "kit welcome --checks exited 3 with nothing on stderr"},
		{errors.New(""), "kit welcome --checks failed"},
		{errors.New("kit welcome refuses"), "kit welcome refuses"},
	} {
		if got := refusalOf(c.err); got != c.want {
			t.Errorf("refusalOf(%#v) = %q, not %q", c.err, got, c.want)
		}
	}
}

// A fix that cannot run at all says why, in red, under its check.
func TestAFixThatCannotRunSaysWhy(t *testing.T) {
	m, _ := loaded(t)
	m.fixed(fixedMsg{id: "jq", err: errors.New("no such file")})
	if n := m.notes["jq"]; !n.refusal || n.text != "kit welcome --fix jq: no such file" {
		t.Errorf("the note: %+v", n)
	}
}

// On the line of a step with checks, up does nothing; down does nothing while the checks have not answered, and the
// step says it is checking.
func TestUpOnTheLineAndDownBeforeTheChecksAnswerDoNothing(t *testing.T) {
	m, _ := loaded(t, esc)
	if m = drive(m, up); m.inRows || m.step != basics {
		t.Errorf("up on the line: in rows %v, step %d", m.inRows, m.step)
	}
	m, _ = fake(t)
	m = drive(m, at120, right, right, right, down)
	if m.inRows {
		t.Error("down entered the rows of a step whose checks have not answered")
	}
	if v := view(m); !strings.Contains(v, "checking…") {
		t.Errorf("the step does not say it is checking:\n%s", v)
	}
}

// On a screen too narrow for any labels but the selected step's, the others shrink to their glyphs, and the selected
// keeps its name.
func TestTheSelectedStepKeepsItsNameWhenNothingElseFits(t *testing.T) {
	for _, width := range []int{20, 8} {
		m, _ := loaded(t, tea.WindowSizeMsg{Width: width, Height: 30})
		for i, f := range m.forms() {
			want := glyphLabel
			if i == m.step {
				want = fullLabel
			}
			if f != want {
				t.Errorf("at %d columns, step %d's label is form %d, not %d", width, i, f, want)
			}
		}
	}
}

// A step with checks, none of them its own once the checks have answered, shows its page alone.
func TestAStepWhoseChecksAnsweredWithNoneOfItsOwnShowsItsPageAlone(t *testing.T) {
	m, _ := loaded(t, esc)
	var basicsOnly []kit.Check
	for _, c := range m.checks {
		if c.Step == "before-toolchains" {
			basicsOnly = append(basicsOnly, c)
		}
	}
	m = drive(m, checksMsg{checks: basicsOnly}, right, right)
	if v := view(m); m.step != setup || strings.Contains(v, "checking…") || strings.Contains(v, "gh is not logged in") {
		t.Errorf("setup with no checks of its own:\n%s", v)
	}
}

// The pane scrolls back up to the cursor's row when its offset has gone past it, and leaves a cursor that is not on
// the step as it is.
func TestThePaneScrollsBackToTheCursorsRow(t *testing.T) {
	m, _ := loaded(t, tea.WindowSizeMsg{Width: 120, Height: 9}, esc, right, right, down)
	if !m.inRows {
		t.Fatal("not in the rows")
	}
	m.offsets[m.step] = 50
	m.fit()
	lines := m.stepLines()
	first := -1
	for k, l := range lines {
		if l.row == m.cursor {
			first = k
			break
		}
	}
	if m.offsets[m.step] > first {
		t.Errorf("the offset %d is past the cursor's first line %d", m.offsets[m.step], first)
	}
	// a cursor that names no row of the step leaves the offset as it is
	m.cursor, m.offsets[m.step] = len(m.checks)+5, 1
	m.fit()
	if m.offsets[m.step] != 1 {
		t.Errorf("a cursor off the step moved the offset to %d", m.offsets[m.step])
	}
}
