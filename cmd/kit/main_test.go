package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/empowerite/kit/internal/cli"
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

func TestVersionSaysTheSameAsJSON(t *testing.T) {
	with(t, built(commit, "2026-11-19T04:42:54Z", "true"))
	var out, err bytes.Buffer
	code := run([]string{"version", "--json"}, &out, &err)
	var got map[string]any
	if jerr := json.Unmarshal(out.Bytes(), &got); code != 0 || jerr != nil {
		t.Fatalf("run = %d, %q; want 0 and JSON", code, out.String())
	}
	want := map[string]any{"schema": "kit/version", "version": float64(1), "commit": commit,
		"time": "2026-11-19T04:42:54Z", "modified": true}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("version --json said %s = %v; want %v", key, got[key], value)
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
		{nil, "usage: kit <command> [--json] [--help]\n"},
		{[]string{"version", "extra"}, "kit version: one argument too many: extra\n"},
		{[]string{"flow"}, "kit: no such command: flow\n"},
	} {
		var out, err bytes.Buffer
		if code := run(c.args, &out, &err); code != 2 || out.String() != "" || !strings.HasPrefix(err.String(), c.err) {
			t.Errorf("run(%q) = %d, %q, %q; want 2, nothing on stdout, and %q first", c.args, code, out.String(), err.String(), c.err)
		}
	}
}

func TestDataShowPrintsKitsOwnFile(t *testing.T) {
	var out, err bytes.Buffer
	if code := run([]string{"data", "show", "commands"}, &out, &err); code != 0 || !strings.Contains(out.String(), "words: [version]") {
		t.Errorf("run(data show commands) = %d, %q, %q; want 0 and commands.yml", code, out.String(), err.String())
	}
	out.Reset()
	code := run([]string{"data", "show", "commands", "--json"}, &out, &err)
	var got map[string]any
	if jerr := json.Unmarshal(out.Bytes(), &got); code != 0 || jerr != nil || got["name"] != "commands" ||
		!strings.Contains(got["text"].(string), "words: [version]") {
		t.Errorf("run(data show commands --json) = %d, %q; want 0 and the file as JSON", code, out.String())
	}
	out.Reset()
	err.Reset()
	if code := run([]string{"data", "show", "nothing"}, &out, &err); code != 1 || !strings.Contains(err.String(), "no file named nothing") {
		t.Errorf("run(data show nothing) = %d, %q; want 1 and that there is no such file", code, err.String())
	}
}

func TestEveryCommandHasAHandler(t *testing.T) {
	commands, err := cli.Commands()
	if err != nil {
		t.Fatalf("cli.Commands() gave the error %v", err)
	}
	for _, c := range commands {
		if _, ok := handlers[strings.Join(c.Words, " ")]; !ok {
			t.Errorf("kit %s has no handler", strings.Join(c.Words, " "))
		}
	}
}

func TestRunSaysWhereKitsCommandsAreWrong(t *testing.T) {
	was := commands
	defer func() { commands = was }()
	commands = func() ([]cli.Command, error) { return nil, os.ErrNotExist }
	var out, err bytes.Buffer
	if code := run([]string{"version"}, &out, &err); code != 2 || !strings.Contains(err.String(), "kit: its commands:") {
		t.Errorf("run = %d, %q; want 2 and that kit's commands are wrong", code, err.String())
	}
}

func TestKitStartedByAToolsNameRunsTheTool(t *testing.T) {
	// Outside a repository that pins it, the tool cannot run, and says so: what runs it for real is a nightly test.
	args, was := os.Args, exit
	defer func() { os.Args, exit = args, was }()
	got := -1
	os.Args, exit = []string{"/nowhere/.kit/bin/shellcheck", "--version"}, func(code int) { got = code }
	main()
	if got != 127 {
		t.Errorf("main() as shellcheck exited %d; want 127", got)
	}
	var stderr bytes.Buffer
	if code := tool("/nowhere/.kit/bin/shellcheck", nil, &stderr); code != 127 || !strings.HasPrefix(stderr.String(), "kit: shellcheck: kit.lock at /nowhere holds no shellcheck") {
		t.Errorf("tool = %d, %q", code, stderr.String())
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

func TestDeskIsTheRealMachine(t *testing.T) {
	d, err := desk()
	if err != nil || d.Dir == "" || d.Files == nil || d.Git == nil || d.Releases == nil || d.Downloads == nil {
		t.Errorf("desk() = %+v, %v; want every system wired", d, err)
	}
	t.Setenv("GH_TOKEN", "a")
	if d, _ := desk(); d.Releases == nil {
		t.Error("desk() with a token gave no releases")
	}
	if _, err := d.Presence.Prove(context.Background(), "approve", "abc"); err == nil || !strings.Contains(err.Error(), "#48") {
		t.Errorf("Prove gave %v; want a refusal that names #48", err)
	}
}
