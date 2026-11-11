// The screen at 120×30, the design size, is pinned as golden files without its escapes: every tutorial, the cards
// grouped By audience, the first card opened; the screen at 60×20; and a catalogue with no tutorial.
package tutorials

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

func TestGoldenViews(t *testing.T) {
	for file, keys := range map[string][]tea.Msg{
		"tutorials.golden":         nil,
		"tutorials-grouped.golden": {up, tabKey, enter, down, enter},
		"tutorials-opened.golden":  {enter},
		"tutorials-narrow.golden":  {at60},
	} {
		m, _ := loaded(t, keys...)
		testkit.Golden(t, filepath.Join("testdata", file), view(m))
	}
	catalogueFile(t, "empty.json")
	m, _ := loaded(t)
	testkit.Golden(t, filepath.Join("testdata", "tutorials-empty.golden"), view(m))
}
