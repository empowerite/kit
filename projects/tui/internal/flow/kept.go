// The last picture: the screen opens on what each stage printed last, as kit flow <stage> --json --cached keeps it,
// while its own load runs. A stage drawn from what was kept is stale, a dot on its tab, until its load lands; a load
// that lands first leaves what was kept unread.
package flow

import (
	"context"
	"slices"

	tea "charm.land/bubbletea/v2"
)

// keptMsg is what kit kept of a stage's last JSON: the document, or why there is none, for the stage's load that gen
// names.
type keptMsg struct {
	stage string
	data  []byte
	err   error
	gen   int
}

// keptAll gives, for every stage kittui draws, the ask of kit for what it kept of the stage's last JSON, with the words
// its load runs with, so it is that load's.
func (m Model) keptAll() []tea.Cmd {
	var cmds []tea.Cmd
	for _, s := range m.stages {
		if known[s.name] {
			cmds = append(cmds, m.kept(s.name))
		}
	}
	return cmds
}

// kept asks kit for what it kept of one stage's last JSON.
func (m Model) kept(name string) tea.Cmd {
	r, args, ctx, limit, gen := m.runner, append(slices.Clone(m.args(name)), "--cached"), m.ctx, m.limit, 0
	if s := m.find(name); s != nil {
		gen = s.gen
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		data, err := r.Status(ctx, name, args)
		return keptMsg{stage: name, data: data, err: err, gen: gen}
	}
}

// takeKept draws a stage from what kit kept, stale, while its load runs: not once the load has landed, nor after it
// began again; a stage kit kept nothing of, or kept what no longer decodes, goes on loading, and its load replaces
// whatever the decode left.
func (m *Model) takeKept(msg keptMsg) {
	s := m.find(msg.stage)
	if s == nil || msg.gen != s.gen || !s.loading || msg.err != nil || s.decode(msg.data) != nil {
		return
	}
	s.loading, s.stale = false, true
	m.rebuild()
	for i := range m.stages {
		m.clamp(i)
	}
	m.fit()
}
