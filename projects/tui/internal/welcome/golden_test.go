// Every step's screen at 120×30, the design width, but customize's, which customize_test.go pins with its others, is
// pinned as a golden file without its escapes; so are the introduction with the cursor on its second link, the first
// step with checks, as its first link opens it, with its failed check left for later, and the terms searched for kit
// flow, the rows with the keyboard.
package welcome

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

func TestGoldenViews(t *testing.T) {
	for file, keys := range map[string][]tea.Msg{
		"introduction.golden":       nil,
		"introduction-links.golden": {down, down},
		"toolchains.golden":         {down, enter},
		"toolchains-later.golden":   {down, enter, press('l')},
		"setup.golden":              {tabKey, tabKey, down},
		"applications.golden":       {tabKey, tabKey, tabKey},
		"terms.golden":              {press('5')},
		"terms-search.golden":       searched("kit flow"),
	} {
		m, _ := loaded(t, keys...)
		testkit.Golden(t, filepath.Join("testdata", file), view(m))
	}
}

// searched gives the keys that select the terms and search them for the text, then give their rows the keyboard.
func searched(text string) []tea.Msg {
	return append([]tea.Msg{press('5'), press('/')}, append(testkit.Typed(text), enter)...)
}
