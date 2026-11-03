package flow

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
)

// big gives a backlog of n issues, each with a title long enough to wrap when opened.
func big(n int) loadedMsg {
	var b []string
	for i := 1; i <= n; i++ {
		b = append(b, fmt.Sprintf(`{"number": %d, "title": "issue %d with a reasonably long title that goes on", `+
			`"assignees": ["ada"], "age": 7200, "branch": null}`, i, i))
	}
	return loadedMsg{stage: "backlog", data: []byte("[" + strings.Join(b, ",") + "]")}
}

// A stage's rows are built once for what it loaded and the width: drawing, moving along the bar, and moving through
// the rows reuse them. A stage that loads again, a new width, or the legend shown beside them builds them anew.
func TestAStagesRowsAreBuiltOnceForItsDataAndTheWidth(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 40}, big(300))
	i := slices.IndexFunc(m.stages, func(s stage) bool { return s.name == "backlog" })
	first := m.rows(i)
	_ = m.View()
	m = drive(m, down, down, esc, right, left)
	_ = m.View()
	if again := m.rows(i); &again[0] != &first[0] {
		t.Error("the rows were built again for the same data and width")
	}
	m = drive(m, press('?'))
	if again := m.rows(i); &again[0] == &first[0] {
		t.Error("the legend beside the rows reused the rows laid out for the whole width")
	}
	m = drive(m, press('?'))
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
	if again := m.rows(i); &again[0] == &first[0] {
		t.Error("a new width reused the rows laid out for the old one")
	}
	wide := m.rows(i)
	m = drive(m, big(300))
	if again := m.rows(i); &again[0] == &wide[0] {
		t.Error("the stage loaded again and kept the rows of what it had")
	}
}

// With three hundred issues in the backlog, drawing the screen takes well under the 16 ms a frame lasts at 60 frames
// a second, and a step along the bar does too. The race detector slows everything about tenfold, so its bound is ten
// times as wide.
func TestADrawingOfThreeHundredIssuesIsWellUnderAFramesTime(t *testing.T) {
	budget := 16 * time.Millisecond / 2
	if raceEnabled {
		budget *= 10
	}
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 40}, big(300))
	const n = 30
	start := time.Now()
	for range n {
		_ = m.View()
	}
	if per := time.Since(start) / n; per > budget {
		t.Errorf("a drawing takes %v, over %v", per, budget)
	}
	start = time.Now()
	for range n {
		next, _ := m.Update(right)
		m = next.(Model)
		_ = m.View()
	}
	if per := time.Since(start) / n; per > budget {
		t.Errorf("a step along the bar and its drawing take %v, over %v", per, budget)
	}
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
