package plan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/empowerite/kit/internal/adapters/fake"
	"github.com/empowerite/kit/internal/ports"
)

var ctx = context.Background()

// world gives fake files, and a git whose clone is at /r and tracks nothing.
func world() (*fake.Files, *fake.Git) {
	return fake.NewFiles(), &fake.Git{RootDir: "/r", Common: "/r/.git", Tracks: map[string]bool{}}
}

func hash(t *testing.T, p Plan) string {
	t.Helper()
	h, err := p.Hash(ctx)
	if err != nil {
		t.Fatalf("Hash gave the error %v", err)
	}
	return h
}

func approve(t *testing.T, p Plan) Approval {
	t.Helper()
	a, err := Approve(ctx, p, &fake.Presence{Proof: ports.Proof{Kind: "touch-id"}})
	if err != nil {
		t.Fatalf("Approve gave the error %v", err)
	}
	return a
}

func TestHashChangesWhenAFactChanges(t *testing.T) {
	files, _ := world()
	p := Plan{Act: "tools add", Operations: []Operation{WriteFile(files, "/r/kit.yml", []byte("tools: {}\n"))}}
	before := hash(t, p)
	if again := hash(t, p); again != before {
		t.Errorf("Hash gave %s, then %s, with nothing changed", before, again)
	}
	if err := files.Write("/r/kit.yml", []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if after := hash(t, p); after == before {
		t.Error("Hash did not change when the file the plan writes changed")
	}
	if other := hash(t, Plan{Act: "tools lock", Operations: p.Operations}); other == hash(t, p) {
		t.Error("Hash did not change with the act")
	}
}

func TestHashRefusesAFactItCannotRead(t *testing.T) {
	files, _ := world()
	files.Fail["read /r/kit.yml"] = errors.New("broken")
	p := Plan{Act: "tools add", Operations: []Operation{WriteFile(files, "/r/kit.yml", nil)}}
	if _, err := p.Hash(ctx); err == nil || !strings.Contains(err.Error(), "broken") {
		t.Errorf("Hash gave the error %v; want broken", err)
	}
	if _, err := Approve(ctx, p, &fake.Presence{}); err == nil {
		t.Error("Approve gave no error for a plan whose facts cannot be read")
	}
	if r := Apply(ctx, p, Approval{}); r.Err == nil || len(r.NotDone) != 1 {
		t.Errorf("Apply = %+v; want an error, and the step not done", r)
	}
}

func TestApproveBindsTheProofToTheHash(t *testing.T) {
	files, _ := world()
	p := Plan{Act: "tools add", Operations: []Operation{WriteFile(files, "/r/kit.yml", []byte("x\n"))}}
	presence := &fake.Presence{Proof: ports.Proof{Kind: "touch-id"}}
	a, err := Approve(ctx, p, presence)
	h := hash(t, p)
	if err != nil || a.Hash != h || a.Proof.Kind != "touch-id" {
		t.Errorf("Approve = %+v, %v; want the hash %s and the proof", a, err, h)
	}
	if want := "approve kit tools add, plan " + h[:12]; !reflect.DeepEqual(presence.Asked, []string{want}) {
		t.Errorf("Approve asked for %q; want %q", presence.Asked, want)
	}
}

func TestApproveRefusesWithNoPresence(t *testing.T) {
	files, _ := world()
	p := Plan{Act: "tools add", Operations: []Operation{WriteFile(files, "/r/kit.yml", nil)}}
	_, err := Approve(ctx, p, &fake.Presence{Err: errors.New("nobody is here")})
	if err == nil || !strings.Contains(err.Error(), "nobody is here") {
		t.Errorf("Approve gave the error %v; want nobody is here", err)
	}
}

func TestApplyRunsEveryStepOfAnApprovedPlan(t *testing.T) {
	files, _ := world()
	p := Plan{Act: "tools add", Operations: []Operation{
		WriteFile(files, "/r/kit.yml", []byte("a\n")), WriteFile(files, "/r/kit.lock", []byte("b\n"))}}
	r := Apply(ctx, p, approve(t, p))
	if r.Err != nil || len(r.Done) != 2 || len(r.NotDone) != 0 {
		t.Errorf("Apply = %+v; want both steps done", r)
	}
	if got, _ := files.Read("/r/kit.lock"); string(got) != "b\n" {
		t.Errorf("kit.lock holds %q; want b", got)
	}
}

func TestApplyRefusesAPlanWhoseFactsChanged(t *testing.T) {
	files, _ := world()
	p := Plan{Act: "tools add", Operations: []Operation{WriteFile(files, "/r/kit.yml", []byte("a\n"))}}
	a := approve(t, p)
	if err := files.Write("/r/kit.yml", []byte("changed meanwhile\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := Apply(ctx, p, a)
	if !errors.Is(r.Err, ErrChanged) || len(r.NotDone) != 1 {
		t.Errorf("Apply = %+v; want ErrChanged and nothing done", r)
	}
	if got, _ := files.Read("/r/kit.yml"); string(got) != "changed meanwhile\n" {
		t.Errorf("Apply wrote over a file whose facts changed: %q", got)
	}
}

func TestApplyStopsAtAFailingStepAndSaysWhatIsDone(t *testing.T) {
	files, _ := world()
	files.Fail["write /r/kit.lock"] = errors.New("disk full")
	p := Plan{Act: "tools add", Operations: []Operation{
		WriteFile(files, "/r/kit.yml", []byte("a\n")),
		WriteFile(files, "/r/kit.lock", []byte("b\n")),
		WriteFile(files, "/r/third", []byte("c\n"))}}
	r := Apply(ctx, p, approve(t, p))
	if r.Err == nil || !strings.Contains(r.Err.Error(), "disk full") || len(r.Done) != 1 || len(r.NotDone) != 2 {
		t.Errorf("Apply = %+v; want disk full, one step done and two not", r)
	}
	if _, err := files.Read("/r/third"); err == nil {
		t.Error("Apply went on past the step that failed")
	}
}

func TestHydrateAppliesAPlanOfHydratingStepsAlone(t *testing.T) {
	files, git := world()
	placing := Plan{Act: "tools fetch", Operations: []Operation{Place(files, git, "/r", "env", []byte("x\n"), "")}}
	if r := Hydrate(ctx, placing); r.Err != nil || len(r.Done) != 1 {
		t.Errorf("Hydrate = %+v; want the step done", r)
	}
	writing := Plan{Act: "tools add", Operations: []Operation{WriteFile(files, "/r/kit.yml", nil)}}
	if r := Hydrate(ctx, writing); r.Err == nil || !strings.Contains(r.Err.Error(), "needs the person's approval") {
		t.Errorf("Hydrate = %+v; want a refusal that says the plan needs approval", r)
	}
	files.Fail["read /r/.kit/env"] = errors.New("broken")
	if r := Hydrate(ctx, placing); r.Err == nil {
		t.Errorf("Hydrate = %+v; want the error of a fact it cannot read", r)
	}
}

func TestPlaceNamesTheFileInKitsGitignoreAndMakesItReadOnly(t *testing.T) {
	files, git := world()
	p := Plan{Act: "tools fetch", Operations: []Operation{
		Place(files, git, "/r", "env", []byte("PATH_add .kit/bin\n"), ""),
		Place(files, git, "/r", "bin/shellcheck", nil, "/cache/kit/abc/kit")}}
	if r := Hydrate(ctx, p); r.Err != nil {
		t.Fatalf("Hydrate gave the error %v", r.Err)
	}
	ignore, _ := files.Read("/r/.kit/.gitignore")
	if want := "# .kit/.gitignore - what kit placed here, which git never sees (written by kit, not by hand)\n" +
		"/.gitignore\n/bin/shellcheck\n/env\n"; string(ignore) != want {
		t.Errorf(".kit/.gitignore = %q; want %q", ignore, want)
	}
	for _, name := range []string{"/r/.kit/.gitignore", "/r/.kit/env"} {
		if info, _ := files.Stat(name); info.Mode().Perm() != 0o444 {
			t.Errorf("%s is %v; want read-only", name, info.Mode())
		}
	}
	if got, _ := files.ReadLink("/r/.kit/bin/shellcheck"); got != "/cache/kit/abc/kit" {
		t.Errorf(".kit/bin/shellcheck leads to %q; want /cache/kit/abc/kit", got)
	}
}

func TestPlaceKeepsTheGitignoreNamingItselfAndPlacesAgainWhatChanged(t *testing.T) {
	files, git := world()
	for _, text := range []string{"one\n", "two\n", "two\n"} {
		p := Plan{Act: "tools fetch", Operations: []Operation{
			Place(files, git, "/r", "env", []byte(text), ""), Place(files, git, "/r", "bin/x", nil, "/c/"+text)}}
		if r := Hydrate(ctx, p); r.Err != nil {
			t.Fatalf("Hydrate gave the error %v", r.Err)
		}
	}
	if got, _ := files.Read("/r/.kit/env"); string(got) != "two\n" {
		t.Errorf(".kit/env = %q; want two", got)
	}
	if got, _ := files.ReadLink("/r/.kit/bin/x"); got != "/c/two\n" {
		t.Errorf(".kit/bin/x leads to %q; want /c/two", got)
	}
	ignore, _ := files.Read("/r/.kit/.gitignore")
	if strings.Count(string(ignore), "/env\n") != 1 || !strings.Contains(string(ignore), "\n/.gitignore\n") {
		t.Errorf(".kit/.gitignore = %q; want /env once, and itself", ignore)
	}
}

func TestPlaceRefusesAFileGitTracks(t *testing.T) {
	for _, tracked := range []string{".kit/env", ".kit/.gitignore"} {
		files, git := world()
		git.Tracks[tracked] = true
		p := Plan{Act: "tools fetch", Operations: []Operation{Place(files, git, "/r", "env", []byte("x\n"), "")}}
		_, err := p.Hash(ctx)
		if err == nil || err.Error() != tracked+" is the repository's own: git tracks it, so kit does not place it" {
			t.Errorf("with %s tracked, Hash gave the error %v; want a refusal that names it", tracked, err)
		}
	}
	files, git := world()
	git.Err = errors.New("not a clone")
	p := Plan{Act: "tools fetch", Operations: []Operation{Place(files, git, "/r", "env", nil, "")}}
	if _, err := p.Hash(ctx); err == nil || !strings.Contains(err.Error(), "not a clone") {
		t.Errorf("Hash gave the error %v; want git's", err)
	}
}

func TestPlaceSaysWhereItFails(t *testing.T) {
	broken := errors.New("broken")
	for _, act := range []string{"write /r/.kit/env", "write /r/.kit/.gitignore", "remove /r/.kit/env", "link /r/.kit/env"} {
		files, git := world()
		if err := files.Write("/r/.kit/env", []byte("old\n"), 0o444); err != nil {
			t.Fatal(err)
		}
		files.Fail[act] = broken
		link := ""
		if strings.HasPrefix(act, "link") {
			link = "/c/kit"
		}
		r := Hydrate(ctx, Plan{Act: "tools fetch", Operations: []Operation{Place(files, git, "/r", "env", []byte("new\n"), link)}})
		if !errors.Is(r.Err, broken) {
			t.Errorf("with %s failing, Hydrate gave %v; want broken", act, r.Err)
		}
	}
}

func TestPlaceSaysWhereItFailsToReplaceAFileWithALink(t *testing.T) {
	files, git := world()
	if err := files.Write("/r/.kit/env", []byte("old\n"), 0o444); err != nil {
		t.Fatal(err)
	}
	files.Fail["remove /r/.kit/env"] = errors.New("broken")
	r := Hydrate(ctx, Plan{Act: "tools fetch", Operations: []Operation{Place(files, git, "/r", "env", nil, "/c/kit")}})
	if r.Err == nil || r.Err.Error() != "broken" {
		t.Errorf("Hydrate gave %v; want broken", r.Err)
	}
}

func TestAnOperationAppliedOnItsOwnSaysWhatItCannotRead(t *testing.T) {
	// Apply reads .kit/.gitignore again: it may have changed since the facts were read.
	files, git := world()
	files.Fail["read /r/.kit/.gitignore"] = errors.New("broken")
	for _, o := range []Operation{Place(files, git, "/r", "env", nil, ""), Unplace(files, git, "/r", "env")} {
		if err := o.Apply(ctx); err == nil || err.Error() != "broken" {
			t.Errorf("%s gave %v; want broken", o.Step().Does, err)
		}
	}
}

func TestUnplaceTakesTheNameOutAgain(t *testing.T) {
	files, git := world()
	place := Plan{Act: "tools fetch", Operations: []Operation{
		Place(files, git, "/r", "env", []byte("x\n"), ""), Place(files, git, "/r", "bin/x", nil, "/c/kit")}}
	if r := Hydrate(ctx, place); r.Err != nil {
		t.Fatal(r.Err)
	}
	if r := Hydrate(ctx, Plan{Act: "tools fetch", Operations: []Operation{Unplace(files, git, "/r", "env")}}); r.Err != nil {
		t.Fatal(r.Err)
	}
	if ignore, _ := files.Read("/r/.kit/.gitignore"); strings.Contains(string(ignore), "/env") {
		t.Errorf(".kit/.gitignore still names env: %q", ignore)
	}
	if _, err := files.Stat("/r/.kit/env"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf(".kit/env is still there: %v", err)
	}
	// With the last of what kit placed taken out, the .gitignore that named them goes too.
	last := Plan{Act: "tools fetch", Operations: []Operation{Unplace(files, git, "/r", "bin/x"), Unplace(files, git, "/r", "never")}}
	if r := Hydrate(ctx, last); r.Err != nil {
		t.Fatal(r.Err)
	}
	if _, err := files.Stat("/r/.kit/.gitignore"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf(".kit/.gitignore is still there: %v", err)
	}
}

func TestUnplaceSaysWhereItFails(t *testing.T) {
	broken := errors.New("broken")
	for _, act := range []string{"remove /r/.kit/env", "remove /r/.kit/.gitignore", "write /r/.kit/.gitignore", "read /r/.kit/.gitignore"} {
		files, git := world()
		place := []Operation{Place(files, git, "/r", "env", []byte("x\n"), "")}
		if act == "write /r/.kit/.gitignore" {
			place = append(place, Place(files, git, "/r", "other", []byte("y\n"), ""))
		}
		if r := Hydrate(ctx, Plan{Act: "tools fetch", Operations: place}); r.Err != nil {
			t.Fatal(r.Err)
		}
		files.Fail[act] = broken
		r := Hydrate(ctx, Plan{Act: "tools fetch", Operations: []Operation{Unplace(files, git, "/r", "env")}})
		if !errors.Is(r.Err, broken) {
			t.Errorf("with %s failing, Hydrate gave %v; want broken", act, r.Err)
		}
	}
}

func TestPrintShowsThePlanAndItsJSON(t *testing.T) {
	files, git := world()
	p := Plan{Act: "tools add", Operations: []Operation{
		WriteFile(files, "/r/kit.yml", []byte("a\n")), Place(files, git, "/r", "env", []byte("x\n"), "")}}
	d, err := p.Doc(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	Print(&out, d)
	want := "Plan  kit tools add, " + d.Hash[:12] + "\n" +
		"  files  write /r/kit.yml\n" +
		"  .kit   place /r/.kit/env, read-only\n"
	if out.String() != want {
		t.Errorf("Print gave\n%s\nwant\n%s", out.String(), want)
	}
	encoded, _ := json.Marshal(d)
	if !strings.Contains(string(encoded), `"system":"files","does":"write /r/kit.yml","undoable":true`) {
		t.Errorf("the plan as JSON is %s", encoded)
	}
	files.Fail["read /r/kit.yml"] = errors.New("broken")
	if _, err := p.Doc(ctx); err == nil {
		t.Error("Doc gave no error for a plan whose facts cannot be read")
	}
}

func TestPrintResultSaysWhatIsDoneAndWhatIsNot(t *testing.T) {
	var out bytes.Buffer
	PrintResult(&out, Result{
		Done:    []Step{{System: "files", Does: "write /r/kit.yml"}},
		NotDone: []Step{{System: "files", Does: "write /r/kit.lock"}},
		Err:     errors.New("disk full")})
	want := "  done      files  write /r/kit.yml\n  not done  files  write /r/kit.lock\nkit: disk full\n"
	if out.String() != want {
		t.Errorf("PrintResult gave %q; want %q", out.String(), want)
	}
}
