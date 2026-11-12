package fix

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// TestCommandReadsWhatKitSaysToRun reads the command out of kit's lines as they read today: up to kit's reason in
// parentheses, a ; or a , that goes on in words, a parenthesis the command did not open, or the line's end.
func TestCommandReadsWhatKitSaysToRun(t *testing.T) {
	for line, want := range map[string]string{
		"run: gh auth refresh --scopes read:project (the iterations could not be read)":  "gh auth refresh --scopes read:project",
		"no kind is defined under .kit/kinds; run: kit hydrate":                          "kit hydrate",
		"x is tracked by git; run: git rm --cached 'a, b', and commit, so it never…":     "git rm --cached 'a, b'",
		"wt exists but is not a worktree of b; remove it (or run: kit tidy) and re-run.": "kit tidy",
		"kit flow: usage does not parse; run: kit syntax --check flow":                   "kit syntax --check flow",
		"run: kit help release.": "kit help release",
		"run: kit install (kit.yml declares no kit version in this repository)": "kit install",
		"the iterations could not be read":                                      "",
		"run: kit tidy --path=a(b)c (it moved)":                                 "kit tidy --path=a(b)c",
	} {
		if got := Command(line); got != want {
			t.Errorf("Command(%q) = %q, want %q", line, got, want)
		}
	}
}

// TestWordsRunsOnlyPlainWords splits a command as a shell would, and refuses one that asks a shell for more, so
// nothing a forge or a path put in kit's line runs as shell.
func TestWordsRunsOnlyPlainWords(t *testing.T) {
	words, ok := Words(`git rm --cached 'a b' "c d" e`)
	if !ok || !slices.Equal(words, []string{"git", "rm", "--cached", "a b", "c d", "e"}) {
		t.Errorf("Words gave %q, %v", words, ok)
	}
	for _, refused := range []string{
		"", "  ", `eval "$(brew shellenv)"`, "kit hydrate; rm -rf ~", "a | b", "a > b", "a $HOME", "a `b`",
		`"a $b"`, "a *", "'open", "a && b", `a\ b`,
	} {
		if _, ok := Words(refused); ok || Runs(refused) {
			t.Errorf("Words(%q) ran", refused)
		}
	}
}

// TestAfterKeepsTheButtonsWhole clips the line, never the pair, and On finds each button where it was drawn, its
// cell either side included.
func TestAfterKeepsTheButtonsWhole(t *testing.T) {
	line := After("❌ run: gh auth refresh --scopes read:project (the iterations could not be read)", 40, true, false)
	plain := ansi.Strip(line)
	if ansi.StringWidth(plain) > 40 {
		t.Fatalf("After drew %d cells: %q", ansi.StringWidth(plain), plain)
	}
	want := "  " + " " + CopyName + " " + " " + " " + RunName + " "
	if got := plain[len(plain)-len(want):]; got != want {
		t.Fatalf("After ended %q, want %q", got, want)
	}
	w := ansi.StringWidth(plain)
	// the line ends " ⧉ ", a cell, " ▶️ ", ▶️ two cells wide: ⧉'s button starts eight cells from the end, ▶️'s four
	copyAt, runAt := w-8, w-4
	for x, b := range map[int]Button{copyAt - 1: None, copyAt: Copy, copyAt + 2: Copy, copyAt + 3: None, runAt - 1: None,
		runAt: Run, w - 1: Run, 0: None} {
		if got := On(line, x); got != b {
			t.Errorf("On(%d) = %v, want %v", x, got, b)
		}
	}
	// a band's padding goes without a …, styles and all; words that do not fit are clipped with one
	padded := ansi.Strip(After(style.Band("  jq is not installed", 40), 40, true, false))
	pair := ansi.Strip(Buttons(true, false))
	if want := "  jq is not installed" + strings.Repeat(" ", 40-21-ansi.StringWidth(pair)) + pair; padded != want {
		t.Errorf("a padded line: %q, want %q", padded, want)
	}
	if words := ansi.Strip(After(strings.Repeat("x", 40), 40, true, false)); !strings.Contains(words, "…") {
		t.Errorf("clipped words drew no …: %q", words)
	}
	copied := ansi.Strip(After("brew install jq", 40, false, true))
	if w := ansi.StringWidth(copied); On(copied, w-2) != Copy || On(copied, w-12) != None {
		t.Errorf("copied, with no run, reads wrong: %q", copied)
	}
}

// TestRunningHandsOverOnlyWords hands the terminal to a command's words with no shell, and never to a command that is
// not words: done takes ErrNotWords instead.
func TestRunningHandsOverOnlyWords(t *testing.T) {
	var ran *exec.Cmd
	handOver := func(c *exec.Cmd, cb tea.ExecCallback) tea.Cmd { ran = c; return func() tea.Msg { return cb(nil) } }
	done := func(err error) tea.Msg { return err }
	if msg := Running(handOver, "gh auth refresh --scopes read:project", done)(); msg != nil {
		t.Fatalf("Running gave %v", msg)
	}
	if ran == nil || !slices.Equal(ran.Args, []string{"gh", "auth", "refresh", "--scopes", "read:project"}) {
		t.Fatalf("Running handed over %v", ran)
	}
	ran = nil
	if msg := Running(handOver, "kit hydrate; rm -rf ~", done)(); !errors.Is(msg.(error), ErrNotWords) || ran != nil {
		t.Errorf("Running a second command gave %v, handed over %v", msg, ran)
	}
}

// Bindings names c, which copies, and r, which runs, only where the command runs here; Copying puts the command on
// the clipboard.
func TestBindingsNameTheKeysAndCopyingPutsTheCommandOnTheClipboard(t *testing.T) {
	names := func(runs bool) []string {
		var out []string
		for _, b := range Bindings(runs) {
			out = append(out, b.Key+" "+b.Does)
		}
		return out
	}
	if got := names(false); !slices.Equal(got, []string{"c copy"}) {
		t.Errorf("a fix kit only shows: %q", got)
	}
	if got := names(true); !slices.Equal(got, []string{"c copy", "r run"}) {
		t.Errorf("a fix that runs here: %q", got)
	}
	if msg := Copying("kit hydrate")(); !strings.Contains(fmt.Sprintf("%T", msg), "Clipboard") ||
		fmt.Sprint(msg) != "kit hydrate" {
		t.Errorf("Copying gives %T %v", msg, msg)
	}
}
