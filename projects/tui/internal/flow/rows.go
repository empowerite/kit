// Each stage's JSON, as kit flow <stage> --json prints it, and the rows a stage draws from it: what a row holds in each
// of the stage's fields, by the field's id, which its renderer draws in the field's column, and what it opens to.
package flow

import (
	"encoding/json"
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
	Member       string        `json:"member"` // over a group, the member kit read it in, host/path
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
		Source string `json:"source"`
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
	Member   string `json:"member"`
}

// checkout is one checkout, as a member of the workspace has its desk: its branch, or none where it is detached, and
// the commits it is ahead and behind its upstream.
type checkout struct {
	Branch   string `json:"branch"`
	Detached bool   `json:"detached"`
	Ahead    int    `json:"ahead"`
	Behind   int    `json:"behind"`
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
	Member     string        `json:"member"`
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
	Member   string `json:"member"`
}

// job is one job of a build: its name, its state as a build's, how long it took, nil while it runs, and the step that
// failed, when one did.
type job struct {
	Name  string `json:"name"`
	State string `json:"state"`
	Took  *int64 `json:"took"`
	Step  string `json:"step"`
}

// release is one release on the forge, as kit flow prints it, a row flat, the types its values must decode as; the
// catalog draws its rows from their raw JSON, as rowRecords reads them.
type release struct {
	Key       string         `json:"key"`
	Version   string         `json:"version"`
	Tag       string         `json:"tag"`
	Name      string         `json:"name"`
	Kind      string         `json:"kind"`
	Latest    *string        `json:"latest"`
	At        string         `json:"at"`
	Age       int64          `json:"age"`
	URL       string         `json:"url"`
	Notes     string         `json:"notes"`
	Assets    []string       `json:"assets"`
	Downloads []string       `json:"downloads"`
	Links     map[string]any `json:"links"`
	Member    string         `json:"member"`
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

// laid gives a stage's rows from its records, as table.Lay lays them out in the stage's fields.
func laid(stage string, records []record, pinned func(id string) string) []row {
	return table.Lay(fieldsOf(stage), entries(records), pinned)
}

// entries gives the records as internal/table holds them: each its row and its values.
func entries(records []record) []table.Record {
	out := make([]table.Record, len(records))
	for k, r := range records {
		out[k] = table.Record{Row: r.row, Values: r.values}
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

// build gives a stage's rows from what it loaded, as its panel lets them through, and what the panel shows of them,
// as table.Build builds them.
func (m Model) build(i int) shown {
	s, p := m.stages[i], m.filter(i)
	b := table.Build(catalog[s.name], entries(m.records(s)), p, m.names(i), m.paneWidth())
	return shown{b.Rows, b.Names, b.Levels, b.All, b.Drawn}
}

// builder is how Go builds the rows of a stage it builds: decode gives its records as Go holds them, fresh, and rows
// its records from them; a builder without rows has its records drawn from the catalog, as catalogRows draws them,
// once they decode, as releases'.
type builder struct {
	decode func([]json.RawMessage) (any, error)
	rows   func(Model, stage) []record
}

// builders are the stages whose rows Go builds, by name: kit flow's but incoming, whose rows kit prints flat, and the
// repos tab. Every other stage's rows are drawn from the catalog alone.
var builders = map[string]builder{
	"backlog":         {decoding[issue], Model.backlogRows},
	"underway":        {decoding[underway], Model.underwayRows},
	"changes":         {decoding[pull], Model.changeRows},
	"builds":          {decoding[run], Model.buildRows},
	"releases":        {decoding[release], nil},
	"workspace/repos": {decoding[repo], Model.repoRows},
}

// decoding gives each record decoded into a fresh T, so no field of one record, nor of a load before, stands in
// another; a record that does not decode as a T is the error.
func decoding[T any](records []json.RawMessage) (any, error) {
	out := make([]T, len(records))
	for k, r := range records {
		if err := json.Unmarshal(r, &out[k]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// of gives a stage's records as Go holds them, as its builder decoded them; none for a stage of another T, or one that
// has not loaded.
func of[T any](s stage) []T {
	out, _ := s.typed.([]T)
	return out
}

// records gives a stage's records from what it loaded, as its builder builds them, else as the catalog draws them,
// then, over a group, a line for each member whose read failed; none for a stage kittui does not draw.
func (m Model) records(s stage) []record {
	var out []record
	if b := builders[s.name]; b.rows != nil {
		out = b.rows(m, s)
	} else {
		out = m.catalogRows(s)
	}
	for _, r := range m.failures(s) {
		out = append(out, record{row: r})
	}
	return out
}

// failures gives a line for each member whose read of the stage failed, over a group: ❌, the member's short name, and
// kit's line, in red, an aside, as a remote whose fetch failed is, which a filter keeps, so a member filtered to shows
// why it has no rows.
func (m Model) failures(s stage) []row {
	var out []row
	for _, f := range s.failed {
		out = append(out, row{Key: m.keyOf(f.Member, "failed"),
			Cells: span(style.Refusal(Failed + " " + m.short(f.Member) + ": " + f.Error)), Aside: true, Lone: true})
	}
	return out
}

// keyOf gives a row's key: the stage's own over a repository, and after the row's member over a group, so the rows of
// two members that share a number stay apart.
func (m Model) keyOf(member, key string) string {
	if member == "" {
		return key
	}
	return member + " " + key
}

// memberValue gives a row's value in member: none over a repository; over a group, the member, host/path, drawn by its
// short name, leading to its row on the repos tab, where the screen reaches one.
func (m Model) memberValue(member string) table.Value {
	if member == "" {
		return table.Value{}
	}
	v := table.Value{Raw: []string{member}, Of: m.short(member)}
	if m.reaches("workspace/repos") {
		v.Link = jumpTo("workspace/repos", member)
	}
	return v
}

// originOf gives the origin a row's links are built from: the screen's, for a row of no member, else the member's
// host and path.
func (m Model) originOf(member string) Origin {
	if member == "" {
		return m.origin
	}
	return parseRemote("https://" + member)
}

// short gives a member's short name: its path past the owner where every member of the workspace shares its host and
// owner, else its host and path, as kit names it, until kit has said the members.
func (m Model) short(member string) string {
	owner := func(r string) string {
		parts := strings.SplitN(r, "/", 3)
		if len(parts) < 3 {
			return ""
		}
		return parts[0] + "/" + parts[1]
	}
	of := owner(member)
	if of == "" || len(m.ws.Members) == 0 {
		return member
	}
	for _, w := range m.ws.Members {
		if owner(w.Repository) != of {
			return member
		}
	}
	return strings.TrimPrefix(member, of+"/")
}

// backlogRows gives each issue: number, source, organization, repository, assignees, age, branch, labels, title, its
// glyph leading to the issue's page; open, every assignee, the title wrapped, and every label. An issue belongs to its assignees.
func (m Model) backlogRows(s stage) []record {
	var out []record
	for _, i := range of[issue](s) {
		key := "#" + strconv.Itoa(i.Number)
		out = append(out, record{row{Key: m.keyOf(i.Member, key)}, map[string]table.Value{
			"id": table.One(key), "source": table.Some(i.Source), "organization": table.Some(i.Organization),
			"repository": table.Some(i.Repository), "assignees": {Raw: i.Assignees}, "age": table.Seconds(i.Age),
			"branch": table.Some(i.Branch), "labels": table.Labelled(i.Labels),
			"title":  {Raw: []string{i.Title}, Link: m.originOf(i.Member).issue(i.Number, i.URL)},
			"member": m.memberValue(i.Member),
		}})
	}
	return out
}

// underwayRows gives each issue underway beside the pull request that closes it, oldest first as kit prints them:
// number, source, organization, repository, assignees, age, the pull request's number and state, where the work is, the fork its branch is in,
// when last pushed, its age right-aligned after the glyph, by whom when not the author, the issue's labels, and the
// title. Where is here
// when a checkout on this desk has the branch, as here draws it, local when this desk has the branch but no checkout of
// it, else the author. The pull request's glyph leads to its row under changes, or to its page when no app draws
// changes; the state's glyph to its page, and the title's to the issue's. Open, every assignee and the issue's
// title wrapped; the pull request's title is under changes, and the checkout's path on Repositories' worktrees
// tab. A row belongs to the issue's assignees, whoever opened the pull request, as kit flow --mine says.
func (m Model) underwayRows(s stage) []record {
	var out []record
	for _, u := range of[underway](s) {
		o := m.originOf(u.Member)
		issue, pr, page := "#"+strconv.Itoa(u.Number), "#"+strconv.Itoa(u.PR.Number), o.pull(u.PR.Number, u.PR.URL)
		var where, pushed, by table.Value
		switch {
		case u.Here:
			where = m.here(u.Member, u.Worktree)
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
		if m.reaches("changes") {
			change = jumpTo("changes", m.pullKey(u.Member, u.PR.Source, u.PR.Number))
		}
		out = append(out, record{row{Key: m.keyOf(u.Member, issue+" "+pr)}, map[string]table.Value{
			"id": table.One(issue), "source": table.Some(u.Source), "organization": table.Some(u.Organization),
			"repository": table.Some(u.Repository), "assignees": {Raw: u.Assignees}, "age": table.Seconds(u.Age),
			"change": {Raw: []string{pr}, Link: change}, "state": {Raw: []string{u.PR.State}, Link: page},
			"where": where, "fork": table.Some(u.PR.Fork), "pushed": pushed, "by": by, "labels": table.Labelled(u.Labels),
			"title": {Raw: []string{u.Title}, Link: o.issue(u.Number, u.URL)}, "member": m.memberValue(u.Member),
		}})
	}
	return out
}

// here gives underway's where for work in a member's checkout on this desk, at path: the desk, when the path is the
// member's desk, as deskOf says, else here, a worktree; either way leading to its row on Repositories' worktrees tab,
// kit workspace's, where the screen reaches it, else nowhere.
func (m Model) here(member, path string) table.Value {
	where := "here"
	if d := m.deskOf(member); d != "" && d == path {
		where = "desk"
	}
	v := table.Value{Raw: []string{where}, Of: where}
	if path != "" && m.reaches("workspace/worktrees") {
		v.Link = jumpTo("workspace/worktrees", m.keyOf(member, path))
	}
	return v
}

// deskOf gives a member's desk, its checkout, as checkoutOf says, and the workspace's checkout for a row of no member,
// over a repository; none until kit has said the workspace.
func (m Model) deskOf(member string) string {
	if member == "" {
		return m.ws.Checkout
	}
	return m.checkoutOf(member)
}

// jumps is how a link to a row of another stage begins, which a click follows on the screen, not in the browser.
const jumps = "kittui:"

// jumpTo gives the link to a row of a stage: the stage's name and the row's key.
func jumpTo(stage, key string) string { return jumps + stage + "/" + key }

// pullKey gives the key of a member's pull request's row under changes: its reference as kit printed it, else, as
// where changes is another app's, #<number>, or !<number> for a GitLab merge request, after the member over a group.
func (m Model) pullKey(member, source string, number int) string {
	if s := m.find("changes"); s != nil {
		for _, p := range of[pull](*s) {
			if p.Number == number && p.Member == member {
				return m.keyOf(member, ref(p))
			}
		}
	}
	if source == "gitlab" {
		return m.keyOf(member, "!"+strconv.Itoa(number))
	}
	return m.keyOf(member, "#"+strconv.Itoa(number))
}

// counts gives "↑a ↓b", ahead right-aligned to the widest in the table: ahead green and behind red, a zero dim.
func counts(c checkout, aheadWidth int) string {
	ahead := "↑" + strconv.Itoa(c.Ahead)
	pad := strings.Repeat(" ", max(0, aheadWidth-len(ahead)))
	return pad + count(ahead, c.Ahead, style.Gain) + " " + count("↓"+strconv.Itoa(c.Behind), c.Behind, style.Loss)
}

// count gives a count's text in its color, dim when it is zero.
func count(text string, n int, paint func(string) string) string {
	if n == 0 {
		return style.Off(text)
	}
	return paint(text)
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
	for _, p := range of[pull](s) {
		page := m.originOf(p.Member).pull(p.Number, "")
		var checked table.Value
		if p.Checks != nil {
			checked = table.Value{Raw: []string{p.Checks.State}, Of: p.Checks}
		}
		r := record{row{Key: m.keyOf(p.Member, ref(p))}, map[string]table.Value{
			"change": {Raw: []string{ref(p)}, Link: page}, "title": table.One(p.Title), "author": table.Some(p.Author),
			"state": {Raw: []string{p.State}, Link: page}, "age": table.Seconds(&p.Age), "checks": checked,
			"threads": table.One(strconv.Itoa(p.Unresolved)), "branch": table.Some(p.Branch), "labels": table.Labelled(p.Labels),
			"member": m.memberValue(p.Member),
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
	for _, b := range of[run](s) {
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
		r := record{row{Key: m.keyOf(b.Member, strconv.FormatInt(b.ID, 10))}, map[string]table.Value{
			"status": {Raw: []string{b.State}, Link: b.URL}, "workflow": table.One(b.Workflow), "run": table.Some(number),
			"event": table.Some(eventOf(b)), "branch": table.One(b.Branch), "commit": table.Some(b.SHA), "title": table.One(title),
			"age": table.Seconds(&b.Age), "took": lasted, "member": m.memberValue(b.Member),
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

// empty gives the dim line an empty stage shows: its catalog entry's, else nothing.
func empty(s stage) string {
	if c := catalog[s.name]; c != nil && c.Empty != "" {
		return c.Empty
	}
	return "nothing"
}
