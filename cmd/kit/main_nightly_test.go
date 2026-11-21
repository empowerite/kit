//go:build nightly

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/empowerite/kit/internal/adapters/fake"
	"github.com/empowerite/kit/internal/plan"
	"github.com/empowerite/kit/internal/toolchain"
)

// TestShellcheckPinnedFetchedAndRunForReal pins shellcheck against the real GitHub, approved here as a test may,
// then has a real build of kit fetch it and run it through its link in .kit/bin.
func TestShellcheckPinnedFetchedAndRunForReal(t *testing.T) {
	ctx := context.Background()
	repo, _ := filepath.EvalSymlinks(t.TempDir())
	kit := filepath.Join(t.TempDir(), "kit")
	for _, c := range [][]string{{"git", "init", "-q", repo}, {"go", "build", "-o", kit, "."}} {
		if out, err := exec.Command(c[0], c[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %v: %s", c, err, out)
		}
	}
	d, err := desk()
	if err != nil {
		t.Fatal(err)
	}
	d.Dir = repo
	r, err := toolchain.Open(ctx, d)
	if err != nil {
		t.Fatal(err)
	}
	list, _ := toolchain.List()
	p, _, err := toolchain.Add(ctx, d, r, list, "shellcheck@0.11.0", false)
	if err != nil {
		t.Fatal(err)
	}
	approval, err := plan.Approve(ctx, p, &fake.Presence{})
	if err != nil {
		t.Fatal(err)
	}
	if result := plan.Apply(ctx, p, approval); result.Err != nil {
		t.Fatal(result.Err)
	}
	// The cache is the test's own: on macOS kit's is under HOME, and on Linux under XDG_CACHE_HOME.
	cache := t.TempDir()
	env := append(os.Environ(), "XDG_CACHE_HOME="+cache, "HOME="+cache)
	fetch := exec.Command(kit, "tools", "fetch")
	fetch.Dir, fetch.Env = repo, env
	if out, err := fetch.CombinedOutput(); err != nil {
		t.Fatalf("kit tools fetch: %v: %s", err, out)
	}
	run := exec.Command(filepath.Join(repo, ".kit/bin/shellcheck"), "--version")
	run.Dir, run.Env = repo, env
	out, err := run.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "version: 0.11.0") {
		t.Errorf(".kit/bin/shellcheck --version = %v, %s; want version 0.11.0", err, out)
	}
	status := exec.Command("git", "-C", repo, "status", "--porcelain", "--untracked-files=all")
	if out, _ := status.Output(); strings.Contains(string(out), ".kit/") {
		t.Errorf("git sees what kit placed: %s", out)
	}
}
