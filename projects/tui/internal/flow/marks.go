// Stale stages: the screen reads kit's mark of each stage, kit flow --marks, after a stage loads and once a minute
// after it first reads them. A stage that loaded takes the next mark as its own; a mark that then differs from it says
// what the stage draws has changed, so the stage is stale, a dot on its tab, until it loads again. A stage kit gives a
// null mark, as kit people gives each of its own, never is, and offers no Watch, as watch.go says. A kit that cannot
// read the marks is asked again only on Refresh All. A change in the moments between a stage's load and the read
// after it is in neither, and shows with the next change. The marks are a source.Poll.
package flow

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/source"
)

// MarksEvery is how often the screen reads the marks again, once it has read them.
const MarksEvery = time.Minute

// tookMarks compares what kit said of each stage with the mark the stage took when it loaded, once a read of the marks
// answered, the wait for the next, wait, batched with what it starts. A stage that loaded since the last read takes
// its mark now; one still loading, or without a mark, kit's null among them, is passed over. A stage whose mark moved
// is stale, and, watched, loads again at once. What Watch All does follows the marks, as watch.go says.
func (m *Model) tookMarks(wait tea.Cmd, took source.Took) tea.Cmd {
	if took != source.Answered {
		return wait
	}
	m.syncOwn()
	cmds := []tea.Cmd{wait}
	for i := range m.stages {
		s := &m.stages[i]
		mark, ok := m.marks.Value[s.name]
		if ok && mark != "" && s.refusal == "" && s.src.Marked(mark) {
			// a stage watched loads again, its rows and its dot staying until the new ones come; the load it starts
			// supersedes one already under way, so a mark that moves twice loads it twice at most
			cmds = append(cmds, m.rewatch(s.name))
		}
	}
	return tea.Batch(cmds...)
}
