package os

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/empowerite/kit/internal/ports"
)

var (
	_ ports.Files     = Files{}
	_ ports.Clock     = Clock{}
	_ ports.Processes = Processes{}
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestFilesDoWhatTheFakeDoes(t *testing.T) {
	// The same acts as the fake's own tests, on the machine's files, in a folder of the test's own.
	dir, f := t.TempDir(), Files{}
	must(t, f.Write(dir+"/a/b/c", []byte("x"), 0o444))
	if got, err := f.Read(dir + "/a/b/c"); err != nil || string(got) != "x" {
		t.Errorf("Read = %q, %v; want x", got, err)
	}
	if info, err := f.Stat(dir + "/a/b/c"); err != nil || info.Mode().Perm() != 0o444 {
		t.Errorf("Stat = %v, %v; want read-only", info, err)
	}
	if err := f.Write(dir+"/a/b/c", []byte("y"), 0o644); !errors.Is(err, fs.ErrPermission) {
		t.Errorf("Write over a read-only file gave %v; want fs.ErrPermission", err)
	}
	must(t, f.Link("b/c", dir+"/a/link"))
	if got, err := f.ReadLink(dir + "/a/link"); err != nil || got != "b/c" {
		t.Errorf("ReadLink = %q, %v; want b/c", got, err)
	}
	if info, _ := f.Stat(dir + "/a/link"); info.Mode()&fs.ModeSymlink == 0 {
		t.Errorf("Stat of a link = %v; want a link", info.Mode())
	}
	if got, _ := f.Read(dir + "/a/link"); string(got) != "x" {
		t.Errorf("Read through a link = %q; want x", got)
	}
	must(t, f.MakeDir(dir+"/d/e"))
	must(t, f.Rename(dir+"/d", dir+"/moved"))
	must(t, f.Chmod(dir+"/a/b/c", 0o644))
	must(t, f.Remove(dir+"/a/b/c"))
	if _, err := f.Read(dir + "/a/b/c"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Read of what was removed gave %v; want fs.ErrNotExist", err)
	}
	if err := f.Write(dir+"/a/link/under", nil, 0o644); err == nil {
		t.Error("Write below a link to nothing gave no error")
	}
	if err := f.Link("x", dir+"/a/link/under"); err == nil {
		t.Error("Link below a link to nothing gave no error")
	}
}

func TestClockSaysTheTime(t *testing.T) {
	if got := (Clock{}).Now(); time.Since(got) > time.Minute {
		t.Errorf("Clock.Now = %v; want now", got)
	}
}

func TestRunSaysAProgramThatCannotBeFoundAndStartsNothing(t *testing.T) {
	// No program has this name, so the run ends before any process starts. The nightly tier runs real programs.
	var out bytes.Buffer
	code, err := Processes{}.Run(context.Background(), ports.Cmd{Path: "kit-no-such-program", Stdout: &out, Env: []string{"A=1"}})
	if err == nil || !strings.Contains(err.Error(), "not found") || code != -1 {
		t.Errorf("Run = %d, %v; want -1 and that it is not found", code, err)
	}
}

// exited is an error as a program that ran and ended with a status gives it.
type exited int

func (e exited) Error() string { return "exit status" }
func (e exited) ExitCode() int { return int(e) }

func TestExitGivesTheStatusOfAProgramThatRan(t *testing.T) {
	if code, err := exit(nil); code != 0 || err != nil {
		t.Errorf("exit(nil) = %d, %v; want 0", code, err)
	}
	if code, err := exit(fmt.Errorf("it ran: %w", exited(3))); code != 3 || err != nil {
		t.Errorf("exit(a program that ran) = %d, %v; want its status and no error", code, err)
	}
}

func TestReplaceSaysAProgramItCannotFindAndReplacesNothing(t *testing.T) {
	// Neither program is there, so kit is not replaced and the test goes on; the nightly tier replaces kit for real.
	dir := t.TempDir()
	// An empty file that may be run is found on the PATH, and is no program the system can start.
	must(t, Files{}.Write(dir+"/empty", nil, 0o755))
	for _, c := range []ports.Cmd{{Path: "kit-no-such-program", Env: []string{"PATH=" + dir}}, {Path: dir + "/none"},
		{Path: "empty", Env: []string{"PATH=" + dir}}} {
		if err := (Processes{}).Replace(c); err == nil {
			t.Errorf("Replace(%s) gave no error", c.Path)
		}
	}
}

func TestLookPathFindsAProgramOnTheGivenPath(t *testing.T) {
	dir := t.TempDir()
	must(t, Files{}.Write(dir+"/b/tool", []byte("#!/bin/sh\n"), 0o755))
	must(t, Files{}.Write(dir+"/a/tool", []byte("not a program"), 0o644))
	got, err := lookPath("tool", []string{"HOME=/h", "PATH=" + dir + "/none:" + dir + "/a:" + dir + "/b"})
	if err != nil || got != dir+"/b/tool" {
		t.Errorf("lookPath = %q, %v; want the program in b", got, err)
	}
	if _, err := lookPath("tool", nil); err == nil || !strings.Contains(err.Error(), "no tool on the PATH") {
		t.Errorf("lookPath with no PATH gave %v", err)
	}
}

func TestMachineSaysWhereKitRuns(t *testing.T) {
	m := Machine{}
	var _ ports.Machine = m
	if p := m.Platform(); !strings.Contains(p, "/") {
		t.Errorf("Platform = %q", p)
	}
	t.Setenv("XDG_STATE_HOME", "/state")
	if s, err := m.State(); s != "/state/kit" || err != nil {
		t.Errorf("State = %q, %v; want /state/kit", s, err)
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "/home/a")
	if s, _ := m.State(); s != "/home/a/.local/state/kit" {
		t.Errorf("State = %q; want under HOME", s)
	}
	t.Setenv("XDG_CACHE_HOME", "/cache")
	if c, err := m.Cache(); !strings.HasSuffix(c, "/kit") || err != nil {
		t.Errorf("Cache = %q, %v", c, err)
	}
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	if _, err := m.Cache(); err == nil {
		t.Error("Cache with no HOME gave no error")
	}
	if _, err := m.State(); err == nil {
		t.Error("State with no HOME gave no error")
	}
	if kit, err := m.Executable(); kit == "" || err != nil {
		t.Errorf("Executable = %q, %v", kit, err)
	}
	if len(m.Environ()) == 0 || len(m.Args()) == 0 {
		t.Error("Environ or Args gave nothing")
	}
}
