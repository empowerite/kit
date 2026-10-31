// Package transcript writes what kit tui leaves in the scrollback when it runs a verb: the command line it built,
// quoted as a shell needs it, behind a runner, so the verb's own output follows the command that made it.
package transcript

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/empowerite/kit/tui/internal/form"
)

// The runners, taken in turn from one run to the next.
var runners = []string{"🏃‍♂️", "🏃‍♀️"}

// Render gives the transcript of a form that ran kit <verb> <args>…: the runner and the command line, on one line.
func Render(verb string, args []string, runner string) string {
	return runner + " " + strings.TrimSpace("kit "+verb+" "+form.Quote(args)) + "\n"
}

// Runner gives the runner for this run, the other one than last time, and remembers it in the file at state; with
// nowhere to remember, it gives the first.
func Runner(state string) string {
	last, _ := os.ReadFile(state)
	next := runners[0]
	if strings.TrimSpace(string(last)) == runners[0] {
		next = runners[1]
	}
	if err := os.MkdirAll(filepath.Dir(state), 0o755); err == nil {
		_ = os.WriteFile(state, []byte(next+"\n"), 0o644)
	}
	return next
}
