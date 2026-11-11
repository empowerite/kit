// Package tutorials is kit tutorials on a terminal: the tutorials kit knows, as kit tutorials --json gives them, a
// card each, under the table's control line, and the keys bar last. The tutorials are a table, internal/table's, of
// tutorials.json, the catalog: each tutorial a row, its title, its audiences, as labels, where it is, its summary, and
// its id. A tutorial is in the kit, a page that ships with it, read here, or online, a link; the where field says
// which, as its label, and so does the glyph on its card's top edge, 📦 or 🌐. The control line is the table's, on the
// screen's second line: the heading over the groups, which names the field the cards are grouped by, and the preset
// line, Default, By audience, or In the kit or online, the person's presets beside them, and Fields…,
// where the cards are sorted, grouped, and filtered by any field a column menu or the columns dialog offers. The cards
// stand as the table's rows stand: in its order, under its groups' headings, each heading drawn as the table draws it,
// a tutorial for several audiences under each of theirs, and a closed group's cards folded away. The tags given on the
// command line, kit tutorials --tag=<audience>, filter the cards to the tutorials for any of the audiences they name,
// once the catalogue is read, as the audience's column menu would; and a tutorial's id given, kit tutorials
// <tutorial>, puts the cursor on its card, the tags' filter dropped when it hides that card.
//
// Under the control line, the pane draws the cards as a grid, as many columns as fit and at least one: each card has
// the tutorial's glyph and title in bold, a blank line, its summary on three lines with … where it runs over, another
// blank line, and its audiences as pills in their colors on its bottom edge; the glyph of where it is stands at the
// right of its top edge. The card under the cursor is drawn in the
// menus' bright cyan, and the pane scrolls a row of cards, or a heading, at a time so that card is always whole. A
// catalogue with no tutorial says so, as a search or a filter that lets none through does.
//
// The cards have the keyboard as the screen opens. Tab and shift-tab move the cursor between them, its stops, in
// reading order, on along the next row, as they move between the stops on any line; up and down move to the row of
// cards over or under it, past a heading, to the last card of a row shorter than the cursor's column; and up from the
// first row, or esc, gives the control line the keyboard. Enter opens the tutorial under the cursor, and a click on a
// card opens that one: one online in the browser; one in the kit here, its page, drawn as internal/markdown draws it,
// in place of the cards, and the line over it, the control line's, a 🔙 back to the cards with the tutorial's glyph
// and title. Up and down scroll the page a line, page up and page down a page, home and end to its ends, as the wheel
// scrolls it; esc, or a click on the line over it, goes back to the cards, the cursor where it was, and / gives the
// search focus over them. Left and right, which fold and unfold, have nothing to fold on a card. The
// mouse wheel scrolls the cards a row, or a heading, at a time, never past their ends, the cursor left where it is
// until a key moves it. The control line takes the keys it takes on every table, its menus and its dialog every key
// while open, and down gives the cards the keyboard back; it is the top of the screen, so up and esc leave it where it
// is. = or a click on the = drops the app's menu down, and - or a click on the workspace menu after the screen's name
// drops it down, as on every screen. The search on the menu bar's right is every screen's, internal/shell's: / or a
// click on it gives it focus, typing narrows the cards to the tutorials whose title, audiences, summary, or id holds
// it, and ⏎, ↓, esc, or tab gives the cards focus, the filter kept, or the control line where it lets no card through.
// Ctrl-c quits with 130 from anywhere, and ctrl-q quits with 0; esc never quits.
//
// Opening a tutorial says so in place of the keys, for Fades or until a key; when the browser cannot be asked, the
// tutorial's link stands there instead until a key, to open by hand. The browser cannot be asked over SSH, where an
// opener would open the page on the other machine, nor on Linux with no display for it. What the table did, as saving
// a preset, or why not, stands there too, while it says it.
package tutorials

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/source"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// Opened is the glyph before the notice that a tutorial opened, the app's menu's glyph for the tutorials.
const Opened = shell.Learn

//go:embed tutorials.json
var catalogJSON []byte

// tutorialsTable is the name of the tutorials' table in the catalog, which keys the person's presets of it too.
const tutorialsTable = "tutorials"

// audienceField is the field that holds a tutorial's audiences, which the tags filter by.
const audienceField = "audience"

// whereField is the field that says where a tutorial is, as its label: inTheKit or online.
const (
	whereField = "where"
	inTheKit   = "in the kit"
	online     = "online"
)

// The glyphs of where a tutorial is, on its card's top edge: in the kit, Shipped, or online, Online; and Back, before
// the line over a tutorial's page, a click on which goes back to the cards. Each is two cells wide, with no variation
// selector.
const (
	Shipped = "📦"
	Online  = "🌐"
	Back    = "🔙"
)

// wheelLines is the lines a turn of the mouse wheel scrolls a tutorial's page.
const wheelLines = 3

// tableColor is the table's color, the accent's.
const tableColor = "12"

// catalog is tutorials.json, read; a catalog that does not read is a bug in the build, so it panics.
var catalog = func() table.Catalog {
	c, err := table.ReadCatalog(catalogJSON)
	if err != nil {
		panic(fmt.Sprintf("tutorials.json: %v", err))
	}
	return c
}()

// browsedMsg says whether the browser was asked to open a tutorial: err is why not.
type browsedMsg struct {
	tutorial kit.Tutorial
	err      error
}

// readMsg brings a tutorial's page, in the kit, or why it could not be read.
type readMsg struct {
	tutorial kit.Tutorial
	page     string
	err      error
}

// reader is a tutorial in the kit open in place of the cards: whether one is, the tutorial, its page's markdown, or
// that it has none, and the first line of the page the pane shows.
type reader struct {
	open     bool
	tutorial kit.Tutorial
	page     string
	missing  bool
	offset   int
}

// Fades is how long the notice that a tutorial opened stays in place of the keys, short of a key. The link left when
// the browser could not be asked stays until a key, and so does why the table did not do something.
const Fades = 5 * time.Second

// fades is how long the screen waits for Fades: Fades, as a test waits less.
var fades = Fades

// Model is the tutorials screen.
type Model struct {
	runner  kit.Runner
	ctx     context.Context            // ends with the screen, and kit's answer still running with it
	cancel  context.CancelFunc         // ends ctx
	browse  func(string) error         // opens a link in the browser; a recorder in tests
	tags    []string                   // the tags from the command line, which filter the cards by audience
	focus   string                     // the tutorial named on the command line, whose card the cursor starts on
	tagged  bool                       // whether the tags have filtered the cards, which they do once
	reading source.Load[kit.Catalogue] // kit tutorials --json: loading until kit answers
	refusal string                     // kit's line, when it did not print the catalogue
	catalog kit.Catalogue
	sheet   table.Sheet // the tutorials' table, whose control line stands over the cards and whose rows they follow
	read    reader      // the tutorial in the kit open in place of the cards, while one is
	inPane  bool        // whether the screen gives the cards the keyboard, which they have while there are some
	cursor  int         // the card under the cursor, an index into the cards
	offset  int         // the first band of the pane it shows, a heading or a row of cards
	wheeled bool        // whether the wheel scrolled the pane, which leaves the cursor's card until a key moves it
	width   int
	height  int
	appMenu shell.Menu      // the app's menu, internal/shell's, which has the keyboard while it is open
	note    shell.Notice    // what opened, or the link to open by hand, in place of the keys
	books   shell.BooksMenu // the menu bar's workspace menu, which has the keyboard while open
	search  shell.Search    // the search, internal/shell's, which narrows the cards to its text
	recent  shell.Recent    // the searches remembered, every screen's, and their dropdown

	interrupted bool
}

// Open opens the screen as kittui does, as New does, from kit tutorials' words: the value of each --tag, and the
// tutorial named, the first other word that is no option, which the cursor starts on; the person's presets of the
// table are read from the file kittui keeps them in. Anything else is ignored.
func Open(r kit.Runner, words []string) Model {
	tags, focus := parse(words)
	m := newModel(r, tags, table.PresetsPath())
	m.focus, m.recent = focus, shell.Recent{Path: shell.RecentFile()}.Loaded()
	return m
}

// parse gives the values of the --tag words, as --tag=<audience> or --tag <audience>, and the first other word that
// is no option.
func parse(words []string) (tags []string, focus string) {
	for i := 0; i < len(words); i++ {
		switch tag, ok := strings.CutPrefix(words[i], "--tag="); {
		case ok:
			tags = append(tags, tag)
		case words[i] == "--tag" && i+1 < len(words):
			tags = append(tags, words[i+1])
			i++
		case focus == "" && !strings.HasPrefix(words[i], "-"):
			focus = words[i]
		}
	}
	return tags, focus
}

// New opens the screen with the cards' keyboard, reading, the person's presets of the table kept for the session
// alone. Init asks kit for the catalogue.
func New(r kit.Runner, tags []string) Model { return newModel(r, tags, "") }

// newModel opens the screen as New does, the person's presets of the table read from and kept in the file at presets.
func newModel(r kit.Runner, tags []string, presets string) Model {
	read := func(ctx context.Context, _ bool) (kit.Catalogue, error) { return r.Tutorials(ctx, nil) }
	m := Model{runner: r, browse: shell.Browse, tags: tags, inPane: true, width: 80, height: 24,
		appMenu: shell.Menu{Here: shell.Tutorials}, reading: source.NewLoad("catalogue", read, nil),
		sheet: table.NewSheet(catalog[tutorialsTable], tutorialsTable, tableColor, presets)}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.place()
	return m
}

// Init asks kit for every tutorial: the tags filter the cards, as the person can change, so kit gives them all.
func (m Model) Init() tea.Cmd { return m.reading.Read(source.Env{Ctx: m.ctx}, false) }

// Interrupted says whether ctrl-c ended the screen.
func (m Model) Interrupted() bool { return m.interrupted }

// Close ends kit's answer still running, as quitting does, for a screen kittui leaves hidden as it ends.
func (m Model) Close() { m.cancel() }

// Noticing says whether the keys bar's line says what opened, or gives a link to open, in place of the keys, until a
// key: kittui keeps it on the screen while the keys bar is hidden.
func (m Model) Noticing() bool { return m.note.Text != "" }

// quit ends the screen, and kit's answer still running with it.
func (m Model) quit() tea.Cmd {
	m.cancel()
	return tea.Quit
}

// chose does what a choice from the app's menu asks: a screen asks kittui to show it in this one's place, and quit
// ends the screen.
func (m Model) chose(c shell.Choice) tea.Cmd {
	switch {
	case c.Quit:
		return m.quit()
	case c.Screen != "":
		return shell.Switch(c.Screen)
	}
	return nil
}

// Update takes a message, as update does; then a notice that a tutorial opened, or that the table did something, new,
// goes after Fades, unless a key or another notice took its place first.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if f, ok := msg.(shell.FadedMsg); ok {
		m.note = m.note.Faded(f)
		m.sheet.Faded(f)
		return m, nil
	}
	before := m.sheet.Notice()
	next, cmd := m.update(msg)
	n := next.(Model)
	if fade := n.note.Fading(m.note, fades); fade != nil {
		cmd = tea.Batch(cmd, fade)
	}
	if fade := n.sheet.Notice().Fading(before, fades); fade != nil {
		cmd = tea.Batch(cmd, fade)
	}
	return n, cmd
}

// update takes a message.
func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.place()
	case shell.SettingsMsg:
		m.sheet.Settings(msg.Settings.FoldShared, msg.Settings.SharedEmoji)
	case source.Answer[kit.Catalogue]:
		m.take(msg)
	case readMsg:
		m.read = reader{open: true, tutorial: msg.tutorial, page: msg.page, missing: msg.err != nil}
	case browsedMsg:
		if msg.err == nil {
			m.note = m.note.Did(fmt.Sprintf("%s Opened %s in the browser", Opened, msg.tutorial.Title))
		} else {
			m.note.Text, m.note.Done = "Open "+msg.tutorial.URL+" in a browser", false
		}
	case shell.WorkspaceMsg:
		m.books = m.books.Told(msg)
	case tea.KeyPressMsg:
		m.note.Text = ""
		next, cmd := m.key(msg)
		if n := next.(Model); n.wheeled && (n.cursor != m.cursor || n.focused() != m.focused()) {
			// the cursor moved, or the cards took the keyboard: the pane follows the cursor again
			n.wheeled = false
			n.fit()
			return n, cmd
		}
		return next, cmd
	case tea.MouseClickMsg:
		return m.click(msg)
	case tea.MouseWheelMsg:
		m.wheel(msg)
	}
	m.fit()
	return m, nil
}

// wheel scrolls the cards a band for a turn of the mouse wheel over them, a row of cards or a heading, down or up,
// never past their ends; the keyboard stays where it is, and so does the cursor, until a key moves it, or gives the
// cards the keyboard, and the pane follows it again. A turn off the cards, or while a menu or a layer of the table is
// open, scrolls nothing.
func (m *Model) wheel(w tea.MouseWheelMsg) {
	d := 0
	switch w.Button {
	case tea.MouseWheelUp:
		d = -1
	case tea.MouseWheelDown:
		d = 1
	}
	if d == 0 || m.appMenu.Open || m.books.Open || m.recent.Open || m.sheet.Layered() || w.Y < paneTop ||
		w.Y >= paneTop+m.paneHeight() {
		return
	}
	if m.read.open {
		m.scroll(d * wheelLines)
		return
	}
	m.offset, m.wheeled = m.offset+d, true
}

// place puts the table on the screen's second line, its control line alone, as the cards stand for its rows.
func (m *Model) place() { m.sheet.Place(controlTop, 1, m.width, m.height) }

// take keeps the catalogue, or kit's refusal, and gives the table its tutorials. The first catalogue filters the cards
// by the audiences the tags name, and puts the cursor on the card of the tutorial named, or gives the control line
// the keyboard where the tags let no card through; the cursor stays among the cards.
func (m *Model) take(msg source.Answer[kit.Catalogue]) {
	if _, landed := m.reading.Take(msg); landed != source.Shown {
		return
	}
	if msg.Err != nil {
		m.refusal = refusalOf(msg.Err)
		return
	}
	m.refusal, m.catalog = "", msg.Value
	m.sheet.SetRecords(m.records())
	if !m.tagged {
		m.tagged = true
		m.filterByTags()
		m.toFocus()
		if len(m.cards()) == 0 && len(m.catalog.Tutorials) > 0 {
			m.toControls()
		}
	}
	m.cursor = min(m.cursor, max(0, len(m.cards())-1))
}

// records gives the table's records, a tutorial each, keyed by its id: its place in the catalogue, its title, its
// audiences as labels, named and colored as the catalogue names and colors them, an audience it lacks named by its
// id, its summary, and its id.
func (m Model) records() []table.Record {
	out := make([]table.Record, len(m.catalog.Tutorials))
	for k, t := range m.catalog.Tutorials {
		var labels []table.Label
		for _, a := range m.audiences(t) {
			labels = append(labels, table.Label{Name: a.Label, Color: a.Color})
		}
		out[k] = table.Record{Row: grid.Row{Key: t.ID}, Values: map[string]table.Value{
			"order": table.One(strconv.Itoa(k + 1)), "title": table.Some(t.Title),
			audienceField: table.Labelled(labels), whereField: table.Labelled([]table.Label{{Name: whereOf(t)}}),
			"summary": table.Some(t.Summary), "tutorial": table.Some(t.ID)}}
	}
	return out
}

// whereOf gives where a tutorial is, as the where field's label names it.
func whereOf(t kit.Tutorial) string {
	if t.Shipped() {
		return inTheKit
	}
	return online
}

// filterByTags filters the cards to the tutorials for any of the audiences the tags name, by their labels, as the
// audience's column menu filters them; tags that name none filter nothing.
func (m *Model) filterByTags() {
	var labels []string
	for _, a := range m.catalog.Audiences {
		if slices.Contains(m.tags, a.ID) && !slices.Contains(labels, a.Label) {
			labels = append(labels, a.Label)
		}
	}
	if len(labels) > 0 {
		m.sheet.SetPanel(m.sheet.Panel().FilteredBy(audienceField, grid.Filter{Values: labels}))
	}
}

// toFocus puts the cursor on the card of the tutorial named on the command line, the cards given the keyboard; where
// the tags' filter hides it, the filter is dropped first. A name that is no tutorial's moves nothing.
func (m *Model) toFocus() {
	if m.focus == "" || !slices.ContainsFunc(m.catalog.Tutorials, func(t kit.Tutorial) bool { return t.ID == m.focus }) {
		return
	}
	at := func() int { return slices.IndexFunc(m.cards(), func(t kit.Tutorial) bool { return t.ID == m.focus }) }
	if at() < 0 {
		m.sheet.SetPanel(m.sheet.Panel().FilteredBy(audienceField, grid.Filter{}))
	}
	if k := at(); k >= 0 {
		m.cursor, m.inPane = k, true
	}
}

// refusalOf gives the line to show when kit did not print the catalogue: kit's line, or that it failed.
func refusalOf(err error) string {
	if err.Error() != "" {
		return err.Error()
	}
	return "kit tutorials --json failed"
}

// focused says whether the cards have the keyboard: the screen gave it them, and there are some.
func (m Model) focused() bool { return m.inPane && len(m.cards()) > 0 }

// controlled says whether the table's control line has the keyboard.
func (m Model) controlled() bool { return m.sheet.Focused() }

// toControls gives the control line the keyboard, from the cards.
func (m *Model) toControls() {
	m.inPane = false
	m.sheet.Focus()
}

// toCards gives the cards the keyboard, from the control line, the cursor where it was among them.
func (m *Model) toCards() {
	m.sheet.Blur()
	m.inPane = true
}

// cards gives the tutorials the table's rows stand for, in their order, one a row: a tutorial under several groups
// stands under each; the groups' headings, and the rows a closed group folds away, are none.
func (m Model) cards() []kit.Tutorial {
	var out []kit.Tutorial
	for _, r := range m.sheet.Rows() {
		if r.Aside {
			continue
		}
		if k := slices.IndexFunc(m.catalog.Tutorials, func(t kit.Tutorial) bool { return t.ID == r.Key }); k >= 0 {
			out = append(out, m.catalog.Tutorials[k])
		}
	}
	return out
}

// key takes a key press: the quitting keys, the app's menu's while it is open, the searches' dropdown's while it is,
// the table's layers', the search's while it has focus, the control line's while it has the keyboard, = for the menu,
// / for the search, then the cards'.
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
			m.typed(query)
		}
		return m, nil
	}
	if m.sheet.Layered() {
		m.sheet.Key(k)
		m.settle()
		return m, nil
	}
	if m.search.Focused {
		// the search takes what it takes on every screen; what it does not, as the arrows but down, nobody does
		s, did := m.search.Key(k)
		switch did {
		case shell.Typed:
			m.typed(s.Query)
			return m, nil
		case shell.Left:
			m.search, m.recent = s, m.recent.Remember(s.Query)
			if len(m.cards()) > 0 {
				m.toCards()
			} else {
				m.toControls()
			}
			m.fit()
			return m, nil
		case shell.Recalled:
			m.recent = m.recent.Opened()
			return m, nil
		}
	}
	if m.read.open && !m.search.Focused {
		return m.paging(k)
	}
	if m.controlled() {
		// the control line is the top of the screen: up and esc, which give a tab bar the keyboard on another, leave
		// it where it is, but where they stop a preset's name being typed
		if key := k.String(); (key == "up" || key == "esc") && !m.sheet.Table().SavingAs {
			return m, nil
		}
		if m.sheet.Key(k) {
			if m.sheet.InRows() {
				m.toCards()
			}
			m.settle()
			return m, nil
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
		m.search.Focused = true
		return m, nil
	}
	n := len(m.cards())
	var cmd tea.Cmd
	switch {
	case m.controlled():
	case !m.focused():
		// nothing has the keyboard, as with no card to give it: the arrows and enter give it the control line
		switch k.String() {
		case "up", "down", "enter", "esc":
			m.toControls()
		}
	case k.String() == "esc":
		m.toControls()
	case k.String() == "shift+tab":
		m.cursor = max(m.cursor-1, 0)
	case k.String() == "tab":
		m.cursor = min(m.cursor+1, n-1)
	case k.String() == "up":
		if to, ok := m.vertical(-1); ok {
			m.cursor = to
		} else {
			m.toControls()
		}
	case k.String() == "down":
		if to, ok := m.vertical(1); ok {
			m.cursor = to
		}
	case k.String() == "enter":
		cmd = m.open(m.cards()[m.cursor])
	}
	m.fit()
	return m, cmd
}

// settle keeps the cursor among the cards once the table changed them, the pane showing it, and gives the control
// line the keyboard where the cards had it and none is left.
func (m *Model) settle() {
	n := len(m.cards())
	m.cursor = min(m.cursor, max(0, n-1))
	if m.inPane && n == 0 && !m.sheet.Layered() {
		m.toControls()
	}
	m.fit()
}

// vertical gives the card the cursor moves to on the row of cards over it, by -1, or under it, by 1, past a heading:
// the card in its column, or the row's last where the row is shorter; and whether there is such a row.
func (m Model) vertical(by int) (int, bool) {
	bands := m.bands()
	b, col := m.bandOf(bands, m.cursor)
	for at := b + by; at >= 0 && at < len(bands); at += by {
		if cards := bands[at].cards; len(cards) > 0 {
			return cards[min(col, len(cards)-1)], true
		}
	}
	return m.cursor, false
}

// click takes a click: on the open app's menu, on =, on a layer of the table's, on its control line, or on a card,
// which it opens.
func (m Model) click(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
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
			m.typed(query)
		}
		return m, nil
	case m.sheet.Layered():
		m.sheet.Click(msg)
		m.settle()
		return m, nil
	case msg.Button != tea.MouseLeft:
		return m, nil
	case shell.OnApp(msg.X, msg.Y, appLabel):
		m.appMenu = m.appMenu.Opened()
		return m, nil
	case m.books.On(msg.X, msg.Y, m.booksAt()):
		m.books = m.books.Opened()
		return m, nil
	case msg.Y == shell.Top && m.recalls() && msg.X == m.recallAt():
		m.recent = m.recent.Opened()
		return m, nil
	case msg.Y == shell.Top && msg.X >= m.searchAt():
		m.search.Focused, m.read.open = true, false
		return m, nil
	case m.read.open:
		// over a tutorial's page, the line over it goes back to the cards; the page itself takes no click
		if msg.Y == controlTop {
			m.read.open = false
			m.fit()
		}
		return m, nil
	case msg.Y == controlTop:
		if m.sheet.Click(msg) {
			m.search.Focused, m.inPane = false, !m.sheet.Focused()
			m.settle()
		}
		return m, nil
	}
	if i := m.cardAt(msg.X, msg.Y); i >= 0 {
		m.sheet.Blur()
		m.search.Focused, m.cursor, m.inPane = false, i, true
		m.fit()
		return m, m.open(m.cards()[i])
	}
	return m, nil
}

// typed puts the text in the search, and the table's, the cursor on the first card it lets through and the pane at its
// top, when it differs.
func (m *Model) typed(query string) {
	if query != m.search.Query {
		m.search.Query, m.cursor, m.offset = query, 0, 0
		m.sheet.Search(query)
	}
	m.fit()
}

// recalls says whether the search, with focus, offers searches to recall.
func (m Model) recalls() bool { return m.search.Focused && len(m.recent.Items) > 0 }

// recallAt gives the screen column of the search's Recall glyph.
func (m Model) recallAt() int { return shell.RecallAt(m.searchAt()) }

// searchAt gives the column the search starts at on the menu bar, as internal/shell puts every screen's.
func (m Model) searchAt() int { return m.menuBar().SearchAt(m.width) }

// booksAt gives the column the menu bar's workspace menu starts at.
func (m Model) booksAt() int { return m.menuBar().WorkspaceAt() }

// searchLine gives the search as the menu bar draws it, its cursor showing while it has focus and the app's menu is
// closed.
func (m Model) searchLine() string {
	return shell.RecallLine(m.search.Query, m.search.Focused && !m.appMenu.Open, len(m.recent.Items) > 0)
}

// open opens a tutorial: one in the kit, by reading its page, to show in place of the cards; one online, by asking
// the browser for its link, as it is, with the kit's version on it.
func (m Model) open(t kit.Tutorial) tea.Cmd {
	if t.Shipped() {
		return func() tea.Msg {
			page, err := os.ReadFile(t.File)
			return readMsg{tutorial: t, page: string(page), err: err}
		}
	}
	b := m.browse
	return func() tea.Msg { return browsedMsg{tutorial: t, err: b(t.URL)} }
}

// paging takes a key while a tutorial's page is open in place of the cards: esc goes back to them, / gives the
// search focus over them, and the arrows, page up and page down, home, and end scroll the page; = and the workspace
// menu's key drop their menus down, as on the cards.
func (m Model) paging(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "=":
		m.appMenu = m.appMenu.Opened()
	case shell.Workspaces:
		m.books = m.books.Opened()
	case "/":
		m.read.open, m.search.Focused = false, true
		m.fit()
	case "esc":
		m.read.open = false
		m.fit()
	case "up":
		m.scroll(-1)
	case "down":
		m.scroll(1)
	case "pgup":
		m.scroll(-m.pageStep())
	case "pgdown":
		m.scroll(m.pageStep())
	case "home":
		m.read.offset = 0
	case "end":
		m.scroll(len(m.pageLines()))
	}
	return m, nil
}

// scroll moves the page's first line shown by d, never before its first line nor past where its last shows.
func (m *Model) scroll(d int) {
	n, h := len(m.pageLines()), m.paneHeight()
	last := 0
	if n > h {
		// the last line shows once the window, with its … above taking a line, reaches it
		last = n - h + 1
	}
	m.read.offset = max(0, min(m.read.offset+d, last))
}

// pageStep gives the lines page up and page down scroll the page: the pane's, less the two that count the hidden
// lines, and one at least.
func (m Model) pageStep() int { return max(1, m.paneHeight()-2) }
