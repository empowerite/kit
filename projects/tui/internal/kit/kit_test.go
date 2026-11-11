// The runner asks kit for the grammar, the help, the choices, and the extension, and execs the verb where the command
// was typed; every test runs it against testdata/fake-kit.
package kit

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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
		_, calls["Tutorials"] = r.Tutorials(context.Background(), nil)
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
	if !reflect.DeepEqual(checks[3], want) {
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

// A folder's check names its folder, repository, desk, branch, and kit, and is told from the same check in another
// folder by its key; a machine check, null in each, has its id for its key.
func TestWelcomeChecksReadEachFoldersCheck(t *testing.T) {
	checks, err := script(t, `echo '[{"id": "tool-git", "ok": true, "folder": null, "repository": null, "desk": null,
		"branch": null, "kit": null}, {"id": "kit-check", "ok": false, "folder": "/a/worktrees/t", "repository":
		"github.com/o/a", "desk": false, "branch": "t", "kit": "0.1.0"}, {"id": "kit-check", "ok": true, "folder": "/a",
		"repository": "github.com/o/a", "desk": true, "branch": "main", "kit": null}]'`).WelcomeChecks(context.Background())
	if err != nil || len(checks) != 3 {
		t.Fatalf("WelcomeChecks: %v, %+v", err, checks)
	}
	want := Check{ID: "kit-check", Folder: "/a/worktrees/t", Repository: "github.com/o/a", Branch: "t", Kit: "0.1.0",
		Runs: true}
	if !reflect.DeepEqual(checks[1], want) || !checks[2].Desk || checks[2].Kit != "" {
		t.Errorf("the folders' checks: %+v", checks[1:])
	}
	if checks[0].Key() != "tool-git" || checks[1].Key() == checks[2].Key() ||
		checks[1].Key() != "/a/worktrees/t kit-check" {
		t.Errorf("keys: %q %q %q", checks[0].Key(), checks[1].Key(), checks[2].Key())
	}
}

// A fix names the workspace the checks were read over, so a forge's login finds its host, and a folder's check its
// folder, where kit runs its fix.
func TestWelcomeFixNamesTheWorkspaceAndAFoldersFolder(t *testing.T) {
	r := Runner{Kit: "kit", Dir: "/w", Group: "w"}
	if got := strings.Join(r.WelcomeFix(Check{ID: "kit-hydrated", Folder: "/a/t"}).Args[1:], " "); got !=
		"--workspace=w welcome --fix kit-hydrated --folder=/a/t" {
		t.Errorf("over a group: %s", got)
	}
	r.Group, r.NoGroup = "", true
	if got := strings.Join(r.WelcomeFix(Check{ID: "forge-login-github.com"}).Args[1:], " "); got !=
		"--workspace=none welcome --fix forge-login-github.com" {
		t.Errorf("over no group: %s", got)
	}
}

func TestWelcomeFixIsACommandNotYetStarted(t *testing.T) {
	record := filepath.Join(t.TempDir(), "record")
	t.Setenv("FAKE_KIT_RECORD", record)
	dir := t.TempDir()
	cmd := fake(t, dir).WelcomeFix(Check{ID: "jq"})
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

// Tutorials runs kit tutorials --json with a --tag for each tag, and decodes the catalogue: the audiences, and each
// tutorial with its link, which carries the kit's version; kit's refusal is an ExitError.
func TestTutorialsDecodesTheCatalogue(t *testing.T) {
	record := filepath.Join(t.TempDir(), "record")
	t.Setenv("FAKE_KIT_RECORD", record)
	r := fake(t, "")
	c, err := r.Tutorials(context.Background(), []string{"ui", "newcomer"})
	if err != nil || len(c.Tutorials) != 2 || c.Version != "unreleased" ||
		c.Base != "https://empowerite.github.io/kit/tutorials/main/" {
		t.Fatalf("Tutorials: %v, %+v", err, c)
	}
	if want := (Audience{ID: "newcomer", Label: "Just starting", Color: "#15803d"}); len(c.Audiences) != 3 ||
		c.Audiences[0] != want {
		t.Errorf("audiences: %+v", c.Audiences)
	}
	first := c.Tutorials[0]
	if first.ID != "kit-tutorials" || first.Title != "Reading kit's tutorials" || first.Glyph != "🎓" ||
		!strings.HasPrefix(first.Summary, "How the tutorials") || strings.Join(first.Audience, " ") != "newcomer" {
		t.Errorf("the first tutorial: %+v", first)
	}
	for k, want := range []string{
		"https://empowerite.github.io/kit/tutorials/main/kit-tutorials/?kit=unreleased",
		"https://empowerite.github.io/kit/tutorials/main/kittui-tour/?kit=unreleased",
	} {
		if c.Tutorials[k].URL != want {
			t.Errorf("tutorial %d: %q", k, c.Tutorials[k].URL)
		}
	}
	if got, _ := os.ReadFile(record); string(got) != "tutorials --json --tag ui --tag newcomer\n" {
		t.Errorf("kit ran as %q", got)
	}
	t.Setenv("FAKE_KIT_REFUSE", "tutorials")
	_, err = r.Tutorials(context.Background(), nil)
	var exit *ExitError
	if !errors.As(err, &exit) || exit.Code != 2 || !strings.HasPrefix(err.Error(), "kit tutorials refuses") {
		t.Errorf("a kit that exits 2: %v", err)
	}
}

// A group's reads carry the workspace after their words, and a runner with no group reads where it is.
func TestAGroupsReadsCarryTheWorkspace(t *testing.T) {
	r := fake(t, "")
	r.Group = "w"
	ctx := context.Background()
	out, err := r.Status(ctx, "backlog", []string{"--mine"})
	if err != nil || strings.TrimSpace(string(out)) != `{"stage": "backlog", "args": "--json --mine --workspace w"}` {
		t.Errorf("Status over w: %v, %q", err, out)
	}
	if stages, err := r.Stages(); err != nil || strings.Join(stages, " ") != "backlog incoming" {
		t.Errorf("Stages over w: %v, %q", err, stages)
	}
	r.Group = ""
	out, err = r.Status(ctx, "backlog", []string{"--mine"})
	if err != nil || strings.TrimSpace(string(out)) != `{"stage": "backlog", "args": "--json --mine"}` {
		t.Errorf("Status here: %v, %q", err, out)
	}
}

// Rows runs a command's words as a stage's catalog entry names them: over the group with over, and as they are without,
// as a command that reads no group's members is run.
func TestRowsCarryTheWorkspaceOnlyWhereTheCommandReadsAGroup(t *testing.T) {
	r := fake(t, "")
	r.Group = "w"
	ctx := context.Background()
	out, err := r.Rows(ctx, []string{"flow", "backlog", "--json", "--mine"}, true)
	if err != nil || strings.TrimSpace(string(out)) != `{"stage": "backlog", "args": "--json --mine --workspace w"}` {
		t.Errorf("Rows over w: %v, %q", err, out)
	}
	out, err = r.Rows(ctx, []string{"flow", "backlog", "--json"}, false)
	if err != nil || strings.TrimSpace(string(out)) != `{"stage": "backlog", "args": "--json"}` {
		t.Errorf("Rows not over w: %v, %q", err, out)
	}
}

// A runner whose workspace is no group reads with --workspace none, so kit draws where Dir is and never the last group
// used; a group's name still wins.
func TestARunnerWithNoGroupReadsWithWorkspaceNone(t *testing.T) {
	r := fake(t, "")
	r.NoGroup = true
	out, err := r.Status(context.Background(), "backlog", []string{"--mine"})
	if err != nil || strings.TrimSpace(string(out)) != `{"stage": "backlog", "args": "--json --mine --workspace none"}` {
		t.Errorf("Status with no group: %v, %q", err, out)
	}
	r.Group = "w"
	out, err = r.Status(context.Background(), "backlog", nil)
	if err != nil || strings.TrimSpace(string(out)) != `{"stage": "backlog", "args": "--json --workspace w"}` {
		t.Errorf("Status over w: %v, %q", err, out)
	}
}

// The rate limits and the marks over a group are the group's, as every read of kit flow is; a null mark is an empty
// one.
func TestAGroupsQuotaAndMarksCarryTheWorkspace(t *testing.T) {
	t.Setenv("FAKE_KIT_POLLS", "1")
	r := fake(t, "")
	r.Group = "w"
	ctx := context.Background()
	quotas, err := r.Quota(ctx)
	if err != nil || len(quotas) != 1 || quotas[0].Forge != "--workspace w" || quotas[0].Limits[0].Remaining != 4999 {
		t.Errorf("Quota over w: %v, %+v", err, quotas)
	}
	marks, err := r.Marks(ctx)
	if desk, ok := marks["desk"]; err != nil || marks["backlog"] != "--workspace w" || !ok || desk != "" {
		t.Errorf("Marks over w: %v, %+v", err, marks)
	}
	r.Group = ""
	if marks, err := r.Marks(ctx); err != nil || marks["backlog"] != "" {
		t.Errorf("Marks here: %v, %+v", err, marks)
	}
}

// Without a forge to read, kit refuses the rate limits and the marks, and so does the runner.
func TestQuotaAndMarksAreKitsRefusal(t *testing.T) {
	r := fake(t, "")
	_, err := r.Quota(context.Background())
	if err == nil || !strings.Contains(err.Error(), "does not answer --quota") {
		t.Errorf("Quota: %v", err)
	}
	_, err = r.Marks(context.Background())
	if err == nil || !strings.Contains(err.Error(), "does not answer --marks") {
		t.Errorf("Marks: %v", err)
	}
}

func TestAStageThatPrintsItsDocumentAndExitsOneIsItsDocument(t *testing.T) {
	ctx := context.Background()
	r := script(t, `echo '{"fetched": true, "remotes": [{"name": "fork", "error": "fatal: no route"}]}'; exit 1`)
	if out, err := r.Status(ctx, "incoming", nil); err != nil || !strings.Contains(string(out), "fatal: no route") {
		t.Fatalf("Status = %q, %v", out, err)
	}
	if marks, err := script(t, `echo '{"desk": "1", "changes": null}'; exit 1`).Marks(ctx); marks["desk"] != "1" ||
		len(marks) != 2 || marks["changes"] != "" {
		t.Errorf("Marks = %v, %v", marks, err)
	}
}

func TestARefusalWhatIsNotJSONAndAnotherExitStayTheError(t *testing.T) {
	ctx := context.Background()
	for _, body := range []string{`echo '❌ no gh' >&2; exit 1`, `echo 'not json'; exit 1`, `echo '{}'; exit 2`} {
		if out, err := script(t, body).Status(ctx, "incoming", nil); err == nil {
			t.Errorf("%s: Status = %q, nil", body, out)
		}
	}
}

// What kit printed with exit 1 is a part only where it is JSON that opens an object or an array: blank, a string, and
// a brace that does not close stay kit's refusal, with what it printed on the error.
func TestOnlyAnObjectOrAnArrayIsAPartRead(t *testing.T) {
	for _, stdout := range []string{"  \n", `"text"`, `{"rows": `} {
		exit := &ExitError{Code: 1, Stdout: []byte(stdout)}
		if out, err := partial(nil, exit); err != exit || out != nil {
			t.Errorf("%q: partial = %q, %v", stdout, out, err)
		}
	}
	if out, err := partial(nil, &ExitError{Code: 1, Stdout: []byte(" [1]\n")}); err != nil || string(out) != " [1]\n" {
		t.Errorf("an array: partial = %q, %v", out, err)
	}
}

// Over a group, the marks name the members kit could not read under errors, which is no stage, beside the marks it
// read; the rate limits read are kept as the marks are. A mark that is neither text nor null is not a mark.
func TestAGroupsMarksAndQuotaKeepWhatWasReadBesideTheMembersThatFailed(t *testing.T) {
	ctx := context.Background()
	marks, err := script(t, `echo '{"desk": "1 2", "errors": [{"member": "github.com/o/b", "error": "no gh"}]}'
exit 1`).MarksOf(ctx, "workspace")
	if _, named := marks["errors"]; err != nil || marks["desk"] != "1 2" || named || len(marks) != 1 {
		t.Errorf("MarksOf = %v, %v", marks, err)
	}
	if marks, err := script(t, `echo '{"desk": 7}'`).Marks(ctx); err == nil {
		t.Errorf("a number as a mark: %v, nil", marks)
	}
	// a part that is JSON, but not the marks or the rate limits, is no answer
	if marks, err := script(t, `echo '[1]'; exit 1`).Marks(ctx); err == nil {
		t.Errorf("an array as the marks: %v, nil", marks)
	}
	if quotas, err := script(t, `echo '[1]'; exit 1`).Quota(ctx); err == nil {
		t.Errorf("an array of numbers as the rate limits: %v, nil", quotas)
	}
	quotas, err := script(t, `echo '[{"forge": "github", "limits": []}]'; echo '❌ o/b: no gh' >&2; exit 1`).Quota(ctx)
	if err != nil || len(quotas) != 1 || quotas[0].Forge != "github" {
		t.Errorf("Quota = %+v, %v", quotas, err)
	}
}
