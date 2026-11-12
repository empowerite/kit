// The screen is driven key by key through Update, against testdata/fake-kit, which prints testdata/catalogue.json, or
// the catalogue FAKE_TUTORIALS_CATALOGUE names, as Bubble Tea would drive it. The browser is a recorder.
package tutorials

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/source"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// TestMain waits no time for a notice to fade, as drive runs the wait's command in place.
func TestMain(m *testing.M) {
	fades = 0
	os.Exit(testkit.Main(m))
}

// at120 is the design size, and at60 the narrow one.
var (
	at120 = tea.WindowSizeMsg{Width: 120, Height: 30}
	at60  = tea.WindowSizeMsg{Width: 60, Height: 20}
)

func press(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

var (
	up      = press(tea.KeyUp)
	down    = press(tea.KeyDown)
	backTab = tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	tabKey  = press(tea.KeyTab)
	enter   = press(tea.KeyEnter)
	esc     = press(tea.KeyEscape)
)

// browser is what the screen asked the browser to open, and why it fails, when it does.
type browser struct {
	urls []string
	fail error
}

// fake gives the screen over testdata/fake-kit, opened with the tags, its browser the recorder.
func fake(t *testing.T, tags ...string) (Model, *browser) {
	t.Helper()
	m := New(testkit.Runner(t, "tutorials"), tags)
	b := &browser{}
	m.browse = func(url string) error {
		b.urls = append(b.urls, url)
		return b.fail
	}
	return m, b
}

// catalogue gives kit's answer of the catalogue c.
func catalogue(c kit.Catalogue) source.Answer[kit.Catalogue] {
	return source.Answer[kit.Catalogue]{Name: "catalogue", Value: c}
}

// loaded opens the screen at 120×30, runs Init, then sends the messages.
func loaded(t *testing.T, msgs ...tea.Msg) (Model, *browser) {
	t.Helper()
	m, b := fake(t)
	return drive(m, append(testkit.Start(m, at120), msgs...)...), b
}

// drive sends the messages, running every command each update returns and feeding its messages back, but a notice's
// time being up, which a test sends itself.
func drive(m Model, msgs ...tea.Msg) Model {
	m, _ = testkit.Drive(m, faded, msgs...)
	return m
}

// faded says a notice's time is up.
func faded(msg tea.Msg) bool {
	_, ok := msg.(shell.FadedMsg)
	return ok
}

// view gives the screen without its escapes.
func view(m Model) string { return ansi.Strip(m.View().Content) }

// lines gives the screen's lines without their escapes.
func lines(m Model) []string { return strings.Split(view(m), "\n") }

// eight gives a catalogue of eight tutorials, Tutorial 1 to Tutorial 8, the even ones for the UI.
func eight() kit.Catalogue {
	c := kit.Catalogue{Version: "unreleased", Audiences: []kit.Audience{{ID: "newcomer", Label: "Just starting"},
		{ID: "ui", Label: "UI"}}}
	for k := 1; k <= 8; k++ {
		audience := []string{"newcomer"}
		if k%2 == 0 {
			audience = []string{"ui"}
		}
		c.Tutorials = append(c.Tutorials, kit.Tutorial{ID: fmt.Sprintf("t%d", k), Title: fmt.Sprintf("Tutorial %d", k),
			Summary: "A tutorial for the test.", Glyph: "📘", Audience: audience,
			URL: fmt.Sprintf("https://example.test/t%d/?kit=unreleased", k)})
	}
	return c
}

// withEight opens the screen at the size, with the eight tutorials, then sends the messages.
func withEight(t *testing.T, size tea.WindowSizeMsg, msgs ...tea.Msg) (Model, *browser) {
	t.Helper()
	m, b := fake(t)
	return drive(m, append([]tea.Msg{size, catalogue(eight())}, msgs...)...), b
}

// catalogueFile names a catalogue in testdata for the fake to print.
func catalogueFile(t *testing.T, name string) {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_TUTORIALS_CATALOGUE", path)
}

// pillText gives a pill as the screen draws it, without its escapes: the text between the half circles.
func pillText(text string) string { return style.PillLeft + text + style.PillRight }

// bottomEdge gives a card's bottom edge without its escapes, as it stands under the pills, the text they draw.
func bottomEdge(pills string) string {
	return "╰─ " + pills + " " + strings.Repeat("─", cardWidth-5-ansi.StringWidth(pills)) + "╯"
}
