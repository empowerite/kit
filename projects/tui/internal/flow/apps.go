// kittui's apps: Flow, the picture kit flow draws, every stage of the flow setting in its order; and the apps built
// around one entity each, whose first tab is that entity's home, and its other tabs the entities around it. Each app
// is this screen over its own stages, kit flow's and kit workspace's together, and a tab whose entity kit has no data
// source for yet is planned: it shows that it has none, and loads nothing.
package flow

import (
	"slices"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/shell"
)

// planned is how a stage kit has no data source for yet begins, before its tab's name.
const planned = "planned"

// ofPlanned says whether a stage, as kittui knows it, is planned.
func ofPlanned(name string) bool { return strings.HasPrefix(name, planned+"/") }

// apps are kittui's apps built around one entity, in the app menu's order, each with its stages in its tabs' order:
// kit flow's by their own names, kit workspace's after workspace/, and those planned after planned/.
var apps = []struct {
	screen shell.Screen
	stages []string
}{
	{shell.Issues, []string{"backlog", "underway", "planned/iterations", "planned/milestones"}},
	{shell.Changes, []string{"changes", "planned/reviews", "planned/checks", "planned/merged"}},
	{shell.Repositories, []string{"workspace/repos", "workspace/worktrees", "workspace/branches", "workspace/commits",
		"workspace/tags", "workspace/remotes", "incoming"}},
	{shell.Delivery, []string{"releases", "builds", "planned/artifacts", "planned/environments", "planned/deployments"}},
	{shell.People, []string{"planned/people", "planned/teams", "planned/agents"}},
	{shell.Trust, []string{"planned/evidence", "planned/findings", "planned/controls", "planned/waivers",
		"planned/policy"}},
	{shell.Platform, []string{"planned/services", "planned/toolchains", "planned/tools", "planned/settings"}},
	{shell.Packages, []string{"packages/packages"}},
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
// as homes says.
func AppOf(verb string, words []string) (shell.Screen, []string) {
	words = slices.Clone(words)
	if !slices.Contains(appVerbs, verb) {
		return shell.Flow, words
	}
	for _, w := range words {
		for _, a := range apps {
			if slices.ContainsFunc(a.stages, func(name string) bool { return label(name) == w }) {
				return a.screen, words
			}
		}
	}
	return homes[verb], words
}

// homes gives the app each app verb opens on with no tab of its named: Repositories for kit workspace, Packages for kit
// packages.
var homes = map[string]shell.Screen{workspaceVerb: shell.Repositories, packagesVerb: shell.Packages}

// OpenApp opens an app: Flow on every stage kit flow --stages prints that kittui draws, in its order, as kit flow's
// desk and worktrees are not; another on its stages that the repository's picture draws: a staged one where its verb's
// --stages prints it, as its catalog entry's command names the verb and the stage, one that is not staged, and every
// one planned. A stage named among the words is the one it opens on, maximized, as open says. Kit's refusal of
// --stages is the error.
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
		if verb == "" {
			stages = append(stages, name)
			continue
		}
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
// each once.
func verbsOf(stages []string) []string {
	var verbs []string
	for _, name := range stages {
		verb, _ := verbOf(name)
		if verb != "" && !slices.Contains(verbs, verb) {
			verbs = append(verbs, verb)
		}
	}
	return verbs
}
