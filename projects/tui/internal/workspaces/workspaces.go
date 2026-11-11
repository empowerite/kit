// Package workspaces is the workspaces' screen, which the workspace menu's Manage… opens: every group kit keeps, as kit
// workspaces --json lists them, a row each on a table, with whether it is a favorite, how many members it has, and
// the members. The actions are the app's menu's own lines, each on the row under the cursor but New Workspace…, and
// each a kit workspaces command kittui runs, never writing the file itself: New Workspace…, Rename…, Add Members…,
// which takes repositories and patterns, Add an Owner's Repositories…, which lists a GitHub owner's with gh, a pattern
// keeping those it matches, Add from a Command…, whose every line is a repository, Remove Members…, Favorite or
// Unfavorite, Switch to It, and Delete…. An action that wants an answer, or that cannot be undone, opens a dialog
// first; kit's line, when it refuses, stands there. When kit has done it, what it said stands in place of the keys,
// the groups are read again, and kittui asks kit for the workspace again, so every screen's workspace menu lists the
// groups as they are now.
//
// The rows have the keyboard as the screen opens. A line's key, at its right in the app's menu, does the same with the
// menu closed: n, r, a, o, c, x, f, d, and enter, which switches to the group under the cursor. = or a click on the =
// drops the app's menu down, and - or a click on the workspace menu after the screen's name drops it down, as on every
// screen; / or a click on the search at the right gives the search the keyboard, which narrows the rows to what is
// typed. The mouse wheel scrolls the rows, the cursor left where it is, but not under an open menu or dialog. Ctrl-c
// quits with 130, and ctrl-q with 0.
package workspaces

import (
	"context"
	_ "embed"
	"regexp"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

//go:embed workspaces.json
var catalogJSON []byte

// groupsTable is the groups' table in the catalog, which keys the person's presets of it too.
const groupsTable = "workspaces/groups"

// catalog is workspaces.json, read; a catalog that does not read is a bug, which the package's tests catch.
var catalog, _ = table.ReadCatalog(catalogJSON)

// appLabel is the screen's name on the title line, after the app menu's =.
const appLabel = "kit workspaces"

// Failed starts kit's line, when it refused.
const Failed = "❌"

// fades is how long what kit did stands in place of the keys, short of a key.
var fades = table.Fades

// The actions, as the app's menu names them.
const (
	New       = "New Workspace…"
	Rename    = "Rename…"
	Add       = "Add Members…"
	Owner     = "Add an Owner's Repositories…"
	Command   = "Add from a Command…"
	Remove    = "Remove Members…"
	Favorite  = "Favorite"
	Disfavor  = "Unfavorite"
	SwitchTo  = "Switch to It"
	DeleteOne = "Delete…"
)

// The actions' glyphs, each two cells wide, as every glyph in a menu is.
const (
	Write  = "📝"
	Letter = "🔤"
	Plus   = "➕"
	Octo   = "🐙"
	Shell  = "🐚"
	Minus  = "➖"
	Star   = "⭐"
	Go     = "🔀"
	Trash  = "🚮"
	NoStar = "🌑"
)

// listedMsg is kit's answer to the read of the groups: its number, and the list, or why there is none.
type listedMsg struct {
	gen  int
	list kit.WorkspaceList
	err  error
}

// actedMsg is kit's answer to an action: what kit printed, or its refusal, and the group renamed from and to, or
// deleted, to empty.
type actedMsg struct {
	out      string
	err      error
	from, to string
}

// Model is the workspaces' screen.
type Model struct {
	runner   kit.Runner
	ctx      context.Context
	cancel   context.CancelFunc
	gen      int    // the latest read of the groups: an older read's answer is dropped
	loading  bool   // whether the groups are being read
	refusal  string // kit's line, when it did not list the groups
	list     kit.WorkspaceList
	sheet    *table.Sheet
	width    int
	height   int
	appMenu  shell.Menu
	books    shell.BooksMenu
	prompt   prompt
	search   shell.Search // the search, every screen's, which narrows the rows to its text
	note     shell.Notice
	settings shell.Settings

	interrupted bool
}

// Open opens the screen as kittui does, reading the groups once Init runs.
func Open(r kit.Runner) Model {
	sheet := table.NewSheet(catalog[groupsTable], "workspaces", "", table.PresetsPath())
	m := Model{runner: r, sheet: &sheet, width: 80, height: 24, appMenu: shell.Menu{Here: shell.Manage},
		loading: true}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.sheet.EnterRows()
	return m
}

// Init asks kit for the groups.
func (m Model) Init() tea.Cmd { return m.read() }

// read gives the command that asks kit for the groups, as the latest read.
func (m Model) read() tea.Cmd {
	r, ctx, gen := m.runner, m.ctx, m.gen
	return func() tea.Msg {
		list, err := r.Workspaces(ctx)
		return listedMsg{gen: gen, list: list, err: err}
	}
}

// Interrupted says whether ctrl-c ended the screen.
func (m Model) Interrupted() bool { return m.interrupted }

// Close ends what kit still runs for the screen.
func (m Model) Close() { m.cancel() }

// Noticing says whether the keys bar's line says what kit did, in place of the keys.
func (m Model) Noticing() bool { return m.note.Text != "" }

// quit ends the screen, and what kit still runs for it.
func (m Model) quit() tea.Cmd {
	m.cancel()
	return tea.Quit
}

// Update takes a message, as update does; then what kit did, new, goes after a while, unless a key took it first.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if f, ok := msg.(shell.FadedMsg); ok {
		m.note = m.note.Faded(f)
		return m, nil
	}
	was := m.note
	next, cmd := m.update(msg)
	n := next.(Model)
	if fade := n.note.Fading(was, fades); fade != nil {
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
		m.settings = msg.Settings
		m.sheet.Settings(msg.Settings.FoldShared, msg.Settings.SharedEmoji)
	case shell.WorkspaceMsg:
		m.books = m.books.Told(msg)
	case listedMsg:
		if msg.gen != m.gen {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.refusal = refusalOf(msg.err)
			return m, nil
		}
		m.refusal, m.list = "", msg.list
		m.sheet.SetRecords(records(msg.list))
	case actedMsg:
		return m.acted(msg)
	case tea.KeyPressMsg:
		m.note.Text = ""
		return m.key(msg)
	case tea.MouseClickMsg:
		return m.click(msg)
	case tea.MouseWheelMsg:
		// the wheel scrolls the rows, as on every table, but not under an open menu or dialog
		if !m.appMenu.Open && !m.books.Open && !m.prompt.open {
			m.sheet.Wheel(msg)
		}
	}
	return m, nil
}

// refusalOf gives kit's line for an error, or that it failed.
func refusalOf(err error) string {
	if line := strings.TrimSpace(err.Error()); line != "" {
		return strings.TrimPrefix(strings.TrimPrefix(line, Failed), " ")
	}
	return "kit workspaces failed"
}

// records gives the groups as the table's records, one a group, keyed by its name.
func records(list kit.WorkspaceList) []table.Record {
	out := make([]table.Record, 0, len(list.Workspaces))
	for _, g := range list.Workspaces {
		favorite := table.Value{}
		if g.Favorite {
			favorite = table.One(Star)
		}
		repos := table.Value{}
		if len(g.Repos) > 0 {
			repos = table.Value{Raw: g.Repos}
		}
		out = append(out, table.Record{Row: grid.Row{Key: g.Name}, Values: map[string]table.Value{
			"favorite": favorite, "name": table.One(g.Name), "members": table.One(strconv.Itoa(g.Members)),
			"repos": repos}})
	}
	return out
}

// place puts the table under the title line, over the keys bar.
func (m *Model) place() { m.sheet.Place(2, max(1, m.height-3), m.width, m.height) }

// current gives the group under the cursor, and whether there is one.
func (m Model) current() (kit.ListedGroup, bool) {
	rows, t := m.sheet.Rows(), m.sheet.Table()
	if !m.sheet.InRows() || t.Cursor < 0 || t.Cursor >= len(rows) {
		return kit.ListedGroup{}, false
	}
	key := rows[t.Cursor].Key
	k := slices.IndexFunc(m.list.Workspaces, func(g kit.ListedGroup) bool { return g.Name == key })
	if k < 0 {
		return kit.ListedGroup{}, false
	}
	return m.list.Workspaces[k], true
}

// actions gives the app's menu's own lines, in its order, each with its key: New Workspace… first, then those on the
// group under the cursor, Favorite or Unfavorite as it is one or not.
func (m Model) actions() []shell.Item {
	star := shell.Item{Glyph: Star, Name: Favorite, Keys: "F"}
	if g, ok := m.current(); ok && g.Favorite {
		star = shell.Item{Glyph: NoStar, Name: Disfavor, Keys: "F"}
	}
	return []shell.Item{
		{Glyph: Write, Name: New, Keys: "N"},
		{Glyph: Go, Name: SwitchTo, Keys: "⏎"},
		{Glyph: Letter, Name: Rename, Keys: "R"},
		{Glyph: Plus, Name: Add, Keys: "A"},
		{Glyph: Octo, Name: Owner, Keys: "O"},
		{Glyph: Shell, Name: Command, Keys: "C"},
		{Glyph: Minus, Name: Remove, Keys: "X"},
		star,
		{Glyph: Trash, Name: DeleteOne, Keys: "D"},
	}
}

// menu gives the app's menu with the screen's own lines, those on a group dim while no group is under the cursor, or
// kit runs an action.
func (m Model) menu() shell.Menu {
	menu := m.appMenu
	menu.Own, menu.Dim = m.actions(), nil
	_, on := m.current()
	for _, a := range menu.Own {
		if (a.Name != New && !on) || m.prompt.busy {
			menu.Dim = append(menu.Dim, a.Name)
		}
	}
	return menu
}

// byKey gives the action a key does with the menu closed, "" for none.
func (m Model) byKey(press string) string {
	if press == "enter" {
		return SwitchTo
	}
	for _, a := range m.actions() {
		if shell.Press(a.Keys) == press && press != "" {
			return a.Name
		}
	}
	return ""
}

// owner is a GitHub owner's name, as gh repo list takes one.
var owner = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// do does an action: one that wants an answer, or cannot be undone, opens its prompt; the others run kit at once.
func (m Model) do(action string) (Model, tea.Cmd) {
	if slices.Contains(m.menu().Dim, action) {
		return m, nil
	}
	g, _ := m.current()
	edit := func(form string) func(string) ([]string, bool) {
		return func(typed string) ([]string, bool) {
			fields := strings.Fields(typed)
			return append([]string{"workspaces", form + "=" + g.Name}, fields...), len(fields) > 0
		}
	}
	p := prompt{open: true, typing: true}
	switch action {
	case New:
		p.title, p.does = "New workspace", "Create"
		p.says = "a name, then its repositories: host/owner/repo, a git URL, a folder, or a pattern such as " +
			"github.com/owner/tool-*"
		p.words = func(typed string) ([]string, bool) {
			fields := strings.Fields(typed)
			if len(fields) == 0 {
				return nil, false
			}
			return append([]string{"workspaces", "--new=" + fields[0]}, fields[1:]...), true
		}
	case Rename:
		p.title, p.does, p.typed = "Rename 📚 "+g.Name, "Rename", g.Name
		p.says = "its new name: lowercase letters, digits, and . _ -"
		p.words = func(typed string) ([]string, bool) {
			return []string{"workspaces", "--rename=" + g.Name, typed}, typed != "" && typed != g.Name &&
				!strings.ContainsAny(typed, " \t")
		}
	case Add:
		p.title, p.does, p.words = "Add to 📚 "+g.Name, "Add", edit("--add")
		p.says = "repositories: host/owner/repo, a git URL, a folder, or a pattern, which matches the checkouts " +
			"under the roots"
	case Remove:
		p.title, p.does, p.words = "Remove from 📚 "+g.Name, "Remove", edit("--remove")
		p.says = "members, or a pattern, which matches the group's members"
	case Owner:
		p.title, p.does = "Add a GitHub owner's repositories to 📚 "+g.Name, "Add"
		p.says = "an owner, then a pattern of names, as empowerite homebrew-*; gh lists the owner's repositories"
		p.words = func(typed string) ([]string, bool) {
			fields := strings.Fields(typed)
			if len(fields) == 0 || len(fields) > 2 {
				return nil, false
			}
			who := strings.TrimPrefix(fields[0], "github.com/")
			if !owner.MatchString(who) {
				return nil, false
			}
			words := []string{"workspaces", "--add=" + g.Name,
				"--from=gh repo list " + who + " --limit 1000 --json url --jq '.[].url'"}
			if len(fields) == 2 {
				words = append(words, "github.com/"+who+"/"+fields[1])
			}
			return words, true
		}
	case Command:
		p.title, p.does = "Add from a command to 📚 "+g.Name, "Add"
		p.says = "a command whose every line is a repository, as gh repo list <owner> --json url --jq '.[].url'"
		p.words = func(typed string) ([]string, bool) {
			return []string{"workspaces", "--add=" + g.Name, "--from=" + typed}, typed != ""
		}
	case DeleteOne:
		p.title, p.does, p.typing = "Delete 📚 "+g.Name, "Delete", false
		p.says = "takes the group, and its favorite, out of the file; its checkouts stay as they are"
		p.words = func(string) ([]string, bool) { return []string{"workspaces", "--delete=" + g.Name}, true }
	case Favorite, Disfavor:
		form := "--favorite="
		if action == Disfavor {
			form = "--unfavorite="
		}
		m.prompt = prompt{busy: true}
		return m, m.act([]string{"workspaces", form + g.Name}, "", "")
	case SwitchTo:
		return m, shell.Pick(g.Name)
	}
	m.prompt = p
	return m, nil
}

// asked runs what a prompt asked kit, when it asked: a rename followed from the group to its new name, a delete to none.
func (m Model) asked(words []string) tea.Cmd {
	if words == nil {
		return nil
	}
	g, _ := m.current()
	from, to := g.Name, g.Name
	switch {
	case strings.HasPrefix(words[1], "--rename="):
		to = words[2]
	case strings.HasPrefix(words[1], "--delete="):
		to = ""
	case strings.HasPrefix(words[1], "--new="):
		from, to = "", ""
	}
	return m.act(words, from, to)
}

// act gives the command that runs kit with the words, from and to the group it renames, or deletes.
func (m Model) act(words []string, from, to string) tea.Cmd {
	r, ctx := m.runner, m.ctx
	return func() tea.Msg {
		out, err := r.Act(ctx, words...)
		return actedMsg{out: out, err: err, from: from, to: to}
	}
}

// acted takes kit's answer to an action: its line, when it refused, in the prompt, which stays open, or in place of the
// keys; else what kit said in place of the keys, the prompt closed, the groups read again, and kittui asked to read
// the workspace again.
func (m Model) acted(msg actedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		line := refusalOf(msg.err)
		if m.prompt.open {
			m.prompt = m.prompt.refused(line)
		} else {
			m.prompt.busy = false
			m.note.Text, m.note.Done = Failed+" "+line, false
		}
		return m, nil
	}
	m.prompt = prompt{}
	m.note = m.note.Did(lastLine(msg.out))
	m.gen++
	m.loading = true
	changed := func() tea.Msg { return shell.ChangedMsg{From: msg.from, To: msg.to} }
	return m, tea.Batch(m.read(), changed)
}

// lastLine gives the last line kit printed that says something.
func lastLine(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// chose does what a choice from the app's menu asks: a screen, quit, or one of the screen's actions.
func (m Model) chose(c shell.Choice) (Model, tea.Cmd) {
	switch {
	case c.Quit:
		return m, m.quit()
	case c.Screen != "":
		return m, shell.Switch(c.Screen)
	case c.Own != "":
		return m.do(c.Own)
	}
	return m, nil
}

// key takes a key press: the quitting keys; an open prompt's, menu's, or workspace menu's; = and -; an action's key;
// then the table's.
func (m Model) key(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "ctrl+c":
		m.interrupted = true
		return m, m.quit()
	case shell.Quit:
		return m, m.quit()
	}
	switch {
	case m.prompt.open:
		var words []string
		m.prompt, words = m.prompt.key(k)
		return m, m.asked(words)
	case m.appMenu.Open:
		menu, c := m.menu().Key(k)
		m.appMenu.Dropdown = menu.Dropdown
		return m.chose(c)
	case m.books.Open:
		var picked string
		m.books, picked = m.books.Key(k)
		return m, shell.Pick(picked)
	case m.sheet.Layered():
		m.sheet.Key(k)
		return m, nil
	case m.search.Focused:
		s, did := m.search.Key(k)
		if did != shell.NotSearch {
			m.search = s
			m.sheet.Search(s.Query)
			if did == shell.Left {
				m.sheet.FocusRows()
			}
			return m, nil
		}
	}
	switch k.String() {
	case "/":
		m.search.Focused = true
		m.sheet.Blur()
		return m, nil
	case "=":
		m.appMenu.Dropdown = m.menu().Opened().Dropdown
		return m, nil
	case shell.Workspaces:
		m.books = m.books.Opened()
		return m, nil
	}
	if action := m.byKey(k.String()); action != "" && (m.sheet.InRows() || action == New) {
		return m.do(action)
	}
	if !m.sheet.Key(k) && !m.sheet.Focused() && k.String() == "down" {
		m.sheet.Focus()
	}
	return m, nil
}

// click takes a left click: on an open prompt, menu, or workspace menu; on = or the screen's name; on the workspace
// menu; then on the table.
func (m Model) click(c tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if c.Button != tea.MouseLeft {
		return m, nil
	}
	switch {
	case m.prompt.open:
		var words []string
		m.prompt, words = m.prompt.click(c, m.height, m.width)
		return m, m.asked(words)
	case m.appMenu.Open:
		menu, choice := m.menu().Click(c.X, c.Y, m.width)
		m.appMenu.Dropdown = menu.Dropdown
		return m.chose(choice)
	case m.books.Open:
		var picked string
		m.books, picked = m.books.Click(c.X, c.Y, m.booksAt(), m.width)
		return m, shell.Pick(picked)
	case shell.OnApp(c.X, c.Y, appLabel):
		m.appMenu.Dropdown = m.menu().Opened().Dropdown
		return m, nil
	case m.books.On(c.X, c.Y, m.booksAt()):
		m.books = m.books.Opened()
		return m, nil
	case c.Y == shell.Top && c.X >= m.menuBar().SearchAt(m.width):
		m.search.Focused = true
		m.sheet.Blur()
		return m, nil
	}
	m.search.Focused = false
	m.sheet.Click(c)
	return m, nil
}

// menuBar gives the menu bar: the kit menu's = before the screen's name, the workspace menu, and the crumb, while
// Manage… led here.
func (m Model) menuBar() shell.MenuBar {
	return shell.MenuBar{Screen: appLabel, Open: m.appMenu.Open, Workspace: m.books.Label(), Crumb: m.appMenu.Crumb,
		Search: shell.SearchLine(m.search.Query, m.search.Focused && !m.appMenu.Open)}
}

// booksAt gives the column the menu bar's workspace menu starts at.
func (m Model) booksAt() int { return m.menuBar().WorkspaceAt() }

// keys gives the path to the keyboard: an open prompt's or menu's alone; else the screen's keys, its actions' among
// them and ⌫ while a crumb shows, and the table's while it has the keyboard.
func (m Model) keys() shell.Path {
	switch {
	case m.prompt.open:
		return shell.Path{m.prompt.keys()}
	case m.appMenu.Open || m.books.Open:
		return shell.Path{shell.MenuKeys()}
	}
	if seg, layer := m.sheet.Keys(); layer {
		return shell.Path{seg}
	}
	own := []shell.Binding{shell.Bind("n", "new")}
	if _, on := m.current(); on {
		own = append(own, shell.Bind("⏎", "switch", "enter"), shell.Bind("r", "rename"), shell.Bind("a", "add"),
			shell.Bind("x", "remove"), shell.Bind("d", "delete"))
	}
	path := shell.Path{shell.WorkspaceScreenKeys(shell.CrumbOf(m.appMenu.Crumb, own...)...)}
	switch seg, _ := m.sheet.Keys(); {
	case m.search.Focused:
		search := shell.SearchKeys("workspaces", m.search.Query, false)
		search.Owns = true
		path = append(path, search)
	case m.sheet.Focused():
		path = append(path, seg)
	}
	return path
}

// View draws the screen: the menu bar and its rule, the table, or kit's line while it lists none, and the keys bar, or
// what kit did in its place; the table's layers, the menus, and the prompt over them.
func (m Model) View() tea.View {
	lines := []string{m.menuBar().Line(m.width), shell.RuleLine(m.width, "")}
	switch {
	case m.refusal != "":
		lines = append(lines, "   "+style.Refusal(Failed+" "+m.refusal))
	case m.loading && len(m.list.Workspaces) == 0:
		lines = append(lines, "   "+style.Off("reading the groups…"))
	case len(m.list.Workspaces) == 0:
		lines = append(lines, "   "+style.Off("no group yet; n makes one, as the menu's "+New+" does"))
	default:
		lines = append(lines, m.sheet.View()...)
		lines = m.sheet.Over(lines)
	}
	// the screen's lines reach the keys bar before anything drops down over them, so a menu over the few lines of no
	// group, or of reading them, is drawn whole
	lines = append(lines, "")
	for len(lines) < m.height-1 {
		lines = append(lines, "")
	}
	lines = m.menu().Overlay(lines, m.width)
	lines = m.books.Overlay(lines, m.booksAt(), m.width)
	lines = m.prompt.overlay(lines, m.width)
	bottom := m.keys().Line(m.width)
	if m.note.Text != "" {
		bottom = " " + m.note.Text
	}
	return shell.View(lines, m.height, bottom)
}
