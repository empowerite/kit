// kit workspaces --current --json is the workspace kittui shows; --use keeps a group as the last used.
package kit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// workspaces points the fake at a directory of the test's own, each file in files written there, and gives the fake
// runner.
func workspaces(t *testing.T, files map[string]string) Runner {
	t.Helper()
	dir := t.TempDir()
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("FAKE_WORKSPACES", dir)
	return fake(t, t.TempDir())
}

func TestWorkspaceIsWhatKitWorkspacesCurrentSays(t *testing.T) {
	r := workspaces(t, map[string]string{"current.json": `{"kind":"group","name":"w","member":null,"checkout":null,
		"why":"last","members":[{"repository":"github.com/o/a","checkout":"/code/a","forge":"github"},
		{"repository":"github.com/o/b","checkout":null}],"groups":[],
		"favorites":[{"name":"w","kind":"group","members":2,"checkout":null},
		{"name":"github.com/o/a","kind":"repository","members":null,"checkout":"/code/a"}]}`})
	ws, err := r.Workspace(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if ws.Kind != "group" || ws.Name != "w" || ws.Member != "" || ws.Checkout != "" || ws.Why != "last" {
		t.Errorf("the workspace: %+v", ws)
	}
	if len(ws.Members) != 2 || ws.Members[0] != (WorkspaceMember{"github.com/o/a", "/code/a"}) ||
		ws.Members[1] != (WorkspaceMember{"github.com/o/b", ""}) {
		t.Errorf("the members: %+v", ws.Members)
	}
	if len(ws.Favorites) != 2 || ws.Favorites[0] != (WorkspaceFavorite{"w", "group", 2, ""}) ||
		ws.Favorites[1] != (WorkspaceFavorite{"github.com/o/a", "repository", 0, "/code/a"}) || len(ws.Groups) != 0 {
		t.Errorf("the favorites and groups: %+v, %+v", ws.Favorites, ws.Groups)
	}
}

func TestWorkspaceGivenNamesItToKit(t *testing.T) {
	r := workspaces(t, map[string]string{"github.com_o_a.json": `{"kind":"repository","name":"github.com/o/a",
		"member":"github.com/o/a","checkout":"/code/a","why":"given","members":[],
		"groups":[{"name":"w","members":2}],"favorites":[]}`})
	ws, err := r.Workspace(context.Background(), "github.com/o/a")
	if err != nil || ws.Kind != "repository" || ws.Checkout != "/code/a" || len(ws.Groups) != 1 ||
		ws.Groups[0] != (WorkspaceGroup{"w", 2}) {
		t.Errorf("the workspace given: %v, %+v", err, ws)
	}
}

func TestWorkspaceRefusedIsKitsLine(t *testing.T) {
	r := fake(t, t.TempDir())
	t.Setenv("FAKE_WORKSPACES", "")
	_, err := r.Workspace(context.Background(), "")
	var refusal *ExitError
	if !errors.As(err, &refusal) || refusal.Code != 2 || !strings.Contains(refusal.Error(), "workspaces") {
		t.Errorf("a kit that cannot say: %v", err)
	}
}

func TestUseWorkspaceTellsKit(t *testing.T) {
	r := workspaces(t, nil)
	record := filepath.Join(t.TempDir(), "record")
	t.Setenv("FAKE_KIT_RECORD", record)
	if err := r.UseWorkspace(context.Background(), "w"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(record); strings.TrimSpace(string(got)) != "use w" {
		t.Errorf("the record: %q", got)
	}
}
