package tabs

import (
	"testing"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// The items every tab's menu shares: Maximize, or Restore once maximized, on M; the folds, as table.Folding names
// them, > and < on the levels; and Select All on ctrl+a, then Clear Selection.
func TestTheTabsMenusShareTheirItems(t *testing.T) {
	if got := SizeItem(false); got != (shell.Item{Glyph: Maximize, Name: "Maximize", Keys: "M"}) {
		t.Errorf("not maximized: %+v", got)
	}
	if got := SizeItem(true); got != (shell.Item{Glyph: Restore, Name: "Restore", Keys: "M"}) {
		t.Errorf("maximized: %+v", got)
	}
	folds := FoldItems()
	want := []shell.Item{{Glyph: More, Name: "Show More", Keys: ">"}, {Glyph: Less, Name: "Show Less", Keys: "<"},
		{Glyph: Expand, Name: "Show Selected"}, {Glyph: Collapse, Name: "Hide Selected"}}
	if len(folds) != len(want) {
		t.Fatalf("folds %+v", folds)
	}
	for k := range want {
		if folds[k] != want[k] {
			t.Errorf("fold %d: %+v, not %+v", k, folds[k], want[k])
		}
	}
	selects := SelectItems()
	if len(selects) != 2 || selects[0] != (shell.Item{Glyph: Every, Name: "Select All", Keys: "⌃A"}) ||
		selects[1] != (shell.Item{Glyph: table.Sweep, Name: "Clear Selection"}) {
		t.Errorf("selects %+v", selects)
	}
}
