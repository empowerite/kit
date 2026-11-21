package plan

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/empowerite/kit/internal/ports"
)

// digest gives the sha256 of what is at a path: a link's target, a file's bytes, or "absent". What it is is read from
// the path itself, never from the kind of an error, since the machine's files and the fake's give different ones.
func digest(files ports.Files, name string) (string, error) {
	info, err := files.Stat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return "absent", nil
	}
	if err != nil {
		return "", err
	}
	var text []byte
	if info.Mode()&fs.ModeSymlink != 0 {
		var target string
		target, err = files.ReadLink(name)
		text = []byte("link " + target)
	} else {
		text, err = files.Read(name)
	}
	return sum(text), err
}

// sum gives the sha256 of bytes, in hex.
func sum(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// gone ignores the error of removing what is not there.
func gone(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// WriteFile writes a file of the repository's, at name, a path from its root. Each line of detail says, in the plan,
// what the write changes, and is covered by the plan's hash with the bytes written.
func WriteFile(files ports.Files, root, name string, data []byte, detail ...string) Operation {
	return writeFile{files, root, name, data, detail}
}

type writeFile struct {
	files      ports.Files
	root, name string
	data       []byte
	detail     []string
}

func (o writeFile) path() string { return o.root + "/" + o.name }

func (o writeFile) Step() Step {
	return Step{System: "files", Does: "write " + o.name, Undoable: true, Detail: o.detail}
}

// Facts covers what the file holds now and what it will hold, so that an approval binds to the bytes written.
func (o writeFile) Facts(context.Context) (string, error) {
	now, err := digest(o.files, o.path())
	return now + " to " + sum(o.data), err
}

func (o writeFile) Apply(context.Context) error { return o.files.Write(o.path(), o.data, 0o644) }

// The header of .kit/.gitignore, which names what kit placed under .kit/, itself first.
const ignoreHeader = "# .kit/.gitignore - what kit placed here, which git never sees (written by kit, not by hand)\n"

// Place places a file kit owns under the repository's .kit/, at name, a path below it: a read-only copy of data, or,
// where link is given, a link to it. .kit/.gitignore names it, so git never sees it. kit refuses to place over a file
// that is the repository's own (git tracks it) or that kit did not place (.kit/.gitignore does not name it).
func Place(files ports.Files, git ports.Git, root, name string, data []byte, link string) Operation {
	return place{kitFile{files, git, root, name}, data, link, 0o444}
}

// PlaceProgram places a program kit owns under .kit/: a copy of data, read-only and runnable.
func PlaceProgram(files ports.Files, git ports.Git, root, name string, data []byte) Operation {
	return place{kitFile{files, git, root, name}, data, "", 0o555}
}

// Unplace takes a file kit placed out of .kit/ again, and its name out of .kit/.gitignore, which goes too once it
// names nothing else.
func Unplace(files ports.Files, git ports.Git, root, name string) Operation {
	return unplace{kitFile{files, git, root, name}}
}

// Placed gives what kit placed under a repository's .kit/, as .kit/.gitignore names it.
func Placed(files ports.Files, root string) ([]string, error) {
	return kitFile{files: files, root: root}.names()
}

// kitFile is a path below .kit/, and what reads and writes it.
type kitFile struct {
	files      ports.Files
	git        ports.Git
	root, name string
}

func (k kitFile) path() string   { return k.root + "/.kit/" + k.name }
func (k kitFile) ignore() string { return k.root + "/.kit/.gitignore" }

// facts refuses a name that leaves .kit/, a file git tracks, and a file kit did not place, and gives the digests of
// the file and of .kit/.gitignore.
func (k kitFile) facts(ctx context.Context) (string, error) {
	if k.name == "" || k.name == "." || path.IsAbs(k.name) || path.Clean(k.name) != k.name ||
		strings.HasPrefix(k.name, "../") || k.name == ".gitignore" {
		return "", fmt.Errorf("%q is not a path below .kit/ that kit may place", k.name)
	}
	for _, name := range []string{".kit/" + k.name, ".kit/.gitignore"} {
		tracked, err := k.git.Tracked(ctx, k.root, name)
		if err != nil {
			return "", err
		}
		if tracked {
			return "", fmt.Errorf("%s is the repository's own: git tracks it, so kit does not place it", name)
		}
	}
	file, err := digest(k.files, k.path())
	if err != nil {
		return "", err
	}
	names, err := k.names()
	if err != nil {
		return "", err
	}
	if file != "absent" && !slices.Contains(names, k.name) {
		return "", fmt.Errorf(".kit/%s is not one kit placed: kit does not place over it", k.name)
	}
	ignore, err := digest(k.files, k.ignore())
	return file + " " + ignore, err
}

// names gives what .kit/.gitignore names now, but itself.
func (k kitFile) names() ([]string, error) {
	text, err := k.files.Read(k.ignore())
	if err := gone(err); err != nil {
		return nil, err
	}
	names := []string{}
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
	return k.replace(k.ignore(), []byte(text), 0o444)
}

// replace writes a read-only file, where it does not already hold data with those permissions.
func (k kitFile) replace(path string, data []byte, perm fs.FileMode) error {
	if now, err := k.files.Read(path); err == nil && bytes.Equal(now, data) {
		if info, err := k.files.Stat(path); err == nil && info.Mode().Perm() == perm {
			return nil
		}
	}
	if err := gone(k.files.Remove(path)); err != nil {
		return err
	}
	return k.files.Write(path, data, perm)
}

type place struct {
	kitFile
	data []byte
	link string
	perm fs.FileMode
}

func (o place) Step() Step {
	does := "place .kit/" + o.name + ", read-only"
	switch {
	case o.link != "":
		does = "place .kit/" + o.name + ", a link to " + o.link
	case o.perm == 0o555:
		does += " and runnable"
	}
	return Step{System: ".kit", Does: does, Undoable: true, Hydrates: true}
}

// Facts covers what is placed, as well as what stands there now.
func (o place) Facts(ctx context.Context) (string, error) {
	now, err := o.facts(ctx)
	return now + " to " + sum(o.data) + " " + o.link, err
}

func (o place) Apply(context.Context) error {
	names, err := o.names()
	if err != nil {
		return err
	}
	if err := o.write(append(names, o.name)); err != nil {
		return err
	}
	if o.link == "" {
		return o.replace(o.path(), o.data, o.perm)
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

func (o unplace) Facts(ctx context.Context) (string, error) { return o.facts(ctx) }

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
