// Origin is read from git's remote URL in each of its shapes, and gives the URLs the rows open to.
package flow

import (
	"os/exec"
	"testing"
)

func TestParseRemote(t *testing.T) {
	for url, want := range map[string]Origin{
		"https://github.com/empowerite/kit.git":   {Host: "github.com", Repo: "empowerite/kit"},
		"https://github.com/empowerite/kit":       {Host: "github.com", Repo: "empowerite/kit"},
		"ssh://git@github.com/empowerite/kit.git": {Host: "github.com", Repo: "empowerite/kit"},
		"git@gitlab.com:group/sub/kit.git":        {Host: "gitlab.com", Repo: "sub/kit"},
		"https://user@gitlab.example.com/a/b/c/":  {Host: "gitlab.example.com", Repo: "b/c"},
		"/home/someone/kit":                       {},
		"file:///home/someone/kit":                {Host: "", Repo: ""},
		"https://github.com/kit":                  {},
		"":                                        {},
	} {
		if got := parseRemote(url); got != want {
			t.Errorf("parseRemote(%q) = %+v, want %+v", url, got, want)
		}
	}
}

func TestTheURLsFollowTheHost(t *testing.T) {
	hub := Origin{Host: "github.com", Repo: "empowerite/kit"}
	lab := Origin{Host: "gitlab.com", Repo: "group/kit"}
	if got := hub.issue(45, ""); got != "https://github.com/empowerite/kit/issues/45" {
		t.Errorf("GitHub issue: %q", got)
	}
	if got := lab.issue(45, ""); got != "https://gitlab.com/group/kit/-/issues/45" {
		t.Errorf("GitLab issue: %q", got)
	}
	if got := hub.pull(2, ""); got != "https://github.com/empowerite/kit/pull/2" {
		t.Errorf("GitHub pull request: %q", got)
	}
	if got := lab.pull(2, ""); got != "https://gitlab.com/group/kit/-/merge_requests/2" {
		t.Errorf("GitLab merge request: %q", got)
	}
	// what kit printed wins, and an unknown origin gives nothing
	if got := hub.issue(1, "https://example.com/1"); got != "https://example.com/1" {
		t.Errorf("a given URL: %q", got)
	}
	if got := (Origin{}).issue(1, ""); got != "" {
		t.Errorf("no origin: %q", got)
	}
	if got := (Origin{}).pull(2, ""); got != "" {
		t.Errorf("no origin: %q", got)
	}
}

func TestReadOriginAsksGit(t *testing.T) {
	dir := t.TempDir()
	if got := ReadOrigin(dir); got != (Origin{}) {
		t.Errorf("no repository: %+v", got)
	}
	for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", "git@github.com:empowerite/kit.git"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if got := ReadOrigin(dir); got != (Origin{Host: "github.com", Repo: "empowerite/kit"}) {
		t.Errorf("ReadOrigin: %+v", got)
	}
}
