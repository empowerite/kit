package os

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/empowerite/kit/internal/adapters/fake"
	"github.com/empowerite/kit/internal/ports"
)

// contract is what kit's code leans on of files: the machine's and the fake's are each held to it, so that what the
// tests prove with the fake holds of the machine's.
func contract(t *testing.T, f ports.Files, root string) {
	t.Helper()
	must(t, f.Write(root+"/d/file", []byte("x"), 0o444))
	must(t, f.Link("file", root+"/d/link"))
	if info, err := f.Stat(root + "/d/link"); err != nil || info.Mode()&fs.ModeSymlink == 0 {
		t.Errorf("Stat of a link = %v, %v; want a link, not what it leads to", info, err)
	}
	if info, err := f.Stat(root + "/d/file"); err != nil || info.Mode()&fs.ModeSymlink != 0 || info.Mode().Perm() != 0o444 {
		t.Errorf("Stat of a file = %v, %v; want a read-only file", info, err)
	}
	if got, err := f.Read(root + "/d/link"); err != nil || string(got) != "x" {
		t.Errorf("Read through a link = %q, %v", got, err)
	}
	if _, err := f.ReadLink(root + "/d/file"); err == nil {
		t.Error("ReadLink of a file gave no error")
	}
	if err := f.Write(root+"/d/file", nil, 0o644); !errors.Is(err, fs.ErrPermission) {
		t.Errorf("Write over a read-only file gave %v; want fs.ErrPermission", err)
	}
	if err := f.Link("x", root+"/d/link"); !errors.Is(err, fs.ErrExist) {
		t.Errorf("Link over a link gave %v; want fs.ErrExist", err)
	}
	for _, err := range []error{f.Remove(root + "/none"), func() error { _, err := f.Stat(root + "/none"); return err }(),
		func() error { _, err := f.Read(root + "/none"); return err }()} {
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("an act on nothing gave %v; want fs.ErrNotExist", err)
		}
	}
	must(t, f.Remove(root+"/d/file"))
	must(t, f.Rename(root+"/d", root+"/e"))
	if _, err := f.Stat(root + "/e/link"); err != nil {
		t.Errorf("Rename did not move what the directory held: %v", err)
	}
}

func TestTheMachinesFilesKeepTheContract(t *testing.T) { contract(t, Files{}, t.TempDir()) }

func TestTheFakesFilesKeepTheContract(t *testing.T) { contract(t, fake.NewFiles(), "/r") }
