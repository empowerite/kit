// Over a group, Welcome runs the checks in every member's checkout and shows them together: a check every member
// reports alike once, one that differs once per member, a refusal as its member's failed check, and the members not
// cloned named in the intro. A fix runs in its member's checkout.
package welcome

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

func check(id string, ok bool, line string) kit.Check {
	return kit.Check{ID: id, Step: "toolchains", OK: ok, Line: line, Runs: true}
}

func TestComposeSaysACheckAlikeInEveryMemberOnce(t *testing.T) {
	got := compose([]run{
		{member: "github.com/o/a", dir: "/a", checks: []kit.Check{check("jq", false, "jq is not installed")}},
		{member: "github.com/o/b", dir: "/b", checks: []kit.Check{check("jq", false, "jq is not installed")}},
	})
	if len(got) != 1 || got[0].Member != "" || strings.Join(got[0].Dirs, " ") != "/a /b" || got[0].Key() != "jq" {
		t.Errorf("composed: %+v", got)
	}
}

func TestComposeShowsACheckThatDiffersPerMember(t *testing.T) {
	got := compose([]run{
		{member: "github.com/o/a", dir: "/a", checks: []kit.Check{check("bash", true, "bash"),
			check("toolchain-project", true, "one toolchain")}},
		{member: "github.com/o/b", dir: "/b", checks: []kit.Check{check("toolchain-project", false, "several"),
			check("bash", true, "bash"), check("only-b", false, "b alone")}},
	})
	var keys []string
	for _, c := range got {
		keys = append(keys, c.Key()+"@"+strings.Join(c.Dirs, " "))
	}
	want := "bash@/a /b|github.com/o/a toolchain-project@/a|github.com/o/b toolchain-project@/b|" +
		"github.com/o/b only-b@/b"
	if strings.Join(keys, "|") != want {
		t.Errorf("composed: %s", strings.Join(keys, "|"))
	}
}

func TestComposeTurnsARefusalIntoItsMembersFailedCheck(t *testing.T) {
	got := compose([]run{
		{member: "github.com/o/a", dir: "/a", checks: []kit.Check{check("jq", true, "jq")}},
		{member: "github.com/o/b", dir: "/b", err: &kit.ExitError{Code: 2, Stderr: "❌ kit welcome refuses here\n"}},
	})
	if len(got) != 2 || got[0].Member != "" || got[1].Member != "github.com/o/b" || got[1].ID != "kit" || got[1].OK ||
		got[1].Line != "kit welcome refuses here" || got[1].Fix != "" {
		t.Errorf("composed: %+v", got)
	}
	// a single member that answered says each check once, with its checkout
	one := []run{{member: "github.com/o/a", dir: "/a", checks: []kit.Check{check("jq", true, "jq")}}}
	if got := compose(one); len(got) != 1 || got[0].Member != "" || strings.Join(got[0].Dirs, " ") != "/a" {
		t.Errorf("one member: %+v", got)
	}
}

// group opens the screen at 120×30 over the group of a and b, each a checkout of its own, and c, not cloned; a's
// checks are the fixture's but toolchain-project, which fails there another way. It gives the screen, what its fixes
// ran, and a's and b's checkouts.
func group(t *testing.T, msgs ...tea.Msg) (Model, *ran, string, string) {
	t.Helper()
	m, r := fake(t)
	a, b := t.TempDir(), t.TempDir()
	fixture, err := os.ReadFile(filepath.Join("testdata", "checks.json"))
	if err != nil {
		t.Fatal(err)
	}
	mine := strings.Replace(string(fixture), "several toolchains are usable for the project",
		"a's toolchain differs", 1)
	if err := os.WriteFile(filepath.Join(a, "checks.json"), []byte(mine), 0o644); err != nil {
		t.Fatal(err)
	}
	ws := kit.Workspace{Kind: "group", Name: "w", Members: []kit.WorkspaceMember{{Repository: "github.com/o/a",
		Checkout: a}, {Repository: "github.com/o/b", Checkout: b}, {Repository: "github.com/o/c"}}}
	g := Open(testkit.Runner(t, "welcome"), ws)
	g.exec = m.exec
	return drive(g, append(testkit.Start(g, at120), msgs...)...), r, a, b
}

func TestAGroupsChecksDrawTogetherWithTheUnclonedNamed(t *testing.T) {
	m, _, _, _ := group(t)
	v := view(m)
	if !strings.Contains(v, "not cloned: o/c") || strings.Contains(v, "☁") || !strings.Contains(v, "- w 3") {
		t.Errorf("the intro:\n%s", v)
	}
	m = drive(m, press('2'))
	v = view(m)
	if !strings.Contains(v, "a's toolchain differs: mise, brew; choose one · a") ||
		!strings.Contains(v, "usable for the project: mise, brew; choose one · b") {
		t.Errorf("the toolchains step:\n%s", v)
	}
	// jq fails alike in both, so it shows once, with no member
	if strings.Count(v, "jq is not installed") != 1 || strings.Contains(v, "jq is not installed ·") {
		t.Errorf("jq on the toolchains step:\n%s", v)
	}
}

// onMember moves the cursor, in the pane, down to the check with the id of the member, and says whether it found it.
func onMember(m Model, id, member string) (Model, bool) {
	for range 8 {
		if c := m.checks[m.cursor]; c.ID == id && c.Member == member {
			return m, true
		}
		m = drive(m, down)
	}
	return m, false
}

func TestAFixRunsInItsMembersCheckout(t *testing.T) {
	m, r, a, b := group(t, press('2'), down)
	m, found := onMember(m, "toolchain-project", "github.com/o/b")
	if !found {
		t.Fatalf("no b's toolchain-project:\n%s", view(m))
	}
	m = drive(m, press('l'))
	v := view(m)
	if !strings.Contains(v, "choose one · b · later") || strings.Contains(v, "choose one · a · later") {
		t.Errorf("b's left for later, a's not:\n%s", v)
	}
	// later moved the cursor down past a's, so home brings it back to the step's first row
	m, found = onMember(drive(m, press(tea.KeyHome)), "toolchain-project", "github.com/o/a")
	if !found {
		t.Fatalf("no a's toolchain-project:\n%s", view(m))
	}
	drive(m, press('n'))
	if len(r.dirs) != 1 || r.dirs[0] != a || r.dirs[0] == b {
		t.Errorf("the fix ran in %v, want %s", r.dirs, a)
	}
}

func TestAGroupWhoseEveryMemberRefusesSaysKitsLine(t *testing.T) {
	t.Setenv("FAKE_WELCOME_REFUSE", "1")
	m, _, _, _ := group(t)
	if v := view(m); !strings.Contains(v, "kit welcome refuses for the test") {
		t.Errorf("every member refused:\n%s", v)
	}
}

func TestAGroupsChecksEndWithTheScreen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ws := kit.Workspace{Kind: "group", Members: []kit.WorkspaceMember{{Repository: "github.com/o/a",
		Checkout: t.TempDir()}}}
	if _, err := checksOver(testkit.Runner(t, "welcome"), ws)(ctx, false); !errors.Is(err, context.Canceled) {
		t.Errorf("ended: %v", err)
	}
}

func TestAFixSaidOnceRunsInEveryMember(t *testing.T) {
	m, r, a, b := group(t, press('2'), down)
	m, found := onMember(m, "jq", "")
	if !found {
		t.Fatalf("no jq said once:\n%s", view(m))
	}
	drive(m, press('n'))
	if strings.Join(r.dirs, " ") != a+" "+b {
		t.Errorf("the fix ran in %v, want %s then %s", r.dirs, a, b)
	}
}
