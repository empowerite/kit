package welcome

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/source"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/tabs"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// The steps, by the ids the checks name them with.
var (
	introduction = indexOf(known, "introduction")
	chains       = indexOf(known, "toolchains")
	setup        = indexOf(known, "setup")
	terms        = indexOf(known, "terms")
	custom       = indexOf(known, "customize")
)

func TestTheKnownStepsAreTheFiveInOrder(t *testing.T) {
	var ids, files []string
	for _, s := range known {
		ids, files = append(ids, s.id), append(files, s.file)
	}
	want := []string{"introduction", "toolchains", "setup", "customize", "terms"}
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

// The screen opens on the introduction, and stays there when the checks answer: the introduction's page ends with how
// many failed, and a link to each, in the order of their steps; each step with a failed check counts them in red on the
// bar.
func TestItStaysOnTheIntroductionAndItSaysWhatTheChecksFound(t *testing.T) {
	m, _ := loaded(t)
	if m.step != introduction || m.inPane || keysAt(m) != stepKeys {
		t.Fatalf("step %d, in the pane %v, keys %q", m.step, m.inPane, keysAt(m))
	}
	want := "\n   5 checks need your attention\n   " + strings.Repeat("─", 28) + "\n" +
		"   ❌ jq is not installed → 🧰 toolchains\n" +
		"   ❌ several toolchains are usable for the project: mise, brew; choose one → 🧰 toolchains\n" +
		"   ❌ brew is installed but not on PATH → 🧰 toolchains\n" +
		"   ❌ gh is not logged in to github.com → 🔧 setup\n" +
		"   ❌ kit check finds 2 differences from the release → 🔧 setup\n"
	if got := view(m); !strings.Contains(got, want) {
		t.Errorf("no %q:\n%s", want, got)
	}
	bar := strings.Split(m.View().Content, "\n")[2]
	for _, count := range []string{"3", "2"} {
		if !strings.Contains(bar, style.Loss(count)) {
			t.Errorf("no count %s in red: %q", count, bar)
		}
	}
	// the counts stay red while the pane has the keyboard and the other tabs are dim
	if m, _ = atChains(t); !strings.Contains(strings.Split(m.View().Content, "\n")[2], style.Loss("2")) {
		t.Errorf("setup's count is not red from the toolchains: %q", strings.Split(m.View().Content, "\n")[2])
	}
	// one failed check is one that needs you
	m, _ = fake(t)
	if err := os.WriteFile(os.Getenv("FAKE_WELCOME_STATE"), []byte("jq\ntoolchain-project\ngh-auth\nkit-check\n"),
		0o644); err != nil {
		t.Fatal(err)
	}
	m = drive(m, testkit.Start(m, at120)...)
	one := "\n   1 check needs your attention\n   " + strings.Repeat("─", 28) + "\n   ❌ brew is installed"
	if got := view(m); m.step != introduction || !strings.Contains(got, one) {
		t.Errorf("step %d:\n%s", m.step, got)
	}
}

// Down gives the introduction's links the keyboard, the cursor on the first; enter goes to its check, on its step, with
// the cursor on it, and back on the introduction the cursor is on the link it left from.
func TestEnterOnALinkGoesToItsCheckOnItsStep(t *testing.T) {
	m, r := loaded(t, down)
	if !m.inPane || m.checks[m.cursor].ID != "jq" || keysAt(m) != linkKeys || m.choosing() {
		t.Fatalf("in the pane %v, cursor %d, keys %q", m.inPane, m.cursor, keysAt(m))
	}
	if got := view(m); !strings.Contains(got, "\n ▸ ❌ jq is not installed → 🧰 toolchains\n") {
		t.Errorf("the cursor is not on jq's link:\n%s", got)
	}
	if m = drive(m, enter); m.step != chains || !m.inPane || m.cursor != 1 || m.choice != now || !m.choosing() ||
		keysAt(m) != choiceKeys || len(r.calls) != 0 {
		t.Fatalf("enter on jq's link: step %d, in the pane %v, cursor %d, choice %d, ran %q", m.step, m.inPane,
			m.cursor, m.choice, r.calls)
	}
	m, _ = loaded(t, down, down, enter)
	if m.step != chains || m.checks[m.cursor].ID != "toolchain-project" || !m.choosing() {
		t.Fatalf("the second link: step %d, cursor %d", m.step, m.cursor)
	}
	if m = drive(m, press('1'), down); m.step != introduction || m.checks[m.cursor].ID != "toolchain-project" {
		t.Errorf("back on the introduction: step %d, cursor %d", m.step, m.cursor)
	}
	// up past the first link gives the bar the keyboard
	if m = drive(m, up); !m.inPane || m.checks[m.cursor].ID != "jq" {
		t.Fatalf("up to the first link: in the pane %v, cursor %d", m.inPane, m.cursor)
	}
	if m = drive(m, up); m.inPane || m.step != introduction {
		t.Errorf("up past the first link: in the pane %v, step %d", m.inPane, m.step)
	}
}

// A click on a link goes to its check, as enter does; a click on the introduction's page is no link's, and does
// nothing.
func TestAClickOnALinkGoesToItsCheck(t *testing.T) {
	m, _ := loaded(t)
	y := slices.IndexFunc(strings.Split(view(m), "\n"), func(l string) bool { return strings.Contains(l, "gh is not") })
	if m = drive(m, testkit.Click(6, y)); m.step != setup || m.checks[m.cursor].ID != "gh-auth" || !m.inPane ||
		!m.choosing() {
		t.Errorf("a click on gh-auth's link: step %d, cursor %d, in the pane %v", m.step, m.cursor, m.inPane)
	}
	if m, _ = loaded(t, testkit.Click(6, paneTop+2)); m.step != introduction || m.inPane {
		t.Errorf("a click on the introduction's page: step %d, in the pane %v", m.step, m.inPane)
	}
}

// A link to a check that its step's search hides clears that search, so the check shows under the cursor.
func TestALinkClearsASearchThatHidesItsCheck(t *testing.T) {
	keys := append([]tea.Msg{press('3'), tea.KeyPressMsg{Code: '/', Text: "/"}}, testkit.Typed("zzz")...)
	m, _ := loaded(t, append(keys, esc, press('1'), down, down, down, down, enter)...)
	if m.step != setup || m.checks[m.cursor].ID != "gh-auth" || m.search().Query != "" ||
		!strings.Contains(view(m), " ▸ ❌ gh is not logged in") {
		t.Errorf("step %d, cursor %d, search %q:\n%s", m.step, m.cursor, m.search().Query, view(m))
	}
}

// The introduction follows the checks as they run again: a fix that holds takes its link away, and while they run,
// checking… follows what they last found.
func TestTheIntroductionFollowsTheChecksAsTheyRunAgain(t *testing.T) {
	m, _ := atChains(t, press('n'), press('1'))
	if got := view(m); m.step != introduction || !strings.Contains(got, "\n   4 checks need your attention\n") ||
		strings.Contains(got, "jq is not installed") {
		t.Errorf("step %d:\n%s", m.step, got)
	}
	m.checking.Again(m.env(), false)
	if got := view(m); !strings.Contains(got, "the release → 🔧 setup\n\n   checking…\n") {
		t.Errorf("no checking under the links:\n%s", got)
	}
	chainsChecking := "add it to your shell profile to keep it\n\n   checking…\n"
	if got := view(drive(m, press('2'))); !strings.Contains(got, chainsChecking) {
		t.Errorf("no checking under the toolchains' checks:\n%s", got)
	}
	// a refusal when they run again comes first, over what they last found
	refused := source.Answer[[]kit.Check]{Name: "checks", Err: errors.New("kit welcome refuses"), Gen: m.checking.Gen()}
	m = drive(m, refused, press('1'))
	if got := view(m); !strings.Contains(got, "\n   ❌ kit welcome refuses\n\n   4 checks need your attention\n") {
		t.Errorf("no refusal over the links:\n%s", got)
	}
	// a check left for later says so in its link
	m, _ = atChains(t, press('l'), press('1'))
	if got := view(m); !strings.Contains(got, "\n   ❌ jq is not installed · later → 🧰 toolchains\n") {
		t.Errorf("jq's link does not say it is left for later:\n%s", got)
	}
}
func TestItOpensOnTheIntroductionWhenEveryCheckPasses(t *testing.T) {
	m, _ := fake(t)
	state := []byte("jq\ntoolchain-project\ngh-auth\nkit-check\nbrew-path\n")
	if err := os.WriteFile(os.Getenv("FAKE_WELCOME_STATE"), state, 0o644); err != nil {
		t.Fatal(err)
	}
	m = drive(m, testkit.Start(m, at120)...)
	if m.step != introduction || m.inPane {
		t.Errorf("step %d, in the pane %v", m.step, m.inPane)
	}
	if got := view(m); strings.Contains(got, "setup 0") || strings.Contains(got, "toolchains 0") ||
		!strings.Contains(got, "\n   ✅ all checks pass\n") || strings.Contains(got, "need your attention") {
		t.Errorf("a step counts no failure, and the introduction says all pass:\n%s", got)
	}
	// with no link, the introduction is a page
	if m = drive(m, down); !m.inPane || keysAt(m) != scrollKeys {
		t.Errorf("down: in the pane %v, keys %q", m.inPane, keysAt(m))
	}
}

func TestAPersonWhoMovedBeforeTheChecksAnsweredStaysThere(t *testing.T) {
	m, _ := fake(t)
	m = drive(m, append([]tea.Msg{at120, tabKey, tabKey}, testkit.Start(m)...)...)
	if m.step != setup || m.inPane {
		t.Errorf("step %d, in the pane %v", m.step, m.inPane)
	}
}

func TestCheckingShowsWhileTheChecksRun(t *testing.T) {
	m, _ := fake(t)
	m = drive(m, at120)
	if got := view(m); !strings.Contains(got, "\n   checking…\n") ||
		strings.Contains(m.View().Content, style.Loss("…")) {
		t.Errorf("the introduction does not say it is checking, or … is red:\n%s", got)
	}
	// a count is right-aligned in its tab, so the spaces before it are folded to one
	labels := regexp.MustCompile(` +`).ReplaceAllString(strings.Split(view(m), "\n")[2], " ")
	for _, want := range []string{"│ 👋 introduction", "🧰 toolchains …", "🔧 setup …", "📖 terms"} {
		if !strings.Contains(labels, want) {
			t.Errorf("no %q: %q", want, labels)
		}
	}
	if m = drive(m, tabKey); !strings.Contains(view(m), "   checking…") {
		t.Errorf("no checking:\n%s", view(m))
	}
}

func TestLeftAndRightMoveAlongTheStepsAndStopAtTheEnds(t *testing.T) {
	m, _ := atChains(t, esc)
	if first := drive(m, backTab, backTab); first.step != introduction {
		t.Errorf("shift-tab past introduction: step %d", first.step)
	}
	for _, want := range []int{setup, custom, terms, terms} {
		if m = drive(m, tabKey); m.step != want {
			t.Fatalf("tab: step %d, want %d", m.step, want)
		}
	}
	// the selected step's name is drawn in the accent, the rest are not
	tabs := strings.Split(m.View().Content, "\n")[2]
	if !strings.Contains(tabs, style.Mark("terms")) || strings.Contains(tabs, style.Mark("customize")) {
		t.Errorf("the terms are not marked: %q", tabs)
	}
	// down enters the rows where the cursor was, once the bar has moved away and back
	m = drive(m, backTab, backTab, down, down)
	if m.step != setup || m.cursor != 4 || m.choosing() {
		t.Fatalf("step %d, cursor %d", m.step, m.cursor)
	}
	if m = drive(m, esc, tabKey); m.step != custom || m.inPane {
		t.Errorf("right on the bar: step %d, in the pane %v", m.step, m.inPane)
	}
	if m = drive(m, backTab, down); m.step != setup || m.cursor != 4 {
		t.Errorf("back on setup: step %d, cursor %d", m.step, m.cursor)
	}
}

func TestLeftAndRightOnAFailedRowMoveBetweenNowAndLater(t *testing.T) {
	m, r := atChains(t, tabKey)
	if m.step != chains || m.choice != later {
		t.Fatalf("step %d, choice %d", m.step, m.choice)
	}
	if !strings.Contains(m.View().Content, style.Cursor(" later ")) {
		t.Errorf("later is not highlighted: %q", m.View().Content)
	}
	m = drive(m, backTab)
	if m.choice != now {
		t.Errorf("shift-tab: choice %d", m.choice)
	}
	// enter on later leaves the check, as l does, and moves to the step's next failed check, the project's toolchain
	m = drive(m, tabKey, enter)
	if !m.deferred["jq"] || m.cursor != 2 || m.step != chains || len(r.calls) != 0 {
		t.Errorf("enter on later: deferred %v, cursor %d, ran %q", m.deferred, m.cursor, r.calls)
	}
}

func TestNRunsTheFixWithTheTerminalThenReadsTheChecksAgain(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_WELCOME_CALLS", record)
	for _, key := range []tea.Msg{press('n'), enter} {
		m, r := atChains(t, key)
		if len(r.calls) != 1 || strings.Join(r.calls[0], " ") != "welcome --fix jq" {
			t.Fatalf("%v ran %q", key, r.calls)
		}
		// the checks ran again, jq is fixed now, and the cursor stays on its row
		if !m.checks[1].OK || m.cursor != 1 || m.step != chains || m.checking.Running() || m.choosing() {
			t.Errorf("%v: jq %+v, cursor %d, checking %v", key, m.checks[1], m.cursor, m.checking.Running())
		}
		if !strings.Contains(view(m), " ▸ ✅ jq is not installed") || strings.Contains(view(m), "brew install jq") {
			t.Errorf("%v:\n%s", key, view(m))
		}
	}
	want := []string{"welcome --checks --json --progress", "welcome --fix jq", "welcome --checks --json --progress"}
	if got := calls(t, record); !slices.Equal(got, append(want, want...)) {
		t.Errorf("the calls: %q", got)
	}
}

func TestTheToolchainsStepOffersNowAndLaterWithTheOthersBeside(t *testing.T) {
	m, r := atChains(t, down)
	if m.step != chains || m.cursor != 2 || !m.choosing() {
		t.Fatalf("step %d, cursor %d", m.step, m.cursor)
	}
	got := view(m)
	for _, want := range []string{" ▸ ❌ several toolchains are usable for the project: mise, brew; choose one",
		"\n      kit toolchains use mise\n", "\n      or: kit toolchains use brew\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q:\n%s", want, got)
		}
	}
	m = drive(m, press('n'))
	if len(r.calls) != 1 || strings.Join(r.calls[0], " ") != "welcome --fix toolchain-project" || !m.checks[2].OK {
		t.Errorf("ran %q, check %+v", r.calls, m.checks[2])
	}
	if labels := regexp.MustCompile(` +`).ReplaceAllString(strings.Split(view(m), "\n")[2], " "); !strings.Contains(
		labels, "toolchains 2 ") {
		t.Errorf("the toolchains still count the project's toolchain:\n%s", view(m))
	}
}

func TestLaterKeepsTheOtherToolchainsBesideTheFix(t *testing.T) {
	m, _ := atChains(t, down, press('l'))
	if !m.deferred["toolchain-project"] {
		t.Fatalf("deferred %v", m.deferred)
	}
	want := "\n   • several toolchains are usable for the project: mise, brew; " +
		"choose one: kit toolchains use mise\n" +
		"     or: kit toolchains use brew\n"
	if got := view(m); !strings.Contains(got, want) {
		t.Errorf("no %q:\n%s", want, got)
	}
}

func TestNOnACheckThatStillFailsSaysSo(t *testing.T) {
	m, r := atChains(t, esc, tabKey, down, press('n'))
	if len(r.calls) != 1 || strings.Join(r.calls[0], " ") != "welcome --fix gh-auth" {
		t.Fatalf("ran %q", r.calls)
	}
	if m.cursor != 3 || m.checks[3].OK || !strings.Contains(view(m), "the fix ran, and the check still fails") {
		t.Errorf("cursor %d:\n%s", m.cursor, view(m))
	}
}

func TestARefusedFixShowsInRed(t *testing.T) {
	t.Setenv("FAKE_WELCOME_REFUSE_FIX", "jq")
	m, _ := atChains(t, press('n'))
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
	m, r := atChains(t, esc, tabKey, down, press('l'))
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
	if m.choosing() || keysAt(m) != rowKeys {
		t.Fatalf("cursor %d, keys %q", m.cursor, keysAt(m))
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
	m = drive(m, press('l'), esc, backTab, down, press('l'))
	if !m.deferred["jq"] || strings.Contains(view(m), "• gh is not logged in") {
		t.Errorf("another step's check is listed:\n%s", view(m))
	}
}

func TestAFailedCheckWithoutAFixShowsWhereToReadMore(t *testing.T) {
	m, _ := atChains(t, esc, tabKey, down, down)
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
	m, _ := house(t, down, enter)
	next, cmd := m.Update(esc)
	m = next.(Model)
	if cmd != nil || m.inPane || keysAt(m) != inHouse(stepKeys) {
		t.Fatalf("esc in the rows: in the pane %v, keys %q", m.inPane, keysAt(m))
	}
	if next, cmd = m.Update(esc); cmd != nil || next.(Model).inPane {
		t.Errorf("esc on the bar did something")
	}
	// a page takes the keyboard on down, and esc gives it back, the page still scrolled
	m = drive(m, backTab, tea.WindowSizeMsg{Width: 120, Height: 12}, stepKey(ourTeam))
	if m.step != ourTeam || keysAt(m) != inHouse(pageKeys) {
		t.Fatalf("step %d, keys %q", m.step, keysAt(m))
	}
	if m = drive(m, down); !m.inPane || keysAt(m) != inHouse(scrollKeys) || m.offsets[ourTeam] != 0 {
		t.Fatalf("down on a page's bar: in the pane %v, keys %q, offset %d", m.inPane, keysAt(m), m.offsets[ourTeam])
	}
	if m = drive(m, down, esc); m.inPane || m.offsets[ourTeam] != 1 {
		t.Errorf("esc in a page: in the pane %v, offset %d", m.inPane, m.offsets[ourTeam])
	}
	// up at the page's top gives the bar the keyboard
	if m = drive(m, down, up); !m.inPane || m.offsets[ourTeam] != 0 {
		t.Fatalf("up in a page: in the pane %v, offset %d", m.inPane, m.offsets[ourTeam])
	}
	if m = drive(m, up); m.inPane {
		t.Errorf("up at the page's top kept the keyboard in the page")
	}
}

// ctrl-q quits; esc with a modifier, which a terminal with its default settings sends as esc, does not.
func TestCtrlQQuitsAndNoEscDoes(t *testing.T) {
	m, _ := loaded(t)
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
	if cmd == nil || cmd() != tea.Quit() || next.(Model).Interrupted() {
		t.Error("ctrl-q did not quit")
	}
	for _, mod := range []tea.KeyMod{0, tea.ModAlt, tea.ModSuper} {
		if _, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape, Mod: mod}); cmd != nil && cmd() == tea.Quit() {
			t.Errorf("%v esc quit", mod)
		}
	}
}

func TestLeftAndRightMoveStepsOnlyFromTheBar(t *testing.T) {
	m, _ := atChains(t)
	// the screen opens in the toolchains' rows, on a check without choices
	m = drive(m, down, down, down)
	if m.choosing() {
		t.Fatalf("cursor %d offers choices", m.cursor)
	}
	if m = drive(m, tabKey, backTab); m.step != chains || !m.inPane {
		t.Errorf("left and right in the rows: step %d, in the pane %v", m.step, m.inPane)
	}
	if m = drive(m, esc, tabKey); m.step != setup {
		t.Errorf("right on the bar: step %d", m.step)
	}
}

// A click on a check's line puts the cursor on that check, and the pane has the keyboard; a click on a page gives the
// page the keyboard; a click on a line that is no check's does nothing.
func TestAClickOnACheckSelectsIt(t *testing.T) {
	click := func(y int) tea.Msg { return testkit.Click(6, y) }
	m, _ := house(t, down, enter, esc)
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
	if m = drive(m, esc, stepKey(ourTeam), click(paneTop+2)); m.step != ourTeam || !m.inPane ||
		keysAt(m) != inHouse(scrollKeys) {
		t.Errorf("a click on a page: step %d, in the pane %v", m.step, m.inPane)
	}
	if m = drive(m, esc, click(0)); m.inPane {
		t.Errorf("a click on the title gave the pane the keyboard")
	}
}

func TestANumberOrAClickSelectsAStepFromAnywhere(t *testing.T) {
	m, _ := atChains(t)
	if m = drive(m, stepKey(terms)); m.step != terms || m.inPane {
		t.Errorf("the terms' number in the rows: step %d, in the pane %v", m.step, m.inPane)
	}
	if m = drive(m, tea.KeyPressMsg{Code: tea.KeyKp1}); m.step != introduction {
		t.Errorf("keypad 1: step %d", m.step)
	}
	x := testkit.Column(strings.Split(view(m), "\n")[1], '➁')
	if m = drive(m, testkit.Click(x, 1)); m.step != chains {
		t.Errorf("a click on ➁: step %d\n%s", m.step, view(m))
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
	m, _ := atChains(t, esc)
	if m = drive(m, down); !m.inPane || m.cursor != 1 {
		t.Fatalf("down from the line: in the pane %v, cursor %d", m.inPane, m.cursor)
	}
	if m = drive(m, down, down, down); m.cursor != 5 {
		t.Errorf("down past the step's last row: cursor %d", m.cursor)
	}
	if m = drive(m, up, up, up); m.cursor != 0 {
		t.Errorf("up: cursor %d", m.cursor)
	}
	if m = drive(m, pgup); !m.inPane || m.cursor != 0 {
		t.Errorf("page up past the first row left the rows: in the pane %v, cursor %d", m.inPane, m.cursor)
	}
	if m = drive(m, up, pgdown, pgup); m.inPane {
		t.Errorf("a page on the bar entered the rows")
	}
	// home and end go to the step's first row and its last, as far as pages go
	if m = drive(m, enter, press(tea.KeyEnd)); !m.inPane || m.cursor != 5 {
		t.Errorf("end: in the pane %v, cursor %d", m.inPane, m.cursor)
	}
	if m = drive(m, press(tea.KeyHome)); !m.inPane || m.cursor != 0 {
		t.Errorf("home: in the pane %v, cursor %d", m.inPane, m.cursor)
	}
	m = drive(m, up)
	if m = drive(m, enter); !m.inPane {
		t.Errorf("enter on the line did not enter the rows")
	}
	// a step entered for the first time starts on its first row
	if m = drive(m, esc, tabKey, down); m.step != setup || m.cursor != 3 {
		t.Errorf("setup: step %d, cursor %d", m.step, m.cursor)
	}
}

// When kit refuses the checks, the screen stays on the introduction, which says what kit said, in red, as each step
// with checks does; those count ! in red on the bar.
func TestARefusalOfTheChecksShowsInRed(t *testing.T) {
	t.Setenv("FAKE_WELCOME_REFUSE", "1")
	m, _ := loaded(t)
	want := "❌ kit welcome refuses for the test; run: kit help welcome"
	fields := strings.Join(strings.Fields(view(m)), " ")
	if m.step != introduction || !strings.Contains(view(m), "\n   "+want+"\n") ||
		strings.Contains(view(m), "checking…") ||
		!strings.Contains(fields, "🧰 toolchains !") || !strings.Contains(fields, "🔧 setup !") {
		t.Errorf("step %d:\n%s", m.step, view(m))
	}
	if !strings.Contains(m.View().Content, style.Refusal(want)) || !strings.Contains(m.View().Content, style.Loss("!")) {
		t.Errorf("the refusal, or a step's !, is not red: %q", m.View().Content)
	}
	if m = drive(m, down); !m.inPane || keysAt(m) != scrollKeys {
		t.Errorf("down on the introduction with no links: in the pane %v, keys %q", m.inPane, keysAt(m))
	}
	if m = drive(m, esc, tabKey); m.step != chains || !strings.Contains(view(m), "   "+want) {
		t.Errorf("the toolchains do not show the refusal:\n%s", view(m))
	}
}
func TestAPageScrollsWithWhatIsHiddenCounted(t *testing.T) {
	m, _ := house(t, tea.WindowSizeMsg{Width: 120, Height: 12}, esc, stepKey(ourTeam))
	if m.step != ourTeam {
		t.Fatal(m.step)
	}
	all := len(m.pageLines(ourTeam))
	pane := func() []string { return strings.Split(view(m), "\n")[4:10] }
	if got := pane(); got[0] != "   Our team" || got[5] != "   … "+strconv.Itoa(all-5)+" more" {
		t.Fatalf("top:\n%s", strings.Join(got, "\n"))
	}
	// the first down gives the page the keyboard, the next scrolls it
	m = drive(m, down, down)
	if got := pane(); got[0] != "   … 1 above" || got[5] != "   … "+strconv.Itoa(all-5)+" more" {
		t.Errorf("one down:\n%s", strings.Join(got, "\n"))
	}
	m = drive(m, pgdown, pgdown, pgdown, pgdown, pgdown, pgdown)
	if got := pane(); !strings.HasPrefix(got[0], "   … ") || !strings.Contains(got[5], "kit flow desk") {
		t.Errorf("the end:\n%s", strings.Join(got, "\n"))
	}
	if m = drive(m, pgup, pgup, pgup, pgup, pgup, up); m.offsets[ourTeam] != 0 {
		t.Errorf("back to the top: offset %d", m.offsets[ourTeam])
	}
	// end scrolls a page to its last line, and home back to its first
	if m = drive(m, down, press(tea.KeyEnd)); !strings.Contains(pane()[5], "kit flow desk") {
		t.Errorf("end:\n%s", strings.Join(pane(), "\n"))
	}
	if m = drive(m, press(tea.KeyHome), up); m.offsets[ourTeam] != 0 {
		t.Errorf("home: offset %d", m.offsets[ourTeam])
	}
	// each step keeps its own place
	m = drive(m, down, pgdown, esc, backTab, tabKey)
	if m.offsets[ourTeam] == 0 {
		t.Errorf("the place was lost")
	}
}

func TestTheCursorsRowStaysInViewOnAShortScreen(t *testing.T) {
	m, _ := atChains(t, tea.WindowSizeMsg{Width: 120, Height: 10}, esc, tabKey, down, down)
	if got := view(m); !strings.Contains(got, " ▸ ❌ kit check finds") || !strings.Contains(got, "above") {
		t.Errorf("the cursor's row is hidden:\n%s", got)
	}
}

// Leaving the screen, with ctrl-q or ctrl-c, ends the checks still running, and what they started with them.
func TestLeavingTheScreenEndsTheChecksStillRunning(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{{Code: 'q', Mod: tea.ModCtrl}, {Code: 'c', Mod: tea.ModCtrl}} {
		t.Setenv("FAKE_WELCOME_SLOW", "1")
		m, _ := fake(t)
		answers := make(chan tea.Msg, 1)
		check := m.check()
		go func() { answers <- check() }()
		time.Sleep(200 * time.Millisecond)
		m = drive(m, key)
		select {
		case msg := <-answers:
			if err := msg.(source.Answer[[]kit.Check]).Err; !errors.Is(err, context.Canceled) {
				t.Errorf("%s: the checks ended with %v, not the quit", key, err)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("%s: the checks outlived the screen", key)
		}
	}
}

// On every step, on the bar and on each of its rows, the keys bar names n and Enter only where they act, on a row that
// offers now or later, and l there and on a fix kit only shows.
func TestTheKeysBarNamesOnlyTheKeysThatActThere(t *testing.T) {
	m, _ := loaded(t)
	for range len(m.tabs) {
		for k := 0; k < 30; k++ {
			bar := keysAt(m)
			for _, c := range []struct {
				key  string
				acts bool
			}{
				{"n now", m.choosing()},
				{"⏎ choose", m.choosing()},
				{"l later", m.choosing() || m.showOnly()},
			} {
				if strings.Contains(bar, c.key) != c.acts {
					t.Errorf("step %d, in the pane %v, cursor %d: %q names %s: %v", m.step, m.inPane, m.cursor, bar, c.key,
						!c.acts)
				}
			}
			before := m.cursor
			if m = drive(m, down); m.cursor == before && k > 0 {
				break
			}
		}
		m = drive(m, esc, tabKey)
	}
}

// Every height from a line to a dozen, on the bar, in the rows, scrolled, and on a page: the screen draws as many lines
// as it has, the keys bar the last, and never panics.
func TestTheKeysBarIsTheLastLineAtEveryHeight(t *testing.T) {
	for _, msgs := range [][]tea.Msg{
		nil,
		{down},
		{down, down, down},
		{esc, tabKey, tabKey, down, down},
		{pgdown, pgdown, pgdown},
		{tabKey, tabKey, tabKey, tabKey, pgdown, pgdown},
	} {
		for _, width := range []int{120, 40} {
			check := func(m Model, height int, how string) {
				got := strings.Split(view(m), "\n")
				if bar := ansi.Strip(m.keys().Line(width)); len(got) != height || got[height-1] != bar {
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
			m = drive(m, esc, tabKey)
		}
	}
}

// At 130 columns every tab is whole; at 80 the bar scrolls, ▹ after the tab it cuts, and the toolchains stay whole.
func TestEveryTabIsWholeAt130AndScrollsAt80(t *testing.T) {
	m, _ := atChains(t, tea.WindowSizeMsg{Width: 130, Height: 30})
	labels := strings.Split(view(m), "\n")[2]
	for _, s := range known {
		if !strings.Contains(labels, s.glyph+" "+s.name) {
			t.Errorf("at 130, no %q whole: %q", s.name, labels)
		}
	}
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 30})
	labels = strings.Split(view(m), "\n")[2]
	if !strings.Contains(labels, "│ 🧰 toolchains  3 │") || strings.Contains(labels, "…") ||
		!strings.HasSuffix(labels, tabs.After) {
		t.Errorf("at 80: %q", labels)
	}
}

func TestTheGlyphsAreTwoCellsWithoutAVariationSelector(t *testing.T) {
	glyphs := []string{shell.Books, Passed, Failed, pageGlyph}
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
	r := repoWith(t, map[string]string{"5-terms.md": "# Our terms\n"})
	// the repository has pages: they are the steps, as kit welcome --text reads them, with no other filling in
	tabs := readPages(r)
	if len(tabs) != 1 || tabs[0].id != "terms" || tabs[0].page != "# Our terms\n" || tabs[0].missing != "" {
		t.Errorf("tabs %+v", tabs)
	}
	// with neither, the known steps say where each was looked for
	t.Setenv("FAKE_WELCOME_ROOT", nowhere)
	r = repoWith(t, nil)
	tabs = readPages(r)
	want := filepath.Join(nowhere, packageDir, pagesDir, known[terms].file)
	if len(tabs) != len(known) || tabs[terms].missing != want {
		t.Errorf("tabs %+v", tabs)
	}
	// the path is broken across lines, so all of it shows at any width
	for _, width := range []int{120, 40} {
		m := drive(New(r, ""), tea.WindowSizeMsg{Width: width, Height: 30}, pagesMsg{tabs: tabs}, stepKey(terms))
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
	want := []string{"1-introduction.md", "2-toolchains.md", "3-setup.md", "4-customize.md", "5-terms.md"}
	if !slices.Equal(files, want) || !strings.HasPrefix(tabs[chains].page, "# Toolchains") {
		t.Errorf("files %q", files)
	}
}

func TestAHousePackagesPagesAreStepsAndItsReplacementKeepsTheStep(t *testing.T) {
	m, _ := house(t)
	if len(m.tabs) != len(known)+1 {
		t.Fatalf("%d steps", len(m.tabs))
	}
	for i, k := range known {
		if m.tabs[i].id != k.id || m.tabs[i].glyph != k.glyph || m.tabs[i].checked != k.checked {
			t.Errorf("step %d: %+v", i, m.tabs[i])
		}
	}
	added := m.tabs[ourTeam]
	if added.id != "our-team" || added.glyph != pageGlyph || added.name != "our team" || added.checked {
		t.Errorf("the added step: %+v", added)
	}
	// the screen stays on the introduction, and its first link still goes to the first failed check, under the toolchains
	if m.step != introduction {
		t.Errorf("step %d", m.step)
	}
	if m = drive(m, down, enter); m.step != chains || !m.choosing() {
		t.Errorf("step %d, choosing %v", m.step, m.choosing())
	}
	// the replaced page shows under terms; the added one last, a page with no checks
	if m = drive(m, esc, stepKey(terms)); m.step != terms || !strings.Contains(view(m), "\n   Our terms\n") {
		t.Errorf("step %d:\n%s", m.step, view(m))
	}
	m = drive(m, stepKey(ourTeam))
	labels := strings.Split(view(m), "\n")[2]
	if m.step != ourTeam || m.hasRows() || keysAt(m) != inHouse(pageKeys) || !strings.Contains(labels, "│ 📄 our team") ||
		!strings.Contains(view(m), "\n   Our team\n") {
		t.Errorf("step %d:\n%s", m.step, view(m))
	}
	// at 120 columns the eight do not fit: the bar scrolls to the last, whole, no name cut; ◃, which would cut it, is
	// hidden
	if strings.Contains(labels, tabs.Before) || strings.Contains(labels, tabs.After) || strings.Contains(labels, "…") {
		t.Errorf("labels %q", labels)
	}
}

func TestThePagesAreInTheOrderKitWelcomeGives(t *testing.T) {
	files := map[string]string{}
	for _, f := range []string{"1-introduction.md", "10-ten.md", "4-setup.md", "45-deploys.md", "5-applications.md",
		"2-toolchains.md", "notes.md", "04-also.md", ".hidden.md", "notes.txt"} {
		files[f] = "# " + f + "\n"
	}
	var got []string
	for _, tab := range readPages(repoWith(t, files)) {
		got = append(got, tab.file)
	}
	// by the number a name starts with, a tie by name, and a name with no number last, as kit welcome --text
	want := []string{"1-introduction.md", "2-toolchains.md", "04-also.md", "4-setup.md", "5-applications.md",
		"10-ten.md", "45-deploys.md", "notes.md"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if idOf("45-deploys.md") != "deploys" || idOf("notes.md") != "notes" ||
		tabOf("9-our-on-call.md").name != "our on call" {
		t.Errorf("ids")
	}
}

func TestTheBarScrollsToKeepTheSelectedStepInView(t *testing.T) {
	files := map[string]string{}
	for _, k := range known {
		files[k.file] = "# " + k.name + "\n"
	}
	for i := range 20 {
		files["7-page-"+strconv.Itoa(i)+".md"] = "# page\n"
	}
	at80 := tea.WindowSizeMsg{Width: 80, Height: 30}
	m, _ := fake(t)
	m.runner = repoWith(t, files)
	m = drive(m, append(testkit.Start(m, at80), esc, backTab)...)
	if len(m.tabs) != len(known)+20 || m.step != 0 {
		t.Fatalf("%d steps, step %d", len(m.tabs), m.step)
	}
	for _, c := range []struct {
		keys          []tea.Msg
		label         string
		start, finish bool // whether each end has a triangle, hidden where it would cut the selected step
	}{
		{nil, "👋 introduction │", false, false},
		{slices.Repeat([]tea.Msg{tabKey}, terms), "📖 terms │", true, true},
		{slices.Repeat([]tea.Msg{tabKey}, len(known)+19-terms), "📄 page 9 │", false, false},
	} {
		m = drive(m, c.keys...)
		lines := strings.Split(view(m), "\n")
		if !strings.Contains(lines[2], c.label) {
			t.Errorf("step %d is not in view:\n%s", m.step, lines[2])
		}
		if strings.HasPrefix(lines[2], " "+tabs.Before) != c.start || strings.HasSuffix(lines[2], tabs.After) != c.finish {
			t.Errorf("step %d's ends:\n%s", m.step, lines[2])
		}
		for _, l := range lines {
			if ansi.StringWidth(l) > 80 {
				t.Errorf("%q is %d cells", l, ansi.StringWidth(l))
			}
		}
	}
}

func TestSixStepsFitAtEightyColumnsWithTheLastInFull(t *testing.T) {
	m, _ := house(t, tea.WindowSizeMsg{Width: 80, Height: 30}, esc, stepKey(ourTeam))
	lines := strings.Split(view(m), "\n")
	if m.step != ourTeam || !strings.Contains(lines[2], "│ 📄 our team │") {
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
	if m, _ := fake(t); !strings.HasPrefix(view(m), "  "+shell.App+" kit welcome · - empowerite/kit ") ||
		!strings.HasSuffix(strings.Split(view(m), "\n")[0], shell.Find+" "+shell.Idle) {
		t.Errorf("title:\n%s", view(m))
	}
	// with no repository known, the control is - alone, which still drops down what to switch to
	if got := strings.Split(view(New(testkit.Runner(t, "welcome"), "")), "\n")[0]; !strings.HasPrefix(got,
		"  "+shell.App+" kit welcome · "+shell.Workspaces+" ") ||
		strings.Contains(strings.ReplaceAll(got, shell.Idle, ""), "/") {
		t.Errorf("title without a repository:\n%s", got)
	}
}

// showOnly opens the screen on the toolchains step's show-only check, brew-path, below the project's toolchain.
func showOnly(t *testing.T, msgs ...tea.Msg) (Model, *ran) {
	t.Helper()
	return atChains(t, append([]tea.Msg{down, down}, msgs...)...)
}

func TestAShowOnlyFixDrawsItsLinesAndNoChoices(t *testing.T) {
	m, _ := showOnly(t)
	if m.checks[m.cursor].ID != "brew-path" || m.choosing() || !m.showOnly() || keysAt(m) != showKeys {
		t.Fatalf("cursor %d, choosing %v, keys %q", m.cursor, m.choosing(), keysAt(m))
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
	m = drive(m, press('n'), enter, backTab, tabKey)
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

// = or a click on the = before the title drops the app's menu down over the guide, the welcome's own screen dim: help
// asks kittui to show it in the guide's place, quit ends the guide, and Close ends the checks still running.
func TestEqualsDropsTheAppsMenuDownOverTheGuide(t *testing.T) {
	m, _ := fake(t)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 30}, tea.KeyPressMsg{Code: '=', Text: "="})
	if v := view(m); !m.appMenu.Open || !strings.Contains(v, "Help") || !strings.HasSuffix(v, shell.Path{shell.MenuKeys()}.String()) {
		t.Fatalf("= did not drop the app's menu down:\n%s", v)
	}
	_, msgs := testkit.Drive(m, func(msg tea.Msg) bool { _, ok := msg.(shell.SwitchMsg); return ok },
		tea.KeyPressMsg{Code: 'h', Text: "h"}, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !slices.Contains(msgs, tea.Msg(shell.SwitchMsg{Screen: shell.Help})) {
		t.Errorf("help asked %v", msgs)
	}
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyEscape}, testkit.Click(shell.At+ansi.StringWidth(shell.App)+len(appLabel), shell.Top))
	if !m.appMenu.Open {
		t.Fatal("a click on the screen's name did not drop the menu down")
	}
	if _, msgs := testkit.Drive(m, nil, testkit.Click(3, shell.Top+17)); !slices.Contains(msgs, tea.Msg(tea.QuitMsg{})) {
		t.Errorf("a click on quit: %v", msgs)
	}
	if m.chose(shell.Choice{}) != nil {
		t.Error("no choice did something")
	}
	m.Close()
	if m.ctx.Err() == nil {
		t.Error("Close did not end the checks")
	}
}

// The guide's search is every screen's, one for each step: on a step with checks it narrows them to those whose id,
// line, or fix holds what is typed; on a page, to the lines that hold it.
func TestTheGuidesSearchNarrowsEachStep(t *testing.T) {
	m, _ := loaded(t, tea.WindowSizeMsg{Width: 100, Height: 40}, tea.KeyPressMsg{Code: tea.KeyEscape})
	for !m.tabs[m.step].checked {
		m = drive(m, tabKey)
	}
	all := len(m.rows(m.step))
	m = drive(m, tea.KeyPressMsg{Code: '/', Text: "/"})
	if !m.search().Focused || !strings.HasSuffix(view(m), "\n ⏎ checks type to search") {
		t.Fatalf("/ did not give the search the keyboard:\n%s", view(m))
	}
	first := m.checks[m.rows(m.step)[0]].ID
	m = drive(m, testkit.Typed(first)...)
	if n := len(m.rows(m.step)); n == 0 || n > all || m.checks[m.cursor].ID != first {
		t.Errorf("searching %q: %d of %d checks, the cursor on %s", first, n, all, m.checks[m.cursor].ID)
	}
	m = drive(m, testkit.Typed("zzz")...)
	if len(m.rows(m.step)) != 0 || !strings.Contains(view(m), "no check matches the search") {
		t.Errorf("a search no check matches:\n%s", view(m))
	}
	// ⌃U clears what is typed, and the hint names it while there is some
	if !strings.Contains(view(m), "⌃U clear") {
		t.Errorf("⌃U is not named:\n%s", view(m))
	}
	if m = drive(m, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}); m.search().Query != "" || !m.search().Focused {
		t.Errorf("⌃U: %+v", m.search())
	}
	m = drive(m, testkit.Typed("zzz")...)
	// esc gives the pane focus, what is typed kept; another step has a search of its own
	if m = drive(m, tea.KeyPressMsg{Code: tea.KeyEscape}); m.search().Focused || m.search().Query == "" || !m.inPane {
		t.Errorf("esc: %+v, the pane focused %v", m.search(), m.inPane)
	}
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyEscape}, backTab)
	if m.search().Query != "" {
		t.Errorf("the step before shares the search: %q", m.search().Query)
	}
	// on a page, the search keeps the lines that hold it; one no line holds says so
	for m.tabs[m.step].checked {
		m = drive(m, backTab)
	}
	m = drive(m, testkit.Click(m.searchAt()+1, shell.Top))
	if !m.search().Focused {
		t.Fatal("a click on the search did not give it the keyboard")
	}
	if m = drive(m, testkit.Typed("qqqq")...); !strings.Contains(view(m), "no line matches the search") {
		t.Errorf("a page no line of which holds the search:\n%s", view(m))
	}
}

// In the pane, esc says it goes back to the tabs, as it does on every screen with a tab bar.
func TestEscSaysItGoesBackToTheTabs(t *testing.T) {
	m, _ := loaded(t)
	m = drive(m, down)
	if want := "│ esc tabs "; !m.inPane || !strings.Contains(m.keys().String(), want) {
		t.Errorf("in the pane %v: %q does not hold %q", m.inPane, m.keys().String(), want)
	}
}
