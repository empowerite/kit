package flow

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/source"
)

// big gives a backlog of n issues, each with a title long enough to wrap when opened.
func big(n int) source.Answer[[]byte] {
	var b []string
	for i := 1; i <= n; i++ {
		b = append(b, fmt.Sprintf(`{"number": %d, "title": "issue %d with a reasonably long title that goes on", `+
			`"assignees": ["ada"], "age": 7200, "branch": null}`, i, i))
	}
	return answer("backlog", "["+strings.Join(b, ",")+"]")
}

// bigNow gives big's answer as the backlog's latest load on the screen, which the fetch has run again.
func bigNow(m Model, n int) source.Answer[[]byte] {
	msg := big(n)
	msg.Gen = m.find("backlog").src.Gen()
	return msg
}

// A stage's rows are built once for what it loaded and the width: drawing, moving along the bar, and moving through
// the rows, and the legend floating over them, reuse them. A stage that loads again, or a new width, builds them anew.
func TestAStagesRowsAreBuiltOnceForItsDataAndTheWidth(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 40})
	m = drive(m, bigNow(m, 300))
	i := slices.IndexFunc(m.stages, func(s stage) bool { return s.name == "backlog" })
	first := m.rows(i)
	_ = m.View()
	m = drive(m, down, down, esc, right, left)
	_ = m.View()
	if again := m.rows(i); &again[0] != &first[0] {
		t.Error("the rows were built again for the same data and width")
	}
	m = drive(m, press('?'))
	_ = m.View()
	if again := m.rows(i); &again[0] != &first[0] {
		t.Error("the legend over the rows built them again")
	}
	m = drive(m, press('?'))
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
	if again := m.rows(i); &again[0] == &first[0] {
		t.Error("a new width reused the rows laid out for the old one")
	}
	wide := m.rows(i)
	m = drive(m, bigNow(m, 300))
	if again := m.rows(i); &again[0] == &wide[0] {
		t.Error("the stage loaded again and kept the rows of what it had")
	}
}

// With three hundred issues in the backlog, drawing the screen takes well under the 16 ms a frame lasts at 60 frames
// a second, and a step along the bar does too. The race detector slows everything about tenfold, so its bound is ten
// times as wide. Each is timed as the best of three batches, so a moment the machine is busy elsewhere, or the test
// runs on a slower core, does not count against the drawing.
func TestADrawingOfThreeHundredIssuesIsWellUnderAFramesTime(t *testing.T) {
	budget := 16 * time.Millisecond / 2
	if raceEnabled {
		budget *= 10
	}
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 40})
	m = drive(m, bigNow(m, 300))
	if per := best(func() { _ = m.View() }); per > budget {
		t.Errorf("a drawing takes %v, over %v", per, budget)
	}
	if per := best(func() {
		next, _ := m.Update(right)
		m = next.(Model)
		_ = m.View()
	}); per > budget {
		t.Errorf("a step along the bar and its drawing take %v, over %v", per, budget)
	}
}

// longWorktree gives the worktrees' answer as the screen's latest load: one worktree, n commits ahead, each with a
// subject long enough to cut, and thirty paths uncommitted, as a branch with a long history has.
func longWorktree(m Model, n int) source.Answer[[]byte] {
	var commits, changes []string
	for i := range n {
		commits = append(commits, fmt.Sprintf(`{"sha": "%07x", "subject": "#89 kittui: commit %d says what it `+
			`changed at a length that runs past the pane"}`, i+1, n-i))
	}
	for i := range 30 {
		changes = append(changes, fmt.Sprintf(`{"status": " M", "path": "projects/tui/internal/flow/file%d.go"}`, i))
	}
	msg := answer("worktrees", fmt.Sprintf(`{"goneBranches": [], "worktrees": [{"path": "/w/1-long", `+
		`"relative": "worktrees/1-long", "branch": "1-long", "ahead": %d, "behind": 0, "added": 9000, `+
		`"removed": 18, "detached": false, "uncommitted": 30, "commits": [%s], "changes": [%s]}]}`,
		n, strings.Join(commits, ","), strings.Join(changes, ",")))
	msg.Gen = m.find("worktrees").src.Gen()
	return msg
}

// A stage's lines are drawn once for what it shows: paging through an open row, and drawing the screen, reuse them.
// Moving the cursor, opening or closing a row, or selecting one, draws them anew.
func TestAStagesLinesAreDrawnOnceForWhatItShows(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 40})
	m = drive(m, longWorktree(m, 100))
	for m.stages[m.stage].name != "worktrees" {
		m = drive(m, right)
	}
	m = drive(intoRows(m), right)
	first := m.allLines(m.stage)
	_ = m.View()
	m = drive(m, press(tea.KeyPgDown), press(tea.KeyPgDown), press(tea.KeyPgUp))
	_ = m.View()
	if again := m.allLines(m.stage); &again[0] != &first[0] {
		t.Error("paging drew the lines again for the same rows")
	}
	for _, change := range []struct {
		name string
		key  tea.Msg
	}{{"closing the row", left}, {"opening it", right}, {"selecting it", press(tea.KeySpace)}, {"moving off it", up}} {
		before := m.allLines(m.stage)
		m = drive(m, change.key)
		if again := m.allLines(m.stage); &again[0] == &before[0] {
			t.Errorf("%s kept the lines drawn before", change.name)
		}
	}
}

// With a worktree of seven hundred commits open, a page through it and its drawing take well under a frame's time,
// down and back up: the work a key does follows the lines on the screen, not every line of the open row, so keys
// held down, or a wheel spun, never pile up behind each other, ^C among them.
func TestPagingThroughAWorktreeOfSevenHundredCommitsIsWellUnderAFramesTime(t *testing.T) {
	budget := 16 * time.Millisecond / 2
	if raceEnabled {
		budget *= 10
	}
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 40})
	m = drive(m, longWorktree(m, 700))
	for m.stages[m.stage].name != "worktrees" {
		m = drive(m, right)
	}
	m = drive(m, down, right)
	for _, key := range []tea.KeyPressMsg{press(tea.KeyPgDown), press(tea.KeyPgUp)} {
		if per := best(func() {
			next, _ := m.Update(key)
			m = next.(Model)
			_ = m.View()
		}); per > budget {
			t.Errorf("%s and its drawing take %v, over %v", key, per, budget)
		}
	}
}

// best gives how long do takes, the least of three batches of ten.
func best(do func()) time.Duration {
	const batches, n = 3, 10
	least := time.Duration(math.MaxInt64)
	for range batches {
		start := time.Now()
		for range n {
			do()
		}
		least = min(least, time.Since(start)/n)
	}
	return least
}

func BenchmarkADrawingOfThreeHundredIssues(b *testing.B) {
	m := New(kit.Runner{}, options())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(Model)
	next, _ = m.Update(big(300))
	m = next.(Model)
	b.ResetTimer()
	for range b.N {
		_ = m.View()
	}
}
