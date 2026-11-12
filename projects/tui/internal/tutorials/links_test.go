// Every link in the pages kit ships, the tutorials' and the welcome's, leads somewhere: a screen kittui has, one of the
// flow's stages, a tutorial the catalogue lists, a tab of the welcome, or a term its Terms defines.
package tutorials

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/empowerite/kit/projects/tui/internal/shell"
)

var mention = regexp.MustCompile("`kit tutorials ([a-z-]+)`")

var linkTarget = regexp.MustCompile(`\]\(kittui:([^)\s]+)\)`)

func TestEveryLinkInTheShippedPagesLeadsSomewhere(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "kit")
	screens := []shell.Screen{shell.Welcome, shell.Help, shell.Tutorials, shell.Flow, shell.Issues, shell.Changes,
		shell.Repositories, shell.Delivery, shell.People, shell.Trust, shell.Platform, shell.Packages}
	stages := []string{"issues", "changes", "builds", "incoming", "releases"}
	catalogue, err := os.ReadFile(filepath.Join(root, "modules", "tutorials.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, m := range regexp.MustCompile(`(?m)^  - id: (\S+)`).FindAllStringSubmatch(string(catalogue), -1) {
		ids = append(ids, m[1])
	}
	welcome := filepath.Join(root, "packages", "kit", "files", "repo", ".kit", "welcome")
	pages, err := filepath.Glob(filepath.Join(welcome, "*.md"))
	if err != nil || len(pages) == 0 {
		t.Fatalf("no welcome pages: %v", err)
	}
	var tabs, terms []string
	for _, p := range pages {
		_, name, _ := strings.Cut(strings.TrimSuffix(filepath.Base(p), ".md"), "-")
		tabs = append(tabs, name)
	}
	termsPage, err := os.ReadFile(filepath.Join(welcome, "5-terms.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range regexp.MustCompile(`(?m)^### (.+)$`).FindAllStringSubmatch(string(termsPage), -1) {
		for _, n := range strings.Split(m[1], ",") {
			terms = append(terms, strings.ToLower(strings.TrimSpace(n)))
		}
	}
	tutorials, err := filepath.Glob(filepath.Join(root, "modules", "tutorials", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	links := 0
	for _, p := range append(tutorials, pages...) {
		page, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		// a mention of a tutorial by its command stands beside a link to it
		for _, l := range strings.Split(string(page), "\n") {
			for _, m := range mention.FindAllStringSubmatch(l, -1) {
				if !strings.Contains(l, "](kittui:tutorials/"+m[1]+")") {
					t.Errorf("%s: %q names a tutorial with no link to it", filepath.Base(p), l)
				}
			}
		}
		for _, m := range linkTarget.FindAllStringSubmatch(string(page), -1) {
			links++
			screen, rest, _ := strings.Cut(m[1], "/")
			second, third, _ := strings.Cut(rest, "/")
			term, _ := url.PathUnescape(third)
			ok := slices.Contains(screens, shell.Screen(screen))
			switch {
			case rest == "":
			case screen == "tutorials":
				ok = ok && slices.Contains(ids, rest)
			case screen == "flow":
				ok = ok && slices.Contains(stages, rest)
			case screen == "welcome":
				ok = ok && slices.Contains(tabs, second) && (third == "" || second == "terms" &&
					slices.Contains(terms, strings.ToLower(term)))
			default:
				ok = false
			}
			if !ok {
				t.Errorf("%s: kittui:%s leads nowhere", filepath.Base(p), m[1])
			}
		}
	}
	if links < 50 {
		t.Errorf("only %d links in the shipped pages", links)
	}
}
