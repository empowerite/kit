// Where origin points, read from git, for the title and the URLs a row opens to.
package flow

import (
	"fmt"
	"os/exec"
	"strings"
)

// Origin is where origin points: its host, and its owner/repo path. Both are empty when git cannot say.
type Origin struct {
	Host, Repo string
}

// ReadOrigin asks git for origin's URL in dir, which touches no network, and reads the host and the path from it.
func ReadOrigin(dir string) Origin {
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Output()
	if err != nil {
		return Origin{}
	}
	return parseRemote(strings.TrimSpace(string(out)))
}

// parseRemote reads a remote's host and owner/repo from its URL: https://host/owner/repo.git, ssh://git@host/owner/
// repo, or git@host:owner/repo. A URL with fewer than two path segments gives nothing.
func parseRemote(url string) Origin {
	host, path := "", ""
	if _, rest, ok := strings.Cut(url, "://"); ok {
		host, path, _ = strings.Cut(rest, "/")
	} else if user, rest, ok := strings.Cut(url, ":"); ok {
		host, path = user, rest
	} else {
		return Origin{}
	}
	if _, after, ok := strings.Cut(host, "@"); ok {
		host = after
	}
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	segments := strings.Split(path, "/")
	if len(segments) < 2 || host == "" {
		return Origin{}
	}
	return Origin{Host: host, Repo: strings.Join(segments[len(segments)-2:], "/")}
}

// issue gives an issue's URL: the one kit printed when it did, else origin's, GitLab's shape on a GitLab host and
// GitHub's otherwise, or nothing when origin is unknown.
func (o Origin) issue(number int, given string) string {
	if given != "" {
		return given
	}
	if o.Repo == "" {
		return ""
	}
	return fmt.Sprintf("https://%s/%s/%sissues/%d", o.Host, o.Repo, o.infix(), number)
}

// pull gives a pull request's URL by its number, as issue gives an issue's: a merge request's on GitLab.
func (o Origin) pull(number int, given string) string {
	if given != "" {
		return given
	}
	if o.Repo == "" {
		return ""
	}
	if o.gitlab() {
		return fmt.Sprintf("https://%s/%s/-/merge_requests/%d", o.Host, o.Repo, number)
	}
	return fmt.Sprintf("https://%s/%s/pull/%d", o.Host, o.Repo, number)
}

// gitlab says whether the host looks like GitLab's, whose paths carry a -/ before the kind.
func (o Origin) gitlab() bool { return strings.Contains(strings.ToLower(o.Host), "gitlab") }

// infix gives the path segment before issues: -/ on GitLab, nothing on GitHub.
func (o Origin) infix() string {
	if o.gitlab() {
		return "-/"
	}
	return ""
}
