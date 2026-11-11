// Package welcome is kit welcome on a terminal: a step per page as tabs along the top, the step's content below, and
// the keys bar last. The steps are the pages under .kit/welcome/, as kit welcome --text finds and orders them: each
// has an id, which kit's checks name as their step, a glyph, a name, and its page. A page kit knows keeps its glyph,
// its name, and its checks; any other, as a house package adds, is a plain page. A step with checks draws its page,
// then its checks, one row each. A failed check with a fix offers now and later on the cursor's row: now runs kit
// welcome --fix <id> with the terminal handed over, then reads the checks again; later leaves it for this session and
// moves to the step's next failed check. A fix kit only shows, as one that must run in the person's own shell, is
// drawn with its lines and no choices: n and enter do nothing there, and l still leaves it for later. The other steps
// are markdown pages drawn as text, but the terms, whose page of sections is a table, as sections.go draws it, its
// keys and clicks its own while the pane has the keyboard. The intro's page ends with what the checks found:
// checking… while they run, then that all of them pass, or how many need the person, each failed check's line a link
// to it, with the glyph and name of its step. Enter or a click on a link goes to that check, on its step, with the
// cursor on it. When kit refused the checks, the intro says what kit said.
//
// The steps are a tab bar, internal/tabs. While it has the keyboard, tab and shift-tab move between the steps; a step's
// number, or a click on the bar, selects it from anywhere and gives the bar the keyboard. Down, or enter, gives the
// pane the keyboard: a step's rows, or the intro's links, where up on the first gives it back, or its page, which up
// and down then scroll, and up at the top gives it back; a click on a check's line puts the cursor there. Inside a row
// with choices, tab and shift-tab move between now and later, its stops, as they move between the stops on any line. Esc gives the bar the keyboard back. Ctrl-c quits with 130
// from anywhere, and ctrl-q quits with 0; esc never quits. The screen
// opens on the intro while the checks run, and stays there when they answer. A step with failed checks counts them in
// red on the bar, which every step shows.
package welcome

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/source"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// introID is the id of the intro, the step whose page ends with what the checks found.
const introID = "intro"

// tab is a step: the id its checks name, its tab's glyph and name, the file of its page, whether it has checks, and
// its page's markdown, or, when the page could not be read, where it was looked for; and, for a step sheetSteps names,
// its table, where its page has sections, and the text before them, its lead. Each glyph is two cells wide with no
// variation selector.
type tab struct {
	id, glyph, name, file string
	checked               bool
	page, missing         string
	sheet                 *table.Sheet
	lead                  string
}

// known gives the pages kit knows, in the kit package's order. A check whose step has no page shows under the last
// step with checks.
var known = []tab{
	{id: "intro", glyph: "👋", name: "intro", file: "1-intro.md"},
	{id: "before-toolchains", glyph: "🧱", name: "basics", file: "2-before-toolchains.md", checked: true},
	{id: "toolchains", glyph: "🧰", name: "toolchains", file: "3-toolchains.md", checked: true},
	{id: "setup", glyph: "🔧", name: "setup", file: "4-setup.md", checked: true},
	{id: "lay-of-the-land", glyph: "🧭", name: "lay of the land", file: "5-lay-of-the-land.md"},
	{id: "terms", glyph: "📖", name: "terms", file: "6-terms.md"},
	{id: "make-it-yours", glyph: "🎨", name: "make it yours", file: "7-make-it-yours.md"},
}

// pageGlyph is the glyph of a page kit does not know.
const pageGlyph = "📄"

// idOf gives the id of the step a page's file is for, as kit welcome reads it: the name without the order and the
// dash before it, and without .md. 2-before-toolchains.md is for before-toolchains.
func idOf(file string) string {
	name := strings.TrimSuffix(file, ".md")
	if _, id, found := strings.Cut(name, "-"); found {
		return id
	}
	return name
}

// tabOf gives the step a page's file makes: the known step with its id, with the file's name, so a house package's
// page in its place keeps its glyph, its name, and its checks; else a plain page, named for its id, dashes as spaces.
func tabOf(file string) tab {
	id := idOf(file)
	if i := indexOf(known, id); i >= 0 {
		t := known[i]
		t.file = file
		return t
	}
	return tab{id: id, glyph: pageGlyph, name: strings.ReplaceAll(id, "-", " "), file: file}
}

// indexOf gives the index of the first step with the id, or -1.
func indexOf(tabs []tab, id string) int {
	return slices.IndexFunc(tabs, func(t tab) bool { return t.id == id })
}

// stepOf gives the index of the step a check shows under: its own when that step has checks, else the last that has.
func (m Model) stepOf(c kit.Check) int {
	if i := indexOf(m.tabs, c.Step); i >= 0 && m.tabs[i].checked {
		return i
	}
	last := 0
	for i, t := range m.tabs {
		if t.checked {
			last = i
		}
	}
	return last
}

// The choices on a failed check's row.
const (
	now = iota
	later
)

// pagesMsg brings the steps the pages make, each with its page's markdown, or the path it was looked for at.
type pagesMsg struct {
	tabs []tab
}

// fixedMsg says a check's fix has run, and how it ended.
type fixedMsg struct {
	id, key string // the check's id, which kit's line names, and its key on the screen
	err     error
}

// note is what the screen says under a check after its fix ran and the check still fails: a warning, or kit's
// refusal.
type note struct {
	text    string
	refusal bool
}

// Model is the welcome screen.
type Model struct {
	runner    kit.Runner
	ctx       context.Context                           // ends with the screen, and the checks still running with it
	cancel    context.CancelFunc                        // ends ctx
	exec      func(*exec.Cmd, tea.ExecCallback) tea.Cmd // hands the terminal to a fix; tea.ExecProcess but in tests
	workspace kit.Workspace                             // the workspace kittui opened the screen on, zero while unknown
	books     shell.BooksMenu                           // the title's workspace menu, which has the keyboard while open
	tabs      []tab                                     // the steps, in order: the known ones until the pages are read
	step      int                                       // the selected step
	inPane    bool                                      // whether the pane, a check's row or the page, has the keyboard
	barView   int                                       // the cells the bar's view scrolled past, as the bar last chose
	places    []string                                  // the check each step's cursor was on when it was left
	checking  source.Load[[]kit.Check]                  // kit welcome --checks --json: loading until it first answers
	checks    []kit.Check
	refusal   string // kit's line, when the checks did not run
	cursor    int    // the check under the cursor, an index into checks, in the selected step
	choice    int    // now or later, on a row that offers them
	deferred  map[string]bool
	notes     map[string]note
	offsets   []int // the first line of each step's pane
	width     int
	height    int
	appMenu   shell.Menu              // the app's menu, internal/shell's, which has the keyboard while it is open
	searches  map[string]shell.Search // each step's search, internal/shell's, by the step's id
	recent    shell.Recent            // the searches remembered, every screen's, and their dropdown
	presets   string                  // the file the person's presets are kept in; none keeps them for the session
	settings  shell.Settings          // kittui's settings, as the screen was last told them

	interrupted bool
}

// Open opens the screen in the directory the runner names, with owner/repo read from git, on the workspace kittui
// says, the zero one while it does not know.
func Open(r kit.Runner, ws kit.Workspace) Model {
	m := New(r, readRepo(r.Dir))
	m.workspace, m.books.Workspace = ws, ws
	m.checking = source.NewLoad("checks", checksOver(r, ws), nil)
	m.recent = shell.Recent{Path: shell.RecentFile()}.Loaded()
	m.presets = table.PresetsPath()
	return m
}

// New opens the screen on the first step, checking. Init runs the checks and reads the pages.
func New(r kit.Runner, repo string) Model {
	checks := func(ctx context.Context, _ bool) ([]kit.Check, error) { return r.WelcomeChecks(ctx) }
	m := Model{runner: r, exec: tea.ExecProcess, checking: source.NewLoad("checks", checks, nil),
		deferred: map[string]bool{},
		notes:    map[string]note{}, width: 80, height: 24, tabs: slices.Clone(known),
		places: make([]string, len(known)), offsets: make([]int, len(known)), appMenu: shell.Menu{Here: shell.Welcome},
		books: shell.BooksMenu{Fallback: repo}, searches: map[string]shell.Search{},
		settings: shell.Settings{KeysBar: true, FoldShared: true}}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	return m
}

// quit ends the screen, and the checks still running with it.
func (m Model) quit() tea.Cmd {
	m.cancel()
	return tea.Quit
}

// chose does what a choice from the app's menu asks: a screen asks kittui to show it in the guide's place, which stays
// as it is; and quit ends the screen.
func (m Model) chose(c shell.Choice) tea.Cmd {
	switch {
	case c.Quit:
		return m.quit()
	case c.Screen != "":
		return shell.Switch(c.Screen)
	}
	return nil
}

// Books gives the title's workspace menu.
func (m Model) Books() shell.BooksMenu { return m.books }

// Init runs the checks and reads the pages, at once.
func (m Model) Init() tea.Cmd { return tea.Batch(m.check(), m.read()) }

// Interrupted says whether ctrl-c ended the screen.
func (m Model) Interrupted() bool { return m.interrupted }

// Close ends the checks still running, as quitting does, for a guide kittui leaves hidden as it ends.
func (m Model) Close() { m.cancel() }

// check runs kit welcome --checks --json, the checks' first run, for as long as the screen lasts and kit's limit lets
// it.
func (m Model) check() tea.Cmd { return m.checking.Read(m.env(), false) }

// env gives what the screen's reads of kit share: the screen's context, and no deadline but kit's own.
func (m Model) env() source.Env { return source.Env{Ctx: m.ctx} }

// read reads the pages, as pages says.
func (m Model) read() tea.Cmd {
	r := m.runner
	return func() tea.Msg {
		return pagesMsg{tabs: readPages(r)}
	}
}

// Update takes a message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.placeSheets()
		m.fit()
	case shell.SettingsMsg:
		m.settings = msg.Settings
		m.eachSheet(func(s *table.Sheet) { s.Settings(msg.Settings.FoldShared, msg.Settings.SharedEmoji) })
	case shell.FadedMsg:
		m.eachSheet(func(s *table.Sheet) { s.Faded(msg) })
	case shell.WorkspaceMsg:
		m.books = m.books.Told(msg)
	case source.Answer[[]kit.Check]:
		m.take(msg)
	case pagesMsg:
		m.setTabs(msg.tabs)
		m.takeSheets()
		m.fit()
	case fixingMsg:
		return m, m.fixIn(msg.id, msg.key, msg.dirs)
	case fixedMsg:
		// the checks run again, superseding a run still out
		m.fixed(msg)
		return m, m.checking.Again(m.env(), false)
	case tea.KeyPressMsg:
		return m.key(msg)
	case tea.MouseClickMsg:
		switch {
		case m.appMenu.Open && msg.Button == tea.MouseLeft:
			var c shell.Choice
			m.appMenu, c = m.appMenu.Click(msg.X, msg.Y, m.width)
			return m, m.chose(c)
		case m.books.Open && msg.Button == tea.MouseLeft:
			var picked string
			m.books, picked = m.books.Click(msg.X, msg.Y, m.booksAt(), m.width)
			return m, shell.Pick(picked)
		case m.recent.Open && msg.Button == tea.MouseLeft:
			var query string
			if m.recent, query = m.recent.Click(msg.X, msg.Y, m.recallAt(), m.width); query != "" {
				s := m.search()
				s.Query = query
				m.setSearch(s)
			}
			return m, nil
		case m.gridded() && m.sheet().Layered():
			return m.sheetClick(msg)
		case msg.Button == tea.MouseLeft && shell.OnApp(msg.X, msg.Y, appLabel):
			m.appMenu = m.appMenu.Opened()
			return m, nil
		case msg.Button == tea.MouseLeft && m.books.On(msg.X, msg.Y, m.booksAt()):
			m.books = m.books.Opened()
			return m, nil
		case msg.Button == tea.MouseLeft && msg.Y == shell.Top && m.recalls() && msg.X == m.recallAt():
			m.recent = m.recent.Opened()
			return m, nil
		case msg.Button == tea.MouseLeft && msg.Y == shell.Top && msg.X >= m.searchAt():
			m.focusSearch(true)
			return m, nil
		}
		bar, ok := m.bar().Update(msg)
		switch {
		case ok:
			m.choose(bar)
		case m.gridded():
			return m.sheetClick(msg)
		case msg.Button == tea.MouseLeft:
			m.click(msg.Y)
		}
		m.fit()
	}
	return m, nil
}

// take keeps the latest run's checks, or kit's refusal, and keeps the cursor on the check it was on. The step stays
// as it is. An earlier run's answer is dropped.
func (m *Model) take(msg source.Answer[[]kit.Check]) {
	if _, landed := m.checking.Take(msg); landed != source.Shown {
		return
	}
	if msg.Err != nil {
		m.refusal = refusalOf(msg.Err)
		m.fit()
		return
	}
	key := m.cursorKey()
	m.refusal, m.checks = "", msg.Value
	m.cursor = m.find(m.step, key)
	for _, c := range m.checks {
		if c.OK {
			delete(m.notes, c.Key())
			delete(m.deferred, c.Key())
		}
	}
	m.fit()
}

// rows gives the indexes of the step's checks, in order, those its search lets through: whose id, line, or fix holds
// what is typed, without regard to case. The intro's are the links to the failed checks, in the order of their steps.
func (m Model) rows(step int) []int {
	query := m.searches[m.tabs[step].id].Query
	intro := m.isIntro(step)
	var out []int
	for i, c := range m.checks {
		mine := m.stepOf(c) == step
		if intro {
			mine = !c.OK
		}
		if mine && (matches(c.ID, query) || matches(c.Line, query) || matches(c.Fix, query)) {
			out = append(out, i)
		}
	}
	if intro {
		slices.SortStableFunc(out, func(a, b int) int { return m.stepOf(m.checks[a]) - m.stepOf(m.checks[b]) })
	}
	return out
}

// isIntro says whether the step is the intro.
func (m Model) isIntro(step int) bool { return m.tabs[step].id == introID }

// matches says whether a text holds what is typed in a search, without regard to case or escapes; anything holds
// nothing typed.
func matches(text, query string) bool {
	return strings.Contains(strings.ToLower(ansi.Strip(text)), strings.ToLower(query))
}

// search gives the selected step's search.
func (m Model) search() shell.Search { return m.searches[m.tabs[m.step].id] }

// setSearch keeps the selected step's search, the cursor on the check it was on when the search lets it through, else
// on the first it does, and the pane back at its top when what is typed changed.
func (m *Model) setSearch(s shell.Search) {
	id := m.tabs[m.step].id
	if s.Query != m.searches[id].Query {
		m.offsets[m.step] = 0
	}
	m.searches[id] = s
	if m.gridded() {
		m.sheet().Search(s.Query)
	}
	if m.cursor < len(m.checks) {
		m.cursor = m.find(m.step, m.checks[m.cursor].Key())
	}
	m.fit()
}

// focusSearch gives the selected step's search the keyboard, or takes it away.
func (m *Model) focusSearch(focused bool) {
	s := m.search()
	s.Focused = focused
	m.setSearch(s)
}

// recalls says whether the search, with the keyboard, offers searches to recall.
func (m Model) recalls() bool { return m.search().Focused && len(m.recent.Items) > 0 }

// recallAt gives the screen column of the search's Recall glyph.
func (m Model) recallAt() int { return shell.RecallAt(m.searchAt()) }

// searchAt gives the column the search starts at on the title line, as internal/shell puts every screen's.
func (m Model) searchAt() int { return m.menuBar().SearchAt(m.width) }

// searchLine gives the search as the title line draws it.
func (m Model) searchLine() string {
	return shell.RecallLine(m.search().Query, m.search().Focused, len(m.recent.Items) > 0)
}

// find gives the index of the check with the key when it is in the step, else the step's first check, else 0.
func (m Model) find(step int, key string) int {
	rows := m.rows(step)
	for _, i := range rows {
		if m.checks[i].Key() == key {
			return i
		}
	}
	if len(rows) > 0 {
		return rows[0]
	}
	return 0
}

// hasRows says whether the cursor moves over rows in the selected step: a step with checks, or the intro while it links
// to a failed check.
func (m Model) hasRows() bool {
	return m.tabs[m.step].checked || m.isIntro(m.step) && len(m.rows(m.step)) > 0
}

// checked says whether the selected step shows checks.
func (m Model) checked() bool { return m.tabs[m.step].checked }

// setTabs takes the steps the pages make. The selected step, and each step's place and offset, go with its id; when
// the selected step's id has no page, the first step is selected.
func (m *Model) setTabs(tabs []tab) {
	if len(tabs) == 0 {
		return
	}
	old, offsets, places, selected := m.tabs, m.offsets, m.places, m.tabs[m.step].id
	m.tabs, m.offsets, m.places = tabs, make([]int, len(tabs)), make([]string, len(tabs))
	for i, t := range tabs {
		if j := indexOf(old, t.id); j >= 0 {
			m.offsets[i], m.places[i] = offsets[j], places[j]
		}
	}
	if m.step = indexOf(tabs, selected); m.step < 0 {
		m.step, m.inPane = 0, false
	}
}

// refusalOf gives the line to show when the checks did not run: kit's line, or how kit died with nothing on stderr.
func refusalOf(err error) string {
	var exit *kit.ExitError
	if err.Error() != "" {
		return err.Error()
	}
	if errors.As(err, &exit) {
		return fmt.Sprintf("kit welcome --checks exited %d with nothing on stderr", exit.Code)
	}
	return "kit welcome --checks failed"
}

// fixed notes how a fix ended: nothing when it now holds, a warning when it still fails, kit's refusal on 2, and why
// it did not run when it could not.
func (m *Model) fixed(msg fixedMsg) {
	var exit *exec.ExitError
	switch {
	case msg.err == nil:
		delete(m.notes, msg.key)
	case errors.As(msg.err, &exit) && exit.ExitCode() == 1:
		m.notes[msg.key] = note{text: "the fix ran, and the check still fails"}
	case errors.As(msg.err, &exit) && exit.ExitCode() == 2:
		m.notes[msg.key] = note{text: fmt.Sprintf("kit refused the fix; quit to read what kit welcome --fix %s printed",
			msg.id), refusal: true}
	default:
		m.notes[msg.key] = note{text: fmt.Sprintf("kit welcome --fix %s: %v", msg.id, msg.err), refusal: true}
	}
}

// key takes a key press: the quitting keys, the app's menu's while it is open, = for it, - for the workspace menu,
// the bar's, then the pane's.
func (m Model) key(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "ctrl+c":
		m.interrupted = true
		return m, m.quit()
	case shell.Quit:
		return m, m.quit()
	}
	if m.appMenu.Open {
		var c shell.Choice
		m.appMenu, c = m.appMenu.Key(k)
		return m, m.chose(c)
	}
	if m.books.Open {
		var picked string
		m.books, picked = m.books.Key(k)
		return m, shell.Pick(picked)
	}
	if m.recent.Open {
		var query string
		if m.recent, query = m.recent.Key(k); query != "" {
			s := m.search()
			s.Query = query
			m.setSearch(s)
		}
		return m, nil
	}
	if s := m.search(); s.Focused {
		// the search has the keyboard, as shell.SearchKey says: what it does not take, as the arrows but down, is
		// nobody's; ctrl+r drops the searches remembered down; and what was typed is remembered when the search gives
		// the pane focus
		s, did := s.Key(k)
		switch did {
		case shell.Recalled:
			m.recent = m.recent.Opened()
			return m, nil
		case shell.Left:
			m.recent = m.recent.Remember(s.Query)
			m.inPane = true
		}
		m.setSearch(s)
		if did == shell.Left && m.gridded() {
			// the table's rows take the keyboard, as a search gives up a stage's to its rows
			m.sheet().FocusRows()
			m.inPane = m.sheet().Focused()
		}
		return m, nil
	}
	if m.gridded() && (m.inPane || m.sheet().Layered()) {
		// the step's table takes what it answers first, a layer of its every key
		before := m.sheet().Notice()
		took := m.sheet().Key(k)
		m.inPane = m.sheet().Focused()
		if took {
			return m, m.fading(before)
		}
	}
	switch k.String() {
	case "=":
		m.appMenu = m.appMenu.Opened()
		return m, nil
	case shell.Workspaces:
		m.books = m.books.Opened()
		return m, nil
	case "/":
		m.focusSearch(true)
		return m, nil
	}
	if bar, ok := m.bar().Update(k); ok {
		m.choose(bar)
		m.fit()
		return m, nil
	}
	if m.gridded() {
		// down or enter from the bar gives the step's table the keyboard, on its control line
		if key := k.String(); !m.inPane && (key == "down" || key == "enter") {
			m.sheet().Focus()
			m.inPane = m.sheet().Focused()
		}
		return m, nil
	}
	choosing := m.choosing()
	switch k.String() {
	case "esc":
		m.back()
	case "shift+tab":
		if choosing {
			m.choice = now
		}
	case "tab":
		if choosing {
			m.choice = later
		}
	case "up":
		m.up(1)
	case "down":
		m.down(1)
	case "pgup":
		m.page(-max(1, m.paneHeight()-1))
	case "pgdown":
		m.page(max(1, m.paneHeight()-1))
	case "home":
		m.page(-ends)
	case "end":
		m.page(ends)
	case "enter":
		switch {
		case !m.inPane:
			m.down(1)
		case m.onLink():
			m.follow()
		case choosing && m.choice == now:
			return m, m.now()
		case choosing:
			m.later()
		}
	case "n":
		if choosing {
			return m, m.now()
		}
	case "l":
		if choosing || m.showOnly() {
			m.later()
		}
	}
	m.fit()
	return m, nil
}

// sheetClick gives a click to the selected step's table, which then has the keyboard where it took it, and the notice
// it puts up fades in its while.
func (m Model) sheetClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	before := m.sheet().Notice()
	if m.sheet().Click(msg) {
		m.inPane = m.sheet().Focused()
	}
	return m, m.fading(before)
}

// fades is how long the screen waits for table.Fades: Fades, as a test waits less.
var fades = table.Fades

// fading gives the command that fades the selected step's table's notice in its while, where it put one up since
// before.
func (m Model) fading(before shell.Notice) tea.Cmd { return m.sheet().Notice().Fading(before, fades) }

// click takes a left click on the pane, on the screen line y: on a check's line, the cursor goes to that check and the
// pane has the keyboard; on a link in the intro, the screen goes to its check; on a page, the page has the keyboard. A
// … line, or a line that is no check's, does nothing.
func (m *Model) click(y int) {
	r, h := y-paneTop, m.paneHeight()
	switch {
	case r < 0 || r >= h:
	case !m.hasRows():
		m.inPane = true
	default:
		lines := m.lines()
		if k := pane.At(len(lines), h, m.offsets[m.step], r); k >= 0 && lines[k].row >= 0 {
			m.cursor, m.inPane, m.choice = lines[k].row, true, now
			if m.onLink() {
				m.follow()
			}
		}
	}
}

// onLink says whether the cursor is on a link in the intro, with the pane's keyboard.
func (m Model) onLink() bool {
	return m.inPane && m.isIntro(m.step) && slices.Contains(m.rows(m.step), m.cursor)
}

// follow goes from the link under the cursor to its check: the check's step selected, the cursor on the check, and the
// pane with the keyboard. A search on that step that hides the check is cleared.
func (m *Model) follow() {
	c := m.checks[m.cursor]
	m.places[m.step] = c.Key()
	m.step = m.stepOf(c)
	if !slices.Contains(m.rows(m.step), m.cursor) {
		delete(m.searches, m.tabs[m.step].id)
		m.offsets[m.step] = 0
	}
	m.inPane, m.choice = true, now
}

// back gives the bar the keyboard, and a step's rows go back to their top; a page keeps where it was scrolled to.
func (m *Model) back() {
	if m.inPane && m.hasRows() {
		m.offsets[m.step] = 0
	}
	m.inPane = false
}

// choosing says whether the cursor is on a row that offers now and later: a failed check with a fix kit runs, in a
// step with checks.
func (m Model) choosing() bool {
	if !m.checked() || !m.inPane || m.cursor >= len(m.checks) {
		return false
	}
	c := m.checks[m.cursor]
	return !c.OK && c.Fix != "" && c.Runs
}

// showOnly says whether the cursor is on a failed check whose fix kit shows and never runs.
func (m Model) showOnly() bool {
	if !m.checked() || !m.inPane || m.cursor >= len(m.checks) {
		return false
	}
	c := m.checks[m.cursor]
	return !c.OK && c.Fix != "" && !c.Runs
}

// cursorKey gives the key of the check under the cursor, or nothing when there is none.
func (m Model) cursorKey() string {
	if m.cursor < len(m.checks) {
		return m.checks[m.cursor].Key()
	}
	return ""
}

// choose takes what the bar chose: where its view is scrolled, the step it selected, and the keyboard. The cursor goes
// back to the check it was on in that step, else to the step's first check.
func (m *Model) choose(bar tabs.Bar) {
	m.barView = bar.Scroll
	if bar.Selected != m.step {
		m.places[m.step] = m.cursorKey()
		m.step = bar.Selected
		m.cursor = m.find(m.step, m.places[m.step])
		m.choice = now
	}
	m.inPane = !bar.Focused
	// a step's table holds no keyboard while the bar, or another step, has it
	selected := m.sheet()
	m.eachSheet(func(s *table.Sheet) {
		if !m.inPane || s != selected {
			s.Blur()
		}
	})
}

// up moves the cursor n rows up in a step with rows, and past its first row gives the bar the keyboard; on a page
// it scrolls, and at the top gives the bar the keyboard.
func (m *Model) up(n int) {
	if !m.inPane {
		return
	}
	if !m.hasRows() {
		if m.offsets[m.step] == 0 {
			m.inPane = false
		}
		m.scroll(-n)
		return
	}
	rows := m.rows(m.step)
	at := slices.Index(rows, m.cursor)
	if at-n < 0 {
		m.inPane, m.offsets[m.step] = false, 0
		return
	}
	m.cursor = rows[at-n]
	m.choice = now
}

// down moves the cursor n rows down in a step with rows, and on a page scrolls; from the bar it gives the pane the
// keyboard first, which takes one of the n.
func (m *Model) down(n int) {
	if !m.hasRows() {
		if !m.inPane {
			m.inPane = true
			n--
		}
		m.scroll(n)
		return
	}
	rows := m.rows(m.step)
	if len(rows) == 0 {
		return
	}
	at := max(0, slices.Index(rows, m.cursor))
	if !m.inPane {
		m.inPane = true
		n--
	}
	m.cursor = rows[min(at+n, len(rows)-1)]
	m.choice = now
}

// page moves the cursor n rows in a step with rows, or scrolls a page n lines, and never leaves the pane: it stops at
// the first row and the last, and at the page's ends; on the bar it does nothing.
func (m *Model) page(n int) {
	switch {
	case !m.inPane:
	case !m.hasRows():
		m.scroll(n)
	default:
		rows := m.rows(m.step)
		at := max(0, slices.Index(rows, m.cursor))
		m.cursor, m.choice = rows[min(max(at+n, 0), len(rows)-1)], now
	}
}

// ends is more lines or rows than a step ever has, so a page of it reaches the first or the last, as home and end do.
const ends = 1 << 30

// scroll moves a page's pane n lines, never past its ends.
func (m *Model) scroll(n int) {
	start, _, _, _ := pane.Window(len(m.lines()), m.paneHeight(), m.offsets[m.step]+n)
	m.offsets[m.step] = start
}

// now hands the terminal to the fix of the check under the cursor; when it ends, the checks run again.
func (m *Model) now() tea.Cmd {
	c := m.checks[m.cursor]
	delete(m.deferred, c.Key())
	m.choice = now
	dirs := c.Dirs
	if len(dirs) == 0 {
		dirs = []string{m.runner.Dir}
	}
	return m.fixIn(c.ID, c.Key(), dirs)
}

// fixingMsg says a fix ran in one checkout and holds there, and the checkouts it is still to run in.
type fixingMsg struct {
	id, key string
	dirs    []string
}

// fixIn hands the terminal to the check's fix in the first of the checkouts; when it holds there and more are left, a
// fixingMsg runs it in the next, else the fix is done, as fixedMsg says.
func (m Model) fixIn(id, key string, dirs []string) tea.Cmd {
	r := m.runner
	r.Dir = dirs[0]
	return m.exec(r.WelcomeFix(id), func(err error) tea.Msg {
		if err == nil && len(dirs) > 1 {
			return fixingMsg{id: id, key: key, dirs: dirs[1:]}
		}
		return fixedMsg{id: id, key: key, err: err}
	})
}

// later leaves the check under the cursor for later and moves to the step's next failed check not left, if there is
// one.
func (m *Model) later() {
	m.deferred[m.checks[m.cursor].Key()] = true
	if next := m.nextFailed(m.cursor); next >= 0 {
		m.cursor = next
	}
	m.choice = now
}

// nextFailed gives the step's first failed check after the check from, round the end, not left for later, or -1.
func (m Model) nextFailed(from int) int {
	rows := m.rows(m.step)
	n, at := len(rows), slices.Index(rows, from)
	for k := 1; k <= n; k++ {
		i := rows[(at+k+n)%n]
		if c := m.checks[i]; !c.OK && !m.deferred[c.Key()] && i != from {
			return i
		}
	}
	return -1
}
