// A sheet is one table on a screen, whole: its catalog's fields and presets, its records, its state, the control line
// over its rows with the heading over the groups and the preset line, the header and its rule, its layers, and the
// keyboard among them. A screen places it in a pane, gives it the keys, the clicks, and the mouse wheel's turns the
// pane takes, draws its lines and its layers, names its keys, and sets its search; the sheet is the host of its own
// controls. Its rows have no links,
// as a screen that opens pages has none to follow, and it reads no caller's login, so @me names no one. The person's
// presets of it are kept in the presets file under its catalog's name for it, beside every other table's.
package table

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// Failed starts what a sheet says it could not do.
const Failed = "❌"

// Fades is how long a notice that says what was done stays in place of the keys, short of a key. A notice that says
// why something was not done stays until a key.
const Fades = 5 * time.Second

// Sheet is one table on a screen: its catalog's part, its name, which titles its dialogs, its color, the presets file,
// and the name its presets are kept under in it; its records; its state; the person's presets of it; whether it folds
// a value every row shares and draws an emoji every row shows, as kittui's settings say; the pane it is placed in, its
// first line, its height, and the screen's width and height; where the keyboard is in it, the rows, or the control
// line on a stop of the heading over the groups, -1 the preset picker; its layers; what it did, or why not, in place
// of the keys; its rows as last built; how it opens a link; and which of its rows can be selected, none while nil.
type Sheet struct {
	schema        *Schema
	name, color   string
	path, store   string
	records       []Record
	tbl           Table
	saved         []Preset
	starred       []string
	fold, hide    bool
	glyphs        bool
	top, height   int
	width, screen int
	inRows        bool
	onControls    bool
	stop          int
	pop           Popover
	cols          Columns
	presets       PresetMenu
	manage        Manage
	note          shell.Notice
	built         *Shown
	drawn         *drawnLines
	browse        func(string) error
	picks         func(key string) bool
}

var _ Keyboard = (*Sheet)(nil)

// NewSheet gives a sheet of the catalog's table, named and colored, its rows folding as kittui's settings do until told
// otherwise, set as the first of the table's own presets, with the person's presets of it read from the file at path,
// under the catalog's name for the table, or the sheet's where the catalog names it not; none keeps them for the
// session alone. A file that does not read is said in place of the keys, and left as it is.
func NewSheet(schema *Schema, name, color, path string) Sheet {
	s := Sheet{schema: schema, name: name, color: color, path: path, store: name, fold: true, hide: true, stop: -1, width: 80,
		browse: shell.Browse,
		screen: 24, height: 20, tbl: Table{Open: grid.Outline{}, Picked: map[string]bool{}, Grouping: true}}
	if schema != nil && schema.Stage != "" {
		s.store = schema.Stage
	}
	all, err := ReadPresets(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		s.note.Text = fmt.Sprintf("%s %v; the file is left as it is", Failed, err)
	}
	s.saved, s.starred = all[s.store], ReadFavorites(path)[s.store]
	opening := s.Views().Opening()
	s.tbl.ViewName, s.tbl.View = opening.Name, opening.Panel(schema.Of(), grid.Panel{})
	return s
}

// SetRecords gives the sheet its records, which its rows are built from; the first rows the view is set for group as
// it loads them, as autoGroup says, the columns none of them holds a value in are blank, as Table's Blank says, and the
// cursor stays on a row.
func (s *Sheet) SetRecords(records []Record) {
	s.records = records
	s.tbl.Blank = grid.Blanks(Lay(s.schema.Of(), records, func(id string) string { return s.tbl.View.Field(id).Rendering }),
		s.schema.Columns())
	s.prune()
	s.rebuild()
	s.autoGroup()
	s.Settle()
}

// Place puts the sheet's pane on the screen: from the line top, height lines tall, on a screen width wide and screen
// tall, the pane a cell narrower than the screen.
func (s *Sheet) Place(top, height, width, screen int) {
	s.top, s.height, s.width, s.screen = top, max(1, height), width, screen
	s.rebuild()
	s.Settle()
}

// Settings sets whether the rows fold a value every row shares, keep the columns no row holds a value in from showing,
// and draw an emoji every row shows, as kittui's settings say.
func (s *Sheet) Settings(set shell.Settings) {
	s.fold, s.hide, s.glyphs = set.FoldShared, set.HideBlank, set.SharedEmoji
	s.rebuild()
	s.Settle()
}

// Search narrows the rows to those whose text holds the query, without regard to case; nothing typed lets every row
// through.
func (s *Sheet) Search(query string) {
	if p := s.Panel(); p.Query != query {
		p.Query = query
		s.SetPanel(p)
	}
}

// SetPicks lets the rows picks says of, by key, be selected, as one of several: space, shift with up or down, ctrl+a,
// and a plain click on a row's box, as Drawing draws it, select them, a group's heading every one of its rows that can
// be. nil, as a new sheet has it, lets none be. A row selected that can no longer be is no longer selected.
func (s *Sheet) SetPicks(picks func(key string) bool) {
	s.picks, s.drawn = picks, nil
	s.prune()
}

// Selects says whether any row can be selected, as SetPicks says.
func (s *Sheet) Selects() bool { return s.picks != nil }

// Picks says whether the row with the key can be selected, as SetPicks says: any, where it set none, as by a host that
// selects for itself, since the sheet's own keys and clicks then select none.
func (s *Sheet) Picks(key string) bool { return s.picks == nil || s.picks(key) }

// Selected gives the keys of the rows selected, in the order the rows stand, each once, though its closed groups
// fold them away or the search hides them.
func (s *Sheet) Selected() []string {
	var out []string
	for _, r := range s.records {
		if s.tbl.Picked[r.Row.Key] && !slices.Contains(out, r.Row.Key) {
			out = append(out, r.Row.Key)
		}
	}
	return out
}

// Unselect selects no row.
func (s *Sheet) Unselect() { clear(s.tbl.Picked) }

// prune unselects each row that is no longer a record's, or can no longer be selected.
func (s *Sheet) prune() {
	keys := map[string]bool{}
	for _, r := range s.records {
		keys[r.Row.Key] = true
	}
	for k := range s.tbl.Picked {
		if !keys[k] || !s.Picks(k) {
			delete(s.tbl.Picked, k)
		}
	}
}

// rebuild drops the rows built, which are built again when next asked for.
func (s *Sheet) rebuild() { s.built, s.drawn = nil, nil }

// shown gives the sheet's rows as its panel lets them through, built once until something they are built from changes.
func (s *Sheet) shown() Shown {
	if s.built == nil {
		b := Build(s.schema, s.records, s.filter(), HeaderNames(s.filter(), s.schema.Columns()), s.paneWidth())
		s.built = &b
	}
	return *s.built
}

// filter gives the sheet's panel as its rows are let through: what the person set in it, what the catalog says of its
// fields, what stands open, and the settings, the blank columns held empty where they hide them; the columns it offers
// are drawn from the rows, so it has none.
func (s *Sheet) filter() grid.Panel {
	p := s.schema.Ready(s.tbl.View)
	p.Outline, p.Fold, p.Glyphs, p.Empty = s.tbl.Open, s.fold, s.glyphs, nil
	if s.hide {
		p = p.Blanked(s.tbl.Blank, s.schema.Columns())
	}
	return p
}

// paneWidth gives the cells the rows have: the screen's, but its last cell.
func (s *Sheet) paneWidth() int { return max(1, s.width-1) }

// autoGroup groups the rows as the view groups them as they load, once there are rows to look at: by each column whose
// value is the same on every row the panel lets through, as the panel's AutoGrouped says, where the rows fold and the
// view groups by nothing. It runs once a view; past that, the groups are the person's, and the preset line names the
// view still.
func (s *Sheet) autoGroup() {
	if !s.tbl.Grouping || len(s.records) == 0 {
		return
	}
	s.tbl.Grouping = false
	p := s.Panel()
	next := p.AutoGrouped(Lay(s.schema.Of(), s.records, func(id string) string { return p.Field(id).Rendering }),
		s.schema.Columns())
	if len(next.Groups()) == len(p.Groups()) {
		return
	}
	s.tbl.View = next
	grouped := s.Views().Current(next)
	s.tbl.Grouped = &grouped
	s.rebuild()
}

// Fields gives the table's fields.
func (s *Sheet) Fields() []Field { return s.schema.Of() }

// Panel gives the sheet's panel, as its rows are let through, with the columns it offers.
func (s *Sheet) Panel() grid.Panel {
	p := s.filter()
	p.Columns = s.shown().Names
	return p
}

// SetPanel keeps what the panel became; the rows are let through again, and the cursor stays within them.
func (s *Sheet) SetPanel(p grid.Panel) {
	s.tbl.View = p
	s.rebuild()
	s.Settle()
}

// Values gives the values the records hold in a field, one a record.
func (s *Sheet) Values(id string) []Value {
	var out []Value
	for _, r := range s.records {
		out = append(out, r.Values[id])
	}
	return out
}

// Label gives the label of a name, with its color, as one of the records holds it.
func (s *Sheet) Label(id, name string) Label { return LabelNamed(s.records, id, name) }

// Me gives the caller's login, which the sheet never reads: none.
func (s *Sheet) Me() string { return "" }

// Drawn gives the rendering a field is drawn by now, as Automatic chose it.
func (s *Sheet) Drawn(id string) string { return s.shown().Drawn[id] }

// Size gives the screen's width and height.
func (s *Sheet) Size() (width, height int) { return s.width, s.screen }

// Name gives the table's name, which titles its dialogs.
func (s *Sheet) Name() string { return s.name }

// Pane gives the screen line the pane starts on, and its width.
func (s *Sheet) Pane() (top, width int) { return s.top, s.paneWidth() }

// ShownColumns gives the columns the header shows, in their order.
func (s *Sheet) ShownColumns() []string { return ShownColumns(s.filter(), s.schema.Columns()) }

// HeaderName gives where a column's name starts in the header, and its first line.
func (s *Sheet) HeaderName(column string) (int, string) { return ColumnName(s, column) }

// HeaderTop gives the screen line the header starts on.
func (s *Sheet) HeaderTop() int { return HeaderTop(s) }

// HeadingName gives where a column's name stands in the heading over the groups.
func (s *Sheet) HeadingName(column string) (x, y int) { return FieldName(s, column) }

// Rows gives the rows as the table lays them out.
func (s *Sheet) Rows() []grid.Row { return s.shown().Rows }

// Names gives the names the header stands over the rows.
func (s *Sheet) Names() []string { return HeaderNames(s.filter(), s.schema.Columns()) }

// GridWidth gives the width the rows are laid out in: the pane's, less their indent under the headings of their
// groups, two a level.
func (s *Sheet) GridWidth() int { return s.paneWidth() - 2*s.shown().Levels }

// Table gives the table's state.
func (s *Sheet) Table() *Table { return &s.tbl }

// Rebuild builds the rows again, once what stands open changed.
func (s *Sheet) Rebuild() { s.rebuild() }

// Settle keeps the cursor on a row, the first it can stand on while the rows have the keyboard, and in the pane's view.
func (s *Sheet) Settle() {
	rows, first := s.Rows(), First(s.Rows())
	if n := len(rows); s.tbl.Cursor >= n {
		s.tbl.Cursor = max(first, n-1)
	}
	if s.inRows {
		s.tbl.Cursor = max(s.tbl.Cursor, first)
	}
	Fit(s)
}

// InRows says whether the rows have the keyboard.
func (s *Sheet) InRows() bool { return s.inRows }

// Views gives the table's views, its own and the person's; a table whose catalog gives it none has Default.
func (s *Sheet) Views() Views {
	var own []Preset
	if s.schema != nil {
		own = slices.Clone(s.schema.Presets)
	}
	if len(own) == 0 {
		own = []Preset{{Name: Default}}
	}
	return Views{Fields: s.schema.Of(), Own: own, Saved: s.saved, Starred: s.starred}
}

// Columnar says whether the table has fields the columns dialog lists.
func (s *Sheet) Columnar() bool { return len(s.schema.Of()) > 0 }

// Color gives the table's color.
func (s *Sheet) Color() string { return s.color }

// Selector gives the screen columns the preset line's selector stands on, from its first to the one past it.
func (s *Sheet) Selector() (from, until int) {
	_, at := s.control()
	return at, at + ansi.StringWidth(s.presetLine())
}

// Lines gives the lines the pane draws, each its row's: the rows as a Drawing draws them, the one under the cursor on
// a band while the rows have the keyboard; or a dim line where no row shows, saying whether the search let none
// through.
func (s *Sheet) Lines() []Line {
	rows := s.Rows()
	switch p := s.filter(); {
	case len(rows) > 0:
	case p.Query != "" || p.Filtering():
		return []Line{{Row: -1, Text: Lead + style.Off("no row matches the search")}}
	default:
		return []Line{{Row: -1, Text: Lead + style.Off("no rows")}}
	}
	chosen := -1
	if s.inRows && s.tbl.Cursor < len(rows) {
		chosen = s.tbl.Cursor
	}
	d := Drawing{Names: s.Names(), Width: s.GridWidth(), Shift: s.tbl.Shift, Pane: s.paneWidth(),
		Indent: s.paneWidth() - s.GridWidth(), Open: s.tbl.Open, Picked: s.tbl.Picked, Color: s.color}
	if s.picks != nil {
		d.Picks = s.Picks
	}
	key := drawnFor(d, rows, chosen, s.picks != nil)
	if s.drawn != nil && s.drawn.key == key {
		return s.drawn.lines
	}
	s.drawn = &drawnLines{key, d.Draw(rows, chosen)}
	return s.drawn.lines
}

// drawnLines is the lines a sheet drew, and what it drew them for, so Lines draws them once and every key, click,
// turn of the wheel, and drawing after reuses them, however many rows there are.
type drawnLines struct {
	key   drawnKey
	lines []Line
}

// drawnKey says what a sheet's lines are drawn for: the rows themselves, held, so their memory is never another's;
// the row under the cursor; the drawing's widths, scroll, and color; whether rows can be selected, which SetPicks,
// the only way to change what can be, says anew; and the names, the rows open, and the rows selected, written out.
type drawnKey struct {
	first                      *grid.Row
	n, chosen                  int
	width, shift, pane, indent int
	color                      string
	picks                      bool
	names, open, picked        string
}

// drawnFor gives the key of the lines the drawing draws of the rows, the one at chosen under the cursor.
func drawnFor(d Drawing, rows []grid.Row, chosen int, picks bool) drawnKey {
	var open, picked strings.Builder
	for _, k := range slices.Sorted(maps.Keys(d.Open)) {
		fmt.Fprintf(&open, "%s=%t\x00", k, d.Open[k])
	}
	for _, k := range slices.Sorted(maps.Keys(d.Picked)) {
		if d.Picked[k] {
			picked.WriteString(k + "\x00")
		}
	}
	return drawnKey{first: &rows[0], n: len(rows), chosen: chosen, width: d.Width, shift: d.Shift, pane: d.Pane,
		indent: d.Indent, color: d.Color, picks: picks, names: strings.Join(d.Names, "\x00"), open: open.String(),
		picked: picked.String()}
}

// Header gives the header's lines and its rule, and whether the header, closed, hides names; none while no row shows.
func (s *Sheet) Header() ([]string, bool) {
	rows := s.Rows()
	if !slices.ContainsFunc(rows, func(r grid.Row) bool { return !r.Aside }) {
		return nil, false
	}
	lines, hides := Headed(rows, s.Names(), s.GridWidth(), s.tbl.Shift, s.paneWidth()-s.GridWidth(),
		s.tbl.Open[HeaderKey])
	if len(lines) == 0 {
		return nil, false
	}
	return append(lines, Rule(s, SortSpots(s))), hides
}

// PaneHeight gives the pane's height.
func (s *Sheet) PaneHeight() int { return s.height }

// Controls says whether the control line stands over the rows: wherever the table has fields, whose preset line it
// holds.
func (s *Sheet) Controls() bool { return s.Columnar() }

// LeaveRows gives up the rows' keyboard, past their first, to the control line, on its first stop.
func (s *Sheet) LeaveRows() {
	s.inRows = false
	if s.HasPresets() || len(Stops(s)) > 0 {
		s.toControls()
	}
}

// EnterRows gives the rows the keyboard.
func (s *Sheet) EnterRows() { s.inRows, s.onControls = true, false }

// Stop gives the stop of the heading over the groups the keyboard stands on, -1 the preset picker.
func (s *Sheet) Stop() int { return s.stop }

// SetStop puts the keyboard on a stop of the heading over the groups, -1 the preset picker.
func (s *Sheet) SetStop(k int) { s.stop = k }

// LeaveControls gives up the control line's keyboard, to the screen.
func (s *Sheet) LeaveControls() { s.onControls = false }

// OpenFieldMenu opens a column's card around its name in the heading over the groups, on the control line, which
// then has the keyboard on that name's stop.
func (s *Sheet) OpenFieldMenu(column string) {
	s.onControls, s.inRows = s.HasPresets(), false
	s.stop = slices.IndexFunc(Stops(s), func(st Stop) bool { return st.Column == column })
	s.pop = OpenCard(s, column, true)
}

// HasPresets says whether the table has a preset line: one with fields, the views of which it names.
func (s *Sheet) HasPresets() bool { return s.Columnar() }

// toControls gives the control line the keyboard, on its first stop: the first name of the heading over the groups,
// or the preset picker where the rows group by nothing.
func (s *Sheet) toControls() {
	s.onControls, s.inRows, s.stop = true, false, -1
	if len(Stops(s)) > 0 {
		s.stop = 0
	}
}

// atStop gives the stop of the heading over the groups the keyboard stands on, and whether it stands on one.
func (s *Sheet) atStop() (Stop, bool) {
	stops := Stops(s)
	if !s.onControls || s.stop < 0 || s.stop >= len(stops) {
		return Stop{}, false
	}
	return stops[s.stop], true
}

// presetLine gives the preset's selector, as PresetLine draws it, the line having the keyboard while the picker does.
func (s *Sheet) presetLine() string {
	_, on := s.atStop()
	return PresetLine(s, s.onControls && !on)
}

// control gives the control line and the screen column the selector starts on, as Control draws them: the heading
// over the groups, the stop under the keyboard drawn as FocusStop draws it, then the preset line's selector.
func (s *Sheet) control() (string, int) { return Control(s, s.controlLeft(), s.presetLine()) }

// controlLeft gives the control line's left part: the heading over the groups, the stop under the keyboard drawn as
// FocusStop draws it.
func (s *Sheet) controlLeft() string {
	left := strings.Repeat(" ", BarLeft) + BarText(s.Rows())
	if stop, ok := s.atStop(); ok {
		left = FocusStop(left, stop)
	}
	return left
}

// Focused says whether the sheet has the keyboard: its rows, or its control line.
func (s *Sheet) Focused() bool { return s.inRows || s.onControls }

// Focus gives the sheet the keyboard from the screen, as down from a tab bar does: the control line, on its first stop,
// where the table has one, else the rows.
func (s *Sheet) Focus() {
	if s.HasPresets() || len(Stops(s)) > 0 {
		s.toControls()
		return
	}
	Down(s, 1)
}

// FocusRows gives the rows the keyboard, the cursor where it was, as a search giving up the keyboard does; a table
// without a row gives its control line the keyboard instead.
func (s *Sheet) FocusRows() {
	if len(s.Rows()) <= First(s.Rows()) {
		s.Focus()
		return
	}
	s.EnterRows()
	s.Settle()
}

// Blur takes the keyboard from the sheet, as a screen does when it gives it elsewhere; a name typed for a preset is
// dropped, and the rows' window keeps its place.
func (s *Sheet) Blur() {
	s.inRows, s.onControls = false, false
	s.tbl.SavingAs, s.tbl.Naming, s.tbl.Renaming = false, "", ""
}

// layers gives the sheet's layers, the topmost first.
func (s *Sheet) layers() shell.Layers[Sheet] {
	return Overlays[Sheet]{
		Host:    func(s *Sheet) Host { return s },
		Pop:     func(s *Sheet) *Popover { return &s.pop },
		Cols:    func(s *Sheet) *Columns { return &s.cols },
		Presets: func(s *Sheet) *PresetMenu { return &s.presets },
		Manage:  func(s *Sheet) *Manage { return &s.manage },
		Managed: (*Sheet).managed,
		Popped:  func(*Sheet) tea.Cmd { return nil },
		Choose:  (*Sheet).choose,
		Star:    (*Sheet).star,
	}.Layers()
}

// Layered says whether a layer of the sheet's is open, which takes every key and click.
func (s *Sheet) Layered() bool {
	_, open := s.layers().Keys(*s)
	return open
}

// Key takes a key, and says whether the sheet took it: an open layer's, the topmost; then the control line's, while
// it has the keyboard; then the rows', while they have it. A key clears the notice in place of the keys; a name typed
// for a preset is dropped once the control line no longer has the keyboard.
func (s *Sheet) Key(k tea.KeyPressMsg) bool {
	s.note.Text = ""
	took := s.key(k)
	if !s.onControls {
		s.tbl.SavingAs, s.tbl.Naming, s.tbl.Renaming = false, "", ""
	}
	return took
}

// key takes a key, as Key says.
func (s *Sheet) key(k tea.KeyPressMsg) bool {
	if next, _, ok := s.layers().Key(*s, k); ok {
		*s = next
		return true
	}
	// > and < show a level more and less of the whole table from the rows and the control line, as they do on a
	// stage of flow, unless a preset's name is being typed
	if (s.inRows || s.onControls) && !s.tbl.SavingAs && Fold(s, k.String()) {
		return true
	}
	switch {
	case s.onControls:
		if _, on := s.atStop(); on {
			return HeadingKey(s, k)
		}
		do, took := PresetKey(s, k)
		switch do {
		case DoSave:
			s.saveAs(strings.TrimSpace(s.tbl.Naming))
		case DoRename:
			s.rename()
		case DoDrop:
			s.presets = s.presets.Drop(s)
		case DoFields:
			s.cols = Columns{Open: true}
		}
		return took
	case s.inRows:
		return s.rowKey(k)
	}
	return false
}

// rowKey takes a key while the rows have the keyboard, and says whether it took it: esc gives the screen the keyboard,
// the rows' window back at its top; up and down move, past the first row to the control line; a page, home, and end
// move further; tab and shift-tab scroll the columns a column, while the view hides some; [ ] { } move between the
// groups' headings; right and left show and hide what the node under the cursor
// holds, a level at a time with ⌥. Where rows can be selected, as SetPicks says, space selects the row under the
// cursor, shift with up or down selects as it moves, and ctrl+a every row, or none.
func (s *Sheet) rowKey(k tea.KeyPressMsg) bool {
	switch key := k.String(); {
	case !s.Selects():
	case key == "space":
		SelectRow(s, false)
		return true
	case key == "ctrl+a":
		SelectAll(s)
		return true
	case key == "shift+up" || key == "shift+down":
		SelectRow(s, true)
		if key == "shift+up" {
			Up(s, 1)
		} else {
			Down(s, 1)
		}
		SelectRow(s, true)
		return true
	}
	switch key := k.String(); key {
	case "esc":
		s.tbl.Cursor, s.tbl.Offset, s.inRows = 0, 0, false
		Fit(s)
	case "up":
		Up(s, 1)
	case "down":
		Down(s, 1)
	case "pgup":
		Page(s, -1)
	case "pgdown":
		Page(s, 1)
	case "home":
		End(s, -1)
	case "end":
		End(s, 1)
	case "[", "]", "{", "}":
		Group(s, key)
	case "tab", "shift+tab":
		if _, before, after := Scrolled(s); !before && !after {
			return false
		}
		d := 1
		if key == "shift+tab" {
			d = -1
		}
		ScrollColumns(s, d)
	case "right", "alt+right":
		Expand(s, true, key == "alt+right")
	case "left", "alt+left":
		Expand(s, false, key == "alt+left")
	default:
		return false
	}
	return true
}

// Click takes a click, and says whether the sheet took it: an open layer takes every click, on it or off it; a plain
// click on the pane takes the rest, on the control line, the rule, the header, or the rows, as click says. Another
// click off the pane is the screen's.
func (s *Sheet) Click(c tea.MouseClickMsg) bool {
	if next, _, ok := s.layers().Click(*s, c); ok {
		*s = next
		return true
	}
	if c.Button != tea.MouseLeft || c.X >= s.paneWidth() || c.Y < s.top || c.Y >= s.top+s.height {
		return false
	}
	s.note.Text = ""
	s.click(c.X, c.Y, c.Mod&tea.ModShift != 0)
	if !s.onControls {
		s.tbl.SavingAs, s.tbl.Naming, s.tbl.Renaming = false, "", ""
	}
	return true
}

// WheelLines is the lines a turn of the mouse wheel scrolls the rows' window.
const WheelLines = 3

// Wheel takes a turn of the mouse wheel, and says whether the sheet took it: over the pane, it scrolls the rows'
// window WheelLines down or up, never past its ends, the cursor left where it is, until a key moves it. A turn off the
// pane, or while a layer is open, is not the sheet's.
func (s *Sheet) Wheel(w tea.MouseWheelMsg) bool {
	d := 0
	switch w.Button {
	case tea.MouseWheelUp:
		d = -WheelLines
	case tea.MouseWheelDown:
		d = WheelLines
	}
	if d == 0 || s.Layered() || w.X >= s.paneWidth() || w.Y < s.top || w.Y >= s.top+s.height {
		return false
	}
	s.tbl.Offset, _, _, _ = pane.Window(len(ScrollLines(s)), RowsHeight(s), s.tbl.Offset+d)
	return true
}

// click takes a plain click on the pane: on the control line, on a level's ▾ or ▸ it opens or closes the level, on a
// column's name it opens the column's card around it, and elsewhere the preset line has the keyboard, and on the
// selector the presets drop down; on the rule under the heading over the groups, as BarRuleClick says; on the rule, a
// sort's target turns its sort, a shift-click off, and a list's ×
// flattens or gathers it; on the header, its ▹ or ◃ scrolls the columns, its fold opens or closes it, and a column's
// name opens the column's card under it; on a row, the cursor goes there, the rows have the keyboard, and its ▸ or ▾ opens
// or closes it, and a value that leads somewhere, a page or a file, is opened, as open says; where rows can be
// selected, a click on a row's box, as BoxAt says, selects it or unselects it. A line about the table does nothing.
func (s *Sheet) click(x, y int, shift bool) {
	switch _, onHeader := HeaderLine(s, y); {
	case s.Controls() && y == s.top:
		s.controlClick(x)
	case s.Controls() && y == s.top+1:
		BarRuleClick(s, x, shift)
	case OnRule(s, y):
		switch column, flatten := SortAt(Rule(s, SortSpots(s)), SortSpots(s), x); {
		case flatten:
			p := s.Panel()
			s.SetPanel(p.Flattened(column, !p.Field(column).Flat))
		case column != "":
			SortBy(s, column, shift)
		}
	case onHeader:
		switch _, fold, column := HeaderHit(s, x, y); {
		case ArrowAt(s, x, y) != 0:
			ScrollColumns(s, ArrowAt(s, x, y))
		case fold:
			ToggleHeader(s)
		case column != "":
			s.pop = OpenCard(s, column, false)
		}
	default:
		row, link, fold, ok := RowAt(s, x, y)
		if !ok {
			return
		}
		if s.Selects() && BoxAt(s, x, y) {
			s.tbl.Cursor, s.inRows, s.onControls = row, true, false
			SelectRow(s, false)
			Fit(s)
			return
		}
		if link != "" {
			s.open(link)
		}
		s.tbl.Cursor, s.inRows, s.onControls = row, true, false
		if fold {
			Disclose(s, !s.tbl.Open.Unfolds(s.Rows()[row]))
		}
		Fit(s)
	}
}

// controlClick takes a click on the control line, as click says: a chip opens its field's card on its filter, and
// FieldsButton the columns dialog.
func (s *Sheet) controlClick(x int) {
	if level, ok := LevelAt(s, x); ok {
		ToggleLevel(s, level)
		return
	}
	if name, ok := NameAt(s, x); ok {
		s.OpenFieldMenu(name.Column)
		return
	}
	if spot, ok := SpotAt(ControlSpots(s, s.controlLeft(), s.presetLine()), x); ok {
		if spot.Field == "" {
			s.cols = Columns{Open: true}
		} else {
			s.pop = OpenChip(s, spot)
		}
		return
	}
	s.onControls, s.inRows, s.stop = true, false, -1
	if from, until := s.Selector(); x >= from && x < until {
		s.presets = s.presets.Drop(s)
	}
}

// View gives the pane's lines, as many as it has, each as wide as the pane: the control line and a blank line under
// it, then the frozen headings, the header, and the rows' window, as Laid lays them out.
func (s *Sheet) View() []string {
	var out []string
	if s.Controls() {
		line, _ := s.control()
		out = append(out, line, BarRule(s))
	}
	texts, _ := Laid(s)
	out = append(out, texts...)
	for len(out) < s.height {
		out = append(out, "")
	}
	return out[:s.height]
}

// Over draws the sheet's open layers over the screen's lines.
func (s *Sheet) Over(lines []string) []string { return s.layers().Draw(*s, lines) }

// Keys gives the keys bar's segment of the part of the sheet the keyboard is in, and whether a layer is open, whose
// keys are then the whole path: a layer's; a stop's of the heading over the groups, or the preset line's; or the rows'.
// Without the keyboard, it has no keys.
func (s *Sheet) Keys() (shell.Segment, bool) {
	if path, open := s.layers().Keys(*s); open {
		return path[0], true
	}
	switch {
	case s.onControls:
		if stop, on := s.atStop(); on {
			return HeadingKeys(s, stop, s.HasPresets()), false
		}
		return PresetKeys(s), false
	case s.inRows:
		return RowKeys(s, false), false
	}
	return shell.Segment{}, false
}

// Notice gives what the sheet did, or why not, which stands in place of the keys.
func (s *Sheet) Notice() shell.Notice { return s.note }

// Faded takes a notice's fade, as shell.Notice's Faded does.
func (s *Sheet) Faded(f shell.FadedMsg) { s.note = s.note.Faded(f) }

// choose closes the presets' menu and takes its line k: Save keeps the panel under the person's preset the line was set
// to; Save As… gives the preset line the keyboard, for a name, that preset's typed already; Manage… opens the presets
// dialog, on the preset the line was set to; a preset applies. A line that does not act now does nothing.
func (s *Sheet) choose(k int) {
	s.presets.Open = false
	if !PresetEnabled(s, k) {
		return
	}
	v, _ := s.Views().Mine(s.tbl.ViewName)
	switch k {
	case ItemSave:
		s.saveAs(v.Name)
	case ItemSaveAs:
		s.tbl.SavingAs, s.tbl.Naming, s.tbl.Renaming = true, v.Name, ""
		s.onControls, s.inRows, s.stop = true, false, -1
	case ItemManage:
		s.manage = Manage{Open: true}
		s.manage.At = max(0, slices.Index(s.manage.Listed(s), s.tbl.ViewName))
	default:
		s.apply(s.Views().Chosen(PresetItems(s)[k].Name))
	}
}

// managed does what the presets dialog asks: keeps the favorites as given; gives the preset line the keyboard, for a
// view's new name, its name typed already; or deletes a view, the table set as the one it opens on where it was set as
// the view.
func (s *Sheet) managed(do ManageDo) {
	switch do.Act {
	case ManageFavorites:
		s.favor(do.Name, do.Favorites)
	case ManageRename:
		s.tbl.SavingAs, s.tbl.Naming, s.tbl.Renaming = true, do.Name, do.Name
		s.onControls, s.inRows, s.stop = true, false, -1
	case ManageDelete:
		if _, ok := s.Views().Mine(do.Name); ok && s.keep(s.Views().Delete(do.Name), "deleted") {
			if s.tbl.ViewName == do.Name {
				s.apply(s.Views().Opening())
			}
			s.note = s.note.Did(Deleted + " deleted the preset " + do.Name)
		}
	}
}

// apply sets the panel as the preset keeps it, its search kept, and names the preset, which then groups the rows as
// they load, as autoGroup says.
func (s *Sheet) apply(v Preset) {
	p := v.Panel(s.schema.Of(), s.tbl.View)
	s.tbl.ViewName, s.tbl.SavingAs, s.tbl.Naming = v.Name, false, ""
	s.tbl.Grouping, s.tbl.Grouped = true, nil
	s.SetPanel(p)
	s.autoGroup()
	s.Settle()
}

// refused says why a name typed for a preset was refused, in place of the keys, until a key: none typed, the table's
// own, or, renaming, the person's already.
func (s *Sheet) refused(name string, err error) {
	switch {
	case errors.Is(err, ErrUnnamed):
		s.note.Text = Failed + " a preset needs a name: type one, then ⏎"
	case errors.Is(err, ErrBuiltin):
		s.note.Text = Failed + " " + name + " is the table's own and does not change: type a new name, then ⏎"
	default:
		s.note.Text = Failed + " " + name + " is a preset already: type a new name, then ⏎"
	}
	s.note.Done = false
}

// saveAs keeps the panel under a name, as Views saves it; the preset line then names it, and the notice says so.
func (s *Sheet) saveAs(name string) {
	saved, err := s.Views().SaveAs(name, s.tbl.View)
	switch {
	case err != nil:
		s.refused(name, err)
		return
	case !s.keep(saved, "saved"):
		return
	}
	s.tbl.ViewName, s.tbl.SavingAs, s.tbl.Naming, s.tbl.Renaming = name, false, "", ""
	s.note = s.note.Did(Save + " saved the preset " + name)
}

// rename gives the person's preset being renamed the name typed, as Views renames it; the same name leaves it as it
// is. The preset line names it by its new name.
func (s *Sheet) rename() {
	from, name := s.tbl.Renaming, strings.TrimSpace(s.tbl.Naming)
	saved, err := s.Views().Rename(from, name)
	switch {
	case err != nil:
		s.refused(name, err)
		return
	case name == from:
	case !s.keepStarred(saved, s.Views().Renamed(from, name), "renamed"):
		return
	default:
		if s.tbl.ViewName == from {
			s.tbl.ViewName = name
		}
		s.note = s.note.Did(Renamed + " renamed the preset " + from + " to " + name)
	}
	s.tbl.SavingAs, s.tbl.Naming, s.tbl.Renaming = false, "", ""
}

// keep writes the person's presets of the table as they are given, under its store, into the file as it stands, as
// KeepPresets does, so another screen's saved since are kept; and says whether it could. When it could not, the notice
// says why, and nothing changes.
func (s *Sheet) keep(saved []Preset, what string) bool { return s.keepStarred(saved, s.starred, what) }

// keepStarred keeps the person's presets, as keep does, and their favorites as starred names them, but those that name
// a view no longer there.
func (s *Sheet) keepStarred(saved []Preset, starred []string, what string) bool {
	v := s.Views()
	v.Saved, v.Starred = saved, starred
	if _, err := KeepPresets(s.path, s.store, saved, v.Favorites()); err != nil {
		s.note.Text, s.note.Done = fmt.Sprintf("%s the preset could not be %s: %v", Failed, what, err), false
		return false
	}
	s.saved, s.starred = saved, v.Favorites()
	return true
}

// star stars the view the presets' menu's line k names, or no longer, as favor says.
func (s *Sheet) star(k int) {
	if name, ok := PresetView(s, k); ok {
		s.favor(name, s.Views().Star(name))
	}
}

// favor keeps the favorites as given, as KeepFavorites does, a view starred, no longer, or moved among them, saying
// what it did to the view, or why it could not.
func (s *Sheet) favor(name string, starred []string) {
	did := Favored(s.Views(), name, starred)
	if err := KeepFavorites(s.path, s.store, starred); err != nil {
		s.note.Text, s.note.Done = fmt.Sprintf("%s the preset could not be %s: %v", Failed, did, err), false
		return
	}
	s.starred = starred
	s.note = s.note.Did(Starring + " " + did + " the preset " + name)
}

// Followed is what a sheet's browse gives for a link the screen follows itself, as to another of its rows: the sheet
// then says nothing of it.
var Followed = errors.New("the screen follows the link")

// SetBrowse sets what opens a link a click falls on: the person's opener, shell.Browse, unless a test records it, or
// the screen, which gives Followed for a link it follows itself.
func (s *Sheet) SetBrowse(browse func(string) error) { s.browse = browse }

// open opens a link, a page or a file, as the person's opener does, and says so in the notice; where no opener can be
// asked, as over SSH, the notice holds the link instead, to open by hand; one the screen follows, nothing.
func (s *Sheet) open(link string) {
	switch err := s.browse(link); {
	case errors.Is(err, Followed):
		return
	case err != nil:
		s.note.Text = fmt.Sprintf("%s no opener here; open it by hand: %s", Failed, link)
		return
	}
	s.note = s.note.Did("opened " + link)
}
