// Welcome over a workspace: kit judges every folder of it itself, the machine once and each folder of every member on
// its own, so the screen reads the checks once, over the workspace, and names a folder's check by its folder.
package welcome

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// The glyphs of a folder: the desk, and a worktree, as Flow draws them.
const (
	deskGlyph     = "💻"
	worktreeGlyph = "🌳"
)

// checksOver gives the read of the checks over the workspace: kit welcome --checks --json --progress over a group, as
// kit --workspace=<group> names it, every member's desk and worktrees; over anything else, where the runner reads.
// The read tells told each line of kit's progress as it comes.
func checksOver(r kit.Runner, ws kit.Workspace) func(context.Context, func(kit.Progress)) ([]kit.Check, error) {
	if ws.Kind == "group" && ws.Name != "" {
		r.Group, r.NoGroup = ws.Name, false
	}
	return func(ctx context.Context, told func(kit.Progress)) ([]kit.Check, error) {
		return r.WelcomeChecks(ctx, told)
	}
}

// uncloned gives the owner/repo of each of a group's members with no checkout, in order.
func uncloned(ws kit.Workspace) []string {
	var out []string
	for _, m := range ws.Members {
		if m.Checkout == "" {
			out = append(out, short(m.Repository, 2))
		}
	}
	return out
}

// short gives the last so many segments of a host/path.
func short(member string, n int) string {
	segments := strings.Split(member, "/")
	return strings.Join(segments[max(0, len(segments)-n):], "/")
}

// folderName gives a folder as a row names it: its glyph, the desk's or a worktree's, and its folder's name.
func folderName(c kit.Check) string {
	glyph := worktreeGlyph
	if c.Desk {
		glyph = deskGlyph
	}
	return glyph + " " + filepath.Base(c.Folder)
}

// folderLine gives the line that names a folder above its checks: its repository, its name, and its branch.
func folderLine(c kit.Check) string {
	text := c.Repository + " · " + folderName(c)
	if c.Branch != "" {
		text += " on " + c.Branch
	}
	return text
}

// folderOf gives what a folder's check says after its line where its folder is not drawn above it: its repository's
// name and its folder's, dim; nothing for a machine check.
func folderOf(c kit.Check) string {
	if c.Folder == "" {
		return ""
	}
	return style.Off(" · " + short(c.Repository, 1) + " " + folderName(c))
}
