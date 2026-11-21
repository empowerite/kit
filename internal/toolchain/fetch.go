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

// direnv is what kit env --shell=direnv prints, for an .envrc at the root: direnv's own words, with kit.yml and
// kit.lock watched, so that direnv loads the environment again when a pin changes.
const direnv = "PATH_add .kit/bin\nwatch_file kit.yml kit.lock\n"

// kitEnv is what .kit/env holds: the same words, for direnv's source_env .kit/env, which reads them from .kit/.
const kitEnv = "PATH_add bin\nwatch_file ../kit.yml ../kit.lock\n"

// programOf gives where the cache keeps a locked file's program: in a folder named for the program's own digest, so
// that two tools from one archive are each kept whole.
func programOf(cache string, f LockedFile) (dir, program string) {
	dir = cache + "/tools/" + hexOf(f.ProgramDigest)
	return dir, dir + "/" + path.Base(f.Program)
}

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

// fill makes a folder of the cache, which is not there yet: it writes the file into the folder's .part, and names the
// folder for its digest only once the file is whole. A .part left by a fetch that stopped is made again.
func fill(files ports.Files, dir, name string, data []byte) error {
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

// Fetch plans to fetch every tool the lock holds for this platform into the cache; to place a copy of kit at
// .kit/kit, a link in .kit/bin to it for each tool, and .kit/env for direnv; and to take out the link of a tool the
// lock no longer holds. The links lead into .kit/, not the cache, so a cache that is cleared costs a fetch and never
// a link that leads nowhere. It changes nothing that git sees, so it is hydrated, with no approval.
func Fetch(ctx context.Context, d Desk, r Repo) (plan.Plan, error) {
	if err := Allowed(ctx, d, r); err != nil {
		return plan.Plan{}, err
	}
	cache, err := d.Machine.Cache()
	if err != nil {
		return plan.Plan{}, err
	}
	kit, err := d.Machine.Executable()
	var body []byte
	if err == nil {
		body, err = d.Files.Read(kit)
	}
	var placed []string
	if err == nil {
		placed, err = plan.Placed(d.Files, r.Root)
	}
	if err != nil {
		return plan.Plan{}, err
	}
	ops := []plan.Operation{plan.PlaceProgram(d.Files, d.Git, r.Root, "kit", body)}
	for _, name := range slices.Sorted(maps.Keys(r.Lock.Tools)) {
		f, err := file(d, r, name)
		if err != nil {
			return plan.Plan{}, err
		}
		dir, _ := programOf(cache, f)
		ops = append(ops, fetch{d, name, f, dir}, plan.Place(d.Files, d.Git, r.Root, "bin/"+name, nil, "../kit"))
	}
	for _, name := range placed {
		if tool, ok := strings.CutPrefix(name, "bin/"); ok && r.Lock.Tools[tool].Version == "" {
			ops = append(ops, plan.Unplace(d.Files, d.Git, r.Root, name))
		}
	}
	ops = append(ops, plan.Place(d.Files, d.Git, r.Root, "env", []byte(kitEnv), ""))
	return plan.Plan{Act: "tools fetch", Operations: ops}, nil
}

// checked gives the path of a tool's program in the cache, fetched where it is not yet, once its digest is the one
// the lock names.
func checked(ctx context.Context, d Desk, r Repo, name string) (string, error) {
	if err := Allowed(ctx, d, r); err != nil {
		return "", err
	}
	f, err := file(d, r, name)
	if err != nil {
		return "", err
	}
	cache, err := d.Machine.Cache()
	if err != nil {
		return "", err
	}
	dir, program := programOf(cache, f)
	body, err := d.Files.Read(program)
	if errors.Is(err, fs.ErrNotExist) {
		var p plan.Plan
		if p, err = Fetch(ctx, d, r); err == nil {
			err = plan.Hydrate(ctx, p).Err
		}
		if err != nil {
			return "", err
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
// in kit's place with the same arguments. The repository is always the one whose .kit/bin holds the link: where kit
// was started by the name alone, that is the first folder on the PATH that holds the name, as the shell found it,
// and never the repository kit happens to run in.
func RunTool(ctx context.Context, d Desk, argv0 string, args []string) error {
	link := argv0
	if !strings.Contains(link, "/") {
		link = ""
		for _, dir := range pathOf(d.Machine.Environ()) {
			if _, err := d.Files.Stat(dir + "/" + argv0); err == nil {
				link = dir + "/" + argv0
				break
			}
		}
	}
	if !path.IsAbs(link) {
		link = path.Join(d.Dir, link)
	}
	bin := path.Dir(link)
	if path.Base(bin) != "bin" || path.Base(path.Dir(bin)) != ".kit" {
		return fmt.Errorf("kit was started as %s, and no .kit/bin on the PATH holds %s", argv0, path.Base(argv0))
	}
	root := path.Dir(path.Dir(bin))
	r, err := openAt(d, root)
	if err != nil {
		return err
	}
	if _, err := file(d, r, path.Base(argv0)); err != nil {
		return err
	}
	if _, r.Common, err = d.Git.Root(ctx, root); err != nil {
		return err
	}
	program, err := checked(ctx, d, r, path.Base(argv0))
	if err != nil {
		return err
	}
	return d.Processes.Replace(ports.Cmd{Path: program, Args: args, Env: d.Machine.Environ()})
}

// Exec runs a command in kit's place, with the pinned tools fetched and checked, and first on the PATH. The PATH leads
// straight to the programs, each checked against the lock as Exec starts, through a folder of links the cache keeps
// for this lock: the command, and what it runs, need no trust of their own, so a trust given for this run alone
// reaches them too.
func Exec(ctx context.Context, d Desk, r Repo, argv []string) error {
	cache, err := d.Machine.Cache()
	if err != nil {
		return err
	}
	run := cache + "/runs/" + hexOf(digestOf(append(r.Lock.JSON(), d.Machine.Platform()...)))
	for _, name := range slices.Sorted(maps.Keys(r.Lock.Tools)) {
		program, err := checked(ctx, d, r, name)
		if err == nil {
			err = linked(d.Files, program, run+"/"+name)
		}
		if err != nil {
			return err
		}
	}
	return d.Processes.Replace(ports.Cmd{Path: argv[0], Args: argv[1:], Env: withPath(d.Machine.Environ(), run)})
}

// linked makes path a link to target, where it is not one already.
func linked(files ports.Files, target, path string) error {
	if now, err := files.ReadLink(path); err == nil && now == target {
		return nil
	}
	if err := files.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return files.Link(target, path)
}

// pathOf gives the folders of the PATH an environment holds.
func pathOf(env []string) []string {
	for _, kv := range env {
		if dirs, ok := strings.CutPrefix(kv, "PATH="); ok {
			return strings.Split(dirs, ":")
		}
	}
	return nil
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
		_, program := programOf(cache, f)
		switch body, readErr := d.Files.Read(program); {
		case err != nil:
			state = strings.Replace(err.Error(), "of "+pin.Name+" ", "of it ", 1)
		case readErr != nil:
		case digestOf(body) == f.ProgramDigest:
			state = "fetched here, and checked"
		default:
			state = "fetched here, and not what kit.lock names"
		}
		for _, f := range r.Lock.Tools[pin.Name].Files {
			if len(f.Confirmed) == 0 {
				state += "; nothing confirmed a file's digest but kit's own download"
				break
			}
		}
		listed = append(listed, Listed{pin.Name, pin.Version, "kit.yml", state})
	}
	slices.SortFunc(listed, func(a, b Listed) int { return strings.Compare(a.Name, b.Name) })
	return listed, nil
}
