// kittui is kit's interactive terminal UI: the palette of verbs that bare kit opens, the form that finishes a verb's
// command line when the verb is short of its arguments, and kit status's picture. kit tui and kit status exec it; it
// runs kit for everything it knows.
package main

import (
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"

	"github.com/empowerite/kit/tui/internal/app"
	"github.com/empowerite/kit/tui/internal/kit"
	"github.com/empowerite/kit/tui/internal/status"
	"github.com/empowerite/kit/tui/internal/transcript"
)

// version is the release's, written by the release build.
var version = "unreleased"

func main() { os.Exit(run(os.Args[1:])) }

// run opens the palette, the form of the verb named after --, or the picture for status, and ends as the person
// ended it.
func run(args []string) int {
	verb, words := parse(args)
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) || !term.IsTerminal(int(os.Stderr.Fd())) {
		fmt.Fprintln(os.Stderr, "❌ "+needsTerminal(verb))
		return 2
	}
	r := kit.New()
	if verb == "status" {
		return runStatus(r, words)
	}
	m, err := app.Open(r, verb, words)
	if err != nil {
		return refuse(err, os.Stderr)
	}
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ kittui %s: %v\n", version, err)
		return 1
	}
	return finish(final.(app.Model), os.Stderr, r.Exec)
}

// needsTerminal gives the line kittui prints without a terminal: the picture has a text form, the rest a help page.
func needsTerminal(verb string) string {
	if verb == "status" {
		return "kit status needs an interactive terminal for its picture; run: kit status --text"
	}
	return "kit tui needs an interactive terminal; run: kit help tui"
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

// refuse prints why kittui did not open: kit's line, ending with kit's code, or kittui's own, ending with 1.
func refuse(err error, stderr io.Writer) int {
	if line, code, ok := app.Refusal(err); ok {
		fmt.Fprintln(stderr, "❌ "+line)
		return code
	}
	fmt.Fprintf(stderr, "❌ kittui %s: %v\n", version, err)
	return 1
}

// runStatus opens the picture on kit status's stages, with the words kit status was given, and ends as the person
// ended it.
func runStatus(r kit.Runner, words []string) int {
	m, err := status.Open(r, words)
	if err != nil {
		return refuse(err, os.Stderr)
	}
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ kittui %s: %v\n", version, err)
		return 1
	}
	return finishStatus(final.(status.Model))
}

// finishStatus ends as the picture ended: 130 on ctrl-c, 0 on q or esc; nothing is printed either way.
func finishStatus(m status.Model) int {
	if m.Interrupted() {
		return 130
	}
	return 0
}

// finish ends as the form ended: 130 and nothing printed on ctrl-c, 1 and nothing printed on esc, and on run the
// command line to stderr, and the verb exec'd in its place, where the command was typed.
func finish(m app.Model, stderr io.Writer, exec func(verb string, args []string) error) int {
	if m.Interrupted() {
		return 130
	}
	ran, verb, args := m.Result()
	if !ran {
		return 1
	}
	fmt.Fprint(stderr, transcript.Render(verb, args))
	if err := exec(verb, args); err != nil {
		fmt.Fprintf(stderr, "❌ kittui could not run kit %s: %v\n", verb, err)
		return 1
	}
	return 0
}
