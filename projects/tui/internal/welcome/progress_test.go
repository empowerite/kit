package welcome

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/source"
)

// hear gives the screen each message, as Update does, and the command the last one gave, which it does not run: a
// listen waits for kit.
func hear(m Model, msgs ...tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	for _, msg := range msgs {
		var next tea.Model
		next, cmd = m.Update(msg)
		m = next.(Model)
	}
	return m, cmd
}

// While the checks run, a step with checks says under checking… the check each folder, and the machine, is on, as kit
// says it starts each; a folder whose checks end leaves the list, an earlier run's lines are dropped, the run's end
// stops the listening, and its answer clears the list.
func TestCheckingSaysWhatEachFolderIsOn(t *testing.T) {
	m, _ := loaded(t, stepKey(chains))
	m.checking.Again(m.env(), false)
	run := int64(m.checking.Gen())
	m.run.Store(run)
	m, cmd := hear(m,
		toldMsg{run: run, p: kit.Progress{Check: "tool-git"}},
		toldMsg{run: run, p: kit.Progress{Check: "repo-git", Folder: "/w/a"}},
		toldMsg{run: run, p: kit.Progress{Check: "kit-check", Folder: "/w/b"}},
		toldMsg{run: run - 1, p: kit.Progress{Check: "stale", Folder: "/w/c"}},
		toldMsg{run: run, p: kit.Progress{Check: "kit-check", Folder: "/w/a"}})
	running := "   checking…\n     tool-git\n     kit-check in a\n     kit-check in b\n"
	if got := view(m); !strings.Contains(got, running) || strings.Contains(got, "stale") || cmd == nil {
		t.Errorf("listening %v:\n%s", cmd != nil, got)
	}
	m, _ = hear(m, toldMsg{run: run, p: kit.Progress{}}, toldMsg{run: run, p: kit.Progress{Folder: "/w/b"}})
	if got := view(m); !strings.Contains(got, "   checking…\n     kit-check in a\n") ||
		strings.Contains(got, "tool-git") || strings.Contains(got, "in b") {
		t.Errorf("ended:\n%s", got)
	}
	if m, cmd = hear(m, toldMsg{run: run, end: true}); cmd != nil || m.listening {
		t.Errorf("the run's end still listens")
	}
	m, _ = hear(m, source.Answer[[]kit.Check]{Name: "checks", Value: m.checks, Gen: m.checking.Gen()})
	if got := view(m); strings.Contains(got, "checking…") || strings.Contains(got, "kit-check in a") {
		t.Errorf("answered:\n%s", got)
	}
}

// Running the checks again while the screen listens to kit starts no second listen: the run is all it asks for. Once
// the screen ends, a listen gives nothing.
func TestARunAgainListensOnceAndAListenEndsWithTheScreen(t *testing.T) {
	m, _ := loaded(t)
	if _, batch := m.again()().(tea.BatchMsg); !batch || !m.listening {
		t.Errorf("a run again not listening gives no listen with the run: listening %v", m.listening)
	}
	if msg, answer := m.again()().(source.Answer[[]kit.Check]); !answer || msg.Err != nil {
		t.Errorf("a run again while listening gives %#v, not the run's answer alone", msg)
	}
	ended, _ := fake(t)
	ended.Close()
	if msg := ended.listen()(); msg != nil {
		t.Errorf("a listen after the screen ended gives %#v", msg)
	}
}
