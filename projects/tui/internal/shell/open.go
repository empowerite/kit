// The Open… dialog, which the workspace menu's Open… opens over the screen shown: the folder of a repository typed, as
// a terminal's prompt takes one, ~ for the home folder and a path without / read from there; tab completes it a folder
// at a time, and the folders it can complete to stand under it, each opened into by a plain click. Up and down move a
// highlight from what is typed onto the folders and back, page up and page down by a page, and the list scrolls to
// keep it in view, as the mouse wheel scrolls it, saying how many folders it hides above and below; enter on a folder
// highlighted goes into it, as a click does, and typing puts the highlight back on what is typed. Shift+backspace and
// ⌃W go up a folder, taking the last name off what is typed; while there is a folder above, a ../ line starts the
// list, pinned as it scrolls, and a click on it, or enter on it, goes up the same way. Enter with no folder
// highlighted, or a click on Open, asks kittui to open what is typed; kit says whether it is a git working tree, and
// its line, when it is not, stands in the dialog, which stays open. A line above Open, or ⌃F, finds the clones under
// the folder typed: a command walks it for git working trees, never blocking the screen, while a dim line says it is
// finding them where the folders were; the clones then stand in their place, scrolling as they do, each asked to open
// by a click or enter on it, as Open asks for what is typed. Esc, or typing, goes back to the folders, and the answer
// of a walk begun before is dropped. Esc over the folders, or a click off the box, closes the dialog.
package shell

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// OpenMsg asks kittui to open the Open… dialog over the screen shown.
type OpenMsg struct{}

// The dialog's glyphs: a folder it can complete to, its line that opens what is typed, its line that finds the clones
// under it, and a clone found. Each is two cells wide, as every glyph in a menu is.
const (
	Folder  = "📁"
	Opening = "📂"
	Finding = "🔍"
	Clone   = "📦"
)

// How far a walk for clones goes: so many folders down from the one typed, and so many clones found.
const (
	cloneDepth = 6
	cloneMost  = 500
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
// list; and the first line of the list shown. A walk for clones holds whether one is on; the folder walked; how many
// walks were begun, which tells an answer from one begun before; and the walk's answer, whose clones stand in place of
// the folders, nil while they do not. The dialog compares as a value, its answer by the walk that gave it.
type OpenDialog struct {
	Open    bool
	Typed   string
	Refusal string
	Busy    bool
	Home    string
	At      int
	Top     int
	finding bool
	under   string
	walks   int
	found   *ClonesMsg
}

// ClonesMsg is a walk's answer: the walk, by its count; the folder walked; the clones under it, each from it; and
// whether the walk stopped at the most it finds.
type ClonesMsg struct {
	Walk   int
	Under  string
	Clones []string
	Capped bool
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
func (d OpenDialog) ask() (OpenDialog, string) { return d.askFor(d.Path()) }

// askFor gives the dialog asking to open the folder given, and the folder; nothing while kit is asked, or with none.
func (d OpenDialog) askFor(path string) (OpenDialog, string) {
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
// and shift+backspace or ⌃W goes up a folder, each putting the highlight back on what is typed and the folders back in
// place of the clones; up and down move the highlight a line, page up and page down a page; enter takes the line
// highlighted, as a click on it does, or with none asks to open what is typed; ⌃F finds the clones under the folder
// typed; and esc goes back from the clones to the folders, or over the folders closes the dialog. It gives the dialog
// and the folder to open, "" for none.
func (d OpenDialog) Key(k tea.KeyPressMsg) (OpenDialog, string) {
	switch key := k.String(); key {
	case "esc":
		if d.finding || d.found != nil {
			return d.typing(d.Typed), ""
		}
		d.Open = false
		return d, ""
	case "up", "down", "pgup", "pgdown":
		return d.move(key), ""
	case "shift+backspace", "ctrl+w":
		return d.typing(d.up()), ""
	case "ctrl+f":
		return d.find(), ""
	case "enter":
		if list, _ := d.list(); d.At > 0 && d.At <= len(list) {
			return d.take(list[d.At-1])
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
// the folders in place of the clones, a walk on left to be dropped, and the list shown from its top.
func (d OpenDialog) typing(typed string) OpenDialog {
	d.Typed, d.Refusal, d.At, d.Top = typed, "", 0, 0
	d.finding, d.found = false, nil
	return d
}

// take gives the dialog a line of its list taken, by a click or enter: a clone asked to open, or a folder typed.
func (d OpenDialog) take(l openLine) (OpenDialog, string) {
	if l.repo != "" {
		return d.askFor(l.repo)
	}
	return d.typing(l.folder), ""
}

// find gives the dialog walking the folder typed for clones, its list given over to them; as it was while nothing is
// typed, kit is asked, or a walk is on.
func (d OpenDialog) find() OpenDialog {
	if d.Path() == "" || d.Busy || d.finding {
		return d
	}
	d = d.typing(d.Typed)
	d.finding, d.under, d.walks = true, d.Path(), d.walks+1
	return d
}

// Walk gives the command that walks the folder for clones when the dialog began a walk since it was was, as a key or a
// click begins one; nil when it began none. The command answers with a ClonesMsg.
func (d OpenDialog) Walk(was OpenDialog) tea.Cmd {
	if !d.finding || d.walks == was.walks {
		return nil
	}
	walk, under := d.walks, d.under
	return func() tea.Msg {
		clones, capped := findClones(under, cloneDepth, cloneMost)
		return ClonesMsg{Walk: walk, Under: under, Clones: clones, Capped: capped}
	}
}

// Found gives the dialog a walk's answer: the clones in place of the folders, when it answers the walk on; else the
// dialog as it was, as for a walk begun before, or one left as the folder typed changed.
func (d OpenDialog) Found(msg ClonesMsg) OpenDialog {
	if !d.finding || msg.Walk != d.walks || msg.Under != d.Path() {
		return d
	}
	d.finding, d.found = false, &msg
	return d
}

// findClones gives the git working trees under a folder, each a folder that holds a .git folder or file, written from
// the folder, . for itself, in the order the walk finds them: folders by name, each before what it holds. It goes into
// a clone, for the worktrees it holds, but never into .git, a hidden folder, or node_modules, nor more than depth
// folders down; it stops at the most clones it finds, and says whether it stopped there.
func findClones(root string, depth, most int) (clones []string, capped bool) {
	_ = filepath.WalkDir(root, func(path string, e fs.DirEntry, err error) error {
		if err != nil || !e.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if rel != "." && (strings.HasPrefix(e.Name(), ".") || e.Name() == "node_modules") {
			return fs.SkipDir
		}
		if _, err := os.Lstat(filepath.Join(path, ".git")); err == nil {
			if clones = append(clones, rel); len(clones) >= most {
				capped = true
				return fs.SkipAll
			}
		}
		if rel != "." && strings.Count(rel, string(filepath.Separator))+1 >= depth {
			return fs.SkipDir
		}
		return nil
	})
	return clones, capped
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

// above says whether there is a folder above what is typed: anywhere but /.
func (d OpenDialog) above() bool { return filepath.Clean(d.expand(d.Typed)) != "/" }

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
// line to move to, page up and page down while it has more than it shows, going up while there is a folder above, and
// finding clones while a folder is typed; esc goes back from the clones.
func (d OpenDialog) Keys() Segment {
	s := Segment{Owns: true, Bindings: []Binding{Bind("esc", "close")}}
	if d.finding || d.found != nil {
		s.Bindings[0].Does = "back"
	}
	list, head := d.list()
	if len(list) > 0 {
		s = s.With(Bind("↑↓", "move", "up", "down"))
	}
	if len(list)-head > openRows {
		s = s.With(Bind("⇞⇟", "page", "pgup", "pgdown"))
	}
	s = s.With(Bind("⇥", "complete", "tab"))
	if d.above() {
		s = s.With(Bind("⇧⌫", "up", "shift+backspace"), Bind("⌃W", "up", "ctrl+w"))
	}
	if d.Path() != "" {
		s = s.With(Bind("⌃F", "find clones", "ctrl+f"))
	}
	return s.With(Bind("⏎", "open", "enter"))
}

// openLine is one line of the dialog's body, and what a click on it does: completes to a folder, asks to open a clone,
// finds clones, or opens what is typed; and whether it is highlighted.
type openLine struct {
	text   string
	folder string
	repo   string
	finds  bool
	opens  bool
	lit    bool
}

// list gives the lines of the dialog's list, and how many of them it pins atop it, never scrolled: ../ while there is a
// folder above what is typed, which a click on it goes up to; then each folder what is typed can complete to, which a
// click on it types. Over a walk for clones, it has the clones found, which a click on one asks to open; none while the
// walk is on.
func (d OpenDialog) list() (lines []openLine, head int) {
	switch {
	case d.finding:
		return nil, 0
	case d.found != nil:
		for _, c := range d.found.Clones {
			lines = append(lines, openLine{text: Clone + " " + c, repo: filepath.Join(d.under, c)})
		}
		return lines, 0
	}
	if d.above() {
		lines, head = append(lines, openLine{text: Folder + " ../", folder: d.up()}), 1
	}
	folder, _ := d.parts()
	for _, n := range d.Folders() {
		lines = append(lines, openLine{text: Folder + " " + n + "/", folder: folder + n + "/"})
	}
	return lines, head
}

// body gives the dialog's lines under its top edge: what is typed, a cursor after it; kit's line, or what the dialog
// takes, or which clones stand under it; the lines its list pins, then those it shows from its first shown, the one
// highlighted among them, and how many it hides above and below, or the line that says the walk is on; the line that
// finds clones, and Open, each dim while nothing is typed or kit is asked, and the first while a walk is on.
func (d OpenDialog) body() []openLine {
	lines := []openLine{{text: "› " + d.Typed + "▏"}}
	switch {
	case d.Refusal != "":
		lines = append(lines, openLine{text: style.Refusal(Failed + " " + d.Refusal)})
	case d.finding || d.found != nil:
		lines = append(lines, openLine{text: style.Off("the clones under " + tilde(d.under, d.Home) +
			"; esc goes back to the folders")})
	default:
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
	switch {
	case d.finding:
		lines = append(lines, openLine{text: style.Off("finding clones…")})
	case d.found != nil && len(list) == 0:
		lines = append(lines, openLine{text: style.Off("no clone under it")})
	case d.found != nil && d.found.Capped:
		lines = append(lines, openLine{text: style.Off("stopped at the first " + strconv.Itoa(len(list)) + " clones")})
	case len(list) == head && d.found == nil:
		lines = append(lines, openLine{text: style.Off("no folder here starts so")})
	}
	find, open := Finding+" Find clones under "+d.Typed, Opening+" Open"
	if d.Busy || d.Path() == "" {
		find, open = style.Off(find), style.Off(open)
	} else if d.finding {
		find = style.Off(find)
	}
	return append(lines, openLine{}, openLine{text: find, finds: true}, openLine{text: open, opens: true})
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
	case k >= 0 && k < len(body) && (body[k].folder != "" || body[k].repo != ""):
		return d.take(body[k])
	case k >= 0 && k < len(body) && body[k].finds:
		return d.find(), ""
	case k >= 0 && k < len(body) && body[k].opens:
		return d.ask()
	}
	return d, ""
}
