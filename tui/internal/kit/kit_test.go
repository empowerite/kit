// The runner asks kit for the grammar, the help, the choices, and the extension, and execs the verb where the command
// was typed; every test runs it against testdata/fake-kit.
package kit

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fake gives a Runner over testdata/fake-kit in the directory dir.
func fake(t *testing.T, dir string) Runner {
	t.Helper()
	path, err := filepath.Abs("testdata/fake-kit")
	if err != nil {
		t.Fatal(err)
	}
	return Runner{Kit: path, Dir: dir}
}

func TestVerbsAndSpec(t *testing.T) {
	r := fake(t, "")
	verbs, err := r.Verbs()
	if err != nil || len(verbs) != 2 || verbs[0].Verb != "commit" || verbs[1].Verb != "version" {
		t.Fatalf("Verbs: %v, %+v", err, verbs)
	}
	spec, err := r.Spec("commit")
	if err != nil || spec.Verb != "commit" || len(spec.Syntax[0]) != 3 {
		t.Fatalf("Spec: %v, %+v", err, spec)
	}
	_, err = r.Spec("nonesuch")
	var refusal *ExitError
	if !errors.As(err, &refusal) || refusal.Code != 2 || !strings.Contains(refusal.Stderr, "no kit verb 'nonesuch'") {
		t.Fatalf("Spec(nonesuch): %v", err)
	}
}

func TestHelpIsTheVerbsPage(t *testing.T) {
	page, err := fake(t, "").Help("commit")
	if err != nil || !strings.HasPrefix(page, "kit commit - a purpose for the test\n") || !strings.Contains(page, "Usage\n") {
		t.Fatalf("Help: %v, %q", err, page)
	}
}

func TestChoicesColumnsAndAnswers(t *testing.T) {
	rows, err := fake(t, "").Choices("package", "<package>", []string{"<package>=house", "--x=y"})
	if err != nil || len(rows) != 3 {
		t.Fatalf("Choices: %v, %+v", err, rows)
	}
	if rows[0] != (Choice{Value: "a", Description: "the first", State: "active"}) {
		t.Errorf("row 0: %+v", rows[0])
	}
	if rows[1] != (Choice{Value: "b", State: "active", Refusal: "b: kept, c depend(s) on it"}) {
		t.Errorf("row 1: %+v", rows[1])
	}
	// the answers reach kit after a --, so one that starts with - is never read as kit's own
	if rows[2].Description != "package <package> <package>=house --x=y" {
		t.Errorf("answers: %q", rows[2].Description)
	}
}

func TestExtendBeforeAndAfterTheAnswers(t *testing.T) {
	r := fake(t, "")
	keys, before, err := r.Extend("new", nil)
	if err != nil || len(before) != 0 || len(keys) != 0 {
		t.Fatalf("Extend(nil): %v, %+v", err, before)
	}
	keys, after, err := r.Extend("new", []string{"--rule=markdown"})
	if err != nil || !after["--title"].Extended || strings.Join(keys, " ") != "--title --purpose" {
		t.Fatalf("Extend(rule): %v, %v, %+v", err, keys, after)
	}
}

// TestExecArgsAndDir observes Exec through a helper process, since Exec replaces the process that calls it: the
// helper execs fake-kit, which records its arguments and working directory.
func TestExecArgsAndDir(t *testing.T) {
	record := filepath.Join(t.TempDir(), "record")
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperExec")
	cmd.Env = append(os.Environ(), "KIT_TEST_HELPER=1", "FAKE_KIT_RECORD="+record, "kit_invoked_in="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("helper: %v\n%s", err, out)
	}
	got, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(got)), "\n")
	if len(lines) != 2 || lines[0] != "commit --message a b x.txt" {
		t.Errorf("argv: %q", lines)
	}
	want, _ := filepath.EvalSymlinks(dir)
	if len(lines) == 2 && lines[1] != want {
		t.Errorf("cwd %q, want %q", lines[1], want)
	}
}

// TestHelperExec is the body of TestExecArgsAndDir's helper process; it runs only there.
func TestHelperExec(t *testing.T) {
	if os.Getenv("KIT_TEST_HELPER") != "1" {
		t.Skip("the helper runs under TestExecArgsAndDir alone")
	}
	path, _ := filepath.Abs("testdata/fake-kit")
	os.Setenv("KIT", path)
	r := New()
	if err := r.Exec("commit", []string{"--message", "a b", "x.txt"}); err != nil {
		t.Fatalf("Exec: %v", err)
	}
}

func TestNewReadsTheEnvironment(t *testing.T) {
	t.Setenv("KIT", "/somewhere/kit")
	t.Setenv("kit_invoked_in", "/there")
	r := New()
	if r.Kit != "/somewhere/kit" || r.Dir != "/there" {
		t.Errorf("New: %+v", r)
	}
	t.Setenv("KIT", "")
	t.Setenv("kit_invoked_in", "")
	r = New()
	cwd, _ := os.Getwd()
	if r.Kit != "kit" || r.Dir != cwd {
		t.Errorf("New defaults: %+v", r)
	}
}

func TestMatchJudgesTheWordsAfterADoubleDash(t *testing.T) {
	m, err := fake(t, "").Match("commit", []string{"--message", "hi"})
	if err != nil || m.Status != 0 || m.Operands[0] != "x" || m.Options["--message"][0] != "--message hi" {
		t.Fatalf("Match: %v, %+v", err, m)
	}
}
