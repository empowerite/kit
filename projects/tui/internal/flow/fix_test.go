package flow

import (
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/fix"
)

// refusedChanges opens the screen on changes, refused with kit's line, which names kit help flow to run, and the
// terminal handed to a stand-in that records what ran.
func refusedChanges(t *testing.T, ran *[][]string) Model {
	t.Helper()
	t.Setenv("FAKE_FLOW_REFUSE", "changes")
	m := loaded(t, options())
	at := slices.IndexFunc(m.stages, func(s stage) bool { return s.name == "changes" })
	m = drive(m, typed(strconv.Itoa(at+1))...)
	m.handOver = func(c *exec.Cmd, done tea.ExecCallback) tea.Cmd {
		*ran = append(*ran, c.Args)
		return func() tea.Msg { return done(nil) }
	}
	return m
}

// remedyLine gives the screen line the refusal's pair stands on, and the line as drawn, or -1.
func remedyLine(m Model) (int, string) {
	for y, l := range strings.Split(raw(m), "\n") {
		if strings.Contains(l, "kit help flow") {
			return y, l
		}
	}
	return -1, ""
}

func TestARefusalThatNamesACommandOffersCopyAndRunOnlyWhenSelected(t *testing.T) {
	var ran [][]string
	m := refusedChanges(t, &ran)
	if _, l := remedyLine(m); strings.Contains(l, fix.CopyName) || strings.Contains(l, " "+fix.RunName+" ") {
		t.Fatalf("the pair shows on a line not selected: %q", l)
	}
	m = drive(m, down)
	_, l := remedyLine(m)
	if !strings.HasSuffix(strings.TrimRight(l, " "), fix.CopyName+"   "+fix.RunName) {
		t.Fatalf("down did not select the line with its pair: %q", l)
	}
	if k := keysOf(m); !strings.Contains(k, "c copy") || !strings.Contains(k, "r run") {
		t.Errorf("the keys bar does not name the pair: %q", k)
	}
	m = drive(m, typed("c")...)
	if _, l := remedyLine(m); !strings.Contains(l, fix.CopiedName) {
		t.Errorf("c did not say copied: %q", l)
	}
	t.Setenv("FAKE_FLOW_REFUSE", "")
	m = drive(m, typed("r")...)
	if !slices.EqualFunc(ran, [][]string{{"kit", "help", "flow"}}, slices.Equal) {
		t.Fatalf("r ran %q", ran)
	}
	if !shows(m, Change, "3") {
		t.Errorf("the stage did not load again after its fix ran:\n%s", view(m))
	}
	m = drive(refusedChanges(t, &ran), down, up)
	if _, l := remedyLine(m); strings.Contains(l, fix.CopyName) {
		t.Errorf("up did not leave the line: %q", l)
	}
}

func TestAClickOnTheRefusalsButtonsCopiesOrRuns(t *testing.T) {
	var ran [][]string
	m := refusedChanges(t, &ran)
	y, l := remedyLine(m)
	// a click on the line's words selects it, and does nothing more
	m = drive(m, tea.MouseClickMsg{X: 5, Y: y, Button: tea.MouseLeft})
	if !m.onRemedy() || len(ran) > 0 {
		t.Fatalf("a click on the words: selected %v, ran %q", m.onRemedy(), ran)
	}
	y, l = remedyLine(m)
	plain := ansi.Strip(l)
	at := func(name string) int {
		i := strings.LastIndex(plain, " "+name+" ")
		return ansi.StringWidth(plain[:i]) + 1
	}
	m = drive(m, tea.MouseClickMsg{X: at(fix.CopyName), Y: y, Button: tea.MouseLeft})
	if _, l := remedyLine(m); !strings.Contains(l, fix.CopiedName) || len(ran) > 0 {
		t.Fatalf("a click on ⧉: %q, ran %q", l, ran)
	}
	y, l = remedyLine(m)
	plain = ansi.Strip(l)
	m = drive(m, tea.MouseClickMsg{X: at(fix.RunName), Y: y, Button: tea.MouseLeft})
	if !slices.EqualFunc(ran, [][]string{{"kit", "help", "flow"}}, slices.Equal) {
		t.Errorf("a click on run ran %q", ran)
	}
}

func TestARefusalThatNamesNoCommandOffersNothing(t *testing.T) {
	t.Setenv("FAKE_FLOW_SILENT", "issues")
	m := loaded(t, options(), down)
	if m.onRemedy() || strings.Contains(raw(m), fix.CopyName) {
		t.Errorf("a refusal naming no command offered the pair:\n%s", raw(m))
	}
	if _, ok := m.remedyKey(press('r')); ok {
		t.Errorf("r was taken on a refusal naming no command")
	}
}
