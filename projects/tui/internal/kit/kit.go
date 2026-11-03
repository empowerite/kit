// Package kit runs the kit CLI for the binary: the grammar and the help it reads, the choices and the extension it
// asks for as a form fills, the exec that finishes a verb, and kit welcome's checks and fixes. The binary knows no
// verb by name; every word it shows comes through here.
package kit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/empowerite/kit/projects/tui/internal/syntax"
)

// Runner names the kit to run and the directory the command was typed in, which every call runs from.
type Runner struct {
	Kit string // the kit executable: $KIT, which kit tui names as the kit that ran it, else kit on PATH
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

// Error gives kit's line without the ❌ kit prints before it, so whatever shows the line adds the glyph once.
func (e *ExitError) Error() string { return strings.TrimPrefix(strings.TrimSpace(e.Stderr), "❌ ") }

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

// Verb is a verb's name, purpose, and group, as kit syntax --list prints them; the group is one of kit help's, and
// empty when the kit printed none.
type Verb struct {
	Name, Purpose, Group string
}

// List gives every verb's name, purpose, and group, from kit syntax --list, which answers at once where Verbs reads
// every verb's grammar.
func (r Runner) List() ([]Verb, error) {
	out, err := r.run("syntax", "--list")
	if err != nil {
		return nil, err
	}
	return parseList(string(out)), nil
}

// parseList reads the lines kit syntax --list prints, each <verb>, <purpose>, and <group> separated by tabs. A line
// of two columns, from a kit that prints no group, leaves the group empty; a line with no tab is no verb.
func parseList(out string) []Verb {
	var verbs []Verb
	for _, line := range strings.Split(out, "\n") {
		columns := strings.SplitN(line, "\t", 3)
		if len(columns) < 2 {
			continue
		}
		v := Verb{Name: columns[0], Purpose: columns[1]}
		if len(columns) == 3 {
			v.Group = columns[2]
		}
		verbs = append(verbs, v)
	}
	return verbs
}

// Spec gives one verb's spec; a name that is no verb is kit's refusal, an ExitError with code 2.
func (r Runner) Spec(verb string) (syntax.Spec, error) {
	out, err := r.run("syntax", verb)
	if err != nil {
		return syntax.Spec{}, err
	}
	return syntax.Decode(bytes.NewReader(out))
}

// Help gives the verb's help page as kit help <verb> draws it at a terminal, its headings bold, or plain when the
// user sets NO_COLOR. kit help reads CLICOLOR_FORCE as a terminal, since the page reaches it through a pipe.
func (r Runner) Help(verb string) (string, error) {
	env := withoutColor(os.Environ())
	if os.Getenv("NO_COLOR") == "" {
		env = append(env, "CLICOLOR_FORCE=1")
	} else {
		env = append(env, "NO_COLOR=1")
	}
	out, err := r.runIn(env, "help", verb)
	return string(out), err
}

// withoutColor gives the environment without NO_COLOR and CLICOLOR_FORCE, so one choice of color can be added.
func withoutColor(env []string) []string {
	var kept []string
	for _, kv := range env {
		if !strings.HasPrefix(kv, "NO_COLOR=") && !strings.HasPrefix(kv, "CLICOLOR_FORCE=") {
			kept = append(kept, kv)
		}
	}
	return kept
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

// Kind asks whether a key's choices, given the answers so far, are closed, open, or a set: closed choices that name the
// whole of something, as kit packages select does, so the form starts from the active rows and marks each by plan.
func (r Runner) Kind(verb, key string, answers []string) (string, error) {
	out, err := r.run(append([]string{"syntax", verb, "--choices", key, "--", "--kind"}, answers...)...)
	return strings.TrimSpace(string(out)), err
}

// Extend gives the arguments the verb learns from the answers so far, as kit syntax <verb> --extend prints them, with
// their keys in the order kit printed them (a template's holes in the template's order): none before the verb knows.
func (r Runner) Extend(verb string, answers []string) ([]string, map[string]syntax.Argument, error) {
	out, err := r.run(withAnswers([]string{"syntax", verb, "--extend"}, answers)...)
	if err != nil {
		return nil, nil, err
	}
	arguments := map[string]syntax.Argument{}
	if err := json.Unmarshal(out, &arguments); err != nil {
		return nil, nil, err
	}
	var keys []string
	dec := json.NewDecoder(bytes.NewReader(out))
	depth := 0
	for {
		token, err := dec.Token()
		if err != nil {
			break
		}
		switch v := token.(type) {
		case json.Delim:
			if v == '{' {
				depth++
			} else if v == '}' {
				depth--
			}
		case string:
			if _, ok := arguments[v]; ok && depth == 1 {
				keys = append(keys, v)
				var skip json.RawMessage
				if err := dec.Decode(&skip); err != nil {
					return nil, nil, err
				}
			}
		}
	}
	return keys, arguments, nil
}

// Match judges a command line for the verb as the guard would, through kit syntax <verb> --match; the words follow
// a --.
func (r Runner) Match(verb string, words []string) (syntax.Match, error) {
	out, err := r.run(withAnswers([]string{"syntax", verb, "--match"}, words)...)
	if err != nil {
		return syntax.Match{}, err
	}
	return syntax.DecodeMatch(bytes.NewReader(out))
}

// Stages gives the stages kit flow draws, in the picture's order, as kit flow --stages prints them, one per line.
func (r Runner) Stages() ([]string, error) {
	out, err := r.run("flow", "--stages")
	if err != nil {
		return nil, err
	}
	var stages []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			stages = append(stages, line)
		}
	}
	return stages, nil
}

// waitDelay is how long Status waits, once ctx has ended, for kit to die and its pipes to close before the pipes
// are closed for it.
const waitDelay = 2 * time.Second

// Status gives one stage's JSON, as kit flow <stage> --json <args>… prints it; the args are the stage's own flags.
// A stage kit cannot draw is kit's refusal, an ExitError. Kit runs in a process group of its own until ctx ends: then
// the group is sent SIGTERM, so a gh or an ssh kit started dies with it, and the error is ctx's.
func (r Runner) Status(ctx context.Context, stage string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, r.Kit, append([]string{"flow", stage, "--json"}, args...)...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) }
	cmd.WaitDelay = waitDelay
	out, err := r.output(cmd)
	if err != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return out, err
}

// Check is one of kit welcome's checks, as kit welcome --checks --json prints it. Fix is the command that fixes it,
// empty when kit knows none. Interactive says the fix needs the person, as gh auth login does. More says where to read
// more, empty when there is nothing to read. Runs says kit runs the fix; it is false for a fix kit only shows, as one
// that must run in the person's own shell. A check that does not say counts as runnable, so an older kit still works.
type Check struct {
	ID          string `json:"id"`
	Step        string `json:"step"`
	OK          bool   `json:"ok"`
	Line        string `json:"line"`
	Fix         string `json:"fix"`
	Interactive bool   `json:"interactive"`
	More        string `json:"more"`
	Runs        bool   `json:"runs"`
}

// UnmarshalJSON reads a check, runnable unless it says runs is false.
func (c *Check) UnmarshalJSON(b []byte) error {
	type plain Check
	p := plain{Runs: true}
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*c = Check(p)
	return nil
}

// WelcomeChecks gives kit welcome's checks, in the order kit ran them.
func (r Runner) WelcomeChecks() ([]Check, error) {
	out, err := r.run("welcome", "--checks", "--json")
	if err != nil {
		return nil, err
	}
	var checks []Check
	if err := json.Unmarshal(out, &checks); err != nil {
		return nil, fmt.Errorf("kit welcome --checks --json: %v", err)
	}
	return checks, nil
}

// WelcomeFix gives the command that runs a check's fix, kit welcome --fix <id>, where the command was typed. It is not
// started: the caller hands it the terminal, since a fix may print and ask. It exits 0 when the check now holds, 1
// when it still fails, and 2 when kit refuses.
func (r Runner) WelcomeFix(id string) *exec.Cmd {
	cmd := exec.Command(r.Kit, "welcome", "--fix", id)
	cmd.Dir = r.Dir
	return cmd
}

// Root gives the directory of the release that runs, as kit root prints it.
func (r Runner) Root() (string, error) {
	out, err := r.run("root")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
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
	return r.runIn(append(os.Environ(), "NO_COLOR=1"), args...)
}

// runIn runs kit as run does, in the environment env.
func (r Runner) runIn(env []string, args ...string) ([]byte, error) {
	cmd := exec.Command(r.Kit, args...)
	cmd.Env = env
	return r.output(cmd)
}

// output runs the command in Dir and gives its stdout, as run says.
func (r Runner) output(cmd *exec.Cmd) ([]byte, error) {
	cmd.Dir = r.Dir
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
