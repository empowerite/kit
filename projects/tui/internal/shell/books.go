// The workspace menu is the workspace on every screen's title line, after the screen's label: - owner/repo for one
// repository, - and a group's name and its count of members, the count dim, for a group, both in the repository's
// color, and - alone, dim, with none. Until kit says which workspace it is, it names the origin's owner/repo, as the
// title did before. A plain click on it drops down what to switch to: the groups the repository belongs to, first;
// the favorites, however many; every other group in the file; the repositories opened by their folder, the newest
// first; and last Open Repository…, which opens a repository by its folder, and Manage…, which opens the workspaces'
// screen; a rule between two of those it has. A line that cannot
// switch now stays, dim: the workspace shown, and a repository not cloned. The repository shown always has its line,
// dim, first among the recent ones, unless a favorite draws it already, and one that reads as another of them, two
// folders of one repository, says its folder, dim. When kit could not say, its line comes first, dim. A pick asks kittui to switch, by the workspace's name as kit names it, a recent repository by its
// folder; Open Repository… asks for the Open Repository… dialog, and Manage… for the workspaces' screen.
package shell

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Workspaces is the workspace menu's glyph: on the title line, before the workspace it names, and the glyph its
// dropdown hangs from; -, the key that drops it down, as = is the app menu's glyph and its key.
const Workspaces = "-"

// WorkspaceScreenKeys gives a screen's segment, as ScreenKeys does, on a screen with a workspace menu: - drops it down,
// named after =, in the title's order.
func WorkspaceScreenKeys(own ...Binding) Segment {
	s := ScreenKeys(own...)
	s.Bindings = slices.Insert(s.Bindings, 1, Bind(Workspaces, "workspace"))
	return s
}

// Books is a group's glyph, beside its line in the dropdown: two cells wide, as every glyph in a menu is.
const Books = "📚"

// Favorite is a favorite's glyph, beside its line in the dropdown.
const Favorite = "⭐"

// Opened is the glyph of a repository opened by its folder, beside its line in the dropdown.
const Opened = "🕘"

// openItem is the dropdown's last line, which opens the Open Repository… dialog, to open a repository by its folder.
var openItem = booksLine{item: Item{Glyph: Opening, Name: "Open Repository…"}, target: openTarget, acts: true}

// manageItem is the line after Open Repository…, which opens the workspaces' screen, where the groups are made,
// renamed, edited, and deleted.
var manageItem = booksLine{item: Item{Glyph: Index, Name: "Manage…"}, target: manageTarget, acts: true}

// Index is Manage…'s glyph.
const Index = "📇"

// openTarget and manageTarget are what Open Repository… and Manage… pick: no workspace's name, as neither a group's nor
// a repository's starts with a space.
const (
	openTarget   = " open"
	manageTarget = " manage"
)

// ChangedMsg says a screen had kit change the groups: kittui asks kit for the workspace again, so every screen's menu
// lists them as they are now; a group renamed From To, or deleted, To empty, is followed, when it is the one shown.
type ChangedMsg struct{ From, To string }

// WorkspaceMsg tells a screen its workspace, when it is made and whenever kit says it anew; when kit could not say,
// kit's line, and the workspace as it was.
type WorkspaceMsg struct {
	Workspace kit.Workspace
	Refusal   string
}

// PickMsg asks kittui to switch to a workspace: a group's name or a repository's host/path, as kit names it.
type PickMsg struct{ Workspace string }

// Pick gives the command for what a line of the dropdown picked: Open Repository… asks kittui for the Open Repository…
// dialog, Manage… for the workspaces' screen, leaving the screen shown as its crumb, and a workspace asks kittui to
// switch to it; none for none.
func Pick(workspace string) tea.Cmd {
	switch workspace {
	case "":
		return nil
	case openTarget:
		return func() tea.Msg { return OpenMsg{} }
	case manageTarget:
		return func() tea.Msg { return SwitchMsg{Screen: Manage, Crumb: true} }
	}
	return func() tea.Msg { return PickMsg{workspace} }
}

// BooksMenu is the workspace menu on a screen's title line: the workspace it names, the origin's owner/repo it names
// until kit says, kit's line when kit could not, and its dropdown.
type BooksMenu struct {
	Workspace kit.Workspace
	Fallback  string
	Refusal   string
	Dropdown
}

// booksLine is one line of the dropdown: the line, the workspace it picks, and whether it picks it now.
type booksLine struct {
	item   Item
	target string
	acts   bool
}

// Told gives the control what kit said: the workspace, or, when kit could not say, its line, the workspace as it was.
func (b BooksMenu) Told(msg WorkspaceMsg) BooksMenu {
	b.Refusal = msg.Refusal
	if msg.Refusal == "" {
		b.Workspace = msg.Workspace
	}
	return b
}

// Label gives what the title line draws after the screen's label, as the header says.
func (b BooksMenu) Label() string {
	ws := b.Workspace
	switch {
	case ws.Kind == "group":
		return Workspaces + " " + style.Repo(ws.Name) + " " + style.Off(strconv.Itoa(len(ws.Members)))
	case ws.Kind == "repository":
		return Workspaces + " " + style.Repo(repoName(ws.Member, ws.Checkout))
	case ws.Kind == "" && b.Fallback != "":
		return Workspaces + " " + style.Repo(b.Fallback)
	}
	return style.Off(Workspaces)
}

// repoName gives a repository's short name: its member's owner/repo, the last two segments of host/path; with no
// member, its checkout's directory's name.
func repoName(member, checkout string) string {
	if member == "" {
		return filepath.Base(checkout)
	}
	segments := strings.Split(member, "/")
	return strings.Join(segments[max(1, len(segments)-2):], "/")
}

// Between stands between the title line's parts, the screen's name, the workspace menu, and a maximized tab: a dot, dim,
// a space either side, so the menus' glyphs beside it stand out as what is clicked.
var Between = " " + style.Off("·") + " "

// BooksAt gives the column the - starts at on a title line drawn as the title given, then Between, then the control.
func BooksAt(title string) int { return ansi.StringWidth(title + Between) }

// On says whether a click at x, y is on the control drawn from the column at on the title line.
func (b BooksMenu) On(x, y, at int) bool {
	return y == Top && x >= at && x < at+ansi.StringWidth(b.Label())
}

// lines gives the dropdown's lines, top to bottom, as the header says.
func (b BooksMenu) lines() []booksLine {
	ws := b.Workspace
	var refusal, groups, favorites, others, recent []booksLine
	if b.Refusal != "" {
		refusal = append(refusal, booksLine{item: Item{Glyph: "❌", Name: b.Refusal}})
	}
	if ws.Kind == "repository" {
		for _, g := range ws.Groups {
			line := Item{Glyph: Books, Name: g.Name + " " + strconv.Itoa(g.Members)}
			groups = append(groups, booksLine{line, g.Name, true})
		}
	}
	for _, f := range ws.Favorites {
		line := booksLine{target: f.Name, acts: true}
		shown := (ws.Kind == "group" && f.Kind == "group" && f.Name == ws.Name) ||
			(ws.Kind == "repository" && f.Kind == "repository" && f.Name == ws.Member)
		if f.Kind == "group" {
			line.item = Item{Glyph: Favorite, Name: f.Name + " " + strconv.Itoa(f.Members)}
		} else {
			line.item = Item{Glyph: Favorite, Name: repoName(f.Name, "")}
			if f.Checkout == "" {
				line.item.Name += " · not cloned"
				line.acts = false
			}
		}
		line.acts = line.acts && !shown
		favorites = append(favorites, line)
	}
	drawn := map[string]bool{}
	for _, l := range slices.Concat(groups, favorites) {
		drawn[l.target] = true
	}
	for _, g := range ws.Workspaces {
		if drawn[g.Name] {
			continue
		}
		line := Item{Glyph: Books, Name: g.Name + " " + strconv.Itoa(g.Members)}
		others = append(others, booksLine{line, g.Name, ws.Kind != "group" || g.Name != ws.Name})
	}
	recent = recents(ws, drawn)
	var out []booksLine
	for _, section := range [][]booksLine{refusal, groups, favorites, others, recent, {openItem, manageItem}} {
		if len(section) == 0 {
			continue
		}
		if len(out) > 0 {
			out = append(out, booksLine{})
		}
		out = append(out, section...)
	}
	return out
}

// recents gives the lines of the recent repositories, each picked by its folder, but those drawn above already, by
// kit's name for them. The repository shown comes first, dim, as it cannot be switched to, unless drawn above already;
// a recent one that is the repository shown, by its folder or as the repository a worktree shown belongs to, since a
// worktree is no workspace of its own, is not drawn again. One that reads as another of them, as a clone and a worktree
// of one repository do, says its folder after its name, dim, to tell them apart.
func recents(ws kit.Workspace, drawn map[string]bool) []booksLine {
	var kept []kit.WorkspaceRecent
	reads := map[string]int{}
	shown := ws.Kind == "repository" && ws.Checkout != "" && !drawn[ws.Name]
	if shown {
		kept = append(kept, kit.WorkspaceRecent{Name: ws.Name, Member: ws.Member, Checkout: ws.Checkout})
		reads[repoName(ws.Member, ws.Checkout)]++
	}
	for _, r := range ws.Recent {
		if drawn[r.Name] || ws.Kind == "repository" &&
			(r.Checkout == ws.Checkout || r.Member != "" && r.Member == ws.Member) {
			continue
		}
		reads[repoName(r.Member, r.Checkout)]++
		kept = append(kept, r)
	}
	var out []booksLine
	for _, r := range kept {
		name := repoName(r.Member, r.Checkout)
		if reads[name] > 1 {
			name += " " + style.Off("· "+filepath.Base(r.Checkout))
		}
		out = append(out, booksLine{Item{Glyph: Opened, Name: name}, r.Checkout, !shown || len(out) > 0})
	}
	return out
}

// Items gives the dropdown's lines.
func (b BooksMenu) Items() []Item {
	var items []Item
	for _, l := range b.lines() {
		items = append(items, l.item)
	}
	return items
}

// enabled says whether the dropdown's line k picks a workspace now.
func (b BooksMenu) enabled(k int) bool {
	lines := b.lines()
	return k >= 0 && k < len(lines) && lines[k].acts
}

// target gives the workspace the line k picks, or nothing for none.
func (b BooksMenu) target(k int) string {
	if !b.enabled(k) {
		return ""
	}
	return b.lines()[k].target
}

// Opened gives the control dropped down, its first line that picks under the keyboard.
func (b BooksMenu) Opened() BooksMenu {
	b.Dropdown = b.Dropdown.Opened(b.Items(), b.enabled)
	return b
}

// Key takes a key while the dropdown is open, as a dropdown does; it gives the control and the workspace picked, ""
// for none.
func (b BooksMenu) Key(k tea.KeyPressMsg) (BooksMenu, string) {
	var at int
	b.Dropdown, at = b.Dropdown.Key(k, b.Items(), b.enabled, Workspaces)
	return b, b.target(at)
}

// Click takes a left click while the dropdown is open, hanging from the - at the column at, the screen's rows as
// wide as room, as a dropdown does; it gives the control and the workspace picked, "" for none.
func (b BooksMenu) Click(x, y, at, room int) (BooksMenu, string) {
	box, left := b.box(at, room)
	var k int
	b.Dropdown, k = b.Dropdown.Click(x, y, Top, left, ansi.StringWidth(box[0]), b.Items(), b.enabled)
	return b, b.target(k)
}

// Overlay draws the open dropdown over the screen's lines, hanging from the - at the column at, the rows as wide as
// room.
func (b BooksMenu) Overlay(lines []string, at, room int) []string {
	if !b.Open {
		return lines
	}
	box, left := b.box(at, room)
	return Over(lines, box, Top, left)
}

// box gives the open dropdown's lines and the column its left edge is drawn at.
func (b BooksMenu) box(at, room int) ([]string, int) {
	return Box(b.Items(), b.enabled, b.At, Workspaces, at, room)
}
