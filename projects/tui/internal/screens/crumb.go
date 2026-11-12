// The root keeps the crumbs, as internal/shell's crumb.go says them, for the session: the screen shown has a crumb
// when a link led to it, the place the link was followed from, which holds that place's own crumb in turn, a chain
// each one deep. A link, or Manage…, leaves the screen shown, as it stands then, as the crumb of the one it shows; the
// crumb followed, by a click on it or by ⌫, shows its place again, put back as the crumb kept it, and that place's own
// crumb with it; the app's menu shows a screen with no crumb, and drops the chain; and so does a move to another
// workspace, which makes anew the screens the chain's places stood on.
package screens

import (
	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// crumb is a place a link was followed from: the screen; its name, as the title draws it; where it stood, as the
// screen says it, nothing for a screen that keeps its place as it was left; and the place's own crumb, none without.
type crumb struct {
	from  shell.Screen
	name  string
	spot  any
	under *crumb
}

// crumbed is a screen whose title draws its crumb: told the crumb's name, none for "", it gives itself drawing it; and
// it says whether a click, or a key, goes back to the crumb, as nothing open over the screen takes it first.
type crumbed interface {
	Crumbed(name string) tea.Model
	OnCrumb(c tea.MouseClickMsg) bool
	Backs(k tea.KeyPressMsg) bool
}

// spotted is a screen that says where it stands, its name and its spot, and is put back there by the spot, as an app
// is, its tab, its view, and its cursor; any other is kept as it was left, and named by its screen.
type spotted interface {
	Spot() (string, any)
	Restore(spot any) tea.Model
}

// linked shows the screen a link leads to, as show does, with the screen shown, where it stands now, as its crumb; a
// screen that draws no crumb is shown with the chain dropped, as the app's menu shows one.
func (m Model) linked(to shell.Screen) (tea.Model, tea.Cmd) {
	from := &crumb{from: m.here, name: string(m.here), under: m.crumb}
	if s, ok := m.made[m.here].(spotted); ok {
		from.name, from.spot = s.Spot()
	}
	next, cmd := m.show(to)
	n := next.(Model)
	if n.here != to || n.here == from.from && !chains(to) {
		return n, cmd
	}
	n.crumb = from
	if _, ok := n.made[to].(crumbed); !ok {
		n.crumb = nil
	}
	return n.toldCrumb(), cmd
}

// chains says whether a link from the screen to itself leaves the place it was followed from as the crumb: a tutorial's
// page, which a link on it replaces with another's, and ⌫ brings back; any other screen shows its place by its own.
func chains(s shell.Screen) bool { return s == shell.Tutorials }

// back shows the crumb's place again, put back where it stood, with its own crumb.
func (m Model) back() (tea.Model, tea.Cmd) {
	c := m.crumb
	if s, ok := m.made[c.from].(spotted); ok && c.spot != nil {
		m.made[c.from] = s.Restore(c.spot)
	}
	next, cmd := m.show(c.from)
	n := next.(Model)
	if n.here == c.from {
		n.crumb = c.under
	}
	return n.toldCrumb(), cmd
}

// dropped gives the root with no crumb, the screen shown told so.
func (m Model) dropped() Model {
	m.crumb = nil
	return m.toldCrumb()
}

// toldCrumb tells the screen shown its crumb, or that it has none.
func (m Model) toldCrumb() Model {
	s, ok := m.made[m.here].(crumbed)
	if !ok {
		return m
	}
	name := ""
	if m.crumb != nil {
		name = m.crumb.name
	}
	m.made[m.here] = s.Crumbed(name)
	return m
}

// crumbHere gives the screen shown, which says whether a key or a click goes back to its crumb, while it draws one;
// false while it has none, or draws none.
func (m Model) crumbHere() (crumbed, bool) {
	s, ok := m.made[m.here].(crumbed)
	return s, ok && m.crumb != nil
}
