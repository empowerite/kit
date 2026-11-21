package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// lockOf writes a kit.lock that names a program's digest for this machine's platform, and gives its path.
func lockOf(t *testing.T, tool string, program []byte) string {
	t.Helper()
	sum := sha256.Sum256(program)
	text := `{"tools": {"` + tool + `": {"version": "1", "files": {"` + runtime.GOOS + "/" + runtime.GOARCH +
		`": {"program_digest": "sha256:` + hex.EncodeToString(sum[:]) + `"}}}}}`
	path := filepath.Join(t.TempDir(), "kit.lock")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// programAt writes a program to a file of the test's own, and gives its path.
func programAt(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "shellcheck")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func check(options ...string) (int, string) {
	var out bytes.Buffer
	code := run(options, strings.NewReader(""), &out, at(0))
	return code, out.String()
}

func TestTheJudgeTakesAProgramTheLockHolds(t *testing.T) {
	program := programAt(t, "the program")
	code, out := check("--program="+program, "--tool=shellcheck", "--lock="+lockOf(t, "shellcheck", []byte("the program")))
	if code != 0 || out != "gate: shellcheck: ok: "+program+" is the program kit.lock names\n" {
		t.Errorf("run = %d, %q", code, out)
	}
}

func TestTheJudgeRefusesAProgramWhoseDigestTheLockDoesNotHold(t *testing.T) {
	program := programAt(t, "another program")
	code, out := check("--program="+program, "--tool=shellcheck", "--lock="+lockOf(t, "shellcheck", []byte("the program")))
	if code != 1 || !strings.HasPrefix(out, "gate: shellcheck: refused: "+program+" has the digest sha256:") ||
		!strings.Contains(out, ", and kit.lock names sha256:") {
		t.Errorf("run = %d, %q", code, out)
	}
}

func TestTheJudgeCannotJudgeWhatItCannotRead(t *testing.T) {
	program := programAt(t, "the program")
	lock := lockOf(t, "shellcheck", []byte("the program"))
	for _, c := range []struct {
		options []string
		want    string
	}{
		{[]string{"--program=" + program, "--tool=shellcheck"}, "usage: gate --program=<path> --tool=<name> --lock=<kit.lock>"},
		{[]string{"--program=/none", "--tool=shellcheck", "--lock=" + lock}, "gate: shellcheck: open /none"},
		{[]string{"--program=" + program, "--tool=shellcheck", "--lock=/none"}, "gate: shellcheck: open /none"},
		{[]string{"--program=" + program, "--tool=lychee", "--lock=" + lock}, "gate: lychee: kit.lock holds no file of lychee for " + runtime.GOOS},
		{[]string{"--program=" + program, "--tool=shellcheck", "--lock=" + programAt(t, "not json")}, "gate: shellcheck: kit.lock: "},
	} {
		code, out := check(c.options...)
		if code != 2 || !strings.HasPrefix(out, c.want) {
			t.Errorf("run(%q) = %d, %q; want 2 and %q", c.options, code, out, c.want)
		}
	}
}
