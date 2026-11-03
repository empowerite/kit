package welcome

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// The steps, by the ids the checks name them with.
var (
	intro  = indexOf(known, "intro")
	basics = indexOf(known, "before-toolchains")
	chains = indexOf(known, "toolchains")
	setup  = indexOf(known, "setup")
	land   = indexOf(known, "lay-of-the-land")
	terms  = indexOf(known, "terms")
	yours  = indexOf(known, "make-it-yours")
)

func TestTheKnownStepsAreTheSevenInOrder(t *testing.T) {
	var ids, files []string
	for _, s := range known {
		ids, files = append(ids, s.id), append(files, s.file)
	}
	want := []string{"intro", "before-toolchains", "toolchains", "setup", "lay-of-the-land", "terms", "make-it-yours"}
	if !slices.Equal(ids, want) {
		t.Errorf("ids %q", ids)
	}
	for i, f := range files {
		if f != strconv.Itoa(i+1)+"-"+want[i]+".md" {
			t.Errorf("step %d's page is %q", i, f)
		}
	}
	if indexOf(known, "nothing") != -1 {
		t.Errorf("an unknown step has an index")
	}
}

func TestItOpensOnTheFirstStepWithAFailedCheckWithNowHighlighted(t *testing.T) {
	m, _ := loaded(t)
	if m.step != basics || !m.inPane || m.cursor != 1 || m.choice != now || !m.choosing() {
		t.Fatalf("step %d, in rows %v, cursor %d, choice %d", m.step, m.inPane, m.cursor, m.choice)
	}
	if m.keys() != choiceKeys {
		t.Errorf("keys %q", m.keys())
	}
	// with what kit needs in place, it opens on the toolchains
	m, _ = fake(t)
	if err := os.WriteFile(os.Getenv("FAKE_WELCOME_STATE"), []byte("jq\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if m = drive(m, testkit.Start(m, at120)...); m.step != chains || !m.inPane || m.cursor != 2 || !m.choosing() {
		t.Errorf("step %d, in rows %v, cursor %d", m.step, m.inPane, m.cursor)
	}
}

func TestItOpensOnTheIntroWhenEveryCheckPasses(t *testing.T) {
	m, _ := fake(t)
	state := []byte("jq\ntoolchain-project\ngh-auth\nkit-check\nbrew-path\n")
	if err := os.WriteFile(os.Getenv("FAKE_WELCOME_STATE"), state, 0o644); err != nil {
		t.Fatal(err)
	}
	m = drive(m, testkit.Start(m, at120)...)
	if m.step != intro || m.inPane {
		t.Errorf("step %d, in rows %v", m.step, m.inPane)
	}
	if got := view(m); strings.Contains(got, "setup 0") || strings.Contains(got, "basics 0") {
		t.Errorf("a step counts no failure:\n%s", got)
	}
}

func TestAPersonWhoMovedBeforeTheChecksAnsweredStaysThere(t *testing.T) {
	m, _ := fake(t)
	m = drive(m, append([]tea.Msg{at120, right, right, right}, testkit.Start(m)...)...)
	if m.step != setup || m.inPane {
		t.Errorf("step %d, in rows %v", m.step, m.inPane)
	}
}

func TestCheckingShowsWhileTheChecksRun(t *testing.T) {
	m, _ := fake(t)
	m = drive(m, at120)
	labels := strings.Split(view(m), "\n")[2]
	for _, want := range []string{"│ 👋 intro", "🧱 basics …", "🧰 toolchains …", "🔧 setup …", "🧭 lay of"} {
		if !strings.Contains(labels, want) {
			t.Errorf("no %q: %q", want, labels)
		}
	}
	if m = drive(m, right); !strings.Contains(view(m), "   checking…") {
		t.Errorf("no checking:\n%s", view(m))
	}
}

func TestLeftAndRightMoveAlongTheStepsAndStopAtTheEnds(t *testing.T) {
	m, _ := loaded(t, esc)
	if first := drive(m, left, left); first.step != intro {
		t.Errorf("left past intro: step %d", first.step)
	}
	for _, want := range []int{chains, setup, land, terms, yours, yours} {
		if m = drive(m, right); m.step != want {
			t.Fatalf("right: step %d, want %d", m.step, want)
		}
	}
	// the selected step's name is drawn in the accent, the rest are not
	tabs := strings.Split(m.View().Content, "\n")[2]
	if !strings.Contains(tabs, style.Mark("make it yours")) || strings.Contains(tabs, style.Mark("terms")) {
		t.Errorf("make it yours is not marked: %q", tabs)
	}
	// down enters the rows where the cursor was, once the bar has moved away and back
	m = drive(m, left, left, left, down, down)
	if m.step != setup || m.cursor != 4 || m.choosing() {
		t.Fatalf("step %d, cursor %d", m.step, m.cursor)
	}
	if m = drive(m, esc, right); m.step != land || m.inPane {
		t.Errorf("right on the bar: step %d, in the pane %v", m.step, m.inPane)
	}
	if m = drive(m, left, down); m.step != setup || m.cursor != 4 {
		t.Errorf("back on setup: step %d, cursor %d", m.step, m.cursor)
	}
}

func TestLeftAndRightOnAFailedRowMoveBetweenNowAndLater(t *testing.T) {
	m, r := loaded(t, right)
	if m.step != basics || m.choice != later {
		t.Fatalf("step %d, choice %d", m.step, m.choice)
	}
	if !strings.Contains(m.View().Content, style.Cursor(" later ")) {
		t.Errorf("later is not highlighted: %q", m.View().Content)
	}
	m = drive(m, left)
	if m.choice != now {
		t.Errorf("left: choice %d", m.choice)
	}
	// enter on later leaves the check, as l does, and moves to the step's other failed check, the show-only one
	m = drive(m, right, enter)
	if !m.deferred["jq"] || m.cursor != 5 || m.step != basics || len(r.calls) != 0 {
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
		if !m.checks[1].OK || m.cursor != 1 || m.step != basics || m.checking || m.choosing() {
			t.Errorf("%v: jq %+v, cursor %d, checking %v", key, m.checks[1], m.cursor, m.checking)
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

func TestTheToolchainsStepOffersNowAndLaterWithTheOthersBeside(t *testing.T) {
	m, r := loaded(t, esc, right, down)
	if m.step != chains || m.cursor != 2 || !m.choosing() {
		t.Fatalf("step %d, cursor %d", m.step, m.cursor)
	}
	got := view(m)
	for _, want := range []string{" ▸ ❌ several toolchains are usable for the project: mise, brew; choose one",
		"\n      kit toolchains choose mise\n", "\n      or: kit toolchains choose brew\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q:\n%s", want, got)
		}
	}
	m = drive(m, press('n'))
	if len(r.calls) != 1 || strings.Join(r.calls[0], " ") != "welcome --fix toolchain-project" || !m.checks[2].OK {
		t.Errorf("ran %q, check %+v", r.calls, m.checks[2])
	}
	if strings.Contains(strings.Split(view(m), "\n")[2], "toolchains 1") {
		t.Errorf("the toolchains still count a failure:\n%s", view(m))
	}
}

func TestLaterKeepsTheOtherToolchainsBesideTheFix(t *testing.T) {
	m, _ := loaded(t, esc, right, down, press('l'))
	if !m.deferred["toolchain-project"] {
		t.Fatalf("deferred %v", m.deferred)
	}
	want := "\n   • several toolchains are usable for the project: mise, brew; " +
		"choose one: kit toolchains choose mise\n" +
		"     or: kit toolchains choose brew\n"
	if got := view(m); !strings.Contains(got, want) {
		t.Errorf("no %q:\n%s", want, got)
	}
}

func TestNOnACheckThatStillFailsSaysSo(t *testing.T) {
	m, r := loaded(t, esc, right, right, down, press('n'))
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
	m, r := loaded(t, esc, right, right, down, press('l'))
	if !m.deferred["gh-auth"] || m.cursor != 4 || len(r.calls) != 0 {
		t.Fatalf("deferred %v, cursor %d, ran %q", m.deferred, m.cursor, r.calls)
	}
	got := view(m)
	for _, want := range []string{"   ❌ gh is not logged in to github.com · later", "   Left for later",
		"   • gh is not logged in to github.com: gh auth login --hostname github.com"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q:\n%s", want, got)
		}
	}
	// the next failed check has no fix: no choices, and n and l do nothing there
	if m.choosing() || m.keys() != rowKeys {
		t.Fatalf("cursor %d, keys %q", m.cursor, m.keys())
	}
	m = drive(m, press('n'), press('l'), enter)
	if len(r.calls) != 0 || m.deferred["kit-check"] {
		t.Errorf("n and l acted on a check without a fix: %q, %v", r.calls, m.deferred)
	}
	// now on a check left for later takes it off the list
	m = drive(m, up, press('n'))
	if m.deferred["gh-auth"] || strings.Contains(view(m), "Left for later") {
		t.Errorf("gh-auth is still left:\n%s", view(m))
	}
	// a step lists only its own checks left for later
	m = drive(m, press('l'), esc, left, left, down, press('l'))
	if !m.deferred["jq"] || strings.Contains(view(m), "• gh is not logged in") {
		t.Errorf("another step's check is listed:\n%s", view(m))
	}
}

func TestAFailedCheckWithoutAFixShowsWhereToReadMore(t *testing.T) {
	m, _ := loaded(t, esc, right, right, down, down)
	got := view(m)
	if m.cursor != 4 || !strings.Contains(got, " ▸ ❌ kit check finds 2 differences from the release\n") ||
		!strings.Contains(got, "\n      read: kit help check\n") {
		t.Errorf("cursor %d:\n%s", m.cursor, got)
	}
	if !strings.Contains(m.View().Content, style.Off("read: kit help check")) {
		t.Errorf("more is not dim")
	}
}

func TestEscGivesTheBarTheKeyboardAndNeverQuits(t *testing.T) {
	m, _ := loaded(t)
	next, cmd := m.Update(esc)
	m = next.(Model)
	if cmd != nil || m.inPane || m.keys() != stepKeys {
		t.Fatalf("esc in the rows: in the pane %v, keys %q", m.inPane, m.keys())
	}
	if next, cmd = m.Update(esc); cmd != nil || next.(Model).inPane {
		t.Errorf("esc on the bar did something")
	}
	// a page takes the keyboard on down, and esc gives it back, the page still scrolled
	m = drive(m, left, tea.WindowSizeMsg{Width: 120, Height: 12}, press('6'))
	if m.step != terms || m.keys() != pageKeys {
		t.Fatalf("step %d, keys %q", m.step, m.keys())
	}
	if m = drive(m, down); !m.inPane || m.keys() != scrollKeys || m.offsets[terms] != 0 {
		t.Fatalf("down on a page's bar: in the pane %v, keys %q, offset %d", m.inPane, m.keys(), m.offsets[terms])
	}
	if m = drive(m, down, esc); m.inPane || m.offsets[terms] != 1 {
		t.Errorf("esc in a page: in the pane %v, offset %d", m.inPane, m.offsets[terms])
	}
	// up at the page's top gives the bar the keyboard
	if m = drive(m, down, up); !m.inPane || m.offsets[terms] != 0 {
		t.Fatalf("up in a page: in the pane %v, offset %d", m.inPane, m.offsets[terms])
	}
	if m = drive(m, up); m.inPane {
		t.Errorf("up at the page's top kept the keyboard in the page")
	}
}

func TestAltOrSuperEscQuits(t *testing.T) {
	for _, mod := range []tea.KeyMod{tea.ModAlt, tea.ModSuper} {
		m, _ := loaded(t)
		next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape, Mod: mod})
		if cmd == nil || cmd() != tea.Quit() || next.(Model).Interrupted() {
			t.Errorf("%v esc did not quit", mod)
		}
	}
}

func TestLeftAndRightMoveStepsOnlyFromTheBar(t *testing.T) {
	m, _ := loaded(t)
	// the screen opens in basics' rows, on a check without choices
	m = drive(m, down, down, down)
	if m.choosing() {
		t.Fatalf("cursor %d offers choices", m.cursor)
	}
	if m = drive(m, right, left); m.step != basics || !m.inPane {
		t.Errorf("left and right in the rows: step %d, in the pane %v", m.step, m.inPane)
	}
	if m = drive(m, esc, right); m.step != chains {
		t.Errorf("right on the bar: step %d", m.step)
	}
}

// A click on a check's line puts the cursor on that check, and the pane has the keyboard; a click on a page gives the
// page the keyboard; a click on a line that is no check's does nothing.
func TestAClickOnACheckSelectsIt(t *testing.T) {
	click := func(y int) tea.Msg { return tea.MouseClickMsg{X: 6, Y: y, Button: tea.MouseLeft} }
	m, _ := loaded(t, esc)
	lines := m.stepLines()
	at := slices.IndexFunc(lines, func(l line) bool { return l.row >= 0 && l.row != m.cursor })
	want := lines[at].row
	// the screen line the check is drawn on, as the person sees it
	drawn := strings.Split(view(m), "\n")
	if y := slices.Index(drawn, ansi.Strip(lines[at].text)); y != paneTop+at {
		t.Fatalf("check %d is drawn on line %d, not %d", want, y, paneTop+at)
	}
	if m = drive(m, click(paneTop+at)); !m.inPane || m.cursor != want {
		t.Errorf("a click on check %d: in the pane %v, cursor %d", want, m.inPane, m.cursor)
	}
	if m = drive(m, click(paneTop)); m.cursor != want {
		t.Errorf("a click on the page's title moved the cursor to %d", m.cursor)
	}
	if m = drive(m, esc, left, click(paneTop+2)); m.step != intro || !m.inPane || m.keys() != scrollKeys {
		t.Errorf("a click on a page: step %d, in the pane %v", m.step, m.inPane)
	}
	if m = drive(m, esc, click(0)); m.inPane {
		t.Errorf("a click on the title gave the pane the keyboard")
	}
}

func TestANumberOrAClickSelectsAStepFromAnywhere(t *testing.T) {
	m, _ := loaded(t)
	if m = drive(m, tea.KeyPressMsg{Code: '6', Text: "6"}); m.step != terms || m.inPane {
		t.Errorf("6 in the rows: step %d, in the pane %v", m.step, m.inPane)
	}
	if m = drive(m, tea.KeyPressMsg{Code: tea.KeyKp1}); m.step != intro {
		t.Errorf("keypad 1: step %d", m.step)
	}
	bar := []rune(strings.Split(view(m), "\n")[1])
	x := 0
	for _, r := range bar {
		if r == '➂' {
			break
		}
		x += ansi.StringWidth(string(r))
	}
	if m = drive(m, tea.MouseClickMsg{X: x, Y: 1, Button: tea.MouseLeft}); m.step != chains {
		t.Errorf("a click on ➂: step %d\n%s", m.step, view(m))
	}
	if m.View().MouseMode != tea.MouseModeCellMotion {
		t.Error("the mouse is off")
	}
}

func TestCtrlCQuitsAsInterrupted(t *testing.T) {
	m, _ := loaded(t)
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil || cmd() != tea.Quit() || !next.(Model).Interrupted() {
		t.Errorf("ctrl-c did not quit as interrupted")
	}
}

func TestUpAndDownMoveOverTheStepsRowsAndLeaveThemAtTheTop(t *testing.T) {
	m, _ := loaded(t, esc)
	if m = drive(m, down); !m.inPane || m.cursor != 1 {
		t.Fatalf("down from the line: in rows %v, cursor %d", m.inPane, m.cursor)
	}
	if m = drive(m, down, down, down); m.cursor != 5 {
		t.Errorf("down past the step's last row: cursor %d", m.cursor)
	}
	if m = drive(m, up, up); m.cursor != 0 {
		t.Errorf("up: cursor %d", m.cursor)
	}
	if m = drive(m, pgup); !m.inPane || m.cursor != 0 {
		t.Errorf("page up past the first row left the rows: in the pane %v, cursor %d", m.inPane, m.cursor)
	}
	if m = drive(m, up, pgdown, pgup); m.inPane {
		t.Errorf("a page on the bar entered the rows")
	}
	if m = drive(m, enter); !m.inPane {
		t.Errorf("enter on the line did not enter the rows")
	}
	// a step entered for the first time starts on its first row
	if m = drive(m, esc, right, right, down); m.step != setup || m.cursor != 3 {
		t.Errorf("setup: step %d, cursor %d", m.step, m.cursor)
	}
}

func TestARefusalOfTheChecksShowsInRed(t *testing.T) {
	t.Setenv("FAKE_WELCOME_REFUSE", "1")
	m, _ := loaded(t)
	want := "❌ kit welcome refuses for the test; run: kit help welcome"
	if m.step != basics || !strings.Contains(view(m), "   "+want) || !strings.Contains(view(m), "🧱 basics !") ||
		!strings.Contains(view(m), "🔧 setup !") {
		t.Errorf("step %d:\n%s", m.step, view(m))
	}
	if !strings.Contains(m.View().Content, style.Refusal(want)) {
		t.Errorf("the refusal is not red: %q", m.View().Content)
	}
	if m = drive(m, left); strings.Contains(view(m), want) {
		t.Errorf("the intro shows the refusal:\n%s", view(m))
	}
}

func TestAPageScrollsWithWhatIsHiddenCounted(t *testing.T) {
	m, _ := loaded(t, tea.WindowSizeMsg{Width: 120, Height: 12}, esc, press('6'))
	if m.step != terms {
		t.Fatal(m.step)
	}
	all := len(m.pageLines(terms))
	pane := func() []string { return strings.Split(view(m), "\n")[4:10] }
	if got := pane(); got[0] != "   Terms" || got[5] != "   … "+strconv.Itoa(all-5)+" more" {
		t.Fatalf("top:\n%s", strings.Join(got, "\n"))
	}
	// the first down gives the page the keyboard, the next scrolls it
	m = drive(m, down, down)
	if got := pane(); got[0] != "   … 1 above" || got[5] != "   … "+strconv.Itoa(all-5)+" more" {
		t.Errorf("one down:\n%s", strings.Join(got, "\n"))
	}
	m = drive(m, pgdown, pgdown, pgdown, pgdown)
	if got := pane(); !strings.HasPrefix(got[0], "   … ") || !strings.Contains(got[5], "kit flow --stages") {
		t.Errorf("the end:\n%s", strings.Join(got, "\n"))
	}
	if m = drive(m, pgup, pgup, pgup, pgup, up); m.offsets[terms] != 0 {
		t.Errorf("back to the top: offset %d", m.offsets[terms])
	}
	// each step keeps its own place
	m = drive(m, down, pgdown, esc, left, right)
	if m.offsets[terms] == 0 {
		t.Errorf("the place was lost")
	}
}

func TestTheCursorsRowStaysInViewOnAShortScreen(t *testing.T) {
	m, _ := loaded(t, tea.WindowSizeMsg{Width: 120, Height: 10}, esc, right, right, down, down)
	if got := view(m); !strings.Contains(got, " ▸ ❌ kit check finds") || !strings.Contains(got, "above") {
		t.Errorf("the cursor's row is hidden:\n%s", got)
	}
}

// Leaving the screen, with alt-esc or ctrl-c, ends the checks still running, and what they started with them.
func TestLeavingTheScreenEndsTheChecksStillRunning(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{{Code: tea.KeyEscape, Mod: tea.ModAlt}, {Code: 'c', Mod: tea.ModCtrl}} {
		t.Setenv("FAKE_WELCOME_SLOW", "1")
		m, _ := fake(t)
		answers := make(chan tea.Msg, 1)
		check := m.check()
		go func() { answers <- check() }()
		time.Sleep(200 * time.Millisecond)
		m = drive(m, key)
		select {
		case msg := <-answers:
			if err := msg.(checksMsg).err; !errors.Is(err, context.Canceled) {
				t.Errorf("%s: the checks ended with %v, not the quit", key, err)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("%s: the checks outlived the screen", key)
		}
	}
}

// On every step, on its line and on each of its rows, the keys bar names n and Enter only where they act, on a row that
// offers now or later, and l there and on a fix kit only shows.
func TestTheKeysBarNamesOnlyTheKeysThatActThere(t *testing.T) {
	m, _ := loaded(t)
	for range len(m.tabs) {
		for k := 0; k < 30; k++ {
			bar := m.keys()
			for _, c := range []struct {
				key  string
				acts bool
			}{
				{"n now", m.choosing()},
				{"⏎ choose", m.choosing()},
				{"l later", m.choosing() || m.showOnly()},
			} {
				if strings.Contains(bar, c.key) != c.acts {
					t.Errorf("step %d, in rows %v, cursor %d: %q names %s: %v", m.step, m.inPane, m.cursor, bar, c.key,
						!c.acts)
				}
			}
			before := m.cursor
			if m = drive(m, down); m.cursor == before && k > 0 {
				break
			}
		}
		m = drive(m, esc, right)
	}
}

// Every height from a line to a dozen, on the line, in the rows, scrolled, and on a page: the screen draws as many lines
// as it has, the keys bar the last, and never panics.
func TestTheKeysBarIsTheLastLineAtEveryHeight(t *testing.T) {
	for _, msgs := range [][]tea.Msg{
		nil,
		{down},
		{down, down, down},
		{esc, right, right, down, down},
		{pgdown, pgdown, pgdown},
		{right, right, right, right, pgdown, pgdown},
	} {
		for _, width := range []int{120, 40} {
			check := func(m Model, height int, how string) {
				got := strings.Split(view(m), "\n")
				if bar := ansi.Strip(style.Bar(m.keys(), width)); len(got) != height || got[height-1] != bar {
					t.Errorf("after %v %s %d×%d: %d lines, the last %q", msgs, how, width, height, len(got),
						got[len(got)-1])
				}
			}
			// the keys pressed at a few heights from the start, then one screen shrunk through every height
			for _, height := range []int{12, 7, 3, 1} {
				m, _ := loaded(t, append([]tea.Msg{tea.WindowSizeMsg{Width: width, Height: height}}, msgs...)...)
				check(m, height, "at")
			}
			m, _ := loaded(t, msgs...)
			for height := 12; height >= 1; height-- {
				m = drive(m, tea.WindowSizeMsg{Width: width, Height: height})
				check(m, height, "shrunk to")
			}
		}
	}
}

func TestEveryLineFitsTheWidth(t *testing.T) {
	for _, width := range []int{120, 80, 40, 24, 12} {
		m, _ := loaded(t, tea.WindowSizeMsg{Width: width, Height: 30})
		for step := range len(m.tabs) {
			for _, l := range strings.Split(view(m), "\n") {
				if ansi.StringWidth(l) > width {
					t.Errorf("%d columns, step %d: %q is %d cells", width, step, l, ansi.StringWidth(l))
				}
			}
			m = drive(m, esc, right)
		}
	}
}

// At 130 columns every tab is whole; at 100 the bar scrolls, no name cut.
func TestEveryTabIsWholeAt130AndScrollsAt100(t *testing.T) {
	m, _ := loaded(t, tea.WindowSizeMsg{Width: 130, Height: 30})
	labels := strings.Split(view(m), "\n")[2]
	for _, s := range known {
		if !strings.Contains(labels, s.glyph+" "+s.name) {
			t.Errorf("at 130, no %q whole: %q", s.name, labels)
		}
	}
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	labels = strings.Split(view(m), "\n")[2]
	if !strings.Contains(labels, "│ 🧱 basics 2") || strings.Contains(labels, "…") || !strings.HasSuffix(labels, tabs.After) {
		t.Errorf("at 100: %q", labels)
	}
}

func TestTheGlyphsAreTwoCellsWithoutAVariationSelector(t *testing.T) {
	glyphs := []string{Repository, Passed, Failed, pageGlyph}
	for _, t := range known {
		glyphs = append(glyphs, t.glyph)
	}
	for _, g := range glyphs {
		if ansi.StringWidth(g) != 2 || strings.ContainsRune(g, '\ufe0f') {
			t.Errorf("%q is %d cells", g, ansi.StringWidth(g))
		}
	}
}

func TestTheRepositorysPagesWinAndTheKitPackagesAreTheFallback(t *testing.T) {
	nowhere := filepath.Join(t.TempDir(), "nowhere")
	r := repoWith(t, map[string]string{"6-terms.md": "# Our terms\n"})
	// the repository has pages: they are the steps, as kit welcome --text reads them, with no other filling in
	tabs := readPages(r)
	if len(tabs) != 1 || tabs[0].id != "terms" || tabs[0].page != "# Our terms\n" || tabs[0].missing != "" {
		t.Errorf("tabs %+v", tabs)
	}
	// with neither, the known steps say where each was looked for
	t.Setenv("FAKE_WELCOME_ROOT", nowhere)
	r = repoWith(t, nil)
	tabs = readPages(r)
	want := filepath.Join(nowhere, packageDir, pagesDir, "5-lay-of-the-land.md")
	if len(tabs) != len(known) || tabs[land].missing != want {
		t.Errorf("tabs %+v", tabs)
	}
	// the path is broken across lines, so all of it shows at any width
	for _, width := range []int{120, 40} {
		m := drive(New(r, ""), tea.WindowSizeMsg{Width: width, Height: 30}, pagesMsg{tabs: tabs}, right, right, right,
			right)
		lines := strings.Split(view(m), "\n")
		at := slices.Index(lines, "   no page at")
		path := ""
		for _, l := range lines[max(at+1, 0):] {
			if !strings.HasPrefix(l, "   ") || strings.TrimSpace(l) == "" {
				break
			}
			path += strings.TrimPrefix(l, "   ")
		}
		if at < 0 || path != want {
			t.Errorf("at %d columns, no missing line, or the path %q is not whole:\n%s", width, path, view(m))
		}
	}
}

func TestWithNoPagesInTheRepositoryTheKitPackagesAreTheSteps(t *testing.T) {
	r := repoWith(t, nil)
	tabs := readPages(r)
	var files []string
	for i, tab := range tabs {
		files = append(files, tab.file)
		if tab.id != known[i].id || tab.glyph != known[i].glyph || tab.checked != known[i].checked || tab.missing != "" {
			t.Errorf("step %d: %+v", i, tab)
		}
	}
	want := []string{"1-intro.md", "2-before-toolchains.md", "3-toolchains.md", "4-setup.md", "5-lay-of-the-land.md",
		"6-terms.md", "7-make-it-yours.md"}
	if !slices.Equal(files, want) || !strings.HasPrefix(tabs[chains].page, "# Toolchains") {
		t.Errorf("files %q", files)
	}
}

func TestAHousePackagesPagesAreStepsAndItsReplacementKeepsTheStep(t *testing.T) {
	m, _ := house(t)
	if len(m.tabs) != 8 {
		t.Fatalf("%d steps", len(m.tabs))
	}
	for i, k := range known {
		if m.tabs[i].id != k.id || m.tabs[i].glyph != k.glyph || m.tabs[i].checked != k.checked {
			t.Errorf("step %d: %+v", i, m.tabs[i])
		}
	}
	added := m.tabs[7]
	if added.id != "our-team" || added.glyph != pageGlyph || added.name != "our team" || added.checked {
		t.Errorf("the added step: %+v", added)
	}
	// the checks still open on the first failed one, under basics
	if m.step != basics || !m.choosing() {
		t.Errorf("step %d, choosing %v", m.step, m.choosing())
	}
	// the replaced page shows under terms; the added one last, a page with no checks
	if m = drive(m, esc, press('6')); m.step != terms || !strings.Contains(view(m), "\n   Our terms\n") {
		t.Errorf("step %d:\n%s", m.step, view(m))
	}
	m = drive(m, right, right)
	labels := strings.Split(view(m), "\n")[2]
	if m.step != 7 || m.hasRows() || m.keys() != pageKeys || !strings.Contains(labels, "│ 📄 our team") ||
		!strings.Contains(view(m), "\n   Our team\n") {
		t.Errorf("step %d:\n%s", m.step, view(m))
	}
	// at 120 columns the eight do not fit: the bar scrolls to the last, whole, no name cut
	if !strings.HasPrefix(labels, tabs.Before) || strings.Contains(labels, tabs.After) || strings.Contains(labels, "…") {
		t.Errorf("labels %q", labels)
	}
}

func TestThePagesAreInTheOrderKitWelcomeGives(t *testing.T) {
	files := map[string]string{}
	for _, f := range []string{"1-intro.md", "10-ten.md", "4-setup.md", "45-deploys.md", "5-lay-of-the-land.md",
		"2-before-toolchains.md", "notes.md", "04-also.md", ".hidden.md", "notes.txt"} {
		files[f] = "# " + f + "\n"
	}
	var got []string
	for _, tab := range readPages(repoWith(t, files)) {
		got = append(got, tab.file)
	}
	// by the number a name starts with, a tie by name, and a name with no number last, as kit welcome --text
	want := []string{"1-intro.md", "2-before-toolchains.md", "04-also.md", "4-setup.md", "5-lay-of-the-land.md",
		"10-ten.md", "45-deploys.md", "notes.md"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if idOf("45-deploys.md") != "deploys" || idOf("notes.md") != "notes" ||
		tabOf("9-our-on-call.md").name != "our on call" {
		t.Errorf("ids")
	}
}

func TestTheTabLineScrollsToKeepTheSelectedStepInView(t *testing.T) {
	files := map[string]string{}
	for _, k := range known {
		files[k.file] = "# " + k.name + "\n"
	}
	for i := range 20 {
		files["8-page-"+strconv.Itoa(i)+".md"] = "# page\n"
	}
	at80 := tea.WindowSizeMsg{Width: 80, Height: 30}
	m, _ := fake(t)
	m.runner = repoWith(t, files)
	m = drive(m, append(testkit.Start(m, at80), esc, left)...)
	if len(m.tabs) != 27 || m.step != 0 {
		t.Fatalf("%d steps, step %d", len(m.tabs), m.step)
	}
	for _, c := range []struct {
		keys          []tea.Msg
		label         string
		start, finish bool // whether each end is dotted
	}{
		{nil, "👋 intro │", false, true},
		{[]tea.Msg{right, right, right, right, right}, "📖 terms │", true, true},
		{slices.Repeat([]tea.Msg{right}, 21), "📄 page 9 │", true, false},
	} {
		m = drive(m, c.keys...)
		lines := strings.Split(view(m), "\n")
		line := lines[3]
		if !strings.Contains(lines[2], c.label) {
			t.Errorf("step %d is not in view:\n%s", m.step, lines[2])
		}
		if strings.HasPrefix(lines[2], tabs.Before) != c.start || strings.HasSuffix(lines[2], tabs.After) != c.finish {
			t.Errorf("step %d's ends:\n%s", m.step, line)
		}
		for _, l := range lines {
			if ansi.StringWidth(l) > 80 {
				t.Errorf("%q is %d cells", l, ansi.StringWidth(l))
			}
		}
	}
}

func TestEightStepsFitAtEightyColumnsWithTheLastInFull(t *testing.T) {
	m, _ := house(t, tea.WindowSizeMsg{Width: 80, Height: 30}, esc, press('8'))
	lines := strings.Split(view(m), "\n")
	if m.step != 7 || !strings.Contains(lines[2], "│ 📄 our team │") {
		t.Errorf("step %d:\n%s", m.step, lines[2])
	}
	if testing.Verbose() {
		t.Logf("\n%s", lines[2])
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

// showOnly opens the screen on the basics step's show-only check, brew-path, below jq.
func showOnly(t *testing.T, msgs ...tea.Msg) (Model, *ran) {
	t.Helper()
	return loaded(t, append([]tea.Msg{down}, msgs...)...)
}

func TestAShowOnlyFixDrawsItsLinesAndNoChoices(t *testing.T) {
	m, _ := showOnly(t)
	if m.checks[m.cursor].ID != "brew-path" || m.choosing() || !m.showOnly() || m.keys() != showKeys {
		t.Fatalf("cursor %d, choosing %v, keys %q", m.cursor, m.choosing(), m.keys())
	}
	got := view(m)
	for _, want := range []string{"brew is installed but not on PATH", `eval "$(/opt/homebrew/bin/brew shellenv)"`,
		"add it to your shell profile to keep it"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q:\n%s", want, got)
		}
	}
	// no choice of now or later on the row, and no way to run it; l still defers it, and the keys say so
	for _, no := range []string{" now ", "n now", "⏎ choose"} {
		if strings.Contains(got, no) {
			t.Errorf("show-only row offers %q:\n%s", no, got)
		}
	}
	if !strings.Contains(got, "l later") {
		t.Errorf("the keys do not say l defers the row:\n%s", got)
	}
}

func TestNAndEnterRunNothingOnAShowOnlyFix(t *testing.T) {
	m, r := showOnly(t)
	m.exec = func(*exec.Cmd, tea.ExecCallback) tea.Cmd {
		t.Error("a show-only fix handed the terminal over")
		return nil
	}
	m = drive(m, press('n'), enter, left, right)
	if len(r.calls) != 0 || len(m.deferred) != 0 {
		t.Errorf("ran %q, deferred %v", r.calls, m.deferred)
	}
}

func TestLaterLeavesAShowOnlyFixWithANote(t *testing.T) {
	m, r := showOnly(t, press('l'))
	if !m.deferred["brew-path"] || len(r.calls) != 0 {
		t.Fatalf("deferred %v, ran %q", m.deferred, r.calls)
	}
	want := "   • brew is installed but not on PATH: eval \"$(/opt/homebrew/bin/brew shellenv)\" · " +
		"kit does not run it\n"
	if got := view(m); !strings.Contains(got, want) {
		t.Errorf("no %q:\n%s", want, got)
	}
}

func TestACheckWithoutRunsIsRunnable(t *testing.T) {
	var c kit.Check
	if err := json.Unmarshal([]byte(`{"id": "x", "fix": "do it"}`), &c); err != nil || !c.Runs {
		t.Errorf("%+v, %v", c, err)
	}
	if err := json.Unmarshal([]byte(`{"id": "x", "fix": "do it", "runs": false}`), &c); err != nil || c.Runs {
		t.Errorf("%+v, %v", c, err)
	}
}
