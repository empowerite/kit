// Every step's screen at 120×30, the design width, but customize's, which customize_test.go pins with its others, is
// pinned as a golden file without its escapes; so are the introduction with its first failed check selected, the first
// step with checks, as the introduction's first failed check opens it, with its failed check left for later, and the
// terms searched for kit flow, the rows with the keyboard.
package welcome

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

func TestGoldenViews(t *testing.T) {
	for file, keys := range map[string][]tea.Msg{
		"introduction.golden":          nil,
		"introduction-selected.golden": {down, down, down, press(' ')},
		"toolchains.golden":            {down, down, down, enter},
		"toolchains-later.golden":      {down, down, down, enter, press('l')},
		"setup.golden":                 {tabKey, tabKey, down},
		"terms.golden":                 {stepKey(terms)},
		"terms-search.golden":          searched("kit flow"),
	} {
		m, _ := loaded(t, keys...)
		testkit.Golden(t, filepath.Join("testdata", file), view(m))
	}
}

// searched gives the keys that select the terms and search them for the text, then give their rows the keyboard.
func searched(text string) []tea.Msg {
	return append([]tea.Msg{stepKey(terms), press('/')}, append(testkit.Typed(text), enter)...)
}
