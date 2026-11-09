// Package tutorials is kit tutorials on a terminal: the tutorials kit knows, as kit tutorials --json gives them, a
// card each, tabbed by audience, and the keys bar last. The tabs are All, then each audience in the catalogue's order,
// each counting its tutorials; a tag given on the command line, kit tutorials --tag <audience>, selects its audience's
// tab once the catalogue is read. Under the bar, the pane draws the selected tab's tutorials as a grid of cards, as
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
// selects it from anywhere and gives the bar the keyboard. = or a click on the = drops the app's menu down. Ctrl-c quits
// with 130 from anywhere, and ctrl-q quits with 0; esc never quits.
//
// Opening a tutorial says so in place of the keys, for Fades or until a key; when the browser cannot be asked, the
// tutorial's link stands there instead until a key, to open by hand. The browser cannot be asked over SSH, where an
// opener would open the page on the other machine, nor on Linux with no display for it.
package tutorials

import (
	"context"
	"fmt"
	"slices"
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
	appMenu shell.Menu   // the app's menu, internal/shell's, which has the keyboard while it is open
	note    shell.Notice // what opened, or the link to open by hand, in place of the keys

	interrupted bool
}

// Open opens the screen as kittui does, as New does: the tags are kit tutorials' --tag words.
func Open(r kit.Runner, tags []string) Model { return New(r, tags) }

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
	case tea.KeyPressMsg:
		m.note.Text = ""
		return m.key(msg)
	case tea.MouseClickMsg:
		return m.click(msg)
	}
	m.fit()
	return m, nil
}

// take keeps the catalogue, or kit's refusal. The first catalogue selects the tab the tags name; the cursor stays in
// the selected tab's cards.
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
	}
	m.tab = min(m.tab, len(m.catalog.Audiences))
	m.cursor = min(m.cursor, max(0, len(m.cards())-1))
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

// cardsOf gives a tab's tutorials, in the catalogue's order: every one under All, else those for its audience.
func (m Model) cardsOf(tab int) []kit.Tutorial {
	if tab == 0 {
		return m.catalog.Tutorials
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

// key takes a key press: the quitting keys, the app's menu's while it is open, = for it, the bar's, then the cards'.
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
	if k.String() == "=" {
		m.appMenu = m.appMenu.Opened()
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
	case shell.OnApp(msg.X, msg.Y, appLabel):
		m.appMenu = m.appMenu.Opened()
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
