// Package tutorials is kit tutorials on a terminal: the tutorials kit knows, as kit tutorials --json gives them, a
// card each, tabbed by audience, and the keys bar last. The tabs are All, then each audience in the catalogue's order,
// each counting its tutorials; a tag given on the command line, kit tutorials --tag <audience>, selects its audience's
// tab once the catalogue is read, and a tutorial's id given, kit tutorials <tutorial>, puts the cursor on its card, in
// that tab when it holds it, else in All. Under the bar, the pane draws the selected tab's tutorials as a grid of cards, as
// many columns as fit and at least one: each card has the tutorial's glyph and title in bold, a blank line, its summary
// on three lines with … where it runs over, and its audiences as pills in their colors on its bottom edge. The card
// under the cursor is drawn in the menus' bright cyan, and the pane scrolls a row of cards at a time so that card is
// always whole. A tab with no tutorial says so.
//
// The cards have the keyboard as the screen opens. The arrows move the cursor between them: left and right in reading
// order, on along the next row; up and down a column, down to the last card from a row it is not under; and up from the
// first row, or esc, gives the bar the keyboard. Enter opens the tutorial under the cursor in the browser, and a click
// on a card opens that one. The bar is internal/tabs: while it has the keyboard, left and right move between the tabs,
// and down or enter gives the cards the keyboard, on a tab that has some; a tab's number, 1 to 9, or a click on it,
// selects it from anywhere and gives the bar the keyboard. = or a click on the = drops the app's menu down, and - or a
// click on the workspace menu after the screen's name drops it down, as on every screen. The search
// on the menu bar's right is every screen's, internal/shell's: / or a click on it gives it focus, typing narrows every
// tab's cards to the tutorials whose id, title, or summary holds it, the tabs still counting all of theirs, and ⏎, ↓,
// esc, or tab gives the cards focus, the filter kept. Ctrl-c quits with 130 from anywhere, and ctrl-q quits with 0; esc
// never quits.
//
// Opening a tutorial says so in place of the keys, for Fades or until a key; when the browser cannot be asked, the
// tutorial's link stands there instead until a key, to open by hand. The browser cannot be asked over SSH, where an
// opener would open the page on the other machine, nor on Linux with no display for it.
package tutorials

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/source"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
)

// Opened is the glyph before the notice that a tutorial opened, the app's menu's glyph for the tutorials.
const Opened = shell.Learn

// browsedMsg says whether the browser was asked to open a tutorial: err is why not.
type browsedMsg struct {
	tutorial kit.Tutorial
	err      error
}

// Fades is how long the notice that a tutorial opened stays in place of the keys, short of a key. The link left when
// the browser could not be asked stays until a key.
const Fades = 5 * time.Second

// fades is how long the screen waits for Fades: Fades, as a test waits less.
var fades = Fades

// Model is the tutorials screen.
type Model struct {
	runner  kit.Runner
	ctx     context.Context            // ends with the screen, and kit's answer still running with it
	cancel  context.CancelFunc         // ends ctx
	browse  func(string) error         // opens a link in the browser; a recorder in tests
	tags    []string                   // the tags from the command line, the first that names an audience selecting its tab
	focus   string                     // the tutorial named on the command line, whose card the cursor starts on
	tagged  bool                       // whether the tags have selected a tab, which they do once
	reading source.Load[kit.Catalogue] // kit tutorials --json: loading until kit answers
	refusal string                     // kit's line, when it did not print the catalogue
	catalog kit.Catalogue
	tab     int  // the selected tab: 0 is All, then each audience in the catalogue's order
	barView int  // the cells the bar's view has scrolled past, as the bar last chose
	inPane  bool // whether the screen gives the cards the keyboard, which they have while the tab has some
	cursor  int  // the card under the cursor, an index into the selected tab's tutorials
	offset  int  // the first row of cards the pane shows
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
// tutorial named, the first other word that is no option, which the cursor starts on. Anything else is ignored.
func Open(r kit.Runner, words []string) Model {
	tags, focus := parse(words)
	m := New(r, tags)
	m.focus, m.recent = focus, shell.Recent{Path: shell.RecentFile()}.Loaded()
	return m
}

// parse gives the values of the --tag words, and the first other word that is no option.
func parse(words []string) (tags []string, focus string) {
	for i := 0; i < len(words); i++ {
		switch {
		case words[i] == "--tag" && i+1 < len(words):
			tags = append(tags, words[i+1])
			i++
		case focus == "" && !strings.HasPrefix(words[i], "-"):
			focus = words[i]
		}
	}
	return tags, focus
}

// New opens the screen with the cards' keyboard, reading. Init asks kit for the catalogue.
func New(r kit.Runner, tags []string) Model {
	read := func(ctx context.Context, _ bool) (kit.Catalogue, error) { return r.Tutorials(ctx, nil) }
	m := Model{runner: r, browse: shell.Browse, tags: tags, inPane: true, width: 80, height: 24,
		appMenu: shell.Menu{Here: shell.Tutorials}, reading: source.NewLoad("catalogue", read, nil)}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	return m
}

// Init asks kit for every tutorial: the tags select a tab, and narrow nothing, so All holds them all.
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

// Update takes a message, as update does; then a notice that a tutorial opened, new, goes after Fades, unless a key or
// another notice took its place first.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if f, ok := msg.(shell.FadedMsg); ok {
		m.note = m.note.Faded(f)
		return m, nil
	}
	next, cmd := m.update(msg)
	n := next.(Model)
	if fade := n.note.Fading(m.note, fades); fade != nil {
		cmd = tea.Batch(cmd, fade)
	}
	return n, cmd
}

// update takes a message.
func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case source.Answer[kit.Catalogue]:
		m.take(msg)
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
		return m.key(msg)
	case tea.MouseClickMsg:
		return m.click(msg)
	}
	m.fit()
	return m, nil
}

// take keeps the catalogue, or kit's refusal. The first catalogue selects the tab the tags name, and puts the cursor
// on the card of the tutorial named; the cursor stays in the selected tab's cards.
func (m *Model) take(msg source.Answer[kit.Catalogue]) {
	if _, landed := m.reading.Take(msg); landed != source.Shown {
		return
	}
	if msg.Err != nil {
		m.refusal = refusalOf(msg.Err)
		return
	}
	m.refusal, m.catalog = "", msg.Value
	if !m.tagged {
		m.tagged = true
		for _, tag := range m.tags {
			if i := slices.IndexFunc(m.catalog.Audiences, func(a kit.Audience) bool { return a.ID == tag }); i >= 0 {
				m.tab = i + 1
				break
			}
		}
		m.toFocus()
	}
	m.tab = min(m.tab, len(m.catalog.Audiences))
	m.cursor = min(m.cursor, max(0, len(m.cards())-1))
}

// toFocus puts the cursor on the card of the tutorial named on the command line, the cards given the keyboard: in the
// selected tab when it holds the card, else in All. A name that is no tutorial's moves nothing.
func (m *Model) toFocus() {
	if m.focus == "" {
		return
	}
	for _, tab := range []int{m.tab, 0} {
		if k := slices.IndexFunc(m.cardsOf(tab), func(t kit.Tutorial) bool { return t.ID == m.focus }); k >= 0 {
			m.tab, m.cursor, m.inPane = tab, k, true
			return
		}
	}
}

// refusalOf gives the line to show when kit did not print the catalogue: kit's line, or that it failed.
func refusalOf(err error) string {
	if err.Error() != "" {
		return err.Error()
	}
	return "kit tutorials --json failed"
}

// focused says whether the cards have the keyboard: the screen gave it them, and the selected tab has some.
func (m Model) focused() bool { return m.inPane && len(m.cards()) > 0 }

// cards gives the selected tab's tutorials, as cardsOf does.
func (m Model) cards() []kit.Tutorial { return m.cardsOf(m.tab) }

// cardsOf gives a tab's tutorials the search lets through, as allOf gives them: those whose id, title, or summary
// holds what is typed, without regard to case.
func (m Model) cardsOf(tab int) []kit.Tutorial {
	query := strings.ToLower(m.search.Query)
	return slices.DeleteFunc(m.allOf(tab), func(t kit.Tutorial) bool {
		return !strings.Contains(strings.ToLower(t.ID+"\n"+t.Title+"\n"+t.Summary), query)
	})
}

// allOf gives a tab's tutorials, in the catalogue's order: every one under All, else those for its audience.
func (m Model) allOf(tab int) []kit.Tutorial {
	if tab == 0 {
		return slices.Clone(m.catalog.Tutorials)
	}
	id := m.catalog.Audiences[tab-1].ID
	var out []kit.Tutorial
	for _, t := range m.catalog.Tutorials {
		if slices.Contains(t.Audience, id) {
			out = append(out, t)
		}
	}
	return out
}

// key takes a key press: the quitting keys, the app's menu's while it is open, the searches' dropdown's while it is,
// the search's while it has focus, = for the menu, / for the search, the bar's, then the cards'.
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
	if m.search.Focused {
		// the search takes what it takes on every screen; what it does not, as the arrows but down, the bar takes
		s, did := m.search.Key(k)
		switch did {
		case shell.Typed:
			m.typed(s.Query)
			return m, nil
		case shell.Left:
			m.search, m.recent = s, m.recent.Remember(s.Query)
			m.inPane = len(m.cards()) > 0
			m.fit()
			return m, nil
		case shell.Recalled:
			m.recent = m.recent.Opened()
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
	if bar, ok := m.bar().Update(k); ok {
		m.choose(bar)
		m.fit()
		return m, nil
	}
	n, cols := len(m.cards()), m.columns()
	var cmd tea.Cmd
	switch {
	case !m.focused():
		if (k.String() == "down" || k.String() == "enter") && n > 0 {
			m.inPane = true
		}
	case k.String() == "esc":
		m.inPane = false
	case n == 0:
	case k.String() == "left":
		m.cursor = max(m.cursor-1, 0)
	case k.String() == "right":
		m.cursor = min(m.cursor+1, n-1)
	case k.String() == "up" && m.cursor < cols:
		m.inPane = false
	case k.String() == "up":
		m.cursor -= cols
	case k.String() == "down" && m.cursor/cols < (n-1)/cols:
		m.cursor = min(m.cursor+cols, n-1)
	case k.String() == "enter":
		cmd = m.open(m.cards()[m.cursor])
	}
	m.fit()
	return m, cmd
}

// click takes a click: on the open app's menu, on =, on the bar, or on a card, which it opens.
func (m Model) click(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Button != tea.MouseLeft:
		return m, nil
	case m.appMenu.Open:
		var c shell.Choice
		m.appMenu, c = m.appMenu.Click(msg.X, msg.Y, m.width)
		return m, m.chose(c)
	case m.books.Open:
		var picked string
		m.books, picked = m.books.Click(msg.X, msg.Y, m.booksAt(), m.width)
		return m, shell.Pick(picked)
	case m.recent.Open:
		var query string
		if m.recent, query = m.recent.Click(msg.X, msg.Y, m.recallAt(), m.width); query != "" {
			m.typed(query)
		}
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
		m.search.Focused = true
		return m, nil
	}
	if bar, ok := m.bar().Update(msg); ok {
		m.choose(bar)
		m.fit()
		return m, nil
	}
	if i := m.cardAt(msg.X, msg.Y); i >= 0 {
		m.cursor, m.inPane = i, true
		m.fit()
		return m, m.open(m.cards()[i])
	}
	return m, nil
}

// typed puts the text in the search, the cursor on the first card it lets through and the pane at its top, when it
// differs.
func (m *Model) typed(query string) {
	if query != m.search.Query {
		m.search.Query, m.cursor, m.offset = query, 0, 0
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

// choose takes what the bar chose: where its view is scrolled, the tab it selected, and the keyboard. Another tab
// starts at its first card, its pane at its top.
func (m *Model) choose(bar tabs.Bar) {
	m.barView = bar.Scroll
	if bar.Selected != m.tab {
		m.tab, m.cursor, m.offset = bar.Selected, 0, 0
	}
	m.inPane = !bar.Focused
}

// open asks the browser for a tutorial's link, as it is, with the kit's version on it.
func (m Model) open(t kit.Tutorial) tea.Cmd {
	b := m.browse
	return func() tea.Msg { return browsedMsg{tutorial: t, err: b(t.URL)} }
}
