package toolchain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/empowerite/kit/internal/plan"
	"github.com/empowerite/kit/internal/ports"
)

// miseFiles are the files of a clone that mise reads: its configuration, where it may stand, the local ones beside
// it, the .tool-versions it reads too, and its lock.
var miseFiles = []string{"mise.toml", ".mise.toml", ".config/mise.toml", "mise/config.toml", ".config/mise/config.toml",
	"mise.local.toml", ".mise.local.toml", ".config/mise/config.local.toml", ".tool-versions", "mise.lock"}

// covers are the files a clone's trust covers, by their paths from the root: what names the tools kit fetches and
// runs, what names the tools mise runs for kit exec, and the .envrc that direnv runs, those of them that exist.
var covers = append([]string{"kit.yml", "kit.lock", ".envrc"}, miseFiles...)

// Covered is what a trust covers: each file by its path and its sha256, and the tools kit.lock pins, so that kit can
// say what changed since.
type Covered struct {
	Files map[string]string `json:"files"`
	Tools map[string]Seen   `json:"tools"`
}

// Seen is a tool as a trust saw it: its version, and the addresses of its files.
type Seen struct {
	Version string   `json:"version"`
	URLs    []string `json:"urls"`
}

// cover reads what a clone's trust covers, as the repository holds it now.
func cover(d Desk, r Repo) (Covered, error) {
	c := Covered{Files: map[string]string{}, Tools: map[string]Seen{}}
	for _, name := range covers {
		text, err := read(d.Files, r.Root+"/"+name)
		if err != nil {
			return c, err
		}
		if text != nil {
			c.Files[name] = digestOf(text)
		}
	}
	for name, l := range r.Lock.Tools {
		var urls []string
		for _, p := range Platforms {
			if f, ok := l.Files[p]; ok {
				urls = append(urls, f.URL)
			}
		}
		c.Tools[name] = Seen{l.Version, urls}
	}
	return c, nil
}

// digest gives the digest of what a trust covers: of the files alone, since the tools are read from them.
func (c Covered) digest() string {
	sum := sha256.New()
	for _, name := range slices.Sorted(maps.Keys(c.Files)) {
		fmt.Fprintf(sum, "%s %s\n", name, c.Files[name])
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil))
}

// store is what kit keeps of the person's trust, in its state: for each clone, by its common directory, which every
// worktree of the clone shares, the digests of each set of contents trusted, and the last one, to say what changed.
type store struct {
	Clones map[string]clone `json:"clones"`
}

type clone struct {
	Trusted []string `json:"trusted"`
	Last    Covered  `json:"last"`
}

// readStore reads the person's store of trust, and gives its path.
func readStore(d Desk) (store, string, error) {
	state, err := d.Machine.State()
	if err != nil {
		return store{}, "", err
	}
	path := state + "/trust.json"
	text, err := read(d.Files, path)
	s := store{Clones: map[string]clone{}}
	if err == nil && text != nil {
		if err = json.Unmarshal(text, &s); err != nil {
			err = fmt.Errorf("%s: %w", path, err)
		}
	}
	return s, path, err
}

// Allowed says whether kit may fetch and run what the clone names: the person trusts what it holds now, or trusted
// the checkout for this run alone.
func Allowed(ctx context.Context, d Desk, r Repo) error {
	if d.Once {
		return nil
	}
	s, _, err := readStore(d)
	if err != nil {
		return err
	}
	c, err := cover(d, r)
	if err != nil {
		return err
	}
	known, ok := s.Clones[r.Common]
	switch {
	case ok && slices.Contains(known.Trusted, c.digest()):
		return nil
	case ok:
		return fmt.Errorf("what kit trusted in this clone has changed since: kit trust shows what changed, and asks you again")
	}
	return fmt.Errorf("kit runs nothing in this clone until you trust it: kit trust shows what it would run, and asks you")
}

// Changes says what changed between what was trusted and what the clone holds now: which tool, which version, and
// which addresses, and which files.
func Changes(before, after Covered) []string {
	var said []string
	for _, name := range slices.Sorted(maps.Keys(after.Tools)) {
		now, was := after.Tools[name], before.Tools[name]
		switch {
		case was.Version == "":
			said = append(said, fmt.Sprintf("%s %s is new, from:", name, now.Version))
		case was.Version != now.Version || !slices.Equal(was.URLs, now.URLs):
			said = append(said, fmt.Sprintf("%s %s, which was %s, from:", name, now.Version, was.Version))
		default:
			continue
		}
		for _, url := range now.URLs {
			said = append(said, "  "+url)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(before.Tools)) {
		if _, ok := after.Tools[name]; !ok {
			said = append(said, name+" is no longer pinned")
		}
	}
	for _, name := range covers {
		now, was := after.Files[name], before.Files[name]
		switch {
		case now == was:
		case was == "":
			said = append(said, name+" is new")
		case now == "":
			said = append(said, name+" is gone")
		default:
			said = append(said, name+" changed")
		}
	}
	return said
}

// TrustPlan plans the person's trust of what the clone holds now, and says what changed since they last trusted it.
// Where there is an .envrc, and direnv is installed, the same act runs direnv allow, so the person trusts once. To
// trust is a person's decision, so the plan is approved with presence.
func TrustPlan(ctx context.Context, d Desk, r Repo) (plan.Plan, []string, error) {
	s, path, err := readStore(d)
	if err != nil {
		return plan.Plan{}, nil, err
	}
	c, err := cover(d, r)
	if err != nil {
		return plan.Plan{}, nil, err
	}
	known := s.Clones[r.Common]
	changes := Changes(known.Last, c)
	if !slices.Contains(known.Trusted, c.digest()) {
		known.Trusted = append(known.Trusted, c.digest())
	}
	known.Last = c
	s.Clones[r.Common] = known
	text, _ := json.MarshalIndent(s, "", "  ")
	ops := []plan.Operation{record{d.Files, path, append(text, '\n'), slices.Sorted(maps.Keys(c.Files))}}
	if _, ok := c.Files[".envrc"]; ok && DirenvState(ctx, d, r.Root) != "direnv is not installed here" {
		ops = append(ops, allow{d.Processes, d.Files, r.Root + "/.envrc"})
	}
	return plan.Plan{Act: "trust", Operations: ops}, changes, nil
}

// record writes the person's store of trust.
type record struct {
	files ports.Files
	path  string
	data  []byte
	names []string
}

func (o record) Step() plan.Step {
	return plan.Step{System: "trust", Does: "trust what this clone holds: " + strings.Join(o.names, ", "), Undoable: true}
}

func (o record) Facts(context.Context) (string, error) {
	now, err := read(o.files, o.path)
	return digestOf(now) + " to " + digestOf(o.data), err
}

func (o record) Apply(context.Context) error { return o.files.Write(o.path, o.data, 0o600) }

// allow runs direnv allow, so that direnv runs the .envrc the person trusted with kit.
type allow struct {
	processes ports.Processes
	files     ports.Files
	envrc     string
}

func (o allow) Step() plan.Step {
	return plan.Step{System: "direnv", Does: "direnv allow " + o.envrc, Command: "direnv allow " + o.envrc, Undoable: true}
}

// Facts covers the .envrc's bytes, so that direnv is asked to allow the .envrc the person approved, and no other.
func (o allow) Facts(context.Context) (string, error) {
	text, err := o.files.Read(o.envrc)
	return o.envrc + " " + digestOf(text), err
}

func (o allow) Apply(ctx context.Context) error {
	code, err := o.processes.Run(ctx, ports.Cmd{Path: "direnv", Args: []string{"allow", o.envrc}})
	if err == nil && code != 0 {
		err = fmt.Errorf("direnv allow %s ended with %d", o.envrc, code)
	}
	return err
}

// DirenvState says whether direnv has allowed the .envrc of a folder, from direnv's own status. kit says it, and
// never takes it as the person's consent.
func DirenvState(ctx context.Context, d Desk, dir string) string {
	var out bytes.Buffer
	code, err := d.Processes.Run(ctx, ports.Cmd{Path: "direnv", Args: []string{"status", "--json"}, Dir: dir, Stdout: &out})
	if err != nil {
		return "direnv is not installed here"
	}
	var status struct {
		State struct {
			FoundRC *struct {
				Allowed int `json:"allowed"`
			} `json:"foundRC"`
		} `json:"state"`
	}
	if code != 0 || json.Unmarshal(out.Bytes(), &status) != nil {
		return "direnv's status cannot be read"
	}
	switch {
	case status.State.FoundRC == nil:
		return "direnv finds no .envrc here"
	case status.State.FoundRC.Allowed == 0:
		return "direnv has allowed this .envrc"
	case status.State.FoundRC.Allowed == 2:
		return "direnv has denied this .envrc: kit trust covers it"
	}
	return "direnv has not allowed this .envrc: kit trust covers it"
}
