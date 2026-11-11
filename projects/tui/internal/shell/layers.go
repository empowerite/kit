// Layers are the controls a screen opens over itself, as its menus, dialogs, and popovers, in one order, the topmost
// first: the one order the keyboard, the clicks, the keys bar, and the drawing follow. While a layer is open, the
// topmost open one takes every key and every click; the keys bar names its keys alone; and the open layers are drawn
// over the screen's lines bottom up, so the topmost stands over the others. A screen keeps no layer of its own: each is
// what the screen's model says of it, by the functions the screen gives it, so the order is a table the screen writes
// once.
package shell

import tea "charm.land/bubbletea/v2"

// Layer is one control a screen opens over itself, as the functions of the screen's model M that say what it does:
// whether it is open; what it does with a key while it is open; what it does with a click, and whether it takes it,
// as a menu takes a left click and leaves the screen the others, a nil Click taking every click and doing nothing, as
// a dialog that asks before anything else happens; the keys the keys bar names for it; and how it draws itself over
// the screen's lines.
type Layer[M any] struct {
	Open  func(M) bool
	Key   func(M, tea.KeyPressMsg) (M, tea.Cmd)
	Click func(M, tea.MouseClickMsg) (M, tea.Cmd, bool)
	Keys  func(M) Segment
	Draw  func(M, []string) []string
}

// Layers is a screen's layers, the topmost first.
type Layers[M any] []Layer[M]

// top gives the index of the topmost open layer, -1 while none is.
func (ls Layers[M]) top(m M) int {
	for k, l := range ls {
		if l.Open(m) {
			return k
		}
	}
	return -1
}

// Opened says whether any layer is open.
func (ls Layers[M]) Opened(m M) bool { return ls.top(m) >= 0 }

// Key gives a key to the topmost open layer, and says whether one took it; with none open, the screen takes it.
func (ls Layers[M]) Key(m M, k tea.KeyPressMsg) (M, tea.Cmd, bool) {
	at := ls.top(m)
	if at < 0 {
		return m, nil, false
	}
	m, cmd := ls[at].Key(m, k)
	return m, cmd, true
}

// Click gives a click to the topmost open layer, and says whether it took it, on the layer or off it; with none open,
// or one that leaves the click, the screen takes it.
func (ls Layers[M]) Click(m M, c tea.MouseClickMsg) (M, tea.Cmd, bool) {
	at := ls.top(m)
	if at < 0 {
		return m, nil, false
	}
	if ls[at].Click == nil {
		return m, nil, true
	}
	return ls[at].Click(m, c)
}

// Keys gives the keys bar's path while a layer is open, the topmost's keys alone, and says whether one is.
func (ls Layers[M]) Keys(m M) (Path, bool) {
	at := ls.top(m)
	if at < 0 {
		return nil, false
	}
	return Path{ls[at].Keys(m)}, true
}

// Draw draws every open layer over the screen's lines, bottom up.
func (ls Layers[M]) Draw(m M, lines []string) []string {
	for k := len(ls) - 1; k >= 0; k-- {
		if ls[k].Open(m) {
			lines = ls[k].Draw(m, lines)
		}
	}
	return lines
}
