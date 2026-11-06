// Every stage's screen at 80×24, closed and with its first row open, is pinned as a golden file without its escapes,
// its preset line over its table; underway's and builds' are at 120×24, the common width. So is the bar at 120 columns,
// with the legend and without, and the legend at 80, scrolled to changes; and backlog's preset line once its table
// differs from every view, filtered to the others' issues and to those not on GitLab, under the keyboard.
package flow

import (
	"maps"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

func TestGoldenViews(t *testing.T) {
	at80 := tea.WindowSizeMsg{Width: 80, Height: 24}
	for i, name := range eight {
		size := at80
		if name == "underway" || name == "builds" {
			size.Width = 120
		}
		m := loaded(t, viewing(""), size)
		for range i {
			m = drive(m, right)
		}
		testkit.Golden(t, filepath.Join("testdata", name+".golden"), raw(m))
		// down reaches the preset line first, where the stage has one, and the rows after it
		if m = drive(m, down); !m.inRows {
			m = drive(m, down)
		}
		m = drive(m, toggle)
		testkit.Golden(t, filepath.Join("testdata", name+"-open.golden"), raw(m))
	}
	golden := func(file string, msgs ...tea.Msg) {
		testkit.Golden(t, filepath.Join("testdata", file), raw(loaded(t, viewing(""), msgs...)))
	}
	golden("legend.golden", at80, press('?'))
	golden("bar-120.golden", tea.WindowSizeMsg{Width: 120, Height: 24})
	golden("legend-120.golden", tea.WindowSizeMsg{Width: 120, Height: 32}, press('?'))
	toChanges := []tea.Msg{press('?'), right, right, right, right}
	golden("legend-120-changes.golden", append([]tea.Msg{tea.WindowSizeMsg{Width: 120, Height: 32}}, toChanges...)...)
	golden("legend-80-changes.golden", append([]tea.Msg{tea.WindowSizeMsg{Width: 80, Height: 32}}, toChanges...)...)
	golden("legend-200.golden", tea.WindowSizeMsg{Width: 200, Height: 32}, press('?'))
	m := loaded(t, viewing(""), at80)
	p := m.panel(0).Shown(grid.Others)
	p.Fields = maps.Clone(p.Fields)
	source := p.Fields["source"]
	source.Filter = grid.Filter{Values: []string{"gitlab"}, Exclude: true}
	p.Fields["source"] = source
	m.setPanel(p)
	testkit.Golden(t, filepath.Join("testdata", "preset-line.golden"), raw(drive(m, down)))
}
