// kittui is kit's interactive terminal UI: kit welcome's guide, which bare kit opens too; the palette of verbs, which
// kit help opens; the form that finishes a verb's command line when the verb is short of its arguments; and kit flow's
// picture and kit workspace's.
// kit tui, kit flow, kit workspace, kit packages, and kit welcome exec it; it runs kit for everything it knows. In a
// clone kit does not trust yet, it asks the person to trust it first, as internal/trust's dialog does.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"

	"github.com/empowerite/kit/projects/tui/internal/app"
	"github.com/empowerite/kit/projects/tui/internal/fix"
	"github.com/empowerite/kit/projects/tui/internal/flow"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/screens"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/transcript"
	"github.com/empowerite/kit/projects/tui/internal/trust"
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

// run opens the guide for welcome or with no verb, the palette, or the palette on a verb's page, for help, the form of
// the verb named after --, an app for flow or an app verb, as workspace or audit, or the cards for tutorials, under
// kittui's root, whose app's menu switches between them, and ends as the person ended the screen shown.
func run(args []string, stderr io.Writer) int {
	verb, words := parse(args)
	if !terminals() {
		fmt.Fprintln(stderr, "❌ "+needsTerminal(verb))
		return 2
	}
	r := kit.New()
	code, on, untrusted := asked(r, stderr)
	if !on {
		return code
	}
	if untrusted {
		// what a clone kit does not trust allows: the palette, which reads kit help and kit syntax alone
		verb, words = "help", nil
	}
	if appVerb(verb) || verb == "welcome" {
		r.Group, words = workspaceOf(words)
	}
	first, m, err := open(r, verb, words)
	if err != nil {
		return refuse(err, stderr)
	}
	root := screens.New(r, first, m)
	if appVerb(verb) {
		// kit flow's words but the tab reach every app made later, as the first was made with them
		root = root.Taking(flow.ParseWords(words).Rest())
	}
	final, err := runModel(root)
	if err != nil {
		return failed(err, stderr)
	}
	root = final.(screens.Model)
	root.Close()
	if shown, ok := root.Shown().(app.Model); ok {
		return finish(shown, stderr, func(verb string, args []string) error { return execKit(r, verb, args) })
	}
	return ended(root.Shown())
}

// asked asks the person to trust the clone kittui runs in, where kit does not trust it yet, as it asks now, before
// kittui reads anything there. It goes on where kit trusts the clone, as it did or after Trust; it goes on untrusted
// after Not now, for kittui to open on what an untrusted clone allows, the palette of kit help; and it ends with 130 on
// ctrl-c. A kit that cannot answer is kit's refusal, or kittui's own, and ends it too.
func asked(r kit.Runner, stderr io.Writer) (code int, on, untrusted bool) {
	name, trusted, err := r.Trust()
	if err != nil {
		return refuse(err, stderr), false, false
	}
	if trusted {
		return 0, true, false
	}
	final, err := runModel(trust.Open(r, name))
	if err != nil {
		return failed(err, stderr), false, false
	}
	m := final.(trust.Model)
	if m.Interrupted() {
		return 130, false, false
	}
	return 0, true, !m.Trusted()
}

// open makes the screen kittui opens on: for flow, Flow, on the stage its words name; for an app verb, as workspace or
// audit, the app with a tab of the stage its words name, else the verb's own, Repositories or Trust, as flow.AppOf
// says, each taking kit flow's words, as flow.ParseWords reads them;
// the guide for welcome and with no verb; the cards for tutorials, with its --tag words and the tutorial named; the
// palette on a verb's page or a group for help; and, as the help's screen, the form of the verb named.
func open(r kit.Runner, verb string, words []string) (shell.Screen, tea.Model, error) {
	if appVerb(verb) {
		app, words := flow.AppOf(verb, words)
		m, err := flow.OpenApp(r, app, flow.ParseWords(words))
		return app, m, err
	}
	switch verb {
	case "welcome", "":
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

// appVerb says whether a verb opens one of kittui's apps: kit flow, and each app verb, as kit workspace, kit packages,
// and kit audit, as flow.AppVerb names them, each of which takes the workspace, before the verb, and has a text form
// for a picture off a terminal.
func appVerb(verb string) bool { return verb == "flow" || flow.AppVerb(verb) }

// workspaceOf takes the workspace kit flow or an app verb was given, --workspace=<w>, out of its words: the group the
// picture's reads span, which kit refuses to give empty, and the words without it.
func workspaceOf(words []string) (string, []string) {
	group, rest := "", []string{}
	for _, w := range words {
		if g, ok := strings.CutPrefix(w, "--workspace="); ok {
			group = g
			continue
		}
		rest = append(rest, w)
	}
	return group, rest
}

// failed says the program could not run, and ends with 1.
func failed(err error, stderr io.Writer) int {
	fmt.Fprintf(stderr, "❌ kittui %s: %v\n", version, err)
	return 1
}

// needsTerminal gives the line kittui prints without a terminal: the picture, the guide, and the tutorials have a text
// form, the rest a help page, each remedy with the cd to the folder kittui runs in, as fix.Here writes it.
func needsTerminal(verb string) string {
	if appVerb(verb) {
		return "kit " + verb + " needs an interactive terminal for its picture; run: " + fix.Here("kit "+verb+" --text")
	}
	if verb == "welcome" {
		return "kit welcome needs an interactive terminal; run: " + fix.Here("kit welcome --text")
	}
	if verb == "tutorials" {
		return "kit tutorials needs an interactive terminal; run: " + fix.Here("kit tutorials --text")
	}
	if verb == "help" {
		return "kit help needs an interactive terminal for its palette; run: " + fix.Here("kit help --text")
	}
	return "kit tui needs an interactive terminal; run: " + fix.Here("kit help tui")
}

// parse takes the verb and its words from [--] <verb> [<word>…]; no verb opens the welcome guide.
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

// ended ends as a screen without an ending of its own ended, the picture, the guide, the tutorials, or the
// workspaces' screen: 130 on ctrl-c, 0 on ctrl-q; nothing is printed either way.
func ended(m tea.Model) int {
	if i, ok := m.(interface{ Interrupted() bool }); ok && i.Interrupted() {
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
