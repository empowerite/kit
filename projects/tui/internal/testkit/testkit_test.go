// The helpers every package's tests share: the runner over a fake kit, the driver, the clock, the key presses of a text,
// and the golden files.
package testkit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// stop is what a recorder's Fatal panics with, so the helper stops there as a test's would.
type stop struct{}

// recorder is a T that keeps what the helpers report, so a test can see a helper fail without failing itself.
type recorder struct {
	*testing.T
	fatals, errors []string
}

func (r *recorder) Fatal(args ...any) {
	r.fatals = append(r.fatals, fmt.Sprint(args...))
	panic(stop{})
}

func (r *recorder) Fatalf(format string, args ...any) {
	r.fatals = append(r.fatals, fmt.Sprintf(format, args...))
	panic(stop{})
}

func (r *recorder) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

// recording runs the helper against a recorder, and gives what it reported.
func recording(t *testing.T, helper func(T)) *recorder {
	r := &recorder{T: t}
	func() {
		defer func() {
			if p := recover(); p != nil && p != (stop{}) {
				panic(p)
			}
		}()
		helper(r)
	}()
	return r
}

func TestTheRunnerIsTheFakeKitOfThePackageInADirectoryOfItsOwn(t *testing.T) {
	r := Runner(t, "kit")
	if !strings.HasSuffix(r.Kit, filepath.Join("internal", "kit", "testdata", "fake-kit")) {
		t.Errorf("Kit = %q", r.Kit)
	}
	if info, err := os.Stat(r.Dir); err != nil || !info.IsDir() {
		t.Errorf("Dir %q: %v", r.Dir, err)
	}
	// outside the module there is no fake kit to find, and the test stops saying so
	t.Chdir(t.TempDir())
	if got := recording(t, func(t T) { Runner(t, "kit") }); len(got.fatals) != 1 ||
		!strings.Contains(got.fatals[0], "no go.mod above") {
		t.Errorf("outside the module: %q", got.fatals)
	}
}

// From a directory that is gone, or one that cannot be searched, there is no go.mod above to find: on Linux the
// working directory of one that is gone cannot be read, and on macOS it is read, and has none above it; one that cannot
// be searched cannot be read, unless the test runs as root, and then has none above it.
func TestTheModuleRootIsNotFoundFromADirectoryThatIsGone(t *testing.T) {
	for name, leave := range map[string]func(dir string) error{
		"gone":       os.Remove,
		"unsearched": func(dir string) error { return os.Chmod(dir, 0) },
	} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), name)
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			t.Chdir(dir)
			if err := leave(dir); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Chmod(dir, 0o755) })
			if root, err := moduleRoot(); err == nil {
				t.Errorf("moduleRoot from a directory %s = %q", name, root)
			}
		})
	}
}

// log is a model that keeps every message it takes, and answers each as its table says.
type log struct {
	took    []string
	answers map[string]tea.Cmd
	init    tea.Cmd
}

func (l log) Init() tea.Cmd { return l.init }

func (l log) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	name := fmt.Sprint(msg)
	l.took = append(l.took, name)
	return l, l.answers[name]
}

func (l log) View() tea.View { return tea.NewView("") }

func say(s string) tea.Cmd { return func() tea.Msg { return s } }

// The driver feeds back what each command gives, a batch's each in turn and before the messages still queued, and
// gives every message the commands produced; a quit, or a message ends says leaves the model, a batch's too, is
// produced and not fed back. A command that gives nothing, and a batch's nil command, are passed over.
func TestDriveFeedsBackWhatTheCommandsGiveAsTheProgramWould(t *testing.T) {
	// tea.Batch drops a nil command, so the batch is built as a program may still meet one
	batch := func() tea.Msg {
		return tea.BatchMsg{say("a"), nil, func() tea.Msg { return nil }, say("done"), say("b")}
	}
	m := log{answers: map[string]tea.Cmd{
		"start": batch,
		"a":     say("leave"),
		"b":     func() tea.Msg { return nil },
		"next":  tea.Quit,
	}}
	got, produced := Drive(m, func(msg tea.Msg) bool { return msg == "leave" || msg == "done" }, "start", "idle", "next")
	if want := "[start a b idle next]"; fmt.Sprint(got.took) != want {
		t.Errorf("took %v, want %s", got.took, want)
	}
	if len(produced) != 4 || produced[1] != "done" || produced[2] != "leave" {
		t.Errorf("produced %v", produced)
	}
	if _, quit := produced[len(produced)-1].(tea.QuitMsg); !quit {
		t.Errorf("the quit was not produced: %v", produced)
	}
}

// Start gives Init's message first, when Init gives one.
func TestStartPutsInitsMessageFirst(t *testing.T) {
	for _, c := range []struct {
		init tea.Cmd
		want string
	}{
		{nil, "[x]"},
		{func() tea.Msg { return nil }, "[x]"},
		{say("first"), "[first x]"},
	} {
		if got := fmt.Sprint(Start(log{init: c.init}, "x")); got != c.want {
			t.Errorf("Start = %s, want %s", got, c.want)
		}
	}
}

func TestTypedPressesEachRune(t *testing.T) {
	got := Typed("a🏭")
	if len(got) != 2 || got[1].(tea.KeyPressMsg).Text != "🏭" || got[0].(tea.KeyPressMsg).Code != 'a' {
		t.Errorf("Typed = %v", got)
	}
}

// A press is the key Bubble Tea names so: a named key, ctrl+ or shift+ before it, and space typed as a space; or a
// character, typed, unless ctrl+ or shift+ comes before it; a modifier with nothing after it is the text it is.
func TestPressGivesTheKeyBubbleTeaNames(t *testing.T) {
	for _, c := range []struct {
		name string
		want tea.KeyPressMsg
	}{
		{"up", tea.KeyPressMsg{Code: tea.KeyUp}},
		{"ctrl+up", tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl}},
		{"shift+tab", tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}},
		{"space", tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}},
		{"x", tea.KeyPressMsg{Code: 'x', Text: "x"}},
		{"🏭", tea.KeyPressMsg{Code: '🏭', Text: "🏭"}},
		{"ctrl+c", tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}},
		{"ctrl+", tea.KeyPressMsg{Code: 'c', Text: "ctrl+"}},
		{"ctrl+shift+x", tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl | tea.ModShift}},
		{"shift+ctrl+up", tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl | tea.ModShift}},
	} {
		if got := Press(c.name); got.Code != c.want.Code || got.Mod != c.want.Mod || got.Text != c.want.Text {
			t.Errorf("Press(%q) = %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestClickReleaseDragAndColumn(t *testing.T) {
	if c := Click(3, 4); c.X != 3 || c.Y != 4 || c.Button != tea.MouseLeft {
		t.Errorf("Click = %+v", c)
	}
	if r := Release(3, 4); r.X != 3 || r.Y != 4 || r.Button != tea.MouseLeft {
		t.Errorf("Release = %+v", r)
	}
	if d := Drag(5, 6); d.X != 5 || d.Y != 6 || d.Button != tea.MouseLeft {
		t.Errorf("Drag = %+v", d)
	}
	// a two-cell glyph before the rune counts as two
	if Column("a🏭b", 'b') != 3 || Column("abc", 'z') != -1 {
		t.Errorf("Column: %d %d", Column("a🏭b", 'b'), Column("abc", 'z'))
	}
}

// A golden file that matches says nothing, one that differs is an error, and a missing one stops the test naming
// -update; with -update the file is written instead, and a file that cannot be written stops the test.
func TestGoldenComparesOrRewrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "view.golden")
	if err := os.WriteFile(path, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := recording(t, func(t T) { Golden(t, path, "same") }); len(got.errors)+len(got.fatals) != 0 {
		t.Errorf("a match reported %q %q", got.errors, got.fatals)
	}
	if got := recording(t, func(t T) { Golden(t, path, "other") }); len(got.errors) != 1 {
		t.Errorf("a difference reported %q", got.errors)
	}
	missing := filepath.Join(t.TempDir(), "none.golden")
	if got := recording(t, func(t T) { Golden(t, missing, "x") }); len(got.fatals) != 1 ||
		!strings.Contains(got.fatals[0], "go test -update") {
		t.Errorf("a missing file reported %q", got.fatals)
	}
	*update = true
	t.Cleanup(func() { *update = false })
	if got := recording(t, func(t T) { Golden(t, path, "rewritten") }); len(got.errors)+len(got.fatals) != 0 {
		t.Errorf("-update reported %q %q", got.errors, got.fatals)
	}
	if b, _ := os.ReadFile(path); string(b) != "rewritten" {
		t.Errorf("-update wrote %q", b)
	}
	unwritable := filepath.Join(t.TempDir(), "no-dir", "view.golden")
	if got := recording(t, func(t T) { Golden(t, unwritable, "x") }); len(got.fatals) != 1 {
		t.Errorf("an unwritable file reported %q", got.fatals)
	}
}

// A test's clock keeps each wait and brings nothing until it fires them, in the order they were set.
func TestAClockFiresWhenTheTestSays(t *testing.T) {
	var c Clock
	if c.After(time.Minute, "one") != nil || c.After(time.Second, "two") != nil {
		t.Error("a wait gave a command")
	}
	if timers := c.Timers(); len(timers) != 2 || timers[0] != (Timer{time.Minute, "one"}) {
		t.Errorf("the timers: %+v", timers)
	}
	if got := c.Fire(); len(got) != 2 || got[0] != "one" || got[1] != "two" || len(c.Timers()) != 0 {
		t.Errorf("fired %v, left %v", got, c.Timers())
	}
}
