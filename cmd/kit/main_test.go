package main

import (
	"bytes"
	"os"
	"runtime/debug"
	"testing"
)

// built gives a build as Go records one: of a commit, made at a time, from a tree that held changes or not. With no
// commit it gives a build Go recorded nothing of, as go run and go test make.
func built(commit, at, modified string) func() (*debug.BuildInfo, bool) {
	info := &debug.BuildInfo{}
	if commit != "" {
		info.Settings = []debug.BuildSetting{
			{Key: "vcs", Value: "git"},
			{Key: "vcs.revision", Value: commit},
			{Key: "vcs.time", Value: at},
			{Key: "vcs.modified", Value: modified},
		}
	}
	return func() (*debug.BuildInfo, bool) { return info, true }
}

// with runs the test with a build of its own in place of this one's.
func with(t *testing.T, stand func() (*debug.BuildInfo, bool)) {
	t.Helper()
	was := build
	build = stand
	t.Cleanup(func() { build = was })
}

const commit = "d9af39f3aee52ddd79b12884c2d11eb11583fa83"

func TestVersionSaysTheCommitTheBuildIsOf(t *testing.T) {
	// A build does not know its release: a release is a tag put on a commit whose build exists already, and two
	// releases can name one build. So kit says the commit, which is the one thing a build knows of itself.
	for _, c := range []struct {
		name  string
		build func() (*debug.BuildInfo, bool)
		out   string
	}{
		{"a build of a commit", built(commit, "2026-11-19T04:42:54Z", "false"),
			"kit, built from commit " + commit + " of 2026-11-19T04:42:54Z\n"},
		{"a build of a commit and of changes it does not hold", built(commit, "2026-11-19T04:42:54Z", "true"),
			"kit, built from commit " + commit + " of 2026-11-19T04:42:54Z, and from changes it does not hold\n"},
		{"a build Go recorded no commit of", built("", "", ""),
			"kit, a build that names no commit\n"},
		{"a build Go recorded nothing of", func() (*debug.BuildInfo, bool) { return nil, false },
			"kit, a build that names no commit\n"},
	} {
		with(t, c.build)
		var out, err bytes.Buffer
		if code := run([]string{"version"}, &out, &err); code != 0 || out.String() != c.out || err.String() != "" {
			t.Errorf("%s: run = %d, %q, %q; want 0, %q, and nothing on stderr", c.name, code, out.String(), err.String(), c.out)
		}
	}
}

func TestVersionOfThisBuildNamesNoCommit(t *testing.T) {
	// go test records no commit in the builds it makes, so the build Go itself reports here is one that names none.
	var out, err bytes.Buffer
	if code := run([]string{"version"}, &out, &err); code != 0 || out.String() != "kit, a build that names no commit\n" {
		t.Errorf("run = %d, %q; want 0 and the line of a build that names no commit", code, out.String())
	}
}

func TestRunRefusesWordsItCannotTake(t *testing.T) {
	for _, c := range []struct {
		args []string
		err  string
	}{
		{nil, "usage: kit version\n"},
		{[]string{"version", "extra"}, "usage: kit version\n"},
		{[]string{"flow"}, "kit: no such command: flow\n"},
	} {
		var out, err bytes.Buffer
		if code := run(c.args, &out, &err); code != 2 || out.String() != "" || err.String() != c.err {
			t.Errorf("run(%q) = %d, %q, %q; want 2, nothing on stdout, %q", c.args, code, out.String(), err.String(), c.err)
		}
	}
}

func TestMainExitsWithWhatRunGives(t *testing.T) {
	args, was := os.Args, exit
	defer func() { os.Args, exit = args, was }()
	got := -1
	os.Args, exit = []string{"kit"}, func(code int) { got = code }
	main()
	if got != 2 {
		t.Errorf("main() exited %d with no command; want 2", got)
	}
}
