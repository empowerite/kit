// The tab's menu, driven as Bubble Tea would: : or a click on the tab drops it down, its lines do what they say, and
// Maximize gives the pane the bar's lines.
package welcome

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// paneTop is the screen line the pane starts on while the tab is not maximized, as most tests have it.
const paneTop = 1 + tabs.Height

// typed gives the press of a key that types a character, as : and m do.
func typed(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

// names gives the names of the tab's menu's lines, a rule as "".
func names(m Model) []string {
	var out []string
	for _, item := range m.tabItems() {
		out = append(out, item.Name)
	}
	return out
}

// menuLine gives the click on the tab's menu's line named, as the box is drawn: its first cell past the left edge.
func menuLine(m Model, name string) tea.MouseClickMsg {
	_, left, first := m.menuBoxIn()
	k := slices.Index(names(m), name)
	return testkit.Click(left+2, m.menuTop()+1+k-first)
}

// The selected tab draws :, which : and ; drop its menu down from, as on every screen with tabs: on a page, Refresh
// and Maximize; the open menu has the keyboard, and the key that opened it, or esc, closes it.
func TestColonDropsTheTabsMenuDown(t *testing.T) {
	m, _ := loaded(t, stepKey(chains))
	if bar := strings.Split(view(m), "\n")[2]; !strings.Contains(bar, "🧰 toolchains  3 "+tabs.Menu+" │") {
		t.Fatalf("the selected tab draws no %s: %q", tabs.Menu, bar)
	}
	m = drive(m, typed(':'))
	if !m.tabMenu.Open || keysAt(m) != (shell.Path{shell.MenuKeys()}).String() ||
		!slices.Equal(names(m), []string{refreshName, "Maximize"}) {
		t.Fatalf("open %v, keys %q, lines %q", m.tabMenu.Open, keysAt(m), names(m))
	}
	if v := view(m); !strings.Contains(v, "Refresh") || !strings.Contains(v, "Maximize") {
		t.Errorf("the menu is not drawn:\n%s", v)
	}
	if m = drive(m, typed(':')); m.tabMenu.Open {
		t.Errorf(": did not close the menu")
	}
	if m = drive(m, typed(';'), esc); m.tabMenu.Open || m.step != chains {
		t.Errorf("; and esc: open %v, step %d", m.tabMenu.Open, m.step)
	}
	// while it is open, the introduction's scrolling keys are the menu's
	m, _ = loaded(t, down, down, down, typed(':'), pgdown)
	if !m.tabMenu.Open || m.offsets[introduction] != 0 {
		t.Errorf("pgdown with the menu open: open %v, offset %d", m.tabMenu.Open, m.offsets[introduction])
	}
}

// Maximize gives the pane the bar's lines but one: the bar goes, the title names the tab with its number and :, a rule
// in the accent under it, and tab and shift-tab move nothing, while a step's number still selects it, maximized too.
// Restore, or m, takes them back.
func TestMaximizeGivesThePaneTheBarsLines(t *testing.T) {
	m, _ := loaded(t, stepKey(chains))
	before := m.paneHeight()
	m = drive(m, typed(':'), down, enter)
	lines := strings.Split(view(m), "\n")
	switch {
	case !m.maximized || m.tabMenu.Open || m.paneHeight() != before+2:
		t.Fatalf("maximized %v, open %v, pane %d lines, not %d", m.maximized, m.tabMenu.Open, m.paneHeight(), before+2)
	case !strings.Contains(lines[0], "➋ toolchains 3 "+tabs.Menu) || strings.Contains(lines[0], "🧰"):
		t.Errorf("the title does not name the tab: %q", lines[0])
	case strings.Trim(lines[1], shell.Rule) != "" || ansi.StringWidth(lines[1]) != at120.Width:
		t.Errorf("no banner under the title: %q", lines[1])
	case strings.Contains(view(m), "🔧 setup") || strings.Contains(keysAt(m), "⇥·⇧⇥"):
		t.Errorf("the bar is still there:\n%s\nkeys %q", view(m), keysAt(m))
	}
	if m = drive(m, tabKey, backTab); m.step != chains {
		t.Errorf("tab and shift-tab moved to step %d", m.step)
	}
	if m = drive(m, stepKey(setup)); m.step != setup || !m.maximized {
		t.Errorf("a step's number: step %d, maximized %v", m.step, m.maximized)
	}
	if m = drive(m, typed(':')); !slices.Contains(names(m), "Restore") {
		t.Errorf("maximized, the menu offers %q", names(m))
	}
	if m = drive(m, esc, typed('m')); m.maximized || m.paneHeight() != before ||
		!strings.Contains(strings.Split(view(m), "\n")[2], "🔧 setup  2") || m.bar().MenuAt() < 0 {
		t.Errorf("m did not restore it: maximized %v, pane %d lines\n%s", m.maximized, m.paneHeight(), view(m))
	}
}

// A click anywhere on the selected tab drops its menu down, hanging from its :; a click on a line chooses it, and a
// click anywhere else closes the menu. Maximized, a click on the tab's name in the title drops it down from there.
func TestAClickOnTheTabDropsItsMenuDown(t *testing.T) {
	m, _ := loaded(t, stepKey(chains))
	x := m.bar().MenuAt()
	if m = drive(m, testkit.Click(x-4, 1)); !m.tabMenu.Open || m.menuTop() != 2 || m.tabMenuAt() != x {
		t.Fatalf("a click on the tab: open %v, top %d, at %d, not %d", m.tabMenu.Open, m.menuTop(), m.tabMenuAt(), x)
	}
	if m = drive(m, testkit.Click(x, m.height-3)); m.tabMenu.Open || m.maximized {
		t.Errorf("a click off the menu: open %v, maximized %v", m.tabMenu.Open, m.maximized)
	}
	m = drive(m, testkit.Click(x, 2))
	if m = drive(m, menuLine(m, "Maximize")); !m.maximized || m.tabMenu.Open {
		t.Fatalf("a click on Maximize: maximized %v, open %v", m.maximized, m.tabMenu.Open)
	}
	from, until, _ := m.menuBar().OnSpan(m.width)
	if m = drive(m, testkit.Click(from, shell.Top)); !m.tabMenu.Open || m.menuTop() != shell.Top ||
		m.tabMenuAt() != until-1 {
		t.Fatalf("a click on the title's tab: open %v, top %d, at %d", m.tabMenu.Open, m.menuTop(), m.tabMenuAt())
	}
	if m = drive(m, menuLine(m, "Restore")); m.maximized {
		t.Errorf("a click on Restore left it maximized")
	}
	// maximized on a screen too narrow for the title to reach the tab's :, the menu hangs against the screen's right
	m = drive(m, typed('m'), tea.WindowSizeMsg{Width: 40, Height: 30})
	if m.tabMenuAt() != -1 {
		t.Errorf("the title is clipped, yet the menu hangs from %d", m.tabMenuAt())
	}
	// a click on a check's line, maximized, is read from the pane's own top
	m, _ = loaded(t, stepKey(chains), typed('m'))
	lines := m.stepLines()
	at := slices.IndexFunc(lines, func(l line) bool { return l.row >= 0 && l.row != m.cursor })
	if m = drive(m, testkit.Click(6, m.paneTop()+at)); !m.inPane || m.cursor != lines[at].row {
		t.Errorf("a click on a check, maximized: in the pane %v, cursor %d, not %d", m.inPane, m.cursor, lines[at].row)
	}
}

// Refresh runs the checks again, as after a fix: checking… shows until they answer.
func TestRefreshRunsTheChecksAgain(t *testing.T) {
	m, _ := loaded(t, stepKey(chains))
	gen := m.checking.Gen()
	var cmd tea.Cmd
	next, _ := m.Update(typed(':'))
	m = next.(Model)
	next, cmd = m.Update(press(tea.KeyEnter))
	if m = next.(Model); m.tabMenu.Open || !m.checking.Running() || m.checking.Gen() == gen || cmd == nil {
		t.Fatalf("Refresh: open %v, running %v, run %d after %d", m.tabMenu.Open, m.checking.Running(),
			m.checking.Gen(), gen)
	}
	if !strings.Contains(view(m), "checking…") {
		t.Errorf("no checking…:\n%s", view(m))
	}
}

// A step drawn as a table adds its folds under a rule, each dim while it would change nothing; one chosen folds the
// table, whose rows then have the keyboard. The table moves with the pane as it is maximized and restored.
func TestATablesStepAddsItsFolds(t *testing.T) {
	m, _ := loaded(t, stepKey(custom), typed(':'))
	want := []string{refreshName, "Maximize", "", "Show More", "Show Less", "Show Selected", "Hide Selected"}
	if !slices.Equal(names(m), want) {
		t.Fatalf("lines %q", names(m))
	}
	if m.enabled(2) || m.enabled(slices.Index(want, "Show Selected")) {
		t.Errorf("the rule, or Show Selected with none selected, is enabled")
	}
	k := slices.Index(want, "Show More")
	if !m.enabled(k) {
		k = slices.Index(want, "Show Less")
	}
	if !m.enabled(k) {
		t.Fatalf("no fold is enabled on the customize step")
	}
	before := maps(m.sheet().Table().Open)
	m = drive(m, menuLine(m, want[k]))
	if m.tabMenu.Open || !m.inPane || !m.sheet().InRows() || maps(m.sheet().Table().Open) == before {
		t.Errorf("%s: open %v, in the pane %v, in rows %v, outline %s", want[k], m.tabMenu.Open, m.inPane,
			m.sheet().InRows(), maps(m.sheet().Table().Open))
	}
	top, _ := m.sheet().Pane()
	if m = drive(m, typed('m')); func() int { t, _ := m.sheet().Pane(); return t }() != top-2 {
		t.Errorf("maximized, the table did not move up two lines from %d", top)
	}
}

// maps gives an outline as a string, to tell one from another.
func maps(o map[string]bool) string {
	var out []string
	for k, v := range o {
		if v {
			out = append(out, k+"+")
		} else {
			out = append(out, k+"-")
		}
	}
	slices.Sort(out)
	return strings.Join(out, " ")
}
