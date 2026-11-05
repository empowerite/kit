// The glyphs the picture draws, and each stage's legend, which ? shows beside the stage's rows. The glyphs region is
// generated from the kit's one table of glyphs, kit/modules/jq/glyphs.json, which kit help flow's legend is made from
// too, so the two say the same of every glyph; the legend test holds what each stage draws to its legend.
package flow

// entry is one line of a legend: a glyph and what it means. A state's meaning is the state word kit prints.
type entry struct {
	glyph, meaning string
}

// The marks after a legend's meaning, dim: a click on the glyph goes to its row in another stage, or opens its page.
const (
	ToRow  = "†"
	ToPage = "🔗"
)

// kit:generate glyphs `jq -nr -L projects/tui/internal/flow 'import "legend" as l; l::go' | gofmt | sed 1,2d; echo`
// The glyphs, as the legend draws them, each beside its meaning.
const (
	Repository = "📚" // the repository
	Issue      = "🎫" // an issue
	GitHub     = "🐙" // on GitHub
	GitLab     = "🦊" // on GitLab
	InFlight   = "🛫" // underway
	Desk       = "💻" // the desk
	Branch     = "🌿" // a branch or a worktree
	Sprout     = "🌱" // a local branch
	Change     = "🔀" // a pull request
	Owner      = "👤" // assignees or author
	Fork       = "🍴" // a fork's owner
	Pushed     = "📤" // the last push
	Path       = "📂" // a desk's path
	Fetched    = "🛬" // the fetch
	Remote     = "📡" // a remote
	Gone       = "🔖" // a branch gone
	Tag        = "📌" // a new tag
	Detached   = "❗" // not on any branch
	Threads    = "💬" // threads unresolved
	Running    = "🏃" // running
	Passed     = "✅" // passed
	Failed     = "❌" // failed
	Cancelled  = "🛑" // cancelled or skipped
	Build      = "🏭" // a build
	Release    = "📦" // a release
	Pre        = "🧪" // a pre-release
	Draft      = "📝" // a draft release
	Latest     = "⭐" // the latest release
	Other      = "🔳" // any other state
)

// title is the title's glyph, which no stage draws.
var title = entry{Repository, "the repository"}

// legends gives each stage's legend: every glyph the stage draws, its own first. A glyph may be in several.
var legends = map[string][]entry{
	"backlog": {
		{Issue, "an issue"},
		{GitHub, "on GitHub"},
		{GitLab, "on GitLab"},
		{Owner, "assignees or author"},
		{Branch, "a branch or a worktree"},
		{Failed, "failed"},
	},
	"underway": {
		{InFlight, "underway"},
		{Owner, "assignees or author"},
		{Change, "a pull request"},
		{Desk, "the desk"},
		{Branch, "a branch or a worktree"},
		{Sprout, "a local branch"},
		{Fork, "a fork's owner"},
		{Pushed, "the last push"},
		{Issue, "an issue"},
		{GitHub, "on GitHub"},
		{GitLab, "on GitLab"},
		{Failed, "failed"},
	},
	"desk": {
		{Desk, "the desk"},
		{Detached, "not on any branch"},
		{Change, "a pull request"},
		{Path, "a desk's path"},
	},
	"worktrees": {
		{Branch, "a branch or a worktree"},
		{Detached, "not on any branch"},
		{Change, "a pull request"},
		{Gone, "a branch gone"},
	},
	"changes": {
		{Change, "a pull request"},
		{Owner, "assignees or author"},
		{Branch, "a branch or a worktree"},
		{Running, "running"},
		{Passed, "passed"},
		{Failed, "failed"},
		{Threads, "threads unresolved"},
		{InFlight, "underway"},
		{Issue, "an issue"},
	},
	"builds": {
		{Build, "a build"},
		{Running, "running"},
		{Passed, "passed"},
		{Failed, "failed"},
		{Cancelled, "cancelled or skipped"},
	},
	"incoming": {
		{Fetched, "the fetch"},
		{Remote, "a remote"},
		{Branch, "a branch or a worktree"},
		{Gone, "a branch gone"},
		{Tag, "a new tag"},
		{Failed, "failed"},
	},
	"releases": {
		{Release, "a release"},
		{Pre, "a pre-release"},
		{Draft, "a draft release"},
		{Latest, "the latest release"},
		{Failed, "failed"},
	},
}

// marks gives the mark after a glyph's meaning in a stage's legend, for each glyph a click on leads somewhere:
// to the row of another stage, or to a page.
var marks = map[string]map[string]string{
	"backlog":   {Issue: ToPage},
	"underway":  {Change: ToRow, Desk: ToRow, Branch: ToRow, Issue: ToPage},
	"desk":      {Change: ToRow},
	"worktrees": {Change: ToRow},
	"changes":   {Change: ToPage},
	"builds":    {Running: ToPage, Passed: ToPage, Failed: ToPage, Cancelled: ToPage},
	"releases":  {Release: ToPage, Pre: ToPage, Draft: ToPage},
}

// states gives a pull request's glyph by its state word, the last for any other; it hangs under changes' legend,
// under a heading marked ToPage, as a click on any of them opens the pull request's page.
var states = []entry{
	{"🔨", "draft"},
	{"🚦", "queued"},
	{"💥", "conflicted"},
	{"🔥", "failing:gate"},
	{"📭", "no-check"},
	{"💬", "comments:open"},
	{"👀", "blocked:approval"},
	{"🏁", "ready"},
	{Other, "any other state"},
}

// stageGlyphs gives each stage's glyph, drawn beside its name on the line of stages.
var stageGlyphs = map[string]string{
	"backlog":   Issue,
	"underway":  InFlight,
	"desk":      Desk,
	"worktrees": Branch,
	"changes":   Change,
	"builds":    Build,
	"incoming":  Fetched,
	"releases":  Release,
}

// stageColors gives each stage its group's color, which the bar draws the stage's tab in while it is selected.
var stageColors = map[string]string{
	"backlog":   "75",
	"underway":  "75",
	"desk":      "114",
	"worktrees": "114",
	"changes":   "221",
	"builds":    "221",
	"incoming":  "177",
	"releases":  "177",
}

// kit:end glyphs

// stateGlyphs gives a pull request's glyph by its state word, read from the states but the last, any other.
var stateGlyphs = func() map[string]string {
	out := map[string]string{}
	for _, e := range states {
		if e.glyph != Other {
			out[e.meaning] = e.glyph
		}
	}
	return out
}()

// stateGlyph gives a pull request's glyph; a state word the legend does not name draws Other.
func stateGlyph(state string) string {
	if g, ok := stateGlyphs[state]; ok {
		return g
	}
	return Other
}

// kindGlyph gives a release's glyph by its kind.
func kindGlyph(kind string) string {
	switch kind {
	case "draft":
		return Draft
	case "pre-release":
		return Pre
	}
	return Release
}

// runGlyph gives a build's glyph, or a job's, by its state; a state #48 does not name draws Other.
func runGlyph(state string) string {
	switch state {
	case "running":
		return Running
	case "passed":
		return Passed
	case "failed":
		return Failed
	case "cancelled":
		return Cancelled
	}
	return Other
}
