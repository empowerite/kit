// What kit tui leaves in the scrollback when it runs a verb: the command line alone, behind a runner.
package transcript

import (
	"path/filepath"
	"testing"
)

func TestTheTranscriptIsTheCommandLineBehindARunner(t *testing.T) {
	got := Render("commit", []string{"--message", "#1 tui: the first form", ".gitignore"}, "🏃‍♂️")
	if want := "🏃‍♂️ kit commit --message '#1 tui: the first form' .gitignore\n"; got != want {
		t.Errorf("transcript %q, want %q", got, want)
	}
	if got := Render("version", nil, "🏃‍♀️"); got != "🏃‍♀️ kit version\n" {
		t.Errorf("a verb with no words: %q", got)
	}
}

func TestTheRunnerAlternatesFromOneRunToTheNext(t *testing.T) {
	state := filepath.Join(t.TempDir(), "kit", "runner")
	first, second, third := Runner(state), Runner(state), Runner(state)
	if first == second || first != third || (first != "🏃‍♂️" && first != "🏃‍♀️") {
		t.Errorf("runners %q %q %q", first, second, third)
	}
	if got := Runner(filepath.Join("/nonexistent", "dir", "runner")); got != "🏃‍♂️" && got != "🏃‍♀️" {
		t.Errorf("a runner with nowhere to remember: %q", got)
	}
}
