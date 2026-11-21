package toolchain

import (
	"context"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/empowerite/kit/internal/ports"
)

// Manager is a tool manager kit found in a repository: its name, and the file that shows it is in use.
type Manager struct {
	Name, File string
}

// managers are the tool managers kit knows by their files. kit reads mise's; it names the others, and says that it
// does not read them yet.
var managers = []struct {
	name  string
	files []string
}{
	{"mise", []string{"mise.toml", ".mise.toml", ".config/mise.toml", "mise/config.toml", ".config/mise/config.toml"}},
	{"asdf", []string{".tool-versions"}},
	{"aqua", []string{"aqua.yaml", ".aqua.yaml", "aqua.yml"}},
	{"Nix", []string{"flake.nix"}},
	{"devbox", []string{"devbox.json"}},
	{"Homebrew", []string{"Brewfile"}},
}

// Detect gives the tool managers a repository shows it uses, in the order kit knows them.
func Detect(files ports.Files, root string) ([]Manager, error) {
	var found []Manager
	for _, m := range managers {
		for _, name := range m.files {
			text, err := read(files, root+"/"+name)
			if err != nil {
				return nil, err
			}
			if text != nil {
				found = append(found, Manager{m.name, name})
				break
			}
		}
	}
	return found, nil
}

// misePlatforms are the four platforms as mise.lock names them.
var misePlatforms = []string{"linux-arm64", "linux-x64", "macos-arm64", "macos-x64"}

// relock is the fix of a lock that lacks a platform or a digest.
const relockMise = "mise lock --platform=linux-arm64,linux-x64,macos-arm64,macos-x64"

// miseTools gives the tools mise.toml pins, with each version as mise.toml writes it.
func miseTools(text []byte) (map[string]string, error) {
	var file struct {
		Tools map[string]any `toml:"tools"`
	}
	if _, err := toml.Decode(string(text), &file); err != nil {
		return nil, fmt.Errorf("mise.toml: %w", err)
	}
	tools := map[string]string{}
	for name, v := range file.Tools {
		switch v := v.(type) {
		case string:
			tools[name] = v
		case []any:
			var each []string
			for _, one := range v {
				each = append(each, fmt.Sprint(one))
			}
			tools[name] = strings.Join(each, ", ")
		case map[string]any:
			tools[name] = fmt.Sprint(v["version"])
		default:
			tools[name] = fmt.Sprint(v)
		}
	}
	return tools, nil
}

// CheckMise holds mise.toml and mise.lock to the charter's bar: a lock that is checked in, with each tool locked for
// each of the four platforms, by a digest. Each finding says what, why, and the fix. kit changes no file of mise's.
func CheckMise(tomlText, lockText []byte, tracked bool) ([]Finding, error) {
	tools, err := miseTools(tomlText)
	if err != nil {
		return nil, err
	}
	if lockText == nil {
		return []Finding{{"mise.toml", "mise keeps no lock",
			"without a lock, mise chooses a version when it installs, so a desk and CI can run different tools",
			"set lockfile = true under [settings] in mise.toml, run " + relockMise + ", and commit mise.lock"}}, nil
	}
	var lock struct {
		Tools map[string][]map[string]any `toml:"tools"`
	}
	if _, err := toml.Decode(string(lockText), &lock); err != nil {
		return nil, fmt.Errorf("mise.lock: %w", err)
	}
	var found []Finding
	add := func(what, why, fix string) { found = append(found, Finding{"mise.lock", what, why, fix}) }
	if !tracked {
		add("mise.lock is not checked in", "CI does not see a lock that only a desk holds", "git add mise.lock, and commit it")
	}
	for _, name := range slices.Sorted(maps.Keys(tools)) {
		entries := lock.Tools[name]
		if len(entries) == 0 {
			add("mise.lock holds no "+name, "mise.toml pins it, and no digest says which bytes to run", "mise lock")
			continue
		}
		entry := entries[0]
		platforms := map[string]map[string]any{}
		for key, v := range entry {
			if p, ok := strings.CutPrefix(key, "platforms."); ok {
				platforms[p], _ = v.(map[string]any)
			}
		}
		if len(platforms) == 0 {
			why, fix := byVersion(name)
			add(fmt.Sprintf("%s %v has a version and no digest", name, entry["version"]), why, fix)
			continue
		}
		for _, p := range misePlatforms {
			file, ok := platforms[p]
			switch {
			case !ok:
				add(fmt.Sprintf("mise.lock holds no file of %s for %s", name, p), "a machine of that platform has nothing locked", relockMise)
			case file["checksum"] == nil || file["checksum"] == "":
				add(fmt.Sprintf("mise.lock holds no digest of %s for %s", name, p), "nothing holds what that platform fetches to its bytes", relockMise)
			}
		}
	}
	return found, nil
}

// byVersion says why a tool mise locks by its version alone breaks "the same tools", and the fix, by its backend.
func byVersion(name string) (string, string) {
	switch {
	case strings.HasPrefix(name, "npm:"):
		return "mise installs it from npm by version, and npm chooses its dependencies when it installs, so a desk " +
				"and CI can run different code",
			"pin it in package.json, with package-lock.json checked in, and run it with npx"
	case strings.HasPrefix(name, "pipx:"):
		return "mise installs it with pipx by version, and pip chooses its dependencies when it installs, so a desk " +
				"and CI can run different code",
			"pin it in the repository's own Python lock, such as uv.lock, with its hashes"
	}
	return "mise installs it by version alone, and nothing holds the bytes it fetches to a digest",
		"pin it through a source that keeps a digest, such as mise's aqua: or github: backend, where the tool publishes programs"
}

// CheckAll holds a repository's tools to the charter's bar, with the tool manager it uses: mise's files, or kit's own
// pins and lock where kit is the tool manager. A tool manager kit does not read is named. Where there is nothing to
// work from, it says so, and offers the two fixes.
func CheckAll(ctx context.Context, d Desk, r Repo, list map[string]Entry) ([]Finding, error) {
	ms, err := Detect(d.Files, r.Root)
	if err != nil {
		return nil, err
	}
	var found []Finding
	mise := false
	for _, m := range ms {
		if m.Name != "mise" {
			found = append(found, Finding{m.File, fmt.Sprintf("%s is in use (%s), and kit does not read it yet", m.Name, m.File),
				"kit cannot check that a desk and CI run the same tools", "none from kit yet: kit reads mise alone for now"})
			continue
		}
		mise = true
		tomlText, _ := read(d.Files, r.Root+"/"+m.File)
		lockText, err := read(d.Files, r.Root+"/mise.lock")
		tracked := false
		if err == nil && lockText != nil {
			tracked, err = d.Git.Tracked(ctx, r.Root, "mise.lock")
		}
		var more []Finding
		if err == nil {
			more, err = CheckMise(tomlText, lockText, tracked)
		}
		if err != nil {
			return nil, err
		}
		found = append(found, more...)
	}
	switch {
	case mise && len(r.Pins) > 0:
		found = append(found, Finding{"kit.yml", "kit.yml pins tools, and mise is in use",
			"two tool managers can pin one tool two ways, and a desk and CI may take different ones",
			"pin them in mise.toml, and take tools: out of kit.yml"})
	case !mise && r.KitYML != "":
		found = append(found, CheckLock(r.Pins, list, r.Lock)...)
	case len(ms) == 0:
		found = append(found, Finding{".", "kit has nothing to work from: no kit.yml, and no tool manager",
			"nothing pins the tools this repository's work runs with",
			"adopt kit: kit tools add <tool> writes kit.yml and kit.lock; or, with a tool manager in place, set up its lock"})
	}
	return found, nil
}

// throughMise runs a command through the mise on the machine, once kit may, saying first what the check finds.
func throughMise(ctx context.Context, d Desk, r Repo, argv []string, said io.Writer) error {
	if err := Allowed(ctx, d, r); err != nil {
		return err
	}
	found, err := CheckAll(ctx, d, r, nil)
	if err != nil {
		return err
	}
	for _, f := range found {
		fmt.Fprintf(said, "kit: %s: %s\n", f.Where, f.What)
	}
	if len(found) > 0 {
		fmt.Fprintln(said, "kit: kit tools check says each in full, with its fix")
	}
	if err := d.Processes.Replace(ports.Cmd{Path: "mise", Args: append([]string{"exec", "--"}, argv...), Env: d.Machine.Environ()}); err != nil {
		return fmt.Errorf("mise is not installed here, and kit does not install it (%v). mise's own page says how to "+
			"install it: https://mise.jdx.dev/getting-started.html", err)
	}
	return nil
}
