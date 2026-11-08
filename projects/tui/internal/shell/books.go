// The 📚 control is the workspace on every screen's title line, after the screen's label: 📚 owner/repo for one
// repository, 📚 and a group's name and its count of members, the count dim, for a group, both in the repository's
// color, and 📚 alone, dim, with none. Until kit says which workspace it is, it names the origin's owner/repo, as the
// title did before. A plain click on it drops down what to switch to: the groups the repository belongs to, first;
// the favorites; and Browse…, New workspace…, and Manage workspaces…. A line that cannot switch now stays, dim: the
// workspace shown, a repository not cloned, and the three that open what is still to come. When kit could not say, its
// line comes first, dim. A pick asks kittui to switch, by the workspace's name as kit names it.
package shell

import (
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Books is the workspace's glyph, on the title line and beside each group in its dropdown.
const Books = "📚"

// The glyphs of the dropdown's lines besides the groups': a favorite, a repository not cloned, kit's line, and the
// three that open what is still to come.
const (
	Favorite = "⭐"
	Cloud    = "☁️"
	browse   = "🔎"
	create   = "➕"
	manage   = "🧰"
)

// WorkspaceMsg tells a screen its workspace, when it is made and whenever kit says it anew; when kit could not say,
// kit's line, and the workspace as it was.
type WorkspaceMsg struct {
	Workspace kit.Workspace
	Refusal   string
}

// PickMsg asks kittui to switch to a workspace: a group's name or a repository's host/path, as kit names it.
type PickMsg struct{ Workspace string }

// Pick gives the command that asks kittui to switch to the workspace.
func Pick(workspace string) tea.Cmd { return func() tea.Msg { return PickMsg{workspace} } }

// BooksMenu is the 📚 control on a screen's title line: the workspace it names, the origin's owner/repo it names
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
		return Books + " " + style.Repo(ws.Name) + " " + style.Off(strconv.Itoa(len(ws.Members)))
	case ws.Kind == "repository":
		return Books + " " + style.Repo(repoName(ws.Member, ws.Checkout))
	case ws.Kind == "" && b.Fallback != "":
		return Books + " " + style.Repo(b.Fallback)
	}
	return style.Off(Books)
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

// BooksAt gives the column the 📚 starts at on a title line drawn as the title given, then " · ", then the control.
func BooksAt(title string) int { return ansi.StringWidth(title) + 3 }

// On says whether a click at x, y is on the control drawn from the column at on the title line.
func (b BooksMenu) On(x, y, at int) bool {
	return y == Top && x >= at && x < at+ansi.StringWidth(b.Label())
}

// lines gives the dropdown's lines, top to bottom, as the header says.
func (b BooksMenu) lines() []booksLine {
	ws := b.Workspace
	var out []booksLine
	rule := booksLine{}
	if b.Refusal != "" {
		out = append(out, booksLine{item: Item{Glyph: "❌", Name: b.Refusal}}, rule)
	}
	if ws.Kind == "repository" && len(ws.Groups) > 0 {
		for _, g := range ws.Groups {
			out = append(out, booksLine{Item{Glyph: Books, Name: g.Name + " " + strconv.Itoa(g.Members)}, g.Name, true})
		}
		out = append(out, rule)
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
				line.item.Name += " " + Cloud
				line.acts = false
			}
		}
		line.acts = line.acts && !shown
		out = append(out, line)
	}
	if len(ws.Favorites) > 0 {
		out = append(out, rule)
	}
	return append(out, booksLine{item: Item{Glyph: browse, Name: "Browse…"}},
		booksLine{item: Item{Glyph: create, Name: "New workspace…"}},
		booksLine{item: Item{Glyph: manage, Name: "Manage workspaces…"}})
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
	b.Dropdown, at = b.Dropdown.Key(k, b.Items(), b.enabled)
	return b, b.target(at)
}

// Click takes a left click while the dropdown is open, hanging from the 📚 at the column at, the screen's rows as wide
// as room, as a dropdown does; it gives the control and the workspace picked, "" for none.
func (b BooksMenu) Click(x, y, at, room int) (BooksMenu, string) {
	box, left := b.box(at, room)
	var k int
	b.Dropdown, k = b.Dropdown.Click(x, y, Top, left, ansi.StringWidth(box[0]), b.Items(), b.enabled)
	return b, b.target(k)
}

// Overlay draws the open dropdown over the screen's lines, hanging from the 📚 at the column at, the rows as wide as
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
	return Box(b.Items(), b.enabled, b.At, Books, at, room)
}
