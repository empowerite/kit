// kittui's root shows one of kit's screens at a time: one is made only when it is first shown, and kept as it was left
// while another is shown; what a hidden screen started reaches it, and a screen that cannot open leaves the one shown.
package screens

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/app"
	"github.com/empowerite/kit/projects/tui/internal/flow"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
	"github.com/empowerite/kit/projects/tui/internal/tutorials"
	"github.com/empowerite/kit/projects/tui/internal/welcome"
)

// help gives the root on the help's palette, over the kit package's fake kit, at 120×30, its searches remembered in
// a directory of the test's own, never the person's.
func help(t *testing.T) Model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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
	if _, made := m.made[shell.Issues]; made || !strings.Contains(ansi.Strip(m.View().Content), "kit help") {
		t.Fatalf("the help's palette, alone:\n%s", ansi.Strip(m.View().Content))
	}
	// the filter typed in the palette stays there while the picture is shown, and when the palette is shown again
	m, _ = testkit.Drive(m, nil, typed("com")...)
	m, _ = testkit.Drive(m, nil, shell.SwitchMsg{Screen: shell.Issues})
	if m.here != shell.Issues || !strings.Contains(ansi.Strip(m.View().Content), "kit issues") {
		t.Fatalf("the picture is not shown:\n%s", ansi.Strip(m.View().Content))
	}
	if _, ok := m.Shown().(flow.Model); !ok {
		t.Errorf("the screen shown is %T", m.Shown())
	}
	// Repositories is a screen of its own, beside Issues, on kit workspace's stages
	m, _ = testkit.Drive(m, nil, shell.SwitchMsg{Screen: shell.Repositories})
	if v := ansi.Strip(m.View().Content); m.here != shell.Repositories || !strings.Contains(v, "kit repositories") ||
		!strings.Contains(v, "branches") {
		t.Fatalf("the workspace is not shown:\n%s", v)
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
	m, _ = testkit.Drive(m, nil, tagged{to: shell.Issues, msg: tea.WindowSizeMsg{Width: 70, Height: 20}})
	if lines := strings.Split(m.made[shell.Issues].View().Content, "\n"); len(lines) != 20 || m.here != shell.Help {
		t.Errorf("the hidden picture has %d lines, %s shown", len(lines), m.here)
	}
	// the welcome is made on its first showing too, beside the palette, the picture, and the workspace's, and an ask
	// for a screen kittui never made goes nowhere
	m, _ = testkit.Drive(m, nil, shell.SwitchMsg{Screen: shell.Welcome})
	if _, ok := m.Shown().(welcome.Model); !ok || len(m.made) != 4 {
		t.Errorf("the welcome: %T, %d made", m.Shown(), len(m.made))
	}
	if next, cmd := m.deliver("nonesuch", tea.WindowSizeMsg{}); cmd != nil || next.(Model).here != shell.Welcome {
		t.Error("a message for a screen never made")
	}
	m.Close()
}

// The menu's Tutorials opens the tutorials screen, made on its first showing.
func TestTheMenuOpensTheTutorials(t *testing.T) {
	m := help(t)
	m, _ = testkit.Drive(m, nil, shell.SwitchMsg{Screen: shell.Tutorials})
	if _, ok := m.Shown().(tutorials.Model); !ok || m.here != shell.Tutorials {
		t.Fatalf("the screen shown is %T, %s", m.Shown(), m.here)
	}
	m.Close()
}

// A jump to a row of another app shows that app, made on its first showing, and gives it the jump, which selects the
// tab the row is on, here the second; a jump to a screen that cannot open leaves the one shown.
func TestAJumpShowsTheAppAndGivesItTheJump(t *testing.T) {
	m := help(t)
	m, _ = testkit.Drive(m, nil, shell.JumpMsg{Screen: shell.Repositories, To: "workspace/remotes/origin"})
	if m.here != shell.Repositories {
		t.Fatalf("shown: %s", m.here)
	}
	if v := ansi.Strip(m.View().Content); !strings.Contains(v, "kit repositories · ") || !strings.Contains(v, "➋╶") {
		t.Errorf("the jump did not select remotes, the second tab:\n%s", v)
	}
	m, _ = testkit.Drive(m, nil, shell.JumpMsg{Screen: "nowhere", To: "x/y"})
	if m.here != shell.Repositories {
		t.Errorf("a jump to no screen shows %s", m.here)
	}
	m.Close()
}

// A screen kit does not have leaves the one shown, with kit's line on its last line until the next key.
func TestAScreenThatCannotOpenLeavesTheOneShown(t *testing.T) {
	m := help(t)
	m, _ = testkit.Drive(m, nil, shell.SwitchMsg{Screen: "nonesuch"})
	lines := strings.Split(ansi.Strip(m.View().Content), "\n")
	if m.here != shell.Help || !strings.Contains(lines[len(lines)-1], "kit has no nonesuch yet") {
		t.Fatalf("the refusal:\n%s", strings.Join(lines, "\n"))
	}
	if m, _ = testkit.Drive(m, nil, typed("x")...); strings.Contains(ansi.Strip(m.View().Content), "nonesuch") {
		t.Error("the refusal stayed after a key")
	}
}

// A command's answer is tagged for the screen that started it, a batch's each in turn; nothing, a screen's ask for
// another, and what the program takes, as quitting, are not.
func TestWhatAScreenStartsIsTaggedForIt(t *testing.T) {
	type answer struct{}
	if tag(shell.Issues, 0, nil) != nil {
		t.Error("no command was tagged")
	}
	for _, c := range []struct {
		cmd  tea.Cmd
		want tea.Msg
	}{
		{func() tea.Msg { return answer{} }, tagged{to: shell.Issues, msg: answer{}}},
		{func() tea.Msg { return nil }, nil},
		{shell.Switch(shell.Help), shell.SwitchMsg{Screen: shell.Help}},
		{tea.Quit, tea.QuitMsg{}},
	} {
		if got := tag(shell.Issues, 0, c.cmd)(); got != c.want {
			t.Errorf("tagged %#v, want %#v", got, c.want)
		}
	}
	batch := tag(shell.Issues, 0, tea.Batch(func() tea.Msg { return answer{} }, func() tea.Msg { return answer{} }))()
	if cmds, ok := batch.(tea.BatchMsg); !ok || len(cmds) != 2 ||
		!slices.ContainsFunc(cmds, func(c tea.Cmd) bool { return c() == tagged{to: shell.Issues, msg: answer{}} }) {
		t.Errorf("the batch: %#v", batch)
	}
}

// Settings… in the app's menu opens the dialog over the screen, which takes the keys: space hides the keys bar, which
// the screen's last line then shows no more, its height the same, and kittui's file keeps it hidden for the next
// screen, and the next run; space again shows it.
func TestTheSettingsDialogHidesTheKeysBarAndKeepsItHidden(t *testing.T) {
	m := help(t)
	bar := func(m Model) string {
		lines := strings.Split(ansi.Strip(m.View().Content), "\n")
		if len(lines) != 30 {
			t.Fatalf("%d lines", len(lines))
		}
		return lines[29]
	}
	// hints says whether the line holds key hints: the screen's, or the palette's search's, which has focus
	hints := func(line string) bool {
		return strings.Contains(line, "= menu") || strings.Contains(line, "type to search")
	}
	if !hints(bar(m)) {
		t.Fatalf("the keys bar is not shown: %q", bar(m))
	}
	m, _ = testkit.Drive(m, nil, shell.SwitchMsg{Screen: shell.SettingsScreen})
	if v := ansi.Strip(m.View().Content); !m.dialog.Open || !strings.Contains(v, "╭─ Settings ") ||
		bar(m) != " esc close ↑↓ move ␣ toggle" {
		t.Fatalf("the dialog is not open over the screen:\n%s", v)
	}
	m, _ = testkit.Drive(m, nil, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.dialog.Open || m.settings.KeysBar || hints(bar(m)) {
		t.Errorf("the keys bar was not hidden: open %v, shown %v, last line %q", m.dialog.Open, m.settings.KeysBar,
			bar(m))
	}
	if shell.LoadSettings(shell.SettingsFile()).KeysBar {
		t.Error("the file does not keep the keys bar hidden")
	}
	// the next screen shown has the line too, and so has the next run
	m, _ = testkit.Drive(m, nil, shell.SwitchMsg{Screen: shell.Welcome})
	if hints(bar(m)) {
		t.Errorf("the welcome draws the keys bar: %q", bar(m))
	}
	if New(m.runner, shell.Help, m.made[shell.Help]).settings.KeysBar {
		t.Error("the next run shows the keys bar")
	}
	m, _ = testkit.Drive(m, nil, shell.SwitchMsg{Screen: shell.SettingsScreen}, tea.KeyPressMsg{Code: tea.KeySpace,
		Text: " "}, tea.KeyPressMsg{Code: tea.KeyEscape})
	if !hints(bar(m)) {
		t.Errorf("the keys bar is not shown again: %q", bar(m))
	}
	m.Close()
}

// A click on the Settings dialog's setting flips it, as space does, and keeps the dialog open; a click on its edge
// changes nothing; a click outside it closes it, as esc does, and reaches nothing under it.
func TestAClickFlipsASettingAndOneOutsideTheDialogClosesIt(t *testing.T) {
	m := help(t)
	m, _ = testkit.Drive(m, nil, shell.SwitchMsg{Screen: shell.SettingsScreen})
	lines := strings.Split(ansi.Strip(m.View().Content), "\n")
	top := slices.IndexFunc(lines, func(l string) bool { return strings.Contains(l, "╭─ Settings ") })
	if top < 0 || !strings.Contains(lines[top+1], "Key hints") {
		t.Fatalf("the dialog:\n%s", strings.Join(lines, "\n"))
	}
	x := ansi.StringWidth(lines[top][:strings.Index(lines[top], "╭")])
	if m, _ = testkit.Drive(m, nil, testkit.Click(x+4, top+1)); !m.dialog.Open || m.settings.KeysBar {
		t.Errorf("a click on Key hints: open %v, shown %v", m.dialog.Open, m.settings.KeysBar)
	}
	if m, _ = testkit.Drive(m, nil, testkit.Click(x, top)); !m.dialog.Open || m.settings.KeysBar {
		t.Errorf("a click on the corner: open %v, shown %v", m.dialog.Open, m.settings.KeysBar)
	}
	if m, _ = testkit.Drive(m, nil, testkit.Click(0, 20)); m.dialog.Open || m.settings.KeysBar {
		t.Errorf("a click outside: open %v, shown %v", m.dialog.Open, m.settings.KeysBar)
	}
	m.Close()
}

// told records the settings a screen is told.
type told struct{ settings []shell.Settings }

func (t told) Init() tea.Cmd { return nil }

func (t told) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s, ok := msg.(shell.SettingsMsg); ok {
		t.settings = append(t.settings, s.Settings)
	}
	return t, nil
}

func (t told) View() tea.View { return tea.NewView("") }

// A screen is told kittui's settings when it is made, and again whenever the dialog flips one but the keys bar, which
// the root follows itself, so a table folds and draws as they say in every view.
func TestEveryScreenIsToldTheSettings(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(testkit.Runner(t, "kit"), shell.Issues, told{})
	if got := m.made[shell.Issues].(told).settings; len(got) != 1 || !got[0].FoldShared {
		t.Fatalf("made: %+v", got)
	}
	m, _ = testkit.Drive(m, nil, tea.WindowSizeMsg{Width: 120, Height: 30}, shell.SwitchMsg{Screen: shell.SettingsScreen},
		tea.KeyPressMsg{Code: tea.KeyDown}, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if got := m.made[shell.Issues].(told).settings; len(got) != 2 || got[1].FoldShared {
		t.Errorf("Auto-group flipped: %+v", got)
	}
	// the keys bar flipped is the root's own: the screen is told its size, not the settings
	m, _ = testkit.Drive(m, nil, tea.KeyPressMsg{Code: tea.KeyUp}, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if got := m.made[shell.Issues].(told).settings; len(got) != 2 {
		t.Errorf("the keys bar flipped told the screen: %+v", got)
	}
}

// heard records every message a screen is given.
type heard struct{ msgs *[]tea.Msg }

func (h heard) Init() tea.Cmd { return nil }

func (h heard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	*h.msgs = append(*h.msgs, msg)
	return h, nil
}

func (h heard) View() tea.View { return tea.NewView("") }

// got says whether a screen was given a message of the type of want.
func got[M tea.Msg](msgs []tea.Msg) bool {
	return slices.ContainsFunc(msgs, func(msg tea.Msg) bool { _, ok := msg.(M); return ok })
}

// What the terminal said of its keyboard reaches the screen shown, and every screen made after it said so.
func TestAScreenMadeLaterIsToldTheKeyboard(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var first, later []tea.Msg
	m := New(testkit.Runner(t, "kit"), shell.Issues, heard{&first})
	m = drive(m, tea.KeyboardEnhancementsMsg{Flags: 1})
	next, _ := m.place(shell.People, heard{&later})
	if !got[tea.KeyboardEnhancementsMsg](first) || !got[tea.KeyboardEnhancementsMsg](later) || next.(Model).here !=
		shell.People {
		t.Errorf("the keyboard: first %v, later %v", got[tea.KeyboardEnhancementsMsg](first),
			got[tea.KeyboardEnhancementsMsg](later))
	}
}

// While a dialog is open, a turn of the mouse wheel, or the mouse moving, reaches no screen; a click before the
// terminal said its size closes the Settings dialog, as one off it does.
func TestADialogKeepsTheMouseFromTheScreen(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var msgs []tea.Msg
	m := New(testkit.Runner(t, "kit"), shell.Issues, heard{&msgs})
	m = drive(m, shell.OpenMsg{}, testkit.Drag(3, 3), testkit.Press("esc"))
	m = drive(m, shell.SwitchMsg{Screen: shell.SettingsScreen}, tea.MouseWheelMsg{Button: tea.MouseWheelDown},
		testkit.Drag(3, 3))
	if got[tea.MouseWheelMsg](msgs) || got[tea.MouseMotionMsg](msgs) {
		t.Error("the mouse reached the screen under a dialog")
	}
	if m = drive(m, testkit.Click(119, 29)); m.dialog.Open {
		t.Error("a click with no size known kept the dialog open")
	}
	if m = drive(m, tea.MouseWheelMsg{Button: tea.MouseWheelDown}); !got[tea.MouseWheelMsg](msgs) {
		t.Error("the wheel, the dialog closed, did not reach the screen")
	}
}

// A screen shown again before the terminal said its size is shown as it was left, told nothing.
func TestAScreenShownAgainBeforeASizeIsToldNothing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var msgs []tea.Msg
	m := New(testkit.Runner(t, "kit"), shell.Issues, heard{&msgs})
	m = drive(m, shell.SwitchMsg{Screen: shell.Help})
	was := len(msgs)
	if m = drive(m, shell.SwitchMsg{Screen: shell.Issues}); m.here != shell.Issues || len(msgs) != was {
		t.Errorf("shown again: %s, told %#v", m.here, msgs[was:])
	}
	m.Close()
}

// noting is a screen that says what it did in place of its keys, on its last line, drawing as many lines as it is told.
type noting struct{ height int }

func (n noting) Init() tea.Cmd { return nil }

func (n noting) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		n.height = size.Height
	}
	return n, nil
}

func (n noting) View() tea.View {
	lines := make([]string, n.height)
	for k := range lines {
		lines[k] = "line " + strconv.Itoa(k)
	}
	lines[len(lines)-1] = "did it"
	return tea.NewView(strings.Join(lines, "\n"))
}

func (n noting) Noticing() bool { return true }

// With the keys bar hidden, what a screen did still stands on the last line, in place of the keys, the line above it
// dropped instead.
func TestWithTheKeysBarHiddenWhatAScreenDidStillStands(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(testkit.Runner(t, "kit"), shell.Issues, noting{})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 10}, shell.SwitchMsg{Screen: shell.SettingsScreen},
		tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}, tea.KeyPressMsg{Code: tea.KeyEscape})
	lines := strings.Split(ansi.Strip(m.View().Content), "\n")
	if len(lines) != 10 || lines[9] != "did it" || lines[8] != "line 8" {
		t.Errorf("%d lines:\n%s", len(lines), strings.Join(lines, "\n"))
	}
}

// kit flow's words, as kittui started with them, reach every app made later, and every app a pick makes anew.
func TestKitFlowsWordsReachAnAppOpenedLater(t *testing.T) {
	w := flow.Words{NoFetch: true, Owner: []string{"--mine"}}
	m := drive(help(t).Taking(w), shell.SwitchMsg{Screen: shell.Repositories})
	if got := m.made[shell.Repositories].(flow.Model).Words(); !reflect.DeepEqual(got, w) {
		t.Errorf("Repositories, opened later, takes %+v", got)
	}
	m, _, _ = workspaced(t, map[string]string{"current.json": repoJSON, "w.json": groupJSON})
	m = drive(m.Taking(w), shell.SwitchMsg{Screen: shell.Repositories}, shell.PickMsg{Workspace: "w"},
		shell.SwitchMsg{Screen: shell.Repositories})
	if got := m.made[shell.Repositories].(flow.Model).Words(); !reflect.DeepEqual(got, w) {
		t.Errorf("Repositories, made anew on w, takes %+v", got)
	}
}
