package fake

import (
	"context"
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/empowerite/kit/internal/ports"
)

// The fake files stand for the machine's, so each test here says what the real files do, as the os adapter's tests
// say it of the real ones.
var _ ports.Files = NewFiles()

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestWriteMakesTheDirectoriesAboveAndReadGivesTheBytes(t *testing.T) {
	f := NewFiles()
	must(t, f.Write("/a/b/c", []byte("x"), 0o644))
	got, err := f.Read("/a/b/c")
	if err != nil || string(got) != "x" {
		t.Errorf("Read = %q, %v; want x", got, err)
	}
	if want := []string{"/a", "/a/b", "/a/b/c"}; !reflect.DeepEqual(f.Paths(), want) {
		t.Errorf("Paths = %q; want %q", f.Paths(), want)
	}
	if info, _ := f.Stat("/a/b"); !info.IsDir() || info.Name() != "b" || info.ModTime() != (time.Time{}) || info.Sys() != nil {
		t.Errorf("Stat(/a/b) = %+v; want a directory named b", info)
	}
}

func TestWriteReplacesAWritableFileAndRefusesAReadOnlyOne(t *testing.T) {
	f := NewFiles()
	must(t, f.Write("/a", []byte("x"), 0o644))
	must(t, f.Write("/a", []byte("y"), 0o600))
	if got, _ := f.Read("/a"); string(got) != "y" {
		t.Errorf("Read = %q; want y", got)
	}
	must(t, f.Chmod("/a", 0o444))
	if err := f.Write("/a", []byte("z"), 0o644); !errors.Is(err, fs.ErrPermission) {
		t.Errorf("Write over a read-only file gave %v; want fs.ErrPermission", err)
	}
	if err := f.Write("/", []byte("z"), 0o644); !errors.Is(err, fs.ErrPermission) {
		t.Errorf("Write over a directory gave %v; want fs.ErrPermission", err)
	}
	if info, _ := f.Stat("/a"); info.Mode() != 0o444 || info.Size() != 1 {
		t.Errorf("Stat(/a) = %v, %d; want -r--r--r-- and one byte", info.Mode(), info.Size())
	}
}

func TestAFileStandsInTheWayOfADirectory(t *testing.T) {
	f := NewFiles()
	must(t, f.Write("/a", []byte("x"), 0o644))
	for _, err := range []error{f.Write("/a/b", nil, 0o644), f.MakeDir("/a/b"), f.Link("t", "/a/b"), f.Rename("/a", "/a/b/c")} {
		if !errors.Is(err, fs.ErrExist) {
			t.Errorf("an act under the file /a gave %v; want fs.ErrExist", err)
		}
	}
}

func TestReadFollowsLinksAndRefusesWhatIsNotAFile(t *testing.T) {
	f := NewFiles()
	must(t, f.Write("/d/file", []byte("x"), 0o644))
	must(t, f.Link("file", "/d/relative"))
	must(t, f.Link("/d/relative", "/abs"))
	if got, err := f.Read("/abs"); err != nil || string(got) != "x" {
		t.Errorf("Read through two links = %q, %v; want x", got, err)
	}
	if got, err := f.ReadLink("/abs"); err != nil || got != "/d/relative" {
		t.Errorf("ReadLink = %q, %v; want /d/relative", got, err)
	}
	if _, err := f.ReadLink("/d/file"); !errors.Is(err, fs.ErrInvalid) {
		t.Errorf("ReadLink of a file gave %v; want fs.ErrInvalid", err)
	}
	if info, _ := f.Stat("/abs"); info.Mode()&fs.ModeSymlink == 0 {
		t.Errorf("Stat of a link = %v; want a link", info.Mode())
	}
	if err := f.Link("x", "/abs"); !errors.Is(err, fs.ErrExist) {
		t.Errorf("Link over a link gave %v; want fs.ErrExist", err)
	}
	for _, name := range []string{"/d", "/none"} {
		if _, err := f.Read(name); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Read(%s) gave %v; want fs.ErrNotExist", name, err)
		}
	}
}

func TestRemoveTakesAFileOrAnEmptyDirectory(t *testing.T) {
	f := NewFiles()
	must(t, f.Write("/d/file", []byte("x"), 0o644))
	if err := f.Remove("/d"); !errors.Is(err, fs.ErrExist) {
		t.Errorf("Remove of a directory that holds a file gave %v; want fs.ErrExist", err)
	}
	must(t, f.Remove("/d/file"))
	must(t, f.Remove("/d"))
	if err := f.Remove("/d"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Remove of nothing gave %v; want fs.ErrNotExist", err)
	}
	if _, err := f.Stat("/d"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Stat of nothing gave %v; want fs.ErrNotExist", err)
	}
}

func TestRenameMovesADirectoryAndAllItHolds(t *testing.T) {
	f := NewFiles()
	must(t, f.Write("/part/x/file", []byte("x"), 0o644))
	must(t, f.Rename("/part", "/whole"))
	if want := []string{"/whole", "/whole/x", "/whole/x/file"}; !reflect.DeepEqual(f.Paths(), want) {
		t.Errorf("Paths = %q; want %q", f.Paths(), want)
	}
	if err := f.Rename("/part", "/other"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Rename of nothing gave %v; want fs.ErrNotExist", err)
	}
	if err := f.Chmod("/none", 0o644); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Chmod of nothing gave %v; want fs.ErrNotExist", err)
	}
}

func TestFailMakesAnActFail(t *testing.T) {
	f := NewFiles()
	must(t, f.Write("/a", []byte("x"), 0o644))
	broken := errors.New("broken")
	for _, act := range []string{"read /a", "write /a", "link /b", "remove /a", "stat /a", "mkdir /d", "rename /a", "chmod /a"} {
		f.Fail[act] = broken
	}
	_, read := f.Read("/a")
	_, stat := f.Stat("/a")
	for _, err := range []error{read, f.Write("/a", nil, 0), f.Link("t", "/b"), f.Remove("/a"), stat, f.MakeDir("/d"),
		f.Rename("/a", "/c"), f.Chmod("/a", 0)} {
		if !errors.Is(err, broken) {
			t.Errorf("a failing act gave %v; want broken", err)
		}
	}
}

func TestTheOthersSayWhatTheyAreSetTo(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC)
	if got := (&Clock{at}).Now(); !got.Equal(at) {
		t.Errorf("Clock.Now = %v; want %v", got, at)
	}
	g := &Git{RootDir: "/r", Common: "/r/.git", Tracks: map[string]bool{"a": true}}
	if root, common, err := g.Root(ctx, "/r/x"); root != "/r" || common != "/r/.git" || err != nil {
		t.Errorf("Git.Root = %s, %s, %v", root, common, err)
	}
	if yes, _ := g.Tracked(ctx, "/r", "a"); !yes {
		t.Error("Git.Tracked(a) = false; want true")
	}
	p := &Presence{Proof: ports.Proof{Kind: "touch-id"}}
	if got, _ := p.Prove(ctx, "approve", "abc"); got.Kind != "touch-id" || !reflect.DeepEqual(p.Asked, []string{"approve"}) {
		t.Errorf("Presence.Prove = %+v, asked %q", got, p.Asked)
	}
	r := &Processes{Answer: func(c ports.Cmd) (int, error) { return len(c.Args), nil }}
	if code, _ := r.Run(ctx, ports.Cmd{Path: "x", Args: []string{"a", "b"}}); code != 2 || len(r.Runs) != 1 {
		t.Errorf("Processes.Run = %d, with %d runs kept; want 2 and 1", code, len(r.Runs))
	}
}

func TestReleasesAndDownloadsGiveWhatTheyHold(t *testing.T) {
	ctx := context.Background()
	r := &Releases{Tags: map[string]string{"o/r": "v1"}}
	r.Put("o/r", "v1", "a", "https://x/a", "")
	if a, err := r.Asset(ctx, "o/r", "v1", "a"); a.URL != "https://x/a" || err != nil {
		t.Errorf("Asset = %+v, %v", a, err)
	}
	if _, err := r.Asset(ctx, "o/r", "v1", "b"); err == nil {
		t.Error("Asset of a file the release does not hold gave no error")
	}
	if tag, err := r.Newest(ctx, "o/r"); tag != "v1" || err != nil {
		t.Errorf("Newest = %q, %v", tag, err)
	}
	if _, err := r.Newest(ctx, "o/none"); err == nil {
		t.Error("Newest of a repository with no release gave no error")
	}
	d := Downloads{"https://x/a": []byte("a")}
	if got, err := d.Get(ctx, "https://x/a"); string(got) != "a" || err != nil {
		t.Errorf("Get = %q, %v", got, err)
	}
	if _, err := d.Get(ctx, "https://x/b"); err == nil {
		t.Error("Get of nothing gave no error")
	}
}

func TestTheMachineAndTheReplacingSayWhatTheyAreSetTo(t *testing.T) {
	m := &Machine{OS: "linux", Arch: "arm64", CacheDir: "/c", StateDir: "/s", Kit: "/k", Env: []string{"A=1"}, Argv: []string{"kit"}}
	var _ ports.Machine = m
	cache, _ := m.Cache()
	state, _ := m.State()
	kit, _ := m.Executable()
	if m.Platform() != "linux/arm64" || cache != "/c" || state != "/s" || kit != "/k" || m.Environ()[0] != "A=1" || m.Args()[0] != "kit" {
		t.Errorf("the machine said %s %s %s %s %v %v", m.Platform(), cache, state, kit, m.Environ(), m.Args())
	}
	p := &Processes{ReplaceErr: errors.New("x")}
	if err := p.Replace(ports.Cmd{Path: "a"}); err == nil || len(p.Replaced) != 1 {
		t.Errorf("Replace = %v, with %d kept", err, len(p.Replaced))
	}
}

func TestProcessesPrintWhatTheyAreSetToWithNoAnswer(t *testing.T) {
	var out strings.Builder
	p := &Processes{Prints: map[string]string{"status --json": "{}"}}
	if code, err := p.Run(context.Background(), ports.Cmd{Path: "direnv", Args: []string{"status", "--json"}, Stdout: &out}); code != 0 || err != nil || out.String() != "{}" {
		t.Errorf("Run = %d, %v, %q", code, err, out.String())
	}
	if code, _ := p.Run(context.Background(), ports.Cmd{Path: "direnv", Args: []string{"allow"}}); code != 0 {
		t.Errorf("Run with nowhere to print = %d", code)
	}
}
