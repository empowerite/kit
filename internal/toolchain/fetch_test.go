package toolchain

import (
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/empowerite/kit/internal/adapters/fake"
	"github.com/empowerite/kit/internal/plan"
)

// pinned gives a desk on linux/amd64 whose repository at /r pins shellcheck 0.11.0, locked, with kit itself at
// /usr/bin/kit.
func pinned(t *testing.T) Desk {
	t.Helper()
	d, list := desk(t)
	d.Machine = &fake.Machine{OS: "linux", Arch: "amd64", CacheDir: "/cache/kit", StateDir: "/state/kit", Kit: "/usr/bin/kit",
		Env: []string{"HOME=/home/a", "PATH=/usr/bin"}}
	d.Processes = &fake.Processes{}
	d.Files.Write("/usr/bin/kit", []byte("kit itself"), 0o755)
	p, _, err := Add(ctx, d, open(t, d), list, "shellcheck@0.11.0", false)
	if err != nil {
		t.Fatal(err)
	}
	apply(t, d, p)
	return d
}

func hydrate(t *testing.T, d Desk) {
	t.Helper()
	p, err := Fetch(ctx, d, open(t, d))
	if err != nil {
		t.Fatalf("Fetch gave the error %v", err)
	}
	if r := plan.Hydrate(ctx, p); r.Err != nil {
		t.Fatalf("hydrating the fetch gave the error %v", r.Err)
	}
}

// program gives where the cache keeps shellcheck's program for linux/amd64.
func program(t *testing.T, d Desk) string {
	t.Helper()
	f := open(t, d).Lock.Tools["shellcheck"].Files["linux/amd64"]
	return "/cache/kit/tools/" + strings.TrimPrefix(f.Digest, "sha256:") + "/shellcheck"
}

func TestFetchNamesAFolderForItsDigestOnlyWhenWhole(t *testing.T) {
	d := pinned(t)
	hydrate(t, d)
	got, err := d.Files.Read(program(t, d))
	if err != nil || string(got) != "shellcheck 0.11.0 for linux/amd64" {
		t.Errorf("the cache holds %q, %v; want shellcheck's program", got, err)
	}
	if info, _ := d.Files.Stat(program(t, d)); info.Mode().Perm() != 0o555 {
		t.Errorf("the program is %v; want read-only, and runnable", info.Mode())
	}
	for _, p := range d.Files.(*fake.Files).Paths() {
		if strings.HasSuffix(p, ".part") {
			t.Errorf("a fetch left %s behind", p)
		}
	}
}

func TestFetchMakesAgainAPartLeftByAStoppedFetch(t *testing.T) {
	d := pinned(t)
	dir := strings.TrimSuffix(program(t, d), "/shellcheck")
	d.Files.Write(dir+".part/shellcheck", []byte("half"), 0o644)
	hydrate(t, d)
	if got, _ := d.Files.Read(program(t, d)); string(got) != "shellcheck 0.11.0 for linux/amd64" {
		t.Errorf("the cache holds %q; want the whole program", got)
	}
	// Whole, it is not fetched again: the forge could be gone.
	d.Downloads = failingDownloads{}
	hydrate(t, d)
}

func TestFetchPlacesALinkToAKeptCopyOfKit(t *testing.T) {
	d := pinned(t)
	hydrate(t, d)
	kept := "/cache/kit/kit/" + strings.TrimPrefix(digestOf([]byte("kit itself")), "sha256:") + "/kit"
	if got, _ := d.Files.ReadLink("/r/.kit/bin/shellcheck"); got != kept {
		t.Errorf(".kit/bin/shellcheck leads to %q; want %q", got, kept)
	}
	if got, _ := d.Files.Read(kept); string(got) != "kit itself" {
		t.Errorf("the kept kit holds %q", got)
	}
	// The kit that placed the link may go, as bin/gate's does: the link leads to the copy, which stays.
	d.Files.Remove("/usr/bin/kit")
	if got, _ := d.Files.Read("/r/.kit/bin/shellcheck"); string(got) != "kit itself" {
		t.Errorf("through the link, the files hold %q; want kit", got)
	}
	if env, _ := d.Files.Read("/r/.kit/env"); string(env) != "PATH_add .kit/bin\nwatch_file kit.yml kit.lock\n" {
		t.Errorf(".kit/env holds %q", env)
	}
}

func TestFetchRefusesADownloadThatIsNotWhatTheLockNames(t *testing.T) {
	d := pinned(t)
	f := open(t, d).Lock.Tools["shellcheck"].Files["linux/amd64"]
	d.Downloads.(*forge).files[f.URL] = []byte("other bytes")
	p, _ := Fetch(ctx, d, open(t, d))
	if r := plan.Hydrate(ctx, p); r.Err == nil || !strings.Contains(r.Err.Error(), "kit.lock names "+f.Digest) {
		t.Errorf("hydrating the fetch gave %v; want one that names the lock's digest", r.Err)
	}
	d.Downloads.(*forge).files[f.URL] = tgz(t, map[string]string{"shellcheck-v0.11.0/shellcheck": "another program"})
	lock := open(t, d).Lock
	l := lock.Tools["shellcheck"]
	f.Digest = digestOf(d.Downloads.(*forge).files[f.URL])
	l.Files["linux/amd64"] = f
	lock.Tools["shellcheck"] = l
	d.Files.Write("/r/kit.lock", lock.JSON(), 0o644)
	p, _ = Fetch(ctx, d, open(t, d))
	if r := plan.Hydrate(ctx, p); r.Err == nil || !strings.Contains(r.Err.Error(), "is not the program kit.lock names") {
		t.Errorf("hydrating the fetch gave %v; want one that says the program is not the lock's", r.Err)
	}
	// A file whose digest is the lock's, and that is no archive, as a lock written by hand could name.
	d.Downloads.(*forge).files[f.URL] = []byte("no archive")
	f.Digest = digestOf([]byte("no archive"))
	l.Files["linux/amd64"] = f
	d.Files.Write("/r/kit.lock", lock.JSON(), 0o644)
	p, _ = Fetch(ctx, d, open(t, d))
	if r := plan.Hydrate(ctx, p); r.Err == nil {
		t.Error("hydrating the fetch of a file that is no archive gave no error")
	}
}

func TestFetchSaysWhatStandsInItsWay(t *testing.T) {
	d := pinned(t)
	d.Machine.(*fake.Machine).Arch = "riscv64"
	if _, err := Fetch(ctx, d, open(t, d)); err == nil || !strings.Contains(err.Error(), "holds no file of shellcheck for linux/riscv64") {
		t.Errorf("Fetch on a platform the lock does not cover gave %v", err)
	}
	d.Machine.(*fake.Machine).Arch = "amd64"
	d.Machine.(*fake.Machine).Err = errors.New("no home")
	if _, err := Fetch(ctx, d, open(t, d)); err == nil || err.Error() != "no home" {
		t.Errorf("Fetch with no cache gave %v", err)
	}
	d.Machine.(*fake.Machine).Err = nil
	d.Files.(*fake.Files).Fail["read /usr/bin/kit"] = errors.New("broken")
	if _, err := Fetch(ctx, d, open(t, d)); err == nil || err.Error() != "broken" {
		t.Errorf("Fetch with kit unreadable gave %v", err)
	}
	delete(d.Files.(*fake.Files).Fail, "read /usr/bin/kit")
	for _, act := range []string{"write", "rename", "remove"} {
		e := pinned(t)
		dir := strings.TrimSuffix(program(t, e), "/shellcheck")
		e.Files.Write(dir+".part/shellcheck", []byte("half"), 0o644)
		target := map[string]string{"write": dir + ".part/shellcheck", "rename": dir + ".part", "remove": dir + ".part/shellcheck"}[act]
		e.Files.(*fake.Files).Fail[act+" "+target] = errors.New("broken")
		p, _ := Fetch(ctx, e, open(t, e))
		if r := plan.Hydrate(ctx, p); r.Err == nil {
			t.Errorf("with %s failing, hydrating the fetch gave no error", act)
		}
	}
	e := pinned(t)
	e.Files.(*fake.Files).Fail["stat /cache/kit/kit/"+strings.TrimPrefix(digestOf([]byte("kit itself")), "sha256:")] = errors.New("broken")
	p, _ := Fetch(ctx, e, open(t, e))
	if r := plan.Hydrate(ctx, p); r.Err == nil {
		t.Error("with the cache unreadable, hydrating the fetch gave no error")
	}
	e = pinned(t)
	e.Downloads = failingDownloads{}
	p, _ = Fetch(ctx, e, open(t, e))
	if r := plan.Hydrate(ctx, p); r.Err == nil || !strings.Contains(r.Err.Error(), "offline") {
		t.Errorf("with the forge away, hydrating the fetch gave %v", r.Err)
	}
}

func TestRunToolRunsTheProgramWithTheSameArguments(t *testing.T) {
	d := pinned(t)
	hydrate(t, d)
	if err := RunTool(ctx, d, "/r/.kit/bin/shellcheck", []string{"--version"}); err != nil {
		t.Fatalf("RunTool gave the error %v", err)
	}
	got := d.Processes.(*fake.Processes).Replaced
	if len(got) != 1 || got[0].Path != program(t, d) || !reflect.DeepEqual(got[0].Args, []string{"--version"}) ||
		!reflect.DeepEqual(got[0].Env, []string{"HOME=/home/a", "PATH=/usr/bin"}) {
		t.Errorf("RunTool ran %+v; want shellcheck, with --version", got)
	}
}

func TestRunToolSaysAProgramItCannotRead(t *testing.T) {
	d := pinned(t)
	hydrate(t, d)
	d.Files.(*fake.Files).Fail["read "+program(t, d)] = errors.New("broken")
	if err := RunTool(ctx, d, "shellcheck", nil); err == nil || err.Error() != "broken" {
		t.Errorf("RunTool gave %v; want broken", err)
	}
}

func TestRunToolFetchesWhatIsNotFetchedYet(t *testing.T) {
	d := pinned(t)
	if err := RunTool(ctx, d, "shellcheck", nil); err != nil {
		t.Fatalf("RunTool gave the error %v", err)
	}
	if _, err := d.Files.Read(program(t, d)); err != nil {
		t.Errorf("RunTool did not fetch: %v", err)
	}
}

func TestRunToolRefusesAProgramWhoseDigestDiffers(t *testing.T) {
	d := pinned(t)
	hydrate(t, d)
	d.Files.Chmod(program(t, d), 0o755)
	d.Files.Write(program(t, d), []byte("changed in the cache"), 0o755)
	err := RunTool(ctx, d, "shellcheck", nil)
	if err == nil || !strings.Contains(err.Error(), "is not the program kit.lock names") {
		t.Errorf("RunTool gave %v; want a refusal", err)
	}
	if len(d.Processes.(*fake.Processes).Replaced) != 0 {
		t.Error("RunTool ran a program whose digest differs")
	}
}

func TestRunToolSaysWhatStandsInItsWay(t *testing.T) {
	d := pinned(t)
	for _, c := range []struct {
		argv0, want string
	}{
		{"lychee", "holds no lychee"},
		{"/elsewhere/.kit/bin/shellcheck", "/elsewhere"},
	} {
		if err := RunTool(ctx, d, c.argv0, nil); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("RunTool(%s) gave %v; want %q", c.argv0, err, c.want)
		}
	}
	d.Machine.(*fake.Machine).Arch = "riscv64"
	if err := RunTool(ctx, d, "shellcheck", nil); err == nil || !strings.Contains(err.Error(), "linux/riscv64") {
		t.Errorf("RunTool on a platform the lock does not cover gave %v", err)
	}
	d.Machine.(*fake.Machine).Arch = "amd64"
	d.Downloads = failingDownloads{}
	if err := RunTool(ctx, d, "shellcheck", nil); err == nil || !strings.Contains(err.Error(), "offline") {
		t.Errorf("RunTool with nothing fetched and the forge away gave %v", err)
	}
	d.Machine.(*fake.Machine).Err = errors.New("no home")
	if err := RunTool(ctx, d, "shellcheck", nil); err == nil {
		t.Error("RunTool with no cache gave no error")
	}
	d.Machine.(*fake.Machine).Err = nil
	d.Git = &fake.Git{Err: errors.New("not a clone")}
	if err := RunTool(ctx, d, "shellcheck", nil); err == nil || err.Error() != "not a clone" {
		t.Errorf("RunTool outside a clone gave %v", err)
	}
}

func TestRunToolFindsTheRootFromAPathRelativeToWhereKitRuns(t *testing.T) {
	d := pinned(t)
	d.Dir = "/r/sub"
	d.Git = &fake.Git{Err: errors.New("git is not asked")}
	hydrate(t, Desk{Dir: "/r", Files: d.Files, Git: &fake.Git{RootDir: "/r", Common: "/r/.git", Tracks: map[string]bool{}},
		Releases: d.Releases, Downloads: d.Downloads, Machine: d.Machine})
	if err := RunTool(ctx, d, "../.kit/bin/shellcheck", nil); err != nil {
		t.Errorf("RunTool by a relative path gave %v", err)
	}
	if _, err := d.Files.Stat("/r/.kit"); errors.Is(err, fs.ErrNotExist) {
		t.Error("nothing was placed")
	}
}

func TestEnvPrintsWhatEachShellReads(t *testing.T) {
	for shell, want := range map[string]string{
		"bash":   "export PATH='/r/my repo/.kit/bin':\"$PATH\"\n",
		"zsh":    "export PATH='/r/my repo/.kit/bin':\"$PATH\"\n",
		"direnv": "PATH_add .kit/bin\nwatch_file kit.yml kit.lock\n",
	} {
		if got, err := Env(shell, "/r/my repo"); got != want || err != nil {
			t.Errorf("Env(%s) = %q, %v; want %q", shell, got, err, want)
		}
	}
	if got, _ := Env("bash", "/r/it's"); got != "export PATH='/r/it'\\''s/.kit/bin':\"$PATH\"\n" {
		t.Errorf("Env of a root with a quote = %q", got)
	}
	if _, err := Env("fish", "/r"); err == nil || !strings.Contains(err.Error(), "bash, zsh, and direnv") {
		t.Errorf("Env(fish) gave %v", err)
	}
}

func TestExecRunsTheCommandWithThePinnedToolsFirstOnThePath(t *testing.T) {
	d := pinned(t)
	if err := Exec(ctx, d, open(t, d), []string{"shellcheck", "x.sh"}); err != nil {
		t.Fatalf("Exec gave the error %v", err)
	}
	got := d.Processes.(*fake.Processes).Replaced[0]
	if got.Path != "shellcheck" || !reflect.DeepEqual(got.Args, []string{"x.sh"}) ||
		!reflect.DeepEqual(got.Env, []string{"HOME=/home/a", "PATH=/r/.kit/bin:/usr/bin"}) {
		t.Errorf("Exec ran %+v", got)
	}
	d.Machine.(*fake.Machine).Env = []string{"HOME=/home/a"}
	Exec(ctx, d, open(t, d), []string{"x"})
	if got := d.Processes.(*fake.Processes).Replaced[1].Env; !reflect.DeepEqual(got, []string{"HOME=/home/a", "PATH=/r/.kit/bin"}) {
		t.Errorf("Exec with no PATH set ran with %q", got)
	}
	d.Downloads = failingDownloads{}
	d.Files = fake.NewFiles()
	d.Files.Write("/usr/bin/kit", []byte("kit"), 0o755)
	d.Files.Write("/r/kit.lock", open(t, pinned(t)).Lock.JSON(), 0o644)
	if err := Exec(ctx, d, open(t, d), []string{"x"}); err == nil {
		t.Error("Exec with nothing fetched and the forge away gave no error")
	}
	d.Machine.(*fake.Machine).Err = errors.New("no home")
	if err := Exec(ctx, d, open(t, d), []string{"x"}); err == nil {
		t.Error("Exec with no cache gave no error")
	}
}

func TestToolsListsEachToolWhereItIsPinned(t *testing.T) {
	d := pinned(t)
	d.Files.Write("/r/go.mod", []byte("module x\n\ngo 1.27\n\ntoolchain go1.27.4\n"), 0o644)
	got, err := Tools(d, open(t, d))
	want := []Listed{
		{Name: "go", Version: "1.27.4", Where: "go.mod", State: "Go fetches it, and checks it against its own checksum database"},
		{Name: "shellcheck", Version: "0.11.0", Where: "kit.yml", State: "not fetched here"},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Tools = %+v, %v; want %+v", got, err, want)
	}
	hydrate(t, d)
	if got, _ := Tools(d, open(t, d)); got[1].State != "fetched here, and checked" {
		t.Errorf("Tools gave %+v; want shellcheck fetched", got[1])
	}
	d.Files.Chmod(program(t, d), 0o755)
	d.Files.Write(program(t, d), []byte("changed"), 0o755)
	if got, _ := Tools(d, open(t, d)); got[1].State != "fetched here, and not what kit.lock names" {
		t.Errorf("Tools gave %+v; want shellcheck not what the lock names", got[1])
	}
	d.Files.Write("/r/go.mod", []byte("module x\n\ngo 1.27\n"), 0o644)
	d.Machine.(*fake.Machine).Arch = "riscv64"
	got, _ = Tools(d, open(t, d))
	if len(got) != 1 || got[0].State != "kit.lock holds no file of it for linux/riscv64" {
		t.Errorf("Tools gave %+v", got)
	}
	d.Machine.(*fake.Machine).Err = errors.New("no home")
	if _, err := Tools(d, open(t, d)); err == nil {
		t.Error("Tools with no cache gave no error")
	}
	d.Machine.(*fake.Machine).Err = nil
	d.Files.(*fake.Files).Fail["read /r/go.mod"] = errors.New("broken")
	if _, err := Tools(d, open(t, d)); err == nil {
		t.Error("Tools with go.mod unreadable gave no error")
	}
}

func TestPathGivesTheProgramOnceItIsFetchedAndChecked(t *testing.T) {
	d := pinned(t)
	got, err := Path(ctx, d, open(t, d), "shellcheck")
	if err != nil || got != program(t, d) {
		t.Errorf("Path = %q, %v; want %q", got, err, program(t, d))
	}
	if _, err := Path(ctx, d, open(t, d), "lychee"); err == nil || !strings.Contains(err.Error(), "holds no lychee") {
		t.Errorf("Path of a tool not pinned gave %v", err)
	}
}
