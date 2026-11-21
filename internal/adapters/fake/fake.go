package fake

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/empowerite/kit/internal/ports"
)

// entry is one thing at a path of the fake files: a file, a link, or a directory.
type entry struct {
	data []byte
	perm fs.FileMode
	link string
	dir  bool
}

// Files is a machine's files, in memory. Fail makes an act fail: its key is the act's name and the path, as
// "write /a/b", and its value is the error the act gives.
type Files struct {
	at   map[string]*entry
	Fail map[string]error
}

// NewFiles gives files that hold nothing but the root.
func NewFiles() *Files {
	return &Files{at: map[string]*entry{"/": {dir: true, perm: 0o755}}, Fail: map[string]error{}}
}

func (f *Files) failing(act, name string) error { return f.Fail[act+" "+name] }

// parent makes the directories above name, as Write does.
func (f *Files) parent(name string) error {
	dir := path.Dir(name)
	if e, ok := f.at[dir]; ok && !e.dir {
		return &fs.PathError{Op: "mkdir", Path: dir, Err: fs.ErrExist}
	}
	return f.MakeDir(dir)
}

func (f *Files) Read(name string) ([]byte, error) {
	if err := f.failing("read", name); err != nil {
		return nil, err
	}
	at := name
	e, ok := f.at[at]
	for ok && e.link != "" {
		at = f.resolve(at, e.link)
		e, ok = f.at[at]
	}
	if !ok || e.dir {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
	}
	return slices.Clone(e.data), nil
}

// resolve gives the path a link at name to target leads to.
func (f *Files) resolve(name, target string) string {
	if path.IsAbs(target) {
		return target
	}
	return path.Join(path.Dir(name), target)
}

func (f *Files) Write(name string, data []byte, perm fs.FileMode) error {
	if err := f.failing("write", name); err != nil {
		return err
	}
	if err := f.parent(name); err != nil {
		return err
	}
	if e, ok := f.at[name]; ok {
		if e.dir || e.perm&0o200 == 0 {
			return &fs.PathError{Op: "write", Path: name, Err: fs.ErrPermission}
		}
		e.data = slices.Clone(data)
		return nil
	}
	f.at[name] = &entry{data: slices.Clone(data), perm: perm}
	return nil
}

func (f *Files) Link(target, name string) error {
	if err := f.failing("link", name); err != nil {
		return err
	}
	if _, ok := f.at[name]; ok {
		return &fs.PathError{Op: "symlink", Path: name, Err: fs.ErrExist}
	}
	if err := f.parent(name); err != nil {
		return err
	}
	f.at[name] = &entry{link: target, perm: 0o777}
	return nil
}

func (f *Files) ReadLink(name string) (string, error) {
	e, ok := f.at[name]
	if !ok || e.link == "" {
		return "", &fs.PathError{Op: "readlink", Path: name, Err: fs.ErrInvalid}
	}
	return e.link, nil
}

func (f *Files) Remove(name string) error {
	if err := f.failing("remove", name); err != nil {
		return err
	}
	if _, ok := f.at[name]; !ok {
		return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrNotExist}
	}
	if len(f.under(name)) > 0 {
		return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrExist}
	}
	delete(f.at, name)
	return nil
}

// under gives the paths below name.
func (f *Files) under(name string) []string {
	var found []string
	for p := range f.at {
		if p != name && strings.HasPrefix(p, strings.TrimSuffix(name, "/")+"/") {
			found = append(found, p)
		}
	}
	slices.Sort(found)
	return found
}

func (f *Files) Stat(name string) (fs.FileInfo, error) {
	if err := f.failing("stat", name); err != nil {
		return nil, err
	}
	e, ok := f.at[name]
	if !ok {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}
	mode := e.perm
	switch {
	case e.dir:
		mode |= fs.ModeDir
	case e.link != "":
		mode |= fs.ModeSymlink
	}
	return info{path.Base(name), int64(len(e.data)), mode}, nil
}

func (f *Files) MakeDir(name string) error {
	if err := f.failing("mkdir", name); err != nil {
		return err
	}
	for dir := name; dir != "/"; dir = path.Dir(dir) {
		if e, ok := f.at[dir]; ok && !e.dir {
			return &fs.PathError{Op: "mkdir", Path: dir, Err: fs.ErrExist}
		}
	}
	for dir := name; dir != "/"; dir = path.Dir(dir) {
		if _, ok := f.at[dir]; !ok {
			f.at[dir] = &entry{dir: true, perm: 0o755}
		}
	}
	return nil
}

func (f *Files) Rename(from, to string) error {
	if err := f.failing("rename", from); err != nil {
		return err
	}
	e, ok := f.at[from]
	if !ok {
		return &fs.PathError{Op: "rename", Path: from, Err: fs.ErrNotExist}
	}
	if err := f.parent(to); err != nil {
		return err
	}
	for _, p := range f.under(from) {
		f.at[to+strings.TrimPrefix(p, from)] = f.at[p]
		delete(f.at, p)
	}
	delete(f.at, from)
	f.at[to] = e
	return nil
}

func (f *Files) Chmod(name string, perm fs.FileMode) error {
	if err := f.failing("chmod", name); err != nil {
		return err
	}
	e, ok := f.at[name]
	if !ok {
		return &fs.PathError{Op: "chmod", Path: name, Err: fs.ErrNotExist}
	}
	e.perm = perm
	return nil
}

// Paths gives every path the files hold, but the root, sorted.
func (f *Files) Paths() []string { return f.under("/") }

// info is what Stat says of a fake file.
type info struct {
	name string
	size int64
	mode fs.FileMode
}

func (i info) Name() string       { return i.name }
func (i info) Size() int64        { return i.size }
func (i info) Mode() fs.FileMode  { return i.mode }
func (i info) ModTime() time.Time { return time.Time{} }
func (i info) IsDir() bool        { return i.mode.IsDir() }
func (i info) Sys() any           { return nil }

// Clock is a clock that says the time it is set to.
type Clock struct{ T time.Time }

func (c *Clock) Now() time.Time { return c.T }

// Git is a git that knows one clone: the root of its working tree, its common directory, and the paths it tracks.
type Git struct {
	RootDir, Common string
	Tracks          map[string]bool
	Err             error
}

func (g *Git) Root(context.Context, string) (string, string, error) {
	return g.RootDir, g.Common, g.Err
}

func (g *Git) Tracked(_ context.Context, _, name string) (bool, error) { return g.Tracks[name], g.Err }

// Presence gives the proof it is set to, or the error, and keeps each reason it was asked for.
type Presence struct {
	Proof ports.Proof
	Err   error
	Asked []string
}

func (p *Presence) Prove(_ context.Context, reason, _ string) (ports.Proof, error) {
	p.Asked = append(p.Asked, reason)
	return p.Proof, p.Err
}

// Processes runs nothing: it keeps each program it is asked to run, and answers as Answer says, or, with no Answer,
// prints what Prints holds for the program's arguments, joined by a space, and ends well. A program kit is replaced
// by is kept in Replaced, and Replace gives ReplaceErr.
type Processes struct {
	Runs       []ports.Cmd
	Answer     func(c ports.Cmd) (int, error)
	Prints     map[string]string
	Replaced   []ports.Cmd
	ReplaceErr error
}

func (p *Processes) Run(_ context.Context, c ports.Cmd) (int, error) {
	p.Runs = append(p.Runs, c)
	if p.Answer != nil {
		return p.Answer(c)
	}
	if c.Stdout != nil {
		io.WriteString(c.Stdout, p.Prints[strings.Join(c.Args, " ")])
	}
	return 0, nil
}

func (p *Processes) Replace(c ports.Cmd) error {
	p.Replaced = append(p.Replaced, c)
	return p.ReplaceErr
}

// Machine is a machine of the test's making.
type Machine struct {
	OS, Arch, CacheDir, StateDir, Kit string
	Env, Argv                         []string
	Err                               error // what Cache, State, and Executable give
}

func (m *Machine) Platform() string            { return m.OS + "/" + m.Arch }
func (m *Machine) Cache() (string, error)      { return m.CacheDir, m.Err }
func (m *Machine) State() (string, error)      { return m.StateDir, m.Err }
func (m *Machine) Executable() (string, error) { return m.Kit, m.Err }
func (m *Machine) Environ() []string           { return m.Env }
func (m *Machine) Args() []string              { return m.Argv }

// Releases is a forge's releases, in memory: each file by its repository, tag, and name.
type Releases struct {
	Assets map[string]ports.Asset // by "repo tag name"
	Tags   map[string]string      // a repository's newest tag
}

// Put puts a file in a release, at an address, with the digest the forge keeps of it.
func (r *Releases) Put(repo, tag, name, url, digest string) {
	if r.Assets == nil {
		r.Assets = map[string]ports.Asset{}
	}
	r.Assets[repo+" "+tag+" "+name] = ports.Asset{URL: url, Digest: digest}
}

func (r *Releases) Asset(_ context.Context, repo, tag, name string) (ports.Asset, error) {
	if a, ok := r.Assets[repo+" "+tag+" "+name]; ok {
		return a, nil
	}
	return ports.Asset{}, fmt.Errorf("github: %s %s has no file %s", repo, tag, name)
}

func (r *Releases) Newest(_ context.Context, repo string) (string, error) {
	if tag, ok := r.Tags[repo]; ok {
		return tag, nil
	}
	return "", fmt.Errorf("github: %s has no release", repo)
}

// Downloads is what each address holds, in memory.
type Downloads map[string][]byte

func (d Downloads) Get(_ context.Context, url string) ([]byte, error) {
	if body, ok := d[url]; ok {
		return body, nil
	}
	return nil, fmt.Errorf("download: %s: 404", url)
}
