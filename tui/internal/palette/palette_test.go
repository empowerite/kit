// The palette lists the verbs, filters them as they are typed, shows the highlighted one's page, and opens it.
package palette

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/tui/internal/kit"
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

func names(m Model) string {
	var out []string
	for _, v := range m.Visible() {
		out = append(out, v.Name)
	}
	return strings.Join(out, " ")
}

func TestEveryVerbButTui(t *testing.T) {
	m, _ := drive(open(t))
	if names(m) != "check commit context version" {
		t.Errorf("verbs: %q", names(m))
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
	_, produced := drive(open(t), tea.KeyPressMsg{Code: tea.KeyDown}, tea.KeyPressMsg{Code: tea.KeyEnter})
	if got, ok := produced[len(produced)-1].(OpenMsg); !ok || got.Verb != "commit" {
		t.Fatalf("produced %v", produced)
	}
}

func TestTheListShowsWholePurposesAndNoPage(t *testing.T) {
	m, _ := drive(open(t), tea.WindowSizeMsg{Width: 120, Height: 30}, tea.KeyPressMsg{Code: tea.KeyDown})
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
	m, _ := drive(open(t), tea.WindowSizeMsg{Width: 120, Height: 30}, tea.KeyPressMsg{Code: tea.KeyDown},
		tea.KeyPressMsg{Code: tea.KeyRight})
	view := m.View().Content
	if !strings.Contains(view, "kit commit - a purpose for the test") || strings.Contains(view, "commits what is staged") {
		t.Fatalf("right did not show the page alone:\n%s", view)
	}
	m, _ = drive(m, tea.KeyPressMsg{Code: tea.KeyDown})
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
