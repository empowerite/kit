// The picture's part in the crumbs, as internal/shell's crumb.go says them: where it stands, which kittui's root keeps
// as the crumb of the app a link of the picture leads to, and puts back once the crumb is followed; the crumb it is
// told, which the title draws last; and whether a click or a key is the way back.
package flow

import (
	"maps"
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// spot is where the picture stands, as a crumb keeps it: the tab selected, by its stage's name, and the bar's view;
// whether it is maximized; what has the keyboard; and each stage's table, its preset, sort, filters, groups, outline,
// rows selected, cursor, and scroll, as they were, and the key of the row under its cursor, which the cursor goes back
// to wherever the row stands now.
type spot struct {
	stage                     string
	barView                   int
	maximized                 bool
	inRows, inPanel, onPreset bool
	stop                      int
	tables                    map[string]table.Table
	under                     map[string]string
}

// Spot gives the name of where the picture stands, the app and the tab selected, the tab alone where the app is named
// after it; and the spot, as Restore puts it back.
func (m Model) Spot() (string, any) {
	name := string(m.app)
	if m.app == "" {
		name = "flow"
	}
	p := spot{barView: m.barView, maximized: m.maximized, inRows: m.inRows, inPanel: m.inPanel,
		onPreset: m.onPreset, stop: m.stop, tables: map[string]table.Table{}, under: map[string]string{}}
	for i, s := range m.stages {
		p.tables[s.name] = cloneTable(s.Table)
		if rows := m.rows(i); s.Cursor >= 0 && s.Cursor < len(rows) {
			p.under[s.name] = rows[s.Cursor].Key
		}
	}
	if len(m.stages) > 0 {
		p.stage = m.stages[m.stage].name
		if tab := label(p.stage); tab != name {
			name += " › " + tab
		}
	}
	return name, p
}

// Restore puts the picture back where a spot says it stood: each stage it had, its table as it was, the cursor on the
// row it was on, or where it was when the row is gone; the tab selected, maximized or not, and what had the keyboard.
// A jump still waiting for its row is dropped.
func (m Model) Restore(v any) tea.Model {
	p, ok := v.(spot)
	if !ok {
		return m
	}
	for i := range m.stages {
		if t, ok := p.tables[m.stages[i].name]; ok {
			m.stages[i].Table = t
		}
	}
	if i := slices.IndexFunc(m.stages, func(s stage) bool { return s.name == p.stage }); i >= 0 {
		m.stage = i
	}
	m.barView, m.maximized, m.inRows, m.inPanel, m.onPreset, m.stop =
		p.barView, p.maximized, p.inRows, p.inPanel, p.onPreset, p.stop
	m.landing = ""
	m.rebuild()
	for i := range m.stages {
		key, ok := p.under[m.stages[i].name]
		if k := slices.IndexFunc(m.rows(i), func(r row) bool { return r.Key == key }); ok && k >= 0 {
			m.stages[i].Cursor = k
		}
		m.clamp(i)
	}
	m.fit()
	return m
}

// Crumbed gives the picture told its crumb, by the name of where it goes back to, none for "".
func (m Model) Crumbed(name string) tea.Model {
	m.appMenu.Crumb = name
	return m
}

// OnCrumb says whether a click is on the crumb, a left one, with nothing open over the title.
func (m Model) OnCrumb(c tea.MouseClickMsg) bool {
	return c.Button == tea.MouseLeft && !m.layers().Opened(m) && m.menuBar().OnCrumb(c.X, c.Y, m.width)
}

// Backs says whether a key goes back to the crumb: ⌫, while a crumb shows, and the rows, the bar, or the preset line
// have the keyboard, not the search, nor a view's name being typed, nor anything open over the picture.
func (m Model) Backs(k tea.KeyPressMsg) bool {
	naming := len(m.stages) > 0 && m.onPreset && m.stages[m.stage].SavingAs
	return k.String() == shell.BackPress && m.appMenu.Crumb != "" && !m.layers().Opened(m) && !m.inPanel && !naming
}

// cloneTable gives a copy of a stage's table that shares nothing a later change to the table writes into.
func cloneTable(t table.Table) table.Table {
	t.Open = maps.Clone(t.Open)
	t.Picked = maps.Clone(t.Picked)
	t.View = clonePanel(t.View)
	if t.Grouped != nil {
		g := *t.Grouped
		g.Fields = maps.Clone(g.Fields)
		t.Grouped = &g
	}
	return t
}

// clonePanel gives a copy of a panel that shares nothing a later change to the panel writes into. Its Given is never
// a stage's: table.Build fills in its own copy of the panel, from the records.
func clonePanel(p grid.Panel) grid.Panel {
	p.Columns = slices.Clone(p.Columns)
	p.Labels, p.Sortable, p.Groupable, p.Ones = maps.Clone(p.Labels), maps.Clone(p.Sortable),
		maps.Clone(p.Groupable), maps.Clone(p.Ones)
	p.Outline = maps.Clone(p.Outline)
	if p.Fields != nil {
		fields := make(map[string]grid.Field, len(p.Fields))
		for id, f := range p.Fields {
			f.Filter.Values = slices.Clone(f.Filter.Values)
			fields[id] = f
		}
		p.Fields = fields
	}
	return p
}
