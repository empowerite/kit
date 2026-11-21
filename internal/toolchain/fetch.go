package toolchain

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/empowerite/kit/internal/plan"
	"github.com/empowerite/kit/internal/ports"
)

// direnv is what .kit/env holds, and kit env --shell=direnv prints: direnv's own words, with kit.yml and kit.lock
// watched, so that direnv loads the environment again when a pin changes.
const direnv = "PATH_add .kit/bin\nwatch_file kit.yml kit.lock\n"

// hexOf gives a digest's hex, without its "sha256:".
func hexOf(digest string) string { return strings.TrimPrefix(digest, "sha256:") }

// whole says whether a folder of the cache is whole: it is named for its digest only once it is.
func whole(files ports.Files, dir string) (bool, error) {
	_, err := files.Stat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// fill makes a folder of the cache: it writes the file into the folder's .part, and names the folder for its digest
// only once the file is whole. A .part left by a fetch that stopped is made again.
func fill(files ports.Files, dir, name string, data []byte) error {
	ready, err := whole(files, dir)
	if err != nil || ready {
		return err
	}
	part := dir + ".part"
	for _, p := range []string{part + "/" + name, part} {
		if err := files.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	if err := files.Write(part+"/"+name, data, 0o555); err != nil {
		return err
	}
	return files.Rename(part, dir)
}

// facts gives what a step of the cache depends on: whether its folder is whole.
func facts(files ports.Files, dir string) (string, error) {
	ready, err := whole(files, dir)
	return fmt.Sprintf("%s whole %v", dir, ready), err
}

// keep keeps a copy of kit in the cache, by its own digest, for the links of .kit/bin to lead to: the kit that
// placed them may be gone, as a build of bin/gate's is.
type keep struct {
	files ports.Files
	dir   string
	kit   []byte
}

func (k keep) Step() plan.Step {
	return plan.Step{System: "cache", Does: "keep this kit in the cache, at " + k.dir, Undoable: true, Hydrates: true}
}

func (k keep) Facts(context.Context) (string, error) { return facts(k.files, k.dir) }

func (k keep) Apply(context.Context) error { return fill(k.files, k.dir, "kit", k.kit) }

// fetch fetches a tool's program for this platform into the cache, holding the file and the program to the lock.
type fetch struct {
	d    Desk
	name string
	f    LockedFile
	dir  string
}

func (o fetch) Step() plan.Step {
	return plan.Step{System: "cache", Does: "fetch " + o.name + " into the cache, from " + o.f.URL, Undoable: true,
		Hydrates: true}
}

func (o fetch) Facts(context.Context) (string, error) { return facts(o.d.Files, o.dir) }

func (o fetch) Apply(ctx context.Context) error {
	if ready, err := whole(o.d.Files, o.dir); err != nil || ready {
		return err
	}
	body, err := o.d.Downloads.Get(ctx, o.f.URL)
	if err != nil {
		return err
	}
	if got := digestOf(body); got != o.f.Digest {
		return fmt.Errorf("the file downloaded from %s has the digest %s, and kit.lock names %s", o.f.URL, got, o.f.Digest)
	}
	program, err := extract(path.Base(o.f.URL), body, o.f.Program)
	if err != nil {
		return err
	}
	if got := digestOf(program); got != o.f.ProgramDigest {
		return fmt.Errorf("%s in %s is not the program kit.lock names: its digest is %s, and kit.lock names %s",
			o.f.Program, o.f.URL, got, o.f.ProgramDigest)
	}
	return fill(o.d.Files, o.dir, path.Base(o.f.Program), program)
}

// file gives the locked file of a tool for the desk's platform.
func file(d Desk, r Repo, name string) (LockedFile, error) {
	l, ok := r.Lock.Tools[name]
	if !ok {
		return LockedFile{}, fmt.Errorf("kit.lock at %s holds no %s", r.Root, name)
	}
	platform := Platform(d.Machine.Platform())
	f, ok := l.Files[platform]
	if !ok {
		return LockedFile{}, fmt.Errorf("kit.lock holds no file of %s for %s", name, platform)
	}
	return f, nil
}

// Fetch plans to fetch every tool the lock holds for this platform into the cache, to place a link in .kit/bin for
// each, and .kit/env for direnv. It changes nothing that git sees, so it is hydrated, with no approval.
func Fetch(ctx context.Context, d Desk, r Repo) (plan.Plan, error) {
	cache, err := d.Machine.Cache()
	if err != nil {
		return plan.Plan{}, err
	}
	kit, err := d.Machine.Executable()
	var body []byte
	if err == nil {
		body, err = d.Files.Read(kit)
	}
	if err != nil {
		return plan.Plan{}, err
	}
	kept := cache + "/kit/" + hexOf(digestOf(body))
	ops := []plan.Operation{keep{d.Files, kept, body}}
	for _, name := range slices.Sorted(maps.Keys(r.Lock.Tools)) {
		f, err := file(d, r, name)
		if err != nil {
			return plan.Plan{}, err
		}
		ops = append(ops, fetch{d, name, f, cache + "/tools/" + hexOf(f.Digest)},
			plan.Place(d.Files, d.Git, r.Root, "bin/"+name, nil, kept+"/kit"))
	}
	ops = append(ops, plan.Place(d.Files, d.Git, r.Root, "env", []byte(direnv), ""))
	return plan.Plan{Act: "tools fetch", Operations: ops}, nil
}

// checked gives the path of a tool's program in the cache, fetched where it is not yet, once its digest is the one
// the lock names.
func checked(ctx context.Context, d Desk, r Repo, name string) (string, error) {
	f, err := file(d, r, name)
	if err != nil {
		return "", err
	}
	cache, err := d.Machine.Cache()
	if err != nil {
		return "", err
	}
	dir := cache + "/tools/" + hexOf(f.Digest)
	program := dir + "/" + path.Base(f.Program)
	body, err := d.Files.Read(program)
	if errors.Is(err, fs.ErrNotExist) {
		p, _ := Fetch(ctx, d, r)
		if result := plan.Hydrate(ctx, p); result.Err != nil {
			return "", result.Err
		}
		body, err = d.Files.Read(program)
	}
	if err != nil {
		return "", err
	}
	if got := digestOf(body); got != f.ProgramDigest {
		return "", fmt.Errorf("%s is not the program kit.lock names: its digest is %s, and kit.lock names %s. "+
			"Something changed the cache: remove %s, and kit fetches it again", program, got, f.ProgramDigest, dir)
	}
	return program, nil
}

// Path gives the path of a tool's program in the cache, fetched and checked.
func Path(ctx context.Context, d Desk, r Repo, name string) (string, error) {
	return checked(ctx, d, r, name)
}

// RunTool is kit started by a tool's name, through a link in .kit/bin: it holds the program to the lock, and runs it
// in kit's place with the same arguments. The repository is the one whose .kit/bin the link is in, or, where kit was
// started by the name alone, the one kit runs in.
func RunTool(ctx context.Context, d Desk, argv0 string, args []string) error {
	var r Repo
	var err error
	if strings.Contains(argv0, "/") {
		bin := argv0
		if !path.IsAbs(bin) {
			bin = path.Join(d.Dir, bin)
		}
		r, err = openAt(d, path.Dir(path.Dir(path.Dir(bin))))
	} else {
		r, err = Open(ctx, d)
	}
	if err != nil {
		return err
	}
	program, err := checked(ctx, d, r, path.Base(argv0))
	if err != nil {
		return err
	}
	return d.Processes.Replace(ports.Cmd{Path: program, Args: args, Env: d.Machine.Environ()})
}

// Exec runs a command in kit's place, with the pinned tools fetched, and first on the PATH.
func Exec(ctx context.Context, d Desk, r Repo, argv []string) error {
	p, err := Fetch(ctx, d, r)
	if err != nil {
		return err
	}
	if result := plan.Hydrate(ctx, p); result.Err != nil {
		return result.Err
	}
	return d.Processes.Replace(ports.Cmd{Path: argv[0], Args: argv[1:], Env: withPath(d.Machine.Environ(), r.Root+"/.kit/bin")})
}

// withPath gives an environment with a folder first on its PATH.
func withPath(env []string, dir string) []string {
	env = slices.Clone(env)
	for i, kv := range env {
		if rest, ok := strings.CutPrefix(kv, "PATH="); ok {
			env[i] = "PATH=" + dir + ":" + rest
			return env
		}
	}
	return append(env, "PATH="+dir)
}

// Env gives what a shell reads to have the pinned tools first on its PATH: bash's and zsh's export, or direnv's own
// words.
func Env(shell, root string) (string, error) {
	switch shell {
	case "bash", "zsh":
		return "export PATH=" + quote(root+"/.kit/bin") + ":\"$PATH\"\n", nil
	case "direnv":
		return direnv, nil
	}
	return "", fmt.Errorf("kit env knows bash, zsh, and direnv, and %s is none of them", shell)
}

// quote quotes a word for a shell.
func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// Listed is one tool a repository's work runs with: where it is pinned, and how it stands on this machine.
type Listed struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Where   string `json:"where"` // "kit.yml", or "go.mod"
	State   string `json:"state"`
}

var goToolchain = regexp.MustCompile(`(?m)^toolchain go(\S+)`)

// Tools lists the tools a repository's work runs with: those kit.yml pins, and Go, which go.mod's toolchain line
// pins, sorted by name.
func Tools(d Desk, r Repo) ([]Listed, error) {
	var listed []Listed
	gomod, err := read(d.Files, r.Root+"/go.mod")
	if err != nil {
		return nil, err
	}
	if m := goToolchain.FindSubmatch(gomod); m != nil {
		listed = append(listed, Listed{"go", string(m[1]), "go.mod", "Go fetches it, and checks it against its own checksum database"})
	}
	cache, err := d.Machine.Cache()
	if err != nil {
		return nil, err
	}
	for _, pin := range r.Pins {
		state := "not fetched here"
		f, err := file(d, r, pin.Name)
		switch body, readErr := d.Files.Read(cache + "/tools/" + hexOf(f.Digest) + "/" + path.Base(f.Program)); {
		case err != nil:
			state = strings.Replace(err.Error(), "of "+pin.Name+" ", "of it ", 1)
		case readErr != nil:
		case digestOf(body) == f.ProgramDigest:
			state = "fetched here, and checked"
		default:
			state = "fetched here, and not what kit.lock names"
		}
		listed = append(listed, Listed{pin.Name, pin.Version, "kit.yml", state})
	}
	slices.SortFunc(listed, func(a, b Listed) int { return strings.Compare(a.Name, b.Name) })
	return listed, nil
}
