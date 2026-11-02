// Each stage's JSON, as kit status <stage> --json prints it, and the rows a stage draws from it: what a row shows
// closed, as cells laid out in columns, and what it opens to.
package status

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/empowerite/kit/tui/internal/style"
)

// issue is one of backlog's, and one a pull request closes. Age is seconds, or nothing for an issue without a time;
// Branch is the one started for it on origin, or empty.
type issue struct {
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	Assignees []string `json:"assignees"`
	Age       *int64   `json:"age"`
	Branch    string   `json:"branch"`
	URL       string   `json:"url"`
}

// commit is one commit: its short hash and its subject.
type commit struct {
	SHA     string `json:"sha"`
	Subject string `json:"subject"`
}

// incoming is what the fetch brought: the commits new on the default branch, the branches new, updated, and gone on
// origin, and the tags new; with --no-fetch, Fetched alone, false.
type incoming struct {
	Fetched  bool     `json:"fetched"`
	Commits  []commit `json:"commits"`
	Branches struct {
		New     []string `json:"new"`
		Updated []string `json:"updated"`
		Gone    []string `json:"gone"`
	} `json:"branches"`
	Tags []string `json:"tags"`
}

// worktrees is every linked worktree, and the branches whose upstream is gone. The desk is a stage of its own.
type worktrees struct {
	Worktrees    []checkout `json:"worktrees"`
	GoneBranches []string   `json:"goneBranches"`
}

// checkout is one checkout: its branch, its lines against the default branch, the commits ahead of it, newest first,
// and its uncommitted paths.
type checkout struct {
	Path        string   `json:"path"`
	Relative    string   `json:"relative"`
	Branch      string   `json:"branch"`
	Detached    bool     `json:"detached"`
	Ahead       int      `json:"ahead"`
	Behind      int      `json:"behind"`
	Added       int      `json:"added"`
	Removed     int      `json:"removed"`
	Commits     []commit `json:"commits"`
	Uncommitted int      `json:"uncommitted"`
	Changes     []struct {
		Status string `json:"status"`
		Path   string `json:"path"`
	} `json:"changes"`
}

// pull is one open pull request, with the issues it closes.
type pull struct {
	Number     int     `json:"number"`
	Ref        string  `json:"ref"`
	Title      string  `json:"title"`
	Author     string  `json:"author"`
	State      string  `json:"state"`
	Age        int64   `json:"age"`
	CreatedAt  string  `json:"createdAt"`
	Checks     *checks `json:"checks"`
	Unresolved int     `json:"unresolved"`
	Branch     string  `json:"branch"`
	Closes     []issue `json:"closes"`
}

// checks is a pull request's checks: SUCCESS, FAILURE, or PENDING, the counts, the failing check's name, and how long
// the slowest took, in seconds, when known.
type checks struct {
	State   string `json:"state"`
	Passed  int    `json:"passed"`
	Total   int    `json:"total"`
	Failing string `json:"failing"`
	Took    *int64 `json:"took"`
}

// release is one release on the forge.
type release struct {
	Version string `json:"version"`
	Tag     string `json:"tag"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Latest  bool   `json:"latest"`
	At      string `json:"at"`
	Age     int64  `json:"age"`
	URL     string `json:"url"`
}

// cell is one column of a row: its text, styled already when it is dim; right is a number, right-aligned in its
// column; fill is the one cell that takes the width left, clipped.
type cell struct {
	text  string
	right bool
	fill  bool
}

// row is one row of a stage: its key, which survives a refresh; its cells; and the lines it opens to, each a row of
// cells again, none for a leaf.
type row struct {
	key    string
	cells  []cell
	detail [][]cell
}

// text gives a plain cell.
func text(s string) cell { return cell{text: s} }

// dim gives a dim cell; an empty text stays empty, so its column can be left out.
func dim(s string) cell {
	if s == "" {
		return cell{}
	}
	return cell{text: style.Off(s)}
}

// number gives a right-aligned cell.
func number(s string) cell { return cell{text: s, right: true} }

// dimNumber gives a dim, right-aligned cell.
func dimNumber(s string) cell {
	c := dim(s)
	c.right = true
	return c
}

// fill gives the cell that takes the width left.
func fill(s string) cell { return cell{text: s, fill: true} }

// span gives a line of one cell, which spans the width.
func span(s string) []cell { return []cell{fill(s)} }

// age gives a duration in its coarsest informative unit, as kit status draws one: 45s, 12m, 3h, 47d.
func age(secs int64) string {
	if secs < 0 {
		secs = 0
	}
	switch {
	case secs < 120:
		return strconv.FormatInt(secs, 10) + "s"
	case secs < 7200:
		return strconv.FormatInt(secs/60, 10) + "m"
	case secs < 172800:
		return strconv.FormatInt(secs/3600, 10) + "h"
	}
	return strconv.FormatInt(secs/86400, 10) + "d"
}

// ageOf gives an age that may be missing.
func ageOf(secs *int64) string {
	if secs == nil {
		return ""
	}
	return age(*secs)
}

// owners gives the assignees after their glyph, or nothing.
func owners(logins []string) string {
	if len(logins) == 0 {
		return ""
	}
	return Owner + " " + strings.Join(logins, " ")
}

// branchOf gives a branch after its glyph, or nothing.
func branchOf(branch string) string {
	if branch == "" {
		return ""
	}
	return Branch + " " + branch
}

// cell gives the checks cell, as kit status draws it: the glyph, the counts, and how long the slowest took, dim; or
// nothing when no check has reported.
func (c *checks) cell() string {
	if c == nil {
		return ""
	}
	var out string
	switch c.State {
	case "SUCCESS":
		out = fmt.Sprintf("%s %d / %d", Passed, c.Total, c.Total)
	case "FAILURE":
		out = fmt.Sprintf("%s %s failing %d / %d", Failed, c.Failing, c.Passed, c.Total)
	case "PENDING":
		out = fmt.Sprintf("%s %d / %d", Running, c.Passed, c.Total)
	default:
		return ""
	}
	if c.Took != nil {
		out += " " + style.Off(age(*c.Took))
	}
	return out
}

// rows gives a stage's rows from what it loaded, laid out for the width; nothing while it loads or after a refusal.
func (m Model) rows(i int) []row {
	s := m.stages[i]
	switch s.name {
	case "backlog":
		return m.backlogRows(s)
	case "incoming":
		return incomingRows(s)
	case "desk":
		return m.deskRows(s)
	case "worktrees":
		return m.worktreeRows(s)
	case "changes":
		return changeRows(s)
	case "releases":
		return m.releaseRows(s)
	}
	return nil
}

// backlogRows draws each issue: number, assignees, age, branch, title; open, the title in full, the URL, and the
// command that starts it.
func (m Model) backlogRows(s stage) []row {
	var rows []row
	for _, i := range s.issues {
		r := row{key: "#" + strconv.Itoa(i.Number)}
		r.cells = []cell{number(r.key), dim(owners(i.Assignees)), dimNumber(ageOf(i.Age)), dim(branchOf(i.Branch)),
			fill(Issue + " " + i.Title)}
		for _, line := range wrap(i.Title, m.width-6) {
			r.detail = append(r.detail, span(line))
		}
		if url := m.origin.issue(i.Number, i.URL); url != "" {
			r.detail = append(r.detail, span(style.Off(url)))
		}
		r.detail = append(r.detail, span(style.Off(fmt.Sprintf("kit issue %d starts it", i.Number))))
		rows = append(rows, r)
	}
	return rows
}

// incomingRows draws what the fetch brought, in kit status's order: each commit on the default branch, then each
// branch updated, new, or gone, then each new tag. Every row is a leaf.
func incomingRows(s stage) []row {
	if s.in == nil || !s.in.Fetched {
		return nil
	}
	var rows []row
	for _, c := range s.in.Commits {
		rows = append(rows, row{key: c.SHA, cells: []cell{dim(c.SHA), fill(c.Subject)}})
	}
	for _, b := range s.in.Branches.Updated {
		rows = append(rows, row{key: "updated " + b, cells: []cell{text(Branch), fill(b + " " + style.Off("updated"))}})
	}
	for _, b := range s.in.Branches.New {
		rows = append(rows, row{key: "new " + b, cells: []cell{text(Branch), fill(b + " " + style.Off("new branch"))}})
	}
	for _, b := range s.in.Branches.Gone {
		rows = append(rows, row{key: "gone " + b, cells: []cell{text(Gone), fill(b + " " + style.Off("upstream gone"))}})
	}
	for _, t := range s.in.Tags {
		rows = append(rows, row{key: "tag " + t, cells: []cell{text(Tag), fill(t + " " + style.Off("new tag"))}})
	}
	return rows
}

// commitsShown is how many commits ahead an open checkout lists before counting the rest.
const commitsShown = 10

// deskRows draws the desk, the root checkout, in one row, as worktreeRows draws a worktree.
func (m Model) deskRows(s stage) []row {
	if s.desk == nil {
		return nil
	}
	c := *s.desk
	return []row{m.checkoutRow(c, Desk, len("↑"+strconv.Itoa(c.Ahead)), true)}
}

// worktreeRows draws each linked worktree, as checkoutRow does. Last, dim, the branches whose upstream is gone, when
// there are any; open, one row per branch.
func (m Model) worktreeRows(s stage) []row {
	if s.trees == nil {
		return nil
	}
	aheadWidth := 0
	for _, c := range s.trees.Worktrees {
		aheadWidth = max(aheadWidth, len("↑"+strconv.Itoa(c.Ahead)))
	}
	var rows []row
	for _, c := range s.trees.Worktrees {
		rows = append(rows, m.checkoutRow(c, Branch, aheadWidth, false))
	}
	if gone := s.trees.GoneBranches; len(gone) > 0 {
		r := row{key: "gone", cells: span(style.Off(Gone + " gone: " + strings.Join(gone, ", ")))}
		for _, b := range gone {
			r.detail = append(r.detail, span(Gone+" "+b+" "+style.Off("upstream gone · kit tidy removes it")))
		}
		rows = append(rows, r)
	}
	return rows
}

// checkoutRow draws a checkout: glyph and branch, ahead and behind, its pull request's number and state, and its
// uncommitted count; open, the uncommitted paths with their status in yellow, then the commits ahead, ten at most,
// as kit status --text draws them.
func (m Model) checkoutRow(c checkout, glyph string, aheadWidth int, desk bool) row {
	name := c.Branch
	if name == "" {
		name = c.Relative
	}
	if name == "" {
		name = c.Path
	}
	label := glyph + " " + name
	if c.Detached {
		label += " " + Detached + " detached"
	}
	r := row{key: c.Path, cells: []cell{text(label), text(counts(c, aheadWidth)), text(m.pullOf(c.Branch, desk)),
		uncommitted(c.Uncommitted)}}
	for _, change := range c.Changes {
		r.detail = append(r.detail, span(style.Warn(fmt.Sprintf("%-2s", change.Status))+" "+change.Path))
	}
	for i, commit := range c.Commits {
		if i == commitsShown {
			break
		}
		r.detail = append(r.detail, span(style.Off(commit.SHA)+" "+commit.Subject))
	}
	if more := c.Ahead - min(len(c.Commits), commitsShown); more > 0 {
		r.detail = append(r.detail, span(style.Off(fmt.Sprintf("… and %d more", more))))
	}
	return r
}

// counts gives "↑a ↓b", ahead right-aligned to the widest in the table, a zero dim.
func counts(c checkout, aheadWidth int) string {
	ahead := "↑" + strconv.Itoa(c.Ahead)
	pad := strings.Repeat(" ", max(0, aheadWidth-len(ahead)))
	return pad + count(ahead, c.Ahead) + " " + count("↓"+strconv.Itoa(c.Behind), c.Behind)
}

// count gives a count's text, dim when it is zero.
func count(text string, n int) string {
	if n == 0 {
		return style.Off(text)
	}
	return text
}

// uncommitted gives the uncommitted cell: the count, or clean, dim.
func uncommitted(n int) cell {
	if n == 0 {
		return dim("clean")
	}
	return text(style.Warn(fmt.Sprintf("%d uncommitted", n)))
}

// pullOf gives the pull request cell of a checkout on the branch: its number and state when changes has loaded and
// holds one, … while changes loads, nothing when changes is not in the picture or failed, nothing for the desk or a
// detached checkout without one, and no pull request yet for a worktree without one.
func (m Model) pullOf(branch string, desk bool) string {
	s := m.find("changes")
	if s == nil || s.refusal != "" {
		return ""
	}
	if s.loading {
		return style.Off("…")
	}
	for _, p := range s.pulls {
		if p.Branch == branch && branch != "" {
			return Change + " " + ref(p) + " " + stateGlyph(p.State) + " " + p.State
		}
	}
	if desk || branch == "" {
		return ""
	}
	return style.Off("no pull request yet")
}

// ref gives a pull request's reference: what kit printed, else #<number>.
func ref(p pull) string {
	if p.Ref != "" {
		return p.Ref
	}
	return "#" + strconv.Itoa(p.Number)
}

// changeRows draws each pull request: number, title, author, state, age, checks, threads, branch; open, the issues
// it closes, one row each, in flight.
func changeRows(s stage) []row {
	var rows []row
	for _, p := range s.pulls {
		threads := ""
		if p.Unresolved > 0 {
			threads = fmt.Sprintf("%s %d unresolved", Threads, p.Unresolved)
		}
		author := ""
		if p.Author != "" {
			author = Owner + " " + p.Author
		}
		r := row{key: ref(p), cells: []cell{text(Change + " " + ref(p)), fill(p.Title), dim(author),
			text(stateGlyph(p.State) + " " + p.State), dimNumber(age(p.Age)), text(p.Checks.cell()), text(threads),
			dim(branchOf(p.Branch))}}
		for _, i := range p.Closes {
			r.detail = append(r.detail, []cell{number("#" + strconv.Itoa(i.Number)), dim(owners(i.Assignees)),
				text(InFlight + " in flight"), dimNumber(ageOf(i.Age)), fill(Issue + " " + i.Title)})
		}
		if len(p.Closes) == 0 {
			r.detail = append(r.detail, span(style.Off("closes no issue")))
		}
		rows = append(rows, r)
	}
	return rows
}

// releaseRows draws each release: version, kind, age, latest; open, the name, the tag, the date, and the URL.
func (m Model) releaseRows(s stage) []row {
	var rows []row
	for _, rel := range s.rels {
		latest := ""
		if rel.Latest {
			latest = Latest + " latest"
		}
		r := row{key: rel.Tag, cells: []cell{text(rel.Version), text(kindGlyph(rel.Kind) + " " + rel.Kind),
			dimNumber(age(rel.Age)), text(latest)}}
		r.detail = [][]cell{span(rel.Name), span(style.Off(rel.Tag)), span(style.Off(rel.At))}
		if url := m.origin.release(rel.Tag, rel.URL); url != "" {
			r.detail = append(r.detail, span(style.Off(url)))
		}
		rows = append(rows, r)
	}
	return rows
}

// empty gives the dim line an empty stage shows.
func empty(s stage) string {
	switch s.name {
	case "backlog":
		return "no open issue without a pull request"
	case "incoming":
		if s.in != nil && !s.in.Fetched {
			return "not fetched: kit status --no-fetch skips the fetch"
		}
		return "nothing new on origin"
	case "desk":
		return "no desk"
	case "worktrees":
		return "no linked worktree"
	case "changes":
		return "no open pull request"
	case "releases":
		return "no release"
	}
	return "nothing"
}
