// The presets dialog: Manage… in the presets' menu opens it under the header's right end, titled by the table's
// presets. It lists every view of the table, a line each: its star, ★ where the person starred it, ☆ where not; its
// name; and where it comes from, built in or mine, dim. The favorites stand first, in their order, a dim rule under
// them, then the table's own, then the person's. Under them stand what can be done with the view under the keyboard:
// Rename… and Delete, on one of the person's own, which a table's own view never has. Up and down move between the
// lines; shift with either moves a favorite a place among the favorites; space or * stars the view, or no longer; r
// renames it, on the preset line, as Rename… does; d deletes it; and esc closes the dialog. A click on a line's star
// stars it, one on its name puts the keyboard there, and one on an action does it; one off the dialog closes it.
package table

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// ManageIt is the presets' menu's line that opens the presets dialog.
const ManageIt = "Manage…"

// NameCells is how many cells a preset's name takes on its line of the presets dialog, after its star and a space.
const NameCells = 22

// Manage is the presets dialog while it is open, and the line under the keyboard, an index into the views it lists.
type Manage struct {
	Open bool
	At   int
}

// ManageAct is what the presets dialog asks of the screen: nothing; to keep the favorites as given, starred, no longer,
// or in a new order; to rename a view, on the preset line; or to delete it.
type ManageAct int

// What the presets dialog asks.
const (
	ManageNothing ManageAct = iota
	ManageFavorites
	ManageRename
	ManageDelete
)

// ManageDo is what the presets dialog asks of the screen, and of which view, and the favorites to keep.
type ManageDo struct {
	Act       ManageAct
	Name      string
	Favorites []string
}

// Listed gives the views the presets dialog lists: the favorites, in their order, then the table's own, then the
// person's, each once.
func (d Manage) Listed(h Host) []string {
	v := h.Views()
	out := v.Favorites()
	for _, p := range v.All() {
		if !slices.Contains(out, p.Name) {
			out = append(out, p.Name)
		}
	}
	return out
}

// view gives the view under the keyboard, and whether there is one.
func (d Manage) view(h Host) (string, bool) {
	listed := d.Listed(h)
	if d.At < 0 || d.At >= len(listed) {
		return "", false
	}
	return listed[d.At], true
}

// isMine says whether a view is the person's own, which they can rename and delete.
func isMine(h Host, name string) bool {
	_, ok := h.Views().Mine(name)
	return ok
}

// actions gives what can be done with the view under the keyboard, as the dialog's last line names them: Rename… and
// Delete on one of the person's own; nothing on a table's own.
func (d Manage) actions(h Host) []string {
	if name, ok := d.view(h); ok && isMine(h, name) {
		return []string{Renamed + " " + RenameIt, Deleted + " " + DeleteIt}
	}
	return nil
}

// Line draws a view's line in the presets dialog: its star, in the accent where the person starred it, dim where not;
// its name; and where it comes from, dim.
func (d Manage) Line(h Host, name string) string {
	star, from := style.Off(Unstarred), BuiltinHeading
	if h.Views().Favorite(name) {
		star = style.Tint(Starred)
	}
	if isMine(h, name) {
		from = MineHeading
	}
	return star + " " + pane.Pad(name, NameCells, false) + style.Off(from)
}

// manageCells is how many cells a line of the presets dialog takes inside its edges.
var manageCells = 2 + NameCells + len(BuiltinHeading)

// Box gives the presets dialog's lines, titled by the table's presets: the headings of its cells, dim; a line per view,
// as Line draws it, a dim rule under the favorites, the line under the keyboard on the menu's color; then, after a dim
// rule, what can be done with the view under the keyboard, where anything can; and the screen line and column it is
// drawn from: under the header's right end, its right edge at the pane's.
func (d Manage) Box(h Host) (lines []string, top, left int) {
	body := []string{style.Off(Unstarred + " " + pane.Pad("preset", NameCells, false) + "from")}
	favorites := len(h.Views().Favorites())
	for k, name := range d.Listed(h) {
		if k == favorites && k > 0 {
			body = append(body, style.Off(strings.Repeat("─", manageCells)))
		}
		line := d.Line(h, name)
		if k == d.At {
			line = style.Selected(pane.Pad(ansi.Strip(line), manageCells, false))
		}
		body = append(body, line)
	}
	if acts := d.actions(h); len(acts) > 0 {
		body = append(body, style.Off(strings.Repeat("─", manageCells)), strings.Join(acts, "  "))
	}
	top, w := h.Pane()
	lines = shell.Dialog(h.Name()+" presets", body, w)
	return lines, top, max(0, w-ansi.StringWidth(lines[0]))
}

// Row gives the view a screen line of the presets dialog draws, as an index into those it lists, or -1 for its edges,
// its headings, its rules, and its actions.
func (d Manage) Row(h Host, y int) int {
	_, top, _ := d.Box(h)
	line, favorites := y-top-2, len(h.Views().Favorites()) // past the top edge and the headings
	for k := range d.Listed(h) {
		if k == favorites && k > 0 {
			if line == 0 {
				return -1
			}
			line-- // past the rule under the favorites
		}
		if line == 0 {
			return k
		}
		line--
	}
	return -1
}

// moved gives the favorites with the one under the keyboard a place up or down, by step, and whether it moved: a
// favorite moves among the favorites alone.
func (d Manage) moved(h Host, step int) ([]string, bool) {
	favorites := h.Views().Favorites()
	to := d.At + step
	if d.At >= len(favorites) || to < 0 || to >= len(favorites) {
		return nil, false
	}
	favorites[d.At], favorites[to] = favorites[to], favorites[d.At]
	return favorites, true
}

// act gives what a key asks of the screen for the view under the keyboard: space and * star it, or no longer; shift with
// up or down moves a favorite, the keyboard with it; r renames, and d deletes, one of the person's own, closing the
// dialog where it renames.
func (d *Manage) act(h Host, key string) ManageDo {
	name, ok := d.view(h)
	if !ok {
		return ManageDo{}
	}
	switch key {
	case "space", "*":
		return ManageDo{Act: ManageFavorites, Name: name, Favorites: h.Views().Star(name)}
	case "shift+up", "shift+down":
		step := map[string]int{"shift+up": -1, "shift+down": 1}[key]
		if favorites, ok := d.moved(h, step); ok {
			d.At += step
			return ManageDo{Act: ManageFavorites, Name: name, Favorites: favorites}
		}
	case "r":
		if isMine(h, name) {
			*d = Manage{}
			return ManageDo{Act: ManageRename, Name: name}
		}
	case "d":
		if isMine(h, name) {
			return ManageDo{Act: ManageDelete, Name: name}
		}
	}
	return ManageDo{}
}

// Key takes a key while the presets dialog is open: up and down move between its lines; esc closes it; the rest ask
// what act says.
func (d *Manage) Key(h Host, k tea.KeyPressMsg) ManageDo {
	switch key := k.String(); key {
	case "up", "down":
		d.At = shell.Step(d.At, len(d.Listed(h)), map[string]int{"up": -1, "down": 1}[key], shell.Every)
	case "esc":
		*d = Manage{}
	default:
		return d.act(h, key)
	}
	return ManageDo{}
}

// Click takes a click while the presets dialog is open: on a view's star, the keyboard goes there and it stars the
// view, or no longer; on its name, the keyboard goes there; on an action, it does it; on the dialog elsewhere, nothing;
// off it, the dialog closes.
func (d *Manage) Click(h Host, x, y int) ManageDo {
	box, top, left := d.Box(h)
	line, on := (shell.Placed{Lines: box, Top: top, Left: left}).Hit(x, y)
	switch k, cell := d.Row(h, y), x-left-2; {
	case !on:
		*d = Manage{}
	case k >= 0:
		d.At = k
		if cell < 2 {
			return d.act(h, "*")
		}
	case line == len(box)-3 && len(d.actions(h)) > 0:
		// the actions' line, the last inside the edges: Rename… first, then Delete
		if cell < ansi.StringWidth(d.actions(h)[0]) {
			return d.act(h, "r")
		}
		if cell >= ansi.StringWidth(d.actions(h)[0])+2 {
			return d.act(h, "d")
		}
	}
	return ManageDo{}
}

// Keys gives the keys bar's segment while the presets dialog is open, each key where it acts on the view under the
// keyboard: up and down where there is a line that way, shift with either where a favorite moves that way, space where
// there is a view, and r and d on one of the person's own.
func (d Manage) Keys(h Host) shell.Segment {
	s := shell.Segment{Owns: true, Bindings: []shell.Binding{shell.Bind("esc", "close")}}
	var moves, orders []string
	for _, step := range []struct {
		key string
		by  int
	}{{"up", -1}, {"down", 1}} {
		if to := d.At + step.by; to >= 0 && to < len(d.Listed(h)) {
			moves = append(moves, step.key)
		}
		if _, ok := d.moved(h, step.by); ok {
			orders = append(orders, "shift+"+step.key)
		}
	}
	if len(moves) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("↑↓", "move", moves...))
	}
	if len(orders) > 0 {
		s.Bindings = append(s.Bindings, shell.Bind("⇧↑↓", "order", orders...))
	}
	if name, ok := d.view(h); ok {
		s.Bindings = append(s.Bindings, shell.Bind("␣", "star", "space", "*"))
		if isMine(h, name) {
			s.Bindings = append(s.Bindings, shell.Bind("r", "rename"), shell.Bind("d", "delete"))
		}
	}
	return s
}
