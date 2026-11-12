// A link to another app leaves a crumb on the title line of the app it shows, the way back to where it was followed,
// the view as it was left and its own crumb with it; only the app's menu shows a screen with no crumb.
package screens

import (
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/markdown"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

// crumbAt gives the column the title's crumb starts at, -1 with none.
func crumbAt(m Model) int { return testkit.Column(title(m), []rune(shell.Back)[0]) }

// onRepositoriesIncoming gives the root on Repositories' third tab, incoming, as the app's menu and the tab's number
// show it, over the fake kit, whose Repositories has branches, remotes, and incoming.
func onRepositoriesIncoming(t *testing.T) Model {
	t.Helper()
	m := help(t)
	m = drive(m, shell.SwitchMsg{Screen: shell.Repositories}, testkit.Press("3"))
	if v := ansi.Strip(m.View().Content); !strings.Contains(v, "➌╶") || crumbAt(m) >= 0 {
		t.Fatalf("incoming is not selected, or a crumb shows:\n%s", v)
	}
	return m
}

func TestALinkToAnotherAppLeavesACrumbThatAClickFollowsBack(t *testing.T) {
	m := onRepositoriesIncoming(t)
	m = drive(m, shell.JumpMsg{Screen: shell.Changes, To: "changes/#2"})
	if m.here != shell.Changes || !strings.Contains(title(m), " · "+shell.Back+" repositories › incoming") {
		t.Fatalf("on %s, the title: %q", m.here, title(m))
	}
	// the keys bar names the way back
	if lines := strings.Split(ansi.Strip(m.View().Content), "\n"); !strings.Contains(lines[len(lines)-1], "⌫ back") {
		t.Errorf("the keys bar: %q", lines[len(lines)-1])
	}
	m = drive(m, testkit.Click(crumbAt(m)+1, shell.Top))
	if v := ansi.Strip(m.View().Content); m.here != shell.Repositories || !strings.Contains(v, "➌╶") {
		t.Fatalf("the click went back to %s:\n%s", m.here, v)
	}
	if crumbAt(m) >= 0 {
		t.Errorf("Repositories, shown by the app's menu, has a crumb: %q", title(m))
	}
	m.Close()
}

// Crumbs chain: a link followed from a place that has a crumb leaves a crumb back to it, and going back there shows
// its own crumb again; going back puts the view back as it was left, though a later link changed it since.
func TestCrumbsChainAndEachGoesBackToTheViewItLeft(t *testing.T) {
	m := onRepositoriesIncoming(t)
	m = drive(m, shell.JumpMsg{Screen: shell.Changes, To: "changes/#2"})
	m = drive(m, shell.JumpMsg{Screen: shell.Repositories, To: "workspace/remotes/origin"})
	if v := ansi.Strip(m.View().Content); m.here != shell.Repositories || !strings.Contains(v, "➋╶") ||
		!strings.Contains(title(m), shell.Back+" changes") {
		t.Fatalf("the second link, on %s, the title %q:\n%s", m.here, title(m), v)
	}
	m = drive(m, testkit.Press("backspace"))
	if m.here != shell.Changes || !strings.Contains(title(m), shell.Back+" repositories › incoming") {
		t.Fatalf("⌫ went back to %s, the title %q", m.here, title(m))
	}
	m = drive(m, testkit.Press("backspace"))
	if v := ansi.Strip(m.View().Content); m.here != shell.Repositories || !strings.Contains(v, "➌╶") ||
		crumbAt(m) >= 0 {
		t.Errorf("⌫ again, on %s, the title %q:\n%s", m.here, title(m), v)
	}
	// with no crumb, ⌫ goes nowhere
	if m = drive(m, testkit.Press("backspace")); m.here != shell.Repositories {
		t.Errorf("⌫ with no crumb went to %s", m.here)
	}
	m.Close()
}

// The app's menu shows a screen with no crumb, and drops the chain: the app a link showed, shown again by the menu,
// has none either; so does a link to a screen that draws no crumb, as the help.
func TestTheAppsMenuShowsAScreenWithNoCrumb(t *testing.T) {
	m := onRepositoriesIncoming(t)
	if m = drive(m, shell.SwitchMsg{Screen: shell.Help, Crumb: true}); m.here != shell.Help || m.crumb != nil {
		t.Errorf("a link to the help, on %s, kept the crumb %+v", m.here, m.crumb)
	}
	m = drive(m, shell.SwitchMsg{Screen: shell.Repositories})
	m = drive(m, shell.JumpMsg{Screen: shell.Changes, To: "changes/#2"})
	m = drive(m, shell.SwitchMsg{Screen: shell.Issues})
	if crumbAt(m) >= 0 {
		t.Errorf("Issues has a crumb: %q", title(m))
	}
	m = drive(m, shell.SwitchMsg{Screen: shell.Changes})
	if crumbAt(m) >= 0 || drive(m, testkit.Press("backspace")).here != shell.Changes {
		t.Errorf("Changes, shown by the menu, has a crumb: %q", title(m))
	}
	m.Close()
}

// While the search has the keyboard, ⌫ erases, and goes nowhere.
func TestTheSearchKeepsBackspace(t *testing.T) {
	m := onRepositoriesIncoming(t)
	m = drive(m, shell.JumpMsg{Screen: shell.Changes, To: "changes/#2"})
	m = drive(m, typed("/ab")...)
	m = drive(m, testkit.Press("backspace"))
	if m.here != shell.Changes || !strings.Contains(title(m), shell.Find+" a") {
		t.Errorf("⌫ in the search, on %s: %q", m.here, title(m))
	}
	m.Close()
}

// Manage… leaves a crumb as a link does; a switch to another workspace, which makes the apps anew, drops the chain.
func TestManageLeavesACrumbAndAMoveDropsIt(t *testing.T) {
	list := `{"file":"/f","roots":[],"favorites":[],"workspaces":[{"name":"w","members":1,` +
		`"repos":["github.com/o/a"],"favorite":false}]}`
	m, _, _ := workspaced(t, map[string]string{"current.json": repoJSON, "w.json": groupJSON, "list.json": list})
	m = drive(m, shell.Pick(" manage")())
	if m.here != shell.Manage || !strings.Contains(title(m), " · "+shell.Back+" welcome") {
		t.Fatalf("Manage…, on %s: %q", m.here, title(m))
	}
	m = drive(m, testkit.Press("backspace"))
	if m.here != shell.Welcome || crumbAt(m) >= 0 {
		t.Fatalf("⌫ from Manage… went to %s: %q", m.here, title(m))
	}
	m = drive(m, shell.Pick(" manage")(), shell.PickMsg{Workspace: "w"})
	if m.here != shell.Manage || crumbAt(m) >= 0 {
		t.Errorf("after the move, on %s: %q", m.here, title(m))
	}
	m.Close()
}

// Every link on a page that ships with the kit's tutorials opens one of kittui's screens with the tutorials as its
// crumb, which ⌫ goes back to; and a link to a tab names one of its app's stages, one of Flow's as kit-flow's header
// orders them, and lands on that tab, as a click on it does: a jump to the app at the stage. Each is followed by step,
// as an app such as People ticks while it loads over the fake kit, which drive would wait on for ever.
func TestEveryLinkOnAShippedTutorialOpensAScreenThatLeadsBack(t *testing.T) {
	t.Setenv("FAKE_KIT_RECORD", filepath.Join(t.TempDir(), "record"))
	root := filepath.Join("..", "..", "..", "..", "kit")
	header, err := os.ReadFile(filepath.Join(root, "bin", "kit-flow"))
	if err != nil {
		t.Fatal(err)
	}
	var flowStages []string
	for _, l := range strings.Split(string(header), "\n") {
		if strings.HasPrefix(l, "#   issues → ") {
			flowStages = strings.Split(strings.TrimPrefix(l, "#   "), " → ")
			break
		}
	}
	pages, err := filepath.Glob(filepath.Join(root, "modules", "tutorials", "*.md"))
	if err != nil || len(pages) == 0 || len(flowStages) == 0 {
		t.Fatalf("no shipped tutorials, or no stages in kit-flow's header: %v", err)
	}
	seen := 0
	for _, page := range pages {
		md, err := os.ReadFile(page)
		if err != nil {
			t.Fatal(err)
		}
		_, links := markdown.Page(string(md), 1000)
		for _, l := range links {
			seen++
			where := filepath.Base(page) + ": " + l.To
			screen, stage, _ := strings.Cut(l.To, "/")
			m := drive(help(t), shell.SwitchMsg{Screen: shell.Tutorials})
			var msg tea.Msg = shell.SwitchMsg{Screen: shell.Screen(screen), Crumb: true}
			if screen == string(shell.Flow) && stage != "" && !slices.Contains(flowStages, stage) {
				t.Errorf("%s: Flow has no stage %s", where, stage)
			}
			if stage != "" {
				msg = shell.JumpMsg{Screen: shell.Screen(screen), To: stage + "/"}
			}
			if m = step(m, msg); m.here != shell.Screen(screen) || !strings.Contains(title(m), shell.Back+" tutorials") {
				t.Errorf("%s opened %s, the title %q, %s", where, m.here, title(m), m.refusal)
				m.Close()
				continue
			}
			if name, _ := m.made[m.here].(spotted).Spot(); msg != (shell.SwitchMsg{Screen: shell.Screen(screen),
				Crumb: true}) && !strings.HasSuffix(name, path.Base(stage)) {
				t.Errorf("%s stands on %q", where, name)
			}
			if m = step(m, testkit.Press("backspace")); m.here != shell.Tutorials {
				t.Errorf("%s: ⌫ went to %s", where, m.here)
			}
			m.Close()
		}
	}
	if seen == 0 {
		t.Error("no shipped tutorial has a link")
	}
}

// step gives the root one message, and runs none of the commands it starts: the screen shown, its crumb, and the tab
// a jump selects are the root's at once.
func step(m Model, msg tea.Msg) Model {
	next, _ := m.Update(msg)
	return next.(Model)
}
