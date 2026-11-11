// The binary refuses without a terminal, and ends as the form ended: silent on esc and ctrl-c, with the transcript
// before the verb on run; the picture and the guide end silent too, 0 on ctrl-q and 130 on ctrl-c.
package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/app"
	"github.com/empowerite/kit/projects/tui/internal/flow"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
	"github.com/empowerite/kit/projects/tui/internal/tutorials"
	"github.com/empowerite/kit/projects/tui/internal/welcome"
	"github.com/empowerite/kit/projects/tui/internal/workspaces"
)

// drive sends the messages to the root model from its start, as the program would. The fake kit is the kit package's
// for the palette and the form, the flow package's for the picture.
func drive(m app.Model, msgs ...tea.Msg) app.Model {
	m, _ = testkit.Drive(m, nil, testkit.Start(m, msgs...)...)
	return m
}

func TestNoTerminalRefuses(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "kittui")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	for args, want := range map[string]string{
		"-- commit": "❌ kit tui needs an interactive terminal; run: kit help tui\n",
		"flow":      "❌ kit flow needs an interactive terminal for its picture; run: kit flow --text\n",
		"workspace": "❌ kit workspace needs an interactive terminal for its picture; run: kit workspace --text\n",
		"packages":  "❌ kit packages needs an interactive terminal for its picture; run: kit packages --text\n",
		"welcome":   "❌ kit welcome needs an interactive terminal; run: kit welcome --text\n",
		"tutorials": "❌ kit tutorials needs an interactive terminal; run: kit tutorials --text\n",
	} {
		cmd := exec.Command(binary, strings.Fields(args)...)
		cmd.Stdin = nil
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err := cmd.Run()
		exit, ok := err.(*exec.ExitError)
		if !ok || exit.ExitCode() != 2 || stderr.String() != want {
			t.Errorf("%s: err %v, stderr %q", args, err, stderr.String())
		}
	}
}

func TestThePictureEndsZeroOnQuitAnd130OnCtrlC(t *testing.T) {
	for _, c := range []struct {
		key  tea.KeyPressMsg
		code int
	}{
		{tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl}, 0},
		{tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, 130},
	} {
		m, err := flow.OpenApp(testkit.Runner(t, "flow"), shell.Flow, nil)
		if err != nil {
			t.Fatal(err)
		}
		next, cmd := m.Update(c.key)
		if cmd == nil || cmd() != tea.Quit() {
			t.Errorf("%s did not quit", c.key)
		}
		if code := ended(next.(flow.Model)); code != c.code {
			t.Errorf("%s: exit %d, want %d", c.key, code, c.code)
		}
	}
}

func TestTheGuideEndsZeroOnQuitAnd130OnCtrlC(t *testing.T) {
	for _, c := range []struct {
		key  tea.KeyPressMsg
		code int
	}{
		{tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl}, 0},
		{tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, 130},
	} {
		next, cmd := welcome.Open(testkit.Runner(t, "welcome"), kit.Workspace{}).Update(c.key)
		if cmd == nil || cmd() != tea.Quit() {
			t.Errorf("%s did not quit", c.key)
		}
		if code := ended(next.(welcome.Model)); code != c.code {
			t.Errorf("%s: exit %d, want %d", c.key, code, c.code)
		}
	}
}

func TestARefusalOfTheStagesPrintsKitsLineWithItsCode(t *testing.T) {
	t.Setenv("FAKE_FLOW_REFUSE", "stages")
	_, err := flow.OpenApp(testkit.Runner(t, "flow"), shell.Flow, nil)
	var stderr bytes.Buffer
	if code := refuse(err, &stderr); code != 2 || stderr.String() != "❌ kit flow refuses for the test; run: kit help flow\n" {
		t.Errorf("exit %d, stderr %q", code, stderr.String())
	}
	// an error that is not kit's is kittui's own, ending with 1
	stderr.Reset()
	if code := refuse(errors.New("boom"), &stderr); code != 1 || stderr.String() != "❌ kittui unreleased: boom\n" {
		t.Errorf("exit %d, stderr %q", code, stderr.String())
	}
}

func TestCtrlCIsSilent(t *testing.T) {
	for _, verb := range []string{"", "commit"} {
		m, err := app.Open(testkit.Runner(t, "kit"), verb, nil)
		if err != nil {
			t.Fatal(err)
		}
		m = drive(m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		var stderr bytes.Buffer
		if code := finish(m, &stderr, nil); code != 130 || stderr.Len() != 0 {
			t.Errorf("%q: exit %d, stderr %q", verb, code, stderr.String())
		}
	}
}

func TestEscIsSilent(t *testing.T) {
	m, _ := app.Open(testkit.Runner(t, "kit"), "commit", nil)
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	var stderr bytes.Buffer
	if code := finish(m, &stderr, nil); code != 1 || stderr.Len() != 0 {
		t.Errorf("exit %d, stderr %q", code, stderr.String())
	}
}

// Esc leaves the palette as kit help ends, with 0, even after a form opened from it went back: only a form cancelled
// ends with 1.
func TestEscFromThePaletteEndsZero(t *testing.T) {
	for _, keys := range [][]tea.Msg{
		{tea.KeyPressMsg{Code: tea.KeyEscape}},
		{tea.KeyPressMsg{Code: tea.KeyEnter}, tea.KeyPressMsg{Code: tea.KeyEscape}, tea.KeyPressMsg{Code: tea.KeyEscape}},
	} {
		m, _ := app.OpenHelp(testkit.Runner(t, "kit"), "commit")
		m = drive(m, keys...)
		var stderr bytes.Buffer
		if code := finish(m, &stderr, nil); code != 0 || stderr.Len() != 0 {
			t.Errorf("after %d keys: exit %d, stderr %q", len(keys), code, stderr.String())
		}
	}
}

func TestAVerbThatCannotRunIsKittuisLine(t *testing.T) {
	m, _ := app.Open(testkit.Runner(t, "kit"), "commit", []string{"--message", "hi"})
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift})
	var stderr bytes.Buffer
	code := finish(m, &stderr, func(string, []string) error { return errors.New("no such file") })
	if code != 1 || !strings.Contains(stderr.String(), "❌ kittui could not run kit commit: no such file") {
		t.Errorf("exit %d, stderr %q", code, stderr.String())
	}
}

// ending gives a stand-in for the program: it drives the model from its start at 120×30, then sends the key, as a
// person would, and gives the model the program would end with.
func ending(key tea.KeyPressMsg) func(tea.Model) (tea.Model, error) {
	return func(m tea.Model) (tea.Model, error) {
		m, _ = testkit.Drive(m, nil, testkit.Start(m, tea.WindowSizeMsg{Width: 120, Height: 30}, key)...)
		return m, nil
	}
}

// onTerminal makes kittui believe it runs on a terminal, over the kit package's fake kit, with the program stood in
// for by run, and kit's exec by a record of what it would run, which it gives; it puts the seams back when the test
// ends.
func onTerminal(t *testing.T, run func(tea.Model) (tea.Model, error)) *[]string {
	t.Helper()
	terminals0, runModel0, execKit0 := terminals, runModel, execKit
	t.Cleanup(func() { terminals, runModel, execKit = terminals0, runModel0, execKit0 })
	ran := &[]string{}
	terminals, runModel = func() bool { return true }, run
	execKit = func(_ kit.Runner, verb string, args []string) error {
		*ran = append([]string{verb}, args...)
		return nil
	}
	t.Setenv("KIT", testkit.Runner(t, "kit").Kit)
	t.Setenv("kit_invoked_in", t.TempDir())
	t.Setenv("FAKE_KIT_RECORD", filepath.Join(t.TempDir(), "record"))
	return ran
}

// On a terminal, each screen opens and ends as the person ended it: with 0 on ctrl-q, a form with 1, and every one of
// them with 130 on ctrl-c.
func TestEveryScreenOpensOnATerminalAndEndsAsThePersonEndedIt(t *testing.T) {
	esc, ctrlC := tea.KeyPressMsg{Code: tea.KeyEscape}, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	ctrlQ := tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl}
	for _, c := range []struct {
		args       []string
		key        tea.KeyPressMsg
		code       int
		stderrHas  string
		stderrNone bool
	}{
		{nil, ctrlQ, 0, "", true},
		{[]string{"help"}, ctrlQ, 0, "", true},
		{[]string{"help", "commit"}, ctrlQ, 0, "", true},
		{[]string{"--", "commit"}, ctrlQ, 1, "", true},
		{[]string{"--", "commit", "--message", "hi"}, tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift}, 0,
			"kit commit ", false},
		{[]string{"flow"}, ctrlQ, 0, "", true},
		{[]string{"welcome"}, ctrlQ, 0, "", true},
		{nil, ctrlC, 130, "", true},
		{[]string{"flow"}, ctrlC, 130, "", true},
		{[]string{"welcome"}, ctrlC, 130, "", true},
		{[]string{"--", "nonesuch"}, esc, 2, "❌ no kit verb 'nonesuch'; run: kit help", false},
	} {
		ran := onTerminal(t, ending(c.key))
		var stderr bytes.Buffer
		code := run(c.args, &stderr)
		if code != c.code || c.stderrNone && stderr.Len() != 0 || !strings.Contains(stderr.String(), c.stderrHas) {
			t.Errorf("kittui %q, then %s: exit %d, stderr %q", c.args, c.key, code, stderr.String())
		}
		if run := c.stderrHas == "kit commit "; run != (len(*ran) > 0 && (*ran)[0] == "commit") {
			t.Errorf("kittui %q, then %s: kit ran %q", c.args, c.key, *ran)
		}
	}
}

// A program that cannot run on the terminal is kittui's own line, ending with 1, on every screen; a picture whose
// stages kit refuses is kit's line with its code.
func TestAProgramThatCannotRunIsKittuisLine(t *testing.T) {
	for _, args := range [][]string{nil, {"flow"}, {"welcome"}, {"tutorials"}} {
		onTerminal(t, func(tea.Model) (tea.Model, error) { return nil, errors.New("no tty") })
		var stderr bytes.Buffer
		if code := run(args, &stderr); code != 1 || stderr.String() != "❌ kittui unreleased: no tty\n" {
			t.Errorf("kittui %q: exit %d, stderr %q", args, code, stderr.String())
		}
	}
	onTerminal(t, ending(tea.KeyPressMsg{Code: tea.KeyEscape}))
	t.Setenv("KIT", testkit.Runner(t, "flow").Kit)
	t.Setenv("FAKE_FLOW_REFUSE", "stages")
	var stderr bytes.Buffer
	if code := run([]string{"flow"}, &stderr); code != 2 || !strings.Contains(stderr.String(), "refuses for the test") {
		t.Errorf("kittui flow over a refusal: exit %d, stderr %q", code, stderr.String())
	}
}

// quitter is a model that ends as soon as it starts.
type quitter struct{}

func (quitter) Init() tea.Cmd                         { return tea.Quit }
func (q quitter) Update(tea.Msg) (tea.Model, tea.Cmd) { return q, nil }
func (quitter) View() tea.View                        { return tea.NewView("") }

// The seams' own defaults: the program runs a model to its end over the streams it is given, kit's exec fails without
// replacing the process when there is no kit to run, and main ends the process with run's code.
func TestTheSeamsDefaultsRunTheProgramExecKitAndEndTheProcess(t *testing.T) {
	programOptions = []tea.ProgramOption{tea.WithInput(strings.NewReader("")), tea.WithOutput(io.Discard)}
	t.Cleanup(func() { programOptions = nil })
	if m, err := runModel(quitter{}); err != nil || m == nil {
		t.Errorf("runModel(quitter) = %v, %v", m, err)
	}
	if err := execKit(kit.Runner{Kit: filepath.Join(t.TempDir(), "no-kit"), Dir: t.TempDir()}, "commit", nil); err == nil {
		t.Error("exec of a kit that is not there did not fail")
	}
	exit0, args0 := exit, os.Args
	t.Cleanup(func() { exit, os.Args = exit0, args0 })
	code := -1
	exit, os.Args = func(c int) { code = c }, []string{"kittui", "--", "commit"}
	main()
	if code != 2 {
		t.Errorf("main off a terminal ended with %d, not 2", code)
	}
}

func TestOffATerminalEachScreenNamesItsTextForm(t *testing.T) {
	for args, want := range map[string]string{
		"welcome":   "kit welcome needs an interactive terminal; run: kit welcome --text",
		"help":      "kit help needs an interactive terminal for its palette; run: kit help --text",
		"":          "kit tui needs an interactive terminal; run: kit help tui",
		"flow":      "kit flow needs an interactive terminal for its picture; run: kit flow --text",
		"workspace": "kit workspace needs an interactive terminal for its picture; run: kit workspace --text",
		"packages":  "kit packages needs an interactive terminal for its picture; run: kit packages --text",
		"tutorials": "kit tutorials needs an interactive terminal; run: kit tutorials --text",
		"-- commit": "kit tui needs an interactive terminal; run: kit help tui",
	} {
		var stderr bytes.Buffer
		if code := run(strings.Fields(args), &stderr); code != 2 || stderr.String() != "❌ "+want+"\n" {
			t.Errorf("%q: exit %d, stderr %q", args, code, stderr.String())
		}
	}
}

func TestRunPrintsTheTranscriptThenExecs(t *testing.T) {
	m, _ := app.Open(testkit.Runner(t, "kit"), "commit", []string{"--message", "hi"})
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift})
	var stderr bytes.Buffer
	var ran []string
	code := finish(m, &stderr, func(verb string, args []string) error {
		ran = append([]string{verb}, args...)
		return nil
	})
	if code != 0 || len(ran) == 0 || ran[0] != "commit" {
		t.Fatalf("exit %d, ran %q", code, ran)
	}
	if !strings.HasPrefix(stderr.String(), "kit commit ") || strings.Contains(stderr.String(), "✓") {
		t.Errorf("transcript %q", stderr.String())
	}
}

func TestWordsAfterADoubleDash(t *testing.T) {
	verb, words := parse([]string{"--", "commit", "--all"})
	if verb != "commit" || strings.Join(words, " ") != "--all" {
		t.Errorf("parse = %q %q", verb, words)
	}
	verb, words = parse(nil)
	if verb != "" || words != nil {
		t.Errorf("parse(nil) = %q %q", verb, words)
	}
	// flow takes its words with or without the double dash
	for _, args := range [][]string{{"flow", "--no-fetch", "--mine"}, {"--", "flow", "--no-fetch", "--mine"}} {
		if verb, words = parse(args); verb != "flow" || strings.Join(words, " ") != "--no-fetch --mine" {
			t.Errorf("parse(%q) = %q %q", args, verb, words)
		}
	}
	_ = os.Stdin
}

// The words after tutorials are kit tutorials' own, which the screen reads.
func TestTutorialsOpensItsScreen(t *testing.T) {
	r := testkit.Runner(t, "tutorials")
	for key, code := range map[string]int{"ctrl-q": 0, "ctrl-c": 130} {
		k := tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl}
		if code == 130 {
			k = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
		}
		onTerminal(t, ending(k))
		t.Setenv("KIT", r.Kit)
		var stderr bytes.Buffer
		if got := run([]string{"tutorials", "--tag", "x", "stray"}, &stderr); got != code || stderr.Len() != 0 {
			t.Errorf("%s: exit %d, stderr %q", key, got, stderr.String())
		}
	}
	screen, m, err := open(r, "tutorials", []string{"--tag", "x"})
	if _, ok := m.(tutorials.Model); err != nil || !ok || screen != shell.Tutorials {
		t.Errorf("open: %s, %T, %v", screen, m, err)
	}
	m.(tutorials.Model).Close()
}

// No verb, as bare kit runs kittui, opens the welcome guide, as kit welcome does; kit help opens the palette.
func TestNoVerbOpensTheWelcomeGuide(t *testing.T) {
	r := testkit.Runner(t, "kit")
	for _, verb := range []string{"", "welcome"} {
		if screen, m, err := open(r, verb, nil); err != nil || screen != shell.Welcome {
			t.Errorf("%q: %s, %T, %v", verb, screen, m, err)
		} else if _, ok := m.(welcome.Model); !ok {
			t.Errorf("%q: %T", verb, m)
		}
	}
	if screen, m, err := open(r, "help", nil); err != nil || screen != shell.Help {
		t.Errorf("help: %s, %T, %v", screen, m, err)
	} else if _, ok := m.(app.Model); !ok {
		t.Errorf("help: %T", m)
	}
}

// kit flow opens Flow, on the stage its words name, its desk too; kit workspace opens the app with the tab its words
// name, maximized on it, and with no tab named, Repositories.
func TestFlowAndWorkspaceOpenTheAppWithTheTabNamed(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	r := testkit.Runner(t, "flow")
	for _, c := range []struct {
		verb  string
		words []string
		want  shell.Screen
	}{
		{"flow", nil, shell.Flow}, {"flow", []string{"changes"}, shell.Flow}, {"flow", []string{"desk"}, shell.Flow},
		{"workspace", nil, shell.Repositories}, {"workspace", []string{"branches"}, shell.Repositories},
		{"packages", nil, shell.Packages}, {"packages", []string{"packages"}, shell.Packages},
	} {
		screen, m, err := open(r, c.verb, c.words)
		if _, ok := m.(flow.Model); err != nil || !ok || screen != c.want {
			t.Errorf("kit %s %q: %s, %T, %v", c.verb, c.words, screen, m, err)
			continue
		}
		m.(flow.Model).Close()
	}
}

func TestFlowTakesTheWorkspaceFromItsWords(t *testing.T) {
	for _, words := range [][]string{{"--workspace", "w", "--no-fetch"}, {"--workspace=w", "--no-fetch"},
		{"--no-fetch", "--workspace", "w"}} {
		group, rest := workspaceOf(words)
		if group != "w" || strings.Join(rest, " ") != "--no-fetch" {
			t.Errorf("%q gives %q and %q", words, group, rest)
		}
	}
	if group, rest := workspaceOf([]string{"--no-fetch"}); group != "" || len(rest) != 1 {
		t.Errorf("no workspace gives %q and %q", group, rest)
	}
}

// interrupted is a screen ctrl-c ended.
type interrupted struct{ tea.Model }

func (interrupted) Interrupted() bool { return true }

// A screen without an ending of its own, as the workspaces' screen, ends 0, or 130 where ctrl-c ended it, and never
// panics on its type.
func TestEveryOtherScreenEndsZeroOr130(t *testing.T) {
	if code := ended(workspaces.Model{}); code != 0 {
		t.Errorf("the workspaces' screen: %d", code)
	}
	if code := ended(interrupted{}); code != 130 {
		t.Errorf("a screen ctrl-c ended: %d", code)
	}
}
