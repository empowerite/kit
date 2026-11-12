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
	Workspace   = "📚" // workspace
	Issue       = "🎫" // issue
	GitHub      = "🐙" // on GitHub
	GitLab      = "🦊" // on GitLab
	Waiting     = "💤" // backlog
	InFlight    = "🛫" // underway
	Desk        = "💻" // desk
	Branch      = "🌿" // branch or worktree
	Sprout      = "🌱" // local branch
	Worktree    = "🌳" // worktree
	Commit      = "🔘" // commit
	Default     = "⚓" // default branch
	Change      = "🔀" // pull request
	Owner       = "👤" // assignees or author
	Fork        = "🍴" // fork owner
	Pushed      = "📤" // last push
	Path        = "📂" // desk path
	Fetched     = "🛬" // fetch
	Remote      = "📡" // remote
	Gone        = "🔖" // branch gone
	Tag         = "📌" // tag
	Behind      = "📥" // behind upstream
	Detached    = "❗" // no branch
	Threads     = "💬" // unresolved threads
	Running     = "🏃" // running
	Passed      = "✅" // passed
	Failed      = "❌" // failed
	Cancelled   = "🛑" // cancelled or skipped
	Build       = "🏭" // build
	Release     = "📦" // release
	Pre         = "🧪" // pre-release
	Draft       = "📝" // draft release
	Latest      = "⭐" // latest release
	Asset       = "📎" // asset
	Package     = "🧩" // package
	Active      = "🟢" // active
	Excluded    = "⛔" // excluded
	Milestone   = "⛳" // milestone
	Iteration   = "📅" // iteration
	Swimlane    = "🏊" // swimlane
	Review      = "🧐" // review
	Check       = "🩺" // check
	Merged      = "🤝" // merged
	Artifact    = "📀" // artifact
	Environment = "🏢" // environment
	Deployment  = "🚀" // deployment
	Person      = "🧑" // person
	Team        = "👪" // team
	Agent       = "🤖" // agent
	Evidence    = "🧾" // evidence
	Finding     = "🐛" // finding
	Control     = "🔒" // control
	Waiver      = "🔓" // waiver
	Policy      = "📏" // policy rule
	Enforcement = "🔐" // enforcement
	Service     = "🔌" // service
	Toolchain   = "🧰" // toolchain
	Usable      = "✅" // usable here
	Inside      = "⚡" // inside now
	InUse       = "📌" // kit uses it
	Adds        = "➕" // adds the rest
	Tool        = "🔩" // tool
	Setting     = "🔧" // setting
	Other       = "🔳" // other state
)

// title is the title's glyph, which no stage draws.
var title = entry{Workspace, "workspace"}

// legends gives each stage's legend: every glyph the stage draws, its own first. A glyph may be in several.
var legends = map[string][]entry{
	"issues": {
		{Issue, "issue"},
		{Waiting, "backlog"},
		{InFlight, "underway"},
		{Owner, "assignees or author"},
		{Milestone, "milestone"},
		{Iteration, "iteration"},
		{Swimlane, "swimlane"},
		{Change, "pull request"},
		{Desk, "desk"},
		{Branch, "branch or worktree"},
		{Sprout, "local branch"},
		{Fork, "fork owner"},
		{Pushed, "last push"},
		{GitHub, "on GitHub"},
		{GitLab, "on GitLab"},
		{Failed, "failed"},
	},
	"desk": {
		{Desk, "desk"},
		{Detached, "no branch"},
		{Change, "pull request"},
		{Path, "desk path"},
	},
	"worktrees": {
		{Branch, "branch or worktree"},
		{Detached, "no branch"},
		{Change, "pull request"},
		{Gone, "branch gone"},
	},
	"changes": {
		{Change, "pull request"},
		{Owner, "assignees or author"},
		{Branch, "branch or worktree"},
		{Running, "running"},
		{Passed, "passed"},
		{Failed, "failed"},
		{Threads, "unresolved threads"},
		{InFlight, "underway"},
		{Issue, "issue"},
	},
	"builds": {
		{Build, "build"},
		{Running, "running"},
		{Passed, "passed"},
		{Failed, "failed"},
		{Cancelled, "cancelled or skipped"},
	},
	"incoming": {
		{Fetched, "fetch"},
		{Remote, "remote"},
		{Commit, "commit"},
		{Branch, "branch or worktree"},
		{Gone, "branch gone"},
		{Tag, "tag"},
		{Behind, "behind upstream"},
		{Issue, "issue"},
		{Worktree, "worktree"},
		{Default, "default branch"},
		{GitHub, "on GitHub"},
		{GitLab, "on GitLab"},
		{Failed, "failed"},
	},
	"releases": {
		{Release, "release"},
		{Pre, "pre-release"},
		{Draft, "draft release"},
		{Latest, "latest release"},
		{Asset, "asset"},
		{Failed, "failed"},
	},
	"workspace/repos": {
		{Workspace, "workspace"},
		{GitHub, "on GitHub"},
		{GitLab, "on GitLab"},
		{Branch, "branch or worktree"},
		{Sprout, "local branch"},
		{Failed, "failed"},
	},
	"workspace/remotes": {
		{Remote, "remote"},
		{GitHub, "on GitHub"},
		{GitLab, "on GitLab"},
		{Default, "default branch"},
		{Failed, "failed"},
	},
	"workspace/branches": {
		{Branch, "branch or worktree"},
		{Sprout, "local branch"},
		{Gone, "branch gone"},
		{Default, "default branch"},
		{Worktree, "worktree"},
		{Commit, "commit"},
		{Failed, "failed"},
	},
	"workspace/worktrees": {
		{Desk, "desk"},
		{Worktree, "worktree"},
		{Branch, "branch or worktree"},
		{Detached, "no branch"},
		{Commit, "commit"},
		{Failed, "failed"},
	},
	"workspace/commits": {
		{Commit, "commit"},
		{Issue, "issue"},
		{Branch, "branch or worktree"},
		{Worktree, "worktree"},
		{Failed, "failed"},
	},
	"workspace/tags": {
		{Tag, "tag"},
		{Commit, "commit"},
		{Failed, "failed"},
	},
	"packages/packages": {
		{Package, "package"},
		{Active, "active"},
		{Excluded, "excluded"},
		{Failed, "failed"},
	},
	"issues/milestones": {
		{Milestone, "milestone"},
		{Failed, "failed"},
	},
	"issues/iterations": {
		{Iteration, "iteration"},
		{Failed, "failed"},
	},
	"changes/reviews": {
		{Review, "review"},
		{Change, "pull request"},
		{Threads, "unresolved threads"},
		{Failed, "failed"},
	},
	"changes/checks": {
		{Check, "check"},
		{Change, "pull request"},
		{Passed, "passed"},
		{Running, "running"},
		{Failed, "failed"},
	},
	"changes/merged": {
		{Merged, "merged"},
		{Change, "pull request"},
		{Owner, "assignees or author"},
		{Branch, "branch or worktree"},
		{Failed, "failed"},
	},
	"delivery/artifacts": {
		{Artifact, "artifact"},
		{Failed, "failed"},
	},
	"delivery/environments": {
		{Environment, "environment"},
		{Branch, "branch or worktree"},
		{Failed, "failed"},
	},
	"delivery/deployments": {
		{Deployment, "deployment"},
		{Passed, "passed"},
		{Running, "running"},
		{Cancelled, "cancelled or skipped"},
		{Failed, "failed"},
		{Commit, "commit"},
	},
	"people/people": {
		{Person, "person"},
		{Failed, "failed"},
	},
	"people/teams": {
		{Team, "team"},
		{Failed, "failed"},
	},
	"people/agents": {
		{Agent, "agent"},
		{Failed, "failed"},
	},
	"audit/evidence": {
		{Evidence, "evidence"},
		{Failed, "failed"},
	},
	"audit/findings": {
		{Finding, "finding"},
		{Failed, "failed"},
	},
	"audit/controls": {
		{Control, "control"},
		{Failed, "failed"},
	},
	"audit/waivers": {
		{Waiver, "waiver"},
		{Failed, "failed"},
	},
	"audit/policy": {
		{Policy, "policy rule"},
		{Failed, "failed"},
	},
	"audit/enforcement": {
		{Enforcement, "enforcement"},
		{Failed, "failed"},
	},
	"platform/services": {
		{Service, "service"},
		{Failed, "failed"},
	},
	"platform/toolchains": {
		{Toolchain, "toolchain"},
		{Usable, "usable here"},
		{Inside, "inside now"},
		{InUse, "kit uses it"},
		{Adds, "adds the rest"},
		{Failed, "failed"},
	},
	"platform/tools": {
		{Tool, "tool"},
		{Failed, "failed"},
	},
	"platform/settings": {
		{Setting, "setting"},
		{Failed, "failed"},
	},
}

// marks gives the mark after a glyph's meaning in a stage's legend, for each glyph a click on leads somewhere:
// to the row of another stage, or to a page.
var marks = map[string]map[string]string{
	"issues":               {Issue: ToPage, Change: ToRow, Desk: ToRow, Branch: ToRow},
	"desk":                 {Change: ToRow},
	"worktrees":            {Change: ToRow},
	"changes":              {Change: ToPage},
	"builds":               {Running: ToPage, Passed: ToPage, Failed: ToPage, Cancelled: ToPage},
	"releases":             {Release: ToPage, Pre: ToPage, Draft: ToPage, Asset: ToPage},
	"workspace/remotes":    {GitHub: ToPage, GitLab: ToPage, Default: ToRow},
	"workspace/branches":   {Branch: ToPage, Worktree: ToRow, Commit: ToPage},
	"workspace/worktrees":  {Branch: ToRow},
	"workspace/commits":    {Commit: ToPage, Issue: ToPage, Worktree: ToRow},
	"delivery/deployments": {Passed: ToPage, Running: ToPage, Cancelled: ToPage, Failed: ToPage, Commit: ToPage},
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
	{Other, "other state"},
}

// stageGlyphs gives each stage's glyph, drawn beside its name on the line of stages.
var stageGlyphs = map[string]string{
	"issues":                Issue,
	"desk":                  Desk,
	"worktrees":             Branch,
	"changes":               Change,
	"builds":                Build,
	"incoming":              Fetched,
	"releases":              Release,
	"workspace/repos":       Workspace,
	"workspace/remotes":     Remote,
	"workspace/branches":    Branch,
	"workspace/worktrees":   Worktree,
	"workspace/commits":     Commit,
	"workspace/tags":        Tag,
	"packages/packages":     Package,
	"issues/milestones":     Milestone,
	"issues/iterations":     Iteration,
	"changes/reviews":       Review,
	"changes/checks":        Check,
	"changes/merged":        Merged,
	"delivery/artifacts":    Artifact,
	"delivery/environments": Environment,
	"delivery/deployments":  Deployment,
	"people/people":         Person,
	"people/teams":          Team,
	"people/agents":         Agent,
	"audit/evidence":        Evidence,
	"audit/findings":        Finding,
	"audit/controls":        Control,
	"audit/waivers":         Waiver,
	"audit/policy":          Policy,
	"audit/enforcement":     Enforcement,
	"platform/services":     Service,
	"platform/toolchains":   Toolchain,
	"platform/tools":        Tool,
	"platform/settings":     Setting,
}

// stageColors gives each stage its group's color, which the bar draws the stage's tab in while it is selected.
var stageColors = map[string]string{
	"issues":                "75",
	"desk":                  "114",
	"worktrees":             "114",
	"changes":               "221",
	"builds":                "221",
	"incoming":              "177",
	"releases":              "177",
	"workspace/repos":       "13",
	"workspace/remotes":     "114",
	"workspace/branches":    "114",
	"workspace/worktrees":   "75",
	"workspace/commits":     "177",
	"workspace/tags":        "177",
	"packages/packages":     "215",
	"issues/milestones":     "75",
	"issues/iterations":     "75",
	"changes/reviews":       "221",
	"changes/checks":        "221",
	"changes/merged":        "177",
	"delivery/artifacts":    "177",
	"delivery/environments": "215",
	"delivery/deployments":  "215",
	"people/people":         "114",
	"people/teams":          "114",
	"people/agents":         "13",
	"audit/evidence":        "75",
	"audit/findings":        "203",
	"audit/waivers":         "203",
	"audit/controls":        "221",
	"audit/policy":          "221",
	"audit/enforcement":     "221",
	"platform/services":     "114",
	"platform/toolchains":   "114",
	"platform/tools":        "177",
	"platform/settings":     "177",
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
