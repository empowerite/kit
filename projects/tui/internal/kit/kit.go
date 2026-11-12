// Package kit runs the kit CLI for the binary: the grammar and the help it reads, the choices and the extension it
// asks for as a form fills, the exec that finishes a verb, kit welcome's checks and fixes, and the tutorials'
// catalogue. The binary knows no verb by name; every word it shows comes through here.
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

// Runner names the kit to run and the directory the command was typed in, which every call runs from, and the group
// kit flow's reads span, when they span one.
type Runner struct {
	Kit     string // the kit executable: $KIT, which kit tui names as the kit that ran it, else kit on PATH
	Dir     string // where the command was typed: $kit_invoked_in, which kit's verbs export, else the current directory
	Group   string // the group kit flow's reads span, as kit --workspace=<Group>; none reads the place Dir is
	NoGroup bool   // the workspace shown is no group, so the reads carry --workspace=none and draw no last group used
}

// over gives kit flow's words as a read takes them: after --workspace=<Group>, the kit option, which comes before the
// verb, while the reads span a group, and after --workspace=none while the workspace shown is no group, so a read
// outside a repository draws the place Dir is, as kittui titles it, and not the last group used. Every read names its
// workspace, as kit sets KIT_WORKSPACE from the option alone, so a switch of workspace is never read stale.
func (r Runner) over(words ...string) []string {
	switch {
	case r.Group != "":
		return append([]string{"--workspace=" + r.Group}, words...)
	case r.NoGroup:
		return append([]string{"--workspace=none"}, words...)
	}
	return words
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

// ExitError is kit's refusal: its exit code, what it printed on stderr, the one line kit prints, and what it printed
// on stdout, which, with exit 1, is the part of a document kit could read.
type ExitError struct {
	Code   int
	Stderr string
	Stdout []byte
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
				// the document decoded whole above, so the value after its key decodes too
				var skip json.RawMessage
				_ = dec.Decode(&skip)
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

// Stages gives the stages kit flow draws, in the picture's order, as kit flow --stages prints them, one per line;
// over a group, the group's.
func (r Runner) Stages() ([]string, error) { return r.StagesOf("flow") }

// StagesOf gives the stages a verb's picture draws, kit flow's or kit workspace's, as <verb> --stages prints them.
func (r Runner) StagesOf(verb string) ([]string, error) {
	out, err := r.run(r.over(verb, "--stages")...)
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

// Me gives the caller's login on the primary's forge, as kit flow --me prints it: the login a row's people are
// matched against. Kit's refusal, as when the forge cannot be read, is the error.
func (r Runner) Me(ctx context.Context) (string, error) {
	cmd := grouped(ctx, r.Kit, "flow", "--me")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := r.output(cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// Quota is one forge's rate limits on the caller, as kit flow --quota gives them: the forge, as kit names it, and each
// limit.
type Quota struct {
	Forge  string  `json:"forge"`
	Limits []Limit `json:"limits"`
}

// Limit is one rate limit: its name, as the forge gives it, how many it allows, how many are left, and when it resets,
// in seconds since the epoch, or none where the forge does not say.
type Limit struct {
	Name      string `json:"name"`
	Limit     int    `json:"limit"`
	Remaining int    `json:"remaining"`
	Reset     int64  `json:"reset"`
}

// Quota asks kit for the caller's rate limits on each forge kit flow reads, as kit flow --quota prints them; reading
// them costs a request of each forge at most. Over a group, the limits of the members kit could read are kept, as
// partial says. ctx ends it with the screen.
func (r Runner) Quota(ctx context.Context) ([]Quota, error) {
	out, err := partial(r.within(ctx, callLimit, append(os.Environ(), "NO_COLOR=1"), r.over("flow", "--quota")...))
	if err != nil {
		return nil, err
	}
	var quotas []Quota
	if err := json.Unmarshal(out, &quotas); err != nil {
		return nil, fmt.Errorf("kit flow --quota: %w", err)
	}
	return quotas, nil
}

// Marks asks kit for a mark per stage, as kit flow --marks prints them, by the stage's name: a text that changes when
// what the stage draws does; a stage kit prints null for, as one kit cannot read a mark of where kit exits 1 as the
// forge cannot answer, or one that has none, has an empty one, and the marks kit read beside it are kept, as partial
// says. ctx ends it with the screen.
func (r Runner) Marks(ctx context.Context) (map[string]string, error) { return r.MarksOf(ctx, "flow") }

// MarksOf asks kit for a mark per stage of a verb's picture, kit flow's or kit workspace's, as <verb> --marks prints
// them, as Marks says. Over a group, errors, beside the marks, names the members kit could not read; it is no stage.
func (r Runner) MarksOf(ctx context.Context, verb string) (map[string]string, error) {
	out, err := partial(r.within(ctx, callLimit, append(os.Environ(), "NO_COLOR=1"), r.over(verb, "--marks")...))
	if err != nil {
		return nil, err
	}
	var marks map[string]json.RawMessage
	if err := json.Unmarshal(out, &marks); err != nil {
		return nil, fmt.Errorf("kit %s --marks: %w", verb, err)
	}
	delete(marks, "errors")
	kept := map[string]string{}
	for stage, raw := range marks {
		var mark string // null leaves it empty
		if err := json.Unmarshal(raw, &mark); err != nil {
			return nil, fmt.Errorf("kit %s --marks: %s: %w", verb, stage, err)
		}
		kept[stage] = mark
	}
	return kept, nil
}

// partial gives what kit printed, and no error, where kit exited 1 after a document that names in itself what it
// could not read: stdout that, trimmed, is JSON and opens an object or an array. Anything else, another exit, nothing
// printed, or what is not such JSON, is the out and the error it was given.
func partial(out []byte, err error) ([]byte, error) {
	var exit *ExitError
	if !errors.As(err, &exit) || exit.Code != 1 {
		return out, err
	}
	doc := bytes.TrimSpace(exit.Stdout)
	if len(doc) == 0 || (doc[0] != '{' && doc[0] != '[') || !json.Valid(doc) {
		return out, err
	}
	return exit.Stdout, nil
}

// waitDelay is how long Status waits, once ctx has ended, for kit to die and its pipes to close before the pipes
// are closed for it.
const waitDelay = 2 * time.Second

// Status gives one stage's JSON, as kit flow <stage> --json <args>… prints it, as Rows reads it, over the group the
// runner spans; the args are the stage's own flags. A stage of another verb's picture is named after the verb and a
// slash, as workspace/branches, which kit workspace branches --json prints.
func (r Runner) Status(ctx context.Context, stage string, args []string) ([]byte, error) {
	verb, name, ok := strings.Cut(stage, "/")
	if !ok {
		verb, name = "flow", stage
	}
	return r.Rows(ctx, append([]string{verb, name, "--json"}, args...), true)
}

// Rows gives what a kit command prints, kit's words after kit, as a stage's catalog entry names them: with over, the
// workspace the runner reads, before the verb, as over says, so a command that reads a group's members reads every
// one. A command kit refuses is kit's refusal, an ExitError; one kit read in part, its document with exit 1, is that
// document, as partial says. Kit runs in a process group of its own until ctx ends: then the group is sent SIGTERM, so
// a gh or an ssh kit started dies with it, and the error is ctx's.
func (r Runner) Rows(ctx context.Context, words []string, over bool) ([]byte, error) {
	if over {
		words = r.over(words...)
	}
	cmd := grouped(ctx, r.Kit, words...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := r.output(cmd)
	if err != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return partial(out, err)
}

// Tracker is a tracker a new issue can be filed in, as kit new issue --list --json gives it: the prefix its references
// start with, its provider, a label that says where, and the page kit opens to file one, or its refusal where it gives
// none.
type Tracker struct {
	Prefix   string `json:"prefix"`
	Provider string `json:"provider"`
	Label    string `json:"label"`
	URL      string `json:"url"`
	Error    string `json:"error"`
}

// Trackers asks kit for the trackers a new issue can be filed in, the forge's own first, which the forge's tool may
// take a moment to say; ctx ends it with the screen.
func (r Runner) Trackers(ctx context.Context) ([]Tracker, error) {
	out, err := r.within(ctx, callLimit, append(os.Environ(), "NO_COLOR=1"), "new", "issue", "--list", "--json")
	if err != nil {
		return nil, err
	}
	var trackers []Tracker
	if err := json.Unmarshal(out, &trackers); err != nil {
		return nil, fmt.Errorf("kit new issue --list --json: %w", err)
	}
	return trackers, nil
}

// Act runs a verb that changes something, kit <words>…, as the tab's menu asks, and gives what it printed, stdout and
// stderr as they came, where kit says what it did. Kit's refusal is an ExitError whose line is the last kit marked
// as failed, or its last line.
func (r Runner) Act(ctx context.Context, words ...string) (string, error) {
	cmd := grouped(ctx, r.Kit, words...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	cmd.Dir = r.Dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return out.String(), nil
	case errors.As(err, &exit):
		return out.String(), &ExitError{Code: exit.ExitCode(), Stderr: refusal(out.String())}
	default:
		return out.String(), fmt.Errorf("%s: %w", r.Kit, err)
	}
}

// refusal gives the line of kit's output that says why it failed: the last marked ❌, or else the last.
func refusal(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for k := len(lines) - 1; k >= 0; k-- {
		if strings.HasPrefix(lines[k], "❌") {
			return lines[k]
		}
	}
	return lines[len(lines)-1]
}

// grouped gives the command kit runs as: in a process group of its own, sent SIGTERM when ctx ends, so what kit
// started dies with it, and waited for waitDelay at most after that.
func grouped(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) }
	cmd.WaitDelay = waitDelay
	return cmd
}

// Check is one of kit welcome's checks, as kit welcome --checks --json prints it. Fix is the command that fixes it,
// empty when kit knows none. Interactive says the fix needs the person, as gh auth login does. More says where to read
// more, empty when there is nothing to read. Runs says kit runs the fix; it is false for a fix kit only shows, as one
// that must run in the person's own shell. A check that does not say counts as runnable, so an older kit still works.
// A folder's check names the folder it holds in, the absolute path its fix runs in; its repository, host/path, or the
// desk's folder name with no remote; whether the folder is the desk, else a worktree; the branch checked out there,
// empty with none; and the kit version its kit.yml declares, empty with none. A machine check names no folder, nor
// does any check of an older kit.
type Check struct {
	ID          string `json:"id"`
	Step        string `json:"step"`
	OK          bool   `json:"ok"`
	Line        string `json:"line"`
	Fix         string `json:"fix"`
	Interactive bool   `json:"interactive"`
	More        string `json:"more"`
	Runs        bool   `json:"runs"`
	Folder      string `json:"folder"`
	Repository  string `json:"repository"`
	Desk        bool   `json:"desk"`
	Branch      string `json:"branch"`
	Kit         string `json:"kit"`
}

// Key gives what tells a check from every other on a screen: its id, after its folder when it has one, as a folder's
// check has the same id in every folder.
func (c Check) Key() string {
	if c.Folder == "" {
		return c.ID
	}
	return c.Folder + " " + c.ID
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

// WelcomeChecks gives kit welcome's checks, in the order kit ran them: the machine's, then each folder's of the
// workspace, as over names it, every member's desk and worktrees over a group. They have checksLimit at most, and end
// when ctx does, as the screen that asked for them ends. told, unless nil, hears what kit says as they run, each check
// that starts and each folder whose checks end, from kit welcome --checks --json --progress.
func (r Runner) WelcomeChecks(ctx context.Context, told func(Progress)) ([]Check, error) {
	args := r.over("welcome", "--checks", "--json")
	if told != nil {
		args = append(args, "--progress")
	}
	out, err := r.telling(ctx, checksLimit, append(os.Environ(), "NO_COLOR=1"), told, args...)
	if err != nil {
		return nil, err
	}
	var checks []Check
	if err := json.Unmarshal(out, &checks); err != nil {
		return nil, fmt.Errorf("kit welcome --checks --json: %v", err)
	}
	return checks, nil
}

// Progress is what kit welcome --checks --progress says as it goes: the check that starts, in Folder, or on the machine
// where Folder is empty; or, with Check empty, that the checks of Folder, or the machine's, end.
type Progress struct {
	Check  string
	Folder string
}

// progressOf reads one line kit welcome --checks --progress says on stderr: ⏳ checking <id>, with " in <folder>" for
// a folder's, or ⏳ checked, with " <folder>" for a folder. Any other line is not progress.
func progressOf(line string) (Progress, bool) {
	if rest, ok := strings.CutPrefix(line, "⏳ checking "); ok {
		id, folder, _ := strings.Cut(rest, " in ")
		return Progress{Check: id, Folder: folder}, id != ""
	}
	if rest, ok := strings.CutPrefix(line, "⏳ checked"); ok && (rest == "" || strings.HasPrefix(rest, " ")) {
		return Progress{Folder: strings.TrimPrefix(rest, " ")}, true
	}
	return Progress{}, false
}

// toldWriter takes a command's stderr: each line that says progress goes to told as it comes, and every other line to
// rest, as kit's refusal; with no told, every line goes to rest.
type toldWriter struct {
	told    func(Progress)
	rest    bytes.Buffer
	partial []byte
}

// Write takes what the command wrote, a line at a time; a line not yet ended waits for the rest.
func (w *toldWriter) Write(b []byte) (int, error) {
	w.partial = append(w.partial, b...)
	for {
		i := bytes.IndexByte(w.partial, '\n')
		if i < 0 {
			return len(b), nil
		}
		w.line(w.partial[:i+1])
		w.partial = w.partial[i+1:]
	}
}

// line sends one line to told when it says progress, and keeps it otherwise.
func (w *toldWriter) line(l []byte) {
	if p, ok := progressOf(strings.TrimRight(string(l), "\n")); ok && w.told != nil {
		w.told(p)
		return
	}
	w.rest.Write(l)
}

// WelcomeFix gives the command that runs a check's fix, kit welcome --fix <id>, where the command was typed, over the
// workspace, as over names it, so a forge's login finds its host among the folders; a folder's check, with
// --folder=<folder>, runs there. It is not started: the caller hands it the terminal, since a fix may print and ask.
// It exits 0 when the check now holds, 1 when it still fails, and 2 when kit refuses.
func (r Runner) WelcomeFix(c Check) *exec.Cmd {
	args := r.over("welcome", "--fix", c.ID)
	if c.Folder != "" {
		args = append(args, "--folder="+c.Folder)
	}
	cmd := exec.Command(r.Kit, args...)
	cmd.Dir = r.Dir
	return cmd
}

// Catalogue is the tutorials kit knows, as kit tutorials --json prints them: the kit's version, the base every link
// starts with, the audiences in the order they are shown, and the tutorials in theirs.
type Catalogue struct {
	Version   string     `json:"version"`
	Base      string     `json:"base"`
	Audiences []Audience `json:"audiences"`
	Tutorials []Tutorial `json:"tutorials"`
}

// Audience is one of the audiences a tutorial is for: the id a tutorial and --tag name it by, its label, and the color
// its pill is drawn on, as #rrggbb, or none from a kit whose catalogue gives none.
type Audience struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Color string `json:"color"`
}

// Tutorial is one tutorial: its id, title, summary, and glyph; the ids of the audiences it is for; where it is, kit
// for one that ships with the kit or online; and, for one online, its link, which carries the kit's version as ?kit=,
// so the page knows the reader's kit, or, for one in the kit, the path of its page, markdown read on this machine.
type Tutorial struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Summary  string   `json:"summary"`
	Glyph    string   `json:"glyph"`
	Audience []string `json:"audience"`
	Where    string   `json:"where"`
	URL      string   `json:"url"`
	File     string   `json:"file"`
}

// InTheKit is where a tutorial that ships with the kit is, as kit tutorials --json names it.
const InTheKit = "kit"

// Shipped says whether a tutorial ships with the kit, its page read on this machine; one from a kit that names no
// place is online.
func (t Tutorial) Shipped() bool { return t.Where == InTheKit }

// Tutorials gives the catalogue, as kit tutorials --json prints it with a --tag for each tag, which keeps the
// tutorials for any of those audiences; no tag keeps them all. It has callLimit at most, and ends when ctx does.
func (r Runner) Tutorials(ctx context.Context, tags []string) (Catalogue, error) {
	args := []string{"tutorials", "--json"}
	for _, tag := range tags {
		args = append(args, "--tag", tag)
	}
	var c Catalogue
	out, err := r.within(ctx, callLimit, append(os.Environ(), "NO_COLOR=1"), args...)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(out, &c); err != nil {
		return c, fmt.Errorf("kit tutorials --json: %w", err)
	}
	return c, nil
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

// runIn runs kit as run does, in the environment env, for callLimit at most.
func (r Runner) runIn(env []string, args ...string) ([]byte, error) {
	return r.within(context.Background(), callLimit, env, args...)
}

// The limits of kit's calls: a call that reads the grammar, a page, or choices has callLimit, and kit welcome's checks
// have checksLimit, as long as a stage of kit flow. The tests shorten them.
var (
	callLimit   = 30 * time.Second
	checksLimit = 2 * time.Minute
)

// within runs kit in the environment env until ctx ends or the limit passes, whichever is first; past the limit, the
// error says how long the call had, and when ctx ends first, the error is ctx's.
func (r Runner) within(ctx context.Context, limit time.Duration, env []string, args ...string) ([]byte, error) {
	return r.telling(ctx, limit, env, nil, args...)
}

// telling runs kit as within does, and tells told, unless nil, each line of progress kit says on stderr as it says it.
func (r Runner) telling(ctx context.Context, limit time.Duration, env []string, told func(Progress),
	args ...string) ([]byte, error) {
	limited, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	cmd := grouped(limited, r.Kit, args...)
	cmd.Env = env
	out, err := r.outputTold(cmd, told)
	switch {
	case err == nil:
		return out, nil
	case ctx.Err() != nil:
		return nil, ctx.Err()
	case limited.Err() != nil:
		return nil, fmt.Errorf("kit %s took longer than %s; it was stopped", args[0], limit)
	}
	return nil, err
}

// output runs the command in Dir and gives its stdout, as run says.
func (r Runner) output(cmd *exec.Cmd) ([]byte, error) { return r.outputTold(cmd, nil) }

// outputTold runs the command as output does, and tells told, unless nil, each line of progress it says on stderr as it
// says it; the rest of stderr is its refusal.
func (r Runner) outputTold(cmd *exec.Cmd, told func(Progress)) ([]byte, error) {
	cmd.Dir = r.Dir
	var stdout bytes.Buffer
	stderr := &toldWriter{told: told}
	cmd.Stdout, cmd.Stderr = &stdout, stderr
	err := cmd.Run()
	stderr.rest.Write(stderr.partial)
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return nil, &ExitError{Code: exit.ExitCode(), Stderr: stderr.rest.String(), Stdout: stdout.Bytes()}
		}
		return nil, fmt.Errorf("%s: %w", r.Kit, err)
	}
	return stdout.Bytes(), nil
}
