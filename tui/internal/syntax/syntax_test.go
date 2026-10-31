// The grammar's types decode every fixture kit syntax prints, and render back to the usage line it was read from.
package syntax

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtures decodes every testdata/<verb>.json, keyed by the verb the file is named after.
func fixtures(t *testing.T) map[string]Spec {
	t.Helper()
	files, err := filepath.Glob("testdata/*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	specs := map[string]Spec{}
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			t.Fatal(err)
		}
		spec, err := Decode(f)
		f.Close()
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		specs[strings.TrimSuffix(filepath.Base(file), ".json")] = spec
	}
	return specs
}

func TestDecodeEveryFixture(t *testing.T) {
	for verb, spec := range fixtures(t) {
		if spec.Verb != verb {
			t.Errorf("%s.json: verb %q", verb, spec.Verb)
		}
		if spec.Error != "" {
			t.Errorf("%s: error %q", verb, spec.Error)
		}
		if spec.Purpose == "" || len(spec.Usage) == 0 || len(spec.Syntax) == 0 {
			t.Errorf("%s: purpose %q, %d usage lines, %d alternatives", verb, spec.Purpose, len(spec.Usage), len(spec.Syntax))
		}
	}
}

func TestCommitShape(t *testing.T) {
	commit := fixtures(t)["commit"]
	if len(commit.Syntax) != 1 || len(commit.Syntax[0]) != 3 {
		t.Fatalf("commit: %d alternatives, %d nodes", len(commit.Syntax), len(commit.Syntax[0]))
	}
	seq := commit.Syntax[0]
	if seq[0].Kind != "group" || !seq[0].Optional || seq[0].Alternatives[0][0].Kind != "flag" || seq[0].Alternatives[0][0].Name != "all" {
		t.Errorf("node 1: %+v", seq[0])
	}
	if seq[1].Kind != "group" || seq[1].Optional || len(seq[1].Alternatives) != 2 {
		t.Errorf("node 2: %+v", seq[1])
	}
	message := seq[1].Alternatives[0][0]
	if message.Kind != "option" || message.Name != "message" || message.Value != "text" || message.Inline || !message.Repeat {
		t.Errorf("message: %+v", message)
	}
	if Key(message) != "--message" {
		t.Errorf("Key(message) = %q", Key(message))
	}
	if Render(seq) != "[--all] (--message <text>… | --file <path>) [<path>…]" {
		t.Errorf("Render = %q", Render(seq))
	}
	if commit.Arguments["<path>"].Choices != "open" || commit.Arguments["--message"].Description == "" {
		t.Errorf("arguments: %+v", commit.Arguments)
	}
	if commit.Extends {
		t.Error("commit extends")
	}
}

func TestKeyOfEveryKind(t *testing.T) {
	cases := map[string]Node{
		"--all":    {Kind: "flag", Name: "all"},
		"--rule":   {Kind: "option", Name: "rule", Value: "rule", Inline: true},
		"--<hole>": {Kind: "option", Name: "hole", Value: "value", Inline: true, Wildcard: true},
		"<path>":   {Kind: "operand", Name: "path", Repeat: true},
		"list":     {Kind: "literal", Name: "list"},
	}
	for want, node := range cases {
		if got := Key(node); got != want {
			t.Errorf("Key(%+v) = %q, want %q", node, got, want)
		}
	}
}

func TestRenderRoundTripsEveryUsageLine(t *testing.T) {
	for verb, spec := range fixtures(t) {
		var remainders, renders []string
		for _, line := range spec.Usage {
			remainders = append(remainders, strings.TrimSpace(strings.TrimPrefix(line, "kit "+verb)))
		}
		for _, alternative := range spec.Syntax {
			renders = append(renders, Render(alternative))
		}
		want := strings.Join(remainders, " | ")
		got := strings.Join(renders, " | ")
		if got != want {
			t.Errorf("%s: rendered %q, usage %q", verb, got, want)
		}
	}
}

func TestNewExtendsAndItsChoicesAreClosed(t *testing.T) {
	specs := fixtures(t)
	if !specs["new"].Extends {
		t.Error("new does not extend")
	}
	for verb, spec := range specs {
		if verb != "new" && spec.Extends {
			t.Errorf("%s extends", verb)
		}
	}
	if specs["new"].Arguments["--rule"].Choices != "closed" || specs["package"].Arguments["<package>"].Choices != "closed" {
		t.Error("closed choices missing")
	}
}

func TestDecodeAllReadsAnArray(t *testing.T) {
	specs, err := DecodeAll(strings.NewReader(`[{"verb":"a","syntax":[[]]},{"verb":"b","syntax":[[]],"extends":true}]`))
	if err != nil || len(specs) != 2 || specs[1].Verb != "b" || !specs[1].Extends {
		t.Fatalf("DecodeAll: %v, %+v", err, specs)
	}
}
