package toolchain

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/empowerite/kit/internal/data"
	"github.com/empowerite/kit/internal/ports"
)

var ctx = context.Background()

func sha(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// tgz gives a .tar.gz that holds each file given, by its path.
func tgz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := gzip.NewWriter(&b)
	w := tar.NewWriter(z)
	for name, text := range files {
		if err := w.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(text))}); err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(text))
	}
	w.Close()
	z.Close()
	return b.Bytes()
}

// zipped gives a .zip that holds each file given, by its path.
func zipped(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for name, text := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		f.Write([]byte(text))
	}
	w.Close()
	return b.Bytes()
}

// forge is a forge's releases and downloads, in memory: each file by its address, and what GitHub keeps of it.
type forge struct {
	files   map[string][]byte // by address
	digests map[string]string // what GitHub keeps, by address; "" for none
	newest  map[string]string // a repository's newest tag
	fail    error
}

func newForge() *forge {
	return &forge{files: map[string][]byte{}, digests: map[string]string{}, newest: map[string]string{}}
}

func address(repo, tag, name string) string {
	return "https://github.com/" + repo + "/releases/download/" + tag + "/" + name
}

// release puts a file in a release, with GitHub's digest of it where keep is true.
func (f *forge) release(repo, tag, name string, body []byte, keep bool) {
	url := address(repo, tag, name)
	f.files[url] = body
	if keep {
		f.digests[url] = sha(body)
	}
}

func (f *forge) Asset(_ context.Context, repo, tag, name string) (ports.Asset, error) {
	url := address(repo, tag, name)
	if _, ok := f.files[url]; !ok || f.fail != nil {
		return ports.Asset{}, errors.Join(f.fail, fmt.Errorf("github: %s %s has no file %s", repo, tag, name))
	}
	return ports.Asset{URL: url, Digest: f.digests[url]}, nil
}

func (f *forge) Newest(_ context.Context, repo string) (string, error) {
	if tag, ok := f.newest[repo]; ok {
		return tag, nil
	}
	return "", fmt.Errorf("github: %s has no release", repo)
}

func (f *forge) Get(_ context.Context, url string) ([]byte, error) {
	if body, ok := f.files[url]; ok {
		return body, nil
	}
	return nil, fmt.Errorf("download: %s: not found", url)
}

// shellcheck is an entry in the shape of kit's own for shellcheck, with each platform's archive in the forge.
func shellcheck(t *testing.T, f *forge, version string, keep bool) Entry {
	t.Helper()
	e := Entry{GitHub: "koalaman/shellcheck", Tag: "v{version}", Program: "shellcheck-v{version}/shellcheck",
		Files: map[Platform]string{}}
	for _, p := range Platforms {
		name := "shellcheck-v{version}." + strings.ReplaceAll(string(p), "/", ".") + ".tar.gz"
		e.Files[p] = name
		body := tgz(t, map[string]string{"shellcheck-v" + version + "/shellcheck": "shellcheck " + version + " for " + string(p),
			"shellcheck-v" + version + "/README.txt": "read me"})
		f.release(e.GitHub, "v"+version, expand(name, version, ""), body, keep)
	}
	return e
}

func TestKitsListNamesAllFourPlatformsForEveryTool(t *testing.T) {
	list, err := List()
	if err != nil {
		t.Fatalf("List gave the error %v", err)
	}
	for _, name := range []string{"shellcheck", "lychee"} {
		e, ok := list[name]
		if !ok {
			t.Errorf("kit's list holds no %s", name)
		}
		for _, p := range Platforms {
			if e.Files[p] == "" {
				t.Errorf("kit's entry of %s names no file for %s", name, p)
			}
		}
	}
	if list["lychee"].Checksums != "{file}.sha256" {
		t.Errorf("lychee's checksums are %q; want {file}.sha256", list["lychee"].Checksums)
	}
}

func TestExpandFillsTheHoles(t *testing.T) {
	if got := expand("{dir}/lychee {version} {file}", "0.24.2", "lychee-x.tar.gz"); got != "lychee-x/lychee 0.24.2 lychee-x.tar.gz" {
		t.Errorf("expand = %q", got)
	}
	if got := expand("{dir}", "1", "a.zip"); got != "a" {
		t.Errorf("expand of a zip's {dir} = %q; want a", got)
	}
}

func TestReadPinsTakesAVersionOrAnEntryOfTheRepositorysOwn(t *testing.T) {
	text := "tools:\n  shellcheck: 0.11.0\n  mine:\n    version: \"2.0\"\n    github: me/mine\n    tag: v{version}\n" +
		"    program: mine\n    files:\n      darwin/arm64: m.zip\n      darwin/amd64: m.zip\n      linux/arm64: m.zip\n      linux/amd64: m.zip\n"
	pins, err := ReadPins(data.Source{Name: "kit.yml", Text: []byte(text)})
	if err != nil {
		t.Fatalf("ReadPins gave the error %v", err)
	}
	if len(pins) != 2 || pins[0].Name != "mine" || pins[0].Version != "2.0" || pins[0].Entry == nil ||
		pins[0].Entry.GitHub != "me/mine" || pins[1] != (Pin{Name: "shellcheck", Version: "0.11.0"}) {
		t.Errorf("ReadPins = %+v", pins)
	}
}

func TestReadPinsRefusesAVersionThatIsANumber(t *testing.T) {
	// YAML reads 1.10 as the number 1.1, so a version must be a string: "1.10".
	_, err := ReadPins(data.Source{Name: "kit.yml", Text: []byte("tools:\n  shellcheck: 1.10\n")})
	if err == nil || !strings.HasPrefix(err.Error(), "kit.yml:2: tools.shellcheck: ") {
		t.Errorf("ReadPins gave the error %v; want one at kit.yml:2", err)
	}
}

func TestReadPinsOfNoFileIsNoPins(t *testing.T) {
	if pins, err := ReadPins(data.Source{Name: "kit.yml"}); pins != nil || err != nil {
		t.Errorf("ReadPins of nothing = %v, %v; want no pins", pins, err)
	}
	if pins, err := ReadPins(data.Source{Name: "kit.yml", Text: []byte("# nothing yet\n")}); pins != nil || err != nil {
		t.Errorf("ReadPins of a comment = %v, %v; want no pins", pins, err)
	}
}

func TestPinWritesTheVersionAndKeepsTheRest(t *testing.T) {
	text := "# the repository's kit\ntools:\n  shellcheck: 0.10.0 # older\n  mine:\n    version: \"1\"\n    github: me/mine\n"
	got, err := Repin(text, "shellcheck", "0.11.0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "# the repository's kit\n") || !strings.Contains(got, "shellcheck: 0.11.0") ||
		!strings.Contains(got, "github: me/mine") {
		t.Errorf("Pin gave %q", got)
	}
	if got, _ = Repin(got, "mine", "2"); !strings.Contains(got, "version: \"2\"") {
		t.Errorf("Pin of an entry of the repository's own gave %q; want its version 2", got)
	}
	if got, _ = Repin("tools:\n  mine:\n    github: me/mine\n", "mine", "2"); !strings.Contains(got, "version: \"2\"") {
		t.Errorf("Pin of an entry with no version gave %q; want its version 2", got)
	}
	if got, _ = Repin("", "lychee", "0.24.2"); got != "tools:\n  lychee: 0.24.2\n" {
		t.Errorf("Pin into no file gave %q", got)
	}
	if got, _ = Repin("other: 1\n", "lychee", "0.24.2"); !strings.Contains(got, "tools:\n  lychee: 0.24.2\n") {
		t.Errorf("Pin into a file with no tools: gave %q", got)
	}
	if _, err := Repin("[a, b]\n", "lychee", "1"); err == nil {
		t.Error("Pin into a file that is not a mapping gave no error")
	}
	if _, err := Repin("tools: [a]\n", "lychee", "1"); err == nil {
		t.Error("Pin into tools: that is not a mapping gave no error")
	}
	if _, err := Repin("a: [\n", "lychee", "1"); err == nil {
		t.Error("Pin into YAML that does not parse gave no error")
	}
}

func TestResolveRecordsBothDigestsForEveryPlatform(t *testing.T) {
	f := newForge()
	e := shellcheck(t, f, "0.11.0", true)
	got, err := Resolve(ctx, f, f, "shellcheck", "0.11.0", e, false)
	if err != nil {
		t.Fatalf("Resolve gave the error %v", err)
	}
	if got.Version != "0.11.0" || len(got.Files) != 4 || !reflect.DeepEqual(got.Entry, e) {
		t.Fatalf("Resolve = %+v", got)
	}
	file := got.Files["linux/amd64"]
	url := address("koalaman/shellcheck", "v0.11.0", "shellcheck-v0.11.0.linux.amd64.tar.gz")
	want := LockedFile{URL: url, Digest: sha(f.files[url]), Program: "shellcheck-v0.11.0/shellcheck",
		ProgramDigest: sha([]byte("shellcheck 0.11.0 for linux/amd64")), Confirmed: []string{"github"}}
	if !reflect.DeepEqual(file, want) {
		t.Errorf("Resolve gave for linux/amd64 %+v; want %+v", file, want)
	}
}

func TestResolveRefusesADownloadGitHubsDigestDisagreesWith(t *testing.T) {
	f := newForge()
	e := shellcheck(t, f, "0.11.0", true)
	url := address("koalaman/shellcheck", "v0.11.0", "shellcheck-v0.11.0.darwin.arm64.tar.gz")
	f.digests[url] = sha([]byte("something else"))
	_, err := Resolve(ctx, f, f, "shellcheck", "0.11.0", e, false)
	if err == nil || !strings.Contains(err.Error(), "shellcheck-v0.11.0.darwin.arm64.tar.gz") ||
		!strings.Contains(err.Error(), "GitHub keeps "+f.digests[url]) {
		t.Errorf("Resolve gave the error %v; want one that names the file and both digests", err)
	}
}

// lychee is an entry that names a file of checksums, as lychee's does, with each file's checksum in the forge.
func lychee(t *testing.T, f *forge, wrong bool) Entry {
	t.Helper()
	e := Entry{GitHub: "lycheeverse/lychee", Tag: "lychee-v{version}", Program: "{dir}/lychee", Checksums: "{file}.sha256",
		Files: map[Platform]string{}}
	for _, p := range Platforms {
		name := "lychee-" + strings.ReplaceAll(string(p), "/", "-") + ".zip"
		e.Files[p] = name
		body := zipped(t, map[string]string{"lychee-" + strings.ReplaceAll(string(p), "/", "-") + "/lychee": "lychee for " + string(p)})
		f.release(e.GitHub, "lychee-v0.24.2", name, body, false)
		sum := strings.TrimPrefix(sha(body), "sha256:")
		if wrong {
			sum = strings.Repeat("0", 64)
		}
		f.release(e.GitHub, "lychee-v0.24.2", name+".sha256", []byte(sum+"  "+name+"\n"), false)
	}
	return e
}

func TestResolveTakesThePublishedChecksum(t *testing.T) {
	f := newForge()
	got, err := Resolve(ctx, f, f, "lychee", "0.24.2", lychee(t, f, false), false)
	if err != nil {
		t.Fatalf("Resolve gave the error %v", err)
	}
	if file := got.Files["darwin/arm64"]; !reflect.DeepEqual(file.Confirmed, []string{"published"}) ||
		file.Program != "lychee-darwin-arm64/lychee" {
		t.Errorf("Resolve gave for darwin/arm64 %+v; want it confirmed by the published checksum", file)
	}
}

func TestResolveRefusesADownloadThePublishedChecksumDisagreesWith(t *testing.T) {
	f := newForge()
	_, err := Resolve(ctx, f, f, "lychee", "0.24.2", lychee(t, f, true), false)
	if err == nil || !strings.Contains(err.Error(), "lychee publishes sha256:0000") {
		t.Errorf("Resolve gave the error %v; want one that says what lychee publishes", err)
	}
}

func TestResolveRefusesAChecksumFileThatNamesNoChecksum(t *testing.T) {
	f := newForge()
	e := lychee(t, f, false)
	for _, p := range Platforms {
		f.files[address(e.GitHub, "lychee-v0.24.2", e.Files[p]+".sha256")] = []byte("no checksum here\n")
	}
	if _, err := Resolve(ctx, f, f, "lychee", "0.24.2", e, false); err == nil || !strings.Contains(err.Error(), "holds no checksum") {
		t.Errorf("Resolve gave the error %v; want one that says the file holds no checksum", err)
	}
	f.files[address(e.GitHub, "lychee-v0.24.2", e.Files["darwin/arm64"]+".sha256")] = nil
	delete(f.files, address(e.GitHub, "lychee-v0.24.2", e.Files["darwin/arm64"]+".sha256"))
	if _, err := Resolve(ctx, f, f, "lychee", "0.24.2", e, false); err == nil || !strings.Contains(err.Error(), ".sha256") {
		t.Errorf("Resolve gave the error %v; want one that names the missing checksum file", err)
	}
}

func TestResolveRefusesADigestNothingConfirms(t *testing.T) {
	f := newForge()
	e := shellcheck(t, f, "0.10.0", false)
	_, err := Resolve(ctx, f, f, "shellcheck", "0.10.0", e, false)
	if err == nil || !strings.Contains(err.Error(), "nothing confirms") || !strings.Contains(err.Error(), "--unconfirmed") {
		t.Errorf("Resolve gave the error %v; want one that says nothing confirms it, and names --unconfirmed", err)
	}
	got, err := Resolve(ctx, f, f, "shellcheck", "0.10.0", e, true)
	if err != nil || len(got.Files["linux/arm64"].Confirmed) != 0 {
		t.Errorf("Resolve with unconfirmed = %+v, %v; want the file pinned, confirmed by nothing", got.Files["linux/arm64"], err)
	}
}

func TestResolveRefusesAnArchiveWithoutTheProgram(t *testing.T) {
	f := newForge()
	e := shellcheck(t, f, "0.11.0", true)
	e.Program = "elsewhere/shellcheck"
	_, err := Resolve(ctx, f, f, "shellcheck", "0.11.0", e, false)
	if err == nil || !strings.Contains(err.Error(), "holds no elsewhere/shellcheck") {
		t.Errorf("Resolve gave the error %v; want one that names the path", err)
	}
}

func TestResolveSaysWhatTheForgeSays(t *testing.T) {
	f := newForge()
	e := shellcheck(t, f, "0.11.0", true)
	if _, err := Resolve(ctx, f, f, "shellcheck", "9.9.9", e, false); err == nil || !strings.Contains(err.Error(), "has no file") {
		t.Errorf("Resolve of a version with no release gave %v", err)
	}
	url := address("koalaman/shellcheck", "v0.11.0", "shellcheck-v0.11.0.linux.arm64.tar.gz")
	f.files[url] = []byte("not gzip")
	f.digests[url] = sha(f.files[url])
	if _, err := Resolve(ctx, f, f, "shellcheck", "0.11.0", e, false); err == nil {
		t.Error("Resolve of a file that is no archive gave no error")
	}
}

type failingDownloads struct{}

func (failingDownloads) Get(context.Context, string) ([]byte, error) {
	return nil, errors.New("offline")
}

func TestResolveSaysADownloadThatFails(t *testing.T) {
	f := newForge()
	e := shellcheck(t, f, "0.11.0", true)
	if _, err := Resolve(ctx, f, failingDownloads{}, "shellcheck", "0.11.0", e, false); err == nil || !strings.Contains(err.Error(), "offline") {
		t.Errorf("Resolve gave %v; want offline", err)
	}
}

func TestArchiveReadsTarGzAndZip(t *testing.T) {
	for _, c := range []struct {
		name string
		body []byte
	}{
		{"a.tar.gz", tgz(t, map[string]string{"d/p": "program", "d/q": "other"})},
		{"a.zip", zipped(t, map[string]string{"d/q": "other", "d/p": "program"})},
	} {
		got, err := extract(c.name, c.body, "d/p")
		if err != nil || string(got) != "program" {
			t.Errorf("extract(%s) = %q, %v; want the program", c.name, got, err)
		}
	}
	if _, err := extract("a.tar.xz", nil, "p"); err == nil || !strings.Contains(err.Error(), ".tar.gz and .zip") {
		t.Errorf("extract of a .tar.xz gave %v; want one that names what kit reads", err)
	}
	if _, err := extract("a.zip", []byte("not a zip"), "p"); err == nil {
		t.Error("extract of a broken zip gave no error")
	}
	broken := tgz(t, map[string]string{"d/p": "program"})
	if _, err := extract("a.tar.gz", broken[:len(broken)/2], "d/p"); err == nil {
		t.Error("extract of a broken tar.gz gave no error")
	}
	if _, err := extract("a.zip", zipped(t, map[string]string{"d/q": "x"}), "d/p"); err == nil ||
		!strings.Contains(err.Error(), "holds no d/p") {
		t.Errorf("extract of a zip without the program gave %v", err)
	}
}
