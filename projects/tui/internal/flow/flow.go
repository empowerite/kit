// Package flow is kit flow on a terminal: the stages as a tab bar, internal/tabs, the selected stage's rows under it,
// and a row opened in place. It draws from kit flow <stage> --json and never parses kit's text. While the bar has the
// keyboard, left and right move along it; a stage's number, or a click on the bar, selects it from anywhere and gives
// the bar the keyboard. Down enters a stage's rows and up on the first row leaves them; in the rows, right opens a row
// and left closes it, space selects it as one of several, shift with up or down selects as it moves, and ctrl+a selects
// every row, and a click on a row puts the cursor there, and on the row under it opens or closes it; = drops the app's
// menu down, and : or ; the selected tab's, as a click on ≡ or ⋮ does; ? shows the sidebar beside the rows, the stage's
// panel over its legend, and again takes it away, as a click anywhere on the rule beside the rows does; tab gives the
// panel the keyboard, and / or a click on the search, always on the screen's top right, gives the search the keyboard,
// the sidebar shown or not. A click on a linked glyph, as an issue's before its title, opens its page in the browser,
// and one on a glyph that names a row of another stage, as underway's pull request, selects that stage with the row
// under the cursor. On a screen as wide as roomy, the sidebar shows from the start; on a narrower one it starts hidden,
// so a row's title has the room; once the person shows or hides it, it stays so for the session, through a resize. Esc
// goes back one level: from the panel, then from the rows to the bar; it leaves the sidebar as it is. Ctrl-c quits with
// 130 from anywhere, and alt-esc or super-esc, as a terminal that reports it sends command-esc, quits with 0. A stage's
// command has two minutes; past them it is a refusal, and quitting kills the commands still running.
package flow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// Options is how the screen opens: the stages in the picture's order, origin for the title and the URLs when known,
// whether the fetch is skipped, the filters backlog, underway, and changes take, whether every stage opens with its
// headers hidden, as they are shown otherwise, and whether the sidebar starts hidden at every width, as if the person
// had hidden it.
type Options struct {
	Stages    []string
	Origin    Origin
	NoFetch   bool
	Filters   []string
	NoHeaders bool
	NoSidebar bool
	Views     string // the file the person's views are kept in; none keeps them for the session alone
	NoViews   bool   // whether the panels offer no dropdown of views, as a test of the panel alone asks
	History   string // the file the searches are remembered in; none remembers them for the session alone
}

// loadedMsg brings a stage's JSON, or why kit did not print it, from the stage's load numbered gen.
type loadedMsg struct {
	stage string
	data  []byte
	err   error
	gen   int
}

// stage is one stage of the picture: its name, what kit printed for it, and where the person is in its rows.
type stage struct {
	name    string
	loading bool
	refusal string // kit's line, when the command failed
	issues  []issue
	under   []underway
	in      *incoming
	desk    *checkout
	trees   *worktrees
	pulls   []pull
	runs    []run
	rels    []release
	cursor  int             // the row under the cursor, kept while another stage is shown
	offset  int             // the first line of the pane
	open    map[string]bool // the rows opened, by key
	headers bool            // whether the names of its columns stand over its rows, as its tab's menu says
	headed  bool            // whether its header is open, every line of names that take more than one
	picked  map[string]bool // the rows selected, by key, several at once, which the tab's menu acts on
	// the view its panel's dropdown names, and the name typed over it while the person renames it
	viewName string
	naming   string
	renaming bool
	view     grid.Panel         // the stage's panel: its search, the columns hidden, whose rows show, and its cursor
	gen      int                // the stage's latest load: an answer from an earlier one is dropped
	ctx      context.Context    // the latest load's, inside the screen's; nil for the first load, which has the screen's
	stop     context.CancelFunc // ends the latest load's command, when r starts another
}

// known names the stages kittui draws.
var known = map[string]bool{"backlog": true, "underway": true, "incoming": true, "desk": true, "worktrees": true,
	"changes": true, "builds": true, "releases": true}

// roomy is the narrowest screen, in cells, the sidebar shows on from the start: on a narrower one it starts hidden,
// since beside it a row's title would have a few cells.
const roomy = 140

// deadline is how long a stage's command may take: past it, the stage is a refusal, and Refresh All runs it again.
const deadline = 2 * time.Minute

// Model is the picture on the screen.
type Model struct {
	runner        kit.Runner
	ctx           context.Context    // ends with the screen, and every stage's command with it
	cancel        context.CancelFunc // ends ctx
	limit         time.Duration      // how long a stage's command may take: the deadline, shorter in a test
	origin        Origin
	noFetch       bool
	filters       []string
	stages        []stage
	stage         int                    // the selected stage
	inRows        bool                   // whether the rows have the keyboard, not the bar
	inPanel       bool                   // whether the panel has the keyboard, before the rows or the bar
	sidebar       bool                   // whether the sidebar, the panel and the legend, shows beside the rows
	sideChosen    bool                   // whether the person showed or hid the sidebar, which a resize then keeps
	sideOffset    int                    // the sidebar's first line, when it is taller than the pane
	widest        int                    // the cells of the widest stage's panel or legend
	me            string                 // the caller's login, once kit has said it, which the panels' mine and others match
	asked         bool                   // whether kit has been asked for the caller's login
	browse        func(url string) error // opens a page in the browser
	notice        string                 // why a page did not open, or what kit did, in place of the keys until a key
	noted         bool                   // whether the notice says what was done, rather than why it was not
	ask           *ask                   // what a backlog action asks before kit runs, while it asks
	saved         map[string][]named     // the person's views, by stage, as their file keeps them
	viewsPath     string                 // that file, or none
	viewMenu      bool                   // whether the panel's dropdown of views is open
	viewAt        int                    // the line of that menu under the keyboard
	noHeaders     bool                   // whether the stages open with their headers hidden
	noViews       bool                   // whether the panels offer no dropdown of views
	recent        shell.Recent           // the searches remembered, every screen's, and their dropdown
	menu          bool                   // whether the selected tab's menu is open, which then has the keyboard
	menuAt        int                    // the tab's menu's item under the keyboard
	appMenu       shell.Menu             // the app's menu, internal/shell's, which has the keyboard while it is open
	drag          drag                   // the column a press on its line in the panel holds, until the release
	width, height int
	interrupted   bool
	built         *built // each stage's rows as last built, shared by the model's copies until it is replaced
}

// built holds each stage's rows as they were last built, and the width they were laid out for. A stage that loads, a
// refresh, or a new width replaces it, since a stage's rows may draw from another's, as the desk's from changes'.
type built struct {
	width  int
	stages map[int]shown
}

// rebuild drops the rows built so far; each is built again when it is next drawn.
func (m *Model) rebuild() {
	m.built = &built{width: m.paneWidth(), stages: map[int]shown{}}
}

// Open reads the stages from kit and origin from git, and opens the screen on them. The words are kit flow's own:
// --no-fetch, and the filters backlog, underway, and changes take. Kit's refusal of --stages is the error.
func Open(r kit.Runner, words []string) (Model, error) {
	stages, err := r.Stages()
	if err != nil {
		return Model{}, err
	}
	o := Options{Stages: stages, Origin: ReadOrigin(r.Dir), Views: viewsFile(), History: shell.RecentFile()}
	for _, word := range words {
		if word == "--no-fetch" {
			o.NoFetch = true
		} else {
			o.Filters = append(o.Filters, word)
		}
	}
	return New(r, o), nil
}

// New opens the screen on the options; every stage loads once Init starts it. A stage kittui does not know shows a
// line saying so, and runs nothing.
func New(r kit.Runner, o Options) Model {
	m := Model{runner: r, limit: deadline, origin: o.Origin, noFetch: o.NoFetch, filters: o.Filters, width: 80, height: 24,
		browse: browser, appMenu: shell.Menu{Here: shell.Flow, Own: ownItems}}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.sideChosen = o.NoSidebar
	m.noHeaders, m.viewsPath, m.noViews = o.NoHeaders, o.Views, o.NoViews
	m.recent = shell.Recent{Path: o.History}.Loaded()
	var err error
	if m.saved, err = readViews(o.Views); err != nil {
		m.notice = fmt.Sprintf("%s %s does not read as views, and is left as it is: %v", Failed, o.Views, err)
	}
	m.rebuild()
	for _, name := range o.Stages {
		s := stage{name: name, loading: true, open: map[string]bool{}, picked: map[string]bool{}, headers: !o.NoHeaders,
			viewName: Opening}
		if !known[name] {
			s.loading, s.refusal = false, fmt.Sprintf("kittui does not draw the stage '%s'", name)
		}
		m.stages = append(m.stages, s)
		m.widest = max(m.widest, pane.Width(legendLines(name)), grid.Panel{Columns: columns[name], Owned: owned[name]}.Width())
	}
	return m
}

// Init starts the loading: every stage at once, so none waits on incoming's fetch, which can take as long as the
// remotes do; the stages the fetch moves load again once it has fetched.
func (m Model) Init() tea.Cmd { return m.all() }

// Interrupted says whether ctrl-c ended the screen.
func (m Model) Interrupted() bool { return m.interrupted }

// Close ends the stages' commands still running, as quitting does, for a picture kittui leaves hidden as it ends.
func (m Model) Close() { m.cancel() }

// Update takes a message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if !m.sideChosen && m.sidebar != (m.width >= roomy) {
			// until the person chooses, the screen's width says whether the sidebar shows
			return m, m.placeSidebar(m.width >= roomy)
		}
		m.rebuild()
		m.fit()
	case loadedMsg:
		if !m.take(msg) || msg.stage != "incoming" {
			return m, nil
		}
		if in := m.find("incoming").in; in != nil && in.Fetched {
			// the fetch, at the start or on Refresh All, moved what some stages read: they load again
			return m, m.reloads()
		}
	case meMsg:
		if msg.err == nil {
			m.me = msg.login
			m.rebuild()
		}
	case browsedMsg:
		if msg.err != nil {
			m.notice, m.noted = fmt.Sprintf("%s could not open %s: %v", Failed, msg.url, msg.err), false
		}
	case actedMsg:
		return m, m.acted(msg)
	case tea.KeyPressMsg:
		m.notice = ""
		return m.key(msg)
	case tea.MouseWheelMsg:
		d := 0
		switch msg.Button {
		case tea.MouseWheelUp:
			d = -1
		case tea.MouseWheelDown:
			d = 1
		}
		switch {
		case msg.X > m.paneWidth()+1 && m.sidebar:
			m.scrollSide(d)
		case msg.X < m.paneWidth() && msg.Y >= paneTop && msg.Y < paneTop+m.paneHeight():
			m.scrollRows(d * wheel)
		}
	case tea.MouseMotionMsg:
		m.dragTo(msg.X, msg.Y)
	case tea.MouseReleaseMsg:
		m.release(msg.X, msg.Y)
	case tea.MouseClickMsg:
		if m.ask != nil {
			return m, nil
		}
		if m.viewMenu {
			m.viewMenuClick(msg.X, msg.Y)
			return m, nil
		}
		if m.recent.Open {
			var query string
			if m.recent, query = m.recent.Click(msg.X, msg.Y, m.recallAt(), m.width); query != "" {
				p := m.panel(m.stage)
				p.Query = query
				m.setPanel(p)
			}
			return m, nil
		}
		if msg.Y == 0 && len(m.stages) > 0 && m.inPanel && msg.X == m.recallAt() &&
			m.panel(m.stage).Recall && m.panel(m.stage).Cursor == 0 {
			m.recent = m.recent.Opened()
			return m, nil
		}
		switch side := m.paneWidth() + 1; {
		case m.appMenu.Open && msg.Button == tea.MouseLeft:
			var c shell.Choice
			m.appMenu, c = m.appMenu.Click(msg.X, msg.Y, m.paneWidth())
			return m, m.chose(c)
		case m.menu && msg.Button == tea.MouseLeft:
			return m, m.menuClick(msg.X, msg.Y)
		case msg.Button == tea.MouseLeft && m.onMenu(msg.X, msg.Y):
			return m, m.dropDown(false)
		case msg.Button == tea.MouseLeft && shell.OnApp(msg.X, msg.Y):
			m.appMenu = m.appMenu.Opened()
		case len(m.stages) > 0 && grouping(msg) && msg.Y >= paneTop && msg.Y < paneTop+m.paneHeight() &&
			msg.X < m.paneWidth():
			m.groupClick(msg.X, msg.Y, msg.Mod&tea.ModShift != 0)
		case msg.Button != tea.MouseLeft:
			if bar, ok := m.bar().Update(msg); ok {
				m.choose(bar)
			}
		case msg.X == side && msg.Y >= paneTop && msg.Y < m.sideHeight():
			return m, m.showSidebar(!m.sidebar)
		case msg.Y == 0 && msg.X >= m.searchAt(), m.sidebar && msg.X >= m.width-m.sideWidth():
			return m, m.clickPanel(msg.X, msg.Y)
		case len(m.stages) > 0 && msg.Y >= paneTop && msg.Y < m.rowsTop() && msg.X < m.paneWidth():
			m.clickHeader(msg.X, msg.Y, msg.Mod&tea.ModShift != 0, false)
		default:
			if bar, ok := m.bar().Update(msg); ok {
				m.choose(bar)
			} else {
				return m, m.click(msg.X, msg.Y)
			}
		}
	}
	return m, nil
}

// onMenu says whether a click at x, y is on the selected tab's menu glyph: its cells on the bar's middle line, or the
// blank cell either side of them, so a click a little off still drops the menu.
func (m Model) onMenu(x, y int) bool {
	at := m.bar().MenuAt()
	return at >= 0 && y == m.bar().Row+1 && x >= at-1 && x <= at+ansi.StringWidth(tabs.Menu)
}

// dropDown drops a menu down: the app's, or the selected tab's, which on backlog asks kit for the caller's login the
// first time, as Add to Pull Request needs it even while the sidebar, which asks too, is hidden.
func (m *Model) dropDown(app bool) tea.Cmd {
	if app {
		m.appMenu = m.appMenu.Opened()
		return nil
	}
	m.openMenu()
	if !m.menu || m.stages[m.stage].name != "backlog" {
		return nil
	}
	return m.askMe()
}

// menuFor says whether a key drops a menu down, and whether it is the app's: = the app's, and : or ; the tab's.
func menuFor(k tea.KeyPressMsg) (menu, app bool) {
	switch k.String() {
	case "=":
		return true, true
	case ":", ";":
		return true, false
	}
	return false, false
}

// browsedMsg says whether the browser was asked to open the page at url: err is why not.
type browsedMsg struct {
	url string
	err error
}

// browser is how a screen opens a page, which New gives it: the browser, as browse opens it.
var browser = browse

// browse opens a page in the browser: open on macOS, xdg-open elsewhere, without waiting for it.
func browse(url string) error {
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}
	return exec.Command(opener, url).Start()
}

// openPage asks the browser for the page at url.
func (m Model) openPage(url string) tea.Cmd {
	b := m.browse
	return func() tea.Msg { return browsedMsg{url, b(url)} }
}

// meMsg is kit's answer to the caller's login: the login, or why it could not be read.
type meMsg struct {
	login string
	err   error
}

// panel gives a stage's panel: what the person set in it, the columns it offers, those its rows group by on their own,
// whether its rows belong to people, the caller's login, and whether it has the keyboard.
func (m Model) panel(i int) grid.Panel {
	p := m.filter(i)
	p.Columns, p.Auto, p.AutoAt, p.Fill = m.offered(i), m.shown(i).auto, m.shown(i).said, m.sideWidth()-1
	s := m.stages[i]
	p.Recall = len(m.recent.Items) > 0
	p.Views, p.View, p.Naming = !m.noViews && len(columns[s.name]) > 0, s.viewName, s.renaming
	if s.renaming {
		p.View = s.naming
	}
	p.Changed = p.Views && m.changed(i)
	return p
}

// filter gives a stage's panel as its rows are let through: what the person set in it, whether its rows belong to
// people, and the caller's login; the columns it offers are drawn from the rows, so it has none.
func (m Model) filter(i int) grid.Panel {
	s := m.stages[i]
	p := s.view
	p.Owned, p.Me, p.Sortable, p.Groupable = owned[s.name], m.me, sortable[s.name], groupable[s.name]
	p.Focused = m.inPanel && i == m.stage
	return p
}

// setPanel keeps what the selected stage's panel became; the rows are let through again, and the cursor stays within
// them.
func (m *Model) setPanel(p grid.Panel) {
	m.stages[m.stage].view = p
	m.rebuild()
	m.clamp(m.stage)
	m.fit()
}

// enterPanel gives the panel the keyboard: the search under it, as it is, shown or not, when find says so; else the
// panel's control under it, the sidebar shown, which asks kit for the caller's login the first time.
func (m *Model) enterPanel(find bool) tea.Cmd {
	if len(m.stages) == 0 {
		return nil
	}
	if find {
		m.inPanel = true
		m.stages[m.stage].view = m.panel(m.stage).Find()
		return nil
	}
	cmd := m.showSidebar(true)
	m.inPanel = true
	return cmd
}

// clickPanel takes a click on a line of the screen's right, from its first: on the search, the panel has the keyboard
// and the search under it, as / does; on one of the panel's other controls, the panel has the keyboard and acts on it
// as space does; on the … above or … more, the sidebar scrolls a page.
func (m *Model) clickPanel(x, line int) tea.Cmd {
	if len(m.stages) == 0 {
		return nil
	}
	if line == 0 {
		return m.enterPanel(true)
	}
	line--
	n, h := m.scrolled()
	start, end, above, below := pane.Window(n, h, m.sideOffset)
	k := pane.At(n, h, m.sideOffset, line)
	switch {
	case above && line == 0:
		m.scrollSide(-max(1, h-2))
	case below && line == end-start+btoi(above):
		m.scrollSide(max(1, h-2))
	case k >= 0:
		p := m.panel(m.stage)
		if p.ViewAt(k + 1) {
			return m.viewClick(x)
		}
		if column := p.ColumnAt(k + 1); column != "" {
			// a column's line is held, to drag or to toggle on the release; it takes the keyboard now
			p, _ = p.Point(k + 1)
			m.inPanel, m.drag = true, drag{column: column}
			m.setPanel(p)
			return nil
		}
		if p, ok := p.Click(k + 1); ok {
			m.inPanel = true
			m.setPanel(p)
		}
	}
	return nil
}

// grouping says whether a click groups rather than sorts or selects: with ctrl or option held, or with the right
// button, as a terminal on macOS may send ctrl with a click.
func grouping(msg tea.MouseClickMsg) bool {
	return msg.Mod&(tea.ModCtrl|tea.ModAlt) != 0 || msg.Button == tea.MouseRight
}

// groupClick takes a click that groups, as grouping says: on a column's name in the header, the rows group by it, as
// clickHeader says; on a group's heading, they no longer group by its column, the other groups kept.
func (m *Model) groupClick(x, y int, nest bool) {
	if y < m.rowsTop() {
		m.clickHeader(x, y, nest, true)
		return
	}
	lines, s := m.lines(m.stage), &m.stages[m.stage]
	k := pane.At(len(lines), m.rowsHeight(), s.offset, y-m.rowsTop())
	if k < 0 || lines[k].row < 0 {
		return
	}
	p := m.panel(m.stage)
	if column := p.Heading(m.rows(m.stage)[lines[k].row]); column != "" {
		m.setPanel(p.Grouped(column, true))
	}
}

// clickHeader takes a click on the header, which is no row: on its gutter it opens the header, or closes it, when it
// hides lines; on a name of a column that sorts, it sorts by it alone, or, with also, as a shift-click asks, by it
// after the others, as the panel's Sorted says; with group, a name groups the rows by its column alone, or, with also,
// within the groups they have, as the panel's Grouped says; elsewhere it does nothing.
func (m *Model) clickHeader(x, y int, also, group bool) {
	s := &m.stages[m.stage]
	x -= m.indent(m.stage)
	if _, hides := m.header(); hides && !group && y == paneTop && x >= 0 && x < 3 {
		s.headed = !s.headed
		m.fit()
		return
	}
	lines, _ := m.header()
	if y >= paneTop+len(lines)-1 {
		return
	}
	if j := grid.HeaderAt(m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), x); j >= 0 {
		column, p := m.shownColumns(m.stage)[j], m.panel(m.stage)
		if group {
			p = p.Grouped(column, also)
		} else {
			p = p.Sorted(column, also)
		}
		m.setPanel(p)
	}
}

// selectRow selects the row under the cursor, as one of several, or unselects it, while the rows have the keyboard;
// with keep, it selects it whether it was or not, as shift with up or down selects as it moves. A heading is no row to
// select.
func (m *Model) selectRow(keep bool) {
	if !m.inRows || len(m.stages) == 0 {
		return
	}
	s, rows := &m.stages[m.stage], m.rows(m.stage)
	if s.cursor >= len(rows) || rows[s.cursor].Aside {
		return
	}
	key := rows[s.cursor].Key
	if keep || !s.picked[key] {
		s.picked[key] = true
	} else {
		delete(s.picked, key)
	}
}

// selectAll selects every row the stage shows, or, when every one is, none.
func (m *Model) selectAll() {
	if len(m.stages) == 0 {
		return
	}
	s, rows := &m.stages[m.stage], m.rows(m.stage)
	all := m.selected() == counted(rows)
	clear(s.picked)
	if all {
		return
	}
	for _, r := range rows {
		if !r.Aside {
			s.picked[r.Key] = true
		}
	}
}

// selectedRows gives the selected stage's rows that are selected, of those it shows.
func (m Model) selectedRows() []row {
	if len(m.stages) == 0 {
		return nil
	}
	var out []row
	for _, r := range m.rows(m.stage) {
		if m.stages[m.stage].picked[r.Key] && !r.Aside {
			out = append(out, r)
		}
	}
	return out
}

// selected gives how many of the selected stage's rows shown are selected.
func (m Model) selected() int { return len(m.selectedRows()) }

// drag is the column whose line in the panel a press holds, and whether a drag has moved it since.
type drag struct {
	column string
	moved  bool
}

// panelLine gives the line of the panel, as its Lines numbers them, that a screen cell falls on: the search's, 0, on
// the title line, else one of the lines under it as the sidebar is scrolled; -1 off the sidebar, or on a … line.
func (m Model) panelLine(x, y int) int {
	switch {
	case !m.sidebar || len(m.stages) == 0 || x < m.width-m.sideWidth():
		return -1
	case y == 0:
		return 0
	}
	n, h := m.scrolled()
	start, end, above, below := pane.Window(n, h, m.sideOffset)
	if line := y - 1; above && line == 0 || below && line == end-start+btoi(above) {
		return -1
	}
	if k := pane.At(n, h, m.sideOffset, y-1); k >= 0 {
		return k + 1
	}
	return -1
}

// dragTo moves the column held, while the mouse drags it, to where the column whose line it is over stands, the
// keyboard with it; over any other line it moves nothing.
func (m *Model) dragTo(x, y int) {
	if m.drag.column == "" {
		return
	}
	line := m.panelLine(x, y)
	p := m.panel(m.stage)
	at := p.ColumnAt(line)
	if at == "" || at == m.drag.column {
		return
	}
	p, _ = p.Move(m.drag.column, at).Point(line)
	m.drag.moved = true
	m.setPanel(p)
}

// release ends a hold on a column's line: a hold no drag moved, released on the checkbox of the line it began on,
// toggles it, as a click on any other control acts on it; released on the column's name, which takes the keyboard to
// sort or group by it, it toggles nothing.
func (m *Model) release(x, y int) {
	held := m.drag
	m.drag = drag{}
	if held.column == "" || held.moved || x > m.width-m.sideWidth()+1 {
		return
	}
	line := m.panelLine(x, y)
	if p := m.panel(m.stage); p.ColumnAt(line) == held.column {
		p, _ = p.Click(line)
		m.setPanel(p)
	}
}

// scrolled gives how many of the sidebar's lines scroll, all but the search on the screen's first line, and the lines
// they have, all under it down to the keys bar.
func (m Model) scrolled() (n, h int) { return len(m.sideLines()) - 1, m.sideHeight() - 1 }

// btoi gives 1 for true, 0 for false.
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// wheel is the lines a turn of the mouse wheel scrolls the rows.
const wheel = 3

// scrollRows scrolls the rows n lines, down when n is positive, never past their ends; the cursor stays where it is,
// until a key moves it and the pane follows it again.
func (m *Model) scrollRows(n int) {
	if len(m.stages) == 0 {
		return
	}
	s := &m.stages[m.stage]
	s.offset, _, _, _ = pane.Window(len(m.lines(m.stage)), m.rowsHeight(), s.offset+n)
}

// scrollSide scrolls the sidebar under the search n lines, down when n is positive, never past its ends.
func (m *Model) scrollSide(n int) {
	lines, h := m.scrolled()
	m.sideOffset, _, _, _ = pane.Window(lines, h, m.sideOffset+n)
}

// fitSide scrolls the sidebar so the panel's control under the keyboard shows; the search always does, and on it the
// sidebar is back at its top.
func (m *Model) fitSide() {
	_, at := m.panel(m.stage).Lines()
	n, h := m.scrolled()
	line := slices.Index(at, m.stages[m.stage].view.Cursor) - 1
	if line < 0 {
		m.sideOffset = 0
	}
	for line >= 0 {
		start, end, _, _ := pane.Window(n, h, m.sideOffset)
		switch {
		case line < start:
			m.sideOffset = line
		case line >= end:
			m.sideOffset++
		default:
			return
		}
	}
}

// askMe runs kit flow --me, once, for the limit at most.
func (m *Model) askMe() tea.Cmd {
	if m.asked {
		return nil
	}
	m.asked = true
	return m.whoami()
}

// whoami asks kit for the caller's login.
func (m Model) whoami() tea.Cmd {
	r, ctx, limit := m.runner, m.ctx, m.limit
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		login, err := r.Me(ctx)
		return meMsg{login, err}
	}
}

// key takes a key press: the quitting keys, the open menu's while one is open, the panel's while it has the keyboard,
// = for the app's menu and : or ; for the tab's, the bar's, then the rows' and the sidebar's.
func (m Model) key(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "ctrl+c":
		m.interrupted = true
		return m, m.quit()
	case "alt+esc", "super+esc":
		return m, m.quit()
	}
	if m.ask != nil {
		return m.askKey(k)
	}
	if m.viewMenu {
		return m.viewMenuKey(k)
	}
	if m.recent.Open {
		var query string
		if m.recent, query = m.recent.Key(k); query != "" {
			p := m.panel(m.stage)
			p.Query = query
			m.setPanel(p)
		}
		return m, nil
	}
	if m.appMenu.Open {
		return m.appKey(k)
	}
	if m.menu {
		return m.menuKey(k)
	}
	if m.inPanel {
		return m.panelKey(k)
	}
	if menu, app := menuFor(k); menu {
		return m, m.dropDown(app)
	}
	if bar, ok := m.bar().Update(k); ok {
		m.choose(bar)
		return m, nil
	}
	switch k.String() {
	case "tab":
		return m, m.enterPanel(false)
	case "/":
		return m, m.enterPanel(true)
	case "esc":
		if m.inRows {
			s := &m.stages[m.stage]
			s.cursor, s.offset, m.inRows = 0, 0, false
		}
		m.fit()
	case "?":
		return m, m.showSidebar(!m.sidebar)
	case "space":
		m.selectRow(false)
	case "ctrl+a":
		m.selectAll()
	case "shift+up", "shift+down":
		if m.inRows {
			m.selectRow(true)
			if k.String() == "shift+up" {
				m.up(1)
			} else {
				m.down(1)
			}
			m.selectRow(true)
		}
	case "up":
		m.up(1)
	case "down":
		m.down(1)
	case "pgup":
		m.page(-1)
	case "pgdown":
		m.page(1)
	case "right":
		m.open(true)
	case "left":
		m.open(false)
	}
	return m, nil
}

// panelKey takes a key while the panel has the keyboard: tab and esc give it back to the rows or the bar, as they had
// it; page up and down scroll the sidebar, and so does down past the last control, so the legend under the panel can
// be read; the panel takes what it can, as typing in its search, the sidebar following its cursor; a number the bar
// takes selects a stage, and gives the bar the keyboard. While the sidebar is hidden the search stands alone: down and
// a page at a time have nothing to move to.
func (m Model) panelKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if len(m.stages) == 0 {
		return m.panelKeys(k)
	}
	before := m.panel(m.stage)
	searching := before.Focused && before.Cursor == 0
	if searching && k.String() == "ctrl+r" {
		m.recent = m.recent.Opened()
		return m, nil
	}
	next, cmd := m.panelKeys(k)
	if after := next.(Model); searching && (!after.inPanel || after.panel(after.stage).Cursor != 0) {
		// the search gave up the keyboard: what was typed is remembered
		after.recent = after.recent.Remember(before.Query)
		return after, cmd
	}
	return next, cmd
}

// recallAt gives the screen column of the search's Recall glyph, as the title line draws it; -1 with no stage.
func (m Model) recallAt() int {
	if len(m.stages) == 0 {
		return -1
	}
	return shell.RecallAt(m.searchAt(), m.panel(m.stage).Query)
}

// panelKeys takes a key while the panel has the keyboard, as panelKey says.
func (m Model) panelKeys(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	h := max(1, m.sideHeight()-3)
	if m.panel(m.stage).OnView() {
		// the dropdown takes its keys first: esc stops a rename before it leaves the panel
		if cmd, ok := m.viewKey(k); ok {
			return m, cmd
		}
	}
	switch k.String() {
	case "tab", "esc":
		m.inPanel = false
		return m, nil
	case "down", "pgdown", "pgup":
		if !m.sidebar {
			return m, nil
		}
	}
	switch k.String() {
	case "pgdown":
		m.scrollSide(h)
		return m, nil
	case "pgup":
		m.scrollSide(-h)
		return m, nil
	}
	before := m.panel(m.stage)
	if menu, app := menuFor(k); menu && before.Cursor != 0 {
		// off the search, the menus' keys drop them down, as they do from the rows
		return m, m.dropDown(app)
	}
	if p, ok := before.Update(k); ok {
		m.setPanel(p)
		if k.String() == "down" && p.Cursor == before.Cursor {
			// past the last control, down scrolls on to the legend under the panel
			m.scrollSide(1)
		} else {
			m.fitSide()
		}
		return m, nil
	}
	m.inPanel = false
	if bar, ok := m.bar().Update(k); ok {
		m.choose(bar)
		return m, nil
	}
	m.inPanel = true
	return m, nil
}

// showSidebar shows the sidebar beside the rows, or takes it away, as the person asks, which a resize then keeps; the
// rows are laid out again for the width left, and the panel loses the keyboard with it. Shown the first time, it asks
// kit for the caller's login.
func (m *Model) showSidebar(on bool) tea.Cmd {
	m.sideChosen = true
	m.inPanel = m.inPanel && on
	return m.placeSidebar(on)
}

// placeSidebar shows the sidebar beside the rows, or takes it away, the rows laid out again for the width left; a
// search that has the keyboard keeps it, as it stands on the screen either way. Shown the first time, it asks kit for
// the caller's login.
func (m *Model) placeSidebar(on bool) tea.Cmd {
	m.sidebar = on
	m.inPanel = m.inPanel && (on || len(m.stages) > 0 && m.stages[m.stage].view.Cursor == 0)
	m.rebuild()
	m.fit()
	if on {
		return m.askMe()
	}
	return nil
}

// choose takes what the bar chose: the stage it selected, where the cursor is as it was left, and the keyboard.
func (m *Model) choose(bar tabs.Bar) {
	if bar.Selected != m.stage {
		m.sideOffset = 0
	}
	m.stage, m.inRows, m.inPanel = bar.Selected, !bar.Focused, false
	m.fit()
}

// quit ends the screen, and the stages' commands still running with it.
func (m Model) quit() tea.Cmd {
	m.cancel()
	return tea.Quit
}

// find gives the stage by name, or nil when it is not in the picture.
func (m Model) find(name string) *stage {
	for i := range m.stages {
		if m.stages[i].name == name {
			return &m.stages[i]
		}
	}
	return nil
}

// args gives the words a stage's command takes after --json: --no-fetch for incoming, the filters for backlog,
// underway, and changes, nothing for the rest.
func (m Model) args(name string) []string {
	switch name {
	case "incoming":
		if m.noFetch {
			return []string{"--no-fetch"}
		}
	case "backlog", "underway", "changes":
		return m.filters
	}
	return nil
}

// load runs a stage's command: kit flow <stage> --json, with the stage's words, for the limit at most, as the stage's
// latest load, which r supersedes. Fresh says the stage reads the forge afresh, as r asks: kit shares one read of the
// open issues and pull requests a minute between the stages that need it, and a load without it takes that read.
func (m Model) load(name string, fresh bool) tea.Cmd {
	r, args, ctx, limit, gen := m.runner, m.args(name), m.ctx, m.limit, 0
	if fresh && shared[name] {
		args = append(slices.Clone(args), "--fresh")
	}
	if s := m.find(name); s != nil {
		gen = s.gen
		if s.ctx != nil {
			ctx = s.ctx
		}
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		data, err := r.Status(ctx, name, args)
		return loadedMsg{stage: name, data: data, err: err, gen: gen}
	}
}

// all runs every stage kittui draws, at once.
func (m Model) all() tea.Cmd {
	var cmds []tea.Cmd
	for _, s := range m.stages {
		if known[s.name] {
			cmds = append(cmds, m.load(s.name, false))
		}
	}
	return tea.Batch(cmds...)
}

// restart runs a stage's command again as its latest load, reading the forge afresh, ending the one before when it
// still runs, and shows the stage loading until it answers.
func (m *Model) restart(name string) tea.Cmd {
	cmd, s := m.again(name, true), m.find(name)
	s.loading, s.refusal = true, ""
	s.issues, s.under, s.in, s.desk, s.trees, s.pulls, s.runs, s.rels = nil, nil, nil, nil, nil, nil, nil, nil
	m.rebuild()
	return cmd
}

// again runs a stage's command again as its latest load, fresh or not as load takes it, ending the one before when it
// still runs; the stage keeps what it shows until the answer replaces it.
func (m *Model) again(name string, fresh bool) tea.Cmd {
	s := m.find(name)
	if s.stop != nil {
		s.stop()
	}
	s.ctx, s.stop = context.WithCancel(m.ctx)
	s.gen++
	return m.load(name, fresh)
}

// shared names the stages that read kit's shared snapshot of the open issues and pull requests, which --fresh reads
// afresh.
var shared = map[string]bool{"backlog": true, "underway": true, "changes": true}

// moved names the stages whose rows the fetch moves: the desk's and the worktrees' counts against the default branch,
// the issues underway, which the pull requests the fetch brings close, and the backlog's branches, which it reads as
// the fetch left them.
var moved = []string{"backlog", "desk", "worktrees", "underway"}

// reloads runs again each stage the fetch moves, when it is in the picture, each keeping its rows on the screen until
// the new ones come. They take kit's snapshot of the forge, which the fetch does not move.
func (m *Model) reloads() tea.Cmd {
	var cmds []tea.Cmd
	for _, name := range moved {
		if m.find(name) != nil {
			cmds = append(cmds, m.again(name, false))
		}
	}
	return tea.Batch(cmds...)
}

// take keeps what a stage's command printed, decoded by the stage, or the refusal when it failed, and says whether it
// did. The answer of a load a later one superseded is dropped. The cursor stays within the rows that came.
func (m *Model) take(msg loadedMsg) bool {
	s := m.find(msg.stage)
	if s == nil || msg.gen != s.gen {
		return false
	}
	s.loading = false
	err := msg.err
	if err == nil {
		err = s.decode(msg.data)
	}
	if err != nil {
		s.refusal = refusalOf(s.name, err)
	}
	m.rebuild()
	for i := range m.stages {
		m.clamp(i)
	}
	m.fit()
	return true
}

// decode keeps what the stage's command printed. A document of another shape, as kit prints above repositories, is
// an error naming it, and JSON that does not decode is one too.
func (s *stage) decode(data []byte) error {
	if err := shape(s.name, data); err != nil {
		return err
	}
	var err error
	switch s.name {
	case "backlog":
		err = json.Unmarshal(data, &s.issues)
	case "underway":
		err = json.Unmarshal(data, &s.under)
	case "incoming":
		s.in = &incoming{}
		err = json.Unmarshal(data, s.in)
	case "desk":
		s.desk = &checkout{}
		err = json.Unmarshal(data, s.desk)
	case "worktrees":
		s.trees = &worktrees{}
		err = json.Unmarshal(data, s.trees)
	case "changes":
		err = json.Unmarshal(data, &s.pulls)
	case "builds":
		err = json.Unmarshal(data, &s.runs)
	case "releases":
		err = json.Unmarshal(data, &s.rels)
	}
	if err != nil {
		return fmt.Errorf("kit flow %s --json: %v", s.name, err)
	}
	return nil
}

// expected is each stage's document, as kit flow's page says: an array of objects for backlog, underway,
// changes, builds, and releases, each object with the stage's key fields; an object for incoming, desk, and
// worktrees, incoming's with its remotes.
var expected = map[string]struct {
	what  string
	array bool
	keys  []string
}{
	"backlog":   {"an array of issues", true, []string{"number"}},
	"underway":  {"an array of issues underway", true, []string{"number", "pr"}},
	"changes":   {"an array of pull requests", true, []string{"number"}},
	"builds":    {"an array of builds", true, []string{"id"}},
	"releases":  {"an array of releases", true, []string{"version"}},
	"incoming":  {"an object of remotes", false, []string{"remotes"}},
	"desk":      {"an object", false, nil},
	"worktrees": {"an object", false, nil},
}

// shape refuses a document that is not the stage's: an array where an object is expected, an object where an array
// is, an array whose objects lack one of the stage's key fields, as the array of desks kit prints above desks, or an
// object without its key field, as incoming's before every remote. Anything else is left to the decoder.
func shape(name string, data []byte) error {
	want, printed := expected[name], ""
	switch data = bytes.TrimSpace(data); {
	case len(data) == 0:
	case data[0] == '[' && !want.array:
		printed = "an array"
	case data[0] == '{' && want.array:
		printed = "an object"
	case data[0] == '[':
		var objects []map[string]json.RawMessage
		if json.Unmarshal(data, &objects) != nil {
			return nil
		}
		for _, o := range objects {
			if key := lacking(o, want.keys); key != "" {
				printed = fmt.Sprintf("an array of objects without %q", key)
				break
			}
		}
	case data[0] == '{':
		var object map[string]json.RawMessage
		if json.Unmarshal(data, &object) != nil {
			return nil
		}
		if key := lacking(object, want.keys); key != "" {
			printed = fmt.Sprintf("an object without %q", key)
		}
	}
	if printed == "" {
		return nil
	}
	return fmt.Errorf("kit flow %s --json printed %s where %s was expected; run kit flow inside a repository",
		name, printed, want.what)
}

// lacking gives the first of the keys an object lacks, or nothing.
func lacking(o map[string]json.RawMessage, keys []string) string {
	for _, key := range keys {
		if _, ok := o[key]; !ok {
			return key
		}
	}
	return ""
}

// refusalOf gives a stage's refusal from its error: that the stage took longer than the deadline and the menu's
// Refresh All runs it again; else kit's line; and when kit died with nothing on stderr, how: exited with its code, or killed.
func refusalOf(name string, err error) string {
	var exit *kit.ExitError
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Sprintf("kit flow %s took longer than %d minutes; the menu's Refresh All runs it again", name,
			int(deadline.Minutes()))
	case err.Error() != "":
		return err.Error()
	case errors.As(err, &exit) && exit.Code >= 0:
		return fmt.Sprintf("kit flow %s exited %d with nothing on stderr", name, exit.Code)
	}
	return fmt.Sprintf("kit flow %s was killed", name)
}

// clamp keeps a stage's cursor within its rows.
func (m *Model) clamp(i int) {
	s := &m.stages[i]
	if n := len(m.rows(i)); s.cursor >= n {
		s.cursor = max(0, n-1)
	}
}

// up moves the cursor n rows up; past the first row, it gives the bar the keyboard, and the pane goes to its top.
func (m *Model) up(n int) {
	if !m.inRows || len(m.stages) == 0 {
		return
	}
	s := &m.stages[m.stage]
	if s.cursor-n < 0 {
		s.cursor, s.offset, m.inRows = 0, 0, false
	} else {
		s.cursor -= n
	}
	m.fit()
}

// down moves the cursor n rows down; from the bar, it enters the rows, when the stage has any, past the headings of
// groups over the row under the cursor, so it lands on a row.
func (m *Model) down(n int) {
	if len(m.stages) == 0 {
		return
	}
	s := &m.stages[m.stage]
	all := m.rows(m.stage)
	rows := len(all)
	if rows == 0 {
		return
	}
	if !m.inRows {
		m.inRows = true
		n--
		for s.cursor+n < rows-1 && grid.Heads(all[s.cursor+n]) {
			n++
		}
	}
	s.cursor = min(s.cursor+n, rows-1)
	m.fit()
}

// open opens the row under the cursor, or closes it; a leaf stays as it is.
func (m *Model) open(open bool) {
	if !m.inRows || len(m.stages) == 0 {
		return
	}
	s := &m.stages[m.stage]
	rows := m.rows(m.stage)
	if s.cursor >= len(rows) || !grid.Opens(rows, m.names(m.stage), s.cursor, m.gridWidth(m.stage)) {
		return
	}
	s.open[rows[s.cursor].Key] = open
	m.fit()
}

// click takes a left click on the pane's rows: the cursor goes to the row clicked, and the rows have the keyboard; a
// click on a linked glyph of a row's first line, as an issue's before its title, opens its page in the browser; one
// on the first line of the row already under the cursor, or on the gutter of a row's first line, its ▸ or ▾, opens
// the row, or closes it, as one on that first line pinned on top, once the row has scrolled it away, does. A click on
// a … line, or on a line about the stage, does nothing.
func (m *Model) click(x, y int) tea.Cmd {
	if len(m.stages) == 0 || x >= m.paneWidth() {
		return nil
	}
	lines, s := m.lines(m.stage), &m.stages[m.stage]
	if y < m.rowsTop() {
		m.clickHeader(x, y, false, false)
		return nil
	}
	k := pane.At(len(lines), m.rowsHeight(), s.offset, y-m.rowsTop())
	if y == m.rowsTop() && m.head(lines, m.rowsHeight()) != "" {
		// the pinned line is the first of the row under the cursor, in place of the … above
		k = slices.IndexFunc(lines, func(l line) bool { return l.row == s.cursor })
	}
	if k < 0 || lines[k].row < 0 {
		return nil
	}
	rows := m.rows(m.stage)
	row, first := lines[k].row, k == 0 || lines[k-1].row != lines[k].row
	link := ""
	if first {
		if !rows[row].Aside {
			x -= m.indent(m.stage)
		}
		link = grid.LinkAt(rows, m.names(m.stage), row, m.gridWidth(m.stage), x)
	}
	toggle := first && link == "" && (x >= 0 && x < 3 || m.inRows && row == s.cursor)
	s.cursor, m.inRows, m.inPanel = row, true, false
	switch {
	case strings.HasPrefix(link, jumps):
		m.fit()
		m.jump(strings.TrimPrefix(link, jumps))
	case link != "":
		m.fit()
		return m.openPage(link)
	case toggle:
		m.open(!s.open[rows[row].Key])
	default:
		m.fit()
	}
	return nil
}

// jump follows a link to a row of another stage, its name and the row's key: the stage is selected, its rows have the
// keyboard, and the cursor is on the row, or where it was when the panel hides the row or the stage has not loaded
// it.
func (m *Model) jump(to string) {
	name, key, _ := strings.Cut(to, "/")
	i := slices.IndexFunc(m.stages, func(s stage) bool { return s.name == name })
	if i < 0 {
		return
	}
	m.choose(tabs.Bar{Selected: i})
	if k := slices.IndexFunc(m.rows(i), func(r row) bool { return r.Key == key }); k >= 0 {
		m.stages[i].cursor = k
	}
	m.fit()
}

// page moves a page down or up. Inside a row taller than the pane, as an open pull request with many issues is, it
// first scrolls through that row, so every line it hides can be reached; at the row's end it moves the cursor a page
// of rows. A page never leaves the rows: up stops at the first row, and down at the last; on the bar it does nothing.
func (m *Model) page(d int) {
	h := max(1, m.rowsHeight()-1)
	if !m.inRows || len(m.stages) == 0 {
		return
	}
	s := &m.stages[m.stage]
	all := m.lines(m.stage)
	first, last := -1, -1
	for k, l := range all {
		if l.row == s.cursor {
			if first < 0 {
				first = k
			}
			last = k
		}
	}
	start, end, _, _ := pane.Window(len(all), m.rowsHeight(), s.offset)
	switch {
	case first >= 0 && d > 0 && last >= end:
		s.offset, _, _, _ = pane.Window(len(all), m.rowsHeight(), s.offset+h)
	case first >= 0 && d < 0 && first < start:
		s.offset, _, _, _ = pane.Window(len(all), m.rowsHeight(), max(first, s.offset-h))
	case d > 0:
		m.down(h)
	default:
		s.cursor = max(0, s.cursor-h)
		m.fit()
	}
}

// fit scrolls the pane so the row under the cursor shows, with what it opens to when that fits, and never past the
// last line.
func (m *Model) fit() {
	if len(m.stages) == 0 {
		return
	}
	s := &m.stages[m.stage]
	lines := m.lines(m.stage)
	h := m.rowsHeight()
	s.offset, _, _, _ = pane.Window(len(lines), h, s.offset)
	if !m.inRows {
		return
	}
	first, last := -1, -1
	for k, l := range lines {
		if l.row == s.cursor {
			if first < 0 {
				first = k
			}
			last = k
		}
	}
	if first < 0 {
		return
	}
	// Scroll until the cursor's row shows whole, or its first line and as much as the pane holds; a … line takes the
	// place of a row at either edge, so the window is asked rather than counted.
	for {
		start, end, _, _ := pane.Window(len(lines), h, s.offset)
		switch {
		case first < start:
			s.offset = first
		case last >= end && first > start:
			s.offset++
		default:
			return
		}
	}
}
