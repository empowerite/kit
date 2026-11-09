// The 📚 repos tab, first in the picture: a row per member of the workspace, as kit flow repos --json gives it, with a
// count of each other tab's rows of the member, which opens that tab filtered to the member.
package flow

import (
	"strconv"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// repo is one member of the workspace, as kit flow repos --json gives it: its host and path, its primary remote's
// forge, its remotes, its checkout or none, the source that made it a member, and its desk's branch, with the commits
// it is ahead and behind, none for a member not cloned; Error is the desk's line where it failed.
type repo struct {
	Repository string  `json:"repository"`
	Forge      *string `json:"forge"`
	Remotes    []struct {
		Name  string `json:"name"`
		URL   string `json:"url"`
		Forge string `json:"forge"`
	} `json:"remotes"`
	Checkout *string `json:"checkout"`
	By       string  `json:"by"`
	Branch   *string `json:"branch"`
	Ahead    *int    `json:"ahead"`
	Behind   *int    `json:"behind"`
	Error    *string `json:"error"`
}

// countedStages names the stages a member's row counts, each a field of the repos tab.
var countedStages = []string{"backlog", "underway", "desk", "worktrees", "changes", "builds", "incoming", "releases"}

// repoRows gives each member: its short name, forge, remotes as pills, checkout, or dim not cloned, branch with what
// it is ahead and behind, each stage's count of its rows, and the source that made it a member. A member whose desk
// could not be read shows ❌ before its checkout, and kit's line when it is opened.
func (m Model) repoRows(s stage) []record {
	var out []record
	for _, r := range s.repos {
		var labels []table.Label
		for _, remote := range r.Remotes {
			labels = append(labels, table.Label{Name: remote.Name})
		}
		where := table.Value{Else: "not cloned"}
		if r.Checkout != nil {
			where = table.One(*r.Checkout)
			if r.Error != nil {
				where = table.One(Failed + " " + *r.Checkout)
			}
		}
		var branch, ahead table.Value
		if r.Branch != nil {
			c := checkout{Branch: *r.Branch}
			if r.Ahead != nil {
				c.Ahead = *r.Ahead
			}
			if r.Behind != nil {
				c.Behind = *r.Behind
			}
			branch = table.Value{Raw: []string{c.Branch}, Of: c}
			ahead = table.Value{Raw: []string{strconv.Itoa(c.Ahead)}, Of: c}
		}
		forge := ""
		if r.Forge != nil {
			forge = *r.Forge
		}
		values := map[string]table.Value{
			"repository": {Raw: []string{r.Repository}, Of: m.short(r.Repository)}, "forge": table.Some(forge),
			"remotes": table.Labelled(labels), "checkout": where, "branch": branch, "ahead": ahead,
			"by": table.Some(r.By),
		}
		for _, name := range countedStages {
			values[name] = m.countOf(name, r)
		}
		rec := record{row{Key: r.Repository}, values}
		if r.Error != nil {
			rec.Detail = [][]cell{span(style.Refusal(Failed + " " + *r.Error))}
		}
		out = append(out, rec)
	}
	return out
}

// countOf gives a member's count of a stage's rows: none for a stage not in the picture; a dim - for a member not
// cloned, whose rows step 4 reads; a dim … while the stage loads; ❌ where the stage refused, or failed for the member;
// else its rows, the asides apart, incoming's what the fetch brought. ❌ and the count lead to the stage, filtered to
// the member over a group.
func (m Model) countOf(name string, r repo) table.Value {
	s := m.find(name)
	switch {
	case s == nil:
		return table.Value{}
	case r.Checkout == nil:
		return table.Value{Else: "-"}
	case s.src.Loading:
		return table.Value{Else: "…"}
	}
	member := ""
	if m.runner.Group != "" {
		member = r.Repository
	}
	link := onlyTo(name, member)
	for _, f := range s.failed {
		if f.Member == member {
			return table.Value{Raw: []string{Failed}, Link: link}
		}
	}
	if s.refusal != "" {
		return table.Value{Raw: []string{Failed}, Link: link}
	}
	n := 0
	if name == "incoming" {
		for _, in := range s.ins {
			if in.Member != member {
				continue
			}
			for _, rm := range in.Remotes {
				n += len(rm.Commits) + len(rm.Branches.New) + len(rm.Branches.Updated) + len(rm.Branches.Gone) +
					len(rm.Tags)
			}
		}
	} else {
		for _, rec := range m.records(*s) {
			if rec.values == nil || rec.Aside {
				continue
			}
			if v := rec.values["member"]; member != "" && (len(v.Raw) == 0 || v.Raw[0] != member) {
				continue
			}
			n++
		}
	}
	return table.Value{Raw: []string{strconv.Itoa(n)}, Link: link}
}

// onlyTo gives the link that selects a stage filtered to a member's rows; for no member, the stage alone.
func onlyTo(stage, member string) string { return jumps + stage + "?" + member }

// only follows a link onlyTo gives: the stage selected, its rows filtered to the member's where it names one, the
// member's filter set aside where it does not.
func (m *Model) only(to string) bool {
	name, member, ok := strings.Cut(to, "?")
	if !ok || strings.Contains(name, "/") {
		return false
	}
	m.jump(name + "/")
	if m.stages[m.stage].name != name {
		return true
	}
	p := m.panel(m.stage)
	if member == "" {
		m.setPanel(p.FilteredBy("member", grid.Filter{}))
	} else {
		m.setPanel(p.FilteredBy("member", grid.Filter{Values: []string{member}}))
	}
	m.fit()
	return true
}
