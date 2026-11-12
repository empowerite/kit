// Package fix is the pair of buttons a line that names a command offers, on every screen: ⧉, which puts the command on
// the clipboard, and run, which hands the terminal to it. A screen draws them on the line under its cursor alone, as
// Buttons draws them, after the line clipped to leave them room, as On reads a click on them; c copies and r runs from
// the keyboard. A command comes from the screen's own data, as a welcome check's fix, or from kit's line, which names
// it after run:, as Command reads it. Copy goes through the terminal, so it reaches the clipboard over ssh too; run
// hands the terminal over until the command ends, and the screen then reads what the command changed. Run is offered
// only for a command of plain words, as Words reads them, which runs with no shell between.
package fix

import (
	"errors"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// The buttons' names, as drawn: copy's glyph, copied in its place once it copied, until the next key or click, and run.
const (
	CopyName   = "⧉"
	CopiedName = "copied"
	RunName    = "run"
)

// The keys that press the buttons on the line under the cursor.
const (
	CopyKey = "c"
	RunKey  = "r"
)

// Button is a button of the pair: none, copy, or run.
type Button int

// The buttons On tells apart.
const (
	None Button = iota
	Copy
	Run
)

// Command gives the command kit's line names after run:, the first time it says run:, as far as the line goes on
// about it: up to a parenthesis that opens kit's reason, as in run: gh auth refresh --scopes read:project (the
// iterations could not be read), one that closes a parenthesis the command did not open, a ; or a , that goes on in
// words, or the line's end, and a full stop that ends the line. Quoted text is the command's own. No run:, no command.
func Command(line string) string {
	_, rest, ok := strings.Cut(line, "run: ")
	if !ok {
		return ""
	}
	var quote rune
	depth, end := 0, len(rest)
scan:
	for i, r := range rest {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
		case r == '(' && depth == 0 && strings.HasSuffix(rest[:i], " "):
			end = i
			break scan
		case r == '(':
			depth++
		case r == ')' && depth == 0:
			end = i
			break scan
		case r == ')':
			depth--
		case (r == ';' || r == ',') && strings.HasPrefix(rest[i+1:], " "):
			end = i
			break scan
		}
	}
	return strings.TrimSuffix(strings.TrimSpace(rest[:end]), ".")
}

// Buttons draws the pair after a line, a cell either side of each name, which a click takes: ⧉, or copied once it
// copied, and run where the command runs here. A fix kit leaves to the person's own shell, as one that changes the
// shell's own environment, offers ⧉ alone.
func Buttons(runs, copied bool) string {
	names := []string{style.Tint(" " + CopyName + " ")}
	if copied {
		names[0] = style.Off(" " + CopiedName + " ")
	}
	if runs {
		names = append(names, style.Tint(" "+RunName+" "))
	}
	return "  " + strings.Join(names, " ")
}

// After gives a line with the pair after it, in width cells: the line clipped to leave the buttons their room, never
// the buttons.
func After(line string, width int, runs, copied bool) string {
	end := Buttons(runs, copied)
	return pane.Clip(line, width-ansi.StringWidth(end)) + end
}

// On gives the button the cell x of a line drawn by After is on, its styles stripped or not: copy, on ⧉ or copied
// and a cell either side, run on run and its cells, else none.
func On(line string, x int) Button {
	plain := ansi.Strip(line)
	if at, ok := cells(plain, " "+RunName+" "); ok && x >= at && x < at+ansi.StringWidth(RunName)+2 {
		return Run
	}
	for _, name := range []string{CopyName, CopiedName} {
		if at, ok := cells(plain, " "+name+" "); ok && x >= at && x < at+ansi.StringWidth(name)+2 {
			return Copy
		}
	}
	return None
}

// cells gives the screen column a button starts at on a plain line, its last time there, as the pair ends the line.
func cells(plain, button string) (int, bool) {
	before, _, ok := strings.CutLast(plain, button)
	return ansi.StringWidth(before), ok
}

// Bindings gives the keys bar's names for the pair: c copies, and r runs where the command runs here.
func Bindings(runs bool) []shell.Binding {
	out := []shell.Binding{shell.Bind(CopyKey, "copy")}
	if runs {
		out = append(out, shell.Bind(RunKey, "run"))
	}
	return out
}

// Copying puts a command on the clipboard, through the terminal.
func Copying(command string) tea.Cmd { return tea.SetClipboard(command) }

// Runs says whether a command can run here: it is words, as Words reads them.
func Runs(command string) bool {
	_, ok := Words(command)
	return ok
}

// Words gives a command's words, as a shell splits them, or false when the command asks a shell for more than words:
// a variable, a substitution, a pipe, a redirection, a glob, a second command, or an escape. Quotes group a word, and
// what a single or double quote holds is the word's own, but a $ or a ` inside double quotes. kit's line can carry
// what the forge or the disk named, as a path, so a command is never handed to a shell: what is not plain words is
// copied, never run.
func Words(command string) ([]string, bool) {
	var words []string
	var word strings.Builder
	var quote rune
	in := false
	for _, r := range command {
		switch {
		case quote == '\'' && r == '\'', quote == '"' && r == '"':
			quote = 0
		case quote == '"' && (r == '$' || r == '`' || r == '\\'):
			return nil, false
		case quote != 0:
			word.WriteRune(r)
		case r == '\'' || r == '"':
			quote, in = r, true
		case r == ' ' || r == '\t':
			if in {
				words, in = append(words, word.String()), false
				word.Reset()
			}
		case strings.ContainsRune("$`\\|&;<>()*?[]{}~#!\n", r):
			return nil, false
		default:
			word.WriteRune(r)
			in = true
		}
	}
	if quote != 0 {
		return nil, false
	}
	if in {
		words = append(words, word.String())
	}
	return words, len(words) > 0
}

// Exec hands the terminal to a process until it ends: tea.ExecProcess, but in tests.
type Exec = func(*exec.Cmd, tea.ExecCallback) tea.Cmd

// Running hands the terminal to a command, its words run in the screen's folder with no shell, through exec; done
// takes how it ended once the screen has the terminal back. A command that is not words, as Runs says, does not run:
// done takes ErrNotWords at once.
func Running(exec Exec, command string, done tea.ExecCallback) tea.Cmd {
	c, ok := Process(command)
	if !ok {
		return func() tea.Msg { return done(ErrNotWords) }
	}
	return exec(c, done)
}

// ErrNotWords is why a command did not run: it asks a shell for more than words.
var ErrNotWords = errors.New("kittui runs only plain words; copy it to run it in your shell")

// Process gives the process that runs a command's words, or false when it is not words.
func Process(command string) (*exec.Cmd, bool) {
	words, ok := Words(command)
	if !ok {
		return nil, false
	}
	return exec.Command(words[0], words[1:]...), true
}
