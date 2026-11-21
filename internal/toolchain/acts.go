package toolchain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/empowerite/kit/internal/data"
	"github.com/empowerite/kit/internal/plan"
	"github.com/empowerite/kit/internal/ports"
)

// Desk is the machine kit runs on: where it runs, and the systems it reads and changes there.
type Desk struct {
	Dir       string
	Files     ports.Files
	Git       ports.Git
	Releases  ports.Releases
	Downloads ports.Downloads
	Presence  ports.Presence
	Machine   ports.Machine
	Processes ports.Processes
}

// Repo is a repository as kit reads it: its root, the clone's common directory, its kit.yml, and its kit.lock.
type Repo struct {
	Root, Common string
	KitYML       string // its text, or "" where there is none
	KitLock      []byte // its text, or nil where there is none
	Pins         []Pin
	Lock         Lock
}

// read gives a file's bytes, or nil where there is no such file.
func read(files ports.Files, path string) ([]byte, error) {
	text, err := files.Read(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return text, err
}

// Open finds the repository that holds the desk's directory, and reads its pins and its lock.
func Open(ctx context.Context, d Desk) (Repo, error) {
	root, common, err := d.Git.Root(ctx, d.Dir)
	if err != nil {
		return Repo{}, err
	}
	r, err := openAt(d, root)
	r.Common = common
	return r, err
}

// openAt reads the pins and the lock of the repository whose root is given.
func openAt(d Desk, root string) (Repo, error) {
	r := Repo{Root: root}
	yml, err := read(d.Files, r.Root+"/kit.yml")
	if err != nil {
		return r, err
	}
	r.KitYML = string(yml)
	if r.Pins, err = ReadPins(data.Source{Name: "kit.yml", Text: yml}); err != nil {
		return r, err
	}
	if r.KitLock, err = read(d.Files, r.Root+"/kit.lock"); err != nil {
		return r, err
	}
	r.Lock, err = ReadLock(r.KitLock)
	return r, err
}

// pinOf gives the repository's pin of a tool, or a pin with no entry of its own where it pins none.
func (r Repo) pinOf(name string) Pin {
	for _, p := range r.Pins {
		if p.Name == name {
			return p
		}
	}
	return Pin{Name: name}
}

// entryFor gives the entry a pin is fetched by, or the finding's words where no list holds one.
func entryFor(pin Pin, list map[string]Entry) (Entry, error) {
	if e := EntryOf(pin, list); e.GitHub != "" {
		return e, nil
	}
	return Entry{}, fmt.Errorf("kit's list holds no %s: give its entry under tools: in kit.yml, "+
		"in the shape kit data show tools gives", pin.Name)
}

// region is kit's line in an .envrc: it has direnv put the pinned tools on the PATH. Its markers let kit find it,
// and take it out again.
const region = "# kit:begin - kit's line: it puts the pinned tools on the PATH; kit put it here, and can take it out\n" +
	"eval \"$(kit env --shell=direnv)\"\n" +
	"# kit:end\n"

// writes gives the steps that write kit.yml, kit.lock, and kit's line in an .envrc that exists, each only where it
// changes, and each saying what it changes.
func (r Repo) writes(d Desk, yml string, pinned []string, lock Lock) ([]plan.Operation, error) {
	var ops []plan.Operation
	if yml != r.KitYML {
		ops = append(ops, plan.WriteFile(d.Files, r.Root, "kit.yml", []byte(yml), pinned...))
	}
	if text := lock.JSON(); !bytes.Equal(text, r.KitLock) {
		var detail []string
		for _, name := range slices.Sorted(maps.Keys(lock.Tools)) {
			if l := lock.Tools[name]; !reflect.DeepEqual(l, r.Lock.Tools[name]) {
				detail = append(detail, said(name, l)...)
			}
		}
		ops = append(ops, plan.WriteFile(d.Files, r.Root, "kit.lock", text, detail...))
	}
	envrc, err := read(d.Files, r.Root+"/.envrc")
	if err != nil {
		return nil, err
	}
	if envrc != nil && !strings.Contains(string(envrc), "# kit:begin") {
		text := string(envrc)
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		ops = append(ops, plan.WriteFile(d.Files, r.Root, ".envrc", []byte(text+region),
			"adds kit's line, eval \"$(kit env --shell=direnv)\", between # kit:begin and # kit:end"))
	}
	return ops, nil
}

// said says a locked tool as a plan shows it: each platform's file, its digest, and what confirmed it.
func said(name string, l Locked) []string {
	lines := []string{fmt.Sprintf("%s %s, for each platform:", name, l.Version)}
	for _, p := range Platforms {
		f := l.Files[p]
		by := "confirmed by " + strings.Join(f.Confirmed, " and ")
		if len(f.Confirmed) == 0 {
			by = "confirmed by nothing: pinned with --unconfirmed"
		}
		lines = append(lines, fmt.Sprintf("  %s  %s", p, f.URL), fmt.Sprintf("    %s, %s", f.Digest, by))
	}
	return lines
}

// Add plans to pin a tool, given as name or name@version, and to lock it for every platform. With no version, it is
// the tool's newest release. The plan writes kit.yml and kit.lock, each only where it changes.
func Add(ctx context.Context, d Desk, r Repo, list map[string]Entry, tool string, unconfirmed bool) (plan.Plan,
	Locked, error) {
	name, version, _ := strings.Cut(tool, "@")
	e, err := entryFor(r.pinOf(name), list)
	if err != nil {
		return plan.Plan{}, Locked{}, err
	}
	if version == "" {
		if version, err = newest(ctx, d.Releases, e); err != nil {
			return plan.Plan{}, Locked{}, err
		}
	}
	l, err := Resolve(ctx, d.Releases, d.Downloads, name, version, e, unconfirmed)
	if err != nil {
		return plan.Plan{}, Locked{}, err
	}
	yml, err := Repin(r.KitYML, name, version)
	if err != nil {
		return plan.Plan{}, Locked{}, err
	}
	lock := Lock{Tools: maps.Clone(r.Lock.Tools)}
	lock.Tools[name] = l
	ops, err := r.writes(d, yml, []string{"pins " + name + " at " + version}, lock)
	return plan.Plan{Act: "tools add " + name + "@" + version, Operations: ops}, l, err
}

// newest gives the version of a tool's newest release, from its tag.
func newest(ctx context.Context, releases ports.Releases, e Entry) (string, error) {
	tag, err := releases.Newest(ctx, e.GitHub)
	if err != nil {
		return "", err
	}
	before, after, _ := strings.Cut(e.Tag, "{version}")
	version, ok := strings.CutPrefix(tag, before)
	version, okAfter := strings.CutSuffix(version, after)
	if !ok || !okAfter || version == "" {
		return "", fmt.Errorf("the newest release of %s is tagged %s, which %s does not name", e.GitHub, tag, e.Tag)
	}
	return version, nil
}

// Relock plans to lock every pin again, for every platform, and to keep nothing else in kit.lock.
func Relock(ctx context.Context, d Desk, r Repo, list map[string]Entry, unconfirmed bool) (plan.Plan, error) {
	lock := Lock{Tools: map[string]Locked{}}
	for _, pin := range r.Pins {
		e, err := entryFor(pin, list)
		if err != nil {
			return plan.Plan{}, err
		}
		if lock.Tools[pin.Name], err = Resolve(ctx, d.Releases, d.Downloads, pin.Name, pin.Version, e, unconfirmed); err != nil {
			return plan.Plan{}, err
		}
	}
	ops, err := r.writes(d, r.KitYML, nil, lock)
	return plan.Plan{Act: "tools lock", Operations: ops}, err
}
