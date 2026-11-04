// Each stage's JSON, as kit flow <stage> --json prints it, and the rows a stage draws from it: what a row shows
// closed, as cells laid out in columns, and what it opens to.
package flow

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/style"
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

// underway is one issue an open pull request closes, with that pull request: one per issue and pull request, so an
// issue two pull requests close is two. Here says a worktree on this desk has the pull request's branch, and
// Worktree is its path. The URLs are kit's when it prints them.
type underway struct {
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	Assignees []string `json:"assignees"`
	Age       *int64   `json:"age"`
	URL       string   `json:"url"`
	PR        struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Author string `json:"author"`
		State  string `json:"state"`
		Branch string `json:"branch"`
		URL    string `json:"url"`
	} `json:"pr"`
	Here     bool   `json:"here"`
	Worktree string `json:"worktree"`
}

// incoming is what the fetch brought from every remote, the primary named; with --no-fetch, Fetched is false and the
// remotes carry no commits, branches, or tags.
type incoming struct {
	Fetched bool     `json:"fetched"`
	Primary string   `json:"primary"`
	Remotes []remote `json:"remotes"`
}

// remote is one remote and what the fetch brought from it: the commits new on its default branch, its branches new,
// updated, and gone, and its tags new. Forge is empty when the remote has none. Error is git's line when its fetch
// failed, and then nothing came.
type remote struct {
	Name     string   `json:"name"`
	Error    string   `json:"error"`
	Forge    string   `json:"forge"`
	Host     string   `json:"host"`
	Default  string   `json:"default"`
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

// run is one build: a workflow run not owned by any open pull request, on the default branch, on a tag, or started by a
// schedule or by hand. Number is the forge's count of the workflow's runs, nil when unknown, and Attempt 1 for a
// first run; Tag says the branch is a tag; SHA and Title are the commit's short hash and the forge's headline, empty
// when it gives none. State is running, passed, failed, or cancelled; Age and Took are seconds, Took nil while it
// runs. Jobs may be empty when the forge does not report them cheaply.
type run struct {
	ID       int64  `json:"id"`
	Number   *int64 `json:"number"`
	Attempt  int    `json:"attempt"`
	Workflow string `json:"workflow"`
	Branch   string `json:"branch"`
	Tag      bool   `json:"tag"`
	Event    string `json:"event"`
	SHA      string `json:"sha"`
	Title    string `json:"title"`
	State    string `json:"state"`
	Age      int64  `json:"age"`
	Took     *int64 `json:"took"`
	URL      string `json:"url"`
	Jobs     []job  `json:"jobs"`
}

// job is one job of a build: its name, its state as a build's, how long it took, nil while it runs, and the step that
// failed, when one did.
type job struct {
	Name  string `json:"name"`
	State string `json:"state"`
	Took  *int64 `json:"took"`
	Step  string `json:"step"`
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

// cell is one column of a row, and row one row, as internal/grid lays them out: a cell's values, a number
// right-aligned, and the one cell that wraps; a row's key, its cells, the lines it opens to, whether it is an aside,
// which a stage's count leaves out, as a remote's heading, and the logins it belongs to, which the panel's show
// matches.
type (
	cell = grid.Cell
	row  = grid.Row
)

// text gives a plain cell.
func text(s string) cell { return cell{Values: []string{s}} }

// dim gives a dim cell; an empty text stays empty, so its column can be left out.
func dim(s string) cell {
	if s == "" {
		return cell{}
	}
	return text(style.Off(s))
}

// number gives a right-aligned cell.
func number(s string) cell { return cell{Values: []string{s}, Right: true} }

// dimNumber gives a dim, right-aligned cell.
func dimNumber(s string) cell {
	c := dim(s)
	c.Right = true
	return c
}

// fill gives the cell that takes the width left, cut while its row is closed and wrapped when it is open.
func fill(s string) cell { return cell{Values: []string{s}, Wrap: true} }

// linked gives the cell with a link, which a click on its glyph opens.
func linked(c cell, url string) cell {
	c.Link = url
	return c
}

// span gives a line of one cell, which spans the width.
func span(s string) []cell { return []cell{fill(s)} }

// age gives a duration in its coarsest informative unit, as kit flow draws one: 45s, 12m, 3h, 47d.
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

// took gives how long a build or a job took, to the second: 45s, 3m 36s, 1h 02m; … while it runs.
func took(secs *int64) string {
	switch {
	case secs == nil:
		return "…"
	case *secs < 60:
		return strconv.FormatInt(max(*secs, 0), 10) + "s"
	case *secs < 3600:
		return fmt.Sprintf("%dm %02ds", *secs/60, *secs%60)
	}
	return fmt.Sprintf("%dh %02dm", *secs/3600, *secs%3600/60)
}

// tookCell gives a duration's cell, right-aligned: dim … while it runs.
func tookCell(secs *int64) cell {
	if secs == nil {
		return dimNumber(took(secs))
	}
	return number(took(secs))
}

// ageOf gives an age that may be missing.
func ageOf(secs *int64) string {
	if secs == nil {
		return ""
	}
	return age(*secs)
}

// owners gives the assignees' cell: each after its glyph, dim, a value of its own, so a closed row shows the first and
// how many more; empty for none.
func owners(logins []string) cell {
	var c cell
	for _, login := range logins {
		c.Values = append(c.Values, style.Off(Owner+" "+login))
	}
	return c
}

// branchOf gives a branch after its glyph, or nothing.
func branchOf(branch string) string {
	if branch == "" {
		return ""
	}
	return Branch + " " + branch
}

// cell gives the checks cell, as kit flow draws it: the glyph, the counts, and how long the slowest took, dim; or
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

// rows gives a stage's rows from what it loaded, laid out for the width, as its panel lets them through; nothing while
// it loads or after a refusal.
func (m Model) rows(i int) []row { return m.shown(i).rows }

// offered gives the columns a stage's panel offers a checkbox for, as filled says, kept as its rows are.
func (m Model) offered(i int) []string { return m.shown(i).names }

// shown gives a stage's rows as its panel lets them through, built once for what the stages hold, the width, and the
// panels, then kept until one changes.
func (m Model) shown(i int) shown {
	if m.built == nil || m.built.width != m.paneWidth() {
		return m.build(i)
	}
	r, ok := m.built.stages[i]
	if !ok {
		r = m.build(i)
		m.built.stages[i] = r
	}
	return r
}

// shown is a stage's rows as its panel lets them through, the columns its panel offers, and how many rows but the
// asides it loaded, the panel apart.
type shown struct {
	rows  []row
	names []string
	all   int
}

// columns names each stage's columns, in the order its rows' cells come, for the panel's checkboxes; a column without
// a name, as a glyph's, has none. A stage without columns, as incoming, whose rows are commits and branches, has only
// the search. An issue's column is its id, the reference its tracker gives it, which need not be a number.
var columns = map[string][]string{
	"backlog":   {"id", "assignees", "age", "branch", "title"},
	"underway":  {"id", "assignees", "", "age", "change", "state", "where", "title"},
	"desk":      {"branch", "ahead", "lines", "change", "uncommitted"},
	"worktrees": {"branch", "ahead", "lines", "change", "uncommitted"},
	"changes":   {"change", "title", "author", "state", "age", "checks", "threads", "branch"},
	"builds":    {"", "workflow", "run", "event", "branch", "commit", "title", "age", "took"},
	"releases":  {"version", "kind", "age", "latest"},
}

// owned names the stages whose rows belong to people, which the panel lets through as every row, the caller's, or
// everyone else's.
var owned = map[string]bool{"backlog": true, "underway": true, "changes": true}

// build gives a stage's rows from what it loaded, as its panel lets them through, and what the panel shows of them.
func (m Model) build(i int) shown {
	all, names := m.unfiltered(i), columns[m.stages[i].name]
	return shown{m.filter(i).Filter(all, names), filled(all, names), counted(all)}
}

// filled gives a stage's columns as its panel offers them: each named column that one of the rows loaded has a value
// in, and no name for the rest, as backlog's branch while no issue has one, since a checkbox there would hide nothing.
func filled(rows []row, names []string) []string {
	out := make([]string, len(names))
	for _, r := range rows {
		if len(r.Cells) < 2 {
			continue
		}
		for j, c := range r.Cells {
			if j < len(names) && len(c.Values) > 0 && c.Values[0] != "" {
				out[j] = names[j]
			}
		}
	}
	return out
}

// unfiltered lays a stage's rows out from what it loaded.
func (m Model) unfiltered(i int) []row {
	s := m.stages[i]
	switch s.name {
	case "backlog":
		return m.backlogRows(s)
	case "underway":
		return m.underwayRows(s)
	case "incoming":
		return incomingRows(s)
	case "desk":
		return m.deskRows(s)
	case "worktrees":
		return m.worktreeRows(s)
	case "changes":
		return m.changeRows(s)
	case "builds":
		return m.buildRows(s)
	case "releases":
		return m.releaseRows(s)
	}
	return nil
}

// backlogRows draws each issue: number, assignees, age, branch, title, its glyph leading to the issue's page; open,
// every assignee and the title wrapped. An issue belongs to its assignees.
func (m Model) backlogRows(s stage) []row {
	var rows []row
	for _, i := range s.issues {
		r := row{Key: "#" + strconv.Itoa(i.Number), Owners: i.Assignees}
		r.Cells = []cell{number(r.Key), owners(i.Assignees), dimNumber(ageOf(i.Age)), dim(branchOf(i.Branch)),
			linked(fill(Issue+" "+i.Title), m.origin.issue(i.Number, i.URL))}
		rows = append(rows, r)
	}
	return rows
}

// underwayRows draws each issue underway beside the pull request that closes it, oldest first as kit prints them:
// number, assignees, age, the pull request's number and state, where the work is, and the title. Where is here when
// a worktree on this desk has the branch, else the author. The pull request's glyph leads to its row under changes,
// or to its page when changes is not in the picture; the state's glyph to its page, and the title's to the issue's.
// Open, every assignee, the issue's title wrapped, and the worktree when here; the pull request's title is under
// changes. A row belongs to the issue's assignees and the pull request's author.
func (m Model) underwayRows(s stage) []row {
	var rows []row
	for _, u := range s.under {
		issue, pr, page := "#"+strconv.Itoa(u.Number), "#"+strconv.Itoa(u.PR.Number), m.origin.pull(u.PR.Number, u.PR.URL)
		where := cell{}
		switch {
		case u.Here:
			where = text(Desk + " here")
		case u.PR.Author != "":
			where = dim(Owner + " " + u.PR.Author)
		}
		change := page
		if m.find("changes") != nil {
			change = jumpTo("changes", u.PR.Number)
		}
		state := stateGlyph(u.PR.State) + " " + u.PR.State
		r := row{Key: issue + " " + pr, Owners: append(slices.Clone(u.Assignees), u.PR.Author),
			Cells: []cell{number(issue), owners(u.Assignees), text(InFlight), dimNumber(ageOf(u.Age)),
				linked(text(Change+" "+pr), change), linked(text(state), page), where,
				linked(fill(Issue+" "+u.Title), m.origin.issue(u.Number, u.URL))}}
		if u.Here && u.Worktree != "" {
			r.Detail = append(r.Detail, span(Desk+" "+u.Worktree))
		}
		rows = append(rows, r)
	}
	return rows
}

// jumps is how a link to a row of another stage begins, which a click follows on the screen, not in the browser.
const jumps = "kittui:"

// jumpTo gives the link to a pull request's row under a stage: the stage's name and the pull request's number.
func jumpTo(stage string, number int) string { return jumps + stage + "/" + strconv.Itoa(number) }

// incomingRows draws what the fetch brought from each remote, the primary first. Each remote has a dim heading, its
// name, forge, and host, then its rows in kit flow's order: each commit on its default branch, each branch updated,
// new, or gone, and each new tag. A remote whose fetch failed shows git's line, in red; one with nothing new says
// unchanged; and without the fetch each says not fetched. Every row is a leaf; the headings and those lines are
// asides.
func incomingRows(s stage) []row {
	if s.in == nil {
		return nil
	}
	var rows []row
	for _, r := range primaryFirst(*s.in) {
		rows = append(rows, row{Key: "remote " + r.Name, Cells: span(style.Off(heading(r))), Aside: true})
		before, key := len(rows), r.Name+" "
		if s.in.Fetched {
			for _, c := range r.Commits {
				rows = append(rows, row{Key: key + c.SHA, Cells: []cell{dim(c.SHA), fill(c.Subject)}})
			}
			for _, b := range r.Branches.Updated {
				rows = append(rows, row{Key: key + "updated " + b,
					Cells: []cell{text(Branch), fill(b + " " + style.Off("updated"))}})
			}
			for _, b := range r.Branches.New {
				rows = append(rows, row{Key: key + "new " + b,
					Cells: []cell{text(Branch), fill(b + " " + style.Off("new branch"))}})
			}
			for _, b := range r.Branches.Gone {
				rows = append(rows, row{Key: key + "gone " + b,
					Cells: []cell{text(Gone), fill(b + " " + style.Off("upstream gone"))}})
			}
			for _, t := range r.Tags {
				rows = append(rows, row{Key: key + "tag " + t,
					Cells: []cell{text(Tag), fill(t + " " + style.Off("new tag"))}})
			}
		}
		if r.Error != "" {
			failed := span(style.Refusal(Failed + " " + r.Error))
			rows = append(rows, row{Key: key + "error", Cells: failed, Aside: true})
		} else if len(rows) == before {
			word := "unchanged"
			if !s.in.Fetched {
				word = "not fetched"
			}
			rows = append(rows, row{Key: key + word, Cells: span(style.Off(word)), Aside: true})
		}
	}
	return rows
}

// primaryFirst gives the remotes with the primary first, the rest in kit's order.
func primaryFirst(in incoming) []remote {
	out := slices.Clone(in.Remotes)
	slices.SortStableFunc(out, func(x, y remote) int {
		return primaryRank(x, in.Primary) - primaryRank(y, in.Primary)
	})
	return out
}

// primaryRank sorts the primary remote before the rest.
func primaryRank(r remote, primary string) int {
	if r.Name == primary {
		return 0
	}
	return 1
}

// heading gives a remote's heading: its glyph, its name, its forge or no forge, and its host when known.
func heading(r remote) string {
	forge := r.Forge
	if forge == "" {
		forge = "no forge"
	}
	parts := []string{Remote + " " + r.Name, forge}
	if r.Host != "" {
		parts = append(parts, r.Host)
	}
	return strings.Join(parts, " · ")
}

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
		r := row{Key: "gone", Cells: span(style.Off(Gone + " gone: " + strings.Join(gone, ", ")))}
		for _, b := range gone {
			r.Detail = append(r.Detail, span(Gone+" "+b+" "+style.Off("upstream gone · kit tidy removes it")))
		}
		rows = append(rows, r)
	}
	return rows
}

// checkoutRow draws a checkout: glyph and branch, ahead and behind, its pull request's number and state, and its
// uncommitted count; open, the uncommitted paths with their status in yellow, then every commit ahead that kit
// listed, which the pane pages through, and a count of any kit did not list.
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
	r := row{Key: c.Path, Cells: []cell{text(label), text(counts(c, aheadWidth)), text(diffstat(c)),
		text(m.pullOf(c.Branch, desk)), uncommitted(c.Uncommitted)}}
	for _, change := range c.Changes {
		r.Detail = append(r.Detail, span(style.Warn(fmt.Sprintf("%-2s", change.Status))+" "+change.Path))
	}
	for _, commit := range c.Commits {
		r.Detail = append(r.Detail, span(style.Off(commit.SHA)+" "+commit.Subject))
	}
	if more := c.Ahead - len(c.Commits); more > 0 {
		r.Detail = append(r.Detail, span(style.Off(fmt.Sprintf("… and %d more", more))))
	}
	return r
}

// counts gives "↑a ↓b", ahead right-aligned to the widest in the table: ahead green and behind red, a zero dim.
func counts(c checkout, aheadWidth int) string {
	ahead := "↑" + strconv.Itoa(c.Ahead)
	pad := strings.Repeat(" ", max(0, aheadWidth-len(ahead)))
	return pad + count(ahead, c.Ahead, style.Gain) + " " + count("↓"+strconv.Itoa(c.Behind), c.Behind, style.Loss)
}

// diffstat gives "+x -y", the lines the branch adds and removes against the default branch: added green and removed
// red, a zero dim.
func diffstat(c checkout) string {
	return count("+"+strconv.Itoa(c.Added), c.Added, style.Gain) + " " +
		count("-"+strconv.Itoa(c.Removed), c.Removed, style.Loss)
}

// count gives a count's text in its color, dim when it is zero.
func count(text string, n int, paint func(string) string) string {
	if n == 0 {
		return style.Off(text)
	}
	return paint(text)
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

// changeRows draws each pull request: number, title, author, state, age, checks, threads, branch; open, the title
// wrapped, then the issues it closes, one row each, in flight. The pull request's glyph leads to its page. A pull
// request belongs to its author.
func (m Model) changeRows(s stage) []row {
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
		r := row{Key: ref(p), Owners: []string{p.Author}, Cells: []cell{linked(text(Change+" "+ref(p)), m.origin.pull(p.Number, "")), fill(p.Title), dim(author),
			text(stateGlyph(p.State) + " " + p.State), dimNumber(age(p.Age)), text(p.Checks.cell()), text(threads),
			dim(branchOf(p.Branch))}}
		for _, i := range p.Closes {
			r.Detail = append(r.Detail, []cell{number("#" + strconv.Itoa(i.Number)), owners(i.Assignees),
				text(InFlight + " in flight"), dimNumber(ageOf(i.Age)), fill(Issue + " " + i.Title)})
		}
		if len(p.Closes) == 0 {
			r.Detail = append(r.Detail, span(style.Off("closes no issue")))
		}
		rows = append(rows, r)
	}
	return rows
}

// buildRows draws each build, newest first as kit prints them: its state's glyph, workflow, number, how it began,
// branch or tag, commit, title, age, and how long it took; the number, how it began, and the commit dim, and a title
// that only repeats the workflow left out; the state's glyph leads to the run's page. Open, the title wrapped, the
// attempt past the first, and its jobs, the failed first, each with its glyph and how long it took and, under a failed
// one, the step that failed.
func (m Model) buildRows(s stage) []row {
	var rows []row
	for _, b := range s.runs {
		number, title := "", b.Title
		if b.Number != nil {
			number = "#" + strconv.FormatInt(*b.Number, 10)
		}
		if title == b.Workflow {
			title = ""
		}
		r := row{Key: strconv.FormatInt(b.ID, 10), Cells: []cell{linked(text(runGlyph(b.State)), b.URL), text(b.Workflow),
			dimNumber(number), dim(eventOf(b)), text(b.Branch), dim(b.SHA), fill(title), dimNumber(age(b.Age)),
			tookCell(b.Took)}}
		if b.Attempt > 1 {
			r.Detail = append(r.Detail, span(style.Off(fmt.Sprintf("attempt %d", b.Attempt))))
		}
		jobs := slices.Clone(b.Jobs)
		slices.SortStableFunc(jobs, func(x, y job) int { return failedFirst(x) - failedFirst(y) })
		for _, j := range jobs {
			r.Detail = append(r.Detail, []cell{text(runGlyph(j.State)), text(j.Name), tookCell(j.Took)})
			if j.Step != "" {
				r.Detail = append(r.Detail, span("    "+style.Off("failed at")+" "+j.Step))
			}
		}
		rows = append(rows, r)
	}
	return rows
}

// eventOf says how a build began, in a word, as kit flow --text does: tag for a push to a tag, push, schedule, or
// manual.
func eventOf(b run) string {
	switch {
	case b.Tag:
		return "tag"
	case b.Event == "workflow_dispatch":
		return "manual"
	}
	return b.Event
}

// failedFirst sorts a failed job before the rest.
func failedFirst(j job) int {
	if j.State == "failed" {
		return 0
	}
	return 1
}

// releaseRows draws each release: version, kind, age, latest, the kind's glyph leading to the release's page; open,
// the name, the tag, and the date.
func (m Model) releaseRows(s stage) []row {
	var rows []row
	for _, rel := range s.rels {
		latest := ""
		if rel.Latest {
			latest = Latest + " latest"
		}
		kind := linked(text(kindGlyph(rel.Kind)+" "+rel.Kind), m.origin.release(rel.Tag, rel.URL))
		r := row{Key: rel.Tag, Cells: []cell{text(rel.Version), kind, dimNumber(age(rel.Age)), text(latest)}}
		r.Detail = [][]cell{span(rel.Name), span(style.Off(rel.Tag)), span(style.Off(rel.At))}
		rows = append(rows, r)
	}
	return rows
}

// empty gives the dim line an empty stage shows.
func empty(s stage) string {
	switch s.name {
	case "backlog":
		return "no open issue without a pull request"
	case "underway":
		return "no issue is underway"
	case "incoming":
		if s.in != nil && !s.in.Fetched {
			return "not fetched: kit flow --no-fetch skips the fetch"
		}
		return "no remote to fetch"
	case "desk":
		return "no desk"
	case "worktrees":
		return "no linked worktree"
	case "changes":
		return "no open pull request"
	case "builds":
		return "no build since the last 20 runs"
	case "releases":
		return "no release"
	}
	return "nothing"
}
