// Package data holds the files inside the binary, the layers, the schemas, and explain. It may import model.
//
// A data file is YAML, or JSON, which YAML reads too. kit checks it against its JSON Schema when it loads it, and an
// error names the file and the line of what is wrong. kit's own files, and their schemas, are inside the binary,
// under files/: a file name.yml holds to name.schema.json.
package data

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"go.yaml.in/yaml/v4"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

//go:embed files
var embedded embed.FS

// Kit is the store of kit's own files, inside the binary.
var Kit = Store{embedded}

// Store holds data files and their schemas, as files/<name>.yml and files/<name>.schema.json.
type Store struct{ files fs.FS }

// Source is a file by the name it is shown by, and its bytes.
type Source struct {
	Name string
	Text []byte
}

// Error is what is wrong with a data file, and where.
type Error struct {
	File    string
	Line    int
	Message string
}

func (e *Error) Error() string { return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Message) }

// Decode checks one of kit's files, or a repository's, against the schema of kit's that it names, and decodes it into
// what into points to.
func Decode(src Source, schema string, into any) error { return Kit.Decode(src, schema, into) }

// Show gives the text of one of kit's own files, by its name.
func Show(name string) ([]byte, error) { return Kit.Show(name) }

// Names gives the names of kit's own files, sorted.
func Names() []string { return Kit.Names() }

// Decode parses src, checks it against the named schema, and decodes it into what into points to. Each error is an
// *Error that names the line: of the YAML that does not parse, or of the value the schema refuses.
func (s Store) Decode(src Source, schema string, into any) error {
	at := func(line int, message string) error { return &Error{src.Name, max(line, 1), message} }
	var doc yaml.Node
	if err := yaml.Unmarshal(src.Text, &doc); err != nil {
		return at(said(err))
	}
	if len(doc.Content) == 0 {
		return at(1, "the file is empty")
	}
	var value any
	if err := doc.Decode(&value); err != nil {
		return at(said(err))
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return at(1, "it holds what JSON cannot: "+err.Error())
	}
	instance, _ := jsonschema.UnmarshalJSON(bytes.NewReader(encoded))
	compiled, err := s.schema(schema)
	if err != nil {
		return err
	}
	if err := compiled.Validate(instance); err != nil {
		leaf := err.(*jsonschema.ValidationError)
		for len(leaf.Causes) > 0 {
			leaf = leaf.Causes[0]
		}
		words := leaf.ErrorKind.LocalizedString(message.NewPrinter(language.English))
		if len(leaf.InstanceLocation) > 0 {
			words = strings.Join(leaf.InstanceLocation, ".") + ": " + words
		}
		location := leaf.InstanceLocation
		if extra, ok := leaf.ErrorKind.(*kind.AdditionalProperties); ok {
			// The line of the first key the schema does not know, and not of the mapping that holds it.
			location = append(slices.Clone(location), extra.Properties[0])
		}
		return at(lineOf(doc.Content[0], location), words)
	}
	if err := doc.Decode(into); err != nil {
		return at(said(err))
	}
	return nil
}

// schema compiles the named schema from the store's own files, and from nothing else: no file of the machine's, and
// nothing from the network.
func (s Store) schema(name string) (*jsonschema.Schema, error) {
	file := name + ".schema.json"
	text, err := fs.ReadFile(s.files, "files/"+file)
	if err != nil {
		return nil, fmt.Errorf("data: no schema named %s", name)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(text))
	if err != nil {
		return nil, fmt.Errorf("data: %s: %v", file, err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(jsonschema.SchemeURLLoader{})
	url := "kit:///" + file
	var compiled *jsonschema.Schema
	if err = compiler.AddResource(url, doc); err == nil {
		compiled, err = compiler.Compile(url)
	}
	if err != nil {
		return nil, fmt.Errorf("data: %s: %v", file, err)
	}
	return compiled, nil
}

// lineOf follows a value's location from the top of a file down through its nodes, and gives the line of the deepest
// node it reaches. Where the last step is a key, the line is the key's: a value may start on a later line, and a key
// the schema does not know has no value of interest.
func lineOf(node *yaml.Node, location []string) int {
	for at, step := range location {
		var next *yaml.Node
		switch node.Kind {
		case yaml.MappingNode:
			for i := 0; i+1 < len(node.Content); i += 2 {
				if node.Content[i].Value == step {
					next = node.Content[i+1]
					if at == len(location)-1 {
						next = node.Content[i]
					}
				}
			}
		case yaml.SequenceNode:
			if i, err := strconv.Atoi(step); err == nil && i < len(node.Content) {
				next = node.Content[i]
			}
		}
		if next == nil {
			break
		}
		node = next
	}
	return node.Line
}

// said gives the line, and the words, of what yaml says is wrong: it names a line as "line 3: …". What names no
// line is put at the first.
func said(err error) (int, string) {
	if m := lineWords.FindStringSubmatch(err.Error()); m != nil {
		line, _ := strconv.Atoi(m[1])
		return line, m[2]
	}
	return 1, err.Error()
}

var lineWords = regexp.MustCompile(`line (\d+): ([^\n;]*)`)

// Show gives the text of one of the store's files, by its name.
func (s Store) Show(name string) ([]byte, error) {
	text, err := fs.ReadFile(s.files, "files/"+name+".yml")
	if err != nil {
		return nil, fmt.Errorf("data: no file named %s: the files are %s", name, strings.Join(s.Names(), ", "))
	}
	return text, nil
}

// Names gives the names of the store's files, sorted.
func (s Store) Names() []string {
	found, _ := fs.Glob(s.files, "files/*.yml")
	var names []string
	for _, name := range found {
		names = append(names, strings.TrimSuffix(path.Base(name), ".yml"))
	}
	slices.Sort(names)
	return names
}

// Over gives base with over laid on it: where both hold a key, over's value wins, whole. Neither is changed.
func Over(base, over map[string]any) map[string]any {
	laid := maps.Clone(base)
	maps.Copy(laid, over)
	return laid
}
