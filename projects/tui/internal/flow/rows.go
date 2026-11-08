// Each stage's JSON, as kit flow <stage> --json prints it, and the rows a stage draws from it: what a row holds in each
// of the stage's fields, by the field's id, which its renderer draws in the field's column, and what it opens to.
package flow

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/grid"
	"github.com/empowerite/kit/projects/tui/internal/style"
	"github.com/empowerite/kit/projects/tui/internal/table"
)

// issue is one of backlog's, and one a pull request closes. Age is seconds, or nothing for an issue without a time;
// Branch is the one started for it on origin, or empty; Source is the tracker that holds it, as kit names it, and
// Organization and Repository where in it the issue is, each empty where the tracker has none; Labels are its forge's.
type issue struct {
	Number       int           `json:"number"`
	Title        string        `json:"title"`
	Assignees    []string      `json:"assignees"`
	Labels       []table.Label `json:"labels"`
	Age          *int64        `json:"age"`
	Branch       string        `json:"branch"`
	URL          string        `json:"url"`
	Source       string        `json:"source"`
	Organization string        `json:"organization"`
	Repository   string        `json:"repository"`
}

// commit is one commit: its short hash and its subject.
type commit struct {
	SHA     string `json:"sha"`
	Subject string `json:"subject"`
}

// underway is one issue an open pull request closes, with that pull request: one per issue and pull request, so an
// issue two pull requests close is two. Here says a worktree on this desk has the pull request's branch, and
// Worktree is its path; Local says this desk has the branch, but no checkout of it. The pull request's Fork is the
// owner of the repository its branch is in, when that is not origin; Pushed is its branch's last push, how long ago
// and by whom, when the forge says. Source is the tracker that holds the issue, and Organization and Repository where
// in it the issue is, and Labels the issue's, not the pull request's. The URLs are kit's when it prints them.
type underway struct {
	Number       int           `json:"number"`
	Title        string        `json:"title"`
	Assignees    []string      `json:"assignees"`
	Labels       []table.Label `json:"labels"`
	Age          *int64        `json:"age"`
	URL          string        `json:"url"`
	Source       string        `json:"source"`
	Organization string        `json:"organization"`
	Repository   string        `json:"repository"`
	PR           struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Author string `json:"author"`
		State  string `json:"state"`
		Branch string `json:"branch"`
		URL    string `json:"url"`
		Fork   string `json:"fork"`
		Pushed *struct {
			Age int64  `json:"age"`
			By  string `json:"by"`
		} `json:"pushed"`
	} `json:"pr"`
	Here     bool   `json:"here"`
	Local    bool   `json:"local"`
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

// pull is one open pull request, with its labels and the issues it closes.
type pull struct {
	Number     int           `json:"number"`
	Ref        string        `json:"ref"`
	Title      string        `json:"title"`
	Author     string        `json:"author"`
	State      string        `json:"state"`
	Age        int64         `json:"age"`
	CreatedAt  string        `json:"createdAt"`
	Checks     *checks       `json:"checks"`
	Unresolved int           `json:"unresolved"`
	Branch     string        `json:"branch"`
	Labels     []table.Label `json:"labels"`
	Closes     []issue       `json:"closes"`
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

// span gives a line of one cell, which spans the width.
func span(s string) []cell { return []cell{fill(s)} }

// tookCell gives a duration's cell, right-aligned: dim … while it runs.
func tookCell(secs *int64) cell {
	if secs == nil {
		return dimNumber(table.Took(secs))
	}
	return number(table.Took(secs))
}

// ageOf gives an age that may be missing.
func ageOf(secs *int64) string {
	if secs == nil {
		return ""
	}
	return table.Age(*secs)
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
		out += " " + style.Off(table.Age(*c.Took))
	}
	return out
}

// record is a row as a stage's builder gives it: the row, without its cells, and its value in each of the stage's
// fields, by the field's id. A row of one cell, which spans the columns, as a heading, has its cell and no values.
type record struct {
	row
	values map[string]table.Value
}

// laid gives a stage's rows from its records: each record's values drawn in the stage's fields, in the order their
// columns stand, as each field's renderer draws its column, so a renderer can line a column up, or the one pinned gives
// it by id; a row of one cell as it is.
func laid(stage string, records []record, pinned func(id string) string) []row {
	out := make([]row, len(records))
	var at []int
	for k, r := range records {
		out[k] = r.row
		if r.values != nil {
			at = append(at, k)
		}
	}
	for _, f := range fieldsOf(stage) {
		vs := make([]table.Value, len(at))
		for n, k := range at {
			vs[n] = records[k].values[f.ID]
		}
		for n, c := range f.Draw(vs, pinned(f.ID)) {
			out[at[n]].Cells = append(out[at[n]].Cells, c)
		}
	}
	return out
}

// rows gives a stage's rows from what it loaded, laid out for the width, as its panel lets them through; nothing while
// it loads or after a refusal.
func (m Model) rows(i int) []row { return m.shown(i).rows }

// offered gives the columns a stage's panel lists in its editor, as filled says, kept as its rows are.
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

// shown is a stage's rows as its panel lets them through, the columns its panel offers, how many levels of headings
// they stand under, how many rows but the asides it loaded, the panel apart, and the rendering each field is drawn by,
// by id, as automatic chose it.
type shown struct {
	rows   []row
	names  []string
	levels int
	all    int
	drawn  map[string]string
}

// build gives a stage's rows from what it loaded, as its panel lets them through, and what the panel shows of them.
// The rows are laid out twice: first as the panel pins each field's rendering, or by its first, which says the rows
// the panel lets through; then by the renderings automatic chooses for those rows and the width, when they differ.
// The second time lets through the rows the first did, so a search keeps the rows whose text had it, though a field
// now draws less of it.
func (m Model) build(i int) shown {
	s, names, p := m.stages[i], columnsOf(m.stages[i].name), m.filter(i)
	if s.name == "incoming" {
		all := incomingRows(s)
		rows := p.Filter(all, names)
		return shown{p.FoldAway(rows), filled(all, names), grid.Levels(rows), table.Counted(all), nil}
	}
	records := m.records(s)
	pinned := func(id string) string { return s.View.Field(id).Rendering }
	all := laid(s.name, records, pinned)
	rows := p.Filter(all, names)
	fields := m.placed(i)
	drawn := table.Automatic(fields, shownValues(records, rows), m.paneWidth()-2*grid.Levels(rows)-len(lead), p.Glyphs)
	if slices.ContainsFunc(fields, func(f table.Placed) bool { return drawn[f.ID] != f.DrawnBy(f.Pinned) }) {
		passed := map[string]bool{}
		for _, r := range rows {
			passed[r.Key] = true
		}
		all = laid(s.name, records, func(id string) string { return drawn[id] })
		q := p
		q.Query = ""
		rows = q.Filter(slices.DeleteFunc(slices.Clone(all), func(r row) bool { return !passed[r.Key] }), names)
	}
	return shown{p.FoldAway(rows), filled(all, names), grid.Levels(rows), table.Counted(all), drawn}
}

// placed gives a stage's fields as its panel places them, for automatic: its columns, in the order they stand, each
// with the name the header stands over it while it shows, then the fields grouped or hidden.
func (m Model) placed(i int) []table.Placed {
	s, p := m.stages[i], m.filter(i)
	fields := fieldsOf(s.name)
	heads := m.names(i)
	var out, rest []table.Placed
	for k, id := range p.Arrange(columnsOf(s.name)) {
		f := fields[k]
		if id != "" {
			f = fields[slices.IndexFunc(fields, func(f table.Field) bool { return f.ID == id })]
		}
		pin := p.Field(f.ID).Rendering
		if id != "" && !p.Shows(id) {
			rest = append(rest, table.Placed{Field: f, Pinned: pin})
			continue
		}
		name := ""
		if len(out) < len(heads) {
			name = heads[len(out)]
		}
		out = append(out, table.Placed{Field: f, Pinned: pin, Column: true, Name: name})
	}
	return append(out, rest...)
}

// shownValues gives the values the records the rows let through hold in each field, by id, one a record.
func shownValues(records []record, rows []row) map[string][]table.Value {
	passed := map[string]bool{}
	for _, r := range rows {
		if !r.Aside {
			passed[r.Key] = true
		}
	}
	out := map[string][]table.Value{}
	for _, r := range records {
		if r.values == nil || !passed[r.Key] {
			continue
		}
		for id, v := range r.values {
			out[id] = append(out[id], v)
		}
	}
	return out
}

// filled gives a stage's columns as its panel offers them: each named column that one of the rows loaded has a value
// in, and no name for the rest, as backlog's branch while no issue has one, since a line there would hide nothing.
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

// records gives a stage's records from what it loaded; none for a stage kittui does not draw.
func (m Model) records(s stage) []record {
	switch s.name {
	case "backlog":
		return m.backlogRows(s)
	case "underway":
		return m.underwayRows(s)
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

// backlogRows gives each issue: number, source, organization, repository, assignees, age, branch, labels, title, its
// glyph leading to the issue's page; open, every assignee, the title wrapped, and every label. An issue belongs to its assignees.
func (m Model) backlogRows(s stage) []record {
	var out []record
	for _, i := range s.issues {
		key := "#" + strconv.Itoa(i.Number)
		out = append(out, record{row{Key: key}, map[string]table.Value{
			"id": table.One(key), "source": table.Some(i.Source), "organization": table.Some(i.Organization),
			"repository": table.Some(i.Repository), "assignees": {Raw: i.Assignees}, "age": table.Seconds(i.Age),
			"branch": table.Some(i.Branch), "labels": table.Labelled(i.Labels),
			"title": {Raw: []string{i.Title}, Link: m.origin.issue(i.Number, i.URL)},
		}})
	}
	return out
}

// underwayRows gives each issue underway beside the pull request that closes it, oldest first as kit prints them:
// number, source, organization, repository, assignees, age, the pull request's number and state, where the work is, the fork its branch is in,
// when last pushed, its age right-aligned after the glyph, by whom when not the author, the issue's labels, and the
// title. Where is here
// when a checkout on this desk has the branch, as here draws it, local when this desk has the branch but no checkout of
// it, else the author. The pull request's glyph leads to its row under changes, or to its page when changes is not in
// the picture; the state's glyph to its page, and the title's to the issue's. Open, every assignee and the issue's
// title wrapped; the pull request's title is under changes, and the checkout's path under desk or worktrees. A row
// belongs to the issue's assignees, whoever opened the pull request, as kit flow --mine says.
func (m Model) underwayRows(s stage) []record {
	var out []record
	for _, u := range s.under {
		issue, pr, page := "#"+strconv.Itoa(u.Number), "#"+strconv.Itoa(u.PR.Number), m.origin.pull(u.PR.Number, u.PR.URL)
		var where, pushed, by table.Value
		switch {
		case u.Here:
			where = m.here(u.Worktree)
		case u.Local:
			where = table.Value{Raw: []string{"local"}, Of: "local"}
		case u.PR.Author != "":
			where = table.Value{Raw: []string{u.PR.Author}, Of: "author"}
		}
		if p := u.PR.Pushed; p != nil {
			pushed = table.Seconds(&p.Age)
			if p.By != u.PR.Author {
				by = table.Some(p.By)
			}
		}
		change := page
		if m.find("changes") != nil {
			change = jumpTo("changes", m.pullKey(u.PR.Number))
		}
		out = append(out, record{row{Key: issue + " " + pr}, map[string]table.Value{
			"id": table.One(issue), "source": table.Some(u.Source), "organization": table.Some(u.Organization),
			"repository": table.Some(u.Repository), "assignees": {Raw: u.Assignees}, "age": table.Seconds(u.Age),
			"change": {Raw: []string{pr}, Link: change}, "state": {Raw: []string{u.PR.State}, Link: page},
			"where": where, "fork": table.Some(u.PR.Fork), "pushed": pushed, "by": by, "labels": table.Labelled(u.Labels),
			"title": {Raw: []string{u.Title}, Link: m.origin.issue(u.Number, u.URL)},
		}})
	}
	return out
}

// here gives underway's where for work in a checkout on this desk, at path: the desk, when the desk is that
// checkout, leading to its row under desk, else here, a worktree, leading to its row under worktrees; without the
// stage in the picture, its glyph leads nowhere.
func (m Model) here(path string) table.Value {
	stage, where := "worktrees", "here"
	if d := m.find("desk"); d != nil && d.desk != nil && d.desk.Path == path {
		stage, where = "desk", "desk"
	}
	v := table.Value{Raw: []string{where}, Of: where}
	if m.find(stage) != nil && path != "" {
		v.Link = jumpTo(stage, path)
	}
	return v
}

// jumps is how a link to a row of another stage begins, which a click follows on the screen, not in the browser.
const jumps = "kittui:"

// jumpTo gives the link to a row of a stage: the stage's name and the row's key.
func jumpTo(stage, key string) string { return jumps + stage + "/" + key }

// pullKey gives the key of a pull request's row under changes: its reference as kit printed it, else #<number>.
func (m Model) pullKey(number int) string {
	if s := m.find("changes"); s != nil {
		for _, p := range s.pulls {
			if p.Number == number {
				return ref(p)
			}
		}
	}
	return "#" + strconv.Itoa(number)
}

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

// deskRows gives the desk, the root checkout, in one row, as worktreeRows gives a worktree.
func (m Model) deskRows(s stage) []record {
	if s.desk == nil {
		return nil
	}
	return []record{m.checkoutRow(*s.desk, true)}
}

// worktreeRows gives each linked worktree, as checkoutRow does. Last, dim, the branches whose upstream is gone, when
// there are any; open, one row per branch.
func (m Model) worktreeRows(s stage) []record {
	if s.trees == nil {
		return nil
	}
	var out []record
	for _, c := range s.trees.Worktrees {
		out = append(out, m.checkoutRow(c, false))
	}
	if gone := s.trees.GoneBranches; len(gone) > 0 {
		r := row{Key: "gone", Cells: span(style.Off(Gone + " gone: " + strings.Join(gone, ", ")))}
		for _, b := range gone {
			r.Detail = append(r.Detail, span(Gone+" "+b+" "+style.Off("upstream gone · kit tidy removes it")))
		}
		out = append(out, record{row: r})
	}
	return out
}

// checkoutRow gives a checkout: its branch, or its path where it has none, ahead and behind, the lines it adds and
// removes, its pull request's number and state, as pullOf gives them, and its uncommitted count; open, the uncommitted
// paths with their status in yellow, then every commit ahead that kit listed, which the pane pages through, and a
// count of any kit did not list.
func (m Model) checkoutRow(c checkout, desk bool) record {
	name := c.Branch
	if name == "" {
		name = c.Relative
	}
	if name == "" {
		name = c.Path
	}
	change, state := m.pullOf(c.Branch, desk)
	r := record{row{Key: c.Path}, map[string]table.Value{
		"branch": {Raw: []string{name}, Of: c}, "ahead": {Raw: []string{strconv.Itoa(c.Ahead)}, Of: c},
		"lines": {Raw: []string{strconv.Itoa(c.Added)}, Of: c}, "change": change, "state": state,
		"uncommitted": table.One(strconv.Itoa(c.Uncommitted)),
	}}
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

// pullOf gives the pull request of a checkout on the branch, its reference and its state, when changes has loaded and
// holds one: the reference's glyph leads to its row under changes, and the state's to its page. Else the reference
// says … while changes loads; both say nothing when changes is not in the picture or failed, or for the desk or a
// detached checkout without one; and the state says no pull request yet for a worktree without one, so the
// references' column stays narrow.
func (m Model) pullOf(branch string, desk bool) (change, state table.Value) {
	s := m.find("changes")
	if s == nil || s.refusal != "" {
		return table.Value{}, table.Value{}
	}
	if s.src.Loading {
		return table.Value{Else: "…"}, table.Value{}
	}
	for _, p := range s.pulls {
		if p.Branch == branch && branch != "" {
			return table.Value{Raw: []string{ref(p)}, Link: jumpTo("changes", ref(p))},
				table.Value{Raw: []string{p.State}, Link: m.origin.pull(p.Number, "")}
		}
	}
	if desk || branch == "" {
		return table.Value{}, table.Value{}
	}
	return table.Value{}, table.Value{Else: "no pull request yet"}
}

// ref gives a pull request's reference: what kit printed, else #<number>.
func ref(p pull) string {
	if p.Ref != "" {
		return p.Ref
	}
	return "#" + strconv.Itoa(p.Number)
}

// changeRows gives each pull request: number, title, author, state, age, checks, threads, labels, branch; open, the
// title wrapped and every label, then the issues it closes, one row each, in flight. The pull request's glyph and its state's lead to its
// page. A pull request belongs to its author.
func (m Model) changeRows(s stage) []record {
	var out []record
	for _, p := range s.pulls {
		page := m.origin.pull(p.Number, "")
		var checked table.Value
		if p.Checks != nil {
			checked = table.Value{Raw: []string{p.Checks.State}, Of: p.Checks}
		}
		r := record{row{Key: ref(p)}, map[string]table.Value{
			"change": {Raw: []string{ref(p)}, Link: page}, "title": table.One(p.Title), "author": table.Some(p.Author),
			"state": {Raw: []string{p.State}, Link: page}, "age": table.Seconds(&p.Age), "checks": checked,
			"threads": table.One(strconv.Itoa(p.Unresolved)), "branch": table.Some(p.Branch), "labels": table.Labelled(p.Labels),
		}}
		for _, i := range p.Closes {
			r.Detail = append(r.Detail, []cell{number("#" + strconv.Itoa(i.Number)), owners(i.Assignees),
				text(InFlight + " in flight"), dimNumber(ageOf(i.Age)), fill(Issue + " " + i.Title)})
		}
		if len(p.Closes) == 0 {
			r.Detail = append(r.Detail, span(style.Off("closes no issue")))
		}
		out = append(out, r)
	}
	return out
}

// buildRows gives each build, newest first as kit prints them: its state's glyph, workflow, number, how it began,
// branch or tag, commit, title, age, and how long it took; the number, how it began, and the commit dim, and a title
// that only repeats the workflow left out; the state's glyph leads to the run's page. Open, the title wrapped, the
// attempt past the first, and its jobs, the failed first, each with its glyph and how long it took and, under a failed
// one, the step that failed.
func (m Model) buildRows(s stage) []record {
	var out []record
	for _, b := range s.runs {
		number, title := "", b.Title
		if b.Number != nil {
			number = "#" + strconv.FormatInt(*b.Number, 10)
		}
		if title == b.Workflow {
			title = ""
		}
		lasted := table.Seconds(b.Took)
		if b.Took == nil {
			lasted.Else = table.Took(nil)
		}
		r := record{row{Key: strconv.FormatInt(b.ID, 10)}, map[string]table.Value{
			"status": {Raw: []string{b.State}, Link: b.URL}, "workflow": table.One(b.Workflow), "run": table.Some(number),
			"event": table.Some(eventOf(b)), "branch": table.One(b.Branch), "commit": table.Some(b.SHA), "title": table.One(title),
			"age": table.Seconds(&b.Age), "took": lasted,
		}}
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
		out = append(out, r)
	}
	return out
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

// releaseRows gives each release: version, kind, age, latest, the kind's glyph leading to the release's page; open,
// the name, the tag, and the date.
func (m Model) releaseRows(s stage) []record {
	var out []record
	for _, rel := range s.rels {
		var latest table.Value
		if rel.Latest {
			latest = table.One("latest")
		}
		r := record{row{Key: rel.Tag}, map[string]table.Value{
			"version": table.One(rel.Version), "kind": {Raw: []string{rel.Kind}, Link: m.origin.release(rel.Tag, rel.URL)},
			"age": table.Seconds(&rel.Age), "latest": latest,
		}}
		r.Detail = [][]cell{span(rel.Name), span(style.Off(rel.Tag)), span(style.Off(rel.At))}
		out = append(out, r)
	}
	return out
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
