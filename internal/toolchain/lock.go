package toolchain

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	"github.com/empowerite/kit/internal/data"
)

// Lock is kit.lock: each tool kit.yml pins, locked for every platform.
type Lock struct {
	Tools map[string]Locked `yaml:"tools" json:"tools"`
}

// Locked is one tool, locked: its version, the entry it was made from, and its file for each platform.
type Locked struct {
	Version string                  `yaml:"version" json:"version"`
	Entry   Entry                   `yaml:"entry" json:"entry"`
	Files   map[Platform]LockedFile `yaml:"files" json:"files"`
}

// LockedFile is the file of one platform: where it is downloaded from, its sha256, the program's path in it and the
// program's own sha256, and what confirmed the file's digest beside kit's own download: "github", "published", or
// nothing, where it was pinned with --unconfirmed.
type LockedFile struct {
	URL           string   `yaml:"url" json:"url"`
	Digest        string   `yaml:"digest" json:"digest"`
	Program       string   `yaml:"program" json:"program"`
	ProgramDigest string   `yaml:"program_digest" json:"program_digest"`
	Confirmed     []string `yaml:"confirmed" json:"confirmed"`
}

// Finding is one thing a check finds wrong: where, what, why it breaks "the same tools", and the fix.
type Finding struct {
	Where string `json:"where"`
	What  string `json:"what"`
	Why   string `json:"why"`
	Fix   string `json:"fix"`
}

// ReadLock reads kit.lock. A lock that is absent locks nothing.
func ReadLock(text []byte) (Lock, error) {
	lock := Lock{Tools: map[string]Locked{}}
	if len(text) == 0 {
		return lock, nil
	}
	err := data.Decode(data.Source{Name: "kit.lock", Text: text}, "lock", &lock)
	return lock, err
}

// JSON gives kit.lock's text: sorted, indented, and with a final newline, so the same lock is the same bytes.
func (l Lock) JSON() []byte {
	text, _ := json.MarshalIndent(l, "", "  ")
	return append(text, '\n')
}

// EntryOf gives the entry a pin is fetched by: the repository's own, or else kit's list's.
func EntryOf(pin Pin, list map[string]Entry) Entry {
	if pin.Entry != nil {
		return *pin.Entry
	}
	return list[pin.Name]
}

// CheckLock holds kit.yml's pins and kit.lock to each other: every pin is locked at its version, from the entry in
// use, with a file and both digests for every platform, and the lock holds nothing else.
func CheckLock(pins []Pin, list map[string]Entry, lock Lock) []Finding {
	var found []Finding
	relock := func(what, why string) {
		found = append(found, Finding{"kit.lock", what, why, "kit tools lock"})
	}
	pinned := map[string]bool{}
	for _, pin := range pins {
		pinned[pin.Name] = true
		entry := EntryOf(pin, list)
		if pin.Entry == nil && list[pin.Name].GitHub == "" {
			found = append(found, Finding{"kit.yml", "kit's list holds no " + pin.Name,
				"kit cannot say where its releases are, so it cannot fetch it",
				"give its entry under tools: in kit.yml, in the shape kit data show tools gives, or pin a tool the list holds"})
			continue
		}
		l, ok := lock.Tools[pin.Name]
		switch {
		case !ok:
			relock("kit.lock holds no "+pin.Name, "kit.yml pins it, and no digest says which bytes to run")
			continue
		case l.Version != pin.Version:
			relock(fmt.Sprintf("kit.lock holds %s %s, and kit.yml pins %s", pin.Name, l.Version, pin.Version),
				"a desk and CI would not agree on the version to run")
			continue
		case !reflect.DeepEqual(l.Entry, entry):
			relock("kit.lock was made from an entry of "+pin.Name+" that is no longer the one in use",
				"the files the lock names are not those the entry now names")
			continue
		}
		for _, p := range Platforms {
			f, ok := l.Files[p]
			switch {
			case !ok || f.URL == "" || f.Digest == "":
				relock(fmt.Sprintf("kit.lock holds no file of %s for %s", pin.Name, p),
					"a machine of that platform has nothing to fetch")
			case f.ProgramDigest == "":
				relock(fmt.Sprintf("kit.lock holds no digest of %s's program for %s", pin.Name, p),
					"kit cannot check the program before it runs it")
			}
		}
	}
	var extra []string
	for name := range lock.Tools {
		if !pinned[name] {
			extra = append(extra, name)
		}
	}
	slices.Sort(extra)
	for _, name := range extra {
		relock("kit.lock holds "+name+", which kit.yml does not pin", "the lock says more than the pins do")
	}
	return found
}
