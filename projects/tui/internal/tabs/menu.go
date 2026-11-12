// The tab's menu every screen with tabs drops down from the selected tab's :, its items' glyphs and the items each
// screen's menu shares: the tab maximized and restored, its source read again, the outline of a tab drawn as a table
// shown a level more and less, and its rows selected, every one and none.
package tabs

import (
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// The glyphs beside the tab's menu's items: the tab maximized and restored, one cell each; its source read again; a
// level more of its table shown and a level less, the selected rows shown, all the way down to them, and hidden; and
// every row selected, the selection swept away beside it as table.Sweep.
const (
	Maximize = "⤢"
	Restore  = "⤡"
	Reload   = "🔄"
	More     = "🔽"
	Less     = "🔼"
	Expand   = "⏬"
	Collapse = "⏫"
	Every    = "📋"
)

// SizeItem gives the tab's menu's item that maximizes the tab, or, maximized, restores it: M, either way.
func SizeItem(maximized bool) shell.Item {
	if maximized {
		return shell.Item{Glyph: Restore, Name: "Restore", Keys: "M"}
	}
	return shell.Item{Glyph: Maximize, Name: "Maximize", Keys: "M"}
}

// FoldItems gives the tab's menu's items that show and hide a table's outline, as table.Folding names them: a level
// more of the whole table and a level less, as > and < do, and the selected rows shown, every heading over them too,
// and hidden.
func FoldItems() []shell.Item {
	return []shell.Item{{Glyph: More, Name: "Show More", Keys: ">"}, {Glyph: Less, Name: "Show Less", Keys: "<"},
		{Glyph: Expand, Name: "Show Selected"}, {Glyph: Collapse, Name: "Hide Selected"}}
}

// SelectItems gives the tab's menu's items that select a table's rows: every one, as ctrl+a does, and none.
func SelectItems() []shell.Item {
	return []shell.Item{{Glyph: Every, Name: "Select All", Keys: "⌃A"}, {Glyph: table.Sweep, Name: "Clear Selection"}}
}
