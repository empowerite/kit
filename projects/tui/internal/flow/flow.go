// Package flow is kit flow on a terminal: the stages as a tab bar, internal/tabs, the selected stage's rows under it,
// and a row opened in place. It draws from kit flow <stage> --json and never parses kit's text. While the bar has the
// keyboard, left and right move along it; a stage's number, or a click on the bar, selects it from anywhere and gives
// the bar the keyboard. Down enters a stage's rows and up on the first row leaves them; in the rows, right opens a row
// and left closes it, space selects it as one of several, shift with up or down selects as it moves, and ctrl+a selects
// every row, and a click on a row puts the cursor there, and on the row under it opens or closes it; = drops the app's
// menu down, and : or ; the selected tab's, as a click on ≡ or ⋮ does; F1 or ? shows the legend over the rows, and again
// hides it; / or a click on the search, always on the screen's top right, gives the search the keyboard. A click on a
// linked glyph, as an issue's before its title, opens its page in the browser, and one on a glyph that names a row of
// another stage, as underway's pull request, selects that stage with the row under the cursor. The rows take the
// screen's width. Maximize, in the tab's menu, gives the rows the bar's lines but one, the tab named in the title, and
// Restore puts the bar back. Esc goes back one level: from the search, then from the rows to the bar. Ctrl-c quits with
// 130 from anywhere, and alt-esc or super-esc, as a terminal that reports it sends command-esc, quits with 0. A stage's
// command has two minutes; past them it is a refusal, and quitting kills the commands still running.
package flow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
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
// whether the fetch is skipped, the filters backlog, underway, and changes take, whether every stage's headers are
// hidden, as a test of the rows alone asks.
type Options struct {
	Stages    []string
	Origin    Origin
	NoFetch   bool
	Filters   []string
	NoHeaders bool
	Presets   string // the file the person's presets are kept in; none keeps them for the session alone
	NoPresets bool   // whether the stages' tables have no preset line, as a test of the rows alone asks
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
	stale   bool   // drawn from what kit kept of its last load, or changed since it loaded, as its mark says
	mark    string // kit's mark of what the stage drew when it loaded, as marks.go reads it
	remark  bool   // whether the stage loaded since the last read of the marks, so it takes the next as its own
	watch   bool   // whether the stage loads again once it is stale, as watch.go says
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
	open    grid.Outline    // which of its headings, rows, and its header stand open, as outline.go says
	picked  map[string]bool // the rows selected, by key, several at once, which the tab's menu acts on
	// the view its preset line was last set to, and the name typed for Save As… or Rename… while the person names a
	// preset, and the person's view Rename… renames
	viewName string
	naming   string
	savingAs bool
	renaming string
	view     grid.Panel // the stage's panel: its search, what it says of each field, and its cursor
	// whether the view the stage was last set to still waits for rows to group by, as it loads, as autoGroup says;
	// and the view as autoGroup left it, which the preset line names as the view still
	grouping bool
	grouped  *preset
	gen      int                // the stage's latest load: an answer from an earlier one is dropped
	ctx      context.Context    // the latest load's, inside the screen's; nil for the first load, which has the screen's
	stop     context.CancelFunc // ends the latest load's command, when r starts another
}

// known names the stages kittui draws.
var known = map[string]bool{"backlog": true, "underway": true, "incoming": true, "desk": true, "worktrees": true,
	"changes": true, "builds": true, "releases": true}

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
	barView       int                    // the cells the stages' bar's view has scrolled past, as the bar last chose
	inRows        bool                   // whether the rows have the keyboard, not the bar
	onPreset      bool                   // whether the preset line has the keyboard, not the bar or the rows
	inPanel       bool                   // whether the panel has the keyboard, before the rows, the line, or the bar
	me            string                 // the caller's login, once kit has said it, which a filter's @me matches
	asked         bool                   // whether kit has been asked for the caller's login
	quotas        []kit.Quota            // each forge's rate limits on the caller, as kit last read them
	quoting       bool                   // whether kit is reading the rate limits now
	quotaOff      bool                   // whether kit could not read them, so the screen asks again only on Refresh All
	quotaSerial   int                    // the last read's, which the tick it waits on names
	marking       bool                   // whether kit is reading the stages' marks now
	marked        bool                   // whether kit has read them once
	marksOff      bool                   // whether kit could not read them, so the screen asks again only on Refresh All
	marksSerial   int                    // the last read's, which the tick it waits on names
	browse        func(url string) error // opens a page in the browser
	notice        string                 // why a page did not open, or what kit did, in place of the keys until a key
	noted         bool                   // whether the notice says what was done, rather than why it was not
	fading        string                 // the notice done put up, which goes after Fades while it is still up
	fades         int                    // the notices done has put up, each the serial its fadedMsg carries
	ask           *ask                   // what a backlog action asks before kit runs, while it asks
	saved         map[string][]preset    // the person's views, by stage, as their file keeps them
	presetsPath   string                 // that file, or none
	presetsMenu   shell.Dropdown         // the preset line's menu of views: whether it is open, and its line under the keyboard
	colMenu       columnMenu             // the menu a click on a column's name drops, while it is open
	fold          bool                   // whether the tables fold a value every row of a group shares, as kittui's settings say
	glyphs        bool                   // whether the tables draw an emoji every row shown shares, as kittui's settings say
	cols          columnsDialog          // the columns dialog, while it is open
	legend        legend                 // the legend, which F1 and ? show and hide, and the corner it stands in
	noHeaders     bool                   // whether the stages' headers are hidden, as a test of the rows alone asks
	noPresets     bool                   // whether the stages' tables have no preset line
	recent        shell.Recent           // the searches remembered, every screen's, and their dropdown
	tabMenu       shell.Dropdown         // the selected tab's menu: whether it is open, and then has the keyboard, and its line under it
	maximized     bool                   // whether the tab's rows have the bar's lines but one, for the session
	appMenu       shell.Menu             // the app's menu, internal/shell's, which has the keyboard while it is open
	pop           popover                // a field's popover, which has the keyboard while it is open
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
	o := Options{Stages: stages, Origin: ReadOrigin(r.Dir), Presets: presetsFile(), History: shell.RecentFile()}
	for _, word := range words {
		if word == "--no-fetch" {
			o.NoFetch = true
		} else {
			o.Filters = append(o.Filters, word)
		}
	}
	return New(r, o), nil
}

// New opens the screen on the options; every stage loads once Init starts it, its panel set as its Default preset
// says. A stage kittui does not know shows a line saying so, and runs nothing.
func New(r kit.Runner, o Options) Model {
	m := Model{runner: r, limit: deadline, origin: o.Origin, noFetch: o.NoFetch, filters: o.Filters, width: 80, height: 24,
		browse: browser, appMenu: shell.Menu{Here: shell.Flow}, fold: true,
		legend: legend{at: bottomRight}}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.noHeaders, m.presetsPath, m.noPresets = o.NoHeaders, o.Presets, o.NoPresets
	m.recent = shell.Recent{Path: o.History}.Loaded()
	var err error
	if m.saved, m.notice, err = readPresets(o.Presets); err != nil {
		m.notice = fmt.Sprintf("%s %v; the file is left as it is", Failed, err)
	}
	m.noted = err == nil
	m.rebuild()
	for _, name := range o.Stages {
		s := stage{name: name, loading: true, open: grid.Outline{}, picked: map[string]bool{},
			viewName: Opening, grouping: true}
		if !known[name] {
			s.loading, s.refusal = false, fmt.Sprintf("kittui does not draw the stage '%s'", name)
		}
		if c := catalog[name]; c != nil {
			s.view = c.Presets[slices.IndexFunc(c.Presets, func(p preset) bool { return p.Name == Opening })].panel(c.Fields, s.view)
		}
		m.stages = append(m.stages, s)
	}
	m.syncOwn()
	return m
}

// Init starts the loading: every stage at once, so none waits on incoming's fetch, which can take as long as the
// remotes do; the stages the fetch moves load again once it has fetched. Each stage is drawn from what kit kept of its
// last load meanwhile, stale, as kept.go says.
func (m Model) Init() tea.Cmd { return tea.Batch(append(m.keptAll(), m.loads()...)...) }

// Interrupted says whether ctrl-c ended the screen.
func (m Model) Interrupted() bool { return m.interrupted }

// Close ends the stages' commands still running, as quitting does, for a picture kittui leaves hidden as it ends.
func (m Model) Close() { m.cancel() }

// Update takes a message, as update does; then a name typed for Save As… or Rename… on a preset line that no longer
// has the keyboard is dropped, so the line does not come back to it; a stage that loaded has the rate limits read
// again; and a notice that says what was done, new, goes after Fades, unless a key or another notice took its place
// first.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if f, ok := msg.(fadedMsg); ok {
		if f.serial == m.fades && m.notice == m.fading {
			m.notice, m.fading = "", ""
		}
		return m, nil
	}
	next, cmd := m.update(msg)
	n := next.(Model)
	if _, ok := msg.(loadedMsg); ok {
		// a load spends the rate limits, and the stage takes the next mark as its own: both are read again
		cmd = tea.Batch(cmd, n.askQuota(), n.askMarks())
	}
	if !n.onPreset {
		for i := range n.stages {
			n.stages[i].savingAs, n.stages[i].naming, n.stages[i].renaming = false, "", ""
		}
	}
	if n.fades != m.fades {
		serial := n.fades
		cmd = tea.Batch(cmd, tea.Tick(fades, func(time.Time) tea.Msg { return fadedMsg{serial} }))
	}
	return n, cmd
}

// Fades is how long a notice that says what was done stays in place of the keys, short of a key. A notice that says
// why something was not done stays until a key.
const Fades = 5 * time.Second

// fades is how long the screen waits for Fades: Fades, as a test waits less.
var fades = Fades

// fadedMsg says a notice that says what was done has been up for Fades: the one done gave that serial.
type fadedMsg struct{ serial int }

// done puts a notice that says what was done in place of the keys, for Fades, short of a key.
func (m *Model) done(notice string) {
	m.notice, m.noted, m.fading = notice, true, notice
	m.fades++
}

// update takes a message.
func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
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
	case keptMsg:
		m.takeKept(msg)
	case marksMsg:
		return m, m.tookMarks(msg)
	case marksTickMsg:
		if msg.serial == m.marksSerial {
			return m, m.askMarks()
		}
	case quotaMsg:
		return m, m.tookQuota(msg)
	case quotaTickMsg:
		if msg.serial == m.quotaSerial {
			return m, m.askQuota()
		}
	case meMsg:
		if msg.err == nil {
			m.me = msg.login
			m.rebuild()
		}
	case browsedMsg:
		switch {
		case errors.Is(msg.err, shell.ErrNoDisplay):
			m.notice, m.noted = msg.link(), false
		case msg.err != nil:
			m.notice, m.noted = fmt.Sprintf("%s could not open %s: %v", Failed, msg.url, msg.err), false
		}
	case shell.SettingsMsg:
		m.fold, m.glyphs = msg.Settings.FoldShared, msg.Settings.SharedEmoji
		m.rebuild()
		m.fit()
	case trackersMsg:
		return m, m.newIssue(msg)
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
		if msg.X < m.paneWidth() && msg.Y >= m.top() && msg.Y < m.top()+m.paneHeight() {
			m.scrollRows(d * wheel)
		}
	case tea.MouseClickMsg:
		if m.ask != nil {
			return m, nil
		}
		if m.pop.open {
			m.popClick(msg.X, msg.Y)
			return m, nil
		}
		if m.cols.open {
			m.columnsClick(msg.X, msg.Y)
			return m, nil
		}
		if m.presetsMenu.Open {
			m.presetMenuClick(msg.X, msg.Y)
			return m, nil
		}
		if m.colMenu.Open {
			return m, m.columnClick(msg.X, msg.Y)
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
			m.panel(m.stage).Recall {
			m.recent = m.recent.Opened()
			return m, nil
		}
		if m.legend.open && !m.tabMenu.Open && !m.appMenu.Open && m.ask == nil && m.legendClick(msg.X, msg.Y) {
			return m, nil
		}
		switch {
		case m.appMenu.Open && msg.Button == tea.MouseLeft:
			var c shell.Choice
			m.appMenu, c = m.appMenu.Click(msg.X, msg.Y, m.paneWidth())
			return m, m.chose(c)
		case m.tabMenu.Open && msg.Button == tea.MouseLeft:
			return m, m.menuClick(msg.X, msg.Y)
		case msg.Button == tea.MouseLeft && m.onMenu(msg.X, msg.Y):
			return m, m.dropDown(false)
		case msg.Button == tea.MouseLeft && shell.OnApp(msg.X, msg.Y):
			m.appMenu = m.appMenu.Opened()
		case len(m.stages) > 0 && grouping(msg) && msg.Y >= m.top() && msg.Y < m.top()+m.paneHeight() &&
			msg.X < m.paneWidth():
			m.groupClick(msg.X, msg.Y, extends(msg), msg.Button == tea.MouseRight)
		case msg.Button != tea.MouseLeft:
			if bar, ok := m.barTakes(msg); ok {
				m.choose(bar)
			}
		case msg.Y == 0 && msg.X >= m.searchAt():
			return m, m.enterPanel()
		case len(m.stages) > 0 && msg.Y == m.presetTop() && msg.X < m.paneWidth() && m.sortAt(msg.X) != "":
			m.sortBy(m.sortAt(msg.X), msg.Mod&tea.ModShift != 0)
		case len(m.stages) > 0 && m.presets(m.stage) && msg.Y == m.presetTop() && msg.X < m.paneWidth():
			return m, m.presetClick(msg.X)
		case len(m.stages) > 0 && msg.Y >= m.top() && msg.Y < m.rowsTop() && msg.X < m.paneWidth():
			return m, m.clickHeader(msg.X, msg.Y, extends(msg), false)
		default:
			if bar, ok := m.barTakes(msg); ok {
				m.choose(bar)
			} else {
				return m, m.click(msg.X, msg.Y, msg.Mod&tea.ModAlt != 0)
			}
		}
	}
	return m, nil
}

// onMenu says whether a click at x, y is on the selected tab's menu glyph: its cells on the bar's middle line, or, once
// maximized, on the title, or the blank cell either side of them, so a click a little off still drops the menu.
func (m Model) onMenu(x, y int) bool {
	at := m.kebab()
	return at >= 0 && y == m.menuTop() && x >= at-1 && x <= at+ansi.StringWidth(tabs.Menu)
}

// barTakes gives what the bar makes of a message, as its Update says, and whether it took it; while maximized, the bar
// is not drawn, so a click is never on it, and left and right have no tabs to move along: it takes a tab's number
// alone.
func (m Model) barTakes(msg tea.Msg) (tabs.Bar, bool) {
	if k, ok := msg.(tea.KeyPressMsg); m.maximized && (!ok || k.String() == "left" || k.String() == "right") {
		return tabs.Bar{}, false
	}
	return m.bar().Update(msg)
}

// dropDown drops a menu down: the app's, or the selected tab's, which on backlog asks kit for the caller's login the
// first time, as Add to Pull Request needs it.
func (m *Model) dropDown(app bool) tea.Cmd {
	if app {
		m.appMenu = m.appMenu.Opened()
		return nil
	}
	m.openMenu()
	if !m.tabMenu.Open || m.stages[m.stage].name != "backlog" {
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

// browsedMsg says whether the browser was asked to open the page at url: err is why not, and shell.ErrNoDisplay that
// the person is not at the machine, so the link is given to open by hand. first and more say which pages were asked
// together: the first of them, and how many came after it.
type browsedMsg struct {
	url   string
	err   error
	first string
	more  int
}

// browser is how a screen opens a page, which New gives it: the browser, as shell.Browse opens it.
var browser = shell.Browse

// openPage asks the browser for the page at url.
func (m Model) openPage(url string) tea.Cmd { return m.openPages([]string{url}) }

// openPages asks the browser for each page. Where the person must open them by hand, each says the first page's link
// and how many more were asked, so what stands is the same whichever answers last.
func (m Model) openPages(urls []string) tea.Cmd {
	b := m.browse
	var cmds []tea.Cmd
	for _, url := range urls {
		cmds = append(cmds, func() tea.Msg { return browsedMsg{url, b(url), urls[0], len(urls) - 1} })
	}
	return tea.Batch(cmds...)
}

// link is the notice that gives the link of the first page, to open by hand, and says how many more were asked.
func (msg browsedMsg) link() string {
	text := "Open " + msg.first + " in a browser"
	switch msg.more {
	case 0:
	case 1:
		text += ", and 1 more page"
	default:
		text += fmt.Sprintf(", and %d more pages", msg.more)
	}
	return text
}

// meMsg is kit's answer to the caller's login: the login, or why it could not be read.
type meMsg struct {
	login string
	err   error
}

// panel gives a stage's panel: what the person set in it, the columns it offers, whether its rows belong to people,
// the caller's login, and whether it has the keyboard.
func (m Model) panel(i int) grid.Panel {
	p := m.filter(i)
	p.Columns = m.offered(i)
	p.Recall = len(m.recent.Items) > 0
	return p
}

// filter gives a stage's panel as its rows are let through: what the person set in it, whether its rows belong to
// people, and the caller's login; the columns it offers are drawn from the rows, so it has none.
func (m Model) filter(i int) grid.Panel {
	s := m.stages[i]
	p := s.view
	p.Owner, p.Me, p.Sortable, p.Groupable = ownerOf(s.name), m.me, sortableOf(s.name), groupableOf(s.name)
	p.Labels = labelsOf(s.name)
	p.Focused, p.Outline, p.Fold, p.Glyphs = m.inPanel && i == m.stage, s.open, m.fold, m.glyphs
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

// enterPanel gives the search the keyboard, what is typed in it as it is.
func (m *Model) enterPanel() tea.Cmd {
	if len(m.stages) == 0 {
		return nil
	}
	m.inPanel = true
	m.stages[m.stage].view = m.panel(m.stage).Find()
	return nil
}

// grouping says whether a click groups rather than sorts or selects: with ctrl held, where the terminal sends it, or
// with the right button, which on a column's name drops its menu down, and on a group's heading ungroups.
func grouping(msg tea.MouseClickMsg) bool {
	return msg.Mod&tea.ModCtrl != 0 || msg.Button == tea.MouseRight
}

// extends says whether a click on a column's name adds to the sort or the groups the rows have, rather than replacing
// them: with option held. Shift would say it in a desktop's table, but a terminal often keeps shift and a click for its
// own selection, and never sends it.
func extends(msg tea.MouseClickMsg) bool { return msg.Mod&tea.ModAlt != 0 }

// groupClick takes a click that groups, as grouping says: a right click on a column's name in the header drops the
// column menu down under it; a ctrl-click there groups the rows by it, as clickHeader says; on a group's heading, they no
// longer group by its column, the other groups kept.
func (m *Model) groupClick(x, y int, nest, right bool) {
	if column := m.headerColumn(x, y); right && column != "" {
		m.openColumnMenu(column)
		return
	}
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

// clickHeader takes a click on the header, which is no row, above its rule, the preset line, where each column's sort
// target stands, as sortAt says: on its gutter it opens the header, or closes it, when it hides lines; on a column's
// name it drops the column menu down under it. Where the terminal sends option or ctrl with a click, as many do not,
// also, as option asks, turns the column's sort, as the panel's Sorted says, and group, as ctrl asks, groups the rows
// by it within the groups they have, innermost, or no longer, as the panel's Grouped says. Elsewhere it does nothing.
func (m *Model) clickHeader(x, y int, also, group bool) tea.Cmd {
	s := &m.stages[m.stage]
	x -= m.indent(m.stage)
	if y < m.top() {
		return nil
	}
	if _, hides := m.header(); hides && !group && y == m.top() && x >= 0 && x < len(lead) {
		next := maps.Clone(s.open)
		next[headerKey] = !s.open[headerKey]
		m.reoutline(next)
		return nil
	}
	lines, _ := m.header()
	if y >= m.top()+len(lines)-1 {
		return nil
	}
	j := grid.HeaderAt(m.rows(m.stage), m.names(m.stage), m.gridWidth(m.stage), x)
	if j < 0 {
		return nil
	}
	column, p := m.shownColumns(m.stage)[j], m.panel(m.stage)
	switch {
	case group:
		m.setPanel(p.Grouped(column, true))
	case also:
		m.setPanel(p.Sorted(column))
	default:
		m.openColumnMenu(column)
	}
	return nil
}

// selectRow selects the row under the cursor, as one of several, or unselects it, while the rows have the keyboard;
// with keep, it selects it whether it was or not, as shift with up or down selects as it moves. On a group's heading it
// selects every row of the group, those folded away too, or, once every one is, none; another aside selects nothing.
func (m *Model) selectRow(keep bool) {
	if !m.inRows || len(m.stages) == 0 {
		return
	}
	s, rows := &m.stages[m.stage], m.rows(m.stage)
	if s.cursor >= len(rows) {
		return
	}
	keys := []string{rows[s.cursor].Key}
	switch {
	case grid.Heads(rows[s.cursor]):
		every := m.every(m.stage)
		keys = groupRows(every, slices.IndexFunc(every, func(r row) bool { return r.Key == rows[s.cursor].Key }))
	case rows[s.cursor].Aside:
		return
	}
	all := !slices.ContainsFunc(keys, func(k string) bool { return !s.picked[k] })
	for _, key := range keys {
		if keep || !all {
			s.picked[key] = true
		} else {
			delete(s.picked, key)
		}
	}
}

// selectAll selects every row the stage shows, those its closed groups fold away too, or, when every one is, none.
func (m *Model) selectAll() {
	if len(m.stages) == 0 {
		return
	}
	s, rows := &m.stages[m.stage], m.every(m.stage)
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

// selectedRows gives the selected stage's rows that are selected, of those it shows, its closed groups' too, each once,
// though grouping by a column of several values stands it under each.
func (m Model) selectedRows() []row {
	if len(m.stages) == 0 {
		return nil
	}
	var out []row
	seen := map[string]bool{}
	for _, r := range m.every(m.stage) {
		if m.stages[m.stage].picked[r.Key] && !r.Aside && !seen[r.Key] {
			seen[r.Key] = true
			out = append(out, r)
		}
	}
	return out
}

// selected gives how many of the selected stage's rows shown are selected.
func (m Model) selected() int { return len(m.selectedRows()) }

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

// key takes a key press: the quitting keys, a field's popover's while it is open, the open menu's while one is open, the
// panel's while it has the keyboard, the preset line's while it has it, = for the app's menu and : or ; for the tab's,
// the bar's, then the rows', F1 and ? showing the legend or hiding it. Down from the bar reaches the preset line, where the stage has one, and the rows from it.
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
	if m.pop.open {
		return m.popKey(k)
	}
	if m.cols.open {
		return m.columnsKey(k)
	}
	if m.presetsMenu.Open {
		return m.presetMenuKey(k)
	}
	if m.colMenu.Open {
		return m.columnKey(k)
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
	if m.tabMenu.Open {
		return m.menuKey(k)
	}
	if m.inPanel {
		return m.panelKey(k)
	}
	if m.onPreset {
		if cmd, ok := m.presetKey(k); ok {
			return m, cmd
		}
	}
	if menu, app := menuFor(k); menu {
		return m, m.dropDown(app)
	}
	if bar, ok := m.barTakes(k); ok {
		m.choose(bar)
		return m, nil
	}
	switch k.String() {
	case "/":
		return m, m.enterPanel()
	case "esc":
		if m.inRows {
			s := &m.stages[m.stage]
			s.cursor, s.offset, m.inRows = 0, 0, false
		}
		m.fit()
	case "?", "f1":
		m.toggleLegend()
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
		if !m.inRows && m.presets(m.stage) {
			// from the bar, down reaches the preset line first
			m.onPreset = true
			return m, nil
		}
		m.down(1)
	case "pgup":
		m.page(-1)
	case "pgdown":
		m.page(1)
	case "enter":
		// on the heading over the groups, ⏎ drops its first column's menu, as a click on the name does
		if !m.inRows || len(m.stages) == 0 {
			break
		}
		if k, rows := m.stages[m.stage].cursor, m.rows(m.stage); k < len(rows) && grid.OverGroups(rows[k]) {
			if names := m.headingNames(rows[k]); len(names) > 0 {
				m.openFieldMenu(k, names[0].column)
			}
		}
	case "right", "alt+right":
		m.expand(true, k.String() == "alt+right")
	case "left", "alt+left":
		m.expand(false, k.String() == "alt+left")
	}
	return m, nil
}

// panelKey takes a key while the search has the keyboard, as panelKeys says; ctrl+r drops the searches remembered
// down, and what was typed is remembered once the search gives up the keyboard.
func (m Model) panelKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if len(m.stages) == 0 {
		return m.panelKeys(k)
	}
	before := m.panel(m.stage)
	searching := before.Focused
	if searching && k.String() == "ctrl+r" {
		m.recent = m.recent.Opened()
		return m, nil
	}
	next, cmd := m.panelKeys(k)
	if after := next.(Model); searching && !after.inPanel {
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
	return shell.RecallAt(m.searchAt())
}

// panelKeys takes a key while the search has the keyboard, as panelKey says: tab, esc, and enter leave it, what is
// typed kept; a printable key is typed, and backspace takes one back; any other key the bar takes, as a stage's number
// does not while it is typed, gives the bar the keyboard.
func (m Model) panelKeys(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "tab", "esc", "enter":
		m.inPanel = false
		return m, nil
	}
	if p, ok := m.panel(m.stage).Update(k); ok {
		m.setPanel(p)
		return m, nil
	}
	m.inPanel = false
	if bar, ok := m.barTakes(k); ok {
		m.choose(bar)
		return m, nil
	}
	m.inPanel = true
	return m, nil
}

// choose takes what the bar chose: where its view is scrolled, the stage it selected, where the cursor is as it was
// left, and the keyboard. A triangle, which scrolls the view alone, leaves the keyboard where it is, in the panel or on
// the preset line too.
func (m *Model) choose(bar tabs.Bar) {
	m.barView = bar.Scroll
	if bar.Selected == m.stage && bar.Focused == m.bar().Focused {
		return
	}
	m.stage, m.inRows, m.inPanel, m.onPreset = bar.Selected, !bar.Focused, false, false
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
func (m Model) all() tea.Cmd { return tea.Batch(m.loads()...) }

// loads gives the load of every stage kittui draws.
func (m Model) loads() []tea.Cmd {
	var cmds []tea.Cmd
	for _, s := range m.stages {
		if known[s.name] {
			cmds = append(cmds, m.load(s.name, false))
		}
	}
	return cmds
}

// restart runs a stage's command again as its latest load, reading the forge afresh, ending the one before when it
// still runs. A stage that shows its rows keeps them, stale, its dot on its tab, until the new ones come; one that
// shows a refusal, or is loading, shows it loading until it answers.
func (m *Model) restart(name string) tea.Cmd {
	cmd, s := m.again(name, true), m.find(name)
	if !s.loading && s.refusal == "" {
		s.stale = true
		return cmd
	}
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
	s.loading, s.stale, s.remark = false, false, true
	err := msg.err
	if err == nil {
		err = s.decode(msg.data)
	}
	if err != nil {
		s.refusal = refusalOf(s.name, err)
	}
	m.rebuild()
	m.autoGroup(s.name)
	for i := range m.stages {
		m.clamp(i)
	}
	m.fit()
	return true
}

// autoGroup groups a stage's rows as its view groups them as it loads, once it has rows to look at: by each column
// whose value is the same on every row the panel lets through, as the panel's AutoGrouped says, where kittui's
// settings auto-group and the view groups by nothing. It runs once a view, on the first rows after the view is set, as
// the screen opens or the preset line sets another; past that, the groups are the person's, as any they chose, and
// the preset line names the view still.
func (m *Model) autoGroup(name string) {
	i := slices.IndexFunc(m.stages, func(s stage) bool { return s.name == name })
	if i < 0 {
		return
	}
	s := &m.stages[i]
	if !s.grouping || s.loading || s.refusal != "" {
		return
	}
	s.grouping = false
	var all []row
	if s.name == "incoming" {
		all = incomingRows(*s)
	} else {
		all = laid(s.name, m.records(*s), func(id string) string { return s.view.Field(id).Rendering })
	}
	p := m.panel(i)
	next := p.AutoGrouped(all, columnsOf(s.name))
	if len(next.Groups()) == len(p.Groups()) {
		return
	}
	s.view = next
	grouped := m.current(i)
	s.grouped = &grouped
	m.rebuild()
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
		// past the first row, the preset line has the keyboard, or the bar when the stage has none
		s.cursor, s.offset, m.inRows = 0, 0, false
		m.onPreset = m.presets(m.stage)
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

// click takes a left click on the pane's rows: the cursor goes to the row clicked, and the rows have the keyboard; a
// click on a linked glyph of a row's first line, as an issue's before its title, opens its page in the browser; one
// on the first line of the row already under the cursor, or on the gutter of a row's first line, its ▸ or ▾, opens
// the row, or closes it, as one on that first line pinned on top, once the row has scrolled it away, does; one on a
// column's name in a group's heading drops the heading menu around the name, and one anywhere else on the heading
// folds its rows away, or unfolds them.
// With subtree, as option asks, everything under the row or the heading opens or closes with it. A click on a … line, or on a line about the stage, does nothing.
func (m *Model) click(x, y int, subtree bool) tea.Cmd {
	if len(m.stages) == 0 || x >= m.paneWidth() {
		return nil
	}
	lines, s := m.lines(m.stage), &m.stages[m.stage]
	if y < m.rowsTop() {
		return m.clickHeader(x, y, false, false)
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
	toggle := first && link == "" && (x >= 0 && x < len(lead) || m.inRows && row == s.cursor)
	s.cursor, m.inRows, m.inPanel, m.onPreset = row, true, false, false
	name, onName := m.nameAt(rows[row], x)
	level, onLevel := m.levelAt(rows[row], x)
	switch {
	case onLevel && !subtree:
		m.foldLevel(level)
		m.fit()
	case onName && !subtree:
		m.openFieldMenu(row, name.column)
	case grid.OverGroups(rows[row]) && (x < 0 || x >= len(lead)):
		// the heading over the groups folds by its ▾ or ▸ alone; its names are its columns', the rest nothing
		m.fit()
	case grid.Heads(rows[row]):
		m.disclose(!s.open.Open(rows[row]), subtree)
		m.fit()
	case strings.HasPrefix(link, jumps):
		m.fit()
		m.jump(strings.TrimPrefix(link, jumps))
	case link != "":
		m.fit()
		return m.openPage(link)
	case toggle:
		m.disclose(!s.open.Open(rows[row]), subtree)
		m.fit()
	default:
		m.fit()
	}
	return nil
}

// rowLine gives the screen line the first line of the selected stage's row k stands on, as the pane is scrolled, or
// -1 while it is scrolled away.
func (m Model) rowLine(k int) int {
	lines, s := m.lines(m.stage), m.stages[m.stage]
	for y := 0; y < m.rowsHeight(); y++ {
		if at := pane.At(len(lines), m.rowsHeight(), s.offset, y); at >= 0 && lines[at].row == k {
			return m.rowsTop() + y
		}
	}
	return -1
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
	m.choose(tabs.Bar{Selected: i, Scroll: m.barView})
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
