// Package flow is kit flow on a terminal: the stages as a tab bar, internal/tabs, the selected stage's rows under it,
// and a row opened in place. It draws from kit flow <stage> --json and never parses kit's text. While the bar has the
// keyboard, tab and shift-tab move along it, as they move between the stops on any line; a stage's number, or a click on the bar, selects it from anywhere and gives
// the bar the keyboard. Down enters a stage's rows and up on the first row leaves them; in the rows, right opens a row
// and left closes it, space selects it as one of several, shift with up or down selects as it moves, and ctrl+a selects
// every row, and a click on a row puts the cursor there, and on the row under it opens or closes it; = drops the app's
// menu down, as a click on = or the screen's name does, - the workspace menu, and : or ; the selected tab's, as a click
// anywhere on the tab does; F1 or ? shows the legend over the rows, and again hides it; / or a click on the search,
// always on the screen's top right, gives the search the keyboard. A click on a linked glyph, as an issue's before its
// title, opens its page in the browser, and one on a glyph that names a row of another stage, as an issue's pull
// request, selects that stage with the row under the cursor. The rows take the screen's width. Maximize, in the tab's
// menu, gives the rows the bar's lines but one, the tab named in the title, and Restore puts the bar back. Esc goes
// back one level: from the search, then from the rows to the bar. Ctrl-c quits with 130 from anywhere, and ctrl-q quits
// with 0; esc never quits. A stage's command has two minutes; past them it is a refusal, and quitting kills the
// commands still running.
package flow

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/fix"
	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/pane"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/source"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// Options is how the screen opens: the app it is, as apps.go defines it; the stages in the picture's order; the stage
// it opens on, maximized, when one is named; origin, when known, for the URLs and for the workspace menu until kit says
// the workspace; whether the fetch is skipped; the filters issues and changes take; whether each stage's
// first read is fresh; and whether every stage's headers are hidden, as a test of the rows alone asks.
type Options struct {
	App       shell.Screen
	Stages    []string
	Focus     string
	Origin    Origin
	NoFetch   bool
	Filters   []string
	Preset    string // the view the stage it opens on is set as, when one is named; else the stage's opening one
	Fresh     bool
	NoHeaders bool
	Presets   string       // the file the person's presets are kept in; none keeps them for the session alone
	Team      string       // the folder the team's presets are kept in, in a clone kit trusts; none where there is none
	NoPresets bool         // whether the stages' tables have no preset line, as a test of the rows alone asks
	History   string       // the file the searches are remembered in; none remembers them for the session alone
	Clock     source.Clock // what the screen's waits go through: the program's, where none is given
}

// failure is what kit could not read of a stage, as its document's errors name it: over a group, the member, its host
// and path; and kit's line, or, for a feature the forge has off here, as secret scanning disabled, the feature and off,
// which is no failure, and lacked too where the forge's provider lacks the feature.
type failure struct {
	Member  string `json:"member"`
	Error   string `json:"error"`
	Feature string `json:"feature"`
	Off     bool   `json:"off"`
	Lacked  bool   `json:"lacked"`
}

// stage is one stage of the picture: its name, as kittui knows it, kit flow's own and kit workspace's after workspace/,
// what kit printed for it, and where the person is in its rows.
type stage struct {
	name    string
	src     source.Load[[]byte] // its read, its catalog entry's command: loading, stale, its mark, and Watch
	refusal string              // kit's line, when the command failed
	rows    []json.RawMessage   // its records, as its catalog entry reads them, each its own bytes
	parsed  []parsed            // its records as decode read them, once each, one a record
	typed   any                 // its records as Go holds them, for a stage whose rows Go builds, as builders says
	failed  []failure           // what kit could not read of it: over a group, the members, and the features off here
	limited int                 // how many rows kit read of more the forge holds, as a capped read says; none, all
	// its table's state: the panel, the view, the outline, the rows selected, and where the cursor is
	table.Table
}

// parsed is a record as decode read it, once: its key, as the catalog entry's key reads it; its member, over a group;
// where each field's value leads, by the field's id; and the record decoded, as table.Decoded gives it, which each
// field's value is read from, so no draw decodes a record again.
type parsed struct {
	key, member string
	links       map[string]any
	value       any
}

// decodeRecord decodes a record once, as decode reads it: table.Decoded, which a test counts.
var decodeRecord = table.Decoded

// commandOf gives the kit command that prints a stage's rows, kit's words after kit, as its catalog entry names it;
// none for a stage the catalog does not name.
func commandOf(name string) []string {
	if c := catalog[name]; c != nil {
		return c.Command
	}
	return nil
}

// sourced says whether kittui draws a stage: its catalog entry names the command that prints its rows.
func sourced(name string) bool { return len(commandOf(name)) > 0 }

// staged says whether a stage is one its verb's --stages and --marks name, as its catalog entry says.
func staged(name string) bool { return catalog[name] != nil && catalog[name].Staged }

// takes says whether a stage's command takes a word beside its own, as its catalog entry's takes names them: owner,
// no-fetch, fresh, cached, or workspace.
func takes(name, word string) bool {
	return catalog[name] != nil && slices.Contains(catalog[name].Takes, word)
}

// everMarked says whether a stage's verb ever marks it: every stage but one whose catalog entry says marked false, as
// kit people's, whose --marks always prints null.
func everMarked(name string) bool {
	return catalog[name] == nil || catalog[name].Marked == nil || *catalog[name].Marked
}

// verbOf gives the verb that prints a staged stage, and the stage's name to that verb, as its catalog entry's command
// names them: workspace and branches for workspace/branches. A stage that is not staged has neither.
func verbOf(name string) (verb, stage string) {
	if !staged(name) {
		return "", ""
	}
	c := catalog[name].Command
	return c[0], c[1]
}

// workspaceVerb is kit workspace's verb, whose stages kittui knows after it and a slash.
const workspaceVerb = "workspace"

// label gives the name a stage goes by, as kit names it, on its tab and in what the screen says of it: kit flow's as
// it is, and another verb's without the verb and a slash, as workspace/branches is branches.
func label(name string) string {
	if _, after, ok := strings.Cut(name, "/"); ok {
		return after
	}
	return name
}

// command gives the command line that reads a stage, as the screen names it when the stage fails: kit and its catalog
// entry's command, or kit and the stage's name for one kittui does not draw.
func command(name string) string {
	if c := commandOf(name); len(c) > 0 {
		return "kit " + strings.Join(c, " ")
	}
	return "kit " + name
}

// deadline is how long a stage's command may take: past it, the stage is a refusal, and Refresh All runs it again.
const deadline = 2 * time.Minute

// Model is the picture on the screen.
type Model struct {
	app           shell.Screen // the app this is, as apps.go defines it
	runner        kit.Runner
	ctx           context.Context    // ends with the screen, and every stage's command with it
	cancel        context.CancelFunc // ends ctx
	limit         time.Duration      // how long a stage's command may take: the deadline, shorter in a test
	origin        Origin
	noFetch       bool
	fresh         bool  // whether each stage's first read is fresh, as kit flow --fresh asks
	words         Words // kit flow's words the screen was opened with, as OpenApp was given them
	altArrows     bool  // the terminal answered the kitty keyboard protocol, so ⌥ with an arrow arrives as itself
	filters       []string
	stages        []stage
	stage         int                       // the selected stage
	barView       int                       // the cells the stages' bar's view has scrolled past, as the bar last chose
	inRows        bool                      // whether the rows have the keyboard, not the bar
	onPreset      bool                      // whether the preset line has the keyboard, not the bar or the rows
	stop          int                       // while onPreset, the heading over the groups' stop the keyboard is on, -1 the preset picker
	inPanel       bool                      // whether the panel has the keyboard, before the rows, the line, or the bar
	me            string                    // the caller's login, once kit has said it, which a filter's @me matches
	asked         bool                      // whether kit has been asked for the caller's login
	browse        func(url string) error    // opens a page in the browser
	note          shell.Notice              // what kit did, or why something was not done, in place of the keys
	copied        bool                      // whether the refusal's command was just copied, until the next key or click
	handOver      fix.Exec                  // hands the terminal to a fix: tea.ExecProcess but in tests
	ask           *ask                      // what an issues action asks before kit runs, while it asks
	saved         map[string][]table.Preset // the person's views, by stage, as their file keeps them
	starred       map[string][]string       // the names of the views the person starred, by stage, in their order
	teamDir       string                    // the folder the team's presets are kept in, none where there is none
	team          map[string][]table.Preset // the team's views, by stage, as their folder keeps them
	presetsPath   string                    // that file, or none
	presetsMenu   table.PresetMenu          // the preset line's menu of views: whether it is open, and its line under the keyboard
	manage        table.Manage              // the presets dialog, which Manage… in the presets' menu opens
	fold          bool                      // whether the tables fold a value every row of a group shares, as kittui's settings say
	hide          bool                      // whether the tables keep the columns no row holds a value in from showing, as kittui's settings say
	glyphs        bool                      // whether the tables draw an emoji every row shown shares, as kittui's settings say
	cols          table.Columns             // the columns dialog, while it is open
	legend        legend                    // the legend, which F1 and ? show and hide, and the corner it stands in
	noHeaders     bool                      // whether the stages' headers are hidden, as a test of the rows alone asks
	noPresets     bool                      // whether the stages' tables have no preset line
	recent        shell.Recent              // the searches remembered, every screen's, and their dropdown
	tabMenu       shell.Dropdown            // the selected tab's menu: whether it is open, and then has the keyboard, and its line under it
	maximized     bool                      // whether the tab's rows have the bar's lines but one, for the session
	appMenu       shell.Menu                // the app's menu, internal/shell's, which has the keyboard while it is open
	books         shell.BooksMenu           // the title's workspace menu, which has the keyboard while it is open
	ws            kit.Workspace             // the workspace kit said last: a group's members, and their checkouts
	pop           table.Popover             // a field's popover, which has the keyboard while it is open
	width, height int
	interrupted   bool
	built         *built // each stage's rows as last built, shared by the model's copies until it is replaced
	landing       string // a jump to a row whose stage has not loaded it yet: the stage, a slash, and the row's key

	clock source.Clock                   // what the screen's waits go through
	quota source.Poll[[]kit.Quota]       // each forge's rate limits on the caller, as quota.go reads them
	marks source.Poll[map[string]string] // each stage's mark, as marks.go reads them
}

// built holds each stage's rows as they were last built, and each stage's lines as they were last drawn from them. A
// stage that loads, a refresh, or a new width replaces it, since a stage's rows may draw from another's, as issues'
// pull requests' keys from changes'; New sets the first, so there always is one, for the width the screen has.
type built struct {
	stages map[int]shown
	lines  map[int]drawnLines
}

// drawnLines is a stage's lines as allLines last drew them, and what they were drawn for, as drawnFor says it.
type drawnLines struct {
	key   string
	lines []table.Line
}

// rebuild drops the rows built so far, and the lines drawn from them; each is built again when it is next drawn.
func (m *Model) rebuild() {
	m.built = &built{stages: map[int]shown{}, lines: map[int]drawnLines{}}
}

// open opens the screen on the stages the options name, origin read from git, and kit flow's words: the tab named,
// which it opens on, maximized, as kit flow <stage> asks, set as the preset named, as kit flow backlog asks;
// --no-fetch; the owner's flags, the filters issues and changes take; and --fresh, which reads each stage afresh the
// first time.
func open(r kit.Runner, o Options, w Words) Model {
	o.Origin, o.Presets, o.History, o.Team = ReadOrigin(r.Dir), table.PresetsPath(), shell.RecentFile(), teamFolder(r)
	if k := slices.IndexFunc(o.Stages, func(name string) bool { return label(name) == w.Tab }); w.Tab != "" && k >= 0 {
		o.Focus, o.Preset = o.Stages[k], w.Preset
	}
	o.NoFetch, o.Filters, o.Fresh = w.NoFetch, slices.Clone(w.Owner), w.Fresh
	m := New(r, o)
	m.words = w
	return m
}

// New opens the screen on the options; every stage loads once Init starts it, its panel set as the first of its own
// presets says, or, for the stage it opens on, as the preset named does. A stage kittui does not know shows a line
// saying so, and runs nothing. The stage named to focus on is selected and maximized, the bar's keys on its title as
// they are on any maximized tab's.
func New(r kit.Runner, o Options) Model {
	m := Model{app: o.App, runner: r, limit: deadline, origin: o.Origin, noFetch: o.NoFetch, fresh: o.Fresh,
		filters: o.Filters, width: 80, height: 24, browse: browser, handOver: tea.ExecProcess, appMenu: shell.Menu{Here: o.App},
		books: shell.BooksMenu{Fallback: o.Origin.Repo}, fold: true, hide: true, legend: legend{at: bottomRight}}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.clock = o.Clock
	m.quota = source.NewPoll("quota", QuotaEvery, r.Quota)
	// its marks are those of the verbs its staged stages are of, as verbsOf gives them, each named as kittui knows it
	verbs := verbsOf(o.Stages)
	m.marks = source.NewPoll("marks", MarksEvery, func(ctx context.Context) (map[string]string, error) {
		named := map[string]string{}
		var errs []error
		for _, verb := range verbs {
			marks, err := r.MarksOf(ctx, verb)
			errs = append(errs, err)
			for _, name := range o.Stages {
				if v, stage := verbOf(name); v == verb {
					if mark, ok := marks[stage]; ok {
						named[name] = mark
					}
				}
			}
		}
		return named, errors.Join(errs...)
	})
	m.noHeaders, m.presetsPath, m.noPresets = o.NoHeaders, o.Presets, o.NoPresets
	m.recent = shell.Recent{Path: o.History}.Loaded()
	var err error
	if m.saved, m.note.Text, err = readPresets(o.Presets); err != nil {
		m.note.Text = fmt.Sprintf("%s %v; the file is left as it is", Failed, err)
	}
	m.starred = table.ReadFavorites(o.Presets)
	m.note.Done = err == nil
	m.teamDir = o.Team
	m.readTeam(o.Stages)
	m.rebuild()
	for _, name := range o.Stages {
		s := stage{name: name, src: m.stageLoad(name),
			Table: table.Table{Open: grid.Outline{}, Picked: map[string]bool{}, Grouping: true}}
		if !sourced(name) {
			s.src.Loading, s.refusal = false, fmt.Sprintf("kittui does not draw the stage '%s'", label(name))
		}
		views := m.viewsNamed(name)
		v := views.Opening()
		if name == o.Focus && o.Preset != "" {
			v = views.Chosen(o.Preset)
		}
		s.ViewName = v.Name
		if catalog[name] != nil {
			s.View = v.Panel(fieldsOf(name), s.View)
		}
		m.stages = append(m.stages, s)
	}
	if k := slices.IndexFunc(m.stages, func(s stage) bool { return s.name == o.Focus }); o.Focus != "" && k >= 0 {
		m.stage = k
		m.maximize(true)
	}
	m.syncOwn()
	return m
}

// Init starts the loading: every stage at once, so none waits on incoming's fetch, which can take as long as the
// remotes do. Each stage is drawn from what kit kept of its last load meanwhile, stale, as kept.go says. Repositories'
// tabs go stale by kit workspace --marks, which reads the refs a fetch moves.
func (m Model) Init() tea.Cmd { return tea.Batch(append(m.keptAll(), m.loads()...)...) }

// Interrupted says whether ctrl-c ended the screen.
func (m Model) Interrupted() bool { return m.interrupted }

// Close ends the stages' commands still running, as quitting does, for a picture kittui leaves hidden as it ends.
func (m Model) Close() { m.cancel() }

// env gives what the screen's reads of kit share: the screen's context, the deadline, and the screen's clock.
func (m Model) env() source.Env { return source.Env{Ctx: m.ctx, Limit: m.limit, Clock: m.clock} }

// Update takes a message, as update does; then a name typed for Save As… or Rename… on a preset line that no longer
// has the keyboard is dropped, so the line does not come back to it; a stage that loaded has the marks read again, and
// the rate limits, where it reads the forge; and a notice that says what was done, new, goes after Fades, unless a key
// or another notice took its place first.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if f, ok := msg.(shell.FadedMsg); ok {
		m.note = m.note.Faded(f)
		return m, nil
	}
	next, cmd := m.update(msg)
	n := next.(Model)
	if a, ok := msg.(source.Answer[[]byte]); ok && !a.Kept {
		// the stage takes the next mark as its own, so the marks are read again; a load of a command that reads the
		// forge, as its catalog entry's forge says, spends the rate limits, which are read again too; one that reads
		// none, as kit workspace's, spends none
		cmd = tea.Batch(cmd, n.marks.Ask(n.env()))
		if c := catalog[a.Name]; c != nil && c.Forge {
			cmd = tea.Batch(cmd, n.quota.Ask(n.env()))
		}
	}
	if !n.onPreset {
		for i := range n.stages {
			n.stages[i].SavingAs, n.stages[i].Naming, n.stages[i].Renaming = false, "", ""
		}
	}
	if fade := n.note.Fading(m.note, fades); fade != nil {
		cmd = tea.Batch(cmd, fade)
	}
	return n, cmd
}

// Fades is how long a notice that says what was done stays in place of the keys, short of a key. A notice that says
// why something was not done stays until a key.
const Fades = 5 * time.Second

// fades is how long the screen waits for Fades: Fades, as a test waits less.
var fades = Fades

// done puts a notice that says what was done in place of the keys, for Fades, short of a key.
func (m *Model) done(notice string) { m.note = m.note.Did(notice) }

// update takes a message.
func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// the rate limits' and the marks' answers and whiles are theirs
	if cmd, took := m.quota.Take(m.env(), msg); took != source.NotMine {
		return m, cmd
	}
	if cmd, took := m.marks.Take(m.env(), msg); took != source.NotMine {
		return m, m.tookMarks(cmd, took)
	}
	switch msg := msg.(type) {
	case tea.KeyboardEnhancementsMsg:
		m.altArrows = msg.SupportsKeyDisambiguation()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.rebuild()
		m.fit()
	case source.Answer[[]byte]:
		if msg.Kept {
			m.takeKept(msg)
			return m, nil
		}
		m.take(msg)
	case meMsg:
		if msg.err == nil {
			m.me = msg.login
			m.rebuild()
		}
	case browsedMsg:
		switch {
		case errors.Is(msg.err, shell.ErrNoDisplay):
			m.note.Text, m.note.Done = msg.link(), false
		case msg.err != nil:
			m.note.Text, m.note.Done = fmt.Sprintf("%s could not open %s: %v", Failed, msg.url, msg.err), false
		}
	case shell.JumpMsg:
		return m, m.jump(msg.To)
	case shell.WorkspaceMsg:
		m.books = m.books.Told(msg)
		if msg.Refusal == "" {
			m.ws = msg.Workspace
		}
	case shell.SettingsMsg:
		m.fold, m.hide, m.glyphs = msg.Settings.FoldShared, msg.Settings.HideBlank, msg.Settings.SharedEmoji
		m.rebuild()
		m.fit()
	case trackersMsg:
		return m, m.newIssue(msg)
	case actedMsg:
		return m, m.acted(msg)
	case ranMsg:
		return m, m.ran(msg)
	case tea.KeyPressMsg:
		m.note.Text, m.copied = "", false
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
		// an open layer, the topmost, takes the click, as layers.go orders them; the menus leave all but a left one
		if n, cmd, ok := m.layers().Click(m, msg); ok {
			return n, cmd
		}
		if cmd, ok := m.remedyClick(msg); ok {
			return m, cmd
		}
		m.copied = false
		if msg.Y == 0 && len(m.stages) > 0 && m.inPanel && msg.X == m.recallAt() &&
			m.panel(m.stage).Recall {
			m.recent = m.recent.Opened()
			return m, nil
		}
		if m.legend.open && !m.tabMenu.Open && !m.appMenu.Open && m.ask == nil && m.legendClick(msg.X, msg.Y) {
			return m, nil
		}
		switch {
		case msg.Button == tea.MouseLeft && m.staleAt(msg.X, msg.Y) >= 0:
			return m, m.refresh(m.stages[m.staleAt(msg.X, msg.Y)].name)
		case msg.Button == tea.MouseLeft && m.onMenu(msg.X, msg.Y):
			return m, m.dropDown(false)
		case msg.Button == tea.MouseLeft && shell.OnApp(msg.X, msg.Y, m.appLabel()):
			m.appMenu = m.appMenu.Opened()
		case msg.Button == tea.MouseLeft && m.books.On(msg.X, msg.Y, m.booksAt()):
			m.books = m.books.Opened()
		case len(m.stages) > 0 && grouping(msg) && msg.Y >= m.top() && msg.Y < m.top()+m.paneHeight() &&
			msg.X < m.paneWidth():
			return m, m.groupClick(msg.X, msg.Y, extends(msg), msg.Button == tea.MouseRight)
		case msg.Button != tea.MouseLeft:
			// the bar takes a left click alone, and nothing else here takes another
		case msg.Y == 0 && msg.X >= m.searchAt():
			return m, m.enterPanel()
		case len(m.stages) > 0 && m.controls() && msg.Y == m.top() && msg.X < m.paneWidth():
			return m, m.controlClick(msg.X)
		case len(m.stages) > 0 && m.controls() && msg.Y == m.top()+1 && msg.X < m.paneWidth():
			table.BarRuleClick(&m, msg.X, msg.Mod&tea.ModShift != 0)
		case len(m.stages) > 0 && m.onRule(msg.Y) && msg.X < m.paneWidth():
			switch column, flatten := m.sortAt(msg.X); {
			case flatten:
				p := m.panel(m.stage)
				m.setPanel(p.Flattened(column, !p.Field(column).Flat))
			case column != "":
				m.sortBy(column, msg.Mod&tea.ModShift != 0)
			}
		case len(m.stages) > 0 && msg.X < m.paneWidth() && m.inHeader(msg.Y):
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

// staleAt gives the stage whose stale dot a click at x, y falls on, which loads it again, as its menu's Refresh does:
// one on the bar, or, maximized, the selected stage's, after its name at the title's end, where the title shows it;
// -1 off every dot.
func (m Model) staleAt(x, y int) int {
	if !m.maximized {
		return m.bar().StaleAt(x, y)
	}
	from, until, _ := m.menuBar().OnSpan(m.width)
	if at := m.bar().LabelStaleAt(); y == shell.Top && from >= 0 && at >= 0 && x == from+at && x < until {
		return m.stage
	}
	return -1
}

// onMenu says whether a click at x, y is on the selected tab, which drops its menu down, as its : does: anywhere on
// its part of the bar, on any of the bar's lines; or, maximized, anywhere on its label at the title's end, as much of it
// as the title shows.
func (m Model) onMenu(x, y int) bool {
	if !m.maximized {
		return m.bar().OnSelected(x, y)
	}
	from, until, _ := m.menuBar().OnSpan(m.width)
	return y == shell.Top && from >= 0 && x >= from && x < until
}

// barTakes gives what the bar makes of a message, as its Update says, and whether it took it; while maximized, the bar
// is not drawn, so a click is never on it, and tab and shift-tab have no tabs to move along: it takes a tab's number
// alone.
func (m Model) barTakes(msg tea.Msg) (tabs.Bar, bool) {
	if k, ok := msg.(tea.KeyPressMsg); m.maximized && (!ok || k.String() == "tab" || k.String() == "shift+tab") {
		return tabs.Bar{}, false
	}
	return m.bar().Update(msg)
}

// dropDown drops a menu down: the app's, or the selected tab's, which on issues asks kit for the caller's login the
// first time, as Add to Pull Request needs it.
func (m *Model) dropDown(app bool) tea.Cmd {
	if app {
		m.appMenu = m.appMenu.Opened()
		return nil
	}
	m.openMenu()
	if !m.tabMenu.Open || m.stages[m.stage].name != issuesStage {
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
// people, and the caller's login, the blank columns held empty where kittui's settings hide them; the columns it
// offers are drawn from the rows, so it has none.
func (m Model) filter(i int) grid.Panel {
	s := m.stages[i]
	p := catalog[s.name].Ready(s.View)
	p.Me = m.me
	p.Focused, p.Outline, p.Fold, p.Glyphs, p.Empty = m.inPanel && i == m.stage, s.Open, m.fold, m.glyphs, nil
	if m.hide {
		p = p.Blanked(s.Blank, columnsOf(s.name))
	}
	return p
}

// setPanel keeps what the selected stage's panel became; the rows are let through again, and the cursor stays within
// them.
func (m *Model) setPanel(p grid.Panel) {
	m.stages[m.stage].View = p
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
	m.stages[m.stage].View = m.panel(m.stage).Find()
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

// groupClick takes a click that groups, as grouping says: a right click on a column's name in the header opens the
// column's card under it; a ctrl-click there groups the rows by it, as clickHeader says; on a group's heading, they no
// longer group by its column, the other groups kept.
func (m *Model) groupClick(x, y int, nest, right bool) tea.Cmd {
	if column := m.headerColumn(x, y); right && column != "" {
		return m.openColumnMenu(column)
	}
	if m.inHeader(y) {
		return m.clickHeader(x, y, nest, true)
	}
	lines := m.allLines(m.stage)
	k, ok := m.placeAt(y)
	if !ok || k < 0 || lines[k].Row < 0 {
		return nil
	}
	p := m.panel(m.stage)
	if column := p.Heading(m.rows(m.stage)[lines[k].Row]); column != "" {
		m.setPanel(p.Grouped(column, true))
	}
	return nil
}

// clickHeader takes a click on the header, which is no row, above its rule, where each column's sort target stands, as
// sortAt says: on its ▹ or ◃ it scrolls the columns, as table.ScrollColumns does; on its gutter it opens the header, or
// closes it, when it hides lines; on a column's name it opens the column's card under it. Where the terminal sends
// option or ctrl with a click, as many do not, also, as option asks, turns the column's sort, as the panel's Sorted
// says, and group, as ctrl asks, groups the rows by it within the groups they have, innermost, or no longer, as the
// panel's Grouped says. Elsewhere it does nothing.
func (m *Model) clickHeader(x, y int, also, group bool) tea.Cmd {
	on, fold, column := table.HeaderHit(m, x, y)
	switch p := m.panel(m.stage); {
	case !on:
	case table.ArrowAt(m, x, y) != 0 && !group:
		table.ScrollColumns(m, table.ArrowAt(m, x, y))
	case fold && !group:
		table.ToggleHeader(m)
	case column == "":
	case group:
		m.setPanel(p.Grouped(column, true))
	case also:
		m.setPanel(p.Sorted(column))
	default:
		return m.openColumnMenu(column)
	}
	return nil
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
	s.Offset, _, _, _ = pane.Window(len(m.lines()), m.rowsHeight(), s.Offset+n)
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
	// over a workspace, the caller's login is the one a member's forge knows, read in the repository given, or in the
	// first member cloned
	if r.Group != "" {
		r.Dir = m.ws.Checkout
		for _, w := range m.ws.Members {
			if r.Dir == "" && w.Checkout != "" {
				r.Dir = w.Checkout
			}
		}
		if r.Dir == "" {
			r.Dir = m.runner.Dir
		}
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		login, err := r.Me(ctx)
		return meMsg{login, err}
	}
}

// key takes a key press: the quitting keys, a field's popover's while it is open, the open menu's while one is open, the
// panel's while it has the keyboard, the preset line's while it has it, = for the app's menu and : or ; for the tab's,
// the bar's, then the rows', tab and shift-tab scrolling their columns while the view hides some, F1 and ? showing the
// legend or hiding it. Down from the bar reaches the preset line, where the stage has one, and the rows from it.
func (m Model) key(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "ctrl+c":
		m.interrupted = true
		return m, m.quit()
	case shell.Quit:
		return m, m.quit()
	}
	// an open layer, the topmost, takes the key, as layers.go orders them
	if n, cmd, ok := m.layers().Key(m, k); ok {
		return n, cmd
	}
	if m.inPanel {
		return m.panelKey(k)
	}
	if m.legendKey(k.String()) {
		return m, nil
	}
	if m.onPreset {
		key := m.presetKey
		if _, on := m.atStop(); on {
			key = m.headingKey
		}
		if cmd, ok := key(k); ok {
			return m, cmd
		}
	}
	if menu, app := menuFor(k); menu {
		return m, m.dropDown(app)
	}
	if cmd, ok := m.remedyKey(k); ok {
		return m, cmd
	}
	if bar, ok := m.barTakes(k); ok {
		m.choose(bar)
		return m, nil
	}
	if j := m.shortcut(k.String()); j >= 0 {
		return m, m.pick(j)
	}
	switch k.String() {
	case "/":
		return m, m.enterPanel()
	case shell.Workspaces:
		m.books = m.books.Opened()
	case "esc":
		if m.inRows {
			s := &m.stages[m.stage]
			s.Cursor, s.Offset, m.inRows = 0, 0, false
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
		if !m.inRows && (m.presets(m.stage) || len(m.stops()) > 0) {
			// from the bar, down reaches the control line first, on the heading over the groups' first stop
			m.toControls()
			return m, nil
		}
		m.down(1)
	case "pgup":
		m.page(-1)
	case "pgdown":
		m.page(1)
	case "tab", "shift+tab":
		// while the view hides columns, tab and shift-tab scroll them, as table.RowKeys names them
		if _, before, after := table.Scrolled(&m); m.inRows && (before || after) {
			d := 1
			if k.String() == "shift+tab" {
				d = -1
			}
			table.ScrollColumns(&m, d)
		}
	case "[", "]", "{", "}":
		if len(m.stages) > 0 {
			table.Group(&m, k.String())
		}
	case "home":
		m.end(-1)
	case "end":
		m.end(1)
	case "right", "alt+right":
		table.Expand(&m, true, k.String() == "alt+right")
	case "left", "alt+left":
		table.Expand(&m, false, k.String() == "alt+left")
	}
	return m, nil
}

// panelKey takes a key while the search has the keyboard, as panelKeys says; ctrl+r drops the searches remembered
// down, and what was typed is remembered once the search gives up the keyboard. The search has it only on a stage, as
// enterPanel gives it.
func (m Model) panelKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

// recallAt gives the screen column of the search's Recall glyph, as the title line draws it. It is asked only on a
// stage, whose search alone recalls.
func (m Model) recallAt() int { return shell.RecallAt(m.searchAt()) }

// panelKeys takes a key while the search has the keyboard, as panelKey says, and as shell.SearchKey takes it on every
// screen: enter, down, esc, and tab give the rows focus, the filter kept, where the stage has rows to stand on, else
// focus goes back where it was; a printable key is typed, backspace takes one back, and ctrl+u clears; any other key
// the bar takes, as a stage's number does not while it is typed, gives the bar the keyboard.
func (m Model) panelKeys(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	p := m.panel(m.stage)
	query, did := shell.SearchKey(p.Query, k)
	switch did {
	case shell.Left:
		m.inPanel = false
		if rows := m.rows(m.stage); m.first(m.stage) < len(rows) {
			m.inRows, m.onPreset = true, false
			m.clamp(m.stage)
			m.fit()
		}
		return m, nil
	case shell.Typed:
		p.Query = query
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

// args gives the words a stage's command takes beside its own, as its catalog entry's takes names them: the owner's
// flags, as kit flow's filters; and --no-fetch, where the fetch is skipped.
func (m Model) args(name string) []string {
	var out []string
	if takes(name, "owner") {
		out = append(out, m.filters...)
	}
	if takes(name, "no-fetch") && m.noFetch {
		out = append(out, "--no-fetch")
	}
	return out
}

// stageLoad gives a stage's read: its catalog entry's command, with the words it takes, for the limit at most, over
// the group the runner spans where it takes workspace; --fresh where it takes it and reads the forge afresh, as again
// asks, since kit shares one read of the open issues and pull requests a minute between the stages that need it, and a
// read without it takes that read; and, where it takes cached, what kit kept of the stage's last JSON, with --cached,
// as kept.go draws it. A stage that takes no cached keeps nothing of its last read: what git and kit.yml hold reads at
// once.
func (m Model) stageLoad(name string) source.Load[[]byte] {
	r, words, over := m.runner, append(slices.Clone(commandOf(name)), m.args(name)...), takes(name, "workspace")
	read := func(ctx context.Context, fresh bool) ([]byte, error) {
		if fresh && takes(name, "fresh") {
			return r.Rows(ctx, append(slices.Clone(words), "--fresh"), over)
		}
		return r.Rows(ctx, words, over)
	}
	var kept func(ctx context.Context) ([]byte, error)
	if takes(name, "cached") {
		kept = func(ctx context.Context) ([]byte, error) {
			return r.Rows(ctx, append(slices.Clone(words), "--cached"), over)
		}
	}
	return source.NewLoad(name, read, kept)
}

// load runs a stage's latest read, fresh or not.
func (m Model) load(name string, fresh bool) tea.Cmd { return m.find(name).src.Read(m.env(), fresh) }

// loads gives the first load of every stage kittui draws: fresh where kit flow --fresh asks.
func (m Model) loads() []tea.Cmd {
	var cmds []tea.Cmd
	for _, s := range m.stages {
		if sourced(s.name) {
			cmds = append(cmds, m.load(s.name, m.fresh))
		}
	}
	return cmds
}

// restart runs a stage's command again as its latest load, reading the forge afresh, ending the one before when it
// still runs. A stage that shows its rows keeps them, stale, its dot on its tab, until the new ones come; one that
// shows a refusal, or is loading, shows it loading until it answers.
func (m *Model) restart(name string) tea.Cmd {
	cmd, s := m.again(name, true), m.find(name)
	if !s.src.Loading && s.refusal == "" {
		s.src.Stale = true
		return cmd
	}
	s.src.Loading, s.refusal = true, ""
	s.rows, s.parsed, s.typed, s.failed, s.limited = nil, nil, nil, nil, 0
	m.rebuild()
	return cmd
}

// again runs a stage's command again as its latest load, fresh or not, ending the one before when it still runs; the
// stage keeps what it shows until the answer replaces it.
func (m *Model) again(name string, fresh bool) tea.Cmd { return m.find(name).src.Again(m.env(), fresh) }

// take keeps what a stage's command printed, decoded by the stage, or the refusal when it failed. The answer of a
// load a later one superseded is dropped, as is one for a stage the screen does not have. The cursor stays within the
// rows that came.
func (m *Model) take(msg source.Answer[[]byte]) {
	s := m.find(msg.Name)
	if s == nil {
		return
	}
	if _, landed := s.src.Take(msg); landed != source.Shown {
		return
	}
	err := msg.Err
	if err == nil {
		err = s.decode(msg.Value, m.runner.Group != "")
	}
	if err != nil {
		s.refusal = refusalOf(s.name, err)
	}
	m.blank(s)
	m.rebuild()
	m.autoGroup(s.name)
	for i := range m.stages {
		m.clamp(i)
	}
	m.land(s.name, true)
	m.fit()
}

// blank keeps which of a stage's columns no row it holds has a value in, as Table's Blank says.
func (m *Model) blank(s *stage) {
	s.Blank = grid.Blanks(laid(s.name, m.records(*s), func(id string) string { return s.View.Field(id).Rendering }),
		columnsOf(s.name))
}

// autoGroup groups a stage's rows as its view groups them as it loads, once it has rows to look at: by each column
// whose value is the same on every row the panel lets through, as the panel's AutoGrouped says, where kittui's
// settings auto-group and the view groups by nothing. A group the catalog places a field in, as source, organization,
// and repository, which the view does not name, is hidden where no row holds a value of it, rather than standing every
// row under a heading of none. It runs once a view, on the first rows after the view is set, as
// the screen opens or the preset line sets another; past that, the groups are the person's, as any they chose, and
// the preset line names the view still. The name is always one of the stages', as the stage that loaded, or was set,
// names itself.
func (m *Model) autoGroup(name string) {
	i := slices.IndexFunc(m.stages, func(s stage) bool { return s.name == name })
	s := &m.stages[i]
	if !s.Grouping || s.src.Loading || s.refusal != "" {
		return
	}
	s.Grouping = false
	all := laid(s.name, m.records(*s), func(id string) string { return s.View.Field(id).Rendering })
	p := m.panel(i)
	columns, named := columnsOf(s.name), m.chosen(i).Fields
	// a group the catalog gives, which the view does not name, stands aside where no row holds a value of it
	base := p
	for _, f := range fieldsOf(s.name) {
		if _, own := named[f.ID]; f.Placement == table.PlaceGroup && !own && grid.Blank(all, columns, f.ID) {
			base = base.Placed(f.ID, grid.Hidden)
		}
	}
	next := base.AutoGrouped(all, columns)
	if m.runner.Group != "" && base.Fold && base.Groupable["member"] && len(base.Groups()) == 0 {
		// over a group, the rows stand under their member first, then under what they share
		ids := slices.DeleteFunc(next.Groups(), func(id string) bool { return id == "member" })
		next = base.GroupedBy(append([]string{"member"}, ids...))
	}
	if slices.Equal(next.Groups(), p.Groups()) {
		return
	}
	s.View = next
	grouped := m.current(i)
	s.Grouped = &grouped
	m.rebuild()
}

// decode keeps what the stage's command printed, as its catalog entry reads it: its records, by the entry's record,
// or, over a group, for a stage that takes workspace, by .rows[] of kit's {rows, errors}, as table.Records reads
// them, fresh; over a group, a record that holds the entry's record, as incoming's object, is a member's document, and
// its records are read from it, each taking its member. Each record is decoded once, as parsed holds it. A refusal,
// named with the command: a document of another shape, as nothing, null, or an array where an object is expected; a
// record without its key, as the key's paths read it; two records of one key; and a record whose member or links are
// not what kit prints them as. Then what kit could not read, in the document's errors: the members whose read failed,
// and the features the forge has off here; how many rows kit read of more, where the document is limited; and, for a
// stage whose rows Go builds, the records as Go holds them, as builders decodes them. Each replaces what the stage held
// whole; a refusal leaves it as it was.
func (s *stage) decode(data []byte, over bool) error {
	c := catalog[s.name]
	if c == nil {
		return fmt.Errorf("kittui does not draw the stage '%s'", label(s.name))
	}
	over = over && takes(s.name, "workspace")
	record := c.Record
	if over {
		record = ".rows[]"
	}
	rows, err := table.Records(data, record, c.Slurp && !over)
	var shape *table.ShapeError
	switch {
	case errors.As(err, &shape):
		return fmt.Errorf("%s %w", command(s.name), err)
	case err != nil:
		return fmt.Errorf("%s: %w", command(s.name), err)
	case over && c.Record != ".[]":
		rows = membersRecords(rows, c.Record)
	}
	// the key's paths a record must hold, as a refusal names them: those not after ?, which may be missing
	var key []string
	for _, path := range strings.Split(cmp.Or(c.Key, ".key"), ",") {
		if path = strings.TrimSpace(path); !strings.HasSuffix(path, "?") {
			key = append(key, path)
		}
	}
	all, seen := make([]parsed, len(rows)), map[string]bool{}
	for k, r := range rows {
		p, ok := parsed{value: decodeRecord(r)}, false
		if p.key, ok = c.KeyFrom(p.value); !ok {
			return fmt.Errorf("%s printed a record without %s", command(s.name), strings.Join(key, ", "))
		}
		if p.member, p.links, ok = memberAndLinks(p.value); !ok {
			return fmt.Errorf("%s printed the record %s, whose member or links cannot be read", command(s.name),
				strings.TrimSpace(p.member+" "+p.key))
		}
		if one := p.member + " " + p.key; seen[one] {
			return fmt.Errorf("%s printed two records of the key %s", command(s.name), strings.TrimSpace(one))
		}
		seen[p.member+" "+p.key], all[k] = true, p
	}
	var typed any
	if b, ok := builders[s.name]; ok && b.decode != nil {
		if typed, err = b.decode(rows); err != nil {
			return fmt.Errorf("%s: %w", command(s.name), err)
		}
	}
	s.rows, s.parsed, s.typed = rows, all, typed
	s.failed, s.limited = failedIn(data)
	return nil
}

// memberAndLinks gives a record's member and its links, by the field's id, as kit prints them: a member is a text, or
// none; links an object of a link each, a text, a list of them, or null, or none. Anything else is not ok.
func memberAndLinks(v any) (member string, links map[string]any, ok bool) {
	object, _ := v.(map[string]any)
	switch m := object["member"].(type) {
	case string:
		member = m
	case nil:
	default:
		return "", nil, false
	}
	switch l := object["links"].(type) {
	case map[string]any:
		links = l
	case nil:
	default:
		return member, nil, false
	}
	for _, link := range links {
		switch link := link.(type) {
		case string, nil:
		case []any:
			if slices.ContainsFunc(link, func(each any) bool { _, text := each.(string); return !text }) {
				return member, nil, false
			}
		default:
			return member, nil, false
		}
	}
	return member, links, true
}

// membersRecords gives the records of each member's document over a group, as its record reads them, each taking the
// member its document names where it names none of its own; a row that is no such document is a member's record, kept
// as it is.
func membersRecords(rows []json.RawMessage, record string) []json.RawMessage {
	var out []json.RawMessage
	for _, doc := range rows {
		inner, err := table.Records(doc, record, false)
		if err != nil {
			out = append(out, doc)
			continue
		}
		var of struct {
			Member string `json:"member"`
		}
		_ = json.Unmarshal(doc, &of) // a document Records read is an object
		for _, r := range inner {
			var fields map[string]json.RawMessage
			if json.Unmarshal(r, &fields) == nil && fields["member"] == nil && of.Member != "" {
				fields["member"], _ = json.Marshal(of.Member)
				r, _ = json.Marshal(fields)
			}
			out = append(out, r)
		}
	}
	return out
}

// failedIn gives what kit could not read, as a document that is an object names it under errors: each member whose
// read failed, with kit's line, and each feature the forge has off here, over a group with its member; and how many
// rows kit read where the forge holds more, as the document's limited says. A document of another shape has neither,
// and an entry that says neither is dropped.
func failedIn(data []byte) (failed []failure, limited int) {
	var doc struct {
		Errors  []failure `json:"errors"`
		Limited int       `json:"limited"`
	}
	if json.Unmarshal(data, &doc) != nil {
		return nil, 0
	}
	return slices.DeleteFunc(doc.Errors, func(f failure) bool { return f.Error == "" && !f.Off }), doc.Limited
}

// failing says whether kit could not read a part of a stage: a member whose read failed, as its errors say, beside
// any feature off here, which is no failure.
func (s stage) failing() bool {
	return slices.ContainsFunc(s.failed, func(f failure) bool { return !f.Off })
}

// refusalOf gives a stage's refusal from its error: that the stage took longer than the deadline and the menu's
// Refresh All runs it again; else kit's line; and when kit died with nothing on stderr, how: exited with its code, or killed.
func refusalOf(name string, err error) string {
	var exit *kit.ExitError
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Sprintf("%s took longer than %d minutes; the menu's Refresh All runs it again", command(name),
			int(deadline.Minutes()))
	case err.Error() != "":
		return err.Error()
	case errors.As(err, &exit) && exit.Code >= 0:
		return fmt.Sprintf("%s exited %d with nothing on stderr", command(name), exit.Code)
	}
	return fmt.Sprintf("%s was killed", command(name))
}

// clamp keeps a stage's cursor within its rows, and, while they have the keyboard, past the heading over the groups,
// which it never stands on there; past the last row where only that heading shows.
func (m *Model) clamp(i int) {
	s, first := &m.stages[i], m.first(i)
	if n := len(m.rows(i)); s.Cursor >= n {
		s.Cursor = max(first, n-1)
	}
	if m.inRows && i == m.stage {
		s.Cursor = max(s.Cursor, first)
	}
}

// click takes a left click on the pane's rows: the cursor goes to the row clicked, and the rows have the keyboard, so
// space then selects it; a click on a linked glyph of a row's first line, as an issue's before its title, opens its
// page in the browser; one on the ▸ or ▾ of a row's first line, or of a heading, as foldAt says, and there alone,
// opens it, or closes it, as one on that first line pinned on top, once the row has scrolled it away, does. With
// subtree, as option asks, a click on a ▸ opens one level more under the node, as grow says, and one on a ▾ closes the
// deepest open level under it, as peel says, a level each click, as option with right or left does. A click on a …
// line, or on a line about the stage, does nothing.
func (m *Model) click(x, y int, subtree bool) tea.Cmd {
	if len(m.stages) == 0 || x >= m.paneWidth() {
		return nil
	}
	row, link, fold, ok := table.RowAt(m, x, y)
	if !ok {
		return nil
	}
	s, rows := &m.stages[m.stage], m.rows(m.stage)
	s.Cursor, m.inRows, m.inPanel, m.onPreset = row, true, false, false
	switch {
	case strings.HasPrefix(link, jumps):
		m.fit()
		return m.jump(strings.TrimPrefix(link, jumps))
	case link != "":
		m.fit()
		return m.openPage(link)
	case fold && subtree && s.Open.Unfolds(rows[row]):
		table.Peel(m)
		m.fit()
	case fold && subtree:
		table.Grow(m)
		m.fit()
	case fold:
		table.Disclose(m, !s.Open.Unfolds(rows[row]))
		m.fit()
	default:
		m.fit()
	}
	return nil
}

// controlClick takes a click on the control line, the pane's first: on the ▾ or ▸ before a column's name in the
// heading over the groups, or after the last, it opens or closes the level, as toggleLevel says; on a column's name
// it opens the column's card around it; on a filter's chip it opens its field's card under it, on its filter; on
// table.FieldsButton it opens the columns dialog; on the preset's selector it drops the presets down; elsewhere it
// gives the preset line the keyboard, where the stage has one.
func (m *Model) controlClick(x int) tea.Cmd {
	if level, ok := table.LevelAt(m, x); ok {
		table.ToggleLevel(m, level)
		return nil
	}
	if name, ok := table.NameAt(m, x); ok {
		return m.openFieldMenu(name.Column)
	}
	if spot, ok := table.SpotAt(m.spots(), x); ok {
		if spot.Field == "" {
			m.cols = table.Columns{Open: true}
			return nil
		}
		m.pop = table.OpenChip(m, spot)
		return m.askMe()
	}
	if m.presets(m.stage) {
		return m.presetClick(x)
	}
	return nil
}

// inHeader says whether the screen line y falls on the header, its rule apart.
func (m Model) inHeader(y int) bool {
	_, ok := m.headerLine(y)
	return ok && !m.onRule(y)
}

// stageOf gives the stage a link leads to, from the link's text after kittui:, as jump reads it: the stage whose name
// and a slash it starts with, the longest where two do, as changes/reviews/3 bob is changes/reviews's, not changes';
// -1 for none.
func (m Model) stageOf(to string) int {
	at := -1
	for i, s := range m.stages {
		if strings.HasPrefix(to, s.name+"/") && (at < 0 || len(s.name) > len(m.stages[at].name)) {
			at = i
		}
	}
	return at
}

// jump follows a link to a row of another stage, from the link's text after kittui:, the stage's name, a slash, and
// the row's key: the stage is selected, its rows have the keyboard, and the cursor is on the row, or where it was when
// the panel hides the row. A stage's name may hold a slash, as kit workspace's do, so the stage is the one the link
// starts with, as stageOf finds it. A row its stage has not loaded yet is under the cursor once it loads. A stage this
// app does not draw is another app's: the jump gives the command that asks kittui for that app, at that row.
func (m *Model) jump(to string) tea.Cmd {
	i := m.stageOf(to)
	if i < 0 {
		if app := appWith(to); app != "" && app != m.app {
			return func() tea.Msg { return shell.JumpMsg{Screen: app, To: to} }
		}
		return nil
	}
	m.choose(tabs.Bar{Selected: i, Scroll: m.barView})
	m.landing = to
	m.land(m.stages[i].name, false)
	m.fit()
	return nil
}

// land puts the row a jump waits for under the cursor, once the stage named has it. The jump is done then, or once
// the stage's load is done, with the row or without it; it waits while the stage loads, or shows what kit kept.
func (m *Model) land(name string, done bool) {
	i := m.stageOf(m.landing)
	if i < 0 || m.stages[i].name != name {
		return
	}
	key := strings.TrimPrefix(m.landing, name+"/")
	if k := slices.IndexFunc(m.rows(i), func(r row) bool { return r.Key == key }); k >= 0 {
		m.stages[i].Cursor, done = k, true
	}
	if done || (!m.stages[i].src.Loading && !m.stages[i].src.Stale) {
		m.landing = ""
	}
}
