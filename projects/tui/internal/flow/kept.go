// The last picture: the screen opens on what each stage printed last, as kit flow <stage> --json --cached keeps it,
// while its own load runs. A stage drawn from what was kept is stale, a dot on its tab, until its load lands; a load
// that lands first leaves what was kept unread. Each stage's read, and what kit kept of it, is a source.Load.
package flow

import (
	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/source"
)

// keptAll gives, for every stage kittui draws, the ask of kit for what it kept of the stage's last JSON, for its first
// load.
func (m Model) keptAll() []tea.Cmd {
	var cmds []tea.Cmd
	for _, s := range m.stages {
		if known[s.name] {
			cmds = append(cmds, s.src.Kept(m.env()))
		}
	}
	return cmds
}

// kept asks kit for what it kept of one stage's last JSON.
func (m Model) kept(name string) tea.Cmd { return m.find(name).src.Kept(m.env()) }

// takeKept draws a stage from what kit kept, stale, while its first load runs, as source.Load lets it stand: not once
// the load has landed, nor after it began again; a stage kit kept nothing of, or kept what no longer decodes, goes on
// loading, and its load replaces whatever the decode left.
func (m *Model) takeKept(msg source.Answer[[]byte]) {
	s := m.find(msg.Name)
	if s == nil {
		return
	}
	if _, landed := s.src.Take(msg); landed != source.Stands || s.decode(msg.Value) != nil {
		return
	}
	s.src.Loading, s.src.Stale = false, true
	m.rebuild()
	m.autoGroup(s.name)
	for i := range m.stages {
		m.clamp(i)
	}
	m.fit()
}
