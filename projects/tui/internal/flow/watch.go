// Watch: a stage watched loads again once it is stale, as its mark says, without waiting for Refresh All. Watch, in
// the tab's menu, watches the selected stage, and reads Stop Watching while it is watched; Watch All, in the app's
// menu under Refresh All, watches every stage, and is dim once every one is; Stop Watching All, after it, stops them
// all, and is dim while none is. The keys bar's right end says watching while the selected stage is watched. Nothing
// is watched at the start. A stage that loads again keeps its rows on the screen, and its dot, until the new ones
// come; it reads the forge afresh, as the change its mark says of is newer than the snapshot the stages share. A stage
// kit gives a null mark, as kit people gives each of its own, never goes stale, so it offers no Watch: Watch is dim on
// its tab's menu, and Watch All passes it by.
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
		return shell.Item{Glyph: StopWatched, Name: StopIt, Keys: "W"}
	}
	return shell.Item{Glyph: Unwatched, Name: WatchIt, Keys: "W"}
}

// watchable says whether the stage at i can be watched: kittui draws it, and kit's last read of the marks gave it
// one, or none yet; a stage kit gives a null mark never goes stale, so Watch would wait on nothing.
func (m Model) watchable(i int) bool {
	mark, read := m.marks.Value[m.stages[i].name]
	return sourced(m.stages[i].name) && (!read || mark != "")
}

// watching gives how many of the stages kittui draws are watched, and how many it can watch, as watchable says, or
// watches already.
func (m Model) watching() (watched, all int) {
	for i, s := range m.stages {
		if m.watchable(i) || s.src.Watch {
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

// watchAll watches every stage it can, as watchable says, or, with on false, stops them all; each stale stage watched
// loads again at once.
func (m *Model) watchAll(on bool) tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.stages {
		s := &m.stages[i]
		if on && !m.watchable(i) || !sourced(s.name) {
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
