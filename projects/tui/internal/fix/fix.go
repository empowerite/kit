// Package fix is the pair of buttons a line that names a command offers, on every screen: ⧉, which puts the command on
// the clipboard, and ▶️, run, which hands the terminal to it. A screen draws them on the line under its cursor alone, as
// Buttons draws them, after the line clipped to leave them room, as On reads a click on them; c copies and r runs from
// the keyboard. A command comes from the screen's own data, as a welcome check's fix, or from kit's line, which names
// it after run:, as Command reads it. A command tied to a folder names it, cd and the folder then && in front, as In
// writes it and kit writes every remedy, so it runs from anywhere it is pasted. Copy goes through the terminal, so it
// reaches the clipboard over ssh too; run hands the terminal over until the command ends, in the folder the command
// names, and the screen then reads what the command changed. Run is offered only for a command of plain words, as
// Words reads them, after its cd, which runs with no shell between.
package fix

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// The buttons' names, as drawn: copy's glyph, copied in its place once it copied, until the next key or click, and
// run's, ▶️, two cells wide.
const (
	CopyName   = "⧉"
	CopiedName = "copied"
	RunName    = "▶️"
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
// copied, and ▶️ where the command runs here. A fix kit leaves to the person's own shell, as one that changes the
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
// the buttons. A line padded past its words, as a chosen row's band pads one, loses only its padding, with no … for it.
func After(line string, width int, runs, copied bool) string {
	end := Buttons(runs, copied)
	room := width - ansi.StringWidth(end)
	if words := strings.TrimRight(ansi.Strip(line), " "); ansi.StringWidth(words) <= room {
		return ansi.Truncate(line, max(0, room), "") + end
	}
	return pane.Clip(line, room) + end
}

// On gives the button the cell x of a line drawn by After is on, its styles stripped or not: copy, on ⧉ or copied
// and a cell either side, run on ▶️ and its cells, else none.
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

// Runs says whether a command can run here: it is words, as Words reads them, after the cd to its folder, as Folder
// reads it, where it names one.
func Runs(command string) bool {
	_, _, ok := Folder(command)
	return ok
}

// In gives a command that runs in a folder from anywhere: cd, the folder, and && in front of it, as kit writes a
// remedy: ~ for the home folder, ~/ before a folder under it, and the rest single-quoted where it holds more than
// letters, digits, and . _ - / + @ : ,. No folder, the command as it is.
func In(folder, command string) string {
	if folder == "" {
		return command
	}
	return "cd " + shown(folder) + " && " + command
}

// Here gives a command that runs from anywhere in the folder kittui runs in, as In writes it, as kit writes the
// remedies it prints itself; where that folder cannot be read, the command as it is.
func Here(command string) string {
	wd, err := getwd()
	if err != nil {
		return command
	}
	return In(wd, command)
}

// getwd reads the folder kittui runs in: os.Getwd, but in tests.
var getwd = os.Getwd

// shown gives a folder as In writes it.
func shown(folder string) string {
	lead := ""
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		switch {
		case folder == home:
			return "~"
		case strings.HasPrefix(folder, home+"/"):
			lead, folder = "~/", strings.TrimPrefix(folder, home+"/")
		}
	}
	if strings.Trim(folder, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._/+@:,-") == "" {
		return lead + folder
	}
	return lead + "'" + strings.ReplaceAll(folder, "'", `'\''`) + "'"
}

// Folder gives the folder a command names, as In writes it, cd and the folder then && in front, none where it names
// none, and the words of the rest, as Words reads them; false where the folder is not one word, as a shell reads it, ~
// or ~/ in front taken for the home folder, or the rest not words.
func Folder(command string) (folder string, words []string, ok bool) {
	if rest, found := strings.CutPrefix(command, "cd "); found {
		dir, after, and := strings.Cut(rest, " && ")
		if !and {
			return "", nil, false
		}
		if folder, ok = home(dir); !ok {
			return "", nil, false
		}
		command = after
	}
	words, ok = Words(command)
	return folder, words, ok
}

// home gives the folder a cd names, ~ or ~/ in front of it the home folder's, and whether it is one word.
func home(dir string) (string, bool) {
	lead := ""
	if dir == "~" || strings.HasPrefix(dir, "~/") {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", false
		}
		lead, dir = h, strings.TrimPrefix(strings.TrimPrefix(dir, "~"), "/")
		if dir == "" {
			return lead, true
		}
		lead += "/"
	}
	words, ok := Words(dir)
	if !ok || len(words) != 1 {
		return "", false
	}
	return lead + words[0], true
}

// Words gives a command's words, as a shell splits them, or false when the command asks a shell for more than words:
// a variable, a substitution, a pipe, a redirection, a glob, or a second command. Quotes group a word, and what a
// single or double quote holds is the word's own, but a $, a `, or a \ inside double quotes; a \ outside quotes makes
// the character after it the word's own, as a quote is put in a quoted folder: the quote closed, \ and the quote, and
// the quote opened again. kit's line can carry what the forge or the disk named, as a path, so a command is never
// handed to a shell: what is not plain words is copied, never run.
func Words(command string) ([]string, bool) {
	var words []string
	var word strings.Builder
	var quote rune
	in, escaped := false, false
	for _, r := range command {
		switch {
		case escaped && r == '\n':
			return nil, false
		case escaped:
			word.WriteRune(r)
			escaped = false
		case quote == 0 && r == '\\':
			escaped, in = true, true
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
	if quote != 0 || escaped {
		return nil, false
	}
	if in {
		words = append(words, word.String())
	}
	return words, len(words) > 0
}

// Exec hands the terminal to a process until it ends, as HandOver does, but in tests. what is the command as the
// person knows it, and why it runs, as HandOver says them.
type Exec = func(c *exec.Cmd, what, why string, done tea.ExecCallback) tea.Cmd

// Running hands the terminal to a command, its words run with no shell, through exec, in the folder it names, else the
// screen's; done takes how it ended once the screen has the terminal back. A command that is not words, as Runs says, does not run:
// done takes ErrNotWords at once. why is what the person is told it is for.
func Running(exec Exec, command, why string, done tea.ExecCallback) tea.Cmd {
	c, ok := Process(command)
	if !ok {
		return func() tea.Msg { return done(ErrNotWords) }
	}
	_, words, _ := Folder(command)
	return exec(c, strings.Join(words, " "), why, done)
}

// ErrNotWords is why a command did not run: it asks a shell for more than words.
var ErrNotWords = errors.New("kittui runs only plain words; copy it to run it in your shell")

// Process gives the process that runs a command's words, in the folder it names, as Folder reads it, or false when it
// is not words.
func Process(command string) (*exec.Cmd, bool) {
	folder, words, ok := Folder(command)
	if !ok {
		return nil, false
	}
	c := exec.Command(words[0], words[1:]...)
	c.Dir = folder
	return c, true
}

// HandOver is Exec for the real terminal, the one way every screen hands it over, so no hand-off lands in the middle
// of what the terminal showed before. It clears the screen, says in one line what is about to run and why, runs the
// process with the terminal its own, says how it ended, and waits for ⏎ before the screen returns, so what the process
// printed can be read first. done takes how the process ended, as tea.ExecProcess gives it.
func HandOver(c *exec.Cmd, what, why string, done tea.ExecCallback) tea.Cmd {
	return tea.Exec(&handover{cmd: c, what: what, why: why}, done)
}

// handover is the tea.ExecCommand HandOver runs: the process, between what is said before it and after it.
type handover struct {
	cmd       *exec.Cmd
	what, why string
	in        io.Reader
	out, err  io.Writer
}

func (h *handover) SetStdin(r io.Reader)  { h.in = r }
func (h *handover) SetStdout(w io.Writer) { h.out = w }
func (h *handover) SetStderr(w io.Writer) { h.err = w }

// Run says what runs, runs it, and waits for ⏎, giving how the process ended; a process the terminal could not start
// is told of too, so the person reads why before the screen returns.
func (h *handover) Run() error {
	// The child gets the terminal itself, an *os.File, which exec hands over as it is: any other reader is copied in
	// a goroutine that takes the ⏎ meant for the wait below.
	tty, ok := h.in.(*os.File)
	if !ok {
		tty = os.Stdin
	}
	if h.cmd.Stdin == nil {
		h.cmd.Stdin = tty
	}
	if h.cmd.Stdout == nil {
		h.cmd.Stdout = h.out
	}
	if h.cmd.Stderr == nil {
		h.cmd.Stderr = h.err
	}
	fmt.Fprint(h.out, Announce(h.what, h.why))
	err := h.cmd.Run()
	fmt.Fprint(h.out, Report(h.what, err))
	key := make([]byte, 1)
	for {
		if n, err := tty.Read(key); n == 1 && key[0] == '\n' || err != nil {
			break
		}
	}
	return err
}

// Announce is what the terminal reads before a hand-off: cleared, then a line saying what runs and why, and that the
// screen returns when it ends.
func Announce(what, why string) string {
	if why != "" {
		why = " " + why + ";"
	} else {
		why = ""
	}
	return "\x1b[2J\x1b[3J\x1b[H" + "Running `" + what + "`:" + why + " it has the terminal until it ends, so answer its questions if it asks any; kit returns when it ends.\n\n"
}

// Report is what the terminal reads once the process ends: how it ended, and that ⏎ brings the screen back.
func Report(what string, err error) string {
	how := "`" + what + "` succeeded."
	if err != nil {
		how = "`" + what + "` failed: " + err.Error() + "."
	}
	return "\n" + how + " Press ⏎ to return to kit.\n"
}
