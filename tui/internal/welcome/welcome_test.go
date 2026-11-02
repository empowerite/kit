package welcome

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/tui/internal/style"
	"github.com/empowerite/kit/tui/internal/testkit"
)

func TestItOpensOnTheFirstFailedChecksRowWithNowHighlighted(t *testing.T) {
	m, _ := loaded(t)
	if m.step != setup || !m.inRows || m.cursor != 2 || m.choice != now || !m.choosing() {
		t.Fatalf("step %d, in rows %v, cursor %d, choice %d", m.step, m.inRows, m.cursor, m.choice)
	}
	if m.keys() != choiceKeys {
		t.Errorf("keys %q", m.keys())
	}
}

func TestItOpensOnTheLayOfTheLandWhenEveryCheckPasses(t *testing.T) {
	m, _ := fake(t)
	if err := os.WriteFile(os.Getenv("FAKE_WELCOME_STATE"), []byte("jq\ngh-auth\nkit-check\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m = drive(m, testkit.Start(m, at120)...)
	if m.step != land || m.inRows {
		t.Errorf("step %d, in rows %v", m.step, m.inRows)
	}
	if strings.Contains(view(m), "setup 0") {
		t.Errorf("setup counts no failure:\n%s", view(m))
	}
}

func TestAPersonWhoMovedBeforeTheChecksAnsweredStaysThere(t *testing.T) {
	m, _ := fake(t)
	m = drive(m, append([]tea.Msg{at120, right, right}, testkit.Start(m)...)...)
	if m.step != terms || m.inRows {
		t.Errorf("step %d, in rows %v", m.step, m.inRows)
	}
}

func TestCheckingShowsWhileTheChecksRun(t *testing.T) {
	m, _ := fake(t)
	m = drive(m, at120)
	if got := view(m); !strings.Contains(got, "   checking…") || !strings.Contains(got, "└ 🔧 setup …") {
		t.Errorf("no checking:\n%s", got)
	}
}

func TestLeftAndRightMoveAlongTheStepsRoundTheEnds(t *testing.T) {
	m, _ := loaded(t, esc)
	for _, want := range []int{land, terms, yours, setup} {
		if m = drive(m, right); m.step != want {
			t.Fatalf("right: step %d, want %d", m.step, want)
		}
	}
	if m = drive(m, left); m.step != yours {
		t.Errorf("left from setup: step %d", m.step)
	}
	// the selected step's name is drawn in the accent, the rest are not
	tabs := strings.Split(m.View().Content, "\n")[2]
	if !strings.Contains(tabs, style.Mark("make it yours")) || strings.Contains(tabs, style.Mark("terms")) {
		t.Errorf("make it yours is not marked: %q", tabs)
	}
	// on a check's row with no choices, left and right move along the steps too; down enters the rows where the
	// cursor was
	m = drive(m, right, down, up)
	if m.cursor != 1 || m.choosing() {
		t.Fatalf("cursor %d", m.cursor)
	}
	if m = drive(m, right); m.step != land || m.inRows {
		t.Errorf("right on a row without choices: step %d, in rows %v", m.step, m.inRows)
	}
}

func TestLeftAndRightOnAFailedRowMoveBetweenNowAndLater(t *testing.T) {
	m, r := loaded(t, right)
	if m.step != setup || m.choice != later {
		t.Fatalf("step %d, choice %d", m.step, m.choice)
	}
	if !strings.Contains(m.View().Content, style.Cursor(" later ")) {
		t.Errorf("later is not highlighted: %q", m.View().Content)
	}
	m = drive(m, left)
	if m.choice != now {
		t.Errorf("left: choice %d", m.choice)
	}
	// enter on later leaves the check, as l does
	m = drive(m, right, enter)
	if !m.deferred["jq"] || m.cursor != 3 || len(r.calls) != 0 {
		t.Errorf("enter on later: deferred %v, cursor %d, ran %q", m.deferred, m.cursor, r.calls)
	}
}

func TestNRunsTheFixWithTheTerminalThenReadsTheChecksAgain(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_WELCOME_CALLS", record)
	for _, key := range []tea.Msg{press('n'), enter} {
		m, r := loaded(t, key)
		if len(r.calls) != 1 || strings.Join(r.calls[0], " ") != "welcome --fix jq" {
			t.Fatalf("%v ran %q", key, r.calls)
		}
		// the checks ran again, jq is fixed now, and the cursor stays on its row
		if !m.checks[2].OK || m.cursor != 2 || m.checking || m.choosing() {
			t.Errorf("%v: jq %+v, cursor %d, checking %v", key, m.checks[2], m.cursor, m.checking)
		}
		if !strings.Contains(view(m), " ▸ ✅ jq is not installed") || strings.Contains(view(m), "brew install jq") {
			t.Errorf("%v:\n%s", key, view(m))
		}
	}
	want := []string{"welcome --checks --json", "welcome --fix jq", "welcome --checks --json"}
	if got := calls(t, record); !slices.Equal(got, append(want, want...)) {
		t.Errorf("the calls: %q", got)
	}
}

func TestNOnACheckThatStillFailsSaysSo(t *testing.T) {
	m, r := loaded(t, down, press('n'))
	if len(r.calls) != 1 || strings.Join(r.calls[0], " ") != "welcome --fix gh-auth" {
		t.Fatalf("ran %q", r.calls)
	}
	if m.cursor != 3 || m.checks[3].OK || !strings.Contains(view(m), "the fix ran, and the check still fails") {
		t.Errorf("cursor %d:\n%s", m.cursor, view(m))
	}
}

func TestARefusedFixShowsInRed(t *testing.T) {
	t.Setenv("FAKE_WELCOME_REFUSE_FIX", "jq")
	m, _ := loaded(t, press('n'))
	want := "kit refused the fix; quit to read what kit welcome --fix jq printed"
	if !strings.Contains(m.View().Content, style.Refusal(want)) {
		t.Errorf("no refusal in red: %q", m.View().Content)
	}
	// a later fix that holds clears it
	t.Setenv("FAKE_WELCOME_REFUSE_FIX", "")
	if m = drive(m, press('n')); strings.Contains(view(m), "refused") {
		t.Errorf("the refusal stayed:\n%s", view(m))
	}
}

func TestLaterLeavesTheCheckAndMovesToTheNextFailedOne(t *testing.T) {
	m, r := loaded(t, press('l'))
	if !m.deferred["jq"] || m.cursor != 3 || len(r.calls) != 0 {
		t.Fatalf("deferred %v, cursor %d, ran %q", m.deferred, m.cursor, r.calls)
	}
	got := view(m)
	for _, want := range []string{"   ❌ jq is not installed · later", "   Left for later",
		"   • jq is not installed: brew install jq"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q:\n%s", want, got)
		}
	}
	// the next failed check has no fix: no choices, and n and l do nothing there
	m = drive(m, press('l'))
	if m.cursor != 4 || m.choosing() || m.keys() != rowKeys {
		t.Fatalf("cursor %d, keys %q", m.cursor, m.keys())
	}
	m = drive(m, press('n'), press('l'), enter)
	if len(r.calls) != 0 || m.deferred["kit-check"] {
		t.Errorf("n and l acted on a check without a fix: %q, %v", r.calls, m.deferred)
	}
	// now on a check left for later takes it off the list
	m = drive(m, up, up, press('n'))
	if m.deferred["jq"] || strings.Contains(view(m), "• jq") || !strings.Contains(view(m), "• gh is not logged in") {
		t.Errorf("jq is still left:\n%s", view(m))
	}
}

func TestAFailedCheckWithoutAFixShowsWhereToReadMore(t *testing.T) {
	m, _ := loaded(t, down, down)
	got := view(m)
	if m.cursor != 4 || !strings.Contains(got, " ▸ ❌ kit check finds 2 differences from the release\n") ||
		!strings.Contains(got, "\n      read: kit help check\n") {
		t.Errorf("cursor %d:\n%s", m.cursor, got)
	}
	if !strings.Contains(m.View().Content, style.Off("read: kit help check")) {
		t.Errorf("more is not dim")
	}
}

func TestEscGoesBackFromTheRowsThenQuits(t *testing.T) {
	m, _ := loaded(t)
	next, cmd := m.Update(esc)
	m = next.(Model)
	if cmd != nil || m.inRows || m.keys() != setupKeys {
		t.Fatalf("esc in the rows: in rows %v, keys %q", m.inRows, m.keys())
	}
	if _, cmd = m.Update(esc); cmd == nil || cmd() != tea.Quit() {
		t.Errorf("esc on the step line did not quit")
	}
	// a page has no rows: esc quits at once
	m = drive(m, right)
	if _, cmd = m.Update(esc); cmd == nil || cmd() != tea.Quit() || m.Interrupted() {
		t.Errorf("esc on a page did not quit")
	}
}

func TestCtrlCQuitsAsInterrupted(t *testing.T) {
	m, _ := loaded(t)
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil || cmd() != tea.Quit() || !next.(Model).Interrupted() {
		t.Errorf("ctrl-c did not quit as interrupted")
	}
}

func TestUpAndDownMoveOverTheRowsAndLeaveThemAtTheTop(t *testing.T) {
	m, _ := loaded(t, esc)
	if m = drive(m, down); !m.inRows || m.cursor != 2 {
		t.Fatalf("down from the line: in rows %v, cursor %d", m.inRows, m.cursor)
	}
	if m = drive(m, down, down, down); m.cursor != 4 {
		t.Errorf("down past the last row: cursor %d", m.cursor)
	}
	if m = drive(m, pgup); m.inRows {
		t.Errorf("page up past the first row stayed in the rows")
	}
	if m = drive(m, enter); !m.inRows {
		t.Errorf("enter on the line did not enter the rows")
	}
}

func TestARefusalOfTheChecksShowsInRed(t *testing.T) {
	t.Setenv("FAKE_WELCOME_REFUSE", "1")
	m, _ := loaded(t)
	want := "❌ kit welcome refuses for the test; run: kit help welcome"
	if m.step != setup || !strings.Contains(view(m), "   "+want) || !strings.Contains(view(m), "└ 🔧 setup !") {
		t.Errorf("step %d:\n%s", m.step, view(m))
	}
	if !strings.Contains(m.View().Content, style.Refusal(want)) {
		t.Errorf("the refusal is not red: %q", m.View().Content)
	}
}

func TestAPageScrollsWithWhatIsHiddenCounted(t *testing.T) {
	m, _ := loaded(t, tea.WindowSizeMsg{Width: 120, Height: 12}, esc, right, right)
	if m.step != terms {
		t.Fatal(m.step)
	}
	all := len(m.pageLines(terms))
	pane := func() []string { return strings.Split(view(m), "\n")[4:10] }
	if got := pane(); got[0] != "   Terms" || got[5] != "   … "+strconv.Itoa(all-5)+" more" {
		t.Fatalf("top:\n%s", strings.Join(got, "\n"))
	}
	m = drive(m, down)
	if got := pane(); got[0] != "   … 1 above" || got[5] != "   … "+strconv.Itoa(all-5)+" more" {
		t.Errorf("one down:\n%s", strings.Join(got, "\n"))
	}
	m = drive(m, pgdown, pgdown, pgdown, pgdown)
	if got := pane(); !strings.HasPrefix(got[0], "   … ") || !strings.Contains(got[5], "kit status --stages") {
		t.Errorf("the end:\n%s", strings.Join(got, "\n"))
	}
	if m = drive(m, pgup, pgup, pgup, pgup, up); m.offsets[terms] != 0 {
		t.Errorf("back to the top: offset %d", m.offsets[terms])
	}
	// each step keeps its own place
	m = drive(m, pgdown, left, right)
	if m.offsets[terms] == 0 {
		t.Errorf("the place was lost")
	}
}

func TestTheCursorsRowStaysInViewOnAShortScreen(t *testing.T) {
	m, _ := loaded(t, tea.WindowSizeMsg{Width: 120, Height: 10}, down, down)
	if got := view(m); !strings.Contains(got, " ▸ ❌ kit check finds") || !strings.Contains(got, "above") {
		t.Errorf("the cursor's row is hidden:\n%s", got)
	}
}

func TestEveryLineFitsTheWidth(t *testing.T) {
	for _, width := range []int{120, 80, 40, 24} {
		m, _ := loaded(t, tea.WindowSizeMsg{Width: width, Height: 30})
		for step := range steps {
			for _, l := range strings.Split(view(m), "\n") {
				if ansi.StringWidth(l) > width {
					t.Errorf("%d columns, step %d: %q is %d cells", width, step, l, ansi.StringWidth(l))
				}
			}
			m = drive(m, esc, right)
		}
	}
}

func TestTheGlyphsAreTwoCellsWithoutAVariationSelector(t *testing.T) {
	glyphs := []string{Repository, Passed, Failed}
	for _, t := range tabs {
		glyphs = append(glyphs, t.glyph)
	}
	for _, g := range glyphs {
		if ansi.StringWidth(g) != 2 || strings.ContainsRune(g, '\ufe0f') {
			t.Errorf("%q is %d cells", g, ansi.StringWidth(g))
		}
	}
}

func TestTheRepositorysPagesComeFirstAndTheKitPackagesFillIn(t *testing.T) {
	dir := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	pages := filepath.Join(dir, ".kit", "welcome")
	if err := os.MkdirAll(pages, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pages, "3-terms.md"), []byte("# Our terms\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := testkit.Runner(t, "welcome")
	r.Dir = dir
	got, missing := readPages(r)
	if got[terms] != "# Our terms\n" || !strings.HasPrefix(got[land], "# The lay of the land") ||
		missing != [steps]string{} {
		t.Errorf("pages %q, missing %q", got, missing)
	}
	// with neither, the page says where it was looked for
	t.Setenv("FAKE_WELCOME_ROOT", filepath.Join(dir, "nowhere"))
	_, missing = readPages(r)
	want := filepath.Join(dir, "nowhere", packageDir, pagesDir, "2-lay-of-the-land.md")
	if missing[terms] != "" || missing[land] != want {
		t.Errorf("missing %q", missing)
	}
	m := drive(New(r, ""), at120, pagesMsg{missing: missing}, right)
	if !strings.Contains(view(m), "   no page at "+missing[land]) {
		t.Errorf("no missing line:\n%s", view(m))
	}
}

func TestTheTitleNamesTheRepositoryWhenKnown(t *testing.T) {
	for url, want := range map[string]string{
		"https://github.com/empowerite/kit.git": "empowerite/kit",
		"git@github.com:empowerite/kit.git":     "empowerite/kit",
		"ssh://git@gitlab.example/a/b/c":        "b/c",
		"https://github.com/kit":                "",
		"nonsense":                              "",
	} {
		if got := repoOf(url); got != want {
			t.Errorf("repoOf(%q) = %q, want %q", url, got, want)
		}
	}
	if m, _ := fake(t); !strings.HasPrefix(view(m), " kit welcome · 📚 empowerite/kit\n") {
		t.Errorf("title:\n%s", view(m))
	}
	if got := view(New(testkit.Runner(t, "welcome"), "")); !strings.HasPrefix(got, " kit welcome\n") {
		t.Errorf("title without a repository:\n%s", got)
	}
}
