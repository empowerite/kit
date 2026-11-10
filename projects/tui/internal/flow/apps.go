// kittui's apps, each built around one entity: its first tab is that entity's home, and its other tabs the entities
// around it. Each app is this screen over its own stages, kit flow's and kit workspace's together, and a tab whose
// entity kit has no data source for yet is planned: it shows that it has none, and loads nothing.
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

// apps are kittui's apps, in the app menu's order, each with its stages in its tabs' order: kit flow's by their own
// names, kit workspace's after workspace/, and those planned after planned/.
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
}

// IsApp says whether a screen is one of kittui's apps.
func IsApp(s shell.Screen) bool { return stagesIn(s) != nil }

// stagesIn gives an app's stages, or none for a screen that is not an app.
func stagesIn(s shell.Screen) []string {
	for _, a := range apps {
		if a.screen == s {
			return a.stages
		}
	}
	return nil
}

// AppOf gives the app kit flow's or kit workspace's words open on, and the words it takes: the app with a tab of a
// stage named among them, else Issues for kit flow and Repositories for kit workspace. kit flow's desk and worktrees
// are both the worktrees tab, the desk first.
func AppOf(verb string, words []string) (shell.Screen, []string) {
	words = slices.Clone(words)
	for k, w := range words {
		if verb != workspaceVerb && w == "desk" {
			words[k] = "worktrees"
		}
	}
	for _, w := range words {
		for _, a := range apps {
			if slices.ContainsFunc(a.stages, func(name string) bool { return label(name) == w }) {
				return a.screen, words
			}
		}
	}
	if verb == workspaceVerb {
		return shell.Repositories, words
	}
	return shell.Issues, words
}

// OpenApp opens an app: its stages that the repository's picture draws, as kit flow --stages and kit workspace
// --stages print them, and every one planned. A stage named among the words is the one it opens on, maximized, as
// Open says. Kit's refusal of --stages is the error.
func OpenApp(r kit.Runner, app shell.Screen, words []string) (Model, error) {
	drawn := map[string][]string{}
	var stages []string
	for _, name := range stagesIn(app) {
		verb, stage := "flow", name
		switch {
		case ofPlanned(name):
			stages = append(stages, name)
			continue
		case ofWorkspace(name):
			verb, stage = workspaceVerb, label(name)
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

// appWith gives the app with the stage a link names, its text after kittui:, the stage's name, a slash, and the row's
// key; none when no app draws it.
func appWith(to string) shell.Screen {
	for _, a := range apps {
		if slices.ContainsFunc(a.stages, func(name string) bool { return strings.HasPrefix(to, name+"/") }) {
			return a.screen
		}
	}
	return ""
}

// reaches says whether a link to a row of the stage leads anywhere: the stage is this screen's, or, in an app,
// another app's.
func (m Model) reaches(name string) bool {
	return m.find(name) != nil || (m.app != "" && appWith(name+"/") != "")
}

// verbsOf gives the verbs whose stages are among the stages, kit flow's and kit workspace's, each once.
func verbsOf(stages []string) []string {
	var verbs []string
	for _, name := range stages {
		verb := "flow"
		switch {
		case ofPlanned(name):
			continue
		case ofWorkspace(name):
			verb = workspaceVerb
		}
		if !slices.Contains(verbs, verb) {
			verbs = append(verbs, verb)
		}
	}
	return verbs
}
