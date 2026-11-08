package welcome

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// A search that gives up the keyboard with something typed is remembered, as on every screen; with the keyboard again,
// ctrl+r or a click on ▾ drops the searches down, and choosing one puts it in the step's search.
func TestWelcomesSearchIsRememberedAndRecalled(t *testing.T) {
	m, _ := loaded(t)
	m.recent = shell.Recent{Path: filepath.Join(t.TempDir(), "search-history.json")}
	m = drive(m, append(testkit.Typed("/jq"), tea.KeyPressMsg{Code: tea.KeyEscape})...)
	if !slices.Equal(m.recent.Items, []string{"jq"}) {
		t.Fatalf("remembered: %q", m.recent.Items)
	}
	m = drive(m, append(testkit.Typed("/"), tea.KeyPressMsg{Code: tea.KeyBackspace}, tea.KeyPressMsg{Code: tea.KeyBackspace})...)
	top := strings.Split(ansi.Strip(m.View().Content), "\n")
	if !m.recalls() || !strings.Contains(top[0], shell.Recall) || !strings.Contains(top[len(top)-1], "⌃R recent") {
		t.Fatalf("no recall: %q, %q", top[0], top[len(top)-1])
	}
	m = drive(m, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if bar := strings.Split(ansi.Strip(m.View().Content), "\n"); !m.recent.Open || !strings.Contains(bar[len(bar)-1], "esc close") {
		t.Fatal("ctrl+r did not drop the searches down")
	}
	if m = drive(m, tea.KeyPressMsg{Code: tea.KeyEnter}); m.recent.Open || m.search().Query != "jq" {
		t.Errorf("choosing jq: %q", m.search().Query)
	}
	m = drive(m, testkit.Click(m.recallAt(), shell.Top))
	if !m.recent.Open {
		t.Fatal("a click on ▾ did not drop the searches down")
	}
	_, left := m.recent.Box(m.recallAt(), m.width)
	if m = drive(m, testkit.Click(left+3, 1)); m.recent.Open || m.search().Query != "jq" {
		t.Errorf("a click on jq: %q", m.search().Query)
	}
	if got := Open(m.runner, kit.Workspace{}); got.recent.Path != shell.RecentFile() {
		t.Errorf("Open remembers in %q", got.recent.Path)
	}
}
