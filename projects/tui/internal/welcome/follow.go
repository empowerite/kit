// What keeps the setup step's table in step with the screen: its sheet, made once kit names the folders and gone when
// it no longer does; its records, drawn again where a check, a note, the cursor, or the keyboard changed them; the
// pane it is placed in under its lead; and the check the cursor's row stands for, which c, r, and k, and the buttons
// act on: the first of the row's checks that has a fix, else the first that failed, else the first.
package welcome

import (
	"reflect"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/fix"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// setupIndex gives the index of the setup step.
func (m Model) setupIndex() int { return indexOf(m.tabs, setupID) }

// followSheet keeps the check the setup table's cursor row stands for in m.cursor, and the table as the screen last
// drew its records: made when kit names the folders, dropped when it no longer does, its records set where they
// changed, and placed under its lead where that moved.
func (m *Model) followSheet() {
	i := m.setupIndex()
	if i < 0 {
		return
	}
	t := &m.tabs[i]
	switch want := m.foldered(i); {
	case want && t.sheet == nil:
		sheet := table.NewSheet(foldersCatalog[foldersTable], "folders", sheetColor, m.presets)
		sheet.Settings(m.settings)
		t.sheet, t.records = &sheet, nil
	case !want && t.sheet != nil:
		t.sheet, t.records = nil, nil
		return
	case !want:
		return
	}
	if m.step == i {
		m.cursor = m.rowCheck(i)
	}
	records := m.setupRecords(i)
	if !reflect.DeepEqual(records, t.records) {
		t.sheet.SetRecords(records)
		t.records = records
		for _, r := range records {
			if _, set := t.sheet.Table().Open[r.Key]; !set && len(r.Detail) > 0 {
				t.sheet.Table().Open[r.Key] = true
			}
		}
	}
	n := len(m.leadLines(i))
	if place := [4]int{m.paneTop() + n, m.paneHeight() - n, m.width, m.height}; place != t.placed {
		t.placed = place
		t.sheet.Place(place[0], place[1], place[2], place[3])
	}
}

// rowCheck gives the check the setup table's cursor row stands for, as the file says; the check the cursor was on where
// the cursor is on no row of a folder or a machine check.
func (m Model) rowCheck(step int) int {
	sheet := m.tabs[step].sheet
	rows, at := sheet.Rows(), sheet.Table().Cursor
	if at < 0 || at >= len(rows) {
		return m.cursor
	}
	var mine []int
	for i, c := range m.checks {
		if m.stepOf(c) == step && !folded(c) && (c.Folder == rows[at].Key || c.Folder == "" && c.Key() == rows[at].Key) {
			mine = append(mine, i)
		}
	}
	for _, ok := range []func(kit.Check) bool{func(c kit.Check) bool { return c.Fix != "" },
		func(c kit.Check) bool { return !c.OK }, func(kit.Check) bool { return true }} {
		if k := slices.IndexFunc(mine, func(i int) bool { return ok(m.checks[i]) }); k >= 0 {
			return mine[k]
		}
	}
	return m.cursor
}

// focused says whether the keyboard is on a check's row: in a step drawn as a page or rows, the pane's; in the setup
// table, its rows'.
func (m Model) focused() bool {
	if m.gridded() && m.step == m.setupIndex() {
		return m.inPane && m.sheet().InRows()
	}
	return m.inPane
}

// pressed takes a click on a button of the current check of the setup table, drawn where the click is: ⧉ copies the
// fix, ▶️ runs it, keep keeps a terminal's finding. It says whether it was one. Only the row under the cursor has
// buttons, so a click that selects a row is no press.
func (m *Model) pressed(msg tea.MouseClickMsg) (tea.Cmd, bool) {
	if m.step != m.setupIndex() || msg.Button != tea.MouseLeft || !m.sheet().InRows() {
		return nil, false
	}
	row, _, _, ok := table.RowAt(m.sheet(), msg.X, msg.Y)
	view := m.sheetPane()
	at := msg.Y - m.paneTop()
	if !ok || row != m.sheet().Table().Cursor || at < 0 || at >= len(view) {
		return nil, false
	}
	plain := ansi.Strip(view[at])
	switch {
	case m.keeping() && onButton(plain, keepName, msg.X):
		return m.keep(), true
	case m.copies() && fix.On(plain, msg.X) == fix.Copy:
		return m.copy(), true
	case m.runsHere() && fix.On(plain, msg.X) == fix.Run:
		return m.runFix(), true
	}
	return nil, false
}

// setupLead gives the lines the setup table has over it: its page, kit's refusal, and, while the checks run again,
// what they are on.
func (m Model) setupLead() []string {
	var out []string
	for _, l := range m.stepHead() {
		out = append(out, l.text)
	}
	if m.checking.Running() {
		add := func(_ int, text string) { out = append(out, text) }
		m.addChecking(add)
		out = append(out, "")
	}
	return out
}
