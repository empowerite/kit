package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/empowerite/kit/internal/adapters/fake"
	"github.com/empowerite/kit/internal/toolchain"
)

// forge gives releases and downloads that hold shellcheck 0.11.0, as kit's own list names its files.
func forge(t *testing.T) (*fake.Releases, fake.Downloads) {
	t.Helper()
	list, err := toolchain.List()
	if err != nil {
		t.Fatal(err)
	}
	e := list["shellcheck"]
	r := &fake.Releases{Tags: map[string]string{e.GitHub: "v0.11.0"}}
	d := fake.Downloads{}
	for _, p := range toolchain.Platforms {
		name := strings.ReplaceAll(e.Files[p], "{version}", "0.11.0")
		var b bytes.Buffer
		z := gzip.NewWriter(&b)
		w := tar.NewWriter(z)
		w.WriteHeader(&tar.Header{Name: "shellcheck-v0.11.0/shellcheck", Mode: 0o755, Size: 2})
		w.Write([]byte("sc"))
		w.Close()
		z.Close()
		url := "https://github.com/" + e.GitHub + "/releases/download/v0.11.0/" + name
		sum := sha256.Sum256(b.Bytes())
		r.Put(e.GitHub, "v0.11.0", name, url, "sha256:"+hex.EncodeToString(sum[:]))
		d[url] = b.Bytes()
	}
	return r, d
}

func toolsDesk(t *testing.T, presence *fake.Presence) toolchain.Desk {
	r, d := forge(t)
	return toolchain.Desk{Dir: "/r", Files: fake.NewFiles(), Git: &fake.Git{RootDir: "/r", Common: "/r/.git"},
		Releases: r, Downloads: d, Presence: presence}
}

func runTools(t *testing.T, d toolchain.Desk, args ...string) (int, string, string) {
	t.Helper()
	all, err := Commands()
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), all, Tools(func() (toolchain.Desk, error) { return d, nil }), args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestToolsAddShowsThePlanAndAppliesItOnceApproved(t *testing.T) {
	presence := &fake.Presence{}
	d := toolsDesk(t, presence)
	code, out, err := runTools(t, d, "tools", "add", "shellcheck")
	if code != 0 || err != "" || !strings.HasPrefix(out, "Plan  kit tools add shellcheck@0.11.0, ") ||
		!strings.Contains(out, "  done      files  write kit.lock\n") {
		t.Errorf("tools add = %d, %q, %q; want the plan, then each step done", code, out, err)
	}
	if len(presence.Asked) != 1 {
		t.Errorf("tools add asked for presence %d times; want once", len(presence.Asked))
	}
	if yml, _ := d.Files.Read("/r/kit.yml"); string(yml) != "tools:\n  shellcheck: 0.11.0\n" {
		t.Errorf("kit.yml holds %q", yml)
	}
	code, out, _ = runTools(t, d, "tools", "add", "shellcheck@0.11.0")
	if code != 0 || out != "kit: nothing to change: kit.yml and kit.lock hold it already\n" {
		t.Errorf("tools add of what is pinned = %d, %q", code, out)
	}
}

func TestToolsAddWithPlanChangesNothing(t *testing.T) {
	presence := &fake.Presence{}
	d := toolsDesk(t, presence)
	code, out, _ := runTools(t, d, "tools", "add", "shellcheck", "--plan")
	if code != 0 || !strings.Contains(out, "  files  write kit.yml\n") || len(presence.Asked) != 0 {
		t.Errorf("tools add --plan = %d, %q, asked %d times; want the plan, and no question", code, out, len(presence.Asked))
	}
	if _, err := d.Files.Read("/r/kit.yml"); err == nil {
		t.Error("tools add --plan wrote kit.yml")
	}
	code, out, _ = runTools(t, d, "tools", "add", "shellcheck", "--plan", "--json")
	var got struct {
		Schema string
		Plan   struct{ Act string }
	}
	if json.Unmarshal([]byte(out), &got); code != 0 || got.Schema != "kit/plan" || got.Plan.Act != "tools add shellcheck@0.11.0" {
		t.Errorf("tools add --plan --json = %d, %q", code, out)
	}
}

func TestToolsAddRefusesWithNoPresence(t *testing.T) {
	d := toolsDesk(t, &fake.Presence{Err: errors.New("presence: nobody is here to approve")})
	code, out, err := runTools(t, d, "tools", "add", "shellcheck")
	if code != 1 || !strings.HasPrefix(out, "Plan  ") || err != "kit: presence: nobody is here to approve\n" {
		t.Errorf("tools add = %d, %q, %q; want the plan, then the refusal", code, out, err)
	}
	if _, err := d.Files.Read("/r/kit.yml"); err == nil {
		t.Error("tools add wrote kit.yml with no approval")
	}
}

func TestToolsAddSaysAResultAsJSON(t *testing.T) {
	d := toolsDesk(t, &fake.Presence{})
	d.Files.(*fake.Files).Fail["write /r/kit.lock"] = errors.New("disk full")
	code, out, _ := runTools(t, d, "tools", "add", "shellcheck", "--json")
	var got struct {
		Result struct {
			Done    []map[string]any
			NotDone []map[string]any `json:"not_done"`
			Error   string
		}
	}
	json.Unmarshal([]byte(out), &got)
	if code != 1 || got.Result.Error != "disk full" || len(got.Result.NotDone) != 1 {
		t.Errorf("tools add --json = %d, %q; want the result, with kit.lock not done", code, out)
	}
	if code, out, _ := runTools(t, toolsDesk(t, &fake.Presence{}), "tools", "add", "shellcheck", "--json"); code != 0 ||
		!strings.Contains(out, `"error": ""`) {
		t.Errorf("tools add --json = %d, %q; want the result with no error", code, out)
	}
}

func TestToolsLockLocksThePinsAgain(t *testing.T) {
	d := toolsDesk(t, &fake.Presence{})
	runTools(t, d, "tools", "add", "shellcheck")
	d.Files.Write("/r/kit.lock", []byte("{\"tools\": {}}\n"), 0o644)
	code, out, err := runTools(t, d, "tools", "lock")
	if code != 0 || !strings.Contains(out, "  done      files  write kit.lock\n") {
		t.Errorf("tools lock = %d, %q, %q", code, out, err)
	}
}

func TestToolsSayWhatStandsInTheirWay(t *testing.T) {
	d := toolsDesk(t, &fake.Presence{})
	for _, c := range []struct {
		desk func() (toolchain.Desk, error)
		args []string
		want string
	}{
		{func() (toolchain.Desk, error) { return d, errors.New("no machine") }, []string{"tools", "lock"}, "kit: no machine\n"},
		{func() (toolchain.Desk, error) {
			e := d
			e.Git = &fake.Git{Err: errors.New("git: /r is in no working tree of a clone")}
			return e, nil
		}, []string{"tools", "add", "x"}, "kit: git: /r is in no working tree of a clone\n"},
		{func() (toolchain.Desk, error) { return d, nil }, []string{"tools", "add", "mystery"}, "kit: kit's list holds no mystery"},
		{func() (toolchain.Desk, error) {
			e := d
			e.Files = fake.NewFiles()
			e.Files.Write("/r/kit.yml", []byte("tools:\n  mystery: 1.0.0\n"), 0o644)
			return e, nil
		}, []string{"tools", "lock"}, "kit: kit's list holds no mystery"},
		{func() (toolchain.Desk, error) {
			e := d
			e.Files = fake.NewFiles()
			e.Files.(*fake.Files).Fail["read /r/kit.yml"] = errors.New("broken")
			e.Files.Write("/r/kit.yml", nil, 0o644)
			return e, nil
		}, []string{"tools", "add", "shellcheck", "--plan"}, "kit: broken"},
	} {
		all, _ := Commands()
		var stdout, stderr bytes.Buffer
		code := Run(context.Background(), all, Tools(c.desk), c.args, &stdout, &stderr)
		if code != 1 || !strings.HasPrefix(stderr.String(), c.want) {
			t.Errorf("%q = %d, %q; want 1 and %q", c.args, code, stderr.String(), c.want)
		}
	}
}
