// Package transcript writes what kit tui leaves in the scrollback when it runs a verb: the command line it built,
// quoted as a shell needs it, behind a runner, so the verb's own output follows the command that made it.
package transcript

import (
	"strings"

	"github.com/empowerite/kit/tui/internal/form"
)

// Runner marks the command line: the plain runner, one code point, which every terminal draws at its width.
const Runner = "🏃"

// Render gives the transcript of a form that ran kit <verb> <args>…: the runner and the command line, on one line.
func Render(verb string, args []string) string {
	return Runner + " " + strings.TrimSpace("kit "+verb+" "+form.Quote(args)) + "\n"
}
