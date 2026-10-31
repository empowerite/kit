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

func TestPreviewIsHelp(t *testing.T) {
	m, _ := drive(open(t), tea.WindowSizeMsg{Width: 120, Height: 30}, tea.KeyPressMsg{Code: tea.KeyDown})
	view := m.View().Content
	if !strings.Contains(view, "kit commit - a purpose for the test") || !strings.Contains(view, "commits what is staged") {
		t.Errorf("view:\n%s", view)
	}
}

func TestEscQuits(t *testing.T) {
	m, produced := drive(open(t), tea.KeyPressMsg{Code: tea.KeyEscape})
	if _, quit := produced[len(produced)-1].(tea.QuitMsg); !quit || !m.Cancelled() {
		t.Errorf("esc: %v", produced)
	}
}
