// Where the screen's words come from besides the checks: owner/repo from origin's URL, and the pages from the
// repository or from the kit package.
package welcome

import (
	"cmp"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/empowerite/kit/tui/internal/kit"
)

// pagesDir is where the pages live, under the repository's root and under the kit package's files.
const pagesDir = ".kit/welcome"

// packageDir is the kit package's repository files, under the release's directory.
const packageDir = "packages/kit/files/repo"

// readPages gives the steps the pages make, as kit welcome --text finds them: every *.md under the repository's
// .kit/welcome, as the active packages placed them, else under the kit package's, where nothing is placed yet. They
// are in the order of the number each name starts with, as kit welcome --text puts them, so 10-ours.md comes after
// 9-theirs.md; a name with no number comes last, and a tie goes by name. A page that cannot be read gives where it was looked for. When neither has a
// page, the steps are the known ones, each saying where it was looked for.
func readPages(r kit.Runner) []tab {
	repo := ""
	if out, err := exec.Command("git", "-C", r.Dir, "rev-parse", "--show-toplevel").Output(); err == nil {
		repo = filepath.Join(strings.TrimSpace(string(out)), pagesDir)
	}
	if tabs := readDir(repo); len(tabs) > 0 {
		return tabs
	}
	packaged := ""
	if root, err := r.Root(); err == nil && root != "" {
		packaged = filepath.Join(root, packageDir, pagesDir)
	}
	if tabs := readDir(packaged); len(tabs) > 0 {
		return tabs
	}
	where := pagesDir
	if packaged != "" {
		where = packaged
	}
	tabs := slices.Clone(known)
	for i := range tabs {
		tabs[i].missing = filepath.Join(where, tabs[i].file)
	}
	return tabs
}

// readDir gives the step of each page in a directory, in order: each file whose name ends in .md and does not start
// with a dot, as the shell's *.md matches.
func readDir(dir string) []tab {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if name := e.Name(); strings.HasSuffix(name, ".md") && !strings.HasPrefix(name, ".") {
			files = append(files, name)
		}
	}
	slices.SortFunc(files, func(a, b string) int {
		return cmp.Or(cmp.Compare(order(a), order(b)), strings.Compare(a, b))
	})
	tabs := make([]tab, len(files))
	for i, f := range files {
		tabs[i] = tabOf(f)
		tabs[i].page, tabs[i].missing = readPage(filepath.Join(dir, f))
	}
	return tabs
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

// order gives the number a page's file name starts with, or a number past every page's when it starts with none.
func order(name string) int {
	digits := name[:len(name)-len(strings.TrimLeft(name, "0123456789"))]
	if n, err := strconv.Atoi(digits); err == nil {
		return n
	}
	return math.MaxInt
}
