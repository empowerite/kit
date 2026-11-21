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
	code, out, _ := runTools(t, d, "tools", "fetch", "--trust=checkout")
	if code != 0 || !strings.Contains(out, "  done      .kit   place .kit/bin/shellcheck, a link to ../kit\n") {
		t.Errorf("tools fetch = %d, %q", code, out)
	}
	code, out, _ = runTools(t, d, "tools", "fetch", "--json", "--trust=checkout")
	if code != 0 || !strings.Contains(out, `"schema": "kit/plan"`) {
		t.Errorf("tools fetch --json = %d, %q", code, out)
	}
	d.Files.(*fake.Files).Fail["write /r/.kit/env"] = errors.New("disk full")
	d.Files.Remove("/r/.kit/env")
	if code, out, _ := runTools(t, d, "tools", "fetch", "--trust=checkout"); code != 1 || !strings.Contains(out, "kit: disk full") {
		t.Errorf("tools fetch that fails = %d, %q", code, out)
	}
	d.Machine.(*fake.Machine).Err = errors.New("no home")
	if code, _, err := runTools(t, d, "tools", "fetch", "--trust=checkout"); code != 1 || err != "kit: no home\n" {
		t.Errorf("tools fetch with no cache = %d, %q", code, err)
	}
}

func TestToolsPathPrintsTheProgram(t *testing.T) {
	d := pinnedDesk(t)
	code, out, _ := runTools(t, d, "tools", "path", "shellcheck", "--trust=checkout")
	if code != 0 || !strings.HasPrefix(out, "/cache/kit/tools/") || !strings.HasSuffix(out, "/shellcheck\n") {
		t.Errorf("tools path = %d, %q", code, out)
	}
	if code, out, _ := runTools(t, d, "tools", "path", "shellcheck", "--json", "--trust=checkout"); code != 0 || !strings.Contains(out, `"path": "/cache/kit/tools/`) {
		t.Errorf("tools path --json = %d, %q", code, out)
	}
	if code, _, err := runTools(t, d, "tools", "path", "lychee", "--trust=checkout"); code != 1 || !strings.Contains(err, "holds no lychee") {
		t.Errorf("tools path lychee = %d, %q", code, err)
	}
}

func TestExecRunsTheCommandInKitsPlace(t *testing.T) {
	d := pinnedDesk(t)
	if code, _, err := runTools(t, d, "exec", "--trust=checkout", "--", "shellcheck", "-x", "a.sh"); code != 0 || err != "" {
		t.Errorf("exec = %d, %q", code, err)
	}
	got := d.Processes.(*fake.Processes).Replaced
	if len(got) != 1 || got[0].Path != "shellcheck" || strings.Join(got[0].Args, " ") != "-x a.sh" {
		t.Errorf("exec ran %+v", got)
	}
	d.Processes.(*fake.Processes).ReplaceErr = errors.New("there is no shellcheck on the PATH")
	if code, _, err := runTools(t, d, "exec", "--trust=checkout", "--", "shellcheck"); code != 127 || err != "kit: there is no shellcheck on the PATH\n" {
		t.Errorf("exec of what cannot run = %d, %q", code, err)
	}
}

func TestEnvPrintsForEachShell(t *testing.T) {
	d := pinnedDesk(t)
	if code, out, _ := runTools(t, d, "env", "--trust=checkout"); code != 0 || out != "export PATH='/r/.kit/bin':\"$PATH\"\n" {
		t.Errorf("env = %d, %q", code, out)
	}
	if code, out, _ := runTools(t, d, "env", "--shell=direnv", "--trust=checkout"); code != 0 || !strings.HasPrefix(out, "PATH_add .kit/bin\n") {
		t.Errorf("env --shell=direnv = %d, %q", code, out)
	}
	if code, out, _ := runTools(t, d, "env", "--json", "--trust=checkout"); code != 0 || !strings.Contains(out, `"path_add": [`) {
		t.Errorf("env --json = %d, %q", code, out)
	}
	if code, _, err := runTools(t, d, "env", "--shell=fish", "--trust=checkout"); code != 2 || !strings.Contains(err, "bash, zsh, and direnv") {
		t.Errorf("env --shell=fish = %d, %q", code, err)
	}
	if _, err := d.Files.ReadLink("/r/.kit/bin/shellcheck"); err != nil {
		t.Errorf("env did not fetch first: %v", err)
	}
	d.Machine.(*fake.Machine).Err = errors.New("no home")
	if code, _, err := runTools(t, d, "env", "--trust=checkout"); code != 1 || err != "kit: no home\n" {
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

func TestTheCommandsThatFetchOrRunAskForTrust(t *testing.T) {
	d := pinnedDesk(t)
	d.Machine.(*fake.Machine).StateDir = "/state/kit"
	for _, args := range [][]string{{"tools", "fetch"}, {"tools", "path", "shellcheck"}, {"exec", "--", "x"}, {"env"}} {
		if code, _, err := runTools(t, d, args...); code == 0 || !strings.Contains(err, "until you trust it") {
			t.Errorf("%q = %d, %q; want a refusal that asks for trust", args, code, err)
		}
	}
	if code, _, err := runTools(t, d, "tools", "fetch", "--trust=everything"); code != 1 || err != "kit: --trust takes checkout, and nothing else\n" {
		t.Errorf("--trust=everything = %d, %q", code, err)
	}
}

func TestTrustShowsWhatItCoversAndRecordsIt(t *testing.T) {
	d := pinnedDesk(t)
	d.Machine.(*fake.Machine).StateDir = "/state/kit"
	d.Processes.(*fake.Processes).Prints = map[string]string{"status --json": `{"state": {"foundRC": {"allowed": 1}}}`}
	d.Files.Write("/r/.envrc", []byte("eval \"$(kit env --shell=direnv)\"\n"), 0o644)
	code, out, err := runTools(t, d, "trust")
	for _, want := range []string{"What changed since you last trusted this clone:", "  shellcheck 0.11.0 is new, from:",
		"direnv has not allowed this .envrc: kit trust covers it", "  trust   trust what this clone holds: .envrc, kit.lock, kit.yml",
		"  done      direnv  direnv allow /r/.envrc"} {
		if !strings.Contains(out, want) {
			t.Errorf("trust printed %q; want %q (%d, %q)", out, want, code, err)
		}
	}
	presence := d.Presence.(*fake.Presence)
	asked := len(presence.Asked)
	if code, out, _ := runTools(t, d, "trust"); code != 0 || out != "kit trusts this clone already, as it holds now: there is nothing to approve.\n" ||
		len(presence.Asked) != asked {
		t.Errorf("trust again = %d, %q, and asked %d times more; want no question", code, out, len(presence.Asked)-asked)
	}
	if code, out, _ := runTools(t, d, "trust", "--json"); code != 0 || !strings.Contains(out, `"trusted": true`) {
		t.Errorf("trust --json again = %d, %q", code, out)
	}
	if code, _, _ := runTools(t, d, "tools", "fetch"); code != 0 {
		t.Errorf("once trusted, tools fetch = %d", code)
	}
	d.Files.Write("/r/kit.yml", []byte("tools:\n  shellcheck: 0.11.0\n# changed\n"), 0o644)
	if code, out, err := runTools(t, d, "trust", "--json"); code != 0 || !strings.Contains(err, "kit.yml changed") || !strings.Contains(out, `"result"`) {
		t.Errorf("trust --json of a change = %d, %q, %q; want what changed on stderr first", code, out, err)
	}
	if code, out, _ := runTools(t, d, "trust"); code != 0 || !strings.HasPrefix(out, "kit trusts this clone already") {
		t.Errorf("trust once more = %d, %q", code, out)
	}
	// Contents trusted before are trusted still: several sets are held at once.
	d.Files.Write("/r/kit.yml", []byte("tools:\n  shellcheck: 0.11.0\n"), 0o644)
	if code, out, _ := runTools(t, d, "trust"); code != 0 || !strings.HasPrefix(out, "kit trusts this clone already") {
		t.Errorf("trust of contents trusted before = %d, %q", code, out)
	}
	d.Machine.(*fake.Machine).Err = errors.New("no home")
	if code, _, err := runTools(t, d, "trust"); code != 1 || err != "kit: no home\n" {
		t.Errorf("trust with no state = %d, %q", code, err)
	}
}

func TestToolsCheckSaysEachFindingAndItsFix(t *testing.T) {
	d := pinnedDesk(t)
	if code, out, _ := runTools(t, d, "tools", "check"); code != 0 || !strings.HasPrefix(out, "kit tools check: ok") {
		t.Errorf("tools check = %d, %q", code, out)
	}
	d.Files.Write("/r/kit.lock", []byte("{\"tools\": {}}\n"), 0o644)
	code, out, _ := runTools(t, d, "tools", "check")
	if code != 1 || out != "kit.lock: kit.lock holds no shellcheck\n  why  kit.yml pins it, and no digest says which bytes to run\n  fix  kit tools lock\n" {
		t.Errorf("tools check = %d, %q", code, out)
	}
	code, out, _ = runTools(t, d, "tools", "check", "--json")
	if code != 1 || !strings.Contains(out, `"fix": "kit tools lock"`) {
		t.Errorf("tools check --json = %d, %q", code, out)
	}
	d.Git = &fake.Git{Err: errors.New("not a clone")}
	if code, _, err := runTools(t, d, "tools", "check"); code != 1 || err != "kit: not a clone\n" {
		t.Errorf("tools check outside a clone = %d, %q", code, err)
	}
}

func TestTrustOfACloneThatHoldsNothingKitWouldRun(t *testing.T) {
	d := toolsDesk(t, &fake.Presence{})
	d.Machine = &fake.Machine{OS: "linux", Arch: "amd64", StateDir: "/state/kit"}
	d.Processes = &fake.Processes{}
	if code, out, _ := runTools(t, d, "trust", "--plan"); code != 0 || !strings.HasPrefix(out, "This clone holds nothing kit would fetch or run.\n") {
		t.Errorf("trust = %d, %q", code, out)
	}
}
