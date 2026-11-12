// Package palette is what kit help opens: every verb but tui, a row each in a table, internal/table's sheet, as Flow,
// the welcome, and the tutorials draw theirs, under the menu bar, over the keys bar. Its columns are the verb's group,
// its command, and its description, as verbs.json, the catalog, says; its Default preset groups the rows by group, the
// groups in kit help's order, workflow first, then a group it does not name and other, and sorts them by command,
// ascending, and the description shows its first line, the verb's purpose as kit lists it.
// Opening a row, the right arrow or a click on its ▸, shows the rest of the description: the purpose's lines past its
// first, then the verb's page, kit help <verb> past its first line, which says the purpose again. The page is asked for
// as the row opens, once while it is on its way, and a dim … stands in its place meanwhile; a page kit could not give
// is kit's line, in red, asked for again when the row is opened again. Everything else a table does, the palette's
// does: the presets, sorting, grouping, filtering, the columns dialog, the selection, and the mouse wheel, which
// scrolls the rows, the cursor left where it is.
//
// The rows have focus from the start, the cursor on the first verb, so the arrows move at once. The search,
// internal/shell's, on the menu bar, works as it does on every screen: / or a click on it gives it focus; a key typed
// narrows the rows at once to those whose text holds it, backspace takes one back, and ⌃U clears; ⏎, ↓, esc, or tab
// gives the rows the keyboard, the filter kept, the cursor on the first verb; ⌃R recalls a past search. With the rows
// focused, the keys are the table's, and esc, or up past the control line, gives the search the keyboard again. Enter
// opens the form of the verb under the cursor, and a form that cannot open leaves kit's line in red under the menu bar,
// until the search changes or another verb opens. = or a click on the = before the screen's name drops the kit menu
// down, internal/shell's, as every screen of kittui has it, while the search has focus too, since no verb's name holds
// =, and - or a click on the workspace menu after it drops that down; esc there does nothing and never quits; ctrl-q
// quits, and ctrl-c quits as interrupted. The keys bar is the last line. Widths are terminal cells, so a two-cell glyph
// never pushes a row past the edge.
package palette

import (
	_ "embed"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// appLabel is the screen's name on the title line, after the app menu's =; a click on either drops the menu down.
const appLabel = "kit help"

//go:embed verbs.json
var verbsJSON []byte

// verbsTable is the name of the verbs' table in the catalog, which keys the person's presets of it too.
const verbsTable = "help/verbs"

// catalog is verbs.json, read.
var catalog = readCatalog(verbsJSON)

// readCatalog reads a catalog, as table.ReadCatalog does, and panics on one that does not read: verbs.json is kittui's
// own, and one that does not read is kittui's fault, which the tests catch before it ships.
func readCatalog(data []byte) table.Catalog {
	c, err := table.ReadCatalog(data)
	if err != nil {
		panic(fmt.Sprintf("kit help's catalog does not read: %v", err))
	}
	return c
}

// color is the table's color, the accent's.
const color = "12"

// other is the group of a verb whose banner names none, as kit help lists it.
const other = "other"

// under is what the grid stands before a line an open row opens to, which the page's width leaves out.
const under = "       "

// OpenMsg asks for the verb's form.
type OpenMsg struct{ Verb string }

// pageMsg brings a verb's page.
type pageMsg struct {
	verb, page string
	err        error
}

// Model is the palette on the screen.
type Model struct {
	verbs         []kit.Verb
	runner        kit.Runner
	sheet         table.Sheet
	search        shell.Search      // the search, internal/shell's, which narrows the rows to its text
	recent        shell.Recent      // the searches remembered, every screen's, and their dropdown
	pages         map[string]string // the pages kit gave, by verb
	asked         map[string]bool   // the verbs whose page is on its way
	failed        map[string]string // kit's line for a page it could not give, shown until the row opens again
	wrapped       int               // the width the records' pages are wrapped to
	placed        [3]int            // the pane's top, the screen's width and height, as the sheet was last placed
	refusal       string            // kit's line about the verb that did not open, under the title
	width, height int
	cancelled     bool
	interrupted   bool
	appMenu       shell.Menu      // the app's menu, internal/shell's, which has the keyboard while it is open
	books         shell.BooksMenu // the menu bar's workspace menu, which has the keyboard while it is open
}

// New opens the palette on the verbs, leaving out tui itself, the person's presets of its table read from the presets
// file. The rows have focus, the cursor on the first verb, so the arrows move at once; / gives the search focus.
func New(verbs []kit.Verb, r kit.Runner) Model {
	m := Model{runner: r, pages: map[string]string{}, asked: map[string]bool{}, failed: map[string]string{}, width: 100,
		height: 30, appMenu: shell.Menu{Here: shell.Help},
		sheet: table.NewSheet(catalog[verbsTable], "verbs", color, table.PresetsPath())}
	for _, v := range verbs {
		if v.Name != "tui" {
			m.verbs = append(m.verbs, v)
		}
	}
	m.place()
	m.lay()
	m.toVerb()
	return m
}

// groupOf gives the group a verb's row stands under: its own, or other.
func groupOf(v kit.Verb) string {
	if v.Group == "" {
		return other
	}
	return v.Group
}

// records gives each verb's record: its group, its command, and its description's first line, the purpose's; and the
// rest of the description, which its row opens to, as detail gives it.
func (m Model) records() []table.Record {
	out := make([]table.Record, 0, len(m.verbs))
	for _, v := range m.verbs {
		first, rest, _ := strings.Cut(v.Purpose, "\n")
		out = append(out, table.Record{Row: grid.Row{Key: v.Name, Detail: m.detail(v.Name, rest)},
			Values: map[string]table.Value{"group": table.One(groupOf(v)), "command": table.One(v.Name),
				"description": table.Some(first)}})
	}
	return out
}

// detail gives the lines a verb's row opens to, wrapped to the width they have: the purpose's lines past its first;
// then the page past its first line, which says the purpose again, or kit's line, in red, where it could not give it,
// or a dim … while it has not.
func (m Model) detail(verb, rest string) [][]grid.Cell {
	var lines []string
	if rest != "" {
		lines = page(rest, m.wrapped)
	}
	switch text, ok := m.pages[verb]; {
	case m.failed[verb] != "":
		for _, l := range page(m.failed[verb], m.wrapped) {
			lines = append(lines, style.Refusal(l))
		}
	case ok:
		body := strings.TrimLeft(text[strings.Index(text+"\n", "\n")+1:], "\n")
		if strings.TrimSpace(body) == "" {
			lines = append(lines, style.Off("kit help "+verb+" says no more"))
			break
		}
		lines = append(lines, page(body, m.wrapped)...)
	default:
		lines = append(lines, style.Off("…"))
	}
	out := make([][]grid.Cell, len(lines))
	for k, l := range lines {
		out[k] = []grid.Cell{{Values: []string{l}, Wrap: true}}
	}
	return out
}

// pageWidth gives the cells the lines an open row opens to have: the rows', less what stands before them.
func (m *Model) pageWidth() int { return max(1, m.sheet.GridWidth()-len(under)) }

// lay gives the table the verbs' records, the pages wrapped to the width they have; a width that changes as the rows
// are laid out, more levels of groups or fewer, lays them again.
func (m *Model) lay() {
	for range 3 {
		m.wrapped = m.pageWidth()
		m.sheet.SetRecords(m.records())
		if m.pageWidth() == m.wrapped {
			return
		}
	}
}

// place puts the table's pane under the lines over it, down to the gap over the keys bar, when the screen or those
// lines changed since; placed again, the rows' window follows the cursor.
func (m *Model) place() {
	top := len(m.top())
	if at := [3]int{top, m.width, m.height}; at != m.placed {
		m.placed = at
		m.sheet.Place(top, max(1, m.height-top-2), m.width, m.height)
	}
}

// Init asks for the page of each row open from the start, as At opens the verb's.
func (m Model) Init() tea.Cmd { return m.fetch() }

// fetch asks kit for the page of each verb whose row is open: once, never while it is on its way, and not again after
// kit could not give it until the row is closed and opened again, which forgets kit's line.
func (m *Model) fetch() tea.Cmd {
	open := m.sheet.Table().Open
	var cmds []tea.Cmd
	forgot := false
	for _, v := range m.verbs {
		verb := v.Name
		if !open[verb] {
			if _, ok := m.failed[verb]; ok {
				delete(m.failed, verb)
				forgot = true
			}
			continue
		}
		if _, ok := m.pages[verb]; ok || m.asked[verb] || m.failed[verb] != "" {
			continue
		}
		m.asked[verb] = true // the map is shared, so the palette that returned this command knows
		r := m.runner
		cmds = append(cmds, func() tea.Msg {
			page, err := r.Help(verb)
			return pageMsg{verb: verb, page: page, err: err}
		})
	}
	if forgot {
		m.lay()
	}
	return tea.Batch(cmds...)
}

// highlighted gives the verb under the cursor, and whether the cursor is on one rather than a heading.
func (m Model) highlighted() (string, bool) {
	rows, at := m.sheet.Rows(), m.sheet.Table().Cursor
	if at >= len(rows) || rows[at].Aside {
		return "", false
	}
	return rows[at].Key, true
}

// Visible gives the verbs the rows let through, in the order they stand, those a closed group folds away too.
func (m Model) Visible() []kit.Verb {
	byName := map[string]kit.Verb{}
	for _, v := range m.verbs {
		byName[v.Name] = v
	}
	var out []kit.Verb
	seen := map[string]bool{}
	for _, r := range grid.Every(m.sheet.Rows()) {
		if v, ok := byName[r.Key]; ok && !r.Aside && !seen[r.Key] {
			seen[r.Key] = true
			out = append(out, v)
		}
	}
	return out
}

// At puts the cursor on a verb, its row open, as kit help <verb> opens the palette, Init asking for its page; or on a
// group's first verb, as kit help <group> does; the rows have the keyboard. A name that is neither leaves the palette
// as it was.
func (m Model) At(name string) Model {
	verb := ""
	for _, v := range m.Visible() {
		if v.Name == name {
			verb = name
			break
		}
	}
	if verb != "" {
		m.sheet.Table().Open[verb] = true
		m.sheet.Rebuild()
	} else {
		for _, v := range m.Visible() {
			if groupOf(v) == name {
				verb = v.Name
				break
			}
		}
	}
	if verb == "" {
		return m
	}
	m.search.Focused = false
	m.sheet.EnterRows()
	for k, r := range m.sheet.Rows() {
		if r.Key == verb {
			m.sheet.Table().Cursor = k
			break
		}
	}
	m.sheet.Settle()
	return m
}

// toVerb gives the rows the keyboard, as the search giving it up does, the cursor on the first verb at or under it,
// past the headings of the groups.
func (m *Model) toVerb() {
	m.sheet.FocusRows()
	rows, t := m.sheet.Rows(), m.sheet.Table()
	for t.Cursor < len(rows)-1 && rows[t.Cursor].Aside {
		t.Cursor++
	}
	m.sheet.Settle()
}

// chose does what a choice from the app's menu asks: a screen asks kittui to show it in the palette's place, which
// stays as it is; and quit ends the palette, as ctrl-q does.
func (m Model) chose(c shell.Choice) tea.Cmd {
	switch {
	case c.Quit:
		return tea.Quit
	case c.Screen != "":
		return shell.Switch(c.Screen)
	}
	return nil
}

// Find gives the search focus, what is typed as it is, as a form's search field asks.
func (m Model) Find() Model {
	m.focusSearch()
	return m
}

// focusSearch gives the search the keyboard, and takes it from the table.
func (m *Model) focusSearch() {
	m.search.Focused = true
	m.sheet.Blur()
}

// Told gives the palette what kittui says of the workspace, which its menu bar names.
func (m Model) Told(msg shell.WorkspaceMsg) Model {
	m.books = m.books.Told(msg)
	return m
}

// Interrupted says whether ctrl-c ended the palette.
func (m Model) Interrupted() bool { return m.interrupted }

// Cancelled says whether ctrl-q, or Quit in the app's menu, ended the palette.
func (m Model) Cancelled() bool { return m.cancelled }

// Refuse shows kit's line, why the verb opened did not open, in red under the title, until the search changes or
// another verb is opened.
func (m Model) Refuse(line string) Model {
	m.refusal = line
	m.place()
	return m
}

// Remembering gives the palette remembering its searches in the file, as every screen of kittui does: one is
// remembered when the search gives the rows focus with it typed, and ctrl+r, or a click on the search's Recall glyph,
// drops them down.
func (m Model) Remembering(path string) Model {
	m.recent = shell.Recent{Path: path}.Loaded()
	return m
}

// typed puts the text in the search, which narrows the rows; a text that differs puts the cursor back at the top and
// clears kit's line about a verb that did not open.
func (m *Model) typed(query string) {
	if query == m.search.Query {
		return
	}
	m.search.Query, m.refusal = query, ""
	t := m.sheet.Table()
	t.Cursor, t.Offset = 0, 0
	m.sheet.Search(query)
}

// fades is how long the screen waits for table.Fades: Fades, as a test waits less.
var fades = table.Fades

// Update takes a message, as update does; then it places the table under the lines over it, lays the pages out again
// at a width that changed, asks for the page of each row opened, and fades a notice the table put up.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	before := m.sheet.Notice()
	next, cmd := m.update(msg)
	n := next.(Model)
	n.place()
	if n.pageWidth() != n.wrapped {
		n.lay()
	}
	return n, tea.Batch(cmd, n.fetch(), n.sheet.Notice().Fading(before, fades))
}

// update takes a message.
func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.lay()
	case shell.WorkspaceMsg:
		m = m.Told(msg)
	case shell.SettingsMsg:
		m.sheet.Settings(msg.Settings.FoldShared, msg.Settings.SharedEmoji)
	case shell.FadedMsg:
		m.sheet.Faded(msg)
	case tea.MouseClickMsg:
		return m.click(msg)
	case tea.MouseWheelMsg:
		if !m.appMenu.Open && !m.books.Open && !m.recent.Open {
			m.sheet.Wheel(msg)
		}
	case tea.KeyPressMsg:
		return m.key(msg)
	}
	return m, nil
}

// click takes a click: on an open menu, or a layer of the table; on the search, its Recall glyph, the workspace menu,
// or the screen's name; else on the table, whose rows, or control line, then have the keyboard where it took it.
func (m Model) click(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.sheet.Layered():
		m.sheet.Click(msg)
	case msg.Button != tea.MouseLeft:
	case m.appMenu.Open:
		var c shell.Choice
		m.appMenu, c = m.appMenu.Click(msg.X, msg.Y, m.width)
		return m, m.chose(c)
	case m.books.Open:
		var picked string
		m.books, picked = m.books.Click(msg.X, msg.Y, m.booksAt(), m.width)
		return m, shell.Pick(picked)
	case m.books.On(msg.X, msg.Y, m.booksAt()):
		m.books = m.books.Opened()
	case m.recent.Open:
		var query string
		if m.recent, query = m.recent.Click(msg.X, msg.Y, m.recallAt(), m.width); query != "" {
			m.typed(query)
		}
	case msg.Y == shell.Top && m.search.Focused && len(m.recent.Items) > 0 && msg.X == m.recallAt():
		m.recent = m.recent.Opened()
	case msg.Y == shell.Top && msg.X >= m.searchAt():
		m.focusSearch()
	case shell.OnApp(msg.X, msg.Y, appLabel):
		m.appMenu = m.appMenu.Opened()
	case m.sheet.Click(msg):
		if m.sheet.Focused() {
			m.search.Focused = false
		}
	}
	return m, nil
}

// key takes a key press: the quitting keys, an open menu's, the searches' dropdown's, a layer of the table's, the
// search's while it has focus, enter on a verb, the table's while it has the keyboard, then = for the app's menu, -
// for the workspace menu, and / for the search.
func (m Model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == shell.Quit:
		// quit, as on every screen, the app's menu open or not
		m.cancelled = true
		return m, tea.Quit
	case m.appMenu.Open && key != "ctrl+c":
		var c shell.Choice
		m.appMenu, c = m.appMenu.Key(msg)
		return m, m.chose(c)
	case m.books.Open && key != "ctrl+c":
		var picked string
		m.books, picked = m.books.Key(msg)
		return m, shell.Pick(picked)
	case m.recent.Open && key != "ctrl+c":
		var query string
		if m.recent, query = m.recent.Key(msg); query != "" {
			m.typed(query)
		}
		return m, nil
	case key == "ctrl+c":
		m.interrupted = true
		return m, tea.Quit
	case m.sheet.Layered():
		m.sheet.Key(msg)
		return m, nil
	}
	if m.search.Focused && key != "=" {
		// the search takes what it takes on every screen; =, which no verb's name holds, is the menu's
		s, did := m.search.Key(msg)
		switch did {
		case shell.Typed:
			m.typed(s.Query)
			return m, nil
		case shell.Left:
			m.search, m.recent = s, m.recent.Remember(s.Query)
			m.toVerb()
			return m, nil
		case shell.Recalled:
			m.recent = m.recent.Opened()
			return m, nil
		}
	}
	if verb, ok := m.highlighted(); ok && key == "enter" && m.sheet.InRows() {
		m.refusal = ""
		return m, func() tea.Msg { return OpenMsg{Verb: verb} }
	}
	if m.sheet.Focused() {
		took := m.sheet.Key(msg)
		if !m.sheet.Focused() {
			// esc, or up past the control line, gives the search the keyboard
			m.focusSearch()
		}
		if took {
			return m, nil
		}
	}
	switch key {
	case "=":
		m.appMenu = m.appMenu.Opened()
	case shell.Workspaces:
		if !m.search.Focused {
			m.books = m.books.Opened()
		}
	case "/":
		m.focusSearch()
	}
	return m, nil
}
