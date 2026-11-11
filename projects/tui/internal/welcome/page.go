// A page's markdown is drawn as text as internal/markdown draws every screen's pages; these are its names here.
package welcome

import "github.com/empowerite/kit/projects/tui/internal/markdown"

var (
	heading = markdown.Heading
	item    = markdown.Item
)

// render draws a page's markdown in lines no wider than the width, in cells, as markdown.Render does.
func render(md string, width int) []string { return markdown.Render(md, width) }

// unlinked gives a text with each link in it as its text, as markdown.Unlinked does.
func unlinked(text string) string { return markdown.Unlinked(text) }

// wrapWords lays words out in lines no wider than the width, as markdown.WrapWords does.
func wrapWords(words []string, width int, first, indent string) []string {
	return markdown.WrapWords(words, width, first, indent)
}
