// Package layers holds the rules this tree keeps: the layers of the design and which of them each may import, who
// may start a process, open a connection, or touch a file, and that every Go file is formatted and in every build.
// Its test reads this tree and holds it to the rules.
package layers

import (
	"bytes"
	"errors"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
)

// File is one Go file of the tree: its path from the root, and what the rules ask of it.
type File struct {
	Path        string // as "internal/model/doc.go"
	Test        bool   // a _test.go file
	Nightly     bool   // a test that begins //go:build nightly, and so runs in the nightly tier alone
	Constraint  string // what holds it to only some builds: its //go:build line, or its name's ending; none is ""
	Unformatted bool   // gofmt would change it
	Imports     []string
}

// Violation is one thing a rule refuses: a file, the import at fault where there is one, and why.
type Violation struct{ File, Import, Why string }

// Layers gives, for each package of the tree by its directory, the packages of kit's it may import. "*" is any.
var Layers = map[string][]string{
	"internal/model":           {},
	"internal/data":            {"internal/model"},
	"internal/expr":            {"internal/model"},
	"internal/ports":           {"internal/model"},
	"internal/lifecycle":       {"internal/model", "internal/data", "internal/expr"},
	"internal/plan":            {"internal/model", "internal/lifecycle", "internal/ports"},
	"internal/toolchain":       {"internal/model", "internal/data", "internal/ports", "internal/plan"},
	"internal/doc":             {"internal/model", "internal/lifecycle", "internal/plan", "internal/ports", "internal/toolchain"},
	"internal/cli":             {"internal/doc", "internal/plan", "internal/data", "internal/toolchain"},
	"internal/tui":             {"internal/doc"},
	"internal/adapters/github": {"internal/ports", "internal/model"},
	"internal/adapters/git":    {"internal/ports", "internal/model"},
	"internal/adapters/os":     {"internal/ports", "internal/model"},
	"internal/adapters/fake":   {"internal/ports", "internal/model"},
	"cmd/kit":                  {"*"},
	"tools/forge":              {},
	"tools/gate":               {},
	"tools/layers":             {},
}

// fake holds the adapters that are in memory. Any test may import it, and it reaches nothing outside the program.
const fake = "internal/adapters/fake"

// What a package of the standard library reaches outside the program.
const (
	reachesNothing = iota // it is pure
	reachesFiles          // it touches a file
	reachesBeyond         // it starts a process or opens a connection
)

// reach says what an import reaches outside the program.
func reach(imported string) int {
	switch {
	case slices.Contains([]string{"os", "io/ioutil", "path/filepath"}, imported):
		return reachesFiles
	case slices.Contains([]string{"net/url", "net/netip"}, imported):
		return reachesNothing
	case imported == "net", strings.HasPrefix(imported, "net/"), strings.HasPrefix(imported, "os/"),
		slices.Contains([]string{"syscall", "plugin", "C", "crypto/tls", "log/syslog"}, imported):
		return reachesBeyond
	}
	return reachesNothing
}

// The systems and machines a file's name can hold it to, as Go knows them.
var (
	systems = []string{"aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js", "linux",
		"nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows", "zos"}
	machines = []string{"386", "amd64", "arm", "arm64", "loong64", "mips", "mips64", "mips64le", "mipsle", "ppc64",
		"ppc64le", "riscv64", "s390x", "wasm"}
)

// Read gives every Go file under fsys, but those under testdata, and under worktrees, vendor, or .git at the root. A
// go.mod below the root is an error: kit is one module, and go test ./... would leave such a directory out.
func Read(fsys fs.FS) ([]File, error) {
	var files []File
	err := fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		root := !strings.Contains(name, "/")
		apart := entry.Name() == "testdata" || root && slices.Contains([]string{"worktrees", "vendor", ".git"}, name)
		switch {
		case entry.IsDir() && apart:
			return fs.SkipDir
		case entry.IsDir():
			return nil
		case entry.Name() == "go.mod" && !root:
			return errors.New(name + ": a go.mod below the root: kit is one module")
		case !strings.HasSuffix(name, ".go"):
			return nil
		}
		source, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		positions := token.NewFileSet()
		parsed, err := parser.ParseFile(positions, name, source, parser.ImportsOnly)
		if err != nil {
			return err
		}
		head := string(source[:positions.Position(parsed.Package).Offset]) // what stands before the package clause
		formatted, err := format.Source(source)
		file := File{Path: name, Test: strings.HasSuffix(name, "_test.go"), Constraint: constraint(name, head)}
		file.Unformatted = err == nil && !bytes.Equal(formatted, source)
		if file.Test && file.Constraint == "//go:build nightly" {
			file.Nightly, file.Constraint = true, ""
		}
		for _, spec := range parsed.Imports {
			imported, _ := strconv.Unquote(spec.Path.Value)
			file.Imports = append(file.Imports, imported)
		}
		files = append(files, file)
		return nil
	})
	return files, err
}

// constraint gives what holds a file to only some builds: the //go:build line in its head, which is what stands
// before its package clause, or the ending of its name that names a system or a machine, as _windows and _arm64
// do. It gives "" for a file in every build.
func constraint(name, head string) string {
	for _, line := range strings.Split(head, "\n") {
		if strings.HasPrefix(line, "//go:build") {
			return strings.TrimSpace(line)
		}
	}
	stem := strings.TrimSuffix(strings.TrimSuffix(path.Base(name), ".go"), "_test")
	if at := strings.LastIndex(stem, "_"); at >= 0 && slices.Contains(append(systems, machines...), stem[at+1:]) {
		return "its name ends " + stem[at:]
	}
	return ""
}

// Check holds the files to the rules, for a module of that path.
func Check(module string, files []File) []Violation {
	var violations []Violation
	for _, file := range files {
		dir := path.Dir(file.Path)
		refuse := func(imported, why string) {
			violations = append(violations, Violation{file.Path, imported, why})
		}
		may, known := Layers[dir]
		if !known {
			refuse("", dir+" is in no layer: add it to the table in tools/layers")
			continue
		}
		if file.Unformatted {
			refuse("", "gofmt would change it: run gofmt -w "+file.Path)
		}
		if file.Constraint != "" {
			refuse("", "only some builds hold this file ("+file.Constraint+"), so the gate cannot say its statements are run")
		}
		adapter := strings.HasPrefix(dir, "internal/adapters/") && dir != fake
		outer := adapter || dir == "cmd/kit" || strings.HasPrefix(dir, "tools/")
		for _, imported := range file.Imports {
			kits, of := strings.CutPrefix(imported, module+"/")
			reaches := reach(imported)
			switch {
			case of && (slices.Contains(may, "*") || slices.Contains(may, kits) || file.Test && (kits == fake || kits == dir)):
			case of:
				refuse(imported, dir+" may not import "+kits)
			case file.Test && !file.Nightly && reaches == reachesBeyond:
				refuse(imported, "a test the gate runs starts no process and reaches no network")
			case file.Test:
			case reaches == reachesBeyond && !adapter, reaches == reachesFiles && !outer:
				refuse(imported, "only an adapter starts a process, opens a connection, or touches a file")
			}
		}
	}
	return violations
}
