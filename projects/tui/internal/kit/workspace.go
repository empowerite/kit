// The workspace kit looks at, as kit workspaces --current --json says it: one repository, or a named group of them.
package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// Workspace is what kit workspaces --current --json prints: its kind, repository, group, or none, empty while kit has
// not said; its name, a group's or a repository's; a repository's member, host/path, empty with no remote, and its
// checkout; why kit starts on it; a group's members; the groups a repository belongs to; and the favorites.
type Workspace struct {
	Kind      string              `json:"kind"`
	Name      string              `json:"name"`
	Member    string              `json:"member"`
	Checkout  string              `json:"checkout"`
	Why       string              `json:"why"`
	Members   []WorkspaceMember   `json:"members"`
	Groups    []WorkspaceGroup    `json:"groups"`
	Favorites []WorkspaceFavorite `json:"favorites"`
}

// WorkspaceMember is one of a group's members: its host/path, and its checkout, empty when it is not cloned.
type WorkspaceMember struct {
	Repository string `json:"repository"`
	Checkout   string `json:"checkout"`
}

// WorkspaceGroup is a group a repository belongs to: its name and how many members it has.
type WorkspaceGroup struct {
	Name    string `json:"name"`
	Members int    `json:"members"`
}

// WorkspaceFavorite is a favorite: its name as kit names it, its kind, a group's count of members, and a repository's
// checkout, empty when it is not cloned.
type WorkspaceFavorite struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Members  int    `json:"members"`
	Checkout string `json:"checkout"`
}

// Workspace gives the workspace kit starts on where the command was typed, or the one given, a group's name or a
// repository's host/path; given "" names none, as kit starts.
func (r Runner) Workspace(ctx context.Context, given string) (Workspace, error) {
	args := []string{"workspaces", "--current", "--json"}
	if given != "" {
		args = append(args, "--workspace", given)
	}
	out, err := r.within(ctx, callLimit, append(os.Environ(), "NO_COLOR=1"), args...)
	if err != nil {
		return Workspace{}, err
	}
	var ws Workspace
	if err := json.Unmarshal(out, &ws); err != nil {
		return Workspace{}, fmt.Errorf("kit workspaces --current --json: %v", err)
	}
	return ws, nil
}

// UseWorkspace keeps a group as the last used, which kit starts on outside a repository.
func (r Runner) UseWorkspace(ctx context.Context, name string) error {
	_, err := r.within(ctx, callLimit, append(os.Environ(), "NO_COLOR=1"), "workspaces", "--use="+name)
	return err
}
