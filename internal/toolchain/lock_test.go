package toolchain

import (
	"strings"
	"testing"
)

// locked gives the lock that Resolve makes of shellcheck 0.11.0.
func locked(t *testing.T) (Lock, Entry) {
	t.Helper()
	f := newForge()
	e := shellcheck(t, f, "0.11.0", true)
	l, err := Resolve(ctx, f, f, "shellcheck", "0.11.0", e, false)
	if err != nil {
		t.Fatal(err)
	}
	return Lock{Tools: map[string]Locked{"shellcheck": l}}, e
}

func TestLockJSONIsStableAndReadsBack(t *testing.T) {
	lock, _ := locked(t)
	text := lock.JSON()
	if again := lock.JSON(); string(again) != string(text) || !strings.HasSuffix(string(text), "}\n") {
		t.Errorf("JSON is not stable, or has no final newline: %s", text)
	}
	back, err := ReadLock(text)
	if err != nil {
		t.Fatalf("ReadLock gave the error %v", err)
	}
	if string(back.JSON()) != string(text) {
		t.Errorf("ReadLock(JSON()) gave back %s", back.JSON())
	}
	if !strings.Contains(string(text), `"program_digest": "sha256:`) {
		t.Errorf("the lock names no program_digest: %s", text)
	}
}

func TestReadLockSaysTheLineOfWhatIsWrong(t *testing.T) {
	_, err := ReadLock([]byte("{\n  \"tools\": {\n    \"x\": {\"version\": 1}\n  }\n}\n"))
	if err == nil || !strings.HasPrefix(err.Error(), "kit.lock:3: ") {
		t.Errorf("ReadLock gave the error %v; want one at kit.lock:3", err)
	}
	if lock, err := ReadLock(nil); err != nil || len(lock.Tools) != 0 {
		t.Errorf("ReadLock of no file = %+v, %v; want an empty lock", lock, err)
	}
}

func finds(t *testing.T, findings []Finding, want ...string) {
	t.Helper()
	if len(findings) != len(want) {
		t.Fatalf("CheckLock found %+v; want %d findings", findings, len(want))
	}
	for i, w := range want {
		if !strings.Contains(findings[i].What, w) || findings[i].Fix == "" || findings[i].Why == "" {
			t.Errorf("finding %d is %+v; want one that says %q, why, and the fix", i, findings[i], w)
		}
	}
}

func TestCheckLockPassesALockThatHoldsEveryPin(t *testing.T) {
	lock, e := locked(t)
	list := map[string]Entry{"shellcheck": e}
	finds(t, CheckLock([]Pin{{Name: "shellcheck", Version: "0.11.0"}}, list, lock))
}

func TestCheckLockFindsWhatTheLockDoesNotHold(t *testing.T) {
	lock, e := locked(t)
	list := map[string]Entry{"shellcheck": e, "lychee": e}
	finds(t, CheckLock([]Pin{{Name: "lychee", Version: "1"}, {Name: "shellcheck", Version: "0.12.0"}}, list, lock),
		"kit.lock holds no lychee", "kit.lock holds shellcheck 0.11.0, and kit.yml pins 0.12.0")
}

func TestCheckLockFindsAToolNoListHolds(t *testing.T) {
	lock, e := locked(t)
	changed := e
	changed.Program = "elsewhere"
	finds(t, CheckLock([]Pin{{Name: "shellcheck", Version: "0.11.0"}, {Name: "mystery", Version: "1"}},
		map[string]Entry{"shellcheck": changed}, lock),
		"kit.lock was made from an entry of shellcheck that is no longer the one in use", "kit's list holds no mystery")
}

func TestCheckLockFindsAToolTheLockHoldsAndThePinsDoNot(t *testing.T) {
	lock, e := locked(t)
	finds(t, CheckLock(nil, map[string]Entry{"shellcheck": e}, lock), "kit.lock holds shellcheck, which kit.yml does not pin")
}

func TestCheckLockFindsAMissingPlatformOrDigest(t *testing.T) {
	lock, e := locked(t)
	l := lock.Tools["shellcheck"]
	delete(l.Files, "darwin/amd64")
	f := l.Files["linux/arm64"]
	f.ProgramDigest = ""
	l.Files["linux/arm64"] = f
	finds(t, CheckLock([]Pin{{Name: "shellcheck", Version: "0.11.0"}}, map[string]Entry{"shellcheck": e}, lock),
		"kit.lock holds no file of shellcheck for darwin/amd64", "kit.lock holds no digest of shellcheck's program for linux/arm64")
}

func TestTheRepositorysEntryWinsOverKits(t *testing.T) {
	lock, e := locked(t)
	kits := e
	kits.GitHub = "someone/else"
	pins := []Pin{{Name: "shellcheck", Version: "0.11.0", Entry: &e}}
	finds(t, CheckLock(pins, map[string]Entry{"shellcheck": kits}, lock))
	if got := EntryOf(pins[0], map[string]Entry{"shellcheck": kits}); got.GitHub != "koalaman/shellcheck" {
		t.Errorf("EntryOf gave %s's entry; want the repository's", got.GitHub)
	}
}
