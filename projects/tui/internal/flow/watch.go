// Watch: a stage watched loads again once it is stale, as its mark says, without waiting for Refresh All. Watch, in
// the tab's menu, watches the selected stage, or stops; Watch All, in the app's menu under Refresh All, watches every
// stage, or, once every one is watched, stops them all. Each says ✅ while it holds, and 🔭 while it does not. Nothing is
// watched at the start. A stage that loads again keeps its rows on the screen, and its dot, until the new ones come;
// it reads the forge afresh, as the change its mark says of is newer than the snapshot the stages share.
package flow

import (
	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// The watch items' names, and the glyph of one that does not hold; one that holds has Checked.
const (
	WatchIt    = "Watch"
	WatchAll   = "Watch All"
	RefreshAll = "Refresh All"
	Unwatched  = "🔭"
)

// watchItem gives the tab's menu's Watch, for the stage at i: Checked while it is watched.
func (m Model) watchItem(i int) shell.Item {
	if i < len(m.stages) && m.stages[i].watch {
		return shell.Item{Glyph: Checked, Name: WatchIt}
	}
	return shell.Item{Glyph: Unwatched, Name: WatchIt}
}

// watchingAll says whether every stage kittui draws is watched; none to watch is not watching them all.
func (m Model) watchingAll() bool {
	any := false
	for _, s := range m.stages {
		if !known[s.name] {
			continue
		}
		if !s.watch {
			return false
		}
		any = true
	}
	return any
}

// ownItems gives the picture's own lines of the app's menu: Refresh All, and Watch All, Checked while every stage is
// watched.
func (m Model) ownItems() []shell.Item {
	all := shell.Item{Glyph: Unwatched, Name: WatchAll}
	if m.watchingAll() {
		all.Glyph = Checked
	}
	return []shell.Item{{Glyph: Reload, Name: RefreshAll}, all}
}

// syncOwn gives the app's menu the picture's own lines as they stand now.
func (m *Model) syncOwn() { m.appMenu.Own = m.ownItems() }

// toggleWatch watches the stage at i, or stops; a stage watched while it is stale loads again at once.
func (m *Model) toggleWatch(i int) tea.Cmd {
	s := &m.stages[i]
	s.watch = !s.watch
	m.syncOwn()
	if s.watch && s.stale && !s.loading {
		return m.rewatch(s.name)
	}
	return nil
}

// watchAll watches every stage kittui draws, or, once every one is, stops them all; each stale stage watched loads
// again at once.
func (m *Model) watchAll() tea.Cmd {
	on := !m.watchingAll()
	var cmds []tea.Cmd
	for i := range m.stages {
		s := &m.stages[i]
		if !known[s.name] {
			continue
		}
		s.watch = on
		if on && s.stale && !s.loading {
			cmds = append(cmds, m.rewatch(s.name))
		}
	}
	m.syncOwn()
	return tea.Batch(cmds...)
}

// rewatch loads a stale stage again, reading the forge afresh, its rows and its dot staying until the new ones come.
func (m *Model) rewatch(name string) tea.Cmd { return m.again(name, true) }
