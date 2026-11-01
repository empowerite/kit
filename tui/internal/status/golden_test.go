// Every stage's screen at 80×24, closed and with its first row open, and the legend, are pinned as golden files without
// their escapes.
package status

import (
	"path/filepath"
	"testing"

	"github.com/empowerite/kit/tui/internal/testkit"
)

func TestGoldenViews(t *testing.T) {
	for i, name := range five {
		m := loaded(t, options())
		for range i {
			m = drive(m, right)
		}
		testkit.Golden(t, filepath.Join("testdata", name+".golden"), view(m))
		m = drive(m, down, enter)
		testkit.Golden(t, filepath.Join("testdata", name+"-open.golden"), view(m))
	}
	testkit.Golden(t, filepath.Join("testdata", "legend.golden"), view(loaded(t, options(), press('?'))))
}
