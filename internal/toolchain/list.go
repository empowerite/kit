// Package toolchain holds the tools a repository pins: kit's list of tools, the pins in kit.yml, the lock in
// kit.lock, and the check that holds them to each other. It reads and writes only through ports. It may import
// model, data, ports, and plan.
package toolchain

import (
	"strings"

	"github.com/empowerite/kit/internal/data"
)

// Platform is a system and a machine, as "darwin/arm64".
type Platform string

// Platforms are the four a lock covers, the four kit builds for.
var Platforms = []Platform{"darwin/arm64", "darwin/amd64", "linux/arm64", "linux/amd64"}

// Entry says where a tool's releases are, which file of a release is for each platform, where the program is inside
// it, and where the tool publishes its own checksums, if it does. {version}, {file}, and {dir} in them are filled in.
type Entry struct {
	GitHub    string              `yaml:"github" json:"github"`
	Tag       string              `yaml:"tag" json:"tag"`
	Files     map[Platform]string `yaml:"files" json:"files"`
	Program   string              `yaml:"program" json:"program"`
	Checksums string              `yaml:"checksums,omitempty" json:"checksums,omitempty"`
}

// List gives kit's own list of tools, by name.
func List() (map[string]Entry, error) {
	text, _ := data.Show("tools")
	var file struct {
		Tools map[string]Entry `yaml:"tools"`
	}
	err := data.Decode(data.Source{Name: "tools.yml", Text: text}, "tools", &file)
	return file.Tools, err
}

// expand fills a template's holes: {version}, {file}, and {dir}, which is the file's name without .tar.gz or .zip.
func expand(template, version, file string) string {
	dir := strings.TrimSuffix(strings.TrimSuffix(file, ".tar.gz"), ".zip")
	return strings.NewReplacer("{version}", version, "{file}", file, "{dir}", dir).Replace(template)
}
