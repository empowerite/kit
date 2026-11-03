// The glyphs the picture draws, as kit help flow's legend names them, and each stage's legend, which ? shows beside
// the stage's rows. The legend test holds every one to that page, so bash and Go draw the same picture.
package flow

// The glyphs, as the legend draws them.
const (
	Repository = "📚" // the repository, owner/name, in the title
	Fetched    = "📥" // what the fetch brought from the remotes
	Remote     = "📡" // a remote, its forge, and its host
	Desk       = "💻" // the desk, the root checkout
	Branch     = "🌿" // a linked worktree, or a branch
	Change     = "🔀" // a pull request
	Owner      = "👤" // an author, or an issue's assignees
	Issue      = "🎫" // an issue
	InFlight   = "🛫" // an issue a pull request closes: underway, and in flight under changes
	Gone       = "🔖" // a branch whose upstream is gone
	Tag        = "📌" // a new tag the fetch brought
	Detached   = "❗" // a checkout not on any branch
	Running    = "🏃" // checks still running
	Passed     = "✅" // every check passed
	Failed     = "❌" // a check failed, and a refusal
	Threads    = "💬" // unresolved review threads
	Release    = "📦" // a release
	Pre        = "🧪" // a pre-release
	Draft      = "📝" // a draft release
	Latest     = "⭐" // the latest release
	Other      = "🔳" // a pull request in any other state

	// A build's glyphs; the variation selector draws them as emoji, two cells wide.
	Build     = "🏭" // a build: a workflow run not owned by any pull request
	Cancelled = "🛑" // a build cancelled or skipped
)

// entry is one line of a legend: a glyph and what it means. A state's meaning is the state word kit prints.
type entry struct {
	glyph, meaning string
}

// title is the title's glyph, which no stage draws.
var title = entry{Repository, "the repository, owner/name"}

// legends gives each stage's legend: every glyph the stage draws, its own first. A glyph may be in several.
// Underway, the desk, and the worktrees draw a pull request's state as changes does; its glyphs are under changes
// alone.
var legends = map[string][]entry{
	"backlog": {
		{Issue, "an issue"},
		{Owner, "its assignees"},
		{Branch, "its branch"},
	},
	"underway": {
		{InFlight, "an issue underway"},
		{Owner, "its assignees, or the author"},
		{Change, "the pull request"},
		{Desk, "here, in a worktree"},
		{Issue, "the issue's title"},
	},
	"incoming": {
		{Fetched, "the fetch"},
		{Remote, "a remote"},
		{Branch, "a branch"},
		{Gone, "a branch gone"},
		{Tag, "a new tag"},
		{Failed, "a fetch failed"},
	},
	"desk": {
		{Desk, "the desk"},
		{Detached, "not on any branch"},
		{Change, "its pull request"},
	},
	"worktrees": {
		{Branch, "a worktree"},
		{Detached, "not on any branch"},
		{Change, "its pull request"},
		{Gone, "a branch gone"},
	},
	"changes": {
		{Change, "a pull request"},
		{Owner, "its author"},
		{Branch, "its branch"},
		{Running, "checks running"},
		{Passed, "checks passed"},
		{Failed, "a check failed"},
		{Threads, "threads unresolved"},
		{InFlight, "an issue it closes"},
		{Issue, "the issue's title"},
	},
	"builds": {
		{Build, "a build"},
		{Running, "running or queued"},
		{Passed, "passed"},
		{Failed, "failed"},
		{Cancelled, "cancelled or skipped"},
	},
	"releases": {
		{Release, "a release"},
		{Pre, "a pre-release"},
		{Draft, "a draft"},
		{Latest, "the latest"},
	},
}

// states gives a pull request's glyph by its state word, the last for any other; it hangs under changes' legend.
var states = []entry{
	{"🔨", "draft"},
	{"🚦", "queued"},
	{"💥", "conflicted"},
	{"🔥", "failing:gate"},
	{"📭", "no-check"},
	{"💬", "comments:open"},
	{"👀", "blocked:approval"},
	{"🛬", "ready"},
	{Other, "any other state"},
}

// stageGlyphs gives each stage's glyph, drawn beside its name on the line of stages.
var stageGlyphs = map[string]string{
	"backlog":   Issue,
	"underway":  InFlight,
	"incoming":  Fetched,
	"desk":      Desk,
	"worktrees": Branch,
	"changes":   Change,
	"builds":    Build,
	"releases":  Release,
}

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
