package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fileOf writes a text to a file of the test's own, and gives its path.
func fileOf(t *testing.T, name, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// forge runs the judge with its options, and gives its status and what it printed.
func forge(options ...string) (int, string) {
	var out bytes.Buffer
	code := run(options, &out)
	return code, out.String()
}

func TestRunSaysOk(t *testing.T) {
	rules, expect := fileOf(t, "rules.json", held), fileOf(t, "expect.json", leaned)
	code, out := forge("--branch=main", "--rules="+rules, "--expect="+expect)
	if wanted := "forge: ok: main holds the 3 rule(s) the gate leans on\n"; code != 0 || out != wanted {
		t.Errorf("run = %d, %q; want 0, %q", code, out, wanted)
	}
}

func TestRunRefusesAndSaysWhatDoesNotHold(t *testing.T) {
	code, out := forge("--branch=main", "--rules="+fileOf(t, "rules.json", `[]`),
		"--expect="+fileOf(t, "expect.json", `{"rules": [{"type": "deletion", "that": "main is never deleted"}]}`))
	wanted := "forge: main: nothing holds that main is never deleted: no rule of the kind deletion applies\n" +
		"forge: refused: GitHub does not hold 1 of the 1 rule(s) the gate leans on\n"
	if code != 1 || out != wanted {
		t.Errorf("run = %d, %q; want 1, %q", code, out, wanted)
	}
}

func TestRunWithoutItsOptionsSaysItsUsage(t *testing.T) {
	for _, options := range [][]string{
		nil,
		{"--rules=r", "--expect=e"},
		{"--branch=main", "--expect=e"},
		{"--branch=main", "--rules=r"},
		{"--branch=main", "--rules=r", "--expect=e", "--more"},
	} {
		wanted := "usage: forge --branch=<branch> --rules=<file> --expect=<file>\n"
		if code, out := forge(options...); code != 2 || out != wanted {
			t.Errorf("run(%q) = %d, %q; want 2 and the usage line", options, code, out)
		}
	}
}

func TestRunWithAFileItCannotReadSaysWhy(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "none")
	rules, expect := fileOf(t, "rules.json", held), fileOf(t, "expect.json", leaned)
	for _, options := range [][]string{
		{"--branch=main", "--rules=" + missing, "--expect=" + expect},
		{"--branch=main", "--rules=" + rules, "--expect=" + missing},
		{"--branch=main", "--rules=" + fileOf(t, "bad.json", "Not Found"), "--expect=" + expect},
	} {
		if code, out := forge(options...); code != 2 || !strings.HasPrefix(out, "forge: ") {
			t.Errorf("run(%q) = %d, %q; want 2 and why", options, code, out)
		}
	}
}

func TestMainExitsWithWhatRunGives(t *testing.T) {
	args, was := os.Args, exit
	defer func() { os.Args, exit = args, was }()
	got := -1
	os.Args, exit = []string{"forge"}, func(code int) { got = code }
	main()
	if got != 2 {
		t.Errorf("main() exited %d with no option; want 2", got)
	}
}
