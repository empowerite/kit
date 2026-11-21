package git

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/empowerite/kit/internal/ports"
)

// Git is the machine's git, which it runs through processes.
type Git struct{ processes ports.Processes }

// New gives the git that processes run.
func New(processes ports.Processes) Git { return Git{processes} }

// git runs git with args, and gives its status and what it printed.
func (g Git) git(ctx context.Context, args ...string) (int, string, error) {
	var out, said bytes.Buffer
	code, err := g.processes.Run(ctx, ports.Cmd{Path: "git", Args: args, Stdout: &out, Stderr: &said})
	if err != nil {
		return code, "", fmt.Errorf("git: %w", err)
	}
	if code > 1 {
		return code, "", fmt.Errorf("git: %s", strings.TrimSpace(said.String()))
	}
	return code, out.String(), nil
}

func (g Git) Root(ctx context.Context, dir string) (string, string, error) {
	code, out, err := g.git(ctx, "-C", dir, "rev-parse", "--path-format=absolute", "--show-toplevel", "--git-common-dir")
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	switch {
	case err != nil:
		return "", "", err
	case code != 0 || len(lines) != 2:
		return "", "", fmt.Errorf("git: %s is in no working tree of a clone", dir)
	}
	return lines[0], lines[1], nil
}

func (g Git) Tracked(ctx context.Context, root, path string) (bool, error) {
	code, _, err := g.git(ctx, "-C", root, "ls-files", "--error-unmatch", "--", path)
	return code == 0 && err == nil, err
}
