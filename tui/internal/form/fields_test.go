// The form's fields come from a verb's grammar and the answers so far; the command line comes back from the fields;
// a static render of every verb's empty form is pinned as a golden file.
package form

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/empowerite/kit/tui/internal/syntax"
)

var update = flag.Bool("update", false, "rewrite the golden renders")

// specs decodes the syntax package's fixtures, keyed by verb.
func specs(t *testing.T) map[string]syntax.Spec {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "syntax", "testdata", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	out := map[string]syntax.Spec{}
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			t.Fatal(err)
		}
		spec, err := syntax.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		out[spec.Verb] = spec
	}
	return out
}

// kinds lists the fields' kinds, for a short assertion.
func kinds(fields []Field) string {
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		parts = append(parts, f.Kind.String())
	}
	return strings.Join(parts, " ")
}

func TestFieldsForEveryVerb(t *testing.T) {
	for verb, spec := range specs(t) {
		fields, err := Fields(spec, Answers{})
		if err != nil {
			t.Errorf("%s: %v", verb, err)
		}
		if verb == "version" && len(fields) != 0 {
			t.Errorf("version has fields: %+v", fields)
		}
	}
}

func TestCommitEmptyAndPrefilled(t *testing.T) {
	commit := specs(t)["commit"]
	fields, err := Fields(commit, Answers{})
	if err != nil {
		t.Fatal(err)
	}
	if kinds(fields) != "Toggle OneOf PickMany" {
		t.Fatalf("empty commit: %s", kinds(fields))
	}
	if fields[1].Options[0] != "--message" || fields[1].Options[1] != "--file" || fields[1].Chosen != -1 || !fields[1].Required {
		t.Errorf("the choice: %+v", fields[1])
	}
	if FirstEmptyRequired(fields) != 1 {
		t.Errorf("first empty required = %d", FirstEmptyRequired(fields))
	}
	given := FromMatch(syntax.Match{Status: 0, Operands: []string{"x"},
		Options: map[string][]string{"--message": {"hi"}}, Values: map[string][]string{"--message": {"hi"}, "<path>": {"x"}}})
	fields, err = Fields(commit, given)
	if err != nil {
		t.Fatal(err)
	}
	if kinds(fields) != "Toggle OneOf Lines PickMany" {
		t.Fatalf("prefilled commit: %s", kinds(fields))
	}
	if fields[1].Chosen != 0 || fields[2].Key != "--message" || fields[2].Values[0] != "hi" || fields[3].Values[0] != "x" {
		t.Errorf("prefilled: %+v", fields)
	}
	if fields[2].Description == "" || fields[3].Choices != "open" || !fields[3].Repeat {
		t.Errorf("descriptions and choices: %+v %+v", fields[2], fields[3])
	}
	if FirstEmptyRequired(fields) != -1 {
		t.Errorf("first empty required = %d", FirstEmptyRequired(fields))
	}
}

func TestPackageLiteralChoosesItsAlternative(t *testing.T) {
	fields, err := Fields(specs(t)["package"], FromMatch(syntax.Match{Literals: []string{"add"}}))
	if err != nil {
		t.Fatal(err)
	}
	if kinds(fields) != "OneOf PickMany" || fields[0].Options[2] != "add" || fields[0].Chosen != 2 || fields[0].Required {
		t.Fatalf("package add: %s %+v", kinds(fields), fields[0])
	}
	if fields[1].Key != "<package>" || fields[1].Choices != "closed" || !fields[1].Required {
		t.Errorf("the packages: %+v", fields[1])
	}
	fields, _ = Fields(specs(t)["package"], Answers{})
	if kinds(fields) != "OneOf" || fields[0].Chosen != -1 {
		t.Errorf("bare package: %s %+v", kinds(fields), fields[0])
	}
}

func TestStatusRowsAndTheOwnerGroup(t *testing.T) {
	fields, err := Fields(specs(t)["status"], Answers{})
	if err != nil {
		t.Fatal(err)
	}
	if kinds(fields) != "AnyOf OneOf" {
		t.Fatalf("status: %s", kinds(fields))
	}
	if strings.Join(fields[0].Options, " ") != "--no-fetch --json" || len(fields[0].Values) != 0 {
		t.Errorf("the any-of row: %+v", fields[0])
	}
	if strings.Join(fields[1].Options, "|") != "--all|--mine --others --owner" {
		t.Errorf("the owner group: %+v", fields[1])
	}
	chosen := FromMatch(syntax.Match{Options: map[string][]string{"--json": {"1"}, "--mine": {"1"}, "--owner": {"ada", "bob"}},
		Values: map[string][]string{"--json": {"1"}, "--mine": {"1"}, "--owner": {"ada", "bob"}}})
	fields, _ = Fields(specs(t)["status"], chosen)
	if kinds(fields) != "AnyOf OneOf AnyOf Lines" || fields[1].Chosen != 1 || fields[3].Values[1] != "bob" {
		t.Errorf("status --json --mine --owner: %s %+v", kinds(fields), fields)
	}
	if strings.Join(fields[0].Values, " ") != "--json" || strings.Join(fields[2].Values, " ") != "--mine" {
		t.Errorf("toggled: %+v %+v", fields[0], fields[2])
	}
}

func TestNewKeepsTheLiteralInTheLabelAndEndsWithTheWildcard(t *testing.T) {
	fields, err := Fields(specs(t)["new"], Answers{})
	if err != nil {
		t.Fatal(err)
	}
	if kinds(fields) != "Text Pick Pick Wildcard" || fields[0].Label != "file <path>" || fields[0].Key != "<path>" || !fields[0].Required {
		t.Fatalf("new: %s %+v", kinds(fields), fields)
	}
	if fields[1].Key != "--rule" || fields[1].Choices != "closed" || fields[3].Key != "--<hole>" {
		t.Errorf("new's options: %+v", fields)
	}
}

func TestContextHasTwoDoors(t *testing.T) {
	fields, err := Fields(specs(t)["context"], Answers{})
	if err != nil {
		t.Fatal(err)
	}
	if kinds(fields) != "OneOf" || strings.Join(fields[0].Options, "|") != "<path>|--claude" || !fields[0].Required {
		t.Fatalf("context: %s %+v", kinds(fields), fields)
	}
	fields, _ = Fields(specs(t)["context"], FromMatch(syntax.Match{Operands: []string{"a.md"}, Values: map[string][]string{"<path>": {"a.md"}}}))
	if kinds(fields) != "OneOf PickMany Toggle" || fields[0].Chosen != 0 {
		t.Errorf("context a.md: %s", kinds(fields))
	}
}

func TestCommandLineAndQuote(t *testing.T) {
	commit := specs(t)["commit"]
	fields, _ := Fields(commit, FromMatch(syntax.Match{Options: map[string][]string{"--message": {"#1 tui: x"}},
		Values: map[string][]string{"--message": {"#1 tui: x"}, "<path>": {".gitignore"}}}))
	args := CommandLine(commit, fields)
	if strings.Join(args, "\x00") != "--message\x00#1 tui: x\x00.gitignore" {
		t.Errorf("args: %q", args)
	}
	if Quote(args) != "--message '#1 tui: x' .gitignore" {
		t.Errorf("Quote = %q", Quote(args))
	}
	if Quote([]string{"it's", "", "a=b"}) != `'it'\''s' '' a=b` {
		t.Errorf("Quote edge = %q", Quote([]string{"it's", "", "a=b"}))
	}
	fields, _ = Fields(specs(t)["new"], FromMatch(syntax.Match{Literals: []string{"file"}, Operands: []string{"docs/x.md"},
		Options: map[string][]string{"--rule": {"markdown"}, "--title": {"kit tui"}},
		Values:  map[string][]string{"<path>": {"docs/x.md"}, "--rule": {"markdown"}, "--title": {"kit tui"}}}))
	if got := Quote(CommandLine(specs(t)["new"], fields)); got != "file docs/x.md --rule=markdown --title='kit tui'" {
		t.Errorf("new's line = %q", got)
	}
	fields, _ = Fields(specs(t)["status"], FromMatch(syntax.Match{Options: map[string][]string{"--json": {"1"}, "--owner": {"ada"}},
		Values: map[string][]string{"--json": {"1"}, "--owner": {"ada"}}}))
	if got := Quote(CommandLine(specs(t)["status"], fields)); got != "--json --owner=ada" {
		t.Errorf("status's line = %q", got)
	}
}

func TestGoldenRenders(t *testing.T) {
	for verb, spec := range specs(t) {
		fields, err := Fields(spec, Answers{})
		if err != nil {
			t.Fatal(err)
		}
		got := RenderStatic(spec, fields, 100)
		golden := filepath.Join("testdata", verb+".golden")
		if *update {
			if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatalf("%s: %v (run with -update)", verb, err)
		}
		if got != string(want) {
			t.Errorf("%s renders differently from %s:\n%s", verb, golden, got)
		}
	}
}
