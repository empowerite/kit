// The palette is a table of the verbs, internal/table's sheet: grouped by group and sorted by command, each row with its
// description's first line, an open row with the rest of it, the verb's page kit help gives, fetched as it opens. The
// search it opens with focus in narrows the rows; enter opens the verb under the cursor; the menus, the refusal, the
// searches remembered, and the mouse wheel are as on every screen.
package palette

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// open gives the palette over the kit package's fake kit, with the verbs it lists, its presets kept in a file of the
// test's own.
func open(t *testing.T) Model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	r := testkit.Runner(t, "kit")
	verbs, err := r.List()
	if err != nil {
		t.Fatal(err)
	}
	return New(verbs, r)
}

// listing gives the palette over the fake kit with the rows focused, as it opens.
func listing(t *testing.T) Model {
	t.Helper()
	m, _ := drive(open(t), tea.WindowSizeMsg{Width: 120, Height: 30})
	return m
}

// searching gives the palette over the fake kit with the search focused, as / leaves it from the rows it opens in.
func searching(t *testing.T) Model {
	t.Helper()
	m, _ := drive(open(t), slash)
	return m
}

// drive sends the messages to the palette from its start, as the program would, and gives the model and every
// message the commands produced; an OpenMsg is the palette's answer, so it is produced and not fed back.
func drive(m Model, msgs ...tea.Msg) (Model, []tea.Msg) {
	open := func(msg tea.Msg) bool { _, ok := msg.(OpenMsg); return ok }
	return testkit.Drive(m, open, testkit.Start(m, msgs...)...)
}

var (
	esc   = tea.KeyPressMsg{Code: tea.KeyEscape}
	enter = tea.KeyPressMsg{Code: tea.KeyEnter}
	slash = tea.KeyPressMsg{Code: '/', Text: "/"}
	ctrlU = tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}
	up    = tea.KeyPressMsg{Code: tea.KeyUp}
	down  = tea.KeyPressMsg{Code: tea.KeyDown}
	right = tea.KeyPressMsg{Code: tea.KeyRight}
	left  = tea.KeyPressMsg{Code: tea.KeyLeft}
)

// names gives the verbs the rows let through, in the order they stand.
func names(m Model) string {
	var out []string
	for _, v := range m.Visible() {
		out = append(out, v.Name)
	}
	return strings.Join(out, " ")
}

// screen gives the screen without its escapes.
func screen(m Model) string { return ansi.Strip(m.View().Content) }

// last gives the screen's last line, without its escapes: the keys bar.
func last(m Model) string {
	lines := strings.Split(screen(m), "\n")
	return lines[len(lines)-1]
}

// cursorOn gives the verb under the cursor, or none on a heading.
func cursorOn(m Model) string {
	verb, _ := m.highlighted()
	return verb
}

func TestEveryVerbButTuiGroupedByGroupAndSortedByCommand(t *testing.T) {
	m, _ := drive(open(t))
	// the groups stand in kit help's order: workflow, files, checks, …
	if names(m) != "commit submit context check version" {
		t.Errorf("verbs: %q", names(m))
	}
	// a group kit help does not name, and other, the group of a verb with none, follow its own, as kit lists them
	m = New([]kit.Verb{{Name: "zeta", Group: "zany"}, {Name: "bare"}, {Name: "commit", Group: "workflow"},
		{Name: "able", Group: "workflow"}, {Name: "tui", Group: "interfaces"}}, testkit.Runner(t, "kit"))
	if names(m) != "able commit zeta bare" {
		t.Errorf("verbs: %q", names(m))
	}
}

// The rows stand under a heading per group, its count beside it, under the header that names the command and the
// description; the description is the purpose's first line; the control line names the Default preset, which groups
// by group, the groups in kit help's order.
func TestTheTableHasAGroupACommandAndADescription(t *testing.T) {
	m := New([]kit.Verb{{Name: "commit", Purpose: "commits\nand says more", Group: "workflow"},
		{Name: "check", Purpose: "checks", Group: "checks"}}, testkit.Runner(t, "kit"))
	m, _ = drive(m, tea.WindowSizeMsg{Width: 100, Height: 20})
	v := screen(m)
	for _, want := range []string{"command", "description", "workflow", "checks", "commits", "Default", "group"} {
		if !strings.Contains(v, want) {
			t.Errorf("no %q on the screen:\n%s", want, v)
		}
	}
	if strings.Contains(v, "and says more") {
		t.Errorf("a closed row shows more than the description's first line:\n%s", v)
	}
	// the groups stand in kit help's order, workflow before checks, not by their names
	if strings.Index(v, "workflow") > strings.Index(v, "checks") {
		t.Errorf("the groups do not stand in kit help's order:\n%s", v)
	}
}

// A catalog that does not read is kittui's own fault, which panics, saying so.
func TestACatalogThatDoesNotReadPanics(t *testing.T) {
	defer func() {
		if p := recover(); p == nil || !strings.Contains(fmt.Sprint(p), "kit help's catalog does not read") {
			t.Errorf("a catalog that does not read: %v", p)
		}
	}()
	readCatalog([]byte("{"))
}

// Opening a row asks kit for the verb's page, once while it is on its way, and shows the rest of the description: the
// purpose's lines past its first, then the page past its first line, which says the purpose again. Closing it hides
// it, and opening it again asks for nothing.
func TestOpeningARowShowsTheRestOfTheDescription(t *testing.T) {
	m := New([]kit.Verb{{Name: "commit", Purpose: "commits\nand says more", Group: "workflow"}},
		testkit.Runner(t, "kit"))
	m, _ = drive(m, tea.WindowSizeMsg{Width: 100, Height: 20}, enter)
	if cursorOn(m) != "commit" || m.search.Focused {
		t.Fatalf("enter from the search: under the cursor %q, search focused %v", cursorOn(m), m.search.Focused)
	}
	next, ask := m.Update(right)
	m = next.(Model)
	if ask == nil || !m.asked["commit"] {
		t.Fatal("opening the row asked for no page")
	}
	if v := screen(m); !strings.Contains(v, "and says more") || !strings.Contains(v, "…") {
		t.Errorf("an open row on its page's way:\n%s", v)
	}
	if _, again := m.Update(down); again != nil && m.asked["commit"] && len(m.pages) > 0 {
		t.Error("a key while the page is on its way asked again")
	}
	m, _ = drive(m, ask())
	v := screen(m)
	if !strings.Contains(v, "Usage") || !strings.Contains(v, "Does it.") || strings.Contains(v, "a purpose for the test") {
		t.Errorf("the open row with its page:\n%s", v)
	}
	if m, _ = drive(m, left); strings.Contains(screen(m), "Usage") {
		t.Errorf("left left the page shown:\n%s", screen(m))
	}
	if next, ask := m.Update(right); ask != nil || !strings.Contains(screen(next.(Model)), "Usage") {
		t.Errorf("opened again: asked %v\n%s", ask != nil, screen(next.(Model)))
	}
}

// A page kit could not give shows kit's line in its place, in red; the row closed and opened again asks for it again.
func TestAPageKitCouldNotGiveIsAskedForAgain(t *testing.T) {
	m := listing(t)
	m.runner = kit.Runner{Kit: filepath.Join(t.TempDir(), "no-kit"), Dir: t.TempDir()}
	verb := cursorOn(m)
	m, _ = drive(m, right)
	line := m.failed[verb]
	if v := m.View().Content; !strings.HasPrefix(line, style.Excluded+" ") || m.asked[verb] ||
		!strings.Contains(v, style.Refusal(page(line, m.wrapped)[0])) {
		t.Errorf("a failed page does not show kit's line, %q, in red:\n%s", line, ansi.Strip(v))
	}
	m, _ = drive(m, left)
	if next, ask := m.Update(right); ask == nil || !next.(Model).asked[verb] {
		t.Error("a failed page is not asked for again")
	}
}

// A page is wrapped to the width the open row gives it, keeping each line's indent; no line passes the screen's edge,
// at any width, with a row open.
func TestEveryLineFitsTheWidthWithARowOpen(t *testing.T) {
	m, _ := drive(listing(t), right)
	for _, width := range []int{12, 20, 40, 60, 80, 120} {
		n, _ := drive(m, tea.WindowSizeMsg{Width: width, Height: 30})
		for _, l := range strings.Split(n.View().Content, "\n") {
			if ansi.StringWidth(l) > width {
				t.Errorf("a line of %d cells at %d columns: %q", ansi.StringWidth(l), width, ansi.Strip(l))
			}
		}
	}
}

func TestAPageWrapsByWhatShowsNotByItsEscapes(t *testing.T) {
	lines := page("\x1b[1mab\x1b[0m cd", 5)
	if len(lines) != 1 || lines[0] != "\x1b[1mab\x1b[0m cd" {
		t.Fatalf("page: %q", lines)
	}
	if lines := page("  one two three", 9); !slices.Equal(lines, []string{"  one two", "  three"}) {
		t.Errorf("an indented line: %q", lines)
	}
}

// Enter opens the verb under the cursor, after a search too; on a group's heading it opens nothing. A click on a row
// puts the cursor there and gives the rows the keyboard.
func TestEnterOpensTheVerbUnderTheCursor(t *testing.T) {
	m, produced := drive(searching(t), append(testkit.Typed("sub"), enter, enter)...)
	if got, ok := produced[len(produced)-1].(OpenMsg); !ok || got.Verb != "submit" {
		t.Errorf("sub, enter, enter: %v", produced)
	}
	m, _ = drive(listing(t), tea.KeyPressMsg{Code: '[', Text: "["})
	if _, produced = drive(m, enter); cursorOn(m) != "" || len(produced) != 0 {
		t.Errorf("enter on a heading: under the cursor %q, %v", cursorOn(m), produced)
	}
	m = listing(t)
	x, y := -1, -1
	for k, l := range strings.Split(screen(m), "\n") {
		if i := strings.Index(l, "version"); i >= 0 && strings.Contains(l, "prints") {
			x, y = ansi.StringWidth(l[:i]), k
		}
	}
	m, _ = drive(m, slash, testkit.Click(x, y))
	if cursorOn(m) != "version" || m.search.Focused {
		t.Errorf("a click on version: under the cursor %q, search focused %v", cursorOn(m), m.search.Focused)
	}
	if _, produced = drive(m, enter); len(produced) == 0 || produced[len(produced)-1] != tea.Msg(OpenMsg{Verb: "version"}) {
		t.Errorf("enter on version: %v", produced)
	}
}

// The keys bar is the last line: the rows' while they have the keyboard, as they do from the start, with ⏎ open on a
// verb and → show on a row that opens, and the search's while it has it.
func TestTheKeysBarNamesWhatHasTheKeyboard(t *testing.T) {
	m, _ := drive(open(t), tea.WindowSizeMsg{Width: 160, Height: 30})
	if m, _ = drive(m, slash); !strings.Contains(last(m), "type to search") {
		t.Errorf("the search's keys: %q", last(m))
	}
	m, _ = drive(m, enter)
	if bar := last(m); !strings.Contains(bar, "⏎ open") || !strings.Contains(bar, "→ show") ||
		!strings.Contains(bar, "esc search") || !strings.Contains(bar, "↑↓ move") {
		t.Errorf("the rows' keys: %q", bar)
	}
	lines := strings.Split(screen(m), "\n")
	if len(lines) != 30 {
		t.Errorf("%d lines on a screen 30 high", len(lines))
	}
}

func TestARefusalShowsRedUnderTheMenuBarUntilTheSearchChangesOrAVerbOpens(t *testing.T) {
	line := "kit commit refuses for the test; run: kit help commit"
	m, _ := drive(open(t), tea.WindowSizeMsg{Width: 120, Height: 30}, slash)
	m = m.Refuse(line)
	lines := strings.Split(screen(m), "\n")
	if strings.Trim(lines[1], shell.Rule) != "" || strings.TrimSpace(lines[2]) != style.Excluded+" "+line {
		t.Fatalf("no refusal under the menu bar's rule:\n%s", screen(m))
	}
	if !strings.Contains(m.View().Content, style.Refusal(style.Excluded+" "+line)) {
		t.Errorf("the refusal is not red:\n%s", m.View().Content)
	}
	if !strings.Contains(screen(m), "description") {
		t.Errorf("the table under a refusal has no header:\n%s", screen(m))
	}
	// typing, erasing, and clearing clear it
	if m, _ = drive(m, tea.KeyPressMsg{Code: 'c', Text: "c"}); strings.Contains(screen(m), line) {
		t.Errorf("typing left the refusal")
	}
	m = m.Refuse(line)
	if m, _ = drive(m, tea.KeyPressMsg{Code: tea.KeyBackspace}); strings.Contains(screen(m), line) {
		t.Errorf("backspace left the refusal")
	}
	m, _ = drive(m, tea.KeyPressMsg{Code: 'c', Text: "c"})
	m = m.Refuse(line)
	if m, _ = drive(m, ctrlU); strings.Contains(screen(m), line) || m.Cancelled() {
		t.Errorf("⌃U left the refusal")
	}
	// moving keeps it, and opening a verb clears it
	m, _ = drive(m, enter)
	m = m.Refuse(line)
	if m, _ = drive(m, down); !strings.Contains(screen(m), line) {
		t.Errorf("down cleared the refusal")
	}
	if m, _ = drive(m, tea.KeyPressMsg{Code: tea.KeyEnd}, enter); strings.Contains(screen(m), line) {
		t.Errorf("enter left the refusal")
	}
}

// Esc gives the rows the keyboard, what is typed kept, and never quits; from the rows, where the palette opens, it gives
// the search the keyboard; ctrl-q quits; ctrl-c interrupts; a letter is typed in the search, and so is /, while =
// drops the app's menu down.
func TestEscLeavesCtrlQQuitsCtrlCInterruptsAndALetterIsTyped(t *testing.T) {
	m, produced := drive(open(t), esc)
	if len(produced) != 0 || m.Cancelled() || m.Interrupted() || !m.search.Focused || m.sheet.Focused() {
		t.Errorf("esc from the start: %v, search focused %v", produced, m.search.Focused)
	}
	m, produced = drive(m, esc)
	if len(produced) != 0 || m.Cancelled() || m.Interrupted() || m.search.Focused || !m.sheet.InRows() {
		t.Errorf("esc with nothing typed: %v, search focused %v", produced, m.search.Focused)
	}
	m, produced = drive(open(t), tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
	if _, quit := produced[len(produced)-1].(tea.QuitMsg); !quit || !m.Cancelled() || m.Interrupted() {
		t.Errorf("ctrl-q: %v", produced)
	}
	m, produced = drive(open(t), tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if _, quit := produced[len(produced)-1].(tea.QuitMsg); !quit || !m.Interrupted() {
		t.Errorf("ctrl-c: %v", produced)
	}
	m, produced = drive(searching(t), testkit.Typed("q/=")...)
	if len(produced) != 0 || m.search.Query != "q/" || !m.appMenu.Open {
		t.Errorf("q/=: %v, search %q, menu %v", produced, m.search.Query, m.appMenu.Open)
	}
	m, _ = drive(m, esc, ctrlU)
	m, produced = drive(m, append(testkit.Typed("cont"), esc)...)
	if len(produced) != 0 || m.search.Query != "cont" || m.search.Focused || m.Cancelled() || names(m) != "context" {
		t.Errorf("esc with cont typed: %v, search %q, focused %v, %q", produced, m.search.Query, m.search.Focused,
			names(m))
	}
	// with the rows focused, a letter is not typed; esc gives the search the keyboard, and so does /
	if m, _ = drive(m, testkit.Typed("x")...); m.search.Query != "cont" {
		t.Errorf("x typed in the rows: %q", m.search.Query)
	}
	if m, produced = drive(m, esc); !m.search.Focused || m.Cancelled() || len(produced) != 0 || m.sheet.Focused() {
		t.Errorf("esc from the rows: search focused %v, rows %v", m.search.Focused, m.sheet.Focused())
	}
	m, _ = drive(m, enter)
	if m, _ = drive(m, slash); !m.search.Focused || m.search.Query != "cont" || m.sheet.Focused() {
		t.Errorf("/: focused %v, search %q", m.search.Focused, m.search.Query)
	}
}

// ctrl-q quits, as on every screen, with something typed, and with the app's menu open, as its Quit line says; ⌥esc,
// which a terminal with its default settings sends as esc, does not.
func TestCtrlQQuitsWhateverIsOpen(t *testing.T) {
	ctrlQ, optEsc := tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl}, tea.KeyPressMsg{Code: tea.KeyEscape, Mod: tea.ModAlt}
	for name, before := range map[string][]tea.Msg{"typed": testkit.Typed("q"), "menu": {esc, testkit.Typed("=")[0]}} {
		m, produced := drive(open(t), append(before, ctrlQ)...)
		if _, quit := produced[len(produced)-1].(tea.QuitMsg); !quit || !m.Cancelled() || m.Interrupted() {
			t.Errorf("%s, then ctrl-q: %v", name, produced)
		}
		if m, produced = drive(open(t), append(before, optEsc)...); m.Cancelled() || len(produced) > 0 &&
			produced[len(produced)-1] == tea.Msg(tea.QuitMsg{}) {
			t.Errorf("%s, then ⌥esc quit", name)
		}
	}
}

// At puts the cursor on a verb, its row open and its page asked for at the start, as kit help <verb> opens the
// palette; or on a group's first verb, as kit help <group> does; the rows have the keyboard. A name that is neither
// leaves the palette as it was.
func TestAtOpensOnAVerbsPageOrAGroupsFirstVerb(t *testing.T) {
	m := open(t).At("submit")
	if cursorOn(m) != "submit" || m.search.Focused || !m.sheet.InRows() {
		t.Fatalf("At(submit): under the cursor %q, search focused %v", cursorOn(m), m.search.Focused)
	}
	m, _ = drive(m)
	if v := screen(m); !strings.Contains(m.pages["submit"], "kit submit") || !strings.Contains(v, "Usage") {
		t.Errorf("the page of submit was not shown at the start:\n%s", v)
	}
	if m = open(t).At("workflow"); cursorOn(m) != "commit" || m.sheet.Table().Open["commit"] {
		t.Errorf("At(workflow): under the cursor %q", cursorOn(m))
	}
	if m = open(t).At("nonesuch"); m.search.Focused || !m.sheet.InRows() || cursorOn(m) != "commit" {
		t.Errorf("At an unknown name moved the palette")
	}
}

// = or a click on the = before the title drops the app's menu down: welcome asks kittui to show it in the palette's
// place, and quit ends the palette; while it is open, the keys bar is the menu's.
func TestEqualsDropsTheAppsMenuDownOverThePalette(t *testing.T) {
	m, _ := drive(open(t), tea.WindowSizeMsg{Width: 100, Height: 30}, slash)
	m, _ = testkit.Drive(m, nil, append(testkit.Typed("co"), esc, tea.KeyPressMsg{Code: '=', Text: "="})...)
	if v := screen(m); !m.appMenu.Open || !strings.Contains(v, "Welcome") ||
		!strings.HasSuffix(v, shell.Path{shell.MenuKeys()}.String()) || m.search.Query != "co" {
		t.Fatalf("= did not drop the app's menu down:\n%s", v)
	}
	if title := strings.Split(screen(m), "\n")[0]; !strings.HasSuffix(title, shell.Find+" co") {
		t.Errorf("the search under the menu: %q", title)
	}
	_, msgs := testkit.Drive(m, func(msg tea.Msg) bool { _, ok := msg.(shell.SwitchMsg); return ok },
		testkit.Typed("w")[0], testkit.Typed("w")[0], enter)
	if !slices.Contains(msgs, tea.Msg(shell.SwitchMsg{Screen: shell.Welcome})) {
		t.Errorf("welcome asked %v", msgs)
	}
	m, _ = testkit.Drive(m, nil, esc)
	m, _ = testkit.Drive(m, nil, testkit.Click(shell.At+ansi.StringWidth(shell.App)+len(appLabel), shell.Top),
		tea.MouseClickMsg{Button: tea.MouseRight})
	if !m.appMenu.Open {
		t.Fatal("a click on the screen's name did not drop the menu down")
	}
	if _, msgs := testkit.Drive(m, nil, testkit.Click(3, shell.Top+17)); !slices.Contains(msgs, tea.Msg(tea.QuitMsg{})) {
		t.Errorf("a click on quit: %v", msgs)
	}
	if m, _ = testkit.Drive(m, nil, testkit.Click(80, 20)); m.appMenu.Open {
		t.Error("a click off the menu did not close it")
	}
	if m.chose(shell.Choice{}) != nil {
		t.Error("no choice did something")
	}
}

// The palette opens with the rows focused, the cursor on the first verb, so the arrows move at once and a letter is not
// typed; after /, a letter typed narrows the rows at once to those whose text holds it, the command, the description,
// or the group; a click on the search gives it focus.
func TestThePaletteOpensWithTheRowsFocused(t *testing.T) {
	m, _ := drive(open(t), tea.WindowSizeMsg{Width: 100, Height: 30})
	title := strings.Split(screen(m), "\n")[0]
	if m.search.Focused || !m.sheet.InRows() || cursorOn(m) != "commit" || strings.HasSuffix(title, "▏") {
		t.Fatalf("the rows at the start: search focused %v, under the cursor %q, title %q", m.search.Focused,
			cursorOn(m), title)
	}
	if m, _ = drive(m, down); cursorOn(m) != "submit" {
		t.Errorf("down from the start: under the cursor %q", cursorOn(m))
	}
	if m, _ = drive(m, testkit.Typed("v")...); m.search.Query != "" || m.search.Focused {
		t.Errorf("v typed in the rows: %q", m.search.Query)
	}
	m, _ = drive(m, append([]tea.Msg{slash}, testkit.Typed("vers")...)...)
	if title = strings.Split(screen(m), "\n")[0]; names(m) != "version" || !strings.HasSuffix(title, shell.Find+" vers▏") {
		t.Errorf("vers: %q, title %q", names(m), title)
	}
	if m, _ = drive(m, ctrlU, testkit.Typed("files")[0]); names(m) != "commit submit context" {
		t.Errorf("f: %q", names(m))
	}
	m, _ = drive(m, testkit.Typed("iles")...)
	if names(m) != "context" {
		t.Errorf("files, a group's name: %q", names(m))
	}
	m, _ = drive(m, enter)
	if m, _ = testkit.Drive(m, nil, testkit.Click(m.searchAt()+1, shell.Top)); !m.search.Focused || m.sheet.Focused() {
		t.Error("a click on the search did not give it focus")
	}
}

// A search is remembered when it gives the rows focus, as on every screen; in the search, ctrl+r or a click on ▾
// drops the searches down, and choosing one types it.
func TestThePalettesSearchIsRememberedAndRecalled(t *testing.T) {
	m := open(t).Remembering(filepath.Join(t.TempDir(), "search-history.json"))
	m, _ = drive(m, append([]tea.Msg{slash}, append(testkit.Typed("co"), enter)...)...)
	if !slices.Equal(m.recent.Items, []string{"co"}) {
		t.Fatalf("remembered: %q", m.recent.Items)
	}
	m, _ = drive(m, slash, ctrlU, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if lines := strings.Split(screen(m), "\n"); !m.recent.Open || !strings.Contains(lines[0], shell.Recall) ||
		!strings.Contains(lines[len(lines)-1], "esc close") {
		t.Fatalf("ctrl+r: open %v\n%s", m.recent.Open, screen(m))
	}
	if m, _ = drive(m, enter); m.recent.Open || m.search.Query != "co" {
		t.Errorf("choosing co: %q", m.search.Query)
	}
	if !strings.Contains(last(m), "⌃R recent") {
		t.Errorf("the keys: %q", last(m))
	}
	m, _ = drive(m, testkit.Click(m.recallAt(), 0))
	_, left := m.recent.Box(m.recallAt(), m.width)
	if m, _ = drive(m, testkit.Click(left+3, 1)); m.recent.Open || m.search.Query != "co" {
		t.Errorf("a click on co: open %v, %q", m.recent.Open, m.search.Query)
	}
}

// The menu bar names the workspace kittui says, as every screen's does; - drops its menu down while the rows have
// focus, and is typed while the search has it; a click on it drops it down too.
func TestThePalettesMenuBarHasTheWorkspaceMenu(t *testing.T) {
	ws := shell.WorkspaceMsg{Workspace: kit.Workspace{Kind: "repository", Member: "github.com/o/r"}}
	m, _ := drive(open(t), tea.WindowSizeMsg{Width: 100, Height: 30}, ws, slash)
	if first := strings.Split(screen(m), "\n")[0]; !strings.HasPrefix(first,
		"  "+shell.App+" kit help · "+shell.Workspaces+" o/r ") {
		t.Errorf("the menu bar: %q", first)
	}
	if m, _ = drive(m, testkit.Typed("-")...); m.books.Open || m.search.Query != "-" {
		t.Errorf("- in the search: open %v, %q", m.books.Open, m.search.Query)
	}
	if m, _ = drive(m, ctrlU, esc, testkit.Typed("-")[0]); !m.books.Open || !strings.Contains(last(m), "esc close") {
		t.Errorf("- in the rows: open %v, keys %q", m.books.Open, last(m))
	}
	m, _ = drive(m, esc)
	if m, _ = drive(m, testkit.Click(m.booksAt(), shell.Top)); !m.books.Open {
		t.Error("a click on the workspace menu did not drop it down")
	}
	// a click on Open… asks kittui for the Open… dialog, and closes the menu
	x, y := at(m, "Open…", shell.Top+1)
	if m, produced := drive(m, testkit.Click(x, y)); m.books.Open ||
		!slices.Contains(produced, tea.Msg(shell.OpenMsg{})) {
		t.Errorf("a click on Open… at %d, %d: open %v, %v", x, y, m.books.Open, produced)
	}
}

// forty gives the palette over 40 verbs, v00 to v39, in three groups, at 100 columns and the height, the rows focused.
func forty(height int) Model {
	var verbs []kit.Verb
	for i := range 40 {
		group := "workflow"
		if i >= 27 {
			group = "checks"
		} else if i >= 14 {
			group = "files"
		}
		verbs = append(verbs, kit.Verb{Name: fmt.Sprintf("v%02d", i), Purpose: "a purpose", Group: group})
	}
	m, _ := drive(New(verbs, kit.Runner{}), tea.WindowSizeMsg{Width: 100, Height: height}, enter)
	return m
}

// turn gives a turn of the mouse wheel at the cell, down or up.
func turn(x, y int, down bool) tea.MouseWheelMsg {
	if down {
		return tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelDown}
	}
	return tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelUp}
}

// The wheel scrolls the rows, the table's own wheel, the cursor left where it is; a key brings the cursor's row back
// into the window; a turn on the menu bar, or with a menu open, scrolls nothing.
func TestTheWheelScrollsTheRowsAndLeavesTheCursor(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := forty(14)
	before, cursor := screen(m), cursorOn(m)
	m, _ = drive(m, turn(5, 8, true))
	if screen(m) == before || cursorOn(m) != cursor {
		t.Fatalf("a turn down: under the cursor %q, want %q\n%s", cursorOn(m), cursor, screen(m))
	}
	if strings.Contains(screen(m), cursor+" ") {
		t.Errorf("the cursor's row still shows after a turn:\n%s", screen(m))
	}
	if n, _ := drive(m, turn(5, 8, false)); screen(n) != before {
		t.Errorf("a turn back up:\n%s", screen(n))
	}
	if n, _ := drive(m, down); !strings.Contains(screen(n), cursorOn(n)) {
		t.Errorf("a key left the cursor's row out of the window:\n%s", screen(n))
	}
	if n, _ := drive(forty(14), turn(5, 0, true)); screen(n) != before {
		t.Errorf("a turn on the menu bar scrolled:\n%s", screen(n))
	}
	if n, _ := drive(forty(14), testkit.Typed("=")[0], turn(5, 8, true)); n.sheet.Table().Offset != 0 {
		t.Errorf("a turn under the menu scrolled:\n%s", screen(n))
	}
}

// What kittui's settings say reaches the table: whether a value every row shares folds.
func TestTheSettingsReachTheTable(t *testing.T) {
	m, _ := drive(open(t), shell.SettingsMsg{Settings: shell.Settings{FoldShared: false, SharedEmoji: true}})
	if p := m.sheet.Panel(); p.Fold || !p.Glyphs {
		t.Errorf("the settings did not reach the table: fold %v, glyphs %v", p.Fold, p.Glyphs)
	}
}

// Find gives the search the keyboard, as a form's search field asks, what is typed kept and the rows without it; the
// keys bar is the search's, and a letter typed narrows the rows further.
func TestFindGivesTheSearchTheKeyboard(t *testing.T) {
	m, _ := drive(searching(t), append(testkit.Typed("co"), enter)...)
	if m.search.Focused || !m.sheet.Focused() {
		t.Fatalf("co, enter: search focused %v, rows %v", m.search.Focused, m.sheet.Focused())
	}
	m = m.Find()
	if !m.search.Focused || m.sheet.Focused() || m.search.Query != "co" || !strings.Contains(last(m), "type to search") {
		t.Errorf("Find: search focused %v, rows %v, %q, keys %q", m.search.Focused, m.sheet.Focused(), m.search.Query,
			last(m))
	}
	if m, _ = drive(m, testkit.Typed("nt")...); m.search.Query != "cont" || names(m) != "context" {
		t.Errorf("nt typed after Find: %q, %q", m.search.Query, names(m))
	}
}

// A page that says nothing past its first line, which says the purpose again, says so, dim, where the rest would be.
func TestAPageThatSaysNoMoreSaysSo(t *testing.T) {
	m := New([]kit.Verb{{Name: "commit", Purpose: "commits", Group: "workflow"}}, testkit.Runner(t, "kit"))
	m, _ = drive(m, tea.WindowSizeMsg{Width: 100, Height: 20}, enter)
	next, _ := m.Update(right)
	m, _ = drive(next.(Model), pageMsg{verb: "commit", page: "kit commit - commits\n\n"})
	if v := m.View().Content; !strings.Contains(v, style.Off("kit help commit says no more")) || m.asked["commit"] {
		t.Errorf("a page with nothing past its first line:\n%s", ansi.Strip(v))
	}
}

// at gives the cell where the text first stands on the screen, past the line from, or -1, -1 where it does not.
func at(m Model, text string, from int) (x, y int) {
	for k, l := range strings.Split(screen(m), "\n") {
		if i := strings.Index(l, text); k >= from && i >= 0 {
			return ansi.StringWidth(l[:i]), k
		}
	}
	return -1, -1
}

// The control line's presets are the table's: a click on the selector drops them down, a layer whose keys are the
// keys bar and which takes every key and click; Save As… saves the table as a preset of the person's, which the keys
// bar says in their place until it fades, and a name it cannot take says why not, in red, until a key.
func TestThePresetsSayWhatTheyDidInPlaceOfTheKeys(t *testing.T) {
	fades = 0
	t.Cleanup(func() { fades = table.Fades })
	m := listing(t)
	x, y := at(m, "Default ▼", 1)
	if m, _ = drive(m, testkit.Click(x, y)); !m.sheet.Layered() || m.search.Focused {
		t.Fatalf("a click on the selector: layered %v, search focused %v\n%s", m.sheet.Layered(), m.search.Focused,
			screen(m))
	}
	if bar := last(m); strings.Contains(bar, "search") || !strings.Contains(bar, "esc") {
		t.Errorf("the presets' keys: %q", bar)
	}
	// s moves to Save As…, and enter takes it; enter with no name typed says why not, in red
	if m, _ = drive(m, testkit.Press("s"), enter, enter); m.sheet.Layered() ||
		!strings.Contains(last(m), "a preset needs a name") ||
		!strings.Contains(m.View().Content, style.Refusal(m.sheet.Notice().Text)) {
		t.Fatalf("Save As…, no name: layered %v, %q", m.sheet.Layered(), last(m))
	}
	// the name typed is the preset's, not the search's, and saving it says so until it fades
	m, _ = drive(m, testkit.Typed("mine")...)
	if strings.Contains(last(m), "needs a name") || m.search.Query != "" {
		t.Errorf("mine typed: search %q, %q", m.search.Query, last(m))
	}
	next, fade := m.Update(enter)
	if m = next.(Model); !strings.Contains(last(m), "saved the preset mine") || fade == nil {
		t.Fatalf("saved: %q", last(m))
	}
	if m, _ = drive(m, fade()); strings.Contains(last(m), "saved") || m.sheet.Table().ViewName != "mine" {
		t.Errorf("the notice did not fade: %q, the preset %q", last(m), m.sheet.Table().ViewName)
	}
	// a click on a preset the dropdown lists, which the layer takes, applies it
	x, y = at(m, "mine ▼", 1)
	m, _ = drive(m, testkit.Click(x, y))
	if x, y = at(m, "Default", y+1); y < 0 {
		t.Fatalf("no Default in the presets:\n%s", screen(m))
	}
	if m, _ = drive(m, testkit.Click(x, y)); m.sheet.Layered() || m.sheet.Table().ViewName != "Default" {
		t.Errorf("a click on Default: layered %v, %q\n%s", m.sheet.Layered(), m.sheet.Table().ViewName, screen(m))
	}
}
