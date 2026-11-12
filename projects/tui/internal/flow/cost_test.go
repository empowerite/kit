package flow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

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

// Drawing the screen, and a step along the bar with its drawing, cost what the screen shows, not every issue: a stage's
// rows are built once for what it loaded, as TestAStagesRowsAreBuiltOnceForItsDataAndTheWidth holds them, so with three
// thousand issues each allocates no more than 64 bytes an issue past what it does with three hundred. What they
// allocate is counted, not timed, so the machine's load weighs on it not at all; BenchmarkADrawingOfThreeHundredIssues
// says how long a drawing takes.
func TestADrawingCostsWhatTheScreenShowsNotTheIssues(t *testing.T) {
	drawing, step := func(n int) uint64 {
		m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 40})
		m = drive(m, bigNow(m, n))
		return allocated(func() { _ = m.View() })
	}, func(n int) uint64 {
		m := loaded(t, options(), tea.WindowSizeMsg{Width: 120, Height: 40})
		m = drive(m, bigNow(m, n))
		return allocated(func() {
			next, _ := m.Update(tab)
			m = next.(Model)
			_ = m.View()
		})
	}
	if few, many := drawing(300), drawing(3000); many > few+2700*perRow {
		t.Errorf("a drawing of three thousand issues allocates %d bytes, of three hundred %d", many, few)
	}
	if few, many := step(300), step(3000); many > few+2700*perRow {
		t.Errorf("a step along the bar over three thousand issues allocates %d bytes, over three hundred %d", many, few)
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

// A page through an open build and its drawing, down and back up, cost what the screen shows, not every line of the
// open row: keys held down, or a wheel spun, never pile up behind each other, ^C among them. With seven hundred jobs
// open a page allocates no more than 64 bytes a job past what it does with seventy, counted, not timed;
// BenchmarkPagingThroughABuildOfSevenHundredJobs says how long a page takes.
func TestPagingThroughABuildCostsWhatTheScreenShowsNotTheJobs(t *testing.T) {
	paging := func(n int, key tea.KeyPressMsg) uint64 {
		m := openBuild(loaded(t, options()), n)
		return allocated(func() {
			next, _ := m.Update(key)
			m = next.(Model)
			_ = m.View()
		})
	}
	for _, key := range []tea.KeyPressMsg{press(tea.KeyPgDown), press(tea.KeyPgUp)} {
		if few, many := paging(70, key), paging(700, key); many > few+630*perRow {
			t.Errorf("%s through seven hundred jobs allocates %d bytes, through seventy %d", key, many, few)
		}
	}
}

// openBuild gives the screen, at 120×40, with builds' first run open, with n jobs, the rows with the keyboard, flat:
// grouped by nothing, though every run is of one source, organization, and repository.
func openBuild(m Model, n int) Model {
	unfold(&m)
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40}, tallBuild(m, n))
	for m.stages[m.stage].name != "builds" {
		m = drive(m, tab)
	}
	return drive(intoRows(m), right)
}

func BenchmarkPagingThroughABuildOfSevenHundredJobs(b *testing.B) {
	m := openBuild(New(kit.Runner{}, options()), 700)
	keys := []tea.KeyPressMsg{press(tea.KeyPgDown), press(tea.KeyPgUp)}
	for i := 0; b.Loop(); i++ {
		next, _ := m.Update(keys[i%2])
		m = next.(Model)
		_ = m.View()
	}
}

// perRow is the most a drawing may allocate for each row past those the smaller of two screens has: the few bytes a
// slice of each row's text takes, where a copy of the row takes hundreds.
const perRow = 64

// allocated gives the bytes do allocates, on average over twenty runs, as the runtime counts them.
func allocated(do func()) uint64 {
	const runs = 20
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range runs {
		do()
	}
	runtime.ReadMemStats(&after)
	return (after.TotalAlloc - before.TotalAlloc) / runs
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

// issuesWheel gives the screen, at 120×40, on issues, with n of them, drawn once, and a turn of the wheel down over
// its rows.
func issuesWheel(m Model, n int) (Model, tea.MouseWheelMsg) {
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40}, bigNow(m, n))
	for range m.stages {
		if m.stages[m.stage].name == "issues" {
			break
		}
		m = drive(m, tab)
	}
	_ = m.View()
	return m, tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 5, Y: m.top() + m.paneHeight()/2}
}

// A turn of the mouse wheel and its drawing cost what the screen shows, not every issue: a wheel spun sends dozens of
// turns, and the screen is drawn after each, so the rows keep up with the wheel however many there are. Over ten
// thousand issues a turn allocates no more than 64 bytes an issue past what it does over a thousand, counted, not
// timed; BenchmarkATurnOfTheWheelOverTenThousandIssues says how long a turn takes.
func TestATurnOfTheWheelCostsWhatTheScreenShowsNotTheIssues(t *testing.T) {
	turning := func(n int) uint64 {
		m, turn := issuesWheel(loaded(t, options()), n)
		bytes := allocated(func() {
			next, _ := m.Update(turn)
			m = next.(Model)
			_ = m.View()
		})
		if m.stages[m.stage].Offset == 0 {
			t.Errorf("the wheel did not scroll %d issues", n)
		}
		return bytes
	}
	if few, many := turning(1000), turning(10000); many > few+9000*perRow {
		t.Errorf("a turn over ten thousand issues allocates %d bytes, over a thousand %d", many, few)
	}
}

func BenchmarkATurnOfTheWheelOverTenThousandIssues(b *testing.B) {
	m, turn := issuesWheel(New(kit.Runner{}, options()), 10000)
	for b.Loop() {
		next, _ := m.Update(turn)
		m = next.(Model)
		_ = m.View()
	}
}
