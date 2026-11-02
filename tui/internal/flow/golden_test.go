// Every stage's screen at 80×24, closed and with its first row open, is pinned as a golden file without its escapes;
// underway's and builds' are at 120×24, the common width. So are the trees at 120 columns, with the legend and
// without, and the legend at 80, scrolled to changes.
package flow

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/tui/internal/testkit"
)

func TestGoldenViews(t *testing.T) {
	at80 := tea.WindowSizeMsg{Width: 80, Height: 24}
	for i, name := range eight {
		size := at80
		if name == "underway" || name == "builds" {
			size.Width = 120
		}
		m := loaded(t, options(), size)
		for range i {
			m = drive(m, right)
		}
		testkit.Golden(t, filepath.Join("testdata", name+".golden"), view(m))
		m = drive(m, down, enter)
		testkit.Golden(t, filepath.Join("testdata", name+"-open.golden"), view(m))
	}
	golden := func(file string, msgs ...tea.Msg) {
		testkit.Golden(t, filepath.Join("testdata", file), view(loaded(t, options(), msgs...)))
	}
	golden("legend.golden", at80, press('?'))
	golden("trees-120.golden", tea.WindowSizeMsg{Width: 120, Height: 24})
	golden("legend-120.golden", tea.WindowSizeMsg{Width: 120, Height: 32}, press('?'))
	toChanges := []tea.Msg{press('?'), right, right, right, right, right}
	golden("legend-120-changes.golden", append([]tea.Msg{tea.WindowSizeMsg{Width: 120, Height: 32}}, toChanges...)...)
	golden("legend-80-changes.golden", append([]tea.Msg{tea.WindowSizeMsg{Width: 80, Height: 32}}, toChanges...)...)
	golden("legend-200.golden", tea.WindowSizeMsg{Width: 200, Height: 32}, press('?'))
}
