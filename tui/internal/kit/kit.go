// Package kit runs the kit CLI for the binary: the grammar and the help it reads, the choices and the extension it
// asks for as a form fills, and the exec that finishes a verb. The binary knows no verb by name; every word it shows
// comes through here.
package kit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/empowerite/kit/tui/internal/syntax"
)

// Runner names the kit to run and the directory the command was typed in, which every call runs from.
type Runner struct {
	Kit string // the kit executable: $KIT, else kit on PATH
	Dir string // where the command was typed: $kit_invoked_in, which kit's verbs export, else the current directory
}

// New reads the runner from the environment.
func New() Runner {
	r := Runner{Kit: os.Getenv("KIT"), Dir: os.Getenv("kit_invoked_in")}
	if r.Kit == "" {
		r.Kit = "kit"
	}
	if r.Dir == "" {
		r.Dir, _ = os.Getwd()
	}
	return r
}

// ExitError is kit's refusal: its exit code, and what it printed on stderr, the one line kit prints.
type ExitError struct {
	Code   int
	Stderr string
}

func (e *ExitError) Error() string { return strings.TrimSpace(e.Stderr) }

// Choice is one row of kit syntax --choices: the value, its description, its state (active, excluded, or empty), and
// the line the verb would print about it given the answers so far, empty when there is none.
type Choice struct {
	Value, Description, State, Refusal string
}

// Verbs gives every verb's spec, as kit syntax prints them.
func (r Runner) Verbs() ([]syntax.Spec, error) {
	out, err := r.run("syntax")
	if err != nil {
		return nil, err
	}
	return syntax.DecodeAll(bytes.NewReader(out))
}

// Spec gives one verb's spec; a name that is no verb is kit's refusal, an ExitError with code 2.
func (r Runner) Spec(verb string) (syntax.Spec, error) {
	out, err := r.run("syntax", verb)
	if err != nil {
		return syntax.Spec{}, err
	}
	return syntax.Decode(bytes.NewReader(out))
}

// Help gives the verb's help page, as kit help <verb> prints it with color off.
func (r Runner) Help(verb string) (string, error) {
	out, err := r.run("help", verb)
	return string(out), err
}

// Choices gives the rows of kit syntax <verb> --choices <key>, given the answers so far, each <key>=<value>; they
// follow a --, so an answer that starts with - is never read as kit's own.
func (r Runner) Choices(verb, key string, answers []string) ([]Choice, error) {
	out, err := r.run(withAnswers([]string{"syntax", verb, "--choices", key}, answers)...)
	if err != nil {
		return nil, err
	}
	var rows []Choice
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		columns := append(strings.SplitN(line, "\t", 4), "", "", "", "")
		rows = append(rows, Choice{Value: columns[0], Description: columns[1], State: columns[2], Refusal: columns[3]})
	}
	return rows, nil
}

// Extend gives the arguments the verb learns from the answers so far, as kit syntax <verb> --extend prints them: none
// before the verb knows.
func (r Runner) Extend(verb string, answers []string) (map[string]syntax.Argument, error) {
	out, err := r.run(withAnswers([]string{"syntax", verb, "--extend"}, answers)...)
	if err != nil {
		return nil, err
	}
	arguments := map[string]syntax.Argument{}
	err = json.Unmarshal(out, &arguments)
	return arguments, err
}

// Match judges a command line for the verb as the guard would, through kit syntax <verb> --match; the words follow a --.
func (r Runner) Match(verb string, words []string) (syntax.Match, error) {
	out, err := r.run(withAnswers([]string{"syntax", verb, "--match"}, words)...)
	if err != nil {
		return syntax.Match{}, err
	}
	return syntax.DecodeMatch(bytes.NewReader(out))
}

// Exec replaces this process with kit <verb> <args>…, run where the command was typed. It returns only on failure.
func (r Runner) Exec(verb string, args []string) error {
	path, err := exec.LookPath(r.Kit)
	if err != nil {
		return err
	}
	if err := os.Chdir(r.Dir); err != nil {
		return err
	}
	return syscall.Exec(path, append([]string{path, verb}, args...), os.Environ())
}

// withAnswers appends the answers after a -- when there are any.
func withAnswers(args, answers []string) []string {
	if len(answers) == 0 {
		return args
	}
	return append(append(args, "--"), answers...)
}

// run runs kit with the arguments, in Dir and with color off, and gives its stdout. A non-zero exit is an ExitError
// carrying kit's stderr; a kit that cannot run at all is a plain error.
func (r Runner) run(args ...string) ([]byte, error) {
	cmd := exec.Command(r.Kit, args...)
	cmd.Dir = r.Dir
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return nil, &ExitError{Code: exit.ExitCode(), Stderr: stderr.String()}
		}
		return nil, fmt.Errorf("%s: %w", r.Kit, err)
	}
	return stdout.Bytes(), nil
}
