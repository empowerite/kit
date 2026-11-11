// The Open… dialog, which the workspace menu's Open… opens over the screen shown: the folder of a repository typed, as
// a terminal's prompt takes one, ~ for the home folder and a path without / read from there; tab completes it a folder
// at a time, and the folders it can complete to stand under it, each opened into by a plain click. Up and down move a
// highlight from what is typed onto the folders and back, page up and page down by a page, and the list scrolls to
// keep it in view, as the mouse wheel scrolls it, saying how many folders it hides above and below; enter on a folder
// highlighted goes into it, as a click does, and typing puts the highlight back on what is typed. Shift+backspace and
// ⌃W go up a folder, taking the last name off what is typed; while there is a folder above, a ../ line starts the
// list, pinned as it scrolls, and a click on it, or enter on it, goes up the same way. Enter with no folder
// highlighted, or a click on Open, asks kittui to open what is typed; kit says whether it is a git working tree, and
// its line, when it is not, stands in the dialog, which stays open. Esc, or a click off the box, closes it.
package shell

import (
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
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

// How the dialog lists the folders under what is typed: so many at its top, then a line that counts the rest; so many
// lines at most, a line that counts those hidden above taking a folder's place once it scrolls; a page, the folders
// shown between the two counts, which page up and page down move by; and the lines a turn of the wheel scrolls.
const (
	openShown = 8
	openRows  = openShown + 1
	openPage  = openRows - 2
	openWheel = 3
)

// OpenDialog is the Open… dialog: whether it is open, which then has the keyboard; the folder typed, as written; kit's
// line for the last folder it was asked to open, when it refused; whether kit is being asked; the home folder, which ~
// stands for and a path without / is read from; the line highlighted, 0 for what is typed and k for the kth line of the
// list; and the first line of the list shown.
type OpenDialog struct {
	Open    bool
	Typed   string
	Refusal string
	Busy    bool
	Home    string
	At      int
	Top     int
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
// and shift+backspace or ⌃W goes up a folder, each putting the highlight back on what is typed; up and down move the highlight a line, page up and page
// down a page; enter goes into the folder highlighted, or with none asks to open what is typed; and esc closes the
// dialog. It gives the dialog and the folder to open, "" for none.
func (d OpenDialog) Key(k tea.KeyPressMsg) (OpenDialog, string) {
	switch key := k.String(); key {
	case "esc":
		d.Open = false
		return d, ""
	case "up", "down", "pgup", "pgdown":
		return d.move(key), ""
	case "shift+backspace", "ctrl+w":
		return d.typing(d.up()), ""
	case "enter":
		if list, _ := d.list(); d.At > 0 && d.At <= len(list) {
			return d.typing(list[d.At-1].folder), ""
		}
		return d.ask()
	case "tab":
		return d.typing(d.complete()), ""
	case "ctrl+u":
		return d.typing(""), ""
	}
	if typed, ok := Type(d.Typed, k); ok {
		return d.typing(typed), ""
	}
	return d, ""
}

// typing gives the dialog with typed in place of what is typed: kit's line gone, the highlight back on what is typed,
// and the list shown from its top.
func (d OpenDialog) typing(typed string) OpenDialog {
	d.Typed, d.Refusal, d.At, d.Top = typed, "", 0, 0
	return d
}

// up gives what is typed a folder up: the folder it names without its last name, with a / after it and ~ in place of
// home; / stays /, and the folder above home is written whole.
func (d OpenDialog) up() string {
	dir := filepath.Dir(filepath.Clean(d.expand(d.Typed)))
	if dir == "/" {
		return dir
	}
	return tilde(dir, d.Home) + "/"
}

// move gives the dialog its highlight moved by a key, between what is typed and the list's last line: up and down a
// line, page down a page, and page up a page, stopping at the list's first line; the list scrolled to keep it in view,
// and to its top on the line it pins.
func (d OpenDialog) move(key string) OpenDialog {
	list, head := d.list()
	n := len(list)
	switch key {
	case "up":
		d.At--
	case "down":
		d.At++
	case "pgup":
		if d.At > 0 {
			d.At = max(1, d.At-openPage)
		}
	case "pgdown":
		d.At += openPage
	}
	if d.At = min(max(d.At, 0), n); d.At > head {
		d.Top = pane.Frame(n-head, openRows, d.Top, d.At-1-head, d.At-1-head).Start
	} else {
		d.Top = 0
	}
	return d
}

// Wheel takes a turn of the mouse wheel while the dialog is open: it scrolls the list a few lines, never past its ends,
// the highlight staying where it is.
func (d OpenDialog) Wheel(w tea.MouseWheelMsg) OpenDialog {
	switch w.Button {
	case tea.MouseWheelUp:
		d.Top -= openWheel
	case tea.MouseWheelDown:
		d.Top += openWheel
	}
	list, head := d.list()
	d.Top = min(max(d.Top, 0), max(0, len(list)-head-openRows+1))
	return d
}

// Keys gives the dialog's segment of the keys bar: it takes every press, and names up and down while the list has a
// line to move to, page up and page down while it has more than it shows, and going up while there is a folder above.
func (d OpenDialog) Keys() Segment {
	s := Segment{Owns: true, Bindings: []Binding{Bind("esc", "close")}}
	list, head := d.list()
	if len(list) > 0 {
		s = s.With(Bind("↑↓", "move", "up", "down"))
	}
	if len(list)-head > openRows {
		s = s.With(Bind("⇞⇟", "page", "pgup", "pgdown"))
	}
	s = s.With(Bind("⇥", "complete", "tab"))
	if head > 0 {
		s = s.With(Bind("⇧⌫", "up", "shift+backspace"), Bind("⌃W", "up", "ctrl+w"))
	}
	return s.With(Bind("⏎", "open", "enter"))
}

// openLine is one line of the dialog's body, and what a click on it does: completes to a folder, or opens; and whether
// it is highlighted.
type openLine struct {
	text   string
	folder string
	opens  bool
	lit    bool
}

// list gives the lines of the dialog's list, and how many of them it pins atop it, never scrolled: ../ while there is a
// folder above what is typed, which a click on it goes up to; then each folder what is typed can complete to, which a
// click on it types.
func (d OpenDialog) list() (lines []openLine, head int) {
	if filepath.Clean(d.expand(d.Typed)) != "/" {
		lines, head = append(lines, openLine{text: Folder + " ../", folder: d.up()}), 1
	}
	folder, _ := d.parts()
	for _, n := range d.Folders() {
		lines = append(lines, openLine{text: Folder + " " + n + "/", folder: folder + n + "/"})
	}
	return lines, head
}

// body gives the dialog's lines under its top edge: what is typed, a cursor after it; kit's line, or what the dialog
// takes; the lines its list pins, then those it shows from its first shown, the one highlighted among them, and how
// many it hides above and below; and Open, dim while nothing is typed or kit is asked.
func (d OpenDialog) body() []openLine {
	lines := []openLine{{text: "› " + d.Typed + "▏"}}
	if d.Refusal != "" {
		lines = append(lines, openLine{text: style.Refusal(Failed + " " + d.Refusal)})
	} else {
		lines = append(lines, openLine{text: style.Off("the folder of a git clone; ⇥ completes, a click opens a folder")})
	}
	lines = append(lines, openLine{})
	list, head := d.list()
	shown := func(k int) {
		l := list[k]
		l.lit = k+1 == d.At
		lines = append(lines, l)
	}
	for k := range head {
		shown(k)
	}
	start, end, above, below := pane.Window(len(list)-head, openRows, d.Top)
	if above {
		lines = append(lines, openLine{text: style.Off(pane.Above(start))})
	}
	for k := start; k < end; k++ {
		shown(head + k)
	}
	if below {
		lines = append(lines, openLine{text: style.Off(pane.More(len(list) - head - end))})
	}
	if len(list) == head {
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
// the screen lets, and centered across it, wide enough for a long path; the line highlighted drawn across the box, as a
// menu draws the line under the keyboard.
func (d OpenDialog) placed(height, width int) Placed {
	var texts []string
	body := d.body()
	for _, l := range body {
		texts = append(texts, l.text)
	}
	texts[0] = ansi.Truncate(texts[0], width-6, "…")
	// a line of spaces, taken out again, keeps the box wide enough for a long path as it is typed
	box := Dialog("Open a repository", append(texts, strings.Repeat(" ", min(60, max(0, width-6)))), width-2)
	box = append(box[:len(box)-2], box[len(box)-1])
	inner := ansi.StringWidth(box[0]) - 2
	for k, l := range body {
		if l.lit {
			box[k+1] = style.Menu("│") + style.Selected(" "+pane.Pad(ansi.Truncate(l.text, inner-2, "…"), inner-2,
				false)+" ") + style.Menu("│")
		}
	}
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
		d = d.typing(body[k].folder)
	case k >= 0 && k < len(body) && body[k].opens:
		return d.ask()
	}
	return d, ""
}
