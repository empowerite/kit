package os

import (
	"context"
	"errors"
	"io/fs"
	goos "os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/empowerite/kit/internal/ports"
)

// Files is the machine's own files.
type Files struct{}

func (Files) Read(path string) ([]byte, error) { return goos.ReadFile(path) }

func (Files) Write(path string, data []byte, perm fs.FileMode) error {
	if err := goos.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return goos.WriteFile(path, data, perm)
}

func (Files) Link(target, path string) error {
	if err := goos.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return goos.Symlink(target, path)
}

func (Files) ReadLink(path string) (string, error)      { return goos.Readlink(path) }
func (Files) Remove(path string) error                  { return goos.Remove(path) }
func (Files) Stat(path string) (fs.FileInfo, error)     { return goos.Lstat(path) }
func (Files) MakeDir(path string) error                 { return goos.MkdirAll(path, 0o755) }
func (Files) Rename(from, to string) error              { return goos.Rename(from, to) }
func (Files) Chmod(path string, perm fs.FileMode) error { return goos.Chmod(path, perm) }

// Clock is the machine's clock.
type Clock struct{}

func (Clock) Now() time.Time { return time.Now() }

// Processes runs the machine's programs.
type Processes struct{}

// Run runs a program to its end. A program that cannot be found is an error, and status -1, before anything starts.
func (Processes) Run(ctx context.Context, c ports.Cmd) (int, error) {
	cmd := exec.CommandContext(ctx, c.Path, c.Args...)
	cmd.Env, cmd.Dir = c.Env, c.Dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = c.Stdin, c.Stdout, c.Stderr
	return exit(cmd.Run())
}

// exit gives the status of a program that ran, whatever it was, or -1 and the error of one that did not.
func exit(err error) (int, error) {
	var ended interface{ ExitCode() int }
	switch {
	case err == nil:
		return 0, nil
	case errors.As(err, &ended):
		return ended.ExitCode(), nil
	}
	return -1, err
}
