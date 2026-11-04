// kittui's root shows one of kit's screens at a time: one is made only when it is first shown, and kept as it was left
// while another is shown; what a hidden screen started reaches it, and a screen that cannot open leaves the one shown.
package screens

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/app"
	"github.com/empowerite/kit/projects/tui/internal/flow"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
	"github.com/empowerite/kit/projects/tui/internal/welcome"
)

// help gives the root on the help's palette, over the kit package's fake kit, at 120×30.
func help(t *testing.T) Model {
	t.Helper()
	r := testkit.Runner(t, "kit")
	p, err := app.OpenHelp(r, "")
	if err != nil {
		t.Fatal(err)
	}
	m, _ := testkit.Drive(New(r, shell.Help, p), nil, testkit.Start(New(r, shell.Help, p),
		tea.WindowSizeMsg{Width: 120, Height: 30})...)
	return m
}

// typed gives the presses that type the text.
func typed(text string) []tea.Msg { return testkit.Typed(text) }

func TestAScreenIsMadeWhenFirstShownAndKeptAsItWasLeft(t *testing.T) {
	m := help(t)
	if _, made := m.made[shell.Flow]; made || !strings.Contains(ansi.Strip(m.View().Content), "kit help") {
		t.Fatalf("the help's palette, alone:\n%s", ansi.Strip(m.View().Content))
	}
	// the filter typed in the palette stays there while the picture is shown, and when the palette is shown again
	m, _ = testkit.Drive(m, nil, typed("/com")...)
	m, _ = testkit.Drive(m, nil, shell.SwitchMsg{Screen: shell.Flow})
	if m.here != shell.Flow || !strings.Contains(ansi.Strip(m.View().Content), "kit flow") {
		t.Fatalf("the picture is not shown:\n%s", ansi.Strip(m.View().Content))
	}
	if _, ok := m.Shown().(flow.Model); !ok {
		t.Errorf("the screen shown is %T", m.Shown())
	}
	m, _ = testkit.Drive(m, nil, tea.WindowSizeMsg{Width: 100, Height: 24}, shell.SwitchMsg{Screen: shell.Help})
	if v := ansi.Strip(m.View().Content); !strings.Contains(v, shell.Find+" com") || !strings.Contains(v, "search") {
		t.Errorf("the palette was not kept as it was left:\n%s", v)
	}
	// the palette shown again was told the terminal's size, as it changed while the picture was shown
	if lines := strings.Split(m.View().Content, "\n"); len(lines) != 24 {
		t.Errorf("the palette shown again has %d lines", len(lines))
	}
	// an answer for the hidden picture reaches it, not the palette
	m, _ = testkit.Drive(m, nil, tagged{shell.Flow, tea.WindowSizeMsg{Width: 70, Height: 20}})
	if lines := strings.Split(m.made[shell.Flow].View().Content, "\n"); len(lines) != 20 || m.here != shell.Help {
		t.Errorf("the hidden picture has %d lines, %s shown", len(lines), m.here)
	}
	// the welcome is made on its first showing too, and an ask for a screen kittui never made goes nowhere
	m, _ = testkit.Drive(m, nil, shell.SwitchMsg{Screen: shell.Welcome})
	if _, ok := m.Shown().(welcome.Model); !ok || len(m.made) != 3 {
		t.Errorf("the welcome: %T, %d made", m.Shown(), len(m.made))
	}
	if next, cmd := m.deliver("nonesuch", tea.WindowSizeMsg{}); cmd != nil || next.(Model).here != shell.Welcome {
		t.Error("a message for a screen never made")
	}
	m.Close()
}

// A screen kit does not have leaves the one shown, with kit's line on its last line until the next key.
func TestAScreenThatCannotOpenLeavesTheOneShown(t *testing.T) {
	m := help(t)
	m, _ = testkit.Drive(m, nil, shell.SwitchMsg{Screen: shell.Tutorials})
	lines := strings.Split(ansi.Strip(m.View().Content), "\n")
	if m.here != shell.Help || !strings.Contains(lines[len(lines)-1], "kit has no tutorials yet") {
		t.Fatalf("the refusal:\n%s", strings.Join(lines, "\n"))
	}
	if m, _ = testkit.Drive(m, nil, typed("x")...); strings.Contains(ansi.Strip(m.View().Content), "tutorials") {
		t.Error("the refusal stayed after a key")
	}
}

// A command's answer is tagged for the screen that started it, a batch's each in turn; nothing, a screen's ask for
// another, and what the program takes, as quitting, are not.
func TestWhatAScreenStartsIsTaggedForIt(t *testing.T) {
	type answer struct{}
	if tag(shell.Flow, nil) != nil {
		t.Error("no command was tagged")
	}
	for _, c := range []struct {
		cmd  tea.Cmd
		want tea.Msg
	}{
		{func() tea.Msg { return answer{} }, tagged{shell.Flow, answer{}}},
		{func() tea.Msg { return nil }, nil},
		{shell.Switch(shell.Help), shell.SwitchMsg{Screen: shell.Help}},
		{tea.Quit, tea.QuitMsg{}},
	} {
		if got := tag(shell.Flow, c.cmd)(); got != c.want {
			t.Errorf("tagged %#v, want %#v", got, c.want)
		}
	}
	batch := tag(shell.Flow, tea.Batch(func() tea.Msg { return answer{} }, func() tea.Msg { return answer{} }))()
	if cmds, ok := batch.(tea.BatchMsg); !ok || len(cmds) != 2 ||
		!slices.ContainsFunc(cmds, func(c tea.Cmd) bool { return c() == tagged{shell.Flow, answer{}} }) {
		t.Errorf("the batch: %#v", batch)
	}
}
