// Package cli holds the command line, from the commands' data. It may import doc, plan, and data.
//
// Each command's words, options, arguments, and help are data, in kit's commands.yml. From them come the parsing,
// the usage line, and the help. Every command takes --json, which prints what it says as JSON with a schema and a
// version, and --help.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/empowerite/kit/internal/data"
)

// Command is one command of kit's: the words that name it, what it does in a line and in full, and what it takes.
type Command struct {
	Words   []string `yaml:"words"`
	Summary string   `yaml:"summary"`
	Help    string   `yaml:"help"`
	Options []Option `yaml:"options"`
	Args    []Arg    `yaml:"args"`
}

// Option is one option of a command's: --name=<value>, or --name alone where it has no value. A many option may be
// given more than once.
type Option struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
	Help  string `yaml:"help"`
	Many  bool   `yaml:"many"`
}

// Arg is one argument of a command's, by the name its usage gives it.
type Arg struct {
	Name     string `yaml:"name"`
	Many     bool   `yaml:"many"`
	Optional bool   `yaml:"optional"`
}

// Call is a command as it was given: its options by name, each with every value given, and its arguments.
type Call struct {
	Command Command
	Options map[string][]string
	Args    []string
	JSON    bool // --json: say it as JSON
	Help    bool // --help: say the command's help, and do nothing
}

// Out is where a command says what it says, and whether to say it as JSON.
type Out struct {
	Stdout, Stderr io.Writer
	JSON           bool
}

// Handler does what a call asks, and gives the exit status.
type Handler func(ctx context.Context, call Call, out Out) int

// ErrNoCommand is what Parse gives when it is given no word at all.
var ErrNoCommand = errors.New("kit: no command")

// Commands gives kit's commands, from its data.
func Commands() ([]Command, error) {
	text, _ := data.Show("commands")
	var file struct {
		Commands []Command `yaml:"commands"`
	}
	err := data.Decode(data.Source{Name: "commands.yml", Text: text}, "commands", &file)
	return file.Commands, err
}

// Parse finds the command that args name, with the longest run of words that names one, and takes the rest as its
// options and arguments. An option with a value is --name=value, and nothing else: what follows -- is arguments all.
func Parse(commands []Command, args []string) (Call, error) {
	if len(args) == 0 {
		return Call{}, ErrNoCommand
	}
	var call Call
	found := -1
	for i, c := range commands {
		if len(c.Words) <= len(args) && slices.Equal(c.Words, args[:len(c.Words)]) &&
			(found < 0 || len(c.Words) > len(commands[found].Words)) {
			found = i
		}
	}
	if found < 0 {
		return Call{}, fmt.Errorf("kit: no such command: %s", args[0])
	}
	call.Command = commands[found]
	call.Options = map[string][]string{}
	refuse := func(format string, a ...any) (Call, error) {
		words := "kit " + strings.Join(call.Command.Words, " ")
		return Call{}, fmt.Errorf("%s: %s\n%s", words, fmt.Sprintf(format, a...), Usage(call.Command))
	}
	rest := args[len(call.Command.Words):]
	for i := 0; i < len(rest); i++ {
		arg := rest[i]
		switch {
		case arg == "--":
			call.Args = append(call.Args, rest[i+1:]...)
			i = len(rest)
		case arg == "--json":
			call.JSON = true
		case arg == "--help":
			call.Help = true
		case strings.HasPrefix(arg, "--"):
			name, value, given := strings.Cut(arg[2:], "=")
			at := slices.IndexFunc(call.Command.Options, func(o Option) bool { return o.Name == name })
			if at < 0 {
				return refuse("no such option: --%s", name)
			}
			option := call.Command.Options[at]
			switch {
			case option.Value == "" && given:
				return refuse("--%s takes no value", name)
			case option.Value != "" && (!given || value == ""):
				return refuse("give it as --%s=<%s>", name, option.Value)
			case !option.Many && len(call.Options[name]) > 0:
				return refuse("--%s is given twice", name)
			}
			call.Options[name] = append(call.Options[name], value)
		default:
			call.Args = append(call.Args, arg)
		}
	}
	if call.Help {
		return call, nil
	}
	taken := 0
	for _, a := range call.Command.Args {
		switch {
		case taken == len(call.Args) && !a.Optional:
			return refuse("<%s> is missing", a.Name)
		case a.Many:
			taken = len(call.Args)
		case taken < len(call.Args):
			taken++
		}
	}
	if taken < len(call.Args) {
		return refuse("one argument too many: %s", call.Args[taken])
	}
	return call, nil
}

// Usage gives a command's usage line, from its data.
func Usage(c Command) string {
	parts := []string{"usage: kit " + strings.Join(c.Words, " ")}
	for _, o := range c.Options {
		part := "[--" + o.Name
		if o.Value != "" {
			part += "=<" + o.Value + ">"
		}
		part += "]"
		if o.Many {
			part += "..."
		}
		parts = append(parts, part)
	}
	for _, a := range c.Args {
		part := "<" + a.Name + ">"
		if a.Optional {
			part = "[" + part + "]"
		}
		if a.Many {
			part += "..."
		}
		parts = append(parts, part)
	}
	return strings.Join(append(parts, "[--json]", "[--help]"), " ")
}

// Help gives kit's help, with words nil, or one command's: its usage, what it does, and its options.
func Help(commands []Command, words []string) string {
	var b strings.Builder
	if words == nil {
		b.WriteString("usage: kit <command> [--json] [--help]\n\ncommands:\n")
		width := 0
		for _, c := range commands {
			width = max(width, len(strings.Join(c.Words, " ")))
		}
		for _, c := range commands {
			fmt.Fprintf(&b, "  %-*s  %s\n", width, strings.Join(c.Words, " "), c.Summary)
		}
		return b.String()
	}
	at := slices.IndexFunc(commands, func(c Command) bool { return slices.Equal(c.Words, words) })
	c := commands[at]
	fmt.Fprintf(&b, "%s\n\n%s", Usage(c), c.Help)
	if len(c.Options) > 0 {
		b.WriteString("\noptions:\n")
		var names []string
		width := 0
		for _, o := range c.Options {
			name := "--" + o.Name
			if o.Value != "" {
				name += "=<" + o.Value + ">"
			}
			names = append(names, name)
			width = max(width, len(name))
		}
		for i, o := range c.Options {
			fmt.Fprintf(&b, "  %-*s  %s\n", width, names[i], o.Help)
		}
	}
	return b.String()
}

// Run parses args, and has the handler of the command they name do it, keyed by the command's words joined by a
// space. It gives the exit status: the handler's, 0 for help, or 2 for words it cannot take.
func Run(ctx context.Context, commands []Command, handlers map[string]Handler, args []string,
	stdout, stderr io.Writer) int {
	if slices.Equal(args, []string{"--help"}) {
		fmt.Fprint(stdout, Help(commands, nil))
		return 0
	}
	call, err := Parse(commands, args)
	if errors.Is(err, ErrNoCommand) {
		fmt.Fprint(stderr, Help(commands, nil))
		return 2
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if call.Help {
		fmt.Fprint(stdout, Help(commands, call.Command.Words))
		return 0
	}
	words := strings.Join(call.Command.Words, " ")
	handler, ok := handlers[words]
	if !ok {
		fmt.Fprintf(stderr, "kit: %s has no handler\n", words)
		return 2
	}
	return handler(ctx, call, Out{stdout, stderr, call.JSON})
}

// Print says value as JSON on out's stdout, with the schema kit/<schema> and its version, 1, beside its own fields.
// value must be an object. It gives the exit status: 0, or 1 where value cannot be said so.
func Print(out Out, schema string, value any) int {
	fields := map[string]json.RawMessage{}
	encoded, err := json.Marshal(value)
	if err == nil {
		err = json.Unmarshal(encoded, &fields)
	}
	if err != nil {
		fmt.Fprintf(out.Stderr, "kit: kit/%s cannot be said as JSON: %v\n", schema, err)
		return 1
	}
	fields["schema"], _ = json.Marshal("kit/" + schema)
	fields["version"] = json.RawMessage("1")
	said, _ := json.MarshalIndent(fields, "", "  ")
	fmt.Fprintf(out.Stdout, "%s\n", said)
	return 0
}
