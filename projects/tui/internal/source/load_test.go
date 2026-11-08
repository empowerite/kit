// A load shows its latest read's answer, a later read superseding and ending an earlier; the copy kit kept stands in
// while the first read runs; and the marks say when what it shows has changed, a watched load read again at once.
package source

import (
	"context"
	"errors"
	"testing"
	"time"
)

// env is a test's env: a second's limit for each read.
func env() Env { return Env{Ctx: context.Background(), Limit: time.Second} }

// answering gives a load's read that answers what it is told, afresh or not, and its kept copy, "kept".
func answering(t *testing.T) Load[string] {
	t.Helper()
	read := func(ctx context.Context, fresh bool) (string, error) {
		if fresh {
			return "fresh", ctx.Err()
		}
		return "read", ctx.Err()
	}
	return NewLoad("l", read, func(context.Context) (string, error) { return "kept", nil })
}

func TestALoadShowsItsLatestReadsAnswer(t *testing.T) {
	l := answering(t)
	if !l.Loading || l.Stale || l.Name() != "l" || l.Gen() != 0 {
		t.Fatalf("a new load: %+v", l)
	}
	first := l.Read(env(), false)()
	// a read started again supersedes the first, and ends it
	again := l.Again(env(), true)
	if l.Gen() != 1 {
		t.Fatalf("again: gen %d", l.Gen())
	}
	if a, landed := l.Take(first); landed != Dropped || a.Value != "read" || !l.Loading {
		t.Errorf("the first read's answer: %v %+v", landed, a)
	}
	ended := l.ctx
	latest := again()
	if a, landed := l.Take(latest); landed != Shown || a.Value != "fresh" || a.Gen != 1 || l.Loading || l.Stale {
		t.Errorf("the latest read's answer: %v %+v, load %+v", landed, a, l)
	}
	l.Again(env(), false)
	if ended.Err() == nil {
		t.Error("a read started again did not end the one before")
	}
	for _, msg := range []any{Answer[string]{Name: "other"}, Answer[int]{Name: "l"}, "nothing"} {
		if _, landed := l.Take(msg); landed != Elsewhere {
			t.Errorf("%#v: %v", msg, landed)
		}
	}
}

func TestAKeptCopyStandsInWhileTheFirstReadRuns(t *testing.T) {
	l := answering(t)
	kept := l.Kept(env())()
	if a, landed := l.Take(kept); landed != Stands || a.Value != "kept" || !a.Kept {
		t.Fatalf("the kept copy: %v %+v", landed, a)
	}
	// the screen shows it, stale; then the read lands, and a kept copy after it is dropped
	l.Loading, l.Stale = false, true
	if _, landed := l.Take(l.Read(env(), false)()); landed != Shown || l.Stale {
		t.Errorf("the read after the kept copy: %v, stale %v", landed, l.Stale)
	}
	if _, landed := l.Take(kept); landed != Dropped {
		t.Errorf("a kept copy after the read: %v", landed)
	}
	// a kept copy kit could not read is dropped; a load without one asks for none
	failed := NewLoad("l", nil, func(context.Context) (string, error) { return "", errors.New("none") })
	if _, landed := failed.Take(failed.Kept(env())()); landed != Dropped {
		t.Errorf("a kept copy that failed: %v", landed)
	}
	if NewLoad[string]("l", nil, nil).Kept(env()) != nil {
		t.Error("a load kit keeps nothing of asked for a kept copy")
	}
}

func TestTheMarksSayWhenWhatALoadShowsHasChanged(t *testing.T) {
	l := answering(t)
	// still loading, it takes no mark
	if l.Marked("m1") || l.Mark() != "" {
		t.Fatalf("loading: mark %q", l.Mark())
	}
	l.Take(l.Read(env(), false)())
	if l.Marked("m1") || l.Mark() != "m1" || l.Stale {
		t.Fatalf("after its answer: mark %q, stale %v", l.Mark(), l.Stale)
	}
	if l.Marked("m1") || l.Stale {
		t.Error("the same mark made it stale")
	}
	// a mark that moves makes it stale, and keeps the mark it took
	if l.Marked("m2") || !l.Stale || l.Mark() != "m1" {
		t.Errorf("a moved mark: stale %v, mark %q", l.Stale, l.Mark())
	}
	// watched, it takes the new mark and is read again
	l.Watch = true
	if !l.Marked("m3") || l.Mark() != "m3" {
		t.Errorf("watched: mark %q", l.Mark())
	}
	// once read again, it takes the next mark as its own, whatever it is
	l.Take(l.Again(env(), true)())
	if l.Marked("m9") || l.Mark() != "m9" || l.Stale {
		t.Errorf("after it was read again: mark %q, stale %v", l.Mark(), l.Stale)
	}
}
