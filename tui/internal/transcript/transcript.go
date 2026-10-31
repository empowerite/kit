// Package transcript writes what kit tui leaves in the scrollback when it runs a verb: one ✓ line per field, in the
// form's order, with the answer, then the command line quoted as a shell needs it. The verb's own output follows.
package transcript

import (
	"strings"

	"github.com/empowerite/kit/tui/internal/form"
	"github.com/empowerite/kit/tui/internal/style"
)

// Render gives the transcript of a form that ran kit <verb> <args>…. A flag left off says no, a value left empty
// none; a one-of row whose chosen alternative has fields of its own leaves them to say it, and one without says its
// choice.
func Render(verb string, fields []form.Field, args []string) string {
	type line struct{ label, value string }
	var lines []line
	for i, f := range fields {
		switch f.Kind {
		case form.Toggle:
			value := "no"
			if len(f.Values) > 0 {
				value = "yes"
			}
			lines = append(lines, line{f.Label, value})
		case form.AnyOf:
			value := strings.Join(f.Values, " ")
			if value == "" {
				value = "no"
			}
			lines = append(lines, line{f.Label, value})
		case form.OneOf:
			if owns(fields, i) {
				continue
			}
			if f.Chosen < 0 {
				lines = append(lines, line{f.Label, "none"})
			} else {
				lines = append(lines, line{f.Options[f.Chosen], "yes"})
			}
		default:
			value := strings.Join(f.Values, " ")
			if value == "" {
				value = "none"
			}
			lines = append(lines, line{f.Label, value})
		}
	}
	widest := 0
	for _, l := range lines {
		if n := len([]rune(l.label)); n > widest {
			widest = n
		}
	}
	var b strings.Builder
	for _, l := range lines {
		b.WriteString("  " + style.Answered + " " + l.label + strings.Repeat(" ", widest-len([]rune(l.label))+2) + l.value + "\n")
	}
	b.WriteString("\n  " + strings.TrimSpace("kit "+verb+" "+form.Quote(args)) + "\n\n")
	return b.String()
}

// owns says whether any field belongs to the row's chosen alternative.
func owns(fields []form.Field, row int) bool {
	for _, f := range fields {
		if f.Group == row {
			return true
		}
	}
	return false
}
