// A load is a read whose answer a screen shows, read again as the screen asks: a read started later supersedes the
// one before, which it ends, and whose answer is dropped. While the first read runs, the copy kit kept of the last
// answer stands in, stale. Once a load has shown an answer, the next read of the marks gives it its mark; a mark that
// differs from it later says what the load shows has changed, so it is stale until it is read again, at once when it
// is watched.
package source

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

// Load is a read a screen shows: whether no answer shows yet, as the first read runs or a refused one runs again;
// whether what shows is not the latest, a kept copy, or changed since, as its mark says; and whether it is watched,
// so it is read again once its mark moves.
type Load[T any] struct {
	Loading bool
	Stale   bool
	Watch   bool
	name    string
	read    func(ctx context.Context, fresh bool) (T, error)
	kept    func(ctx context.Context) (T, error)
	gen     int                // the latest read's: an answer from an earlier one is dropped
	ctx     context.Context    // the latest read's, inside the screen's; nil for the first, which has the screen's
	stop    context.CancelFunc // ends the latest read, when another starts
	mark    string             // the mark the load took after it last answered
	remark  bool               // whether it answered since the last read of the marks, so it takes the next as its own
}

// Answer is what a load's read brought: the load's name, which no other source of its screen has; the answer, or why
// there is none; the read it answers, from 0, the first; and whether it is the copy kit kept.
type Answer[T any] struct {
	Name  string
	Value T
	Err   error
	Gen   int
	Kept  bool
}

// NewLoad gives the load named name, loading: read answers it, afresh where fresh says, skipping what kit shares
// between reads; kept gives what kit kept of its last answer, or is nil where kit keeps none.
func NewLoad[T any](name string, read func(context.Context, bool) (T, error),
	kept func(context.Context) (T, error)) Load[T] {
	return Load[T]{Loading: true, name: name, read: read, kept: kept}
}

// Name gives the load's name.
func (l Load[T]) Name() string { return l.name }

// Gen gives the latest read's number, from 0, the first.
func (l Load[T]) Gen() int { return l.gen }

// Mark gives the mark the load took after it last answered; none before.
func (l Load[T]) Mark() string { return l.mark }

// Read gives the latest read, afresh or not, within the env's limit.
func (l Load[T]) Read(env Env, fresh bool) tea.Cmd {
	name, read, ctx, limit, gen := l.name, l.read, env.Ctx, env.Limit, l.gen
	if l.ctx != nil {
		ctx = l.ctx
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		value, err := read(ctx, fresh)
		return Answer[T]{Name: name, Value: value, Err: err, Gen: gen}
	}
}

// Kept gives the read of what kit kept of the last answer, for the latest read; nil where kit keeps none.
func (l Load[T]) Kept(env Env) tea.Cmd {
	if l.kept == nil {
		return nil
	}
	name, kept, ctx, limit, gen := l.name, l.kept, env.Ctx, env.Limit, l.gen
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		value, err := kept(ctx)
		return Answer[T]{Name: name, Value: value, Err: err, Gen: gen, Kept: true}
	}
}

// Again gives a new latest read, afresh or not, ending the one before when it still runs; what the load shows stays
// until the answer replaces it.
func (l *Load[T]) Again(env Env, fresh bool) tea.Cmd {
	if l.stop != nil {
		l.stop()
	}
	l.ctx, l.stop = context.WithCancel(env.Ctx)
	l.gen++
	return l.Read(env, fresh)
}

// Landed says what an answer did to a load.
type Landed int

const (
	// Elsewhere is an answer of another load, or a message that is none.
	Elsewhere Landed = iota
	// Dropped is an answer the load no longer wants: an earlier read's, or a kept copy that came after an answer, or
	// failed.
	Dropped
	// Shown is the latest read's answer, which now shows, the load neither loading nor stale.
	Shown
	// Stands is a kept copy that came while the first read runs, for the screen to show, stale, as it can.
	Stands
)

// Take takes a message that may be the load's answer, gives it, and says what it did.
func (l *Load[T]) Take(msg tea.Msg) (Answer[T], Landed) {
	a, ok := msg.(Answer[T])
	switch {
	case !ok || a.Name != l.name:
		return a, Elsewhere
	case a.Gen != l.gen:
		return a, Dropped
	case a.Kept && (!l.Loading || a.Err != nil):
		return a, Dropped
	case a.Kept:
		return a, Stands
	}
	l.Loading, l.Stale, l.remark = false, false, true
	return a, Shown
}

// Marked takes the load's mark as the latest read of the marks gives it, and says whether to read it again: a load
// that answered since the read before takes the mark as its own; one whose mark moved since is stale, and, watched,
// takes the new mark, to be read again at once; one still loading is passed over.
func (l *Load[T]) Marked(mark string) bool {
	switch {
	case l.Loading:
	case l.mark == "" || l.remark:
		l.mark, l.remark = mark, false
	case mark != l.mark && l.Watch:
		l.Stale, l.mark = true, mark
		return true
	case mark != l.mark:
		l.Stale = true
	}
	return false
}
