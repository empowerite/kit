// The Open… dialog, which the workspace menu's Open… opens over the screen shown: the folder of a repository typed, as
// a terminal's prompt takes one, ~ for the home folder and a path without / read from there; tab completes it a folder
// at a time, and the folders it can complete to stand under it, each opened into by a plain click. Enter, or a click on
// Open, asks kittui to open what is typed; kit says whether it is a git working tree, and its line, when it is not,
// stands in the dialog, which stays open. Esc, or a click off the box, closes it.
package shell

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/style"
)

// OpenMsg asks kittui to open the Open… dialog over the screen shown.
type OpenMsg struct{}

// The dialog's glyphs: a folder it can complete to, and its line that opens what is typed. Each is two cells wide, as
// every glyph in a menu is.
const (
	Folder  = "📁"
	Opening = "📂"
)

// openShown is how many folders the dialog lists under what is typed; the rest it counts.
const openShown = 8

// OpenDialog is the Open… dialog: whether it is open, which then has the keyboard; the folder typed, as written; kit's
// line for the last folder it was asked to open, when it refused; whether kit is being asked; and the home folder, which
// ~ stands for and a path without / is read from.
type OpenDialog struct {
	Open    bool
	Typed   string
	Refusal string
	Busy    bool
	Home    string
}

// NewOpen gives the dialog open, the folder start typed, with a / after it, ~ in place of home; with no start, ~/.
func NewOpen(start, home string) OpenDialog {
	d := OpenDialog{Open: true, Home: home, Typed: "~/"}
	if start != "" {
		d.Typed = strings.TrimSuffix(tilde(filepath.Clean(start), home), "/") + "/"
	}
	return d
}

// tilde writes a path under home with ~ in its place.
func tilde(path, home string) string {
	switch {
	case home == "":
	case path == home:
		return "~"
	case strings.HasPrefix(path, home+"/"):
		return "~" + path[len(home):]
	}
	return path
}

// expand gives a path as written in the dialog, absolute: ~ the home folder, and a path without / read from there.
func (d OpenDialog) expand(written string) string {
	switch {
	case written == "~" || strings.HasPrefix(written, "~/"):
		return d.Home + written[1:]
	case strings.HasPrefix(written, "/"):
		return written
	}
	return filepath.Join(d.Home, written)
}

// Path gives the folder typed, absolute and clean; nothing when nothing is typed.
func (d OpenDialog) Path() string {
	if strings.TrimSpace(d.Typed) == "" {
		return ""
	}
	return filepath.Clean(d.expand(d.Typed))
}

// parts gives what is typed split at its last /: the folder it stands in, as written with its /, and the start of the
// name typed after it.
func (d OpenDialog) parts() (folder, start string) {
	k := strings.LastIndex(d.Typed, "/")
	return d.Typed[:k+1], d.Typed[k+1:]
}

// Folders gives the names of the folders what is typed can complete to, in order: those in the folder it stands in
// whose names start with what is typed after its last /, a hidden one only once a . is typed.
func (d OpenDialog) Folders() []string {
	folder, start := d.parts()
	dir := d.expand(folder)
	if folder == "" {
		dir = d.Home
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, start) || (strings.HasPrefix(name, ".") && !strings.HasPrefix(start, ".")) {
			continue
		}
		if info, err := os.Stat(filepath.Join(dir, name)); err == nil && info.IsDir() {
			out = append(out, name)
		}
	}
	return out
}

// complete gives what is typed completed as far as the folders it can complete to agree: one folder, whole, with a /
// after it; several, the start they share; none, as it was.
func (d OpenDialog) complete() string {
	names := d.Folders()
	folder, _ := d.parts()
	switch len(names) {
	case 0:
		return d.Typed
	case 1:
		return folder + names[0] + "/"
	}
	shared := names[0]
	for _, n := range names[1:] {
		for !strings.HasPrefix(n, shared) {
			shared = shared[:len(shared)-1]
		}
	}
	return folder + shared
}

// ask gives the dialog asking to open the folder typed, and the folder; nothing while kit is asked, or with nothing
// typed.
func (d OpenDialog) ask() (OpenDialog, string) {
	path := d.Path()
	if d.Busy || path == "" {
		return d, ""
	}
	d.Busy, d.Refusal = true, ""
	return d, path
}

// Refused gives the dialog told kit's line for the folder it asked to open: open still, the line under what is typed.
func (d OpenDialog) Refused(line string) OpenDialog {
	d.Busy, d.Refusal = false, line
	return d
}

// Key takes a key while the dialog is open: a printable key types, backspace takes one back, ⌃U clears, tab completes,
// enter asks to open what is typed, and esc closes the dialog. It gives the dialog and the folder to open, "" for none.
func (d OpenDialog) Key(k tea.KeyPressMsg) (OpenDialog, string) {
	switch k.String() {
	case "esc":
		d.Open = false
		return d, ""
	case "enter":
		return d.ask()
	case "tab":
		d.Typed, d.Refusal = d.complete(), ""
		return d, ""
	case "ctrl+u":
		d.Typed, d.Refusal = "", ""
		return d, ""
	}
	if typed, ok := Type(d.Typed, k); ok {
		d.Typed, d.Refusal = typed, ""
	}
	return d, ""
}

// Keys gives the dialog's segment of the keys bar: it takes every press.
func (d OpenDialog) Keys() Segment {
	s := DialogKeys("esc close · ⇥ complete · ⏎ open")
	s.Bindings[1].Presses = []string{"tab"}
	s.Bindings[2].Presses = []string{"enter"}
	return s
}

// openLine is one line of the dialog's body, and what a click on it does: completes to a folder, or opens.
type openLine struct {
	text   string
	folder string
	opens  bool
}

// body gives the dialog's lines under its top edge: what is typed, a cursor after it; kit's line, or what the dialog
// takes; the folders it can complete to, as many as it lists, then how many more; and Open, dim while nothing is typed
// or kit is asked.
func (d OpenDialog) body() []openLine {
	lines := []openLine{{text: "› " + d.Typed + "▏"}}
	if d.Refusal != "" {
		lines = append(lines, openLine{text: style.Refusal(Failed + " " + d.Refusal)})
	} else {
		lines = append(lines, openLine{text: style.Off("the folder of a git clone; ⇥ completes, a click opens a folder")})
	}
	lines = append(lines, openLine{})
	names := d.Folders()
	folder, _ := d.parts()
	for k, n := range names {
		if k == openShown {
			lines = append(lines, openLine{text: style.Off("… " + strconv.Itoa(len(names)-openShown) + " more")})
			break
		}
		lines = append(lines, openLine{text: Folder + " " + n + "/", folder: folder + n + "/"})
	}
	if len(names) == 0 {
		lines = append(lines, openLine{text: style.Off("no folder here starts so")})
	}
	open := Opening + " Open"
	if d.Busy || d.Path() == "" {
		open = style.Off(open)
	}
	return append(lines, openLine{}, openLine{text: open, opens: true})
}

// Failed starts kit's line in the dialog.
const Failed = "❌"

// placed gives the dialog's box where it stands over a screen of so many lines, so wide: its third line, or as near as
// the screen lets, and centered across it, wide enough for a long path.
func (d OpenDialog) placed(height, width int) Placed {
	var texts []string
	for _, l := range d.body() {
		texts = append(texts, l.text)
	}
	texts[0] = ansi.Truncate(texts[0], width-6, "…")
	// a line of spaces, taken out again, keeps the box wide enough for a long path as it is typed
	box := Dialog("Open a repository", append(texts, strings.Repeat(" ", min(60, max(0, width-6)))), width-2)
	box = append(box[:len(box)-2], box[len(box)-1])
	return Placed{Lines: box, Top: min(2, max(0, height-len(box))), Left: max(0, (width-ansi.StringWidth(box[0]))/2)}
}

// Overlay draws the open dialog over the screen's lines, so wide.
func (d OpenDialog) Overlay(lines []string, width int) []string {
	if !d.Open {
		return lines
	}
	return d.placed(len(lines), width).Draw(lines)
}

// Click takes a click while the dialog is open, over a screen of so many lines, so wide: a left one on a folder
// completes to it, one on Open asks to open what is typed, as enter does, and one off the box closes the dialog;
// anything else changes nothing. It gives the dialog and the folder to open, "" for none.
func (d OpenDialog) Click(msg tea.MouseClickMsg, height, width int) (OpenDialog, string) {
	mouse := msg.Mouse()
	if mouse.Button != tea.MouseLeft {
		return d, ""
	}
	k, on := d.placed(height, width).Hit(mouse.X, mouse.Y)
	body := d.body()
	switch {
	case !on:
		d.Open = false
	case k >= 0 && k < len(body) && body[k].folder != "":
		d.Typed, d.Refusal = body[k].folder, ""
	case k >= 0 && k < len(body) && body[k].opens:
		return d.ask()
	}
	return d, ""
}
