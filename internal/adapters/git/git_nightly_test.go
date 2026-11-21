//go:build nightly

package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/empowerite/kit/internal/ports"
)

// machine runs real programs, as the os adapter does; a test of this package may not import that one.
type machine struct{}

func (machine) Run(ctx context.Context, c ports.Cmd) (int, error) {
	cmd := exec.CommandContext(ctx, c.Path, c.Args...)
	cmd.Stdout, cmd.Stderr = c.Stdout, c.Stderr
	var ended *exec.ExitError
	if err := cmd.Run(); errors.As(err, &ended) {
		return ended.ExitCode(), nil
	} else if err != nil {
		return -1, err
	}
	return 0, nil
}

func (machine) Replace(ports.Cmd) error { return errors.New("not here") }

func TestGitFindsTheRootAndWhatItTracksForReal(t *testing.T) {
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	for _, args := range [][]string{{"init", "-q", dir}, {"-C", dir, "worktree", "add", "-q", "--orphan", "-b", "w", dir + "/wt"}} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	if err := os.WriteFile(dir+"/tracked", nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", dir, "add", "tracked").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}
	g := New(machine{})
	root, common, err := g.Root(ctx, dir+"/wt")
	if root != dir+"/wt" || common != dir+"/.git" || err != nil {
		t.Errorf("Root = %s, %s, %v; want the worktree and the clone's .git", root, common, err)
	}
	if yes, err := g.Tracked(ctx, dir, "tracked"); !yes || err != nil {
		t.Errorf("Tracked(tracked) = %v, %v; want true", yes, err)
	}
	if yes, err := g.Tracked(ctx, dir, "other"); yes || err != nil {
		t.Errorf("Tracked(other) = %v, %v; want false", yes, err)
	}
	if _, _, err := g.Root(ctx, "/"); err == nil {
		t.Error("Root of / gave no error")
	}
}
