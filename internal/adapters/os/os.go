package os

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	goos "os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
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

// Replace runs a program in kit's place. It returns only where the program cannot be run.
func (Processes) Replace(c ports.Cmd) error {
	env := c.Env
	if env == nil {
		env = goos.Environ()
	}
	path := c.Path
	if !strings.Contains(path, "/") {
		found, err := lookPath(path, env)
		if err != nil {
			return err
		}
		path = found
	}
	return syscall.Exec(path, append([]string{c.Path}, c.Args...), env)
}

// lookPath finds a program by its name on the PATH an environment holds.
func lookPath(name string, env []string) (string, error) {
	for _, kv := range env {
		if dirs, ok := strings.CutPrefix(kv, "PATH="); ok {
			for _, dir := range filepath.SplitList(dirs) {
				path := filepath.Join(dir, name)
				if info, err := goos.Stat(path); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
					return path, nil
				}
			}
		}
	}
	return "", fmt.Errorf("there is no %s on the PATH", name)
}

// Machine is the machine kit runs on.
type Machine struct{}

func (Machine) Platform() string { return runtime.GOOS + "/" + runtime.GOARCH }

// Cache is the person's cache, as the system names it, with kit's folder in it.
func (Machine) Cache() (string, error) {
	dir, err := goos.UserCacheDir()
	return dir + "/kit", err
}

// State is where kit keeps what is the person's, as XDG_STATE_HOME names it, or ~/.local/state.
func (Machine) State() (string, error) {
	if dir := goos.Getenv("XDG_STATE_HOME"); dir != "" {
		return dir + "/kit", nil
	}
	home := goos.Getenv("HOME")
	if home == "" {
		return "", errors.New("kit has no home: HOME is not set")
	}
	return home + "/.local/state/kit", nil
}

// Executable gives the path of the kit that runs, with no link in it.
func (Machine) Executable() (string, error) {
	path, err := goos.Executable()
	if err == nil {
		path, err = filepath.EvalSymlinks(path)
	}
	return path, err
}

func (Machine) Environ() []string { return goos.Environ() }
func (Machine) Args() []string    { return goos.Args }
