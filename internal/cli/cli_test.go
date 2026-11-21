package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// commands are the tests' own: a command of one word, two that share a first word, and one that takes the rest.
var commands = []Command{
	{Words: []string{"version"}, Summary: "say which build this is", Help: "Says which build this is.\n"},
	{Words: []string{"tools"}, Summary: "list the tools", Help: "Lists the tools.\n"},
	{Words: []string{"tools", "add"}, Summary: "pin a tool", Help: "Pins a tool.\n",
		Options: []Option{{Name: "unconfirmed", Help: "pin a file nothing confirms"}},
		Args:    []Arg{{Name: "tool", Many: true}}},
	{Words: []string{"env"}, Summary: "print the environment", Help: "Prints the environment.\n",
		Options: []Option{
			{Name: "shell", Value: "name", Help: "the shell to print for"},
			{Name: "watch", Value: "file", Help: "a file to watch", Many: true},
		},
		Args: []Arg{{Name: "dir", Optional: true}}},
	{Words: []string{"exec"}, Summary: "run a command", Help: "Runs a command.\n", Args: []Arg{{Name: "command", Many: true}}},
}

func parse(t *testing.T, args ...string) Call {
	t.Helper()
	call, err := Parse(commands, args)
	if err != nil {
		t.Fatalf("Parse(%q) gave the error %v", args, err)
	}
	return call
}

func TestParseTakesTheLongestRunOfWords(t *testing.T) {
	call := parse(t, "tools", "add", "shellcheck")
	if got := strings.Join(call.Command.Words, " "); got != "tools add" || !reflect.DeepEqual(call.Args, []string{"shellcheck"}) {
		t.Errorf("Parse gave %q with %q; want tools add with [shellcheck]", got, call.Args)
	}
	if call = parse(t, "tools"); strings.Join(call.Command.Words, " ") != "tools" {
		t.Errorf("Parse(tools) gave %q; want tools", call.Command.Words)
	}
}

func TestParseTakesOnlyOptionEqualsValue(t *testing.T) {
	call := parse(t, "env", "--shell=bash")
	if !reflect.DeepEqual(call.Options["shell"], []string{"bash"}) {
		t.Errorf("Parse gave the options %v; want shell=bash", call.Options)
	}
	for _, args := range [][]string{{"env", "--shell", "bash"}, {"env", "--shell="}} {
		_, err := Parse(commands, args)
		if err == nil || !strings.Contains(err.Error(), "--shell=<name>") ||
			!strings.Contains(err.Error(), "usage: kit env") {
			t.Errorf("Parse(%q) gave the error %v; want one that says --shell=<name> and the usage", args, err)
		}
	}
}

func TestParseRefusesAValueForASwitch(t *testing.T) {
	_, err := Parse(commands, []string{"tools", "add", "--unconfirmed=yes", "x"})
	if err == nil || !strings.Contains(err.Error(), "--unconfirmed takes no value") {
		t.Errorf("Parse gave the error %v; want one that says --unconfirmed takes no value", err)
	}
}

func TestParseRepeatsAManyOption(t *testing.T) {
	call := parse(t, "env", "--watch=a", "--watch=b")
	if !reflect.DeepEqual(call.Options["watch"], []string{"a", "b"}) {
		t.Errorf("Parse gave %v; want watch=a and watch=b", call.Options)
	}
	if _, err := Parse(commands, []string{"env", "--shell=a", "--shell=b"}); err == nil ||
		!strings.Contains(err.Error(), "--shell is given twice") {
		t.Errorf("Parse gave the error %v; want one that says --shell is given twice", err)
	}
}

func TestParseRefusesAWordItDoesNotKnow(t *testing.T) {
	_, err := Parse(commands, []string{"flow"})
	if err == nil || err.Error() != "kit: no such command: flow" {
		t.Errorf("Parse gave the error %v; want kit: no such command: flow", err)
	}
	_, err = Parse(commands, nil)
	if !errors.Is(err, ErrNoCommand) {
		t.Errorf("Parse(nothing) gave the error %v; want ErrNoCommand", err)
	}
}

func TestParseRefusesAnOptionItDoesNotKnow(t *testing.T) {
	_, err := Parse(commands, []string{"version", "--loud"})
	if err == nil || !strings.HasPrefix(err.Error(), "kit version: no such option: --loud\nusage: kit version") {
		t.Errorf("Parse gave the error %v; want one that names --loud, then the usage", err)
	}
}

func TestParseCountsTheArguments(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"version", "extra"}, "kit version: one argument too many: extra"},
		{[]string{"tools", "add"}, "kit tools add: <tool> is missing"},
		{[]string{"env", "a", "b"}, "kit env: one argument too many: b"},
	} {
		_, err := Parse(commands, c.args)
		if err == nil || !strings.HasPrefix(err.Error(), c.want+"\nusage: ") {
			t.Errorf("Parse(%q) gave the error %v; want %q and the usage", c.args, err, c.want)
		}
	}
	if call := parse(t, "env"); len(call.Args) != 0 {
		t.Errorf("Parse(env) gave the arguments %q; want none", call.Args)
	}
}

func TestParseTakesWhatFollowsTwoDashesAsArguments(t *testing.T) {
	call := parse(t, "exec", "--", "shellcheck", "--version")
	if !reflect.DeepEqual(call.Args, []string{"shellcheck", "--version"}) {
		t.Errorf("Parse gave the arguments %q; want shellcheck --version", call.Args)
	}
}

func TestParseTakesJSONAndHelpForEveryCommand(t *testing.T) {
	call := parse(t, "version", "--json")
	if !call.JSON {
		t.Error("Parse(version --json) did not ask for JSON")
	}
	if call = parse(t, "tools", "add", "--help"); !call.Help {
		t.Error("Parse(tools add --help) did not ask for help")
	}
}

func TestUsageNamesTheOptionsAndArguments(t *testing.T) {
	want := "usage: kit env [--shell=<name>] [--watch=<file>]... [<dir>] [--json] [--help]"
	if got := Usage(commands[3]); got != want {
		t.Errorf("Usage(env) = %q; want %q", got, want)
	}
	want = "usage: kit tools add [--unconfirmed] <tool>... [--json] [--help]"
	if got := Usage(commands[2]); got != want {
		t.Errorf("Usage(tools add) = %q; want %q", got, want)
	}
}

func TestHelpComesFromTheData(t *testing.T) {
	got := Help(commands, nil)
	for _, want := range []string{"usage: kit <command>", "  tools add  pin a tool\n", "  version    say which build this is\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("Help() = %q; want it to hold %q", got, want)
		}
	}
	got = Help(commands, []string{"env"})
	for _, want := range []string{"usage: kit env", "Prints the environment.", "  --shell=<name>  the shell to print for\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("Help(env) = %q; want it to hold %q", got, want)
		}
	}
}

func TestCommandsComeFromKitsData(t *testing.T) {
	got, err := Commands()
	if err != nil || len(got) == 0 || got[0].Words[0] == "" {
		t.Errorf("Commands() = %v, %v; want kit's commands", got, err)
	}
}

// handlers are the tests' own: each prints what it was given.
func handlers() map[string]Handler {
	said := func(ctx context.Context, call Call, out Out) int {
		if out.JSON {
			return Print(out, "said", map[string]any{"args": call.Args})
		}
		out.Stdout.Write([]byte(strings.Join(call.Args, " ") + "\n"))
		return 0
	}
	return map[string]Handler{"version": said, "tools": said, "tools add": said, "env": said, "exec": said}
}

func run(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), commands, handlers(), args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestRunGivesTheHandlerTheCall(t *testing.T) {
	if code, out, err := run("tools", "add", "shellcheck", "lychee"); code != 0 || out != "shellcheck lychee\n" || err != "" {
		t.Errorf("Run = %d, %q, %q; want 0 and the arguments", code, out, err)
	}
}

func TestRunGivesJSONWithSchemaAndVersion(t *testing.T) {
	code, out, _ := run("exec", "--json", "--", "a")
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); code != 0 || err != nil {
		t.Fatalf("Run = %d, %q; want 0 and JSON", code, out)
	}
	want := map[string]any{"schema": "kit/said", "version": float64(1), "args": []any{"a"}}
	if !reflect.DeepEqual(got, want) || !strings.HasSuffix(out, "}\n") {
		t.Errorf("Run printed %q; want %v, and a final newline", out, want)
	}
}

func TestRunPrintsHelpAndEndsWell(t *testing.T) {
	if code, out, _ := run("env", "--help"); code != 0 || !strings.Contains(out, "usage: kit env") {
		t.Errorf("Run(env --help) = %d, %q; want 0 and env's help", code, out)
	}
	if code, out, _ := run("--help"); code != 0 || !strings.Contains(out, "usage: kit <command>") {
		t.Errorf("Run(--help) = %d, %q; want 0 and kit's help", code, out)
	}
}

func TestRunRefusesWordsItCannotTake(t *testing.T) {
	if code, out, err := run(); code != 2 || out != "" || !strings.Contains(err, "usage: kit <command>") {
		t.Errorf("Run() = %d, %q, %q; want 2 and kit's help on stderr", code, out, err)
	}
	if code, out, err := run("flow"); code != 2 || out != "" || err != "kit: no such command: flow\n" {
		t.Errorf("Run(flow) = %d, %q, %q; want 2 and the refusal on stderr", code, out, err)
	}
}

func TestRunRefusesACommandWithNoHandler(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), commands, map[string]Handler{}, []string{"version"}, &stdout, &stderr)
	if code != 2 || stderr.String() != "kit: version has no handler\n" {
		t.Errorf("Run = %d, %q; want 2 and that version has no handler", code, stderr.String())
	}
}

func TestPrintRefusesAValueThatIsNotAnObject(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Print(Out{&stdout, &stderr, true}, "x", []int{1}); code != 1 || !strings.Contains(stderr.String(), "kit/x") {
		t.Errorf("Print = %d, %q; want 1 and an error that names kit/x", code, stderr.String())
	}
	stderr.Reset()
	if code := Print(Out{&stdout, &stderr, true}, "x", func() {}); code != 1 || !strings.Contains(stderr.String(), "kit/x") {
		t.Errorf("Print = %d, %q; want 1 and an error that names kit/x", code, stderr.String())
	}
}
