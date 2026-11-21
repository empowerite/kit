package toolchain

import (
	"errors"
	"strings"
	"testing"

	"github.com/empowerite/kit/internal/adapters/fake"
	"github.com/empowerite/kit/internal/plan"
	"github.com/empowerite/kit/internal/ports"
)

// trust has the person trust the clone the desk is in, and gives what kit said changed.
func trust(t *testing.T, d Desk) []string {
	t.Helper()
	p, changes, err := TrustPlan(ctx, d, open(t, d))
	if err != nil {
		t.Fatalf("TrustPlan gave the error %v", err)
	}
	apply(t, d, p)
	return changes
}

func TestNothingRunsInACloneThePersonHasNotTrusted(t *testing.T) {
	d := pinned(t)
	d.Once = false
	want := "kit runs nothing in this clone until you trust it"
	if _, err := Fetch(ctx, d, open(t, d)); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Fetch gave %v; want %q", err, want)
	}
	if err := RunTool(ctx, d, "/r/.kit/bin/shellcheck", nil); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("RunTool gave %v; want %q", err, want)
	}
	if err := Exec(ctx, d, open(t, d), []string{"x"}); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Exec gave %v; want %q", err, want)
	}
	trust(t, d)
	if _, err := Fetch(ctx, d, open(t, d)); err != nil {
		t.Errorf("once trusted, Fetch gave %v", err)
	}
}

func TestTrustHoldsForEveryWorktreeOfTheClone(t *testing.T) {
	d := pinned(t)
	d.Once = false
	trust(t, d)
	// A worktree of the same clone, with the same pins: its root differs, its common directory does not.
	lock, _ := d.Files.Read("/r/kit.lock")
	yml, _ := d.Files.Read("/r/kit.yml")
	d.Files.Write("/r/worktrees/7-x/kit.lock", lock, 0o644)
	d.Files.Write("/r/worktrees/7-x/kit.yml", yml, 0o644)
	d.Git = &fake.Git{RootDir: "/r/worktrees/7-x", Common: "/r/.git", Tracks: map[string]bool{}}
	if _, err := Fetch(ctx, d, open(t, d)); err != nil {
		t.Errorf("in a worktree of a trusted clone, Fetch gave %v", err)
	}
	// Another clone is not trusted for it.
	d.Git = &fake.Git{RootDir: "/r/worktrees/7-x", Common: "/other/.git", Tracks: map[string]bool{}}
	if _, err := Fetch(ctx, d, open(t, d)); err == nil {
		t.Error("in another clone, Fetch ran with no trust")
	}
}

func TestTrustHoldsSeveralDigestsAtOnce(t *testing.T) {
	d := pinned(t)
	d.Once = false
	trust(t, d)
	first, _ := d.Files.Read("/r/kit.yml")
	d.Files.Write("/r/kit.yml", append(first, []byte("# a worktree's own change\n")...), 0o644)
	trust(t, d)
	d.Files.Write("/r/kit.yml", first, 0o644)
	if err := Allowed(ctx, d, open(t, d)); err != nil {
		t.Errorf("the first set of contents is no longer trusted: %v", err)
	}
}

func TestAChangedPinAsksAgainAndSaysWhatChanged(t *testing.T) {
	d := pinned(t)
	d.Once = false
	if changes := trust(t, d); !strings.Contains(strings.Join(changes, "\n"), "shellcheck 0.11.0 is new, from:") {
		t.Errorf("the first trust said %q; want shellcheck named as new", changes)
	}
	lock := open(t, d).Lock
	l := lock.Tools["shellcheck"]
	l.Version = "0.12.0"
	f := l.Files["linux/amd64"]
	f.URL = "https://example.com/elsewhere.tar.gz"
	l.Files["linux/amd64"] = f
	lock.Tools["shellcheck"] = l
	lock.Tools["lychee"] = l
	d.Files.Write("/r/kit.lock", lock.JSON(), 0o644)
	if err := Allowed(ctx, d, open(t, d)); err == nil || !strings.Contains(err.Error(), "has changed since") {
		t.Errorf("Allowed gave %v; want that what was trusted has changed", err)
	}
	_, changes, _ := TrustPlan(ctx, d, open(t, d))
	said := strings.Join(changes, "\n")
	for _, want := range []string{"shellcheck 0.12.0, which was 0.11.0, from:", "https://example.com/elsewhere.tar.gz",
		"lychee 0.12.0 is new, from:", "kit.lock changed"} {
		if !strings.Contains(said, want) {
			t.Errorf("TrustPlan said %q; want %q", said, want)
		}
	}
	delete(lock.Tools, "shellcheck")
	d.Files.Write("/r/kit.lock", lock.JSON(), 0o644)
	d.Files.Remove("/r/kit.yml")
	_, changes, _ = TrustPlan(ctx, d, open(t, d))
	if said := strings.Join(changes, "\n"); !strings.Contains(said, "shellcheck is no longer pinned") || !strings.Contains(said, "kit.yml is gone") {
		t.Errorf("TrustPlan said %q", said)
	}
}

func TestTrustCheckoutLastsOneRunAndRecordsNothing(t *testing.T) {
	d := pinned(t)
	d.Once = true
	if _, err := Fetch(ctx, d, open(t, d)); err != nil {
		t.Errorf("with the checkout trusted for this run, Fetch gave %v", err)
	}
	if _, err := d.Files.Read("/state/kit/trust.json"); err == nil {
		t.Error("trusting the checkout for one run recorded a trust")
	}
	d.Once = false
	if _, err := Fetch(ctx, d, open(t, d)); err == nil {
		t.Error("the next run, with no trust given, ran")
	}
}

func TestTrustRunsDirenvAllowWhereThereIsAnEnvrc(t *testing.T) {
	d := pinned(t)
	d.Once = false
	d.Files.Write("/r/.envrc", []byte("eval \"$(kit env --shell=direnv)\"\n"), 0o644)
	runs := d.Processes.(*fake.Processes)
	runs.Answer = func(c ports.Cmd) (int, error) {
		if c.Args[0] == "status" {
			c.Stdout.Write([]byte(`{"state": {"foundRC": {"allowed": 1, "path": "/r/.envrc"}}}`))
		}
		return 0, nil
	}
	p, _, err := TrustPlan(ctx, d, open(t, d))
	if err != nil {
		t.Fatal(err)
	}
	apply(t, d, p)
	last := runs.Runs[len(runs.Runs)-1]
	if last.Path != "direnv" || strings.Join(last.Args, " ") != "allow /r/.envrc" {
		t.Errorf("kit trust ran %s %q; want direnv allow /r/.envrc", last.Path, last.Args)
	}
	if err := Allowed(ctx, d, open(t, d)); err != nil {
		t.Errorf("once trusted with .envrc, Allowed gave %v", err)
	}
	runs.Answer = func(ports.Cmd) (int, error) { return 1, nil }
	p, _, _ = TrustPlan(ctx, d, open(t, d))
	a, _ := plan.Approve(ctx, p, d.Presence)
	if r := plan.Apply(ctx, p, a); r.Err == nil || !strings.Contains(r.Err.Error(), "direnv allow") {
		t.Errorf("with direnv allow failing, apply gave %v", r.Err)
	}
}

func TestNoEnvironmentVariableGrantsTrust(t *testing.T) {
	d := pinned(t)
	d.Once = false
	d.Machine.(*fake.Machine).Env = []string{"KIT_TRUST=checkout", "KIT_TRUSTED=1", "CI=true", "GITHUB_ACTIONS=true"}
	if err := Allowed(ctx, d, open(t, d)); err == nil {
		t.Error("an environment variable granted trust")
	}
}

func TestKitSaysWhatDirenvHasNotAllowed(t *testing.T) {
	d := pinned(t)
	runs := d.Processes.(*fake.Processes)
	for _, c := range []struct {
		said string
		code int
		err  error
		want string
	}{
		{`{"state": {"foundRC": {"allowed": 0}}}`, 0, nil, "direnv has allowed this .envrc"},
		{`{"state": {"foundRC": {"allowed": 1}}}`, 0, nil, "direnv has not allowed this .envrc: kit trust covers it"},
		{`{"state": {"foundRC": {"allowed": 2}}}`, 0, nil, "direnv has denied this .envrc: kit trust covers it"},
		{`{"state": {"foundRC": null}}`, 0, nil, "direnv finds no .envrc here"},
		{``, 0, errors.New("not found"), "direnv is not installed here"},
		{`not json`, 0, nil, "direnv's status cannot be read"},
		{``, 1, nil, "direnv's status cannot be read"},
	} {
		runs.Answer = func(cmd ports.Cmd) (int, error) { cmd.Stdout.Write([]byte(c.said)); return c.code, c.err }
		if got := DirenvState(ctx, d, "/r"); got != c.want {
			t.Errorf("with direnv saying %q, DirenvState = %q; want %q", c.said, got, c.want)
		}
	}
}

func TestTrustSaysWhatItCannotRead(t *testing.T) {
	d := pinned(t)
	d.Once = false
	d.Machine.(*fake.Machine).Err = errors.New("no home")
	if _, _, err := TrustPlan(ctx, d, open(t, d)); err == nil {
		t.Error("TrustPlan with no state gave no error")
	}
	if err := Allowed(ctx, d, open(t, d)); err == nil {
		t.Error("Allowed with no state gave no error")
	}
	d.Machine.(*fake.Machine).Err = nil
	d.Files.Write("/state/kit/trust.json", []byte("not json"), 0o644)
	if _, _, err := TrustPlan(ctx, d, open(t, d)); err == nil || !strings.Contains(err.Error(), "trust.json") {
		t.Errorf("TrustPlan with a broken store gave %v", err)
	}
	d.Files.Remove("/state/kit/trust.json")
	d.Files.(*fake.Files).Fail["read /r/.envrc"] = errors.New("broken")
	if _, _, err := TrustPlan(ctx, d, open(t, d)); err == nil {
		t.Error("TrustPlan with .envrc unreadable gave no error")
	}
	if err := Allowed(ctx, d, open(t, d)); err == nil {
		t.Error("Allowed with .envrc unreadable gave no error")
	}
}
