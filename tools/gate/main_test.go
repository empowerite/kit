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

// gate runs the judge over a run's events, with its three options, twelve seconds after the gate started.
func gate(events string, options ...string) (int, string) {
	var out bytes.Buffer
	code := run(options, strings.NewReader(events), &out, at(1012))
	return code, out.String()
}

func TestRunSaysOk(t *testing.T) {
	code, out := gate(ok, "--profile="+profileOf(t, covered), "--since=1000", "--status=0")
	if wanted := "gate: ok: 1 package(s) tested, 1 with nothing to test, 1 test(s), 12s\n"; code != 0 || out != wanted {
		t.Errorf("run = %d, %q; want 0, %q", code, out, wanted)
	}
}

func TestRunRefusesAndListsEachFinding(t *testing.T) {
	code, out := gate(ok, "--profile="+profileOf(t, "mode: set\nx/b/b.go:7.2,7.10 1 0\n"), "--since=1000", "--status=0")
	wanted := "gate: coverage: x/b/b.go:7: 1 statement(s) no test runs\ngate: refused: 1 finding(s)\n"
	if code != 1 || out != wanted {
		t.Errorf("run = %d, %q; want 1, %q", code, out, wanted)
	}
}

func TestRunSetsInTheLaterLinesOfAFinding(t *testing.T) {
	events := `{"ImportPath":"x/d [x/d.test]","Action":"build-output","Output":"d/d.go:3:17: undefined: missing\n"}
{"ImportPath":"x/d [x/d.test]","Action":"build-output","Output":"d/d.go:4:2: undefined: gone\n"}
{"ImportPath":"x/d [x/d.test]","Action":"build-fail"}
{"Action":"start","Package":"x/d"}
{"Action":"fail","Package":"x/d","Elapsed":0}
`
	code, out := gate(events, "--profile="+profileOf(t, "mode: set\n"), "--since=1000", "--status=1")
	wanted := "gate: build: x/d: d/d.go:3:17: undefined: missing\n  d/d.go:4:2: undefined: gone\ngate: refused: 1 finding(s)\n"
	if code != 1 || out != wanted {
		t.Errorf("run = %d, %q; want 1, %q", code, out, wanted)
	}
}

func TestRunRefusesAStatusThatNothingExplains(t *testing.T) {
	code, out := gate(ok, "--profile="+profileOf(t, covered), "--since=1000", "--status=137")
	wanted := "gate: go test: go test: it ended with status 137, and nothing above says why\ngate: refused: 1 finding(s)\n"
	if code != 1 || out != wanted {
		t.Errorf("run = %d, %q; want 1, %q", code, out, wanted)
	}
}

func TestRunWithNoProfileRefuses(t *testing.T) {
	code, out := gate(ok, "--profile="+filepath.Join(t.TempDir(), "none"), "--since=1000", "--status=0")
	wanted := "gate: coverage: the profile: go test wrote none that can be read\ngate: refused: 1 finding(s)\n"
	if code != 1 || out != wanted {
		t.Errorf("run = %d, %q; want 1, %q", code, out, wanted)
	}
}

func TestRunWithoutItsOptionsSaysItsUsage(t *testing.T) {
	for _, options := range [][]string{
		nil,
		{"--since=1000", "--status=0"},
		{"--profile=cover.out", "--status=0"},
		{"--profile=cover.out", "--since=1000"},
		{"--profile=cover.out", "--since=soon", "--status=0"},
		{"--profile=cover.out", "--since=1000", "--status=bad"},
		{"--profile=cover.out", "--since=1000", "--status=0", "--fast"},
	} {
		wanted := "usage: gate --profile=<file> --since=<unix seconds> --status=<go test's exit status>\n"
		if code, out := gate(ok, options...); code != 2 || out != wanted {
			t.Errorf("run(%q) = %d, %q; want 2 and the usage line", options, code, out)
		}
	}
}

func TestRunWithEventsItCannotReadSaysWhy(t *testing.T) {
	code, out := gate("FAIL\n", "--profile="+profileOf(t, covered), "--since=1000", "--status=0")
	if code != 2 || !strings.HasPrefix(out, "gate: go test's line 1 is no event") {
		t.Errorf("run = %d, %q; want 2 and the error", code, out)
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
