// kittui is kit's interactive terminal UI: the palette of verbs that bare kit opens, and the form that finishes a
// verb's command line when the verb is short of its arguments. kit tui execs it; it runs kit for everything it knows.
package main

import (
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"

	"github.com/empowerite/kit/tui/internal/app"
	"github.com/empowerite/kit/tui/internal/kit"
	"github.com/empowerite/kit/tui/internal/transcript"
)

// version is the release's, written by the release build.
var version = "unreleased"

func main() { os.Exit(run(os.Args[1:])) }

// run opens the palette, or the form of the verb named after --, and ends as the person ended it.
func run(args []string) int {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) || !term.IsTerminal(int(os.Stderr.Fd())) {
		fmt.Fprintln(os.Stderr, "❌ kit tui needs an interactive terminal; run: kit help tui")
		return 2
	}
	r := kit.New()
	verb, words := parse(args)
	m, err := app.Open(r, verb, words)
	if err != nil {
		if line, code, ok := app.Refusal(err); ok {
			fmt.Fprintln(os.Stderr, line)
			return code
		}
		fmt.Fprintf(os.Stderr, "❌ kittui %s: %v\n", version, err)
		return 1
	}
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ kittui %s: %v\n", version, err)
		return 1
	}
	return finish(final.(app.Model), os.Stderr, r.Exec)
}

// parse takes the verb and its words from [--] <verb> [<word>…]; no verb opens the palette.
func parse(args []string) (string, []string) {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		return "", nil
	}
	return args[0], args[1:]
}

// finish ends as the form ended: 130 and nothing printed on ctrl-c, 1 and nothing printed on esc, and on run the
// transcript to stderr and the verb exec'd in its place, where the command was typed.
func finish(m app.Model, stderr io.Writer, exec func(verb string, args []string) error) int {
	if m.Interrupted() {
		return 130
	}
	ran, verb, args := m.Result()
	if !ran {
		return 1
	}
	fmt.Fprint(stderr, transcript.Render(verb, m.Fields(), args))
	if err := exec(verb, args); err != nil {
		fmt.Fprintf(stderr, "❌ kittui could not run kit %s: %v\n", verb, err)
		return 1
	}
	return 0
}
