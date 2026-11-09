// The form's fields come from a verb's grammar and the answers so far; the command line comes back from the fields;
// a static render of every verb's empty form is pinned as a golden file.
package form

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/empowerite/kit/projects/tui/internal/syntax"
	"github.com/empowerite/kit/projects/tui/internal/testkit"
)

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

func TestPackagesLiteralChoosesItsAlternative(t *testing.T) {
	// kit packages opens with a one-of row over its two usage lines; with the acting line chosen, the literal add
	// opens its word in the row that follows, and the packages come after it
	fields, err := Fields(specs(t)["packages"], Answers{Literals: []string{"add"}, Chosen: map[int]int{0: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if kinds(fields) != "OneOf OneOf PickMany" || fields[1].Options[1] != "add" || fields[1].Chosen != 1 || !fields[1].Required {
		t.Fatalf("packages add: %s %+v", kinds(fields), fields)
	}
	if fields[2].Key != "<package>" || fields[2].Choices != "closed" || !fields[2].Required {
		t.Errorf("the packages: %+v", fields[2])
	}
	// bare, kit packages lists: the line that adds no word is chosen, named kit packages, and the row is optional
	fields, _ = Fields(specs(t)["packages"], Answers{})
	if kinds(fields) != "OneOf OneOf OneOf PickMany" || fields[0].Chosen != 0 || fields[0].Options[0] != "kit packages" ||
		fields[0].Required || fields[0].Description != specs(t)["packages"].Purpose {
		t.Errorf("bare packages: %s %+v", kinds(fields), fields[0])
	}
	// a judged kit packages add style/bash chooses the acting line by its word alone: add opens the line through the
	// group it starts with, as the matcher reads it, and the row under the line then takes the word
	judged := FromMatch(syntax.Match{Literals: []string{"add"}, Operands: []string{"style/bash"},
		Values: map[string][]string{"<package>": {"style/bash"}}})
	fields, _ = Fields(specs(t)["packages"], judged)
	if kinds(fields) != "OneOf OneOf PickMany" || fields[0].Chosen != 1 || fields[1].Chosen != 1 ||
		strings.Join(fields[2].Values, " ") != "style/bash" {
		t.Errorf("packages add style/bash: %s %+v", kinds(fields), fields)
	}
	// the word alone, as kit tui -- packages add hands it over, opens the acting line with add and no package yet
	fields, _ = Fields(specs(t)["packages"], FromMatch(syntax.Match{Literals: []string{"add"}}))
	if kinds(fields) != "OneOf OneOf PickMany" || fields[0].Chosen != 1 || fields[1].Chosen != 1 ||
		len(fields[2].Values) != 0 {
		t.Errorf("packages add: %s %+v", kinds(fields), fields)
	}
}

func TestStatusRowsAndTheOwnerGroup(t *testing.T) {
	// kit flow opens with a one-of row over its seven usage lines; the picture's line, chosen, brings its rows
	fields, err := Fields(specs(t)["flow"], Answers{Chosen: map[int]int{0: 0}})
	if err != nil {
		t.Fatal(err)
	}
	if kinds(fields) != "OneOf AnyOf OneOf OneOf" || fields[0].Chosen != 0 ||
		strings.Join(fields[1].Options, " ") != "--no-fetch --fresh" {
		t.Fatalf("flow: %s %+v", kinds(fields), fields[1])
	}
	if strings.Join(fields[2].Options, "|") != "--text|--json|--jq" || fields[2].Required {
		t.Errorf("the output row, one or the other or neither: %+v", fields[2])
	}
	if strings.Join(fields[3].Options, "|") != "--all|--mine --others --owner" {
		t.Errorf("the owner group: %+v", fields[3])
	}
	// the flags given choose the picture's line without a word: it is the first line that holds them
	chosen := FromMatch(syntax.Match{Options: map[string][]string{"--json": {"1"}, "--mine": {"1"}, "--owner": {"ada", "bob"}},
		Values: map[string][]string{"--json": {"1"}, "--mine": {"1"}, "--owner": {"ada", "bob"}}})
	fields, _ = Fields(specs(t)["flow"], chosen)
	// the chosen --json brings its required flag as a field of its own, as every chosen flag alternative does
	if kinds(fields) != "OneOf AnyOf OneOf Toggle OneOf AnyOf Lines" || fields[0].Chosen != 0 || fields[2].Chosen != 1 ||
		fields[4].Chosen != 1 || fields[6].Values[1] != "bob" {
		t.Errorf("flow --json --mine --owner: %s %+v", kinds(fields), fields)
	}
	if strings.Join(fields[5].Options, " ") != "--mine --others" || strings.Join(fields[5].Values, " ") != "--mine" {
		t.Errorf("the any-of row of owner flags: %+v", fields[5])
	}
}

func TestNewChoosesWhatToMakeAndEndsWithTheWildcard(t *testing.T) {
	fields, err := Fields(specs(t)["new"], Answers{})
	if err != nil {
		t.Fatal(err)
	}
	options := strings.Join(fields[0].Options, " ")
	if kinds(fields) != "OneOf" || !strings.HasPrefix(options, "file kind ") || !strings.HasSuffix(options, " verb issue") || !fields[0].Required {
		t.Fatalf("bare new: %s %+v", kinds(fields), fields)
	}
	fields, _ = Fields(specs(t)["new"], FromMatch(syntax.Match{Literals: []string{"file"}}))
	if kinds(fields) != "OneOf Text Pick Pick Wildcard" || fields[1].Key != "<path>" || !fields[1].Required {
		t.Fatalf("new file: %s %+v", kinds(fields), fields)
	}
	if fields[2].Key != "--rule" || fields[2].Choices != "closed" || fields[4].Key != "--<hole>" {
		t.Errorf("new file's options: %+v", fields)
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
	if kinds(fields) != "OneOf PickMany OneOf" || fields[0].Chosen != 0 || strings.Join(fields[2].Options, " ") != "--json --jq" {
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
	fields, _ = Fields(specs(t)["flow"], FromMatch(syntax.Match{Options: map[string][]string{"--json": {"1"}, "--owner": {"ada"}},
		Values: map[string][]string{"--json": {"1"}, "--owner": {"ada"}}}))
	if got := Quote(CommandLine(specs(t)["flow"], fields)); got != "--json --owner=ada" {
		t.Errorf("flow's line = %q", got)
	}
}

func TestGoldenRenders(t *testing.T) {
	for verb, spec := range specs(t) {
		fields, err := Fields(spec, Answers{})
		if err != nil {
			t.Fatal(err)
		}
		testkit.Golden(t, filepath.Join("testdata", verb+".golden"), RenderStatic(spec, fields, 100))
	}
}

func TestNodesInsideBracketsAreOptionalEvenSeveralTogether(t *testing.T) {
	// kit tui [<verb> [<argument>…]]: neither is needed for the command line to be complete
	fields, err := Fields(specs(t)["tui"], Answers{})
	if err != nil {
		t.Fatal(err)
	}
	if kinds(fields) != "Text Lines" || fields[0].Required || fields[1].Required || FirstEmptyRequired(fields) != -1 {
		t.Errorf("tui: %s %+v", kinds(fields), fields)
	}
	// kit new module <name> --purpose=<text> [--jq]: the flag is optional, the operand and the option are not
	fields, _ = Fields(specs(t)["new"], FromMatch(syntax.Match{Literals: []string{"module"}}))
	if kinds(fields) != "OneOf Text Text Toggle" || !fields[1].Required || !fields[2].Required || fields[3].Required {
		t.Errorf("new module: %s %+v", kinds(fields), fields)
	}
}

func TestARequiredFieldIsTaggedUntilItHasAnAnswer(t *testing.T) {
	spec := specs(t)["new"]
	fields, err := Fields(spec, FromMatch(syntax.Match{Literals: []string{"file"}}))
	if err != nil {
		t.Fatal(err)
	}
	view := RenderStatic(spec, fields, 100)
	if !strings.Contains(view, "\n     <path> required   the new file") || strings.Contains(view, "--rule required") {
		t.Errorf("new file: <path> is tagged, --rule is not:\n%s", view)
	}
	fields, _ = Fields(spec, FromMatch(syntax.Match{Literals: []string{"file"}, Operands: []string{"a.md"},
		Values: map[string][]string{"<path>": {"a.md"}}}))
	if view = RenderStatic(spec, fields, 100); strings.Contains(view, "required") {
		t.Errorf("new file a.md: a tag with the answer given:\n%s", view)
	}
	// the tag's room is kept once the answer is in, so nothing moves
	release := specs(t)["release"]
	fields, _ = Fields(release, FromMatch(syntax.Match{Operands: []string{"1.0.0"}, Values: map[string][]string{"<version>": {"1.0.0"}}}))
	if view = RenderStatic(release, fields, 100); !strings.Contains(view, "\n   <version>           the version") {
		t.Errorf("release 1.0.0: the description moved:\n%s", view)
	}
	// a one-of row drawn one name per line is headed by one of, and the tag after it until one is chosen
	commit := specs(t)["commit"]
	fields, _ = Fields(commit, Answers{})
	if view = RenderStatic(commit, fields, 100); !strings.Contains(view, "\n   one of required\n   --message  the message") {
		t.Errorf("commit: the row is not headed by the tag:\n%s", view)
	}
	fields, _ = Fields(commit, Answers{Chosen: map[int]int{0: 0}})
	if view = RenderStatic(commit, fields, 100); strings.Contains(view, "required") {
		t.Errorf("commit --message: a tag with the alternative chosen:\n%s", view)
	}
	// a one-of row of names on one line carries the tag after its names, with its description after the tag
	flags := syntax.Spec{Verb: "x", Syntax: [][]syntax.Node{{{Kind: "group", Alternatives: [][]syntax.Node{
		{{Kind: "flag", Name: "up"}}, {{Kind: "flag", Name: "down"}}}}}},
		Arguments: map[string]syntax.Argument{"--up": {Description: "goes up"}, "--down": {Description: "goes down"}}}
	fields, _ = Fields(flags, Answers{})
	if view = RenderStatic(flags, fields, 100); !strings.Contains(view, "\n   --up   --down required  goes up") {
		t.Errorf("a row of flags:\n%s", view)
	}
	fields, _ = Fields(flags, Answers{Chosen: map[int]int{0: 1}})
	if view = RenderStatic(flags, fields, 100); !strings.Contains(view, "\n   --up   *--down*           goes down") {
		t.Errorf("a row of flags, one chosen, keeps its column:\n%s", view)
	}
}

func TestAChosenRequiredFlagDrawsNoValueUnderItsName(t *testing.T) {
	spec := specs(t)["context"]
	fields, err := Fields(spec, Answers{Chosen: map[int]int{0: 1}})
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(RenderStatic(spec, fields, 100), "\n") {
		if strings.TrimSpace(line) == "1" {
			t.Fatalf("a 1 under --claude:\n%s", RenderStatic(spec, fields, 100))
		}
	}
}
