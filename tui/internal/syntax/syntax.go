// Package syntax holds the grammar kit syntax prints for a verb, decoded from its JSON, with the key an argument is
// known by and the usage text a sequence of nodes renders back to, as kit's own syntax.jq gives them.
package syntax

import (
	"encoding/json"
	"io"
	"strings"
)

// Spec is one verb's syntax as kit syntax <verb> prints it: the usage lines as written, the grammar as alternatives of
// node sequences, and each argument's description with its choices, closed or open, when the verb lists them. Extends
// is true for a verb whose arguments grow with the answers; Error is set, instead of Syntax, when a usage line did
// not parse.
type Spec struct {
	Verb      string              `json:"verb"`
	Purpose   string              `json:"purpose"`
	Usage     []string            `json:"usage"`
	Syntax    [][]Node            `json:"syntax"`
	Arguments map[string]Argument `json:"arguments"`
	Extends   bool                `json:"extends"`
	Error     string              `json:"error"`
}

// Node is one element of a sequence. Kind is literal, operand, flag, option, or group. Name is the literal's word, the
// operand's or option's name; Value an option's value name; Inline whether the value is written --name=<value>;
// Wildcard an option that stands for any --x=y; Repeat whether the node may repeat; Optional and Alternatives a
// group's.
type Node struct {
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	Value        string   `json:"value"`
	Inline       bool     `json:"inline"`
	Wildcard     bool     `json:"wildcard"`
	Repeat       bool     `json:"repeat"`
	Optional     bool     `json:"optional"`
	Alternatives [][]Node `json:"alternatives"`
}

// Argument is what the header's argument block says of one key: its description, its choices (closed, open, or
// empty), and whether --extend added it.
type Argument struct {
	Description string `json:"description"`
	Choices     string `json:"choices"`
	Extended    bool   `json:"extended"`
}

// Decode reads one spec, as kit syntax <verb> prints it.
func Decode(r io.Reader) (Spec, error) {
	var spec Spec
	err := json.NewDecoder(r).Decode(&spec)
	return spec, err
}

// DecodeAll reads an array of specs, as kit syntax prints it for several verbs.
func DecodeAll(r io.Reader) ([]Spec, error) {
	var specs []Spec
	err := json.NewDecoder(r).Decode(&specs)
	return specs, err
}

// Key gives the key a node is known by, as the arguments map is keyed: --name for a flag or an option, --<name> for a
// wildcard option, <name> for an operand, and the word itself for a literal.
func Key(n Node) string {
	switch n.Kind {
	case "flag", "option":
		if n.Wildcard {
			return "--<" + n.Name + ">"
		}
		return "--" + n.Name
	case "operand":
		return "<" + n.Name + ">"
	}
	return n.Name
}

// Render gives a sequence of nodes as usage text: a group in its brackets with its alternatives separated by |, an
// option with its value inline or apart, and … after what repeats.
func Render(seq []Node) string {
	parts := make([]string, 0, len(seq))
	for _, n := range seq {
		parts = append(parts, render(n))
	}
	return strings.Join(parts, " ")
}

func render(n Node) string {
	mark := ""
	if n.Repeat {
		mark = "…"
	}
	switch n.Kind {
	case "group":
		inner := make([]string, 0, len(n.Alternatives))
		for _, alternative := range n.Alternatives {
			inner = append(inner, Render(alternative))
		}
		text := strings.Join(inner, " | ")
		if n.Optional {
			return "[" + text + "]" + mark
		}
		return "(" + text + ")" + mark
	case "literal":
		return n.Name + mark
	case "operand":
		return "<" + n.Name + ">" + mark
	case "flag":
		return "--" + n.Name
	}
	if n.Wildcard {
		return "--<" + n.Name + ">=<" + n.Value + ">" + mark
	}
	if n.Inline {
		return "--" + n.Name + "=<" + n.Value + ">" + mark
	}
	return "--" + n.Name + " <" + n.Value + ">" + mark
}
