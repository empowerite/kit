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

// tree gives a tree of files, each by its path and its text.
func tree(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for path, text := range files {
		fsys[path] = &fstest.MapFile{Data: []byte(text)}
	}
	return fsys
}

// read gives the files Read finds in a tree, and fails the test on an error.
func read(t *testing.T, files map[string]string) []File {
	t.Helper()
	got, err := Read(tree(files))
	if err != nil {
		t.Fatalf("Read gave the error %v", err)
	}
	return got
}

func TestReadGivesEachFilesImports(t *testing.T) {
	got := read(t, map[string]string{
		"internal/plan/plan.go": "package plan\n\nimport (\n\t\"fmt\"\n\n\t\"example.com/kit/internal/model\"\n)\n",
		"README.md":             "# not Go\n",
		"go.mod":                "module example.com/kit\n",
	})
	want := []File{{Path: "internal/plan/plan.go", Imports: []string{"fmt", "example.com/kit/internal/model"}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Read = %+v; want %+v", got, want)
	}
}

func TestReadSkipsWorktreesVendorAndGitAtTheRootAndTestdataAnywhere(t *testing.T) {
	got := read(t, map[string]string{
		"cmd/kit/main.go":                 "package main\n",
		"worktrees/7-x/cmd/kit/main.go":   "package main\n",
		"worktrees/7-x/go.mod":            "module example.com/kit\n",
		"vendor/example.com/dep/dep.go":   "package dep\n",
		".git/hooks/not-really-a-file.go": "package hooks\n",
		"internal/doc/testdata/case.go":   "package broken (\n",
		"internal/worktrees/worktrees.go": "package worktrees\n",
		"internal/vendor/vendor.go":       "package vendor\n",
	})
	want := []File{{Path: "cmd/kit/main.go"}, {Path: "internal/vendor/vendor.go"}, {Path: "internal/worktrees/worktrees.go"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Read = %+v; want %+v", got, want)
	}
}

func TestReadMarksATestANightlyTestAndAFileOfOnlySomeBuilds(t *testing.T) {
	got := read(t, map[string]string{
		"a/a_test.go":       "package a\n",
		"a/night_test.go":   "//go:build nightly\n\npackage a\n",
		"a/day_test.go":     "//go:build !nightly\n\npackage a\n",
		"a/night.go":        "//go:build nightly\n\npackage a\n",
		"a/linux_only.go":   "// A comment first.\n\n//go:build linux && amd64\n\npackage a\n",
		"a/a_windows.go":    "package a\n",
		"a/a_linux_test.go": "package a\n",
		"a/a_arm64.go":      "package a\n",
		"a/a_plain_name.go": "package a\n",
	})
	want := []File{
		{Path: "a/a_arm64.go", Constraint: "its name ends _arm64"},
		{Path: "a/a_linux_test.go", Test: true, Constraint: "its name ends _linux"},
		{Path: "a/a_plain_name.go"},
		{Path: "a/a_test.go", Test: true},
		{Path: "a/a_windows.go", Constraint: "its name ends _windows"},
		{Path: "a/day_test.go", Test: true, Constraint: "//go:build !nightly"},
		{Path: "a/linux_only.go", Constraint: "//go:build linux && amd64"},
		{Path: "a/night.go", Constraint: "//go:build nightly"},
		{Path: "a/night_test.go", Test: true, Nightly: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Read:\n got  %+v\n want %+v", got, want)
	}
}

func TestAConstraintUnderThePackageClauseDoesNotCount(t *testing.T) {
	got := read(t, map[string]string{
		"a/a_test.go": "/* a comment that shares the clause's line */ package a\n\n//go:build nightly\n",
	})
	if len(got) != 1 || got[0].Nightly || got[0].Constraint != "" {
		t.Errorf("Read = %+v; want one file, in every build and not nightly", got)
	}
}

func TestReadMarksAFileGofmtWouldChange(t *testing.T) {
	got := read(t, map[string]string{
		"a/a.go": "package a\n\nfunc F() int { return 1 }\n",
		"a/b.go": "package a\n\nfunc G()   int {\nreturn 1 }\n",
	})
	want := []File{{Path: "a/a.go"}, {Path: "a/b.go", Unformatted: true}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Read = %+v; want %+v", got, want)
	}
}

func TestReadFailsOnAFileThatDoesNotParse(t *testing.T) {
	_, err := Read(tree(map[string]string{"a/a.go": "package a\n\nimport (\n"}))
	if err == nil || !strings.HasPrefix(err.Error(), "a/a.go:") {
		t.Errorf("Read gave the error %v; want one that names a/a.go", err)
	}
}

func TestReadFailsOnAGoModBelowTheRoot(t *testing.T) {
	// go test ./... leaves out a directory that is a module of its own, so the gate would never judge it.
	_, err := Read(tree(map[string]string{"go.mod": "module example.com/kit\n", "cmd/kit/go.mod": "module example.com/kit/cmd/kit\n"}))
	if err == nil || err.Error() != "cmd/kit/go.mod: a go.mod below the root: kit is one module" {
		t.Errorf("Read gave the error %v; want one that names cmd/kit/go.mod", err)
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
	// refused gives a case in which one import of a file is refused.
	refused := func(name, path string, test bool, imported, why string) struct {
		name string
		file File
		want []Violation
	} {
		return struct {
			name string
			file File
			want []Violation
		}{name, File{Path: path, Test: test, Imports: []string{imported}}, []Violation{{path, imported, why}}}
	}
	// passes gives a case in which every import of a file is taken.
	passes := func(name string, file File) struct {
		name string
		file File
		want []Violation
	} {
		return struct {
			name string
			file File
			want []Violation
		}{name, file, nil}
	}
	cases := []struct {
		name string
		file File
		want []Violation
	}{
		passes("a layer imports those it may, and what is pure",
			File{Path: "internal/plan/plan.go", Imports: []string{"fmt", "io/fs", "net/url", "net/netip", "path", "embed",
				module + "/internal/model", module + "/internal/ports"}}),
		refused("a layer imports one it may not", "internal/model/model.go", false,
			module+"/internal/data", "internal/model may not import internal/data"),
		{"a package is in no layer",
			File{Path: "internal/extra/extra.go", Imports: []string{"fmt"}},
			[]Violation{{"internal/extra/extra.go", "", "internal/extra is in no layer: add it to the table in tools/layers"}}},
		{"gofmt would change a file",
			File{Path: "internal/plan/plan.go", Unformatted: true},
			[]Violation{{"internal/plan/plan.go", "", "gofmt would change it: run gofmt -w internal/plan/plan.go"}}},
		{"a file is in only some builds",
			File{Path: "internal/plan/plan_windows.go", Constraint: "its name ends _windows"},
			[]Violation{{"internal/plan/plan_windows.go", "",
				"only some builds hold this file (its name ends _windows), so the gate cannot say its statements are run"}}},
		passes("an adapter starts a process and opens a connection",
			File{Path: "internal/adapters/git/git.go", Imports: []string{"os", "os/exec", "net/http", "syscall", "path/filepath"}}),
		passes("the command touches a file",
			File{Path: "cmd/kit/main.go", Imports: []string{"os", "path/filepath", module + "/internal/cli", module + "/internal/adapters/github"}}),
		passes("a tool touches a file", File{Path: "tools/gate/main.go", Imports: []string{"os", "io/ioutil"}}),
		refused("the command starts a process", "cmd/kit/main.go", false, "os/exec", process),
		refused("a tool opens a connection", "tools/gate/main.go", false, "net", process),
		refused("a tool imports a package of kit's", "tools/gate/main.go", false,
			module+"/internal/model", "tools/gate may not import internal/model"),
		refused("a layer imports the fakes", "internal/plan/plan.go", false,
			module+"/internal/adapters/fake", "internal/plan may not import internal/adapters/fake"),
		passes("a layer's test imports the fakes, its own package, and a file",
			File{Path: "internal/plan/plan_test.go", Test: true, Imports: []string{"os", "path/filepath",
				module + "/internal/adapters/fake", module + "/internal/plan"}}),
		refused("a layer's test imports a layer it may not", "internal/model/model_test.go", true,
			module+"/internal/data", "internal/model may not import internal/data"),
		passes("a nightly test starts a process and reaches the network",
			File{Path: "internal/adapters/git/git_test.go", Test: true, Nightly: true, Imports: []string{"os/exec", "net/http", "syscall"}}),
	}
	for _, imported := range []string{"os", "io/ioutil", "path/filepath"} {
		cases = append(cases,
			refused("a layer touches a file", "internal/doc/doc.go", false, imported, process),
			refused("the fakes touch a file", "internal/adapters/fake/fake.go", false, imported, process))
	}
	for _, imported := range []string{"os/exec", "os/signal", "os/user", "net", "net/http", "net/rpc", "net/smtp",
		"syscall", "plugin", "C", "crypto/tls", "log/syslog"} {
		cases = append(cases,
			refused("a layer starts a process or opens a connection", "internal/plan/plan.go", false, imported, process),
			refused("the fakes start a process or open a connection", "internal/adapters/fake/fake.go", false, imported, process),
			refused("a test starts a process or reaches the network", "internal/adapters/git/git_test.go", true, imported, network))
	}
	cases = append(cases,
		refused("a test reaches the network", "internal/plan/plan_test.go", true, "net/http/httptest", network),
		refused("a test reaches the network", "internal/plan/plan_test.go", true, "net/http/cgi", network))
	for _, c := range cases {
		if got := Check(module, []File{c.file}); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Check(%+v) = %q; want %q", c.name, c.file, got, c.want)
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
