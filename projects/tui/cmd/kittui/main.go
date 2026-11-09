// kittui is kit's interactive terminal UI: the palette of verbs that bare kit opens, the form that finishes a verb's
// command line when the verb is short of its arguments, kit flow's picture, and kit welcome's guide. kit tui, kit
// flow, and kit welcome exec it; it runs kit for everything it knows.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"

	"github.com/empowerite/kit/projects/tui/internal/app"
	"github.com/empowerite/kit/projects/tui/internal/flow"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/screens"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/transcript"
	"github.com/empowerite/kit/projects/tui/internal/tutorials"
	"github.com/empowerite/kit/projects/tui/internal/welcome"
)

// version is the release's, written by the release build.
var version = "unreleased"

// The seams the tests drive: whether stdin, stdout, and stderr are all terminals; what runs a model until the person
// ends it, a program on the terminal, with the options a test gives it in place of the terminal's streams; what puts
// kit in this process's place to run the command line; and how the process ends.
var (
	terminals = func() bool {
		return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) &&
			term.IsTerminal(int(os.Stderr.Fd()))
	}
	programOptions []tea.ProgramOption
	runModel       = func(m tea.Model) (tea.Model, error) { return tea.NewProgram(m, programOptions...).Run() }
	execKit        = func(r kit.Runner, verb string, args []string) error { return r.Exec(verb, args) }
	exit           = os.Exit
)

func main() { exit(run(os.Args[1:], os.Stderr)) }

// run opens the palette, the palette on a verb's page for help, the form of the verb named after --, the picture for
// flow, the guide for welcome, or the cards for tutorials, under kittui's root, whose app's menu switches between
// them, and ends as the person ended the screen shown.
func run(args []string, stderr io.Writer) int {
	verb, words := parse(args)
	if !terminals() {
		fmt.Fprintln(stderr, "❌ "+needsTerminal(verb))
		return 2
	}
	r := kit.New()
	if verb == "flow" {
		r.Group, words = workspaceOf(words)
	}
	first, m, err := open(r, verb, words)
	if err != nil {
		return refuse(err, stderr)
	}
	final, err := runModel(screens.New(r, first, m))
	if err != nil {
		return failed(err, stderr)
	}
	root := final.(screens.Model)
	root.Close()
	switch shown := root.Shown().(type) {
	case flow.Model:
		return finishFlow(shown)
	case welcome.Model:
		return finishWelcome(shown)
	case tutorials.Model:
		return finishTutorials(shown)
	}
	return finish(root.Shown().(app.Model), stderr, func(verb string, args []string) error { return execKit(r, verb, args) })
}

// open makes the screen kittui opens on: the picture for flow, with kit flow's words; the guide for welcome; the cards
// for tutorials, with its --tag words and the tutorial named; the palette on a verb's page or a group for help; and, as the help's screen, the
// palette, or the form of the verb named.
func open(r kit.Runner, verb string, words []string) (shell.Screen, tea.Model, error) {
	switch verb {
	case "flow":
		m, err := flow.Open(r, words)
		return shell.Flow, m, err
	case "welcome":
		return shell.Welcome, welcome.Open(r, kit.Workspace{}), nil
	case "tutorials":
		return shell.Tutorials, tutorials.Open(r, words), nil
	case "help":
		name := ""
		if len(words) > 0 {
			name = words[0]
		}
		m, err := app.OpenHelp(r, name)
		return shell.Help, m, err
	}
	m, err := app.Open(r, verb, words)
	return shell.Help, m, err
}

// workspaceOf takes the workspace kit flow was given, --workspace <w> or --workspace=<w>, out of its words: the group
// the picture's reads span, which kit refuses to give empty, and the words without it.
func workspaceOf(words []string) (string, []string) {
	group, rest := "", []string{}
	for k := 0; k < len(words); k++ {
		switch w := words[k]; {
		case w == "--workspace" && k+1 < len(words):
			group = words[k+1]
			k++
		case strings.HasPrefix(w, "--workspace="):
			group = strings.TrimPrefix(w, "--workspace=")
		default:
			rest = append(rest, w)
		}
	}
	return group, rest
}

// failed says the program could not run, and ends with 1.
func failed(err error, stderr io.Writer) int {
	fmt.Fprintf(stderr, "❌ kittui %s: %v\n", version, err)
	return 1
}

// needsTerminal gives the line kittui prints without a terminal: the picture, the guide, and the tutorials have a text
// form, the rest a help page.
func needsTerminal(verb string) string {
	if verb == "flow" {
		return "kit flow needs an interactive terminal for its picture; run: kit flow --text"
	}
	if verb == "welcome" {
		return "kit welcome needs an interactive terminal; run: kit welcome --text"
	}
	if verb == "tutorials" {
		return "kit tutorials needs an interactive terminal; run: kit tutorials --text"
	}
	if verb == "help" {
		return "kit help needs an interactive terminal for its palette; run: kit help --text"
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

// finishTutorials ends as the tutorials ended: 130 on ctrl-c, 0 on ctrl-q; nothing is printed either way.
func finishTutorials(m tutorials.Model) int {
	if m.Interrupted() {
		return 130
	}
	return 0
}

// finishWelcome ends as the guide ended: 130 on ctrl-c, 0 on ctrl-q; nothing is printed either way.
func finishWelcome(m welcome.Model) int {
	if m.Interrupted() {
		return 130
	}
	return 0
}

// finishFlow ends as the picture ended: 130 on ctrl-c, 0 on ctrl-q; nothing is printed either way.
func finishFlow(m flow.Model) int {
	if m.Interrupted() {
		return 130
	}
	return 0
}

// finish ends as the screen ended: 130 and nothing printed on ctrl-c; 0 and nothing printed when ctrl-q leaves the
// palette, as kit help ends; 1 and nothing printed when ctrl-q cancels a form; and on run the command line to stderr, and
// the verb exec'd in its place, where the command was typed.
func finish(m app.Model, stderr io.Writer, exec func(verb string, args []string) error) int {
	if m.Interrupted() {
		return 130
	}
	if !m.InForm() {
		return 0
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
