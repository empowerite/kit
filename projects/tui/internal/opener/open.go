// Package opener is the Open Repository… dialog, which the workspace menu's Open Repository… opens over the screen
// shown: the folder of a repository typed, as a terminal's prompt takes one, ~ for the home folder and a path without /
// read from there; tab completes it a folder at a time, and the folders it can complete to stand under it, each opened
// into by a plain click. Up and down move a highlight from what is typed onto the folders and back, page up and page
// down by a page, and the list scrolls to keep it in view, as the mouse wheel scrolls it, saying how many folders it
// hides above and below; enter on a folder highlighted goes into it, as a click does, and typing puts the highlight
// back on what is typed. Shift+backspace and ⌃W go up a folder, taking the last name off what is typed; while there is
// a folder above, a ../ line starts the list, pinned as it scrolls, and a click on it, or enter on it, goes up the same
// way. Enter with no folder highlighted, or a click on Open, asks kittui to open what is typed; kit says whether it is
// a git working tree, and its line, when it is not, stands in the dialog, which stays open.  A line above Open, or ⌃F,
// finds the clones under the folder typed: a command walks it for git working trees, never blocking the screen, while a
// dim line says it is finding them where the folders were. The clones then stand in their place as a table, as
// clones.go says: a tree of repositories, each with its worktrees, selected and added to a workspace, or taken out of
// the one shown, or opened. Esc, or typing, goes back to the folders, and the answer of a walk begun before is dropped.
// Esc over the folders, or a click off the box, closes the dialog.
package opener

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// The dialog's glyphs: a folder it can complete to, its line that opens what is typed, its line that finds the clones
// under it, a clone found, and the lines that add the clones to a workspace and take them out of the one shown. Each
// is two cells wide, as every glyph in a menu is.
const (
	Folder   = "📁"
	Opening  = shell.Opening
	Finding  = "🔍"
	Clone    = "📦"
	Adding   = "➕"
	Removing = "➖"
	Group    = "📚"
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

// Dialog is the Open Repository… dialog: whether it is open, which then has the keyboard; the folder typed, as written;
// kit's line for the last thing it was asked, when it refused; what kit said it did, when it did; whether kit is being
// asked; the home folder, which ~ stands for and a path without / is read from; the line highlighted, 0 for what is
// typed and k for the kth line of the list; and the first line of the list shown. A walk for clones holds whether one
// is on; the folder walked; how many walks were begun, which tells an answer from one begun before; and the walk's
// answer, whose clones stand in place of the folders as a table, nil while they do not. The workspace shown says which
// clones are its members, and the groups they can be added to; the dropdown of those groups hangs from the line that
// adds.
type Dialog struct {
	Open    bool
	Typed   string
	Refusal string
	Did     string
	Busy    bool
	Home    string
	At      int
	Top     int
	finding bool
	under   string
	walks   int
	found   *ClonesMsg
	table   *clones
	ws      kit.Workspace
	adding  shell.Dropdown
}

// Ask is what the dialog asks kittui to do: open the folder of a repository, or run kit with the words, as kit
// workspaces --add=<group> <folder>… is; nothing when both are empty.
type Ask struct {
	Open  string
	Words []string
}

// ClonesMsg is a walk's answer: the walk, by its count; the folder walked; the clones under it, each from it; the
// desk of each, the repository's own working tree a linked worktree belongs to, the clone itself for a desk; and
// whether the walk stopped at the most it finds.
type ClonesMsg struct {
	Walk   int
	Under  string
	Clones []string
	Desks  []string
	Capped bool
}

// New gives the dialog open over the workspace shown, the folder start typed, with a / after it, ~ in place of home;
// with no start, ~/.
func New(start, home string, ws kit.Workspace) Dialog {
	d := Dialog{Open: true, Home: home, Typed: "~/", ws: ws}
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
func (d Dialog) expand(written string) string {
	switch {
	case written == "~" || strings.HasPrefix(written, "~/"):
		return d.Home + written[1:]
	case strings.HasPrefix(written, "/"):
		return written
	}
	return filepath.Join(d.Home, written)
}

// Path gives the folder typed, absolute and clean; nothing when nothing is typed.
func (d Dialog) Path() string {
	if strings.TrimSpace(d.Typed) == "" {
		return ""
	}
	return filepath.Clean(d.expand(d.Typed))
}

// parts gives what is typed split at its last /: the folder it stands in, as written with its /, and the start of the
// name typed after it.
func (d Dialog) parts() (folder, start string) {
	k := strings.LastIndex(d.Typed, "/")
	return d.Typed[:k+1], d.Typed[k+1:]
}

// Folders gives the names of the folders what is typed can complete to, in order: those in the folder it stands in
// whose names start with what is typed after its last /, a hidden one only once a . is typed.
func (d Dialog) Folders() []string {
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
func (d Dialog) complete() string {
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

// ask gives the dialog asking to open the folder typed, and the ask; nothing while kit is asked, or with nothing typed.
func (d Dialog) ask() (Dialog, Ask) { return d.askFor(d.Path()) }

// askFor gives the dialog asking to open the folder given, and the ask; nothing while kit is asked, or with none.
func (d Dialog) askFor(path string) (Dialog, Ask) {
	if d.Busy || path == "" {
		return d, Ask{}
	}
	d.Busy, d.Refusal, d.Did = true, "", ""
	return d, Ask{Open: path}
}

// run gives the dialog asking kit to run the words, and the ask; nothing while kit is asked.
func (d Dialog) run(words ...string) (Dialog, Ask) {
	if d.Busy {
		return d, Ask{}
	}
	d.Busy, d.Refusal, d.Did = true, "", ""
	return d, Ask{Words: words}
}

// Refused gives the dialog told kit's line for what it asked: open still, the line under what is typed.
func (d Dialog) Refused(line string) Dialog {
	d.Busy, d.Refusal = false, line
	return d
}

// Done gives the dialog told kit did what it asked it to run, and kit's line for it, which stands under what is typed;
// the clones selected are no longer.
func (d Dialog) Done(line string) Dialog {
	d.Busy, d.Did = false, line
	if d.table != nil {
		clear(d.table.Table().Picked)
	}
	return d
}

// Told gives the dialog the workspace shown, as kit says it anew: the clones its members are marked so.
func (d Dialog) Told(ws kit.Workspace) Dialog {
	d.ws = ws
	if d.table != nil {
		d.table.mark(ws)
	}
	return d
}

// Key takes a key while the dialog is open: a printable key types, backspace takes one back, ⌃U clears, tab completes,
// and shift+backspace or ⌃W goes up a folder, each putting the highlight back on what is typed and the folders back in
// place of the clones; up and down move the highlight a line, page up and page down a page; enter takes the line
// highlighted, as a click on it does, or with none asks to open what is typed; ⌃F finds the clones under the folder
// typed; and esc closes the dialog. Over the clones, the table takes the keys first, as clones.go says. It gives the
// dialog and what it asks.
func (d Dialog) Key(k tea.KeyPressMsg) (Dialog, Ask) {
	if d.found != nil {
		if next, ask, took := d.cloneKey(k); took {
			return next, ask
		}
	}
	switch key := k.String(); key {
	case "esc":
		if d.finding || d.found != nil {
			return d.typing(d.Typed), Ask{}
		}
		d.Open = false
		return d, Ask{}
	case "up", "down", "pgup", "pgdown":
		return d.move(key), Ask{}
	case "shift+backspace", "ctrl+w":
		return d.typing(d.up()), Ask{}
	case "ctrl+f":
		return d.find(), Ask{}
	case "enter":
		if list, _ := d.list(); d.At > 0 && d.At <= len(list) {
			return d.take(list[d.At-1])
		}
		return d.ask()
	case "tab":
		return d.typing(d.complete()), Ask{}
	case "ctrl+u":
		return d.typing(""), Ask{}
	}
	if typed, ok := shell.Type(d.Typed, k); ok {
		return d.typing(typed), Ask{}
	}
	return d, Ask{}
}

// typing gives the dialog with typed in place of what is typed: kit's lines gone, the highlight back on what is typed,
// the folders in place of the clones, a walk on left to be dropped, and the list shown from its top.
func (d Dialog) typing(typed string) Dialog {
	d.Typed, d.Refusal, d.Did, d.At, d.Top = typed, "", "", 0, 0
	d.finding, d.found, d.table, d.adding = false, nil, nil, shell.Dropdown{}
	return d
}

// take gives the dialog a line of its list taken, by a click or enter: a folder typed.
func (d Dialog) take(l openLine) (Dialog, Ask) { return d.typing(l.folder), Ask{} }

// find gives the dialog walking the folder typed for clones, its list given over to them; as it was while nothing is
// typed, kit is asked, or a walk is on.
func (d Dialog) find() Dialog {
	if d.Path() == "" || d.Busy || d.finding {
		return d
	}
	d = d.typing(d.Typed)
	d.finding, d.under, d.walks = true, d.Path(), d.walks+1
	return d
}

// Walk gives the command that walks the folder for clones when the dialog began a walk since it was was, as a key or a
// click begins one; nil when it began none. The command answers with a ClonesMsg.
func (d Dialog) Walk(was Dialog) tea.Cmd {
	if !d.finding || d.walks == was.walks {
		return nil
	}
	walk, under := d.walks, d.under
	return func() tea.Msg {
		clones, desks, capped := findClones(under, cloneDepth, cloneMost)
		return ClonesMsg{Walk: walk, Under: under, Clones: clones, Desks: desks, Capped: capped}
	}
}

// Found gives the dialog a walk's answer: the clones in place of the folders, as a table whose rows have the keyboard,
// when it answers the walk on; else the dialog as it was, as for a walk begun before, or one left as the folder typed
// changed.
func (d Dialog) Found(msg ClonesMsg) Dialog {
	if !d.finding || msg.Walk != d.walks || msg.Under != d.Path() {
		return d
	}
	d.finding, d.found = false, &msg
	d.table = newClones(msg, d.Home, d.ws)
	return d
}

// findClones gives the git working trees under a folder, each a folder that holds a .git folder or file, written from
// the folder, . for itself, in the order the walk finds them: folders by name, each before what it holds; and the desk
// of each, absolute, as desk says. It goes into a clone, for the worktrees it holds, but never into .git, a hidden
// folder, or node_modules, nor more than depth folders down; it stops at the most clones it finds, and says whether it
// stopped there.
func findClones(root string, depth, most int) (clones, desks []string, capped bool) {
	_ = filepath.WalkDir(root, func(path string, e fs.DirEntry, err error) error {
		if err != nil || !e.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if rel != "." && (strings.HasPrefix(e.Name(), ".") || e.Name() == "node_modules") {
			return fs.SkipDir
		}
		if git, err := os.Lstat(filepath.Join(path, ".git")); err == nil {
			clones, desks = append(clones, rel), append(desks, desk(path, git))
			if len(clones) >= most {
				capped = true
				return fs.SkipAll
			}
		}
		if rel != "." && strings.Count(rel, string(filepath.Separator))+1 >= depth {
			return fs.SkipDir
		}
		return nil
	})
	return clones, desks, capped
}

// desk gives the repository's own working tree a working tree belongs to, from its .git: a linked worktree's .git is a
// file whose gitdir: names <desk>/.git/worktrees/<name>, from the worktree where it is not absolute; any other is its
// own desk.
func desk(path string, git fs.FileInfo) string {
	if git.IsDir() {
		return path
	}
	data, err := os.ReadFile(filepath.Join(path, ".git"))
	if err != nil {
		return path
	}
	dir := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(data)), "gitdir:"))
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(path, dir)
	}
	if k := strings.LastIndex(filepath.Clean(dir), string(filepath.Separator)+".git"+string(filepath.Separator)); k > 0 {
		return filepath.Clean(dir)[:k]
	}
	return path
}

// up gives what is typed a folder up: the folder it names without its last name, with a / after it and ~ in place of
// home; / stays /, and the folder above home is written whole.
func (d Dialog) up() string {
	dir := filepath.Dir(filepath.Clean(d.expand(d.Typed)))
	if dir == "/" {
		return dir
	}
	return tilde(dir, d.Home) + "/"
}

// above says whether there is a folder above what is typed: anywhere but /.
func (d Dialog) above() bool { return filepath.Clean(d.expand(d.Typed)) != "/" }

// move gives the dialog its highlight moved by a key, between what is typed and the list's last line: up and down a
// line, page down a page, and page up a page, stopping at the list's first line; the list scrolled to keep it in view,
// and to its top on the line it pins.
func (d Dialog) move(key string) Dialog {
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

// Wheel takes a turn of the mouse wheel while the dialog is open, over a screen of so many lines, so wide: over the
// clones, the table scrolls, as clones.go says; else it scrolls the list of folders a few lines, never past its ends,
// the highlight staying where it is.
func (d Dialog) Wheel(w tea.MouseWheelMsg, height, width int) Dialog {
	if d.found != nil {
		p := d.placed(height, width)
		w.X -= p.Left + 2
		d.table.Wheel(w)
		return d
	}
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

// Keys gives the dialog's segment of the keys bar: it takes every press. Over the clones, they are the table's, as
// clones.go says; over the folders, it names up and down while the list has a line to move to, page up and page down
// while it has more than it shows, going up while there is a folder above, and finding clones while a folder is typed.
func (d Dialog) Keys() shell.Segment {
	if d.found != nil {
		return d.cloneKeys()
	}
	s := shell.Segment{Owns: true, Bindings: []shell.Binding{shell.Bind("esc", "close")}}
	if d.finding {
		s.Bindings[0].Does = "back"
	}
	list, head := d.list()
	if len(list) > 0 {
		s = s.With(shell.Bind("↑↓", "move", "up", "down"))
	}
	if len(list)-head > openRows {
		s = s.With(shell.Bind("⇞⇟", "page", "pgup", "pgdown"))
	}
	s = s.With(shell.Bind("⇥", "complete", "tab"))
	if d.above() {
		s = s.With(shell.Bind("⇧⌫", "up", "shift+backspace"), shell.Bind("⌃W", "up", "ctrl+w"))
	}
	if d.Path() != "" {
		s = s.With(shell.Bind("⌃F", "find clones", "ctrl+f"))
	}
	return s.With(shell.Bind("⏎", "open", "enter"))
}

// openLine is one line of the dialog's body, and what a click on it does: completes to a folder, finds clones, opens
// what is typed, is a line of the clones' table, adds the clones to a workspace, takes them out of the one shown, or
// opens the clone under the cursor; and whether it is highlighted.
type openLine struct {
	text    string
	folder  string
	finds   bool
	opens   bool
	sheet   bool
	adds    bool
	removes bool
	opensAt bool
	lit     bool
}

// list gives the lines of the dialog's list of folders, and how many of them it pins atop it, never scrolled: ../
// while there is a folder above what is typed, which a click on it goes up to; then each folder what is typed can
// complete to, which a click on it types; none while a walk is on, or over its clones.
func (d Dialog) list() (lines []openLine, head int) {
	if d.finding || d.found != nil {
		return nil, 0
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

// head gives the dialog's first lines: what is typed, a cursor after it; kit's line, what kit did, or what the
// dialog takes, or which clones stand under it; and a blank line.
func (d Dialog) head() []openLine {
	lines := []openLine{{text: "› " + d.Typed + "▏"}}
	switch {
	case d.Refusal != "":
		lines = append(lines, openLine{text: style.Refusal(Failed + " " + d.Refusal)})
	case d.Did != "":
		lines = append(lines, openLine{text: d.Did})
	case d.finding || d.found != nil:
		lines = append(lines, openLine{text: style.Off("the clones under " + tilde(d.under, d.Home) +
			"; esc goes back to the folders")})
	default:
		lines = append(lines, openLine{text: style.Off("the folder of a git clone; ⇥ completes, a click opens a folder")})
	}
	return append(lines, openLine{})
}

// body gives the dialog's lines under its top edge, so wide inside, over the clones as clones.go says: its head; the
// lines its list pins, then those it shows from its first shown, the one highlighted among them, and how many it hides
// above and below, or the line that says the walk is on; the line that finds clones, and Open, each dim while nothing
// is typed or kit is asked, and the first while a walk is on.
func (d Dialog) body(inner int) []openLine {
	if d.found != nil {
		return d.cloneBody(inner)
	}
	lines := d.head()
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
	case len(list) == head:
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

// boxed gives the dialog's box, its lines drawn under its top edge, and the lines; the box's top line and left column
// over a screen of so many lines, so wide: its third line, or as near as the screen lets, and centered across it,
// wide enough for a long path, and over the clones, wide enough for their table; the line highlighted drawn across the
// box, as a menu draws the line under the keyboard.
func (d Dialog) placed(height, width int) shell.Placed {
	inner := min(60, max(0, width-6))
	if d.found != nil {
		inner = d.tableWidth(width)
		d.table.fit(d, height, width)
	}
	body := d.body(inner)
	var texts []string
	for _, l := range body {
		texts = append(texts, l.text)
	}
	texts[0] = ansi.Truncate(texts[0], width-6, "…")
	// a line of spaces, taken out again, keeps the box wide enough for a long path as it is typed
	box := shell.Dialog("Open a repository", append(texts, strings.Repeat(" ", inner)), width-2)
	box = append(box[:len(box)-2], box[len(box)-1])
	in := ansi.StringWidth(box[0]) - 2
	for k, l := range body {
		if l.lit {
			box[k+1] = style.Menu("│") + style.Selected(" "+pane.Pad(ansi.Truncate(l.text, in-2, "…"), in-2,
				false)+" ") + style.Menu("│")
		}
	}
	return shell.Placed{Lines: box, Top: boxTop(height, len(box)), Left: max(0, (width-ansi.StringWidth(box[0]))/2)}
}

// boxTop gives the screen line a box so tall stands on over a screen so high: its third, or as near as the screen lets.
func boxTop(height, lines int) int { return min(2, max(0, height-lines)) }

// Overlay draws the open dialog over the screen's lines, so wide, and over the clones the table's open layers.
func (d Dialog) Overlay(lines []string, width int) []string {
	if !d.Open {
		return lines
	}
	p := d.placed(len(lines), width)
	lines = p.Draw(lines)
	if d.found != nil {
		lines = d.cloneOver(lines, p, width)
	}
	return lines
}

// Click takes a click while the dialog is open, over a screen of so many lines, so wide: a left one on a folder
// completes to it, one on Open asks to open what is typed, as enter does, and one off the box closes the dialog; over
// the clones, as clones.go says; anything else changes nothing. It gives the dialog and what it asks.
func (d Dialog) Click(msg tea.MouseClickMsg, height, width int) (Dialog, Ask) {
	mouse := msg.Mouse()
	if mouse.Button != tea.MouseLeft {
		return d, Ask{}
	}
	p := d.placed(height, width)
	if d.found != nil {
		if next, ask, took := d.cloneClick(msg, p, height, width); took {
			return next, ask
		}
	}
	k, on := p.Hit(mouse.X, mouse.Y)
	body := d.body(ansi.StringWidth(p.Lines[0]) - 4)
	switch {
	case !on:
		d.Open = false
	case k >= 0 && k < len(body) && body[k].folder != "":
		return d.take(body[k])
	case k >= 0 && k < len(body) && body[k].finds:
		return d.find(), Ask{}
	case k >= 0 && k < len(body) && body[k].opens:
		return d.ask()
	}
	return d, Ask{}
}

// count writes how many, and what one is, or many are.
func count(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}
