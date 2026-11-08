// Welcome over a group: kit welcome --checks --json runs in every member's checkout, at once, and the answers are
// composed into one list, a check every member reports alike said once, one that differs said per member.
package welcome

import (
	"context"
	"strings"
	"sync"

	"github.com/empowerite/kit/projects/tui/internal/kit"
)

// refusedID is the id of the failed check a member's refusal is.
const refusedID = "kit"

// run is one member's answer: its host/path, its checkout, and its checks, or why kit gave none.
type run struct {
	member, dir string
	checks      []kit.Check
	err         error
}

// compose gives the checks of every member's run as one list, in the order each id first appears. A check every
// member that answered reports alike, its result, line, and fix the same, is said once, with no member, its fix
// running in every member's checkout in turn; any other is said once per member that reports it, in the members'
// order, each with its member and checkout. A run kit refused is one failed check of its own, kit, with kit's line and no fix.
func compose(runs []run) []kit.Check {
	var order []string
	seen := map[string][]kit.Check{}
	answered := 0
	add := func(c kit.Check) {
		if _, ok := seen[c.ID]; !ok {
			order = append(order, c.ID)
		}
		seen[c.ID] = append(seen[c.ID], c)
	}
	for _, r := range runs {
		if r.err != nil {
			add(kit.Check{ID: refusedID, Line: refusalOf(r.err), Member: r.member, Dirs: []string{r.dir}})
			continue
		}
		answered++
		for _, c := range r.checks {
			c.Member, c.Dirs = r.member, []string{r.dir}
			add(c)
		}
	}
	var out []kit.Check
	for _, id := range order {
		same := seen[id]
		if alike(same, answered) {
			c := same[0]
			c.Member, c.Dirs = "", nil
			for _, each := range same {
				c.Dirs = append(c.Dirs, each.Dirs...)
			}
			out = append(out, c)
			continue
		}
		out = append(out, same...)
	}
	return out
}

// alike says whether the checks of one id are one check said once: one from every member that answered, kit's own
// refusals never, each with the first's result, line, and fix.
func alike(same []kit.Check, answered int) bool {
	if len(same) != answered || same[0].ID == refusedID {
		return false
	}
	for _, c := range same[1:] {
		if c.OK != same[0].OK || c.Line != same[0].Line || c.Fix != same[0].Fix {
			return false
		}
	}
	return true
}

// checksOver gives the read of the checks over the workspace: over a group, every cloned member's, at once, composed,
// or the first member's refusal when every one refused; over anything else, the checks where kittui runs.
func checksOver(r kit.Runner, ws kit.Workspace) func(context.Context, bool) ([]kit.Check, error) {
	if ws.Kind != "group" {
		return func(ctx context.Context, _ bool) ([]kit.Check, error) { return r.WelcomeChecks(ctx) }
	}
	var cloned []kit.WorkspaceMember
	for _, m := range ws.Members {
		if m.Checkout != "" {
			cloned = append(cloned, m)
		}
	}
	return func(ctx context.Context, _ bool) ([]kit.Check, error) {
		runs := make([]run, len(cloned))
		var wg sync.WaitGroup
		for i, m := range cloned {
			wg.Add(1)
			go func() {
				defer wg.Done()
				in := r
				in.Dir = m.Checkout
				checks, err := in.WelcomeChecks(ctx)
				runs[i] = run{member: m.Repository, dir: m.Checkout, checks: checks, err: err}
			}()
		}
		wg.Wait()
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		refused := 0
		for _, r := range runs {
			if r.err != nil {
				refused++
			}
		}
		if refused > 0 && refused == len(runs) {
			return nil, runs[0].err
		}
		return compose(runs), nil
	}
}

// uncloned gives the owner/repo of each of a group's members with no checkout, in order.
func uncloned(ws kit.Workspace) []string {
	var out []string
	for _, m := range ws.Members {
		if m.Checkout == "" {
			out = append(out, short(m.Repository, 2))
		}
	}
	return out
}

// short gives the last so many segments of a host/path.
func short(member string, n int) string {
	segments := strings.Split(member, "/")
	return strings.Join(segments[max(0, len(segments)-n):], "/")
}
