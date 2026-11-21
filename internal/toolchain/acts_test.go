package toolchain

import (
	"errors"
	"strings"
	"testing"

	"github.com/empowerite/kit/internal/adapters/fake"
	"github.com/empowerite/kit/internal/plan"
	"github.com/empowerite/kit/internal/ports"
)

// desk gives a desk at /r/sub of a clone at /r, whose forge holds shellcheck 0.11.0 as kit's list names it.
func desk(t *testing.T) (Desk, map[string]Entry) {
	t.Helper()
	f := newForge()
	e := shellcheck(t, f, "0.11.0", true)
	f.newest["koalaman/shellcheck"] = "v0.11.0"
	files := fake.NewFiles()
	d := Desk{Dir: "/r/sub", Files: files, Git: &fake.Git{RootDir: "/r", Common: "/r/.git", Tracks: map[string]bool{}},
		Releases: f, Downloads: f, Presence: &fake.Presence{Proof: ports.Proof{Kind: "touch-id"}}}
	return d, map[string]Entry{"shellcheck": e}
}

func open(t *testing.T, d Desk) Repo {
	t.Helper()
	r, err := Open(ctx, d)
	if err != nil {
		t.Fatalf("Open gave the error %v", err)
	}
	return r
}

func apply(t *testing.T, d Desk, p plan.Plan) {
	t.Helper()
	a, err := plan.Approve(ctx, p, d.Presence)
	if err != nil {
		t.Fatal(err)
	}
	if r := plan.Apply(ctx, p, a); r.Err != nil {
		t.Fatal(r.Err)
	}
}

func TestOpenFindsTheRootFromASubdirectoryAndReadsNothingThatIsNotThere(t *testing.T) {
	d, _ := desk(t)
	r := open(t, d)
	if r.Root != "/r" || r.Common != "/r/.git" || r.Pins != nil || len(r.Lock.Tools) != 0 || r.KitYML != "" {
		t.Errorf("Open = %+v; want the clone at /r, with nothing pinned", r)
	}
}

func TestOpenSaysWhatItCannotRead(t *testing.T) {
	for _, c := range []struct{ file, text string }{{"/r/kit.yml", "tools: [\n"}, {"/r/kit.lock", "{\"tools\": 1}\n"}} {
		d, _ := desk(t)
		d.Files.Write(c.file, []byte(c.text), 0o644)
		if _, err := Open(ctx, d); err == nil {
			t.Errorf("Open of a broken %s gave no error", c.file)
		}
	}
	d, _ := desk(t)
	d.Files.(*fake.Files).Fail["read /r/kit.yml"] = errors.New("broken")
	if _, err := Open(ctx, d); err == nil || err.Error() != "broken" {
		t.Errorf("Open gave %v; want broken", err)
	}
	delete(d.Files.(*fake.Files).Fail, "read /r/kit.yml")
	d.Files.(*fake.Files).Fail["read /r/kit.lock"] = errors.New("broken")
	if _, err := Open(ctx, d); err == nil || err.Error() != "broken" {
		t.Errorf("Open gave %v; want broken", err)
	}
	d.Git = &fake.Git{Err: errors.New("not a clone")}
	if _, err := Open(ctx, d); err == nil || err.Error() != "not a clone" {
		t.Errorf("Open gave %v; want not a clone", err)
	}
}

func TestAddPinsTheNewestReleaseAndLocksIt(t *testing.T) {
	d, list := desk(t)
	p, l, err := Add(ctx, d, open(t, d), list, "shellcheck", false)
	if err != nil {
		t.Fatalf("Add gave the error %v", err)
	}
	if p.Act != "tools add shellcheck@0.11.0" || len(p.Operations) != 2 || l.Version != "0.11.0" {
		t.Errorf("Add = %s with %d steps, %s; want shellcheck@0.11.0 in two steps", p.Act, len(p.Operations), l.Version)
	}
	apply(t, d, p)
	r := open(t, d)
	if len(r.Pins) != 1 || r.Pins[0].Version != "0.11.0" || len(CheckLock(r.Pins, list, r.Lock)) != 0 {
		t.Errorf("after Add, the repository pins %+v, and the check finds %+v", r.Pins, CheckLock(r.Pins, list, r.Lock))
	}
	// Added again, nothing changes, so the plan has no step.
	if p, _, _ := Add(ctx, d, r, list, "shellcheck@0.11.0", false); len(p.Operations) != 0 {
		t.Errorf("Add of what is pinned already has %d steps; want none", len(p.Operations))
	}
}

func TestAddRefusesWhatItCannotPin(t *testing.T) {
	d, list := desk(t)
	r := open(t, d)
	for _, c := range []struct{ tool, want string }{
		{"mystery", "kit's list holds no mystery"},
		{"shellcheck@9.9.9", "has no file"},
	} {
		if _, _, err := Add(ctx, d, r, list, c.tool, false); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("Add(%s) gave %v; want %q", c.tool, err, c.want)
		}
	}
	d.Releases.(*forge).newest["koalaman/shellcheck"] = "release-0.11.0"
	if _, _, err := Add(ctx, d, r, list, "shellcheck", false); err == nil || !strings.Contains(err.Error(), "which v{version} does not name") {
		t.Errorf("Add of a newest tag the entry does not name gave %v", err)
	}
	delete(d.Releases.(*forge).newest, "koalaman/shellcheck")
	if _, _, err := Add(ctx, d, r, list, "shellcheck", false); err == nil || !strings.Contains(err.Error(), "has no release") {
		t.Errorf("Add with no newest release gave %v", err)
	}
	r.KitYML = "[a, b]\n"
	if _, _, err := Add(ctx, d, r, list, "shellcheck@0.11.0", false); err == nil {
		t.Error("Add into a kit.yml that is not a mapping gave no error")
	}
}

func TestLockLocksEveryPinAgain(t *testing.T) {
	d, list := desk(t)
	p, _, _ := Add(ctx, d, open(t, d), list, "shellcheck", false)
	apply(t, d, p)
	d.Files.Write("/r/kit.lock", []byte("{\"tools\": {}}\n"), 0o644)
	r := open(t, d)
	p, err := Relock(ctx, d, r, list, false)
	if err != nil || p.Act != "tools lock" || len(p.Operations) != 1 {
		t.Fatalf("Relock = %+v, %v; want one step", p, err)
	}
	apply(t, d, p)
	r = open(t, d)
	if found := CheckLock(r.Pins, list, r.Lock); len(found) != 0 {
		t.Errorf("after Relock, the check finds %+v", found)
	}
	if _, err := Relock(ctx, d, r, map[string]Entry{}, false); err == nil || !strings.Contains(err.Error(), "kit's list holds no shellcheck") {
		t.Errorf("Relock of a pin no list holds gave %v", err)
	}
	d.Releases.(*forge).fail = errors.New("offline")
	if _, err := Relock(ctx, d, r, list, false); err == nil {
		t.Error("Relock with the forge away gave no error")
	}
}
