// The glyphs the picture draws, as kit help status's legend names them, and the legend ? shows. The legend test holds
// every one to that page, so bash and Go draw the same picture.
package status

// The glyphs, as the legend draws them.
const (
	Desk     = "🏠" // the desk, the root checkout
	Branch   = "🌿" // a linked worktree, or a branch
	Change   = "🔀" // a pull request
	Owner    = "👤" // an author, or an issue's assignees
	Issue    = "🎫" // an issue
	InFlight = "🛫" // an issue the pull request above it closes
	Gone     = "🔖" // a branch whose upstream is gone
	Tag      = "📌" // a new tag the fetch brought
	Detached = "❗" // a checkout on no branch
	Running  = "🏃" // checks still running
	Passed   = "✅" // every check passed
	Failed   = "❌" // a check failed, and a refusal
	Threads  = "💬" // unresolved review threads
	Release  = "📦" // a release
	Pre      = "🧪" // a pre-release
	Draft    = "📝" // a draft release
	Latest   = "⭐" // the latest release
	Other    = "🔳" // a pull request in any other state
)

// The legend's groups, in the order it draws them: the glyphs more than one stage draws, then each stage's own, in
// the order of the line of stages.
const (
	groupEvery     = "every stage"
	groupIncoming  = "incoming"
	groupWorktrees = "worktrees"
	groupChanges   = "changes"
	groupStates    = "changes: state"
	groupReleases  = "releases"
)

// entry is one line of the legend: a glyph, what it means, and the group it is drawn under. A state's meaning is the
// state word kit prints.
type entry struct {
	glyph, meaning, group string
}

// legend lists every glyph the picture draws, group by group; ? draws it, and stateGlyph reads the states from it.
var legend = []entry{
	{Branch, "a branch, or a linked worktree", groupEvery},
	{Owner, "an author, or an issue's assignees", groupEvery},
	{Issue, "an issue", groupEvery},
	{Change, "a pull request", groupEvery},
	{Gone, "a branch gone from origin", groupEvery},
	{Failed, "a check failed, or a stage kit refused", groupEvery},
	{Tag, "a new tag", groupIncoming},
	{Desk, "the desk, the root checkout", groupWorktrees},
	{Detached, "a checkout on no branch", groupWorktrees},
	{Threads, "unresolved review threads", groupChanges},
	{InFlight, "an issue the pull request closes", groupChanges},
	{Running, "checks still running", groupChanges},
	{Passed, "every check passed", groupChanges},
	{"🔨", "draft", groupStates},
	{"🚦", "queued", groupStates},
	{"💥", "conflicted", groupStates},
	{"🔥", "failing:gate", groupStates},
	{"📭", "no-check", groupStates},
	{"💬", "comments:open", groupStates},
	{"👀", "blocked:approval", groupStates},
	{"🛬", "ready", groupStates},
	{Other, "any other state", groupStates},
	{Release, "a release", groupReleases},
	{Pre, "a pre-release", groupReleases},
	{Draft, "a draft", groupReleases},
	{Latest, "the latest", groupReleases},
}

// stageGlyphs gives each stage's glyph, drawn beside its name on the line of stages.
var stageGlyphs = map[string]string{
	"backlog":   Issue,
	"incoming":  "📥",
	"worktrees": Branch,
	"changes":   Change,
	"releases":  Release,
}

// stateGlyphs gives a pull request's glyph by its state word, read from the legend's states but the last, any other.
var stateGlyphs = func() map[string]string {
	out := map[string]string{}
	for _, e := range legend {
		if e.group == groupStates && e.glyph != Other {
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
