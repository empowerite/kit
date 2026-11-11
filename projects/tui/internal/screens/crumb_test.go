// A link to another app leaves a crumb on the title line of the app it shows, the way back to where it was followed,
// the view as it was left and its own crumb with it; only the app's menu shows a screen with no crumb.
package screens

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

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
// has none either.
func TestTheAppsMenuShowsAScreenWithNoCrumb(t *testing.T) {
	m := onRepositoriesIncoming(t)
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
