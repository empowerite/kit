package ports

import (
	"context"
	"io"
	"io/fs"
	"time"
)

// Files is the machine's files. Every path is absolute.
type Files interface {
	// Read gives a file's bytes.
	Read(path string) ([]byte, error)
	// Write writes a file, making the directories above it, with perm where the file is new.
	Write(path string, data []byte, perm fs.FileMode) error
	// Link makes path a symbolic link to target.
	Link(target, path string) error
	// ReadLink gives where a symbolic link leads.
	ReadLink(path string) (string, error)
	// Remove removes a file, a link, or an empty directory.
	Remove(path string) error
	// Stat says what is at path, and does not follow a link there.
	Stat(path string) (fs.FileInfo, error)
	// MakeDir makes a directory, and those above it.
	MakeDir(path string) error
	// Rename moves a file or a directory to a new path.
	Rename(from, to string) error
	// Chmod sets a file's permissions.
	Chmod(path string, perm fs.FileMode) error
}

// Clock is the time.
type Clock interface {
	Now() time.Time
}

// Git is what kit asks of git.
type Git interface {
	// Root gives the root of the working tree that holds dir, and the clone's common directory, which every worktree
	// of the clone shares.
	Root(ctx context.Context, dir string) (root, common string, err error)
	// Tracked says whether git tracks the file at path, a path from the root.
	Tracked(ctx context.Context, root, path string) (bool, error)
}

// Proof is the proof that a person is there: how it was given, and whether it is weaker than the machine's own, as a
// typed answer at a terminal is.
type Proof struct {
	Kind   string // "touch-id", "password", or "typed"
	Weaker bool
}

// Presence asks the machine to prove that the person is there, for the reason given, which names what is approved.
// hash is the plan's hash, which a typed answer repeats in part.
type Presence interface {
	Prove(ctx context.Context, reason, hash string) (Proof, error)
}

// Cmd is a program to run: its path or its name, its arguments, its environment (nil for kit's own), where it runs,
// and its streams.
type Cmd struct {
	Path   string
	Args   []string
	Env    []string
	Dir    string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Processes runs programs.
type Processes interface {
	// Run runs a program to its end, and gives its exit status. An error is a program that could not be run.
	Run(ctx context.Context, c Cmd) (int, error)
}

// Asset is one file of a release: where it is downloaded from, and the digest the forge keeps of it, as
// "sha256:<hex>", or "" where it keeps none.
type Asset struct {
	URL, Digest string
}

// Releases reads a forge's releases of a repository.
type Releases interface {
	// Asset gives the file of that name of the release with that tag.
	Asset(ctx context.Context, repo, tag, name string) (Asset, error)
	// Newest gives the tag of the repository's newest release.
	Newest(ctx context.Context, repo string) (string, error)
}

// Downloads fetches what an address holds.
type Downloads interface {
	Get(ctx context.Context, url string) ([]byte, error)
}
