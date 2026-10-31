// The palette lists the verbs as a tree of their groups, filters them as they are typed, moves over the verbs and
// between the groups, shows the highlighted one's page, and opens it.
package palette

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/tui/internal/kit"
	"github.com/empowerite/kit/tui/internal/style"
)

func open(t *testing.T) Model {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "kit", "testdata", "fake-kit"))
	if err != nil {
		t.Fatal(err)
	}
	r := kit.Runner{Kit: path, Dir: t.TempDir()}
	verbs, err := r.List()
	if err != nil {
		t.Fatal(err)
	}
	return New(verbs, r)
}

// drive sends the messages, running each command and feeding its message back unless it is the palette's answer.
func drive(m Model, msgs ...tea.Msg) (Model, []tea.Msg) {
	var produced []tea.Msg
	queue := append([]tea.Msg{}, msgs...)
	if cmd := m.Init(); cmd != nil {
		queue = append([]tea.Msg{cmd()}, queue...)
	}
	for len(queue) > 0 {
		next, cmd := m.Update(queue[0])
		queue = queue[1:]
		m = next.(Model)
		if cmd != nil {
			out := cmd()
			produced = append(produced, out)
			switch out.(type) {
			case OpenMsg, tea.QuitMsg:
			default:
				queue = append([]tea.Msg{out}, queue...)
			}
		}
	}
	return m, produced
}

func typed(text string) []tea.Msg {
	var msgs []tea.Msg
	for _, r := range text {
		msgs = append(msgs, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return msgs
}

var (
	up        = tea.KeyPressMsg{Code: tea.KeyUp}
	down      = tea.KeyPressMsg{Code: tea.KeyDown}
	shiftUp   = tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}
	shiftDown = tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift}
)

func names(m Model) string {
	var out []string
	for _, v := range m.Visible() {
		out = append(out, v.Name)
	}
	return strings.Join(out, " ")
}

// highlighted gives the name of the verb under the cursor.
func highlighted(m Model) string { return m.Visible()[m.cursor].Name }

// drawn gives the rows of the list as the view draws them, one word each: a heading as itself, a verb as its name,
// and the highlighted verb as ▸ and its name.
func drawn(m Model) string {
	var out []string
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	for i, line := range lines {
		if strings.Contains(line, style.Filter) {
			lines = lines[i+1:]
			break
		}
	}
	for _, line := range lines {
		fields := strings.Fields(line)
		switch {
		case len(fields) == 0:
		case fields[0] == style.Focus:
			out = append(out, style.Focus+fields[1])
		default:
			out = append(out, fields[0])
		}
	}
	return strings.Join(out, " ")
}

func TestEveryVerbButTuiInTheOrderOfTheGroups(t *testing.T) {
	m, _ := drive(open(t))
	if names(m) != "commit submit context check version" {
		t.Errorf("verbs: %q", names(m))
	}
}

func TestTheVerbsAreATreeUnderTheirGroupsHeadings(t *testing.T) {
	m, _ := drive(open(t), tea.WindowSizeMsg{Width: 120, Height: 30})
	if drawn(m) != "WORKFLOW ▸commit submit FILES context CHECKS check INSTALLATION version" {
		t.Errorf("tree: %q", drawn(m))
	}
	view := m.View().Content
	if !strings.Contains(view, " "+style.Off("WORKFLOW")+"\n") {
		t.Errorf("the heading is not dimmed on a line of its own:\n%s", view)
	}
	if first := strings.SplitN(view, "\n", 2)[0]; !strings.Contains(first, "↑↓ move · ⇧↑↓ group") {
		t.Errorf("the keys do not say what shift does: %q", first)
	}
}

func TestUpAndDownMoveOverTheVerbsAndSkipTheHeadings(t *testing.T) {
	m, _ := drive(open(t), down, down)
	if drawn(m) != "WORKFLOW commit submit FILES ▸context CHECKS check INSTALLATION version" {
		t.Errorf("down twice: %q", drawn(m))
	}
	m, _ = drive(m, up)
	if highlighted(m) != "submit" {
		t.Errorf("up: %q", highlighted(m))
	}
	m, _ = drive(m, up, up)
	if highlighted(m) != "commit" {
		t.Errorf("up at the first verb: %q", highlighted(m))
	}
}

func TestShiftDownAndShiftUpMoveToTheFirstVerbOfTheNextAndPreviousGroup(t *testing.T) {
	m, _ := drive(open(t))
	for _, want := range []string{"context", "check", "version", "version"} {
		m, _ = drive(m, shiftDown)
		if highlighted(m) != want {
			t.Errorf("shift+down: %q, want %q", highlighted(m), want)
		}
	}
	for _, want := range []string{"check", "context", "commit", "commit"} {
		m, _ = drive(m, shiftUp)
		if highlighted(m) != want {
			t.Errorf("shift+up: %q, want %q", highlighted(m), want)
		}
	}
	// from a verb below the first of its group, shift+up goes to that first verb
	m, _ = drive(m, down, shiftUp)
	if highlighted(m) != "commit" {
		t.Errorf("shift+up within a group: %q", highlighted(m))
	}
}

func TestAFilterThatEmptiesAGroupHidesItsHeading(t *testing.T) {
	m, _ := drive(open(t), typed("co")...)
	if drawn(m) != "WORKFLOW ▸commit FILES context" {
		t.Errorf("co: %q", drawn(m))
	}
	m, _ = drive(open(t), typed("zz")...)
	if drawn(m) != "no" || !strings.Contains(m.View().Content, "no verb matches") {
		t.Errorf("zz: %q", drawn(m))
	}
}

func TestTheWindowCountsTheHeadingsSoTheCursorStaysOnScreen(t *testing.T) {
	m, _ := drive(open(t), tea.WindowSizeMsg{Width: 120, Height: 8}, down, down, down, down)
	if highlighted(m) != "version" || !strings.HasSuffix(drawn(m), "▸version") {
		t.Errorf("the cursor's row is off the screen: %q", drawn(m))
	}
	if n := strings.Count(strings.TrimRight(m.View().Content, "\n"), "\n") + 1; n > 8 {
		t.Errorf("%d lines on a screen of 8:\n%s", n, m.View().Content)
	}
}

func TestFilterNarrows(t *testing.T) {
	m, _ := drive(open(t), typed("co")...)
	if names(m) != "commit context" {
		t.Errorf("co: %q", names(m))
	}
	m, _ = drive(m, typed("m")...)
	if names(m) != "commit" {
		t.Errorf("com: %q", names(m))
	}
	m, _ = drive(open(t), typed("tui")...)
	if names(m) != "" || !strings.Contains(m.View().Content, "no verb matches") {
		t.Errorf("tui: %q", names(m))
	}
}

func TestEnterOpensHighlighted(t *testing.T) {
	_, produced := drive(open(t), down, tea.KeyPressMsg{Code: tea.KeyEnter})
	if got, ok := produced[len(produced)-1].(OpenMsg); !ok || got.Verb != "submit" {
		t.Fatalf("produced %v", produced)
	}
}

func TestTheListShowsWholePurposesAndNoPage(t *testing.T) {
	m, _ := drive(open(t), tea.WindowSizeMsg{Width: 120, Height: 30}, down)
	view := m.View().Content
	if !strings.Contains(view, "commits what is staged, after the checks a commit's rule requires") {
		t.Errorf("the purpose is not whole:\n%s", view)
	}
	if strings.Contains(view, "kit commit - a purpose for the test") {
		t.Errorf("a page in the list view:\n%s", view)
	}
	if first := strings.SplitN(view, "\n", 2)[0]; !strings.Contains(first, "↑↓ move") {
		t.Errorf("the keys are not on the first line: %q", first)
	}
}

func TestARowWiderThanTheScreenIsClippedAtItsEdge(t *testing.T) {
	m, _ := drive(open(t), tea.WindowSizeMsg{Width: 40, Height: 30})
	for _, line := range strings.Split(m.View().Content, "\n") {
		if n := len([]rune(stripANSI(line))); n > 40 {
			t.Errorf("a line of %d columns: %q", n, line)
		}
	}
	if !strings.Contains(m.View().Content, "…") {
		t.Errorf("no row was clipped:\n%s", m.View().Content)
	}
}

func TestRightShowsThePagesAndLeftTheListAgain(t *testing.T) {
	m, _ := drive(open(t), tea.WindowSizeMsg{Width: 120, Height: 30}, down, tea.KeyPressMsg{Code: tea.KeyRight})
	view := m.View().Content
	if !strings.Contains(view, "kit submit - a purpose for the test") || strings.Contains(view, "commits what is staged") {
		t.Fatalf("right did not show the page alone:\n%s", view)
	}
	// the names' column is wide enough for every heading, so none wraps beside the page
	if drawn(m) != "WORKFLOW commit ▸submit FILES context CHECKS check INSTALLATION version" {
		t.Errorf("the tree beside the page: %q", drawn(m))
	}
	m, _ = drive(m, down)
	if !strings.Contains(m.View().Content, "kit context - a purpose for the test") {
		t.Errorf("down in the pages did not follow:\n%s", m.View().Content)
	}
	m, _ = drive(m, tea.KeyPressMsg{Code: tea.KeyLeft})
	view = m.View().Content
	if strings.Contains(view, "kit context - a purpose for the test") || !strings.Contains(view, "prints the rules that govern a file") {
		t.Errorf("left did not bring the list back:\n%s", view)
	}
}

func TestTheKeysStayOnTopWhileFiltering(t *testing.T) {
	m, _ := drive(open(t), typed("zz")...)
	if first := strings.SplitN(m.View().Content, "\n", 2)[0]; !strings.Contains(first, "↑↓ move") {
		t.Errorf("the keys moved: %q", first)
	}
}

// stripANSI drops the escape sequences of styled text, for measuring it.
func stripANSI(s string) string {
	var b strings.Builder
	escape := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			escape = true
		case escape && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'):
			escape = false
		case !escape:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestEscQuits(t *testing.T) {
	m, produced := drive(open(t), tea.KeyPressMsg{Code: tea.KeyEscape})
	if _, quit := produced[len(produced)-1].(tea.QuitMsg); !quit || !m.Cancelled() {
		t.Errorf("esc: %v", produced)
	}
}

func TestAPageWrapsByWhatShowsNotByItsEscapes(t *testing.T) {
	lines := page("\x1b[1mab\x1b[0m cd", 5)
	if len(lines) != 1 || lines[0] != "\x1b[1mab\x1b[0m cd" {
		t.Fatalf("page: %q", lines)
	}
}
