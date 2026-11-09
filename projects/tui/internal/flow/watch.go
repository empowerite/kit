// Watch: a stage watched loads again once it is stale, as its mark says, without waiting for Refresh All. Watch, in
// the tab's menu, watches the selected stage, and reads Stop Watching while it is watched; Watch All, in the app's
// menu under Refresh All, watches every stage, and is dim once every one is; Stop Watching All, after it, stops them
// all, and is dim while none is. The keys bar's right end says watching while the selected stage is watched. Nothing
// is watched at the start. A stage that loads again keeps its rows on the screen, and its dot, until the new ones
// come; it reads the forge afresh, as the change its mark says of is newer than the snapshot the stages share.
package flow

import (
	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// The watch items' names and glyphs: 🔭 to watch, and 🛑 to stop.
const (
	WatchIt     = "Watch"
	StopIt      = "Stop Watching"
	WatchAll    = "Watch All"
	StopAll     = "Stop Watching All"
	RefreshAll  = "Refresh All"
	Unwatched   = "🔭"
	StopWatched = "🛑"
)

// watchItem gives the tab's menu's line for the stage at i: Watch, or Stop Watching while it is watched.
func (m Model) watchItem(i int) shell.Item {
	if i < len(m.stages) && m.stages[i].src.Watch {
		return shell.Item{Glyph: StopWatched, Name: StopIt, Keys: "w"}
	}
	return shell.Item{Glyph: Unwatched, Name: WatchIt, Keys: "w"}
}

// watching gives how many of the stages kittui draws are watched, and how many it draws.
func (m Model) watching() (watched, all int) {
	for _, s := range m.stages {
		if known[s.name] {
			all++
			if s.src.Watch {
				watched++
			}
		}
	}
	return watched, all
}

// ownItems gives the picture's own lines of the app's menu, Refresh All, Watch All, and Stop Watching All, and the
// names of those that do nothing now: Watch All once every stage is watched, Stop Watching All while none is.
func (m Model) ownItems() (items []shell.Item, dim []string) {
	items = []shell.Item{{Glyph: Reload, Name: RefreshAll}, {Glyph: Unwatched, Name: WatchAll},
		{Glyph: StopWatched, Name: StopAll}}
	watched, all := m.watching()
	if watched == all {
		dim = append(dim, WatchAll)
	}
	if watched == 0 {
		dim = append(dim, StopAll)
	}
	return items, dim
}

// syncOwn gives the app's menu the picture's own lines as they stand now.
func (m *Model) syncOwn() { m.appMenu.Own, m.appMenu.Dim = m.ownItems() }

// toggleWatch watches the stage at i, or stops; a stage watched while it is stale loads again at once.
func (m *Model) toggleWatch(i int) tea.Cmd {
	s := &m.stages[i]
	s.src.Watch = !s.src.Watch
	m.syncOwn()
	if s.src.Watch && s.src.Stale && !s.src.Loading {
		return m.rewatch(s.name)
	}
	return nil
}

// watchAll watches every stage kittui draws, or, with on false, stops them all; each stale stage watched loads again
// at once.
func (m *Model) watchAll(on bool) tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.stages {
		s := &m.stages[i]
		if !known[s.name] {
			continue
		}
		if on && !s.src.Watch && s.src.Stale && !s.src.Loading {
			cmds = append(cmds, m.rewatch(s.name))
		}
		s.src.Watch = on
	}
	m.syncOwn()
	return tea.Batch(cmds...)
}

// watchedText gives what the keys bar's right end says of Watch: watching, in the accent, while the selected stage is
// watched; nothing while it is not.
func (m Model) watchedText() string {
	if len(m.stages) == 0 || !m.stages[m.stage].src.Watch {
		return ""
	}
	return style.Tint(Unwatched + " watching")
}

// rewatch loads a stale stage again, reading the forge afresh, its rows and its dot staying until the new ones come.
func (m *Model) rewatch(name string) tea.Cmd { return m.again(name, true) }
