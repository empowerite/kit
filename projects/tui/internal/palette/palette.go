// Package palette is what bare kit opens: every verb but tui, as a tree under the groups kit help lists them in, each
// verb with its whole purpose, narrowed by the search, internal/shell's, on the title line, over the keys bar. The up
// and down arrows move over the verbs, never onto a heading, and shift with either jumps to the first verb of the
// previous or the next group. Page down and page up move a page, and home and end to the first and the last verb. The
// right arrow trades the purposes for the highlighted verb's page beside the names, and the left arrow brings them
// back. / or a click on the search gives it the keyboard, as on every screen. Enter opens the verb's form, and a form
// that cannot open leaves kit's line under the title; = or a click on ≡ before the title drops the app's menu down,
// internal/shell's, as every screen of kittui has it; esc quits, and ctrl-c quits as interrupted. Where the tree or the
// page is cut, a dim line says how many verbs or lines are hidden. The keys bar is the last line. Widths are terminal
// cells, so a two-cell glyph never pushes a row past the edge.
package palette

import (
	"cmp"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// OpenMsg asks for the verb's form.
type OpenMsg struct{ Verb string }

// pageMsg brings a verb's page.
type pageMsg struct {
	verb, page string
	err        error
}

// groups is the order kit help lists the groups in; a group it does not name follows them.
var groups = []string{"workflow", "files", "checks", "installation", "interfaces", "help"}

// minPage is the fewest cells beside the names that a page is drawn in; a screen narrower shows the names alone.
const minPage = 12

// Model is the palette on the screen.
type Model struct {
	verbs         []kit.Verb
	runner        kit.Runner
	search        shell.Search // what is typed in the search, internal/shell's, which narrows the verbs to it
	cursor        int
	pages         map[string]string // the pages kit gave, by verb
	asked         map[string]bool   // the verbs whose page is on its way
	failed        map[string]string // kit's line for a page it could not give, shown until it is asked for again
	showPages     bool
	refusal       string // kit's line about the verb that did not open, under the title
	width, height int
	cancelled     bool
	interrupted   bool
	appMenu       shell.Menu // the app's menu, internal/shell's, which has the keyboard while it is open
}

// New opens the palette on the verbs, leaving out tui itself, in the order the tree draws them: by group as kit help
// orders the groups, a group kit help does not name after them by its name, and within a group as kit listed them.
func New(verbs []kit.Verb, r kit.Runner) Model {
	m := Model{runner: r, pages: map[string]string{}, asked: map[string]bool{}, failed: map[string]string{}, width: 100,
		height: 30, appMenu: shell.Menu{Here: shell.Help}}
	for _, v := range verbs {
		if v.Name != "tui" {
			m.verbs = append(m.verbs, v)
		}
	}
	slices.SortStableFunc(m.verbs, func(a, b kit.Verb) int {
		return cmp.Or(cmp.Compare(rank(a.Group), rank(b.Group)), strings.Compare(a.Group, b.Group))
	})
	return m
}

// rank gives a group's place in the tree: its index among kit help's groups, or past them all for one kit help does
// not name.
func rank(group string) int {
	if i := slices.Index(groups, group); i >= 0 {
		return i
	}
	return len(groups)
}

// heading gives the heading drawn over a group's verbs: the group's name upper-cased, as kit help prints it, or OTHER
// for the verbs whose banner names no group, as kit help lists them.
func heading(group string) string {
	if group == "" {
		return "OTHER"
	}
	return strings.ToUpper(group)
}

// Init starts nothing, unless the palette was opened on a verb's page: then it fetches that page.
func (m Model) Init() tea.Cmd { return m.page() }

// At puts the cursor on a verb and shows its page, as kit help <verb> opens the palette; or on a group's first verb,
// with the purposes shown, as kit help <group> does. A name that is neither leaves the palette as it was.
func (m Model) At(name string) Model {
	for i, v := range m.verbs {
		if v.Name == name {
			m.cursor, m.showPages = i, true
			return m
		}
	}
	for i, v := range m.verbs {
		if v.Group == name {
			m.cursor = i
			return m
		}
	}
	return m
}

// chose does what a choice from the app's menu asks: a screen asks kittui to show it in the palette's place, which
// stays as it is; and quit ends the palette, as esc does.
func (m Model) chose(c shell.Choice) tea.Cmd {
	switch {
	case c.Quit:
		return tea.Quit
	case c.Screen != "":
		return shell.Switch(c.Screen)
	}
	return nil
}

// searchAt gives the column the search starts at on the title line, as internal/shell puts every screen's.
func (m Model) searchAt() int { return shell.SearchAt(m.width, nil, m.search.InUse()) }

// Interrupted says whether ctrl-c ended the palette.
func (m Model) Interrupted() bool { return m.interrupted }

// Cancelled says whether esc ended the palette.
func (m Model) Cancelled() bool { return m.cancelled }

// Refuse shows kit's line, why the verb opened did not open, in red under the title, until the search changes or
// another verb is opened.
func (m Model) Refuse(line string) Model {
	m.refusal = line
	return m
}

// Visible gives the verbs the search leaves, those whose name holds the text typed, in the order the tree draws them,
// which the cursor counts along.
func (m Model) Visible() []kit.Verb {
	var out []kit.Verb
	for _, v := range m.verbs {
		if strings.Contains(v.Name, strings.ToLower(m.search.Query)) {
			out = append(out, v)
		}
	}
	return out
}

// Update takes a message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case pageMsg:
		delete(m.asked, msg.verb)
		if msg.err != nil {
			m.failed[msg.verb] = style.Excluded + " " + msg.err.Error()
		} else {
			m.pages[msg.verb] = msg.page
			delete(m.failed, msg.verb)
		}
	case tea.MouseClickMsg:
		switch {
		case msg.Button != tea.MouseLeft:
		case m.appMenu.Open:
			var c shell.Choice
			m.appMenu, c = m.appMenu.Click(msg.X, msg.Y, m.width)
			return m, m.chose(c)
		case shell.OnApp(msg.X, msg.Y):
			m.appMenu = m.appMenu.Opened()
		case msg.Y == shell.Top && msg.X >= m.searchAt():
			m.search.Focused = true
		}
	case tea.KeyPressMsg:
		visible := m.Visible()
		if m.appMenu.Open && msg.String() != "ctrl+c" {
			var c shell.Choice
			m.appMenu, c = m.appMenu.Key(msg)
			return m, m.chose(c)
		}
		if m.search.Focused && msg.String() != "ctrl+c" {
			// the search has the keyboard: what it does not take, as the arrows, is nobody's
			before := m.search.Query
			m.search, _ = m.search.Key(msg)
			if m.search.Query != before {
				m.cursor, m.refusal = 0, ""
			}
			return m, m.page()
		}
		switch msg.String() {
		case "=":
			m.appMenu = m.appMenu.Opened()
		case "/":
			m.search.Focused = true
		case "esc":
			m.cancelled = true
			return m, tea.Quit
		case "ctrl+c":
			m.interrupted = true
			return m, tea.Quit
		case "enter":
			if m.cursor < len(visible) {
				verb := visible[m.cursor].Name
				m.refusal = ""
				return m, func() tea.Msg { return OpenMsg{Verb: verb} }
			}
		case "right":
			m.showPages = true
		case "left":
			m.showPages = false
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(visible)-1 {
				m.cursor++
			}
		case "shift+up":
			// to the first verb of the cursor's group, or, from there, of the group before; the first group stays
			if m.cursor < len(visible) {
				if f := first(visible, m.cursor); f < m.cursor {
					m.cursor = f
				} else if f > 0 {
					m.cursor = first(visible, f-1)
				}
			}
		case "shift+down":
			if n := next(visible, m.cursor); n < len(visible) {
				m.cursor = n
			}
		case "pgdown", "pgup", "home", "end":
			page := len(visible)
			if rows := m.rows(); rows > 0 {
				page = max(1, rows-1)
			}
			switch msg.String() {
			case "pgdown":
				m.cursor += page
			case "pgup":
				m.cursor -= page
			case "home":
				m.cursor = 0
			case "end":
				m.cursor = len(visible) - 1
			}
			m.cursor = max(0, min(m.cursor, len(visible)-1))
		}
		return m, m.page()
	}
	return m, nil
}

// first gives the index of the first verb in the group of the verb at i.
func first(visible []kit.Verb, i int) int {
	for i > 0 && visible[i-1].Group == visible[i].Group {
		i--
	}
	return i
}

// next gives the index of the first verb of the group after the group of the verb at i, or len(visible) when that
// group is the last.
func next(visible []kit.Verb, i int) int {
	for j := i + 1; j < len(visible); j++ {
		if visible[j].Group != visible[i].Group {
			return j
		}
	}
	return len(visible)
}

// page fetches the highlighted verb's page when the pages are shown: once, and never while it is on its way. A page
// kit could not give is asked for again, its line shown meanwhile.
func (m Model) page() tea.Cmd {
	visible := m.Visible()
	if !m.showPages || m.cursor >= len(visible) {
		return nil
	}
	verb := visible[m.cursor].Name
	if _, ok := m.pages[verb]; ok || m.asked[verb] {
		return nil
	}
	m.asked[verb] = true // the map is shared, so the palette that returned this command knows
	r := m.runner
	return func() tea.Msg {
		page, err := r.Help(verb)
		return pageMsg{verb: verb, page: page, err: err}
	}
}

// row is one line of the tree: a group's heading, or a verb by its index among the visible verbs.
type row struct {
	heading string
	index   int
}

// tree gives the rows the tree draws for the visible verbs: each group's heading, then its verbs. A group the search
// leaves empty has no row.
func tree(visible []kit.Verb) []row {
	var rows []row
	for i, v := range visible {
		if i == 0 || v.Group != visible[i-1].Group {
			rows = append(rows, row{heading: heading(v.Group), index: -1})
		}
		rows = append(rows, row{index: i})
	}
	return rows
}

// widest gives the width of the names' column, in cells: the widest name, or more when a heading, which starts two
// columns before the names, would run past it.
func (m Model) widest() int {
	widest := 0
	for _, v := range m.verbs {
		widest = max(widest, ansi.StringWidth(v.Name), ansi.StringWidth(heading(v.Group))-2)
	}
	return widest
}

// top gives the lines over the tree: the title, as internal/shell draws it, after the app's menu's glyph, with the
// search on its right; kit's line under it when a verb did not open; and a gap.
func (m Model) top() []string {
	title, at := shell.Title(style.Label("kit help"), m.appMenu.Open), m.searchAt()
	top := []string{pane.Pad(pane.Clip(title, at-1), at, false) + shell.SearchLine(m.search.Query, m.search.Focused)}
	if m.refusal != "" {
		for _, line := range page(style.Excluded+" "+m.refusal, m.width-2) {
			top = append(top, " "+style.Refusal(line))
		}
	}
	return append(top, "")
}

// rows gives the lines the tree's window takes, between the top and the gap over the keys bar, or 0 before the
// terminal says its height, when nothing is cut. A page key moves a page of that, less one.
func (m Model) rows() int {
	if m.height <= 0 {
		return 0
	}
	return max(1, m.height-len(m.top())-2)
}

// verbs counts the verbs among the tree's rows.
func verbs(rows []row) int {
	n := 0
	for _, r := range rows {
		if r.index >= 0 {
			n++
		}
	}
	return n
}

// View draws the title and the search, kit's line under them when a verb did not open, and the tree: each group's
// heading, dimmed, over its verbs, each with its whole purpose, clipped at the screen's edge, or, after the right
// arrow, the names beside the highlighted verb's page. The rows shown are the window that holds the cursor's row, with
// a dim line for the verbs it hides above and one for those below, each in a row's place; a page cut short says how
// many lines it hides. The keys bar is the last line, one line clipped to the width, so neither searching nor a refusal
// moves it; while the search has the keyboard it is the search's, as on every screen.
func (m Model) View() tea.View {
	visible := m.Visible()
	keys := "↑↓ move · ⇞⇟ page · ⇧↑↓ group · / search · → pages · ⏎ open · = menu"
	if m.showPages {
		keys = "↑↓ move · ⇞⇟ page · ⇧↑↓ group · / search · ← purposes · ⏎ open · = menu"
	}
	switch {
	case m.appMenu.Open:
		keys = shell.Keys
	case m.search.Focused:
		keys = shell.SearchKeys
	}
	top := m.top()
	widest := m.widest()
	rows := m.rows()
	lines := tree(visible)
	at := 0
	for i, r := range lines {
		if r.index == m.cursor {
			at = i
		}
	}
	span := pane.Frame(len(lines), rows, 0, at, at)
	var list []string
	if len(visible) == 0 {
		list = append(list, "   "+style.Off("no verb matches"))
	}
	if span.Above {
		list = append(list, "   "+style.Off(pane.Above(verbs(lines[:span.Start]))))
	}
	for i := span.Start; i < span.End; i++ {
		r := lines[i]
		if r.index < 0 {
			list = append(list, " "+style.Off(r.heading))
			continue
		}
		v := visible[r.index]
		lead, name := "   ", v.Name
		if r.index == m.cursor {
			lead, name = " "+style.Focus+" ", style.Chosen(v.Name)
		}
		line := lead + name + strings.Repeat(" ", widest-ansi.StringWidth(v.Name))
		if !m.showPages {
			line += "  " + style.Off(pane.Clip(v.Purpose, m.width-3-widest-2))
		}
		list = append(list, line)
	}
	if span.Below {
		list = append(list, "   "+style.Off(pane.More(verbs(lines[span.End:]))))
	}
	leftWidth := 3 + widest + 2
	rightWidth := m.width - leftWidth
	if m.showPages && rightWidth >= minPage {
		right := ""
		if m.cursor < len(visible) {
			name := visible[m.cursor].Name
			right = m.pages[name]
			if line, ok := m.failed[name]; ok {
				right = line
			}
		}
		pageLines := page(right, rightWidth-1)
		if cut := pane.Frame(len(pageLines), rows, 0, 0, 0); cut.Below {
			pageLines = append(pageLines[:cut.End], style.Off(pane.More(len(pageLines)-cut.End)))
		} else {
			pageLines = pageLines[:cut.End]
		}
		list = strings.Split(lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(leftWidth).Render(strings.Join(list, "\n")),
			lipgloss.NewStyle().Width(rightWidth).Render(strings.Join(pageLines, "\n"))), "\n")
	}
	// a screen narrower than the names' column cuts every line at its edge
	var drawn []string
	for _, l := range append(top, list...) {
		drawn = append(drawn, pane.Clip(l, m.width))
	}
	drawn = m.appMenu.Overlay(drawn, m.width)
	v := tea.NewView(strings.Join(style.Bottom(drawn, m.height, style.Bar(keys, m.width)), "\n"))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// page wraps a text to the width, keeping each line's indent, and gives its lines. A width counts what shows, so the
// escapes that make a heading bold take none.
func page(text string, width int) []string {
	var out []string
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
		words := strings.Fields(line)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		current := indent + words[0]
		for _, word := range words[1:] {
			if ansi.StringWidth(current)+1+ansi.StringWidth(word) > width {
				out = append(out, current)
				current = indent + word
				continue
			}
			current += " " + word
		}
		out = append(out, current)
	}
	return out
}
