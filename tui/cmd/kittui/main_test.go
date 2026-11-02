// The binary refuses without a terminal, and ends as the form ended: silent on esc and ctrl-c, with the transcript
// before the verb on run; the picture and the guide end silent too, 0 on esc and 130 on ctrl-c.
package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/tui/internal/app"
	"github.com/empowerite/kit/tui/internal/flow"
	"github.com/empowerite/kit/tui/internal/testkit"
	"github.com/empowerite/kit/tui/internal/welcome"
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
		"welcome":   "❌ kit welcome needs an interactive terminal; run: kit welcome --text\n",
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
		{tea.KeyPressMsg{Code: tea.KeyEscape}, 0},
		{tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, 130},
	} {
		m, err := flow.Open(testkit.Runner(t, "flow"), nil)
		if err != nil {
			t.Fatal(err)
		}
		next, cmd := m.Update(c.key)
		if cmd == nil || cmd() != tea.Quit() {
			t.Errorf("%s did not quit", c.key)
		}
		if code := finishFlow(next.(flow.Model)); code != c.code {
			t.Errorf("%s: exit %d, want %d", c.key, code, c.code)
		}
	}
}

func TestTheGuideEndsZeroOnQuitAnd130OnCtrlC(t *testing.T) {
	for _, c := range []struct {
		key  tea.KeyPressMsg
		code int
	}{
		{tea.KeyPressMsg{Code: tea.KeyEscape}, 0},
		{tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, 130},
	} {
		next, cmd := welcome.Open(testkit.Runner(t, "welcome")).Update(c.key)
		if cmd == nil || cmd() != tea.Quit() {
			t.Errorf("%s did not quit", c.key)
		}
		if code := finishWelcome(next.(welcome.Model)); code != c.code {
			t.Errorf("%s: exit %d, want %d", c.key, code, c.code)
		}
	}
}

func TestARefusalOfTheStagesPrintsKitsLineWithItsCode(t *testing.T) {
	t.Setenv("FAKE_FLOW_REFUSE", "stages")
	_, err := flow.Open(testkit.Runner(t, "flow"), nil)
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
