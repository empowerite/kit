package git

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/empowerite/kit/internal/adapters/fake"
	"github.com/empowerite/kit/internal/ports"
)

var ctx = context.Background()

// answering gives processes that answer each run of git with what said gives for its arguments.
func answering(said func(args []string) (string, int, error)) *fake.Processes {
	return &fake.Processes{Answer: func(c ports.Cmd) (int, error) {
		out, code, err := said(c.Args)
		io.WriteString(c.Stdout, out)
		io.WriteString(c.Stderr, "fatal: what git said\n")
		return code, err
	}}
}

func TestRootGivesTheWorkingTreeAndTheCommonDirectory(t *testing.T) {
	p := answering(func([]string) (string, int, error) { return "/r/worktrees/7 x\n/r/.git\n", 0, nil })
	root, common, err := New(p).Root(ctx, "/r/worktrees/7-x/internal")
	if root != "/r/worktrees/7 x" || common != "/r/.git" || err != nil {
		t.Errorf("Root = %s, %s, %v; want the worktree and the clone's .git", root, common, err)
	}
	want := []string{"-C", "/r/worktrees/7-x/internal", "rev-parse", "--path-format=absolute", "--show-toplevel", "--git-common-dir"}
	if p.Runs[0].Path != "git" || !reflect.DeepEqual(p.Runs[0].Args, want) {
		t.Errorf("Root ran %s %q; want git %q", p.Runs[0].Path, p.Runs[0].Args, want)
	}
}

func TestRootSaysWhatGitSaidWhereItIsNoClone(t *testing.T) {
	for _, p := range []*fake.Processes{
		answering(func([]string) (string, int, error) { return "", 128, nil }),
		answering(func([]string) (string, int, error) { return "", -1, errors.New("git: not found") }),
		answering(func([]string) (string, int, error) { return "one line\n", 0, nil }),
	} {
		if _, _, err := New(p).Root(ctx, "/tmp"); err == nil || !strings.HasPrefix(err.Error(), "git: ") {
			t.Errorf("Root gave the error %v; want one that begins git:", err)
		}
	}
}

func TestTrackedSaysWhetherGitTracksAPath(t *testing.T) {
	p := answering(func(args []string) (string, int, error) {
		if args[len(args)-1] == ".kit/env" {
			return ".kit/env\n", 0, nil
		}
		return "", 1, nil
	})
	g := New(p)
	if yes, err := g.Tracked(ctx, "/r", ".kit/env"); !yes || err != nil {
		t.Errorf("Tracked(.kit/env) = %v, %v; want true", yes, err)
	}
	if yes, err := g.Tracked(ctx, "/r", ".kit/bin/x"); yes || err != nil {
		t.Errorf("Tracked(.kit/bin/x) = %v, %v; want false", yes, err)
	}
	want := []string{"-C", "/r", "ls-files", "--error-unmatch", "--", ".kit/bin/x"}
	if !reflect.DeepEqual(p.Runs[1].Args, want) {
		t.Errorf("Tracked ran git %q; want git %q", p.Runs[1].Args, want)
	}
	bad := New(answering(func([]string) (string, int, error) { return "", 128, nil }))
	if _, err := bad.Tracked(ctx, "/tmp", "x"); err == nil || !strings.Contains(err.Error(), "what git said") {
		t.Errorf("Tracked gave the error %v; want what git said", err)
	}
}
