package flow

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/source"
)

// big gives issues' document of n issues, each with a title long enough to wrap when opened.
func big(n int) source.Answer[[]byte] {
	var b []string
	for i := 1; i <= n; i++ {
		b = append(b, fmt.Sprintf(`{"number": %d, "title": "issue %d with a reasonably long title that goes on", `+
			`"assignees": ["ada"], "age": 7200, "branch": null}`, i, i))
	}
	return answer("issues", "["+strings.Join(b, ",")+"]")
}

// bigNow gives big's answer as issues' latest load on the screen, which the fetch has run again.
func bigNow(m Model, n int) source.Answer[[]byte] {
	msg := big(n)
	msg.Gen = m.find("issues").src.Gen()
	return msg
}

// A stage's rows are built once for what it loaded and the width: drawing, moving along the bar, and moving through
// the rows, and the legend floating over them, reuse them. A stage that loads again, or a new width, builds them anew.
func TestAStagesRowsAreBuiltOnceForItsDataAndTheWidth(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 40})
	m = drive(m, bigNow(m, 300))
	i := slices.IndexFunc(m.stages, func(s stage) bool { return s.name == "issues" })
	first := m.rows(i)
	_ = m.View()
	m = drive(m, down, down, esc, tab, backTab)
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

// With three hundred issues, drawing the screen takes well under the 16 ms a frame lasts at 60 frames
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
		next, _ := m.Update(tab)
		m = next.(Model)
		_ = m.View()
	}); per > budget {
		t.Errorf("a step along the bar and its drawing take %v, over %v", per, budget)
	}
}

// tallBuild gives builds' answer as the screen's latest load: the runs testdata/builds.json holds, the first with n
// jobs, each with a name long enough to cut, every third failed at a step, so its open row is taller than any pane.
func tallBuild(m Model, n int) source.Answer[[]byte] {
	var runs []map[string]any
	if data, err := os.ReadFile(filepath.Join("testdata", "builds.json")); err != nil || json.Unmarshal(data, &runs) != nil {
		panic(fmt.Sprintf("testdata/builds.json: %v", err))
	}
	jobs := make([]map[string]any, n)
	for i := range n {
		jobs[i] = map[string]any{"name": fmt.Sprintf("job %d checks what it says at a length that runs past the pane",
			i+1), "state": "passed", "took": i + 1, "step": nil}
		if i%3 == 2 {
			jobs[i]["state"], jobs[i]["step"] = "failed", fmt.Sprintf("step %d", i+1)
		}
	}
	runs[0]["jobs"] = jobs
	data, _ := json.Marshal(runs)
	msg := answer("builds", string(data))
	msg.Gen = m.find("builds").src.Gen()
	return msg
}

// A stage's lines are drawn once for what it shows: paging through an open row, and drawing the screen, reuse them.
// Moving the cursor, opening or closing a row, or selecting one, draws them anew.
func TestAStagesLinesAreDrawnOnceForWhatItShows(t *testing.T) {
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 40})
	m = drive(m, tallBuild(m, 100))
	for m.stages[m.stage].name != "builds" {
		m = drive(m, tab)
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

// With a build of seven hundred jobs open, a page through it and its drawing take well under a frame's time,
// down and back up: the work a key does follows the lines on the screen, not every line of the open row, so keys
// held down, or a wheel spun, never pile up behind each other, ^C among them.
func TestPagingThroughABuildOfSevenHundredJobsIsWellUnderAFramesTime(t *testing.T) {
	budget := 16 * time.Millisecond / 2
	if raceEnabled {
		budget *= 10
	}
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 40})
	m = drive(m, tallBuild(m, 700))
	for m.stages[m.stage].name != "builds" {
		m = drive(m, tab)
	}
	m = drive(intoRows(m), right)
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

func BenchmarkACatalogStagesRows(b *testing.B) {
	o := options()
	o.Stages = []string{"incoming"}
	m := New(kit.Runner{}, o)
	data, err := os.ReadFile(filepath.Join("testdata", "incoming.json"))
	if err != nil {
		b.Fatal(err)
	}
	next, _ := m.Update(answer("incoming", string(data)))
	m = next.(Model)
	b.ResetTimer()
	for range b.N {
		_ = m.catalogRows(*m.find("incoming"))
		_ = m.count(0)
	}
}

// With ten thousand issues, a turn of the mouse wheel over them and its drawing take well under a frame's time: a
// wheel spun sends dozens of turns, and the screen is drawn after each, so a turn costs what the columns and the lines
// on the screen do, not every issue, and the rows keep up with the wheel.
func TestATurnOfTheWheelOverTenThousandIssuesIsWellUnderAFramesTime(t *testing.T) {
	budget := 16 * time.Millisecond / 2
	if raceEnabled {
		budget *= 10
	}
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 40})
	m = drive(m, bigNow(m, 10000))
	for range m.stages {
		if m.stages[m.stage].name == "issues" {
			break
		}
		m = drive(m, tab)
	}
	_ = m.View()
	turn := tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 5, Y: m.top() + m.paneHeight()/2}
	if per := best(func() {
		next, _ := m.Update(turn)
		m = next.(Model)
		_ = m.View()
	}); per > budget {
		t.Errorf("a turn of the wheel and its drawing take %v, over %v", per, budget)
	}
	if m.stages[m.stage].Offset == 0 {
		t.Error("the wheel did not scroll the issues")
	}
}
