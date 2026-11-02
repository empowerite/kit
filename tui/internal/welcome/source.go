// Where the screen's words come from besides the checks: owner/repo from origin's URL, and the pages from the
// repository or from the kit package.
package welcome

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/empowerite/kit/tui/internal/kit"
)

// pagesDir is where the pages live, under the repository's root and under the kit package's files.
const pagesDir = ".kit/welcome"

// packageDir is the kit package's repository files, under the release's directory.
const packageDir = "packages/kit/files/repo"

// readPages reads each step's page: from the repository's .kit/welcome when it has the page, else from the kit
// package under kit root. A page in neither gives the path it was last looked for at.
func readPages(r kit.Runner) (pages, missing [steps]string) {
	repo, packaged, asked := "", "", false
	if out, err := exec.Command("git", "-C", r.Dir, "rev-parse", "--show-toplevel").Output(); err == nil {
		repo = filepath.Join(strings.TrimSpace(string(out)), pagesDir)
	}
	for i, t := range tabs {
		missing[i] = filepath.Join(pagesDir, t.file)
		if repo != "" {
			if pages[i], missing[i] = readPage(filepath.Join(repo, t.file)); missing[i] == "" {
				continue
			}
		}
		if !asked {
			asked = true
			if root, err := r.Root(); err == nil && root != "" {
				packaged = filepath.Join(root, packageDir, pagesDir)
			}
		}
		if packaged != "" {
			pages[i], missing[i] = readPage(filepath.Join(packaged, t.file))
		}
	}
	return pages, missing
}

// readPage gives a page's markdown, or the path when it cannot be read.
func readPage(path string) (page, missing string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", path
	}
	return string(data), ""
}

// readRepo gives origin's owner/repo in dir, read from its URL without touching the network, or nothing when git
// cannot say.
func readRepo(dir string) string {
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return repoOf(strings.TrimSpace(string(out)))
}

// repoOf reads owner/repo from a remote's URL: https://host/owner/repo.git, ssh://git@host/owner/repo, or
// git@host:owner/repo. A URL with fewer than two path segments gives nothing.
func repoOf(url string) string {
	path := ""
	if _, rest, ok := strings.Cut(url, "://"); ok {
		_, path, _ = strings.Cut(rest, "/")
	} else if _, rest, ok := strings.Cut(url, ":"); ok {
		path = rest
	}
	segments := strings.Split(strings.TrimSuffix(strings.Trim(path, "/"), ".git"), "/")
	if len(segments) < 2 || segments[len(segments)-2] == "" {
		return ""
	}
	return strings.Join(segments[len(segments)-2:], "/")
}
