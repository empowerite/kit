// kittui's root reads the workspace from kit once it starts, tells every screen made, and switches on a pick: a pick
// that moves the place the screens read remakes them there, all but the tutorials and the help.
package screens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
	"github.com/empowerite/kit/projects/tui/internal/welcome"
)

// The workspaces the tests' kit says: the repository github.com/o/a, at the directory checkout stands for, and the
// group w of it.
const (
	repoJSON = `{"kind":"repository","name":"github.com/o/a","member":"github.com/o/a","checkout":CHECKOUT,` +
		`"why":"repository","members":[],"groups":[{"name":"w","members":1}],"favorites":[]}`
	groupJSON = `{"kind":"group","name":"w","member":null,"checkout":null,"why":"given",` +
		`"members":[{"repository":"github.com/o/a","checkout":CHECKOUT}],"groups":[],"favorites":[]}`
)

// workspaced gives the root on the welcome, over the kit package's fake kit, at 120×30, kit workspaces answering with
// the files given, CHECKOUT in each the directory of a's checkout; the file kit's calls are recorded in; and that
// directory. No files leaves kit without kit workspaces.
func workspaced(t *testing.T, files map[string]string) (Model, string, string) {
	t.Helper()
	return workspacedOver(t, "", files)
}

// workspacedOver gives the root as workspaced does, its runner reading over the group given, as kittui started with
// --workspace does.
func workspacedOver(t *testing.T, group string, files map[string]string) (Model, string, string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	checkout, dir, record := t.TempDir(), t.TempDir(), filepath.Join(t.TempDir(), "record")
	t.Setenv("FAKE_KIT_RECORD", record)
	t.Setenv("FAKE_WORKSPACES", "")
	if files != nil {
		t.Setenv("FAKE_WORKSPACES", dir)
	}
	for name, text := range files {
		text = strings.ReplaceAll(text, "CHECKOUT", `"`+checkout+`"`)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := testkit.Runner(t, "kit")
	r.Group = group
	root := New(r, shell.Welcome, welcome.Open(r, kit.Workspace{}))
	m, _ := testkit.Drive(root, nil, testkit.Start(root, tea.WindowSizeMsg{Width: 120, Height: 30})...)
	return m, record, checkout
}

// title gives the root's title line, without its escapes.
func title(m Model) string { return strings.Split(ansi.Strip(m.View().Content), "\n")[0] }

// drive sends the messages to the root.
func drive(m Model, msgs ...tea.Msg) Model {
	m, _ = testkit.Drive(m, nil, msgs...)
	return m
}

func TestTheWorkspaceReachesEveryScreenMade(t *testing.T) {
	m, _, _ := workspaced(t, map[string]string{"current.json": repoJSON})
	if got := title(m); !strings.Contains(got, "- o/a") {
		t.Errorf("the welcome, made first: %q", got)
	}
	m = drive(m, shell.SwitchMsg{Screen: shell.Help}, shell.SwitchMsg{Screen: shell.Tutorials})
	if got := m.made[shell.Welcome].(welcome.Model).Books().Workspace.Name; got != "github.com/o/a" {
		t.Errorf("the welcome's control: %q", got)
	}
}

func TestAKitWithoutWorkspacesLeavesEveryScreenOpen(t *testing.T) {
	m, _, _ := workspaced(t, nil)
	lines := strings.Split(ansi.Strip(m.View().Content), "\n")
	if last := lines[len(lines)-1]; strings.Contains(last, "❌") || !strings.Contains(last, "menu") {
		t.Errorf("the last line: %q", last)
	}
	books := m.made[shell.Welcome].(welcome.Model).Books()
	if items := books.Items(); items[0].Glyph != "❌" || !strings.Contains(items[0].Name, "workspaces") {
		t.Errorf("the control's first line: %+v", items[0])
	}
}

func TestPickingAGroupRemembersItAndRemakesTheScreens(t *testing.T) {
	m, record, _ := workspaced(t, map[string]string{"current.json": repoJSON, "w.json": groupJSON})
	m = drive(m, shell.SwitchMsg{Screen: shell.Help})
	m = drive(m, typed("com")...)
	m = drive(m, shell.SwitchMsg{Screen: shell.Welcome})
	m = drive(m, typed("/xyz")...)
	if !strings.Contains(title(m), "xyz") {
		t.Fatalf("the search typed: %q", title(m))
	}
	m = drive(m, shell.PickMsg{Workspace: "w"})
	if got, _ := os.ReadFile(record); !strings.Contains(string(got), "use w") {
		t.Errorf("kit was not told: %q", got)
	}
	if got := title(m); !strings.Contains(got, "- w 1") || strings.Contains(got, "xyz") {
		t.Errorf("the welcome is not made anew on w: %q", got)
	}
	m = drive(m, shell.SwitchMsg{Screen: shell.Help})
	if v := ansi.Strip(m.View().Content); !strings.Contains(v, shell.Find+" com") {
		t.Errorf("the help was made anew:\n%s", v)
	}
}

func TestOnlyTheLatestPickLands(t *testing.T) {
	m, _, _ := workspaced(t, map[string]string{"current.json": repoJSON})
	m.asked = 2
	m = drive(m, workspaceMsg{gen: 1, ws: kit.Workspace{Kind: "group", Name: "w"}})
	if m.workspace.Name != "github.com/o/a" {
		t.Errorf("an earlier pick landed: %+v", m.workspace)
	}
}

func TestTheFirstLoadInsideARepositoryRemakesNothing(t *testing.T) {
	m, _, _ := workspaced(t, map[string]string{"current.json": repoJSON})
	m = drive(m, typed("/xyz")...)
	m = drive(m, workspaceMsg{gen: 0, ws: kit.Workspace{Kind: "repository", Name: "github.com/o/a",
		Member: "github.com/o/a"}})
	if !strings.Contains(title(m), "xyz") {
		t.Errorf("the welcome was made anew: %q", title(m))
	}
	// started where kit says the last group is, the screens are made again over it
	m = drive(m, workspaceMsg{gen: 0, ws: kit.Workspace{Kind: "group", Name: "w"}})
	if strings.Contains(title(m), "xyz") || !strings.Contains(title(m), "- w") {
		t.Errorf("a group first: %q", title(m))
	}
}

func TestARepositoryPickedIsWhereLaterScreensRead(t *testing.T) {
	m, _, checkout := workspaced(t, map[string]string{"current.json": repoJSON, "github.com_o_a.json": repoJSON,
		"w.json": groupJSON})
	// kit said a's checkout at the start, which the runner reads already
	home := m.home
	if m.runner.Dir != checkout || home == checkout {
		t.Errorf("at the start, the runner reads %q, home %q", m.runner.Dir, home)
	}
	m = drive(m, shell.PickMsg{Workspace: "w"}, shell.PickMsg{Workspace: "github.com/o/a"})
	if m.runner.Dir != checkout {
		t.Errorf("the runner reads %q, want %q", m.runner.Dir, checkout)
	}
	m = drive(m, shell.PickMsg{Workspace: "w"})
	if m.runner.Dir != home {
		t.Errorf("over a group, the runner reads %q, want %q", m.runner.Dir, home)
	}
}

func TestARefusedPickKeepsTheWorkspaceAndSaysKitsLine(t *testing.T) {
	m, _, _ := workspaced(t, map[string]string{"current.json": repoJSON})
	m = drive(m, shell.PickMsg{Workspace: "github.com/o/gone"})
	books := m.made[shell.Welcome].(welcome.Model).Books()
	if m.workspace.Name != "github.com/o/a" || books.Workspace.Name != "github.com/o/a" || books.Refusal == "" {
		t.Errorf("a refused pick: %+v, %+v", m.workspace, books)
	}
}

func TestAGroupKitWillNotKeepIsNotSwitchedTo(t *testing.T) {
	m, record, _ := workspaced(t, map[string]string{"current.json": repoJSON, "w.json": groupJSON})
	t.Setenv("FAKE_KIT_REFUSE", "workspaces")
	m = drive(m, shell.PickMsg{Workspace: "w"})
	books := m.made[shell.Welcome].(welcome.Model).Books()
	if got, _ := os.ReadFile(record); m.workspace.Name != "github.com/o/a" || !strings.Contains(books.Refusal,
		"refuses") || strings.Contains(string(got), "use w") {
		t.Errorf("a refused --use: %+v, %q", m.workspace, books.Refusal)
	}
}

func TestAScreenThatCannotOpenOverTheNewWorkspaceGivesWayToTheWelcome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(testkit.Runner(t, "kit"), shell.Issues, told{})
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 30})
	// kit cannot run where the picture would open now, so it cannot be made there
	m.runner.Kit = filepath.Join(t.TempDir(), "no-kit")
	m = drive(m, workspaceMsg{gen: 0, ws: kit.Workspace{Kind: "group", Name: "w"}})
	if _, ok := m.Shown().(welcome.Model); !ok || m.here != shell.Welcome {
		t.Fatalf("shown: %T on %s", m.Shown(), m.here)
	}
	lines := strings.Split(ansi.Strip(m.View().Content), "\n")
	if last := lines[len(lines)-1]; !strings.Contains(last, "❌") {
		t.Errorf("kit's line is not said: %q", last)
	}
}

func TestAnAnswerForAScreenMadeAgainIsDropped(t *testing.T) {
	m, _, _ := workspaced(t, map[string]string{"current.json": repoJSON, "w.json": groupJSON})
	was := m.epochs[shell.Welcome]
	m = drive(m, shell.PickMsg{Workspace: "w"})
	if m.epochs[shell.Welcome] == was {
		t.Fatalf("the welcome made again keeps its epoch %d", was)
	}
	// an answer the welcome closed started reaches nothing; the one made again takes its own
	m = drive(m, tagged{to: shell.Welcome, epoch: was, msg: tea.WindowSizeMsg{Width: 70, Height: 20}})
	if n := len(strings.Split(m.made[shell.Welcome].View().Content, "\n")); n != 30 {
		t.Errorf("an old answer reached the welcome: %d lines", n)
	}
	own := tagged{to: shell.Welcome, epoch: m.epochs[shell.Welcome], msg: tea.WindowSizeMsg{Width: 70, Height: 20}}
	m = drive(m, own)
	if n := len(strings.Split(m.made[shell.Welcome].View().Content, "\n")); n != 20 {
		t.Errorf("the welcome's own answer: %d lines", n)
	}
}

func TestTheRunnerFollowsTheGroupKitSays(t *testing.T) {
	m, _, checkout := workspaced(t, map[string]string{"current.json": repoJSON, "github.com_o_a.json": repoJSON,
		"w.json": groupJSON})
	if m.runner.Group != "" {
		t.Errorf("inside a repository, the runner spans %q", m.runner.Group)
	}
	m = drive(m, shell.PickMsg{Workspace: "w"})
	if m.runner.Group != "w" || m.runner.NoGroup {
		t.Errorf("over w, the runner spans %q, no group %v", m.runner.Group, m.runner.NoGroup)
	}
	m = drive(m, shell.PickMsg{Workspace: "github.com/o/a"})
	if m.runner.Group != "" || m.runner.Dir != checkout || !m.runner.NoGroup {
		t.Errorf("back on a, the runner spans %q in %q, no group %v", m.runner.Group, m.runner.Dir, m.runner.NoGroup)
	}
}

func TestAGroupGivenAtStartIsAskedAndNotMadeTwice(t *testing.T) {
	// kit has no current workspace here: only an ask for w is answered
	m, _, _ := workspacedOver(t, "w", map[string]string{"w.json": groupJSON})
	if m.workspace.Name != "w" || m.runner.Group != "w" {
		t.Fatalf("the ask at the start: %+v, the runner over %q", m.workspace, m.runner.Group)
	}
	m = drive(m, typed("/xyz")...)
	m = drive(m, workspaceMsg{gen: 0, ws: kit.Workspace{Kind: "group", Name: "w"}})
	if !strings.Contains(title(m), "xyz") {
		t.Errorf("the screen made over w was made again: %q", title(m))
	}
}

// open drives the root's Open… dialog: open, what is typed cleared, the folder typed, and enter.
func open(m Model, folder string) Model {
	m = drive(m, shell.OpenMsg{}, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	m = drive(m, typed(folder)...)
	return drive(m, tea.KeyPressMsg{Code: tea.KeyEnter})
}

func TestOpenOpensARepositoryByItsFolderAndKeepsItAmongTheRecent(t *testing.T) {
	m, record, _ := workspaced(t, map[string]string{"current.json": repoJSON})
	far := t.TempDir()
	text := `{"kind":"repository","name":"` + far + `","member":null,"checkout":"` + far + `","why":"given",` +
		`"members":[],"groups":[],"favorites":[]}`
	if err := os.WriteFile(filepath.Join(os.Getenv("FAKE_WORKSPACES"), strings.ReplaceAll(far, "/", "_")+".json"),
		[]byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	m = drive(m, shell.OpenMsg{})
	if v := ansi.Strip(m.View().Content); !m.opener.Open || !strings.Contains(v, "Open a repository") ||
		!strings.Contains(v, "esc close") {
		t.Fatalf("Open… opened no dialog:\n%s", v)
	}
	m = open(m, far)
	if got, _ := os.ReadFile(record); !strings.Contains(string(got), "use "+far) {
		t.Errorf("kit was not told to keep it: %q", got)
	}
	if m.opener.Open || m.runner.Dir != far || !strings.Contains(title(m), "- "+filepath.Base(far)) {
		t.Errorf("not opened: open %v, the runner in %q, %q", m.opener.Open, m.runner.Dir, title(m))
	}
}

func TestOpenSaysKitsLineInTheDialogWhenKitRefuses(t *testing.T) {
	m, _, _ := workspaced(t, map[string]string{"current.json": repoJSON})
	t.Setenv("FAKE_KIT_REFUSE", "workspaces")
	m = open(m, t.TempDir())
	if v := ansi.Strip(m.View().Content); !m.opener.Open || !strings.Contains(v, "refuses for the test") {
		t.Errorf("the refusal is not in the dialog:\n%s", v)
	}
	if m.workspace.Name != "github.com/o/a" {
		t.Errorf("the workspace moved: %+v", m.workspace)
	}
	if m = drive(m, tea.KeyPressMsg{Code: tea.KeyEscape}); m.opener.Open {
		t.Error("esc leaves the dialog open")
	}
}

// The repository kittui starts in is kept among the recent, as one opened by its folder is, so the workspace menu
// offers it again once another is switched to.
func TestTheRepositoryStartedInIsKeptAmongTheRecent(t *testing.T) {
	_, record, checkout := workspaced(t, map[string]string{"current.json": repoJSON})
	if got, _ := os.ReadFile(record); !strings.Contains(string(got), "use "+checkout) {
		t.Errorf("kit was not told to keep it: %q", got)
	}
}

func TestARecentRepositoryPickedIsKeptFirstAgain(t *testing.T) {
	m, record, _ := workspaced(t, map[string]string{"current.json": repoJSON})
	m = drive(m, shell.PickMsg{Workspace: "/srv/far"})
	if got, _ := os.ReadFile(record); !strings.Contains(string(got), "use /srv/far") {
		t.Errorf("kit was not told: %q", got)
	}
}

func TestManageOpensTheWorkspacesScreenWhichAMoveLeavesAsItWas(t *testing.T) {
	list := `{"file":"/f","roots":[],"favorites":[],"workspaces":[{"name":"w","members":1,` +
		`"repos":["github.com/o/a"],"favorite":false}]}`
	m, _, _ := workspaced(t, map[string]string{"current.json": repoJSON, "w.json": groupJSON, "list.json": list})
	m = drive(m, shell.SwitchMsg{Screen: shell.Manage})
	if v := ansi.Strip(m.View().Content); m.here != shell.Manage || !strings.Contains(v, "kit workspaces") ||
		!strings.Contains(v, "github.com/o/a") {
		t.Fatalf("Manage… opened %s:\n%s", m.here, v)
	}
	was := m.epochs[shell.Manage]
	m = drive(m, shell.PickMsg{Workspace: "w"})
	if m.epochs[shell.Manage] != was || m.here != shell.Manage {
		t.Errorf("the workspaces' screen was made again on a move: %d, on %s", m.epochs[shell.Manage], m.here)
	}
}

func TestAGroupRenamedIsFollowedAndOneDeletedGivesWayToWhereKitStarts(t *testing.T) {
	core := strings.ReplaceAll(groupJSON, `"name":"w"`, `"name":"core"`)
	m, record, _ := workspaced(t, map[string]string{"current.json": repoJSON, "w.json": groupJSON,
		"core.json": core})
	m = drive(m, shell.PickMsg{Workspace: "w"})
	m = drive(m, shell.ChangedMsg{From: "w", To: "core"})
	if got, _ := os.ReadFile(record); m.workspace.Name != "core" || !strings.Contains(string(got), "use core") {
		t.Errorf("the rename was not followed: %+v, %q", m.workspace, got)
	}
	m = drive(m, shell.ChangedMsg{From: "core"})
	if m.workspace.Name != "github.com/o/a" {
		t.Errorf("the delete did not give way to where kit starts: %+v", m.workspace)
	}
	// a change to another group asks again for the one shown
	m = drive(m, shell.PickMsg{Workspace: "w"}, shell.ChangedMsg{From: "x", To: "y"})
	if m.workspace.Name != "w" {
		t.Errorf("another group's change moved the workspace: %+v", m.workspace)
	}
}

// The Open… dialog takes plain clicks over the screen: one on a folder completes to it, one on Open asks kit for it;
// an answer to an earlier ask is dropped, the dialog left open.
func TestOpenIsDoneByPlainClicksOverTheScreen(t *testing.T) {
	m, record, _ := workspaced(t, map[string]string{"current.json": repoJSON})
	parent := t.TempDir()
	if err := os.Mkdir(filepath.Join(parent, "far"), 0o755); err != nil {
		t.Fatal(err)
	}
	m = drive(m, shell.OpenMsg{}, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	m = drive(m, typed(parent+"/")...)
	at := func(text string) (int, int) {
		for y, l := range strings.Split(ansi.Strip(m.View().Content), "\n") {
			if k := strings.Index(l, text); k >= 0 {
				return ansi.StringWidth(l[:k]), y
			}
		}
		t.Fatalf("no line holds %q:\n%s", text, ansi.Strip(m.View().Content))
		return 0, 0
	}
	if m = drive(m, testkit.Click(at(shell.Folder+" far/"))); !m.opener.Open || m.opener.Typed != parent+"/far/" {
		t.Fatalf("a click on far/: %+v", m.opener)
	}
	m = drive(m, testkit.Click(at(shell.Opening+" Open")))
	if got, _ := os.ReadFile(record); !strings.Contains(string(got), "use "+filepath.Join(parent, "far")) {
		t.Errorf("a click on Open did not ask kit: %q", got)
	}
	m = drive(m, shell.OpenMsg{})
	m.asked = 5
	if m = drive(m, openedMsg{gen: 4, ws: kit.Workspace{Kind: "group", Name: "w"}}); !m.opener.Open ||
		m.workspace.Name != "github.com/o/a" {
		t.Errorf("an earlier answer landed: open %v, %+v", m.opener.Open, m.workspace)
	}
}

// While the Open… dialog is open, a turn of the mouse wheel scrolls its folders.
func TestTheWheelScrollsTheOpenDialogsFolders(t *testing.T) {
	m, _, _ := workspaced(t, map[string]string{"current.json": repoJSON})
	parent := t.TempDir()
	for k := range 20 {
		if err := os.Mkdir(filepath.Join(parent, "f"+string(rune('a'+k))), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	m = drive(m, shell.OpenMsg{}, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	m = drive(m, typed(parent+"/")...)
	m = drive(m, tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if v := ansi.Strip(m.View().Content); m.opener.Top == 0 || !strings.Contains(v, "above") ||
		strings.Contains(v, " fa/") {
		t.Errorf("the wheel scrolled nothing: top %d\n%s", m.opener.Top, v)
	}
}

// ⌃F in the Open… dialog walks the folder typed, and the clones it finds stand in the dialog, each opened by a click.
func TestTheOpenDialogFindsAndOpensTheClonesUnderAFolder(t *testing.T) {
	m, record, _ := workspaced(t, map[string]string{"current.json": repoJSON})
	parent := t.TempDir()
	if err := os.MkdirAll(filepath.Join(parent, "far", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	m = drive(m, shell.OpenMsg{}, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	m = drive(m, typed(parent+"/")...)
	m = drive(m, tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, shell.Clone+" far") {
		t.Fatalf("no clone found:\n%s", view)
	}
	for y, l := range strings.Split(view, "\n") {
		if k := strings.Index(l, shell.Clone+" far"); k >= 0 {
			m = drive(m, testkit.Click(ansi.StringWidth(l[:k]), y))
			break
		}
	}
	if got, _ := os.ReadFile(record); !strings.Contains(string(got), "use "+filepath.Join(parent, "far")) {
		t.Errorf("a click on the clone did not ask kit: %q", got)
	}
}
