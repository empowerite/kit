// The glyphs the picture draws, as kit help status's legend names them. The legend test holds every one to that
// page, so bash and Go draw the same picture.
package status

// The glyphs, as the legend draws them.
const (
	Desk     = "🏠" // the desk, the root checkout
	Branch   = "🌿" // a linked worktree, or a branch
	Fetch    = "📥" // what the fetch brought from origin
	Change   = "🔀" // a pull request
	Owner    = "👤" // an author, or an issue's assignees
	Issue    = "🎫" // an issue
	InFlight = "🛫" // an issue the pull request above it closes
	Gone     = "🔖" // a branch whose upstream is gone
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

// states gives a pull request's glyph by its state word; a word not here draws Other.
var states = map[string]string{
	"draft":            "🔨",
	"queued":           "🚦",
	"conflicted":       "💥",
	"failing:gate":     "🔥",
	"no-check":         "📭",
	"comments:open":    "💬",
	"blocked:approval": "👀",
	"ready":            "🛬",
}

// legend lists every glyph the picture draws, for the test that holds them to kit help status.
var legend = []string{Desk, Branch, Fetch, Change, Owner, Issue, InFlight, Gone, Detached, Running, Passed, Failed,
	Threads, Release, Pre, Draft, Latest, Other}

// stateGlyph gives a pull request's glyph.
func stateGlyph(state string) string {
	if g, ok := states[state]; ok {
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
