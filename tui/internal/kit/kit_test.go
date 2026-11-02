// The runner asks kit for the grammar, the help, the choices, and the extension, and execs the verb where the command
// was typed; every test runs it against testdata/fake-kit.
package kit

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// fake gives a Runner over testdata/fake-kit in the directory dir.
func fake(t *testing.T, dir string) Runner {
	t.Helper()
	path, err := filepath.Abs("testdata/fake-kit")
	if err != nil {
		t.Fatal(err)
	}
	return Runner{Kit: path, Dir: dir}
}

func TestVerbsAndSpec(t *testing.T) {
	r := fake(t, "")
	verbs, err := r.Verbs()
	if err != nil || len(verbs) != 2 || verbs[0].Verb != "commit" || verbs[1].Verb != "version" {
		t.Fatalf("Verbs: %v, %+v", err, verbs)
	}
	spec, err := r.Spec("commit")
	if err != nil || spec.Verb != "commit" || len(spec.Syntax[0]) != 3 {
		t.Fatalf("Spec: %v, %+v", err, spec)
	}
	_, err = r.Spec("nonesuch")
	var refusal *ExitError
	if !errors.As(err, &refusal) || refusal.Code != 2 || !strings.Contains(refusal.Stderr, "no kit verb 'nonesuch'") {
		t.Fatalf("Spec(nonesuch): %v", err)
	}
	// the line is kit's without its glyph, which what shows the line adds; stderr stays as kit printed it
	if refusal.Error() != "no kit verb 'nonesuch'; run: kit help" || !strings.HasPrefix(refusal.Stderr, "❌ ") {
		t.Errorf("Error() = %q, Stderr = %q", refusal.Error(), refusal.Stderr)
	}
}

func TestHelpIsTheVerbsPageWithItsHeadingsBold(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	page, err := fake(t, "").Help("commit")
	if err != nil || !strings.HasPrefix(page, "kit commit - a purpose for the test\n") ||
		!strings.Contains(page, "\x1b[1mUsage\x1b[0m\n") {
		t.Fatalf("Help: %v, %q", err, page)
	}
}

func TestHelpIsPlainWhenTheUserSetsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	page, err := fake(t, "").Help("commit")
	if err != nil || !strings.Contains(page, "\nUsage\n") {
		t.Fatalf("Help: %v, %q", err, page)
	}
}

func TestChoicesColumnsAndAnswers(t *testing.T) {
	rows, err := fake(t, "").Choices("packages", "<package>", []string{"<package>=house", "--x=y"})
	if err != nil || len(rows) != 3 {
		t.Fatalf("Choices: %v, %+v", err, rows)
	}
	if rows[0] != (Choice{Value: "a", Description: "the first", State: "active"}) {
		t.Errorf("row 0: %+v", rows[0])
	}
	if rows[1] != (Choice{Value: "b", State: "active", Refusal: "b: kept, c depend(s) on it"}) {
		t.Errorf("row 1: %+v", rows[1])
	}
	// the answers reach kit after a --, so one that starts with - is never read as kit's own
	if rows[2].Description != "packages <package> <package>=house --x=y" {
		t.Errorf("answers: %q", rows[2].Description)
	}
}

func TestExtendBeforeAndAfterTheAnswers(t *testing.T) {
	r := fake(t, "")
	keys, before, err := r.Extend("new", nil)
	if err != nil || len(before) != 0 || len(keys) != 0 {
		t.Fatalf("Extend(nil): %v, %+v", err, before)
	}
	keys, after, err := r.Extend("new", []string{"--rule=markdown"})
	if err != nil || !after["--title"].Extended || strings.Join(keys, " ") != "--title --purpose" {
		t.Fatalf("Extend(rule): %v, %v, %+v", err, keys, after)
	}
}

// TestExecArgsAndDir observes Exec through a helper process, since Exec replaces the process that calls it: the
// helper execs fake-kit, which records its arguments and working directory.
func TestExecArgsAndDir(t *testing.T) {
	record := filepath.Join(t.TempDir(), "record")
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperExec")
	cmd.Env = append(os.Environ(), "KIT_TEST_HELPER=1", "FAKE_KIT_RECORD="+record, "kit_invoked_in="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("helper: %v\n%s", err, out)
	}
	got, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(got)), "\n")
	if len(lines) != 2 || lines[0] != "commit --message a b x.txt" {
		t.Errorf("argv: %q", lines)
	}
	want, _ := filepath.EvalSymlinks(dir)
	if len(lines) == 2 && lines[1] != want {
		t.Errorf("cwd %q, want %q", lines[1], want)
	}
}

// TestHelperExec is the body of TestExecArgsAndDir's helper process; it runs only there.
func TestHelperExec(t *testing.T) {
	if os.Getenv("KIT_TEST_HELPER") != "1" {
		t.Skip("the helper runs under TestExecArgsAndDir alone")
	}
	path, _ := filepath.Abs("testdata/fake-kit")
	os.Setenv("KIT", path)
	r := New()
	if err := r.Exec("commit", []string{"--message", "a b", "x.txt"}); err != nil {
		t.Fatalf("Exec: %v", err)
	}
}

func TestNewReadsTheEnvironment(t *testing.T) {
	t.Setenv("KIT", "/somewhere/kit")
	t.Setenv("kit_invoked_in", "/there")
	r := New()
	if r.Kit != "/somewhere/kit" || r.Dir != "/there" {
		t.Errorf("New: %+v", r)
	}
	t.Setenv("KIT", "")
	t.Setenv("kit_invoked_in", "")
	r = New()
	cwd, _ := os.Getwd()
	if r.Kit != "kit" || r.Dir != cwd {
		t.Errorf("New defaults: %+v", r)
	}
}

func TestMatchJudgesTheWordsAfterADoubleDash(t *testing.T) {
	m, err := fake(t, "").Match("commit", []string{"--message", "hi"})
	if err != nil || m.Status != 0 || m.Operands[0] != "x" || m.Options["--message"][0] != "--message hi" {
		t.Fatalf("Match: %v, %+v", err, m)
	}
}

func TestKindAsksWithTheAnswers(t *testing.T) {
	r := fake(t, "")
	if kind, err := r.Kind("packages", "<package>", []string{"select", "<package>=a"}); err != nil || kind != "set" {
		t.Errorf("Kind(select) = %q, %v", kind, err)
	}
	if kind, err := r.Kind("packages", "<package>", []string{"add"}); err != nil || kind != "closed" {
		t.Errorf("Kind(add) = %q, %v", kind, err)
	}
}

func TestStagesAndStatusRunKitStatus(t *testing.T) {
	r := fake(t, "")
	stages, err := r.Stages()
	if err != nil || strings.Join(stages, " ") != "backlog incoming" {
		t.Fatalf("Stages: %v, %q", err, stages)
	}
	// the stage's words are kit status <stage> --json <args>…, --json first
	ctx := context.Background()
	out, err := r.Status(ctx, "incoming", []string{"--no-fetch"})
	if err != nil || strings.TrimSpace(string(out)) != `{"stage": "incoming", "args": "--json --no-fetch"}` {
		t.Errorf("Status(incoming): %v, %q", err, out)
	}
	out, err = r.Status(ctx, "worktrees", nil)
	if err != nil || strings.TrimSpace(string(out)) != `{"stage": "worktrees", "args": "--json"}` {
		t.Errorf("Status(worktrees): %v, %q", err, out)
	}
	_, err = r.Status(ctx, "nonesuch", nil)
	var refusal *ExitError
	if !errors.As(err, &refusal) || refusal.Code != 2 || !strings.Contains(refusal.Stderr, "no stage 'nonesuch'") {
		t.Errorf("Status(nonesuch): %v", err)
	}
}

func TestStatusEndsWithItsContextAndKitsChildrenDieWithIt(t *testing.T) {
	record := filepath.Join(t.TempDir(), "record")
	t.Setenv("FAKE_KIT_SLEEP", "5")
	t.Setenv("FAKE_KIT_RECORD", record)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := fake(t, "").Status(ctx, "incoming", nil)
	// the sleep holds kit's pipes: Status returns at the deadline only when the sleep is killed with kit
	if took := time.Since(start); !errors.Is(err, context.DeadlineExceeded) || took > time.Second {
		t.Fatalf("Status past its deadline: %v after %s", err, took)
	}
	got, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(got)))
	if err != nil {
		t.Fatal(err)
	}
	for syscall.Kill(pid, 0) == nil {
		if time.Since(start) > 2*time.Second {
			t.Fatalf("the sleep %d outlived kit", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestListReadsEachVerbsNamePurposeAndGroup(t *testing.T) {
	verbs, err := fake(t, "").List()
	if err != nil || len(verbs) != 6 {
		t.Fatalf("List: %v, %+v", err, verbs)
	}
	want := Verb{Name: "commit", Purpose: "commits what is staged, after the checks a commit's rule requires",
		Group: "workflow"}
	if verbs[1] != want {
		t.Errorf("verb 1: %+v", verbs[1])
	}
	// a kit that prints two columns, and no group, leaves the group empty; a line with no tab is no verb
	got := parseList("version\tprints this release's version\n\n")
	if len(got) != 1 || got[0] != (Verb{Name: "version", Purpose: "prints this release's version"}) {
		t.Errorf("two columns: %+v", got)
	}
}

func TestWelcomeChecksDecodesEachCheck(t *testing.T) {
	checks, err := fake(t, "").WelcomeChecks()
	if err != nil || len(checks) != 6 {
		t.Fatalf("WelcomeChecks: %v, %+v", err, checks)
	}
	want := Check{ID: "gh-auth", Step: "setup", Line: "gh is not logged in to github.com",
		Fix: "gh auth login --hostname github.com", Interactive: true, Runs: true}
	if checks[3] != want {
		t.Errorf("check 3: %+v", checks[3])
	}
	// a null fix and a null more are empty
	if checks[4].Fix != "" || checks[4].More == "" || !checks[0].OK {
		t.Errorf("checks 0 and 4: %+v %+v", checks[0], checks[4])
	}
	// a show-only fix says so
	if checks[5].ID != "brew-path" || checks[5].Runs {
		t.Errorf("check 5: %+v", checks[5])
	}
}

func TestWelcomeFixIsACommandNotYetStarted(t *testing.T) {
	record := filepath.Join(t.TempDir(), "record")
	t.Setenv("FAKE_KIT_RECORD", record)
	dir := t.TempDir()
	cmd := fake(t, dir).WelcomeFix("jq")
	if cmd.Process != nil || strings.Join(cmd.Args[1:], " ") != "welcome --fix jq" || cmd.Dir != dir {
		t.Fatalf("WelcomeFix: %q in %q", cmd.Args, cmd.Dir)
	}
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(record)
	want, _ := filepath.EvalSymlinks(dir)
	if string(got) != "welcome --fix jq\n"+want+"\n" {
		t.Errorf("the fix ran as %q", got)
	}
}

func TestRootIsKitRoot(t *testing.T) {
	if root, err := fake(t, "").Root(); err != nil || root != "/the/release" {
		t.Errorf("Root: %q, %v", root, err)
	}
}
