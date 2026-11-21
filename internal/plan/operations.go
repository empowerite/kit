package plan

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/empowerite/kit/internal/ports"
)

// digest gives the sha256 of what is at a path: a file's bytes, a link's target, or "absent".
func digest(files ports.Files, name string, link bool) (string, error) {
	var text []byte
	var err error
	if link {
		var target string
		target, err = files.ReadLink(name)
		text = []byte("link " + target)
	} else {
		text, err = files.Read(name)
	}
	switch {
	case errors.Is(err, fs.ErrNotExist) || link && errors.Is(err, fs.ErrInvalid):
		return "absent", nil
	case err != nil:
		return "", err
	}
	sum := sha256.Sum256(text)
	return hex.EncodeToString(sum[:]), nil
}

// gone ignores the error of removing what is not there.
func gone(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// WriteFile writes a file of the repository's, at name, a path from its root.
func WriteFile(files ports.Files, root, name string, data []byte) Operation {
	return writeFile{files, root, name, data}
}

type writeFile struct {
	files      ports.Files
	root, name string
	data       []byte
}

func (o writeFile) path() string { return o.root + "/" + o.name }

func (o writeFile) Step() Step { return Step{System: "files", Does: "write " + o.name, Undoable: true} }

func (o writeFile) Facts(context.Context) (string, error) { return digest(o.files, o.path(), false) }

func (o writeFile) Apply(context.Context) error { return o.files.Write(o.path(), o.data, 0o644) }

// The header of .kit/.gitignore, which names what kit placed under .kit/, itself first.
const ignoreHeader = "# .kit/.gitignore - what kit placed here, which git never sees (written by kit, not by hand)\n"

// Place places a file kit owns under the repository's .kit/, at name, a path below it: a read-only copy of data, or,
// where link is given, a link to it. .kit/.gitignore names it, so git never sees it. A file of that name that git
// tracks is the repository's own, and kit refuses to place over it.
func Place(files ports.Files, git ports.Git, root, name string, data []byte, link string) Operation {
	return place{kitFile{files, git, root, name}, data, link}
}

// Unplace takes a file kit placed out of .kit/ again, and its name out of .kit/.gitignore, which goes too once it
// names nothing else.
func Unplace(files ports.Files, git ports.Git, root, name string) Operation {
	return unplace{kitFile{files, git, root, name}}
}

// kitFile is a path below .kit/, and what reads and writes it.
type kitFile struct {
	files      ports.Files
	git        ports.Git
	root, name string
}

func (k kitFile) path() string   { return k.root + "/.kit/" + k.name }
func (k kitFile) ignore() string { return k.root + "/.kit/.gitignore" }

// facts refuses a file git tracks, and gives the digests of the file and of .kit/.gitignore.
func (k kitFile) facts(ctx context.Context, link bool) (string, error) {
	for _, name := range []string{".kit/" + k.name, ".kit/.gitignore"} {
		tracked, err := k.git.Tracked(ctx, k.root, name)
		if err != nil {
			return "", err
		}
		if tracked {
			return "", fmt.Errorf("%s is the repository's own: git tracks it, so kit does not place it", name)
		}
	}
	file, err := digest(k.files, k.path(), link)
	if err != nil {
		return "", err
	}
	ignore, err := digest(k.files, k.ignore(), false)
	return file + " " + ignore, err
}

// names gives what .kit/.gitignore names now, but itself.
func (k kitFile) names() ([]string, error) {
	text, err := k.files.Read(k.ignore())
	if err := gone(err); err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(string(text), "\n") {
		if strings.HasPrefix(line, "/") && line != "/.gitignore" {
			names = append(names, line[1:])
		}
	}
	return names, nil
}

// write writes .kit/.gitignore to name what it is given, itself first, or removes it where it is given nothing.
func (k kitFile) write(names []string) error {
	if len(names) == 0 {
		return gone(k.files.Remove(k.ignore()))
	}
	slices.Sort(names)
	text := ignoreHeader + "/.gitignore\n/" + strings.Join(slices.Compact(names), "\n/") + "\n"
	return k.replace(k.ignore(), []byte(text))
}

// replace writes a read-only file, where it does not already hold data.
func (k kitFile) replace(path string, data []byte) error {
	if now, err := k.files.Read(path); err == nil && bytes.Equal(now, data) {
		return nil
	}
	if err := gone(k.files.Remove(path)); err != nil {
		return err
	}
	return k.files.Write(path, data, 0o444)
}

type place struct {
	kitFile
	data []byte
	link string
}

func (o place) Step() Step {
	does := "place .kit/" + o.name + ", read-only"
	if o.link != "" {
		does = "place .kit/" + o.name + ", a link to " + o.link
	}
	return Step{System: ".kit", Does: does, Undoable: true, Hydrates: true}
}

func (o place) Facts(ctx context.Context) (string, error) { return o.facts(ctx, o.link != "") }

func (o place) Apply(context.Context) error {
	names, err := o.names()
	if err != nil {
		return err
	}
	if err := o.write(append(names, o.name)); err != nil {
		return err
	}
	if o.link == "" {
		return o.replace(o.path(), o.data)
	}
	if now, err := o.files.ReadLink(o.path()); err == nil && now == o.link {
		return nil
	}
	if err := gone(o.files.Remove(o.path())); err != nil {
		return err
	}
	return o.files.Link(o.link, o.path())
}

type unplace struct{ kitFile }

func (o unplace) Step() Step {
	return Step{System: ".kit", Does: "take out .kit/" + o.name, Undoable: true, Hydrates: true}
}

func (o unplace) Facts(ctx context.Context) (string, error) { return o.facts(ctx, false) }

func (o unplace) Apply(context.Context) error {
	names, err := o.names()
	if err != nil {
		return err
	}
	if err := gone(o.files.Remove(o.path())); err != nil {
		return err
	}
	return o.write(slices.DeleteFunc(names, func(n string) bool { return n == o.name }))
}
