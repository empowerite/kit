// Package layers holds the rule on imports: the layers of the design, which of them each may import, and who may
// start a process, open a connection, or touch a file. Its test reads this tree and holds it to the rule.
package layers

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
)

// File is one Go file of the tree: its path from the root, and what it imports.
type File struct {
	Path          string // as "internal/model/doc.go"
	Test, Nightly bool   // a _test.go file; a file that begins //go:build nightly
	Imports       []string
}

// Violation is one import the rule refuses.
type Violation struct{ File, Import, Why string }

// Layers gives, for each package of the tree by its directory, the packages of kit's it may import. "*" is any.
var Layers = map[string][]string{
	"internal/model":           {},
	"internal/data":            {"internal/model"},
	"internal/expr":            {"internal/model"},
	"internal/ports":           {"internal/model"},
	"internal/lifecycle":       {"internal/model", "internal/data", "internal/expr"},
	"internal/plan":            {"internal/model", "internal/lifecycle", "internal/ports"},
	"internal/doc":             {"internal/model", "internal/lifecycle", "internal/plan", "internal/ports"},
	"internal/cli":             {"internal/doc", "internal/plan", "internal/data"},
	"internal/tui":             {"internal/doc"},
	"internal/adapters/github": {"internal/ports", "internal/model"},
	"internal/adapters/git":    {"internal/ports", "internal/model"},
	"internal/adapters/os":     {"internal/ports", "internal/model"},
	"internal/adapters/fake":   {"internal/ports", "internal/model"},
	"cmd/kit":                  {"*"},
	"tools/gate":               {},
	"tools/layers":             {},
}

// fake is the one package of kit's that any test may import, beside what its layer may.
const fake = "internal/adapters/fake"

// What only an adapter may import, and what no test that the gate runs may.
var (
	adaptersOnly = []string{"os/exec", "net", "net/http", "syscall"}
	nightlyOnly  = []string{"os/exec", "net", "net/http", "net/http/httptest"}
)

// Read gives every Go file under fsys, but those under a directory named worktrees, testdata, vendor, or .git.
func Read(fsys fs.FS) ([]File, error) {
	var files []File
	err := fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if slices.Contains([]string{"worktrees", "testdata", "vendor", ".git"}, entry.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		source, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), name, source, parser.ImportsOnly)
		if err != nil {
			return err
		}
		file := File{Path: name, Test: strings.HasSuffix(name, "_test.go"), Nightly: nightly(string(source))}
		for _, spec := range parsed.Imports {
			imported, _ := strconv.Unquote(spec.Path.Value)
			file.Imports = append(file.Imports, imported)
		}
		files = append(files, file)
		return nil
	})
	return files, err
}

// nightly says whether a file's source begins //go:build nightly: a line so, before its package clause.
func nightly(source string) bool {
	for _, line := range strings.Split(source, "\n") {
		if strings.HasPrefix(line, "package ") {
			return false
		}
		if strings.HasPrefix(line, "//go:build") && strings.Contains(line, "nightly") {
			return true
		}
	}
	return false
}

// Check holds the files to the rule, for a module of that path.
func Check(module string, files []File) []Violation {
	var violations []Violation
	for _, file := range files {
		dir := path.Dir(file.Path)
		may, known := Layers[dir]
		if !known {
			why := dir + " is in no layer: add it to the table in tools/layers"
			violations = append(violations, Violation{file.Path, "", why})
			continue
		}
		adapter := strings.HasPrefix(dir, "internal/adapters/")
		outer := adapter || dir == "cmd/kit" || strings.HasPrefix(dir, "tools/")
		for _, imported := range file.Imports {
			why := ""
			kits, of := strings.CutPrefix(imported, module+"/")
			switch {
			case of && (slices.Contains(may, "*") || slices.Contains(may, kits) || file.Test && kits == fake):
			case of:
				why = dir + " may not import " + kits
			case file.Test && !file.Nightly && slices.Contains(nightlyOnly, imported):
				why = "a test the gate runs starts no process and reaches no network"
			case file.Test:
			case !adapter && slices.Contains(adaptersOnly, imported), !outer && imported == "os":
				why = "only an adapter starts a process, opens a connection, or touches a file"
			}
			if why != "" {
				violations = append(violations, Violation{file.Path, imported, why})
			}
		}
	}
	return violations
}
