// Package transcript writes what kit tui leaves in the scrollback when it runs a verb: the command line it built,
// quoted as a shell needs it, so the verb's own output follows the command that made it.
package transcript

import (
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/form"
)

// Render gives the transcript of a form that ran kit <verb> <args>…: the command line alone, on one line.
func Render(verb string, args []string) string {
	return strings.TrimSpace("kit "+verb+" "+form.Quote(args)) + "\n"
}
