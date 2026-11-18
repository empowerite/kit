package layers

import (
	"errors"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

const module = "example.com/kit"

// tree gives a tree of Go files, each by its path and its source.
func tree(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for path, source := range files {
		fsys[path] = &fstest.MapFile{Data: []byte(source)}
	}
	return fsys
}

func TestReadGivesEachFilesImports(t *testing.T) {
	got, err := Read(tree(map[string]string{
		"internal/plan/plan.go": "package plan\n\nimport (\n\t\"fmt\"\n\n\t\"example.com/kit/internal/model\"\n)\n",
		"README.md":             "# not Go\n",
	}))
	want := []File{{Path: "internal/plan/plan.go", Imports: []string{"fmt", "example.com/kit/internal/model"}}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Read = %+v, %v; want %+v", got, err, want)
	}
}

func TestReadSkipsWorktreesTestdataVendorAndGit(t *testing.T) {
	got, err := Read(tree(map[string]string{
		"cmd/kit/main.go":                 "package main\n",
		"worktrees/7-x/cmd/kit/main.go":   "package main\n",
		"internal/doc/testdata/case.go":   "package broken (\n",
		"vendor/example.com/dep/dep.go":   "package dep\n",
		".git/hooks/not-really-a-file.go": "package hooks\n",
	}))
	want := []File{{Path: "cmd/kit/main.go"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Read = %+v, %v; want %+v", got, err, want)
	}
}

func TestReadMarksATestAndANightlyFile(t *testing.T) {
	got, err := Read(tree(map[string]string{
		"a/a_test.go":       "package a\n",
		"a/night_test.go":   "//go:build nightly\n\npackage a\n",
		"a/other_test.go":   "//go:build linux\n\npackage a\n",
		"a/package_test.go": "// A comment that says //go:build nightly lower down does not count.\npackage a\n\n//go:build nightly\n",
		"a/shared_test.go":  "/* a comment that shares the clause's line */ package a\n",
	}))
	want := []File{
		{Path: "a/a_test.go", Test: true},
		{Path: "a/night_test.go", Test: true, Nightly: true},
		{Path: "a/other_test.go", Test: true},
		{Path: "a/package_test.go", Test: true},
		{Path: "a/shared_test.go", Test: true},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Read = %+v, %v; want %+v", got, err, want)
	}
}

func TestReadFailsOnAFileThatDoesNotParse(t *testing.T) {
	_, err := Read(tree(map[string]string{"a/a.go": "package a\n\nimport (\n"}))
	if err == nil || !strings.HasPrefix(err.Error(), "a/a.go:") {
		t.Errorf("Read gave the error %v; want one that names a/a.go", err)
	}
}

// unreadable is a tree in which one path cannot be opened. It is a tree's Open alone, so that every read goes
// through it.
type unreadable struct {
	tree fs.FS
	path string
}

func (u unreadable) Open(name string) (fs.File, error) {
	if name == u.path {
		return nil, errors.New("cannot open " + name)
	}
	return u.tree.Open(name)
}

func TestReadFailsOnWhatItCannotOpen(t *testing.T) {
	for _, path := range []string{"a", "a/a.go"} {
		_, err := Read(unreadable{tree(map[string]string{"a/a.go": "package a\n"}), path})
		if err == nil || !strings.Contains(err.Error(), "cannot open "+path) {
			t.Errorf("Read gave the error %v for a tree whose %s cannot be opened; want that error", err, path)
		}
	}
}

func TestCheck(t *testing.T) {
	process := "only an adapter starts a process, opens a connection, or touches a file"
	network := "a test the gate runs starts no process and reaches no network"
	for _, c := range []struct {
		name string
		file File
		want []Violation
	}{
		{"a layer imports one it may",
			File{Path: "internal/plan/plan.go", Imports: []string{"fmt", module + "/internal/model", module + "/internal/ports"}}, nil},
		{"a layer imports one it may not",
			File{Path: "internal/model/model.go", Imports: []string{module + "/internal/data"}},
			[]Violation{{"internal/model/model.go", module + "/internal/data", "internal/model may not import internal/data"}}},
		{"a package is in no layer",
			File{Path: "internal/extra/extra.go", Imports: []string{"fmt"}},
			[]Violation{{"internal/extra/extra.go", "", "internal/extra is in no layer: add it to the table in tools/layers"}}},
		{"a layer starts a process",
			File{Path: "internal/plan/plan.go", Imports: []string{"os/exec"}},
			[]Violation{{"internal/plan/plan.go", "os/exec", process}}},
		{"a layer touches a file",
			File{Path: "internal/doc/doc.go", Imports: []string{"os"}},
			[]Violation{{"internal/doc/doc.go", "os", process}}},
		{"an adapter starts a process and opens a connection",
			File{Path: "internal/adapters/git/git.go", Imports: []string{"os", "os/exec", "net/http", "syscall"}}, nil},
		{"the command touches a file",
			File{Path: "cmd/kit/main.go", Imports: []string{"os", module + "/internal/cli", module + "/internal/adapters/github"}}, nil},
		{"the command starts a process",
			File{Path: "cmd/kit/main.go", Imports: []string{"os/exec"}},
			[]Violation{{"cmd/kit/main.go", "os/exec", process}}},
		{"a tool touches a file",
			File{Path: "tools/gate/main.go", Imports: []string{"os"}}, nil},
		{"a tool opens a connection",
			File{Path: "tools/gate/main.go", Imports: []string{"net"}},
			[]Violation{{"tools/gate/main.go", "net", process}}},
		{"a tool imports a package of kit's",
			File{Path: "tools/gate/main.go", Imports: []string{module + "/internal/model"}},
			[]Violation{{"tools/gate/main.go", module + "/internal/model", "tools/gate may not import internal/model"}}},
		{"a layer imports the fakes",
			File{Path: "internal/plan/plan.go", Imports: []string{module + "/internal/adapters/fake"}},
			[]Violation{{"internal/plan/plan.go", module + "/internal/adapters/fake", "internal/plan may not import internal/adapters/fake"}}},
		{"a layer's test imports the fakes, and a file",
			File{Path: "internal/plan/plan_test.go", Test: true, Imports: []string{"os", module + "/internal/adapters/fake"}}, nil},
		{"a layer's test imports a layer it may not",
			File{Path: "internal/model/model_test.go", Test: true, Imports: []string{module + "/internal/data"}},
			[]Violation{{"internal/model/model_test.go", module + "/internal/data", "internal/model may not import internal/data"}}},
		{"a test starts a process",
			File{Path: "internal/adapters/git/git_test.go", Test: true, Imports: []string{"os/exec"}},
			[]Violation{{"internal/adapters/git/git_test.go", "os/exec", network}}},
		{"a test reaches the network",
			File{Path: "internal/plan/plan_test.go", Test: true, Imports: []string{"net/http/httptest"}},
			[]Violation{{"internal/plan/plan_test.go", "net/http/httptest", network}}},
		{"a nightly test starts a process and reaches the network",
			File{Path: "internal/adapters/git/git_test.go", Test: true, Nightly: true, Imports: []string{"os/exec", "net/http"}}, nil},
	} {
		if got := Check(module, []File{c.file}); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Check = %q; want %q", c.name, got, c.want)
		}
	}
}

func TestTheTreeKeepsTheRule(t *testing.T) {
	files, err := Read(os.DirFS("../.."))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("Read found no Go file in the tree")
	}
	for _, v := range Check("github.com/empowerite/kit", files) {
		if v.Import == "" {
			t.Errorf("%s: %s", v.File, v.Why)
			continue
		}
		t.Errorf("%s imports %q: %s", v.File, v.Import, v.Why)
	}
}
