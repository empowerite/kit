// Each forge's rate limit at the keys bar's right end: the scarcest, after the forge's glyph, dim, yellow, or red as
// it runs low; read after a stage loads and each minute after, and not again until Refresh All once kit cannot.
package flow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// quotas is GitHub's GraphQL, its scarcest, and REST, and a GitLab whose limit kit could not name.
const quotas = `[{"forge":"github","limits":[{"name":"graphql","limit":5000,"remaining":4871,"reset":1800000300},` +
	`{"name":"rest","limit":5000,"remaining":4990,"reset":1800000000}]},{"forge":"gitlab","limits":[]}]`

func TestEachForgesScarcestLimitIsDrawnAfterItsGlyph(t *testing.T) {
	now := time.Unix(1800000000, 0)
	reset := " ↻" + time.Unix(1800000300, 0).Format("15:04")
	for _, c := range []struct {
		name   string
		quotas []kit.Quota
		want   string
	}{
		{"plenty, dim", []kit.Quota{{Forge: "github", Limits: []kit.Limit{
			{Name: "graphql", Limit: 5000, Remaining: 4871, Reset: 1800000300},
			{Name: "rest", Limit: 5000, Remaining: 4990}}}}, style.Off(GitHub + " 4871/5000")},
		{"under a fifth, yellow, with its reset", []kit.Quota{{Forge: "github", Limits: []kit.Limit{
			{Name: "graphql", Limit: 5000, Remaining: 900, Reset: 1800000300}}}}, style.Warn(GitHub + " 900/5000" + reset)},
		{"spent, red", []kit.Quota{{Forge: "gitlab", Limits: []kit.Limit{
			{Name: "api", Limit: 2000, Remaining: 0, Reset: 1800000300}}}}, style.Refusal(GitLab + " 0/2000" + reset)},
		{"the scarcest by share, not by count", []kit.Quota{{Forge: "gitlab", Limits: []kit.Limit{
			{Name: "a", Limit: 5000, Remaining: 2000}, {Name: "b", Limit: 600, Remaining: 500}}}},
			style.Off(GitLab + " 2000/5000")},
		{"a forge kit has no glyph for, by name", []kit.Quota{{Forge: "gitea", Limits: []kit.Limit{
			{Name: "api", Limit: 10, Remaining: 9}}}}, style.Off("gitea 9/10")},
		{"a forge without a limit, nothing", []kit.Quota{{Forge: "gitlab"}}, ""},
		{"a reset already past, not said", []kit.Quota{{Forge: "github", Limits: []kit.Limit{
			{Name: "graphql", Limit: 5000, Remaining: 1, Reset: 1799999999}}}}, style.Warn(GitHub + " 1/5000")},
	} {
		if got := quotaText(c.quotas, now); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	two := quotaText([]kit.Quota{{Forge: "github", Limits: []kit.Limit{{Limit: 10, Remaining: 9}}},
		{Forge: "gitlab", Limits: []kit.Limit{{Limit: 10, Remaining: 8}}}}, now)
	if got := ansi.Strip(two); got != GitHub+" 9/10  "+GitLab+" 8/10" {
		t.Errorf("two forges: %q", got)
	}
}

// The screen reads the limits once its stages load, and draws them at the keys bar's right end.
func TestTheKeysBarEndsWithEachForgesLimit(t *testing.T) {
	t.Setenv("FAKE_FLOW_QUOTA", quotas)
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	last := lines(m)[len(lines(m))-1]
	if !strings.HasSuffix(strings.TrimRight(last, " "), GitHub+" 4871/5000") || strings.Contains(last, GitLab) {
		t.Errorf("the keys bar: %q", last)
	}
	if m.quoting || m.quotaOff || m.quotaSerial == 0 {
		t.Errorf("after the read: quoting %v, off %v, serial %d", m.quoting, m.quotaOff, m.quotaSerial)
	}
	// the minute's tick reads them again, and a tick an older read waits on does nothing
	if _, cmd := m.Update(quotaTickMsg{m.quotaSerial - 1}); cmd != nil {
		t.Error("a stale tick reads the limits again")
	}
	next, cmd := m.Update(quotaTickMsg{m.quotaSerial})
	if cmd == nil || !next.(Model).quoting {
		t.Error("the minute's tick does not read the limits again")
	}
	// a read that fails after one that did keeps what that one said, and goes on reading each minute
	n := next.(Model)
	after, tick := n.Update(quotaMsg{err: &kit.ExitError{Stderr: "❌ no"}})
	if a := after.(Model); len(a.quotas) != 2 || a.quotaOff || tick == nil {
		t.Errorf("after a failed read: %d quotas, off %v, tick %v", len(a.quotas), a.quotaOff, tick != nil)
	}
}

// A kit that cannot read the limits draws none, and is asked again only on Refresh All: once, however many stages load.
func TestAKitThatCannotReadTheLimitsIsAskedAgainOnRefreshAll(t *testing.T) {
	record := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_FLOW_CALLS", record)
	m := loaded(t, options(), tea.WindowSizeMsg{Width: 160, Height: 24})
	if !m.quotaOff || m.quoting || m.quotas != nil || strings.Contains(lines(m)[len(lines(m))-1], GitHub+" ") {
		t.Fatalf("without the limits: off %v, quoting %v, %v", m.quotaOff, m.quoting, m.quotas)
	}
	if next, _ := m.Update(loadedMsg{stage: "desk"}); next.(Model).quoting {
		t.Error("a load asks again, after kit could not read them")
	}
	asked := func() int {
		got, _ := os.ReadFile(record)
		return strings.Count(string(got), "flow --quota\n")
	}
	if n := asked(); n != 1 {
		t.Errorf("the stages' loads asked %d times", n)
	}
	drive(m, refreshAll...)
	if n := asked(); n != 2 {
		t.Errorf("after Refresh All, asked %d times", n)
	}
}
