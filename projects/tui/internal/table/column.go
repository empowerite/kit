// The column menu: a click on a column's name in the header drops it under the name, as does a right click where the
// terminal sends one. Many terminals keep option, ctrl, and the right button with a click for their own selection and
// menu, so the menu holds what those would do: the column's sort, a radio that shows it, Ascending or Descending, a
// column that did not sort coming after the keys there are, one that did keeping its place, or Unsorted, which takes
// it out of the keys, as a shift-click on its sort does; for a column that holds a list, the rows standing once per
// value of it, or once with all of them; the rows grouped by it too, innermost, or no longer; the column hidden; and
// the field's popover, which leaves the sort to the menu. The menu's title is the column's name, so no line names it
// again. Shift with left or right moves the column a place among those the header shows, the menu with it.
package table

import (
	"cmp"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// ColumnMenu is the column menu while it is open: a dropdown, the column it acts on, and whether it hangs from the
// column's name in the heading over the groups rather than in the header.
type ColumnMenu struct {
	shell.Dropdown
	Column  string
	Heading bool
}

// ColAct is what a line of the column menu does.
type ColAct int

// What the column menu's lines do: none, the rules; the radio of the column's sort, Ascending, Descending, and
// Unsorted; the rows once per value of a column that holds a list, or once with all of them; the rows grouped by the
// column, innermost, or no longer; the column hidden; and More, the field's popover.
const (
	ActRule ColAct = iota
	ActUp
	ActDown
	ActUnsorted
	ActFlatten
	ActGather
	ActGroup
	ActUngroup
	ActHide
	ActMore
)

// The words of the column menu's lines, and their glyphs: flattening's, the rows spreading down, and gathering's, their
// coming together; grouping's is the mark a column grouping the rows has, hiding's the box of a hidden column in the
// columns dialog; and a sweep, which a menu line that clears draws.
const (
	Unsorted  = "Unsorted"
	BySize    = " by size"
	GroupIt   = "Group"
	UngroupIt = "Ungroup"
	Hide      = "Hide"
	More      = "Options…"

	Sweep        = "🧹"
	FlattenGlyph = "⇊"
	GatherGlyph  = "⇈"
	GroupGlyph   = grid.Grouping
	UngroupGlyph = "▢"
	HideGlyph    = grid.Unticked
	MoreGlyph    = "⋯"
)

// FlattenIt gives the words of the line that flattens a column that holds a list, by the name of one of its values.
func FlattenIt(one string) string { return "One row per " + one }

// GatherIt gives the words of the line that gathers a column's values onto one row again, by its label.
func GatherIt(label string) string { return "All " + label + " on one row" }

// colLine is a line of the column menu: what it draws, and what it does.
type colLine struct {
	item shell.Item
	act  ColAct
}

// OpenColumnMenu gives the column menu dropped down from a column's name, in the header or, as heading says, in the
// heading over the groups, the first line that acts under the keyboard.
func OpenColumnMenu(h Host, column string, heading bool) ColumnMenu {
	c := ColumnMenu{Column: column, Heading: heading}
	c.Dropdown = c.Opened(c.Items(h), func(k int) bool { return c.Enabled(h, k) })
	return c
}

// lines gives the column menu's lines, every column's in one order: its sort, then its grouping, then its flattening,
// then Hide and Options…. The radio's choice picked is the column's sort, Unsorted where it does not sort; a list a row
// holds whole sorts by size, how many values each row holds, as its lines say. Where something else orders the rows, a
// group or another key, its triangles carry ⁿ and ₙ, as the header's mark carries the place the column takes, raised
// after ▲ and lowered after ▼. A section the field cannot use is left out, with its rule: the sort, for one that
// cannot sort; Group and Ungroup, for one that cannot group; and flattening, for one that holds no list. A line the
// field can use but that cannot act now stays, dim.
func (c ColumnMenu) lines(h Host) []colLine {
	p, column := h.Panel(), c.Column
	d, up, down := p.Field(column).Sort, "", ""
	if place, total := p.Place(column); total-min(place, 1) > 0 {
		up, down = "ⁿ", "ₙ"
	}
	radio := func(picked bool, name string, act ColAct) colLine {
		if picked {
			return colLine{shell.Item{Glyph: style.Tint(grid.Picked), Name: name}, act}
		}
		return colLine{shell.Item{Glyph: style.Off(grid.Unpicked), Name: name}, act}
	}
	var lines []colLine
	if p.Sorts(column) {
		by := ""
		if p.Counts(column) {
			by = BySize
		}
		lines = append(lines, radio(d == grid.Ascending, "▲"+up+" Ascending"+by, ActUp),
			radio(d == grid.Descending, "▼"+down+" Descending"+by, ActDown),
			radio(d == grid.Unsorted, Unsorted, ActUnsorted), colLine{act: ActRule})
	}
	if p.Groupable[column] {
		lines = append(lines, colLine{shell.Item{Glyph: GroupGlyph, Name: GroupIt}, ActGroup},
			colLine{shell.Item{Glyph: UngroupGlyph, Name: UngroupIt}, ActUngroup}, colLine{act: ActRule})
	}
	if p.Lists(column) {
		lines = append(lines, colLine{shell.Item{Glyph: FlattenGlyph, Name: FlattenIt(p.Ones[column])}, ActFlatten},
			colLine{shell.Item{Glyph: GatherGlyph, Name: GatherIt(cmp.Or(p.Labels[column], column))}, ActGather},
			colLine{act: ActRule})
	}
	return append(lines, colLine{shell.Item{Glyph: HideGlyph, Name: Hide}, ActHide},
		colLine{shell.Item{Glyph: MoreGlyph, Name: More}, ActMore})
}

// Items gives the column menu's lines as the menu draws them.
func (c ColumnMenu) Items(h Host) []shell.Item {
	var items []shell.Item
	for _, l := range c.lines(h) {
		items = append(items, l.item)
	}
	return items
}

// LineOf gives the index of the column menu's line that does act, -1 where none does.
func (c ColumnMenu) LineOf(h Host, act ColAct) int {
	return slices.IndexFunc(c.lines(h), func(l colLine) bool { return l.act == act })
}

// Enabled says whether the column menu's line k acts: the radio always, as the menu has it only for a column that
// sorts; flattening on a list not flat yet, gathering on one that is; Group on one that does not group yet, Ungroup
// on one that does; and Hide and Options… always; the rules never do.
func (c ColumnMenu) Enabled(h Host, k int) bool {
	lines := c.lines(h)
	if k < 0 || k >= len(lines) {
		return false
	}
	p := h.Panel()
	grouped := slices.Contains(p.Groups(), c.Column)
	switch lines[k].act {
	case ActUp, ActDown, ActUnsorted, ActHide, ActMore:
		return true
	case ActFlatten, ActGather:
		return p.Field(c.Column).Flat == (lines[k].act == ActGather)
	case ActGroup:
		return !grouped
	case ActUngroup:
		return grouped
	}
	return false
}

// Shifted gives the panel with a column moved a place, left for a negative step, right for a positive one, among those
// it stands with: a column among the columns the header names, past those no row fills and those the rows group by on
// their own; a group among the groups, a depth outward or inward. It says whether it moved; at either end it does not,
// nor does it cross into the groups or the hidden.
func Shifted(h Host, column string, d int) (grid.Panel, bool) {
	p := h.Panel()
	placement := p.Field(column).Placement
	order := func(q grid.Panel) []string { return q.Groups() }
	switch placement {
	case grid.Column:
		// those the header names: not one the rows group by on their own, whose value their heading says
		shown := slices.DeleteFunc(slices.Clone(h.ShownColumns()), func(id string) bool {
			_, name := h.HeaderName(id)
			return name == ""
		})
		order = func(q grid.Panel) []string {
			return slices.DeleteFunc(q.Listed(), func(id string) bool {
				return q.Field(id).Placement != grid.Column || !slices.Contains(shown, id)
			})
		}
	case grid.Hidden:
		return p, false
	}
	before := order(p)
	for q := p; ; {
		var ok bool
		if q, ok = q.Step(column, d); !ok || q.Field(column).Placement != placement {
			return p, false
		}
		if !slices.Equal(order(q), before) {
			return q, true
		}
	}
}

// ShiftStep gives the step shift with left or right asks for, and whether the key is one.
func ShiftStep(key string) (int, bool) {
	switch key {
	case "shift+left":
		return -1, true
	case "shift+right":
		return 1, true
	}
	return 0, false
}

// ShiftKeys gives the binding of shift with left and right on a menu that moves a column, each where it moves it.
func ShiftKeys(h Host, column string) []shell.Binding {
	var presses []string
	for _, key := range []string{"shift+left", "shift+right"} {
		if d, _ := ShiftStep(key); func() bool { _, ok := Shifted(h, column, d); return ok }() {
			presses = append(presses, key)
		}
	}
	if len(presses) == 0 {
		return nil
	}
	return []shell.Binding{shell.Bind("⇧←→", "move", presses...)}
}

// Key takes a key while the column menu is open: shift with left or right moves the column, the menu with it, the
// keyboard on the same line; the rest go to the dropdown, and the line chosen does what it says, as Act does.
func (c *ColumnMenu) Key(h Host, k tea.KeyPressMsg) (Popover, bool) {
	if d, ok := ShiftStep(k.String()); ok {
		if p, ok := Shifted(h, c.Column, d); ok {
			h.SetPanel(p)
		}
		return Popover{}, false
	}
	d, chosen := c.Dropdown.Key(k, c.Items(h), func(k int) bool { return c.Enabled(h, k) })
	return c.chose(h, d, chosen)
}

// chose takes what the dropdown did: a line chosen does what it says; a dropdown closed without one closes the menu;
// one still open goes on.
func (c *ColumnMenu) chose(h Host, d shell.Dropdown, chosen int) (Popover, bool) {
	switch {
	case chosen >= 0:
		return c.Act(h, chosen)
	case !d.Open:
		*c = ColumnMenu{}
	default:
		c.Dropdown = d
	}
	return Popover{}, false
}

// Act closes the column menu and does what its line k says, when it acts: Options… gives the field's popover to open,
// hanging from the column's name, and says it does.
func (c *ColumnMenu) Act(h Host, k int) (Popover, bool) {
	was, enabled := *c, c.Enabled(h, k)
	*c = ColumnMenu{}
	if !enabled {
		return Popover{}, false
	}
	p := h.Panel()
	switch act := was.lines(h)[k].act; act {
	case ActUp, ActDown, ActUnsorted:
		// a column that did not sort comes after the keys there are; one that did turns in its place, or leaves them
		p = p.SortedAs(was.Column, map[ColAct]grid.Direction{ActUnsorted: grid.Unsorted, ActUp: grid.Ascending,
			ActDown: grid.Descending}[act])
	case ActFlatten, ActGather:
		p = p.Flattened(was.Column, act == ActFlatten)
	case ActHide:
		p = p.Placed(was.Column, grid.Hidden)
	case ActGroup, ActUngroup:
		// a column joins the groups innermost, or leaves them from wherever it stands
		p = p.Grouped(was.Column, true)
	default:
		x, _ := h.HeaderName(was.Column)
		y := h.HeaderTop()
		if was.Heading {
			x, y = h.HeadingName(was.Column)
		}
		return OpenPopover(h, was.Column, x, y, true), true
	}
	h.SetPanel(p)
	return Popover{}, false
}

// Box gives the column menu's lines, the box around the column's name, which its top edge draws where the header does,
// as the app's menu hangs from =, or where the heading over the groups does; and the screen line and column it is
// drawn from. Once the heading no longer shows, there are none.
func (c ColumnMenu) Box(h Host) (lines []string, top, left int) {
	enabled := func(k int) bool { return c.Enabled(h, k) }
	width, _ := h.Size()
	if c.Heading {
		x, y := h.HeadingName(c.Column)
		if y < 0 {
			return nil, 0, 0
		}
		_, pane := h.Pane()
		lines, left = shell.Box(c.Items(h), enabled, c.At, c.Column, x, pane)
		return lines, y, left
	}
	x, name := h.HeaderName(c.Column)
	lines, left = shell.Box(c.Items(h), enabled, c.At, name, x, width)
	return lines, h.HeaderTop(), left
}

// Click takes a click while the column menu is open, as a dropdown does: on a line that acts, it does what the line
// says; on another line of the box, nothing; anywhere else, on the name it hangs from too, it closes the menu, as it
// does when its box no longer shows.
func (c *ColumnMenu) Click(h Host, x, y int) (Popover, bool) {
	box, top, left := c.Box(h)
	if len(box) == 0 {
		*c = ColumnMenu{}
		return Popover{}, false
	}
	d, chosen := c.Dropdown.Click(x, y, top, left, ansi.StringWidth(box[0]), c.Items(h),
		func(k int) bool { return c.Enabled(h, k) })
	return c.chose(h, d, chosen)
}
