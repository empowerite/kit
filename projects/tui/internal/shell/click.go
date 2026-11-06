package shell

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// Bubble Tea reports a click's cell, its button, and the modifiers held, but not whether it is the second of a double
// click. Clicks tells.

// DoubleClick is the longest a second click may come after the first, on the same cell, to make a double click.
const DoubleClick = 400 * time.Millisecond

// Clicks tells a double click from two clicks: a left click on the cell the left click before it fell on, within
// DoubleClick of it. A double click starts over, so a third click is the first of the next pair; a click of another
// button forgets the one before. Now is the clock it reads, time.Now where it is nil, which a test sets.
type Clicks struct {
	Now   func() time.Time
	at    time.Time
	x, y  int
	armed bool
}

// Click takes a click, and says whether it is the second of a double click.
func (c Clicks) Click(msg tea.MouseClickMsg) (Clicks, bool) {
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	t := now()
	if msg.Button != tea.MouseLeft {
		c.armed = false
		return c, false
	}
	since := t.Sub(c.at)
	double := c.armed && msg.X == c.x && msg.Y == c.y && since >= 0 && since <= DoubleClick
	c.at, c.x, c.y, c.armed = t, msg.X, msg.Y, !double
	return c, double
}
