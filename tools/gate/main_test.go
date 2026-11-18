package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// at gives a clock that says so many seconds after the epoch.
func at(seconds int64) func() time.Time {
	return func() time.Time { return time.Unix(seconds, 0) }
}

// profileOf writes a coverage profile to a file of the test's own, and gives its path.
func profileOf(t *testing.T, profile string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cover.out")
	if err := os.WriteFile(path, []byte(profile), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunSaysOk(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--profile=" + profileOf(t, covered), "--since=1000"}, strings.NewReader(ok), &out, at(1012))
	if code != 0 || out.String() != "gate: ok: 2 package(s), 1 test(s), 12s\n" {
		t.Errorf("run = %d, %q; want 0 and the ok line", code, out.String())
	}
}

func TestRunRefusesAndListsEachFinding(t *testing.T) {
	var out bytes.Buffer
	profile := profileOf(t, "mode: set\nx/b/b.go:7.2,7.10 1 0\n")
	code := run([]string{"--profile=" + profile, "--since=1000"}, strings.NewReader(ok), &out, at(1012))
	wanted := "gate: coverage: x/b/b.go:7: 1 statement(s) no test runs\ngate: refused: 1 finding(s)\n"
	if code != 1 || out.String() != wanted {
		t.Errorf("run = %d, %q; want 1, %q", code, out.String(), wanted)
	}
}

func TestRunWithoutItsOptionsSaysItsUsage(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"--profile=cover.out"},
		{"--since=1000"},
		{"--profile=cover.out", "--since=soon"},
		{"--profile=cover.out", "--since=1000", "--fast"},
	} {
		var out bytes.Buffer
		if code := run(args, strings.NewReader(ok), &out, at(1012)); code != 2 ||
			out.String() != "usage: gate --profile=<file> --since=<unix seconds>\n" {
			t.Errorf("run(%q) = %d, %q; want 2 and the usage line", args, code, out.String())
		}
	}
}

func TestRunWithAProfileItCannotOpenRefuses(t *testing.T) {
	var out bytes.Buffer
	missing := filepath.Join(t.TempDir(), "none")
	code := run([]string{"--profile=" + missing, "--since=1000"}, strings.NewReader(ok), &out, at(1012))
	if code != 2 || !strings.HasPrefix(out.String(), "gate: ") || !strings.Contains(out.String(), "none") {
		t.Errorf("run = %d, %q; want 2 and a line that names the file", code, out.String())
	}
}

func TestRunWithEventsItCannotReadRefuses(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--profile=" + profileOf(t, covered), "--since=1000"}, strings.NewReader("FAIL\n"), &out, at(1012))
	if code != 2 || !strings.HasPrefix(out.String(), "gate: ") {
		t.Errorf("run = %d, %q; want 2 and the error", code, out.String())
	}
}

func TestMainExitsWithWhatRunGives(t *testing.T) {
	args, was := os.Args, exit
	defer func() { os.Args, exit = args, was }()
	got := -1
	os.Args, exit = []string{"gate"}, func(code int) { got = code }
	main()
	if got != 2 {
		t.Errorf("main() exited %d with no option; want 2", got)
	}
}
