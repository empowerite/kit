// kittui's apps: Flow, the picture kit flow draws, every stage of the flow setting in its order; and the apps built
// around one entity each, whose first tab is that entity's home, and its other tabs the entities around it. Each app
// is this screen over its own stages, kit flow's and its own verb's together, each tab a stage of kit's, read as its
// catalog entry says.
package flow

import (
	"slices"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// entityApp is an app built around one entity: its screen; the verb besides kit flow that opens it, and whose stages
// it draws; and its stages in its tabs' order, kit flow's by their own names, the verb's after it and a slash, as each
// one's catalog entry's command names it.
type entityApp struct {
	screen shell.Screen
	verb   string
	stages []string
}

// apps are kittui's apps built around one entity, in the app menu's order.
var apps = []entityApp{
	{shell.Issues, "issues", []string{"backlog", "underway", "issues/milestones", "issues/iterations"}},
	{shell.Changes, "changes", []string{"changes", "changes/reviews", "changes/checks", "changes/merged"}},
	{shell.Repositories, workspaceVerb, []string{"workspace/repos", "workspace/worktrees", "workspace/branches",
		"workspace/commits", "workspace/tags", "workspace/remotes", "incoming"}},
	{shell.Delivery, "delivery", []string{"releases", "builds", "delivery/artifacts", "delivery/environments",
		"delivery/deployments"}},
	{shell.People, "people", []string{"people/people", "people/teams", "people/agents"}},
	{shell.Trust, "audit", []string{"audit/evidence", "audit/findings", "audit/controls", "audit/waivers",
		"audit/policy"}},
	{shell.Platform, "platform", []string{"platform/services", "platform/toolchains", "platform/tools",
		"platform/settings"}},
	{shell.Packages, packagesVerb, []string{"packages/packages"}},
}

// IsApp says whether a screen is one of kittui's apps: Flow, or one built around an entity.
func IsApp(s shell.Screen) bool { return s == shell.Flow || stagesIn(s) != nil }

// stagesIn gives the stages of an app built around an entity, or none for another screen, Flow's included, whose
// stages are kit flow's.
func stagesIn(s shell.Screen) []string {
	for _, a := range apps {
		if a.screen == s {
			return a.stages
		}
	}
	return nil
}

// AppOf gives the app kit flow's or an app verb's words open on, and the words: Flow for kit flow, which opens on
// the stage named among them; for an app verb, the app with a tab of a stage named among them, else the verb's home,
// the app whose verb it is.
func AppOf(verb string, words []string) (shell.Screen, []string) {
	words = slices.Clone(words)
	home := slices.IndexFunc(apps, func(a entityApp) bool { return a.verb == verb })
	if home < 0 {
		return shell.Flow, words
	}
	for _, w := range words {
		for _, a := range apps {
			if slices.ContainsFunc(a.stages, func(name string) bool { return label(name) == w }) {
				return a.screen, words
			}
		}
	}
	return apps[home].screen, words
}

// AppVerb says whether a verb besides kit flow opens one of kittui's apps, as kit workspace opens Repositories and kit
// audit Trust.
func AppVerb(verb string) bool {
	return slices.ContainsFunc(apps, func(a entityApp) bool { return a.verb == verb })
}

// OpenApp opens an app: Flow on every stage kit flow --stages prints that kittui draws, in its order, as kit flow's
// desk and worktrees are not; another on its stages that the repository's picture draws, each where its verb's
// --stages prints it, as its catalog entry's command names the verb and the stage; every one is staged. A stage named
// among the words is the one it opens on, maximized, as open says. Kit's refusal of --stages is the error.
func OpenApp(r kit.Runner, app shell.Screen, words Words) (Model, error) {
	if app == shell.Flow {
		stages, err := r.StagesOf("flow")
		if err != nil {
			return Model{}, err
		}
		stages = slices.DeleteFunc(stages, func(name string) bool { return !sourced(name) })
		return open(r, Options{App: app, Stages: stages}, words), nil
	}
	drawn := map[string][]string{}
	var stages []string
	for _, name := range stagesIn(app) {
		verb, stage := verbOf(name)
		if _, ok := drawn[verb]; !ok {
			names, err := r.StagesOf(verb)
			if err != nil {
				return Model{}, err
			}
			drawn[verb] = names
		}
		if slices.Contains(drawn[verb], stage) {
			stages = append(stages, name)
		}
	}
	return open(r, Options{App: app, Stages: stages}, words), nil
}

// Words are kit flow's words, as an app takes them: the tab it opens on, by the name the tab gives it; whether the
// fetch is skipped, as --no-fetch asks; whether each stage's first read is fresh, as --fresh asks; and the owner's
// flags, --all, --mine, --others, and --owner=<login>, which each command whose catalog entry takes owner is given.
type Words struct {
	Tab     string
	NoFetch bool
	Fresh   bool
	Owner   []string
}

// ParseWords gives kit flow's words, as kittui was started with them: --no-fetch, --fresh, the owner's flags, and the
// first other word that is no flag, as the tab; any other word, as --jq, which the picture on a terminal has no use
// for, is no word of the app's.
func ParseWords(words []string) Words {
	var w Words
	for _, word := range words {
		switch {
		case word == "--no-fetch":
			w.NoFetch = true
		case word == "--fresh":
			w.Fresh = true
		case word == "--all" || word == "--mine" || word == "--others" || strings.HasPrefix(word, "--owner="):
			w.Owner = append(w.Owner, word)
		case w.Tab == "" && !strings.HasPrefix(word, "-"):
			w.Tab = word
		}
	}
	return w
}

// Rest gives the words without the tab, as every app opened later takes them.
func (w Words) Rest() Words {
	w.Tab = ""
	return w
}

// Words gives kit flow's words the screen was opened with.
func (m Model) Words() Words { return m.words }

// appWith gives the app built around an entity with the stage a link names, its text after kittui:, the stage's name,
// a slash, and the row's key; none when no such app draws it. Flow is never the app a link leads to: a link from
// another app leads to the entity's own.
func appWith(to string) shell.Screen {
	for _, a := range apps {
		if slices.ContainsFunc(a.stages, func(name string) bool { return strings.HasPrefix(to, name+"/") }) {
			return a.screen
		}
	}
	return ""
}

// reaches says whether a link to a row of the stage leads anywhere: the stage is this screen's, or another app's.
func (m Model) reaches(name string) bool { return m.find(name) != nil || appWith(name+"/") != "" }

// verbsOf gives the verbs whose staged stages are among the stages, as their catalog entries' commands name them,
// each once; a verb none of whose stages among them kit ever marks, as kit people's, is not, as its --marks would
// print null for each.
func verbsOf(stages []string) []string {
	var verbs []string
	for _, name := range stages {
		verb, _ := verbOf(name)
		if verb != "" && everMarked(name) && !slices.Contains(verbs, verb) {
			verbs = append(verbs, verb)
		}
	}
	return verbs
}
