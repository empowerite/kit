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
	// the stage's words are kit flow <stage> --json <args>…, --json first
	ctx := context.Background()
	out, err := r.Status(ctx, "incoming", []string{"--no-fetch"})
	if err != nil || strings.TrimSpace(string(out)) != `{"stage": "incoming", "args": "--json --no-fetch"}` {
		t.Errorf("Status(incoming): %v, %q", err, out)
	}
	out, err = r.Status(ctx, "worktrees", nil)
	if err != nil || strings.TrimSpace(string(out)) != `{"stage": "worktrees", "args": "--json"}` {
		t.Errorf("Status(worktrees): %v, %q", err, out)
	}
	if me, err := r.Me(ctx); err != nil || me != "ada" {
		t.Errorf("Me: %v, %q", err, me)
	}
	// the trackers are read from kit's JSON alone, a warning on stderr apart; a refusal, or what is not JSON, is the
	// error
	if trackers, err := r.Trackers(ctx); err != nil || len(trackers) != 1 || trackers[0].Prefix != "#" ||
		trackers[0].URL != "https://github.com/empowerite/kit/issues/new" {
		t.Errorf("Trackers: %v, %+v", err, trackers)
	}
	t.Setenv("FAKE_KIT_REFUSE", "new")
	if _, err := r.Trackers(ctx); err == nil || !strings.Contains(err.Error(), "the forge cannot be read") {
		t.Errorf("Trackers refused: %v", err)
	}
	t.Setenv("FAKE_KIT_REFUSE", "")
	t.Setenv("FAKE_KIT_UNPARSED", "new")
	if _, err := r.Trackers(ctx); err == nil || !strings.Contains(err.Error(), "kit new issue --list --json") {
		t.Errorf("Trackers unparsed: %v", err)
	}
	t.Setenv("FAKE_KIT_UNPARSED", "")
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

// script gives a Runner over a shell script of the body, standing in for a kit that misbehaves.
func script(t *testing.T, body string) Runner {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kit")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return Runner{Kit: path, Dir: t.TempDir()}
}

// A kit that cannot run is an error from every call, and so is one that prints what is not the JSON asked for; a
// check whose fields are of the wrong type is one too. Exec into a directory that is not there fails before it runs.
func TestEveryCallFailsWhenKitCannotRunOrPrintsWhatIsNotItsJSON(t *testing.T) {
	missing := Runner{Kit: filepath.Join(t.TempDir(), "no-kit"), Dir: t.TempDir()}
	garbled := script(t, "echo '{not json'")
	for _, r := range []Runner{missing, garbled} {
		calls := map[string]error{}
		_, calls["Verbs"] = r.Verbs()
		_, calls["Spec"] = r.Spec("commit")
		_, _, calls["Extend"] = r.Extend("new", nil)
		_, calls["Match"] = r.Match("commit", nil)
		_, calls["WelcomeChecks"] = r.WelcomeChecks(context.Background())
		if r.Kit == missing.Kit {
			_, calls["List"] = r.List()
			_, calls["Choices"] = r.Choices("commit", "<path>", nil)
			_, calls["Stages"] = r.Stages()
			_, calls["Root"] = r.Root()
			_, calls["Help"] = r.Help("commit")
		}
		for call, err := range calls {
			if err == nil {
				t.Errorf("%s over %s: no error", call, filepath.Base(r.Kit))
			}
		}
	}
	if _, err := script(t, `echo '[{"id": "jq", "ok": "yes"}]'`).WelcomeChecks(context.Background()); err == nil ||
		!strings.Contains(err.Error(), "kit welcome --checks --json") {
		t.Errorf("a check with ok a string: %v", err)
	}
	r := script(t, "exit 0")
	r.Dir = filepath.Join(t.TempDir(), "gone")
	if err := r.Exec("commit", nil); err == nil {
		t.Error("exec into a directory that is not there did not fail")
	}
	if err := missing.Exec("commit", nil); err == nil {
		t.Error("exec of a kit that is not there did not fail")
	}
}

// A kit that hangs is stopped at the call's limit, its process group with it, and the error says how long it had;
// the checks are stopped too when the screen that asked for them ends.
func TestAKitThatHangsIsStoppedAtItsLimitOrWhenTheScreenEnds(t *testing.T) {
	limit0 := callLimit
	t.Cleanup(func() { callLimit = limit0 })
	callLimit = 300 * time.Millisecond
	t.Setenv("FAKE_KIT_HANG", "5")
	start := time.Now()
	_, err := fake(t, "").Spec("commit")
	if took := time.Since(start); took > 3*time.Second || err == nil ||
		!strings.Contains(err.Error(), "kit syntax took longer than 300ms") {
		t.Errorf("a hung kit: %v after %s", err, took)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()
	start = time.Now()
	if _, err := fake(t, "").WelcomeChecks(ctx); !errors.Is(err, context.Canceled) || time.Since(start) > 3*time.Second {
		t.Errorf("checks whose screen ended: %v after %s", err, time.Since(start))
	}
}

func TestWelcomeChecksDecodesEachCheck(t *testing.T) {
	checks, err := fake(t, "").WelcomeChecks(context.Background())
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

// Act runs a verb that acts and gives everything kit printed; a refusal is kit's ❌ line, even with lines after it.
func TestActGivesWhatKitSaidAndItsRefusal(t *testing.T) {
	r := fake(t, "")
	out, err := r.Act(context.Background(), "assign", "#7")
	if err != nil || !strings.Contains(out, "out assign #7") || !strings.Contains(out, "✅ kit assign #7 done") {
		t.Fatalf("assign: %q, %v", out, err)
	}
	t.Setenv("FAKE_KIT_REFUSE", "close")
	out, err = r.Act(context.Background(), "close", "--not-planned", "#7")
	var exit *ExitError
	if !errors.As(err, &exit) || exit.Code != 1 || err.Error() != "kit close refuses for the test" ||
		!strings.Contains(out, "and says more") {
		t.Errorf("close: %q, %v", out, err)
	}
	if refusal("→ a step\n  the last word") != "  the last word" {
		t.Error("a refusal with no ❌ line is not the last line")
	}
	if _, err = (Runner{Kit: filepath.Join(t.TempDir(), "none")}).Act(context.Background(), "assign"); err == nil ||
		errors.As(err, &exit) {
		t.Errorf("a kit that is not there: %v", err)
	}
}
