package toolchain

import (
	"bytes"
	"errors"
	"slices"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/empowerite/kit/internal/data"
)

// Pin is one tool kit.yml pins: its name, its version, and an entry of the repository's own, or nil for kit's.
type Pin struct {
	Name, Version string
	Entry         *Entry
}

// pinValue is a pin as kit.yml writes it: a version, or an entry with a version of its own.
type pinValue struct {
	version string
	entry   *Entry
}

func (p *pinValue) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		p.version = n.Value
		return nil
	}
	var full struct {
		Version string `yaml:"version"`
		Entry   `yaml:",inline"`
	}
	err := n.Decode(&full)
	p.version, p.entry = full.Version, &full.Entry
	return err
}

// ReadPins gives the tools kit.yml pins, sorted by name. A kit.yml that is absent or holds nothing pins nothing.
func ReadPins(src data.Source) ([]Pin, error) {
	var doc yaml.Node
	if yaml.Unmarshal(src.Text, &doc) == nil && len(doc.Content) == 0 {
		return nil, nil
	}
	var file struct {
		Tools map[string]pinValue `yaml:"tools"`
	}
	if err := data.Decode(src, "kit", &file); err != nil {
		return nil, err
	}
	var pins []Pin
	for name, v := range file.Tools {
		pins = append(pins, Pin{name, v.version, v.entry})
	}
	slices.SortFunc(pins, func(a, b Pin) int { return strings.Compare(a.Name, b.Name) })
	return pins, nil
}

// Repin gives kit.yml's text with a tool pinned at a version: an entry of the repository's own gets the version, and
// a tool not yet pinned is added. The rest of the file, and its comments, are kept.
func Repin(text, name, version string) (string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(text), &doc); err != nil {
		return "", err
	}
	if len(doc.Content) == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	top := doc.Content[0]
	if top.Kind != yaml.MappingNode {
		return "", errors.New("kit.yml: it is not a mapping of keys to values")
	}
	tools := value(top, "tools", func() *yaml.Node { return &yaml.Node{Kind: yaml.MappingNode} })
	if tools.Kind != yaml.MappingNode {
		return "", errors.New("kit.yml: tools: is not a mapping of names to versions")
	}
	pinned := value(tools, name, func() *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode} })
	if pinned.Kind == yaml.MappingNode {
		pinned = value(pinned, "version", func() *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode} })
	}
	pinned.Value, pinned.Tag, pinned.Style = version, "!!str", 0
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	err := enc.Encode(&doc)
	enc.Close()
	return b.String(), err
}

// value gives the value of a mapping's key, adding the key with the value made where the mapping has none.
func value(mapping *yaml.Node, key string, made func() *yaml.Node) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	v := made()
	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, v)
	return v
}
