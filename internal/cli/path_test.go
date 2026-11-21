package cli

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/empowerite/kit/internal/adapters/fake"
	"github.com/empowerite/kit/internal/toolchain"
)

// pinnedDesk gives a desk whose repository at /r pins shellcheck 0.11.0, on linux/amd64.
func pinnedDesk(t *testing.T) toolchain.Desk {
	t.Helper()
	d := toolsDesk(t, &fake.Presence{})
	d.Git = &fake.Git{RootDir: "/r", Common: "/r/.git", Tracks: map[string]bool{}}
	d.Machine = &fake.Machine{OS: "linux", Arch: "amd64", CacheDir: "/cache/kit", Kit: "/usr/bin/kit", Env: []string{"PATH=/usr/bin"}}
	d.Processes = &fake.Processes{}
	d.Files.Write("/usr/bin/kit", []byte("kit"), 0o755)
	if code, _, err := runTools(t, d, "tools", "add", "shellcheck@0.11.0"); code != 0 {
		t.Fatalf("tools add gave %d, %s", code, err)
	}
	return d
}

func TestToolsListsTheTools(t *testing.T) {
	d := pinnedDesk(t)
	code, out, _ := runTools(t, d, "tools")
	if code != 0 || out != "  shellcheck  0.11.0  kit.yml  not fetched here\n" {
		t.Errorf("tools = %d, %q", code, out)
	}
	code, out, _ = runTools(t, d, "tools", "--json")
	var got struct{ Tools []map[string]string }
	if json.Unmarshal([]byte(out), &got); code != 0 || len(got.Tools) != 1 || got.Tools[0]["state"] != "not fetched here" {
		t.Errorf("tools --json = %d, %q", code, out)
	}
	d.Machine.(*fake.Machine).Err = errors.New("no home")
	if code, _, err := runTools(t, d, "tools"); code != 1 || err != "kit: no home\n" {
		t.Errorf("tools with no cache = %d, %q", code, err)
	}
}

func TestToolsFetchFetchesAndSaysWhatItDid(t *testing.T) {
	d := pinnedDesk(t)
	code, out, _ := runTools(t, d, "tools", "fetch")
	if code != 0 || !strings.Contains(out, "  done      .kit   place .kit/bin/shellcheck, a link to ../kit\n") {
		t.Errorf("tools fetch = %d, %q", code, out)
	}
	code, out, _ = runTools(t, d, "tools", "fetch", "--json")
	if code != 0 || !strings.Contains(out, `"schema": "kit/plan"`) {
		t.Errorf("tools fetch --json = %d, %q", code, out)
	}
	d.Files.(*fake.Files).Fail["write /r/.kit/env"] = errors.New("disk full")
	d.Files.Remove("/r/.kit/env")
	if code, out, _ := runTools(t, d, "tools", "fetch"); code != 1 || !strings.Contains(out, "kit: disk full") {
		t.Errorf("tools fetch that fails = %d, %q", code, out)
	}
	d.Machine.(*fake.Machine).Err = errors.New("no home")
	if code, _, err := runTools(t, d, "tools", "fetch"); code != 1 || err != "kit: no home\n" {
		t.Errorf("tools fetch with no cache = %d, %q", code, err)
	}
}

func TestToolsPathPrintsTheProgram(t *testing.T) {
	d := pinnedDesk(t)
	code, out, _ := runTools(t, d, "tools", "path", "shellcheck")
	if code != 0 || !strings.HasPrefix(out, "/cache/kit/tools/") || !strings.HasSuffix(out, "/shellcheck\n") {
		t.Errorf("tools path = %d, %q", code, out)
	}
	if code, out, _ := runTools(t, d, "tools", "path", "shellcheck", "--json"); code != 0 || !strings.Contains(out, `"path": "/cache/kit/tools/`) {
		t.Errorf("tools path --json = %d, %q", code, out)
	}
	if code, _, err := runTools(t, d, "tools", "path", "lychee"); code != 1 || !strings.Contains(err, "holds no lychee") {
		t.Errorf("tools path lychee = %d, %q", code, err)
	}
}

func TestExecRunsTheCommandInKitsPlace(t *testing.T) {
	d := pinnedDesk(t)
	if code, _, err := runTools(t, d, "exec", "--", "shellcheck", "-x", "a.sh"); code != 0 || err != "" {
		t.Errorf("exec = %d, %q", code, err)
	}
	got := d.Processes.(*fake.Processes).Replaced
	if len(got) != 1 || got[0].Path != "shellcheck" || strings.Join(got[0].Args, " ") != "-x a.sh" {
		t.Errorf("exec ran %+v", got)
	}
	d.Processes.(*fake.Processes).ReplaceErr = errors.New("there is no shellcheck on the PATH")
	if code, _, err := runTools(t, d, "exec", "--", "shellcheck"); code != 127 || err != "kit: there is no shellcheck on the PATH\n" {
		t.Errorf("exec of what cannot run = %d, %q", code, err)
	}
}

func TestEnvPrintsForEachShell(t *testing.T) {
	d := pinnedDesk(t)
	if code, out, _ := runTools(t, d, "env"); code != 0 || out != "export PATH='/r/.kit/bin':\"$PATH\"\n" {
		t.Errorf("env = %d, %q", code, out)
	}
	if code, out, _ := runTools(t, d, "env", "--shell=direnv"); code != 0 || !strings.HasPrefix(out, "PATH_add .kit/bin\n") {
		t.Errorf("env --shell=direnv = %d, %q", code, out)
	}
	if code, out, _ := runTools(t, d, "env", "--json"); code != 0 || !strings.Contains(out, `"path_add": [`) {
		t.Errorf("env --json = %d, %q", code, out)
	}
	if code, _, err := runTools(t, d, "env", "--shell=fish"); code != 2 || !strings.Contains(err, "bash, zsh, and direnv") {
		t.Errorf("env --shell=fish = %d, %q", code, err)
	}
	if _, err := d.Files.ReadLink("/r/.kit/bin/shellcheck"); err != nil {
		t.Errorf("env did not fetch first: %v", err)
	}
	d.Machine.(*fake.Machine).Err = errors.New("no home")
	if code, _, err := runTools(t, d, "env"); code != 1 || err != "kit: no home\n" {
		t.Errorf("env with no cache = %d, %q", code, err)
	}
}

func TestTheCommandsOfTheToolsSayWhereThereIsNoRepository(t *testing.T) {
	d := pinnedDesk(t)
	d.Git = &fake.Git{Err: errors.New("git: / is in no working tree of a clone")}
	for _, args := range [][]string{{"tools"}, {"tools", "fetch"}, {"tools", "path", "x"}, {"exec", "--", "x"}, {"env"}} {
		if code, _, err := runTools(t, d, args...); code != 1 || !strings.Contains(err, "no working tree") {
			t.Errorf("%q = %d, %q", args, code, err)
		}
	}
}
