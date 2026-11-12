// Every step's screen at 120×30, the design width, but customize's, which customize_test.go pins with its others, is
// pinned as a golden file without its escapes: the introduction, its page alone; the toolchains with the cursor on
// their first failed check; setup with the cursor on its first row; the terms; and the terms searched for kit flow,
// the rows with the keyboard.
package welcome

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// The screens the file says match their golden files.
func TestGoldenViews(t *testing.T) {
	for file, keys := range map[string][]tea.Msg{
		"introduction.golden": nil,
		"toolchains.golden":   toChains,
		"setup.golden":        {tabKey, tabKey, down},
		"terms.golden":        {stepKey(terms)},
		"terms-search.golden": searched("kit flow"),
	} {
		m, _ := loaded(t, keys...)
		testkit.Golden(t, filepath.Join("testdata", file), view(m))
	}
}

// searched gives the keys that select the terms and search them for the text, then give their rows the keyboard.
func searched(text string) []tea.Msg {
	return append([]tea.Msg{stepKey(terms), press('/')}, append(testkit.Typed(text), enter)...)
}
