// Stale stages: the screen reads kit's mark of each stage, kit flow --marks, after a stage loads and once a minute
// after it first reads them. A stage that loaded takes the next mark as its own; a mark that then differs from it says
// what the stage draws has changed, so the stage is stale, a dot on its tab, until it loads again. A kit that cannot
// read the marks is asked again only on Refresh All. A change in the moments between a stage's load and the read
// after it is in neither, and shows with the next change.
package flow

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
)

// MarksEvery is how often the screen reads the marks again, once it has read them, while nothing loads.
const MarksEvery = time.Minute

// marksEvery is how long the screen waits for MarksEvery: MarksEvery, as a test waits less.
var marksEvery = MarksEvery

// marksMsg is kit's answer to the marks: each stage's, by its name, or why they could not be read.
type marksMsg struct {
	marks map[string]string
	err   error
}

// marksTickMsg says MarksEvery has passed since the read the serial names.
type marksTickMsg struct{ serial int }

// askMarks asks kit for the marks, unless a read is under way, or kit could not read them, until Refresh All.
func (m *Model) askMarks() tea.Cmd {
	if m.marking || m.marksOff {
		return nil
	}
	m.marking = true
	r, ctx, limit := m.runner, m.ctx, m.limit
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		marks, err := r.Marks(ctx)
		return marksMsg{marks, err}
	}
}

// tookMarks compares what kit said of each stage with the mark the stage took when it loaded, and has the screen read
// them again in MarksEvery. A stage that loaded since the last read takes its mark now; one still loading, or without a
// mark, is passed over. A stage whose mark moved is stale, and, watched, loads again at once. A read that fails changes
// nothing; the first to fail before any read stops the asking, until Refresh All.
func (m *Model) tookMarks(msg marksMsg) tea.Cmd {
	m.marking = false
	var cmds []tea.Cmd
	switch {
	case msg.err == nil:
		m.marked = true
		for i := range m.stages {
			s := &m.stages[i]
			mark, ok := msg.marks[s.name]
			switch {
			case !ok || s.loading || s.refusal != "":
			case s.mark == "" || s.remark:
				s.mark, s.remark = mark, false
			case mark != s.mark && s.watch:
				// a stage watched loads again, its rows and its dot staying until the new ones come; the load it starts
				// supersedes one already under way, so a mark that moves twice loads it twice at most
				s.stale = true
				s.mark = mark
				cmds = append(cmds, m.rewatch(s.name))
			case mark != s.mark:
				s.stale = true
			}
		}
	case !m.marked:
		m.marksOff = true
		return nil
	}
	m.marksSerial++
	serial := m.marksSerial
	return tea.Batch(append(cmds, tea.Tick(marksEvery, func(time.Time) tea.Msg { return marksTickMsg{serial} }))...)
}
