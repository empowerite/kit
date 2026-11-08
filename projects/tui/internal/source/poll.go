// A poll is a read that the screen asks for when it likes, and that asks again by itself a while after each answer.
// Its answer stands until the next. A read that fails keeps the last answer and goes on asking; the first to fail
// before any answered stops the asking, until the screen resets it, as Refresh All does.
package source

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Poll is a read asked again a while after each answer: its last answer, and whether it has had one.
type Poll[T any] struct {
	Value  T
	Got    bool
	name   string
	every  time.Duration
	read   func(context.Context) (T, error)
	asking bool
	off    bool
	serial int
}

// Took says what a message did to a poll.
type Took int

const (
	// NotMine is a message of another source, or none.
	NotMine Took = iota
	// Answered is a read's answer, which the poll now holds.
	Answered
	// Failed is a read that failed, the answer before it kept.
	Failed
	// Ticked is the poll's while passed, the read asked again when the wait was the latest.
	Ticked
)

// answer is what a poll's read brought: the poll's name, the answer, and why there is none.
type answer[T any] struct {
	name  string
	value T
	err   error
}

// tick says a poll's while has passed since the answer serial names.
type tick struct {
	name   string
	serial int
}

// NewPoll gives the poll named name, which no other source of its screen has, that read answers, asked again every
// while after each answer.
func NewPoll[T any](name string, every time.Duration, read func(context.Context) (T, error)) Poll[T] {
	return Poll[T]{name: name, every: every, read: read}
}

// Asking says whether a read is under way.
func (p Poll[T]) Asking() bool { return p.asking }

// Off says whether the first read failed, so the poll asks again only once it is reset.
func (p Poll[T]) Off() bool { return p.off }

// Reset has a poll that stopped asking ask again.
func (p *Poll[T]) Reset() { p.off = false }

// Ask gives the read, unless one is under way, or the poll is off; nil then.
func (p *Poll[T]) Ask(env Env) tea.Cmd {
	if p.asking || p.off {
		return nil
	}
	p.asking = true
	name, read, ctx, limit := p.name, p.read, env.Ctx, env.Limit
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		value, err := read(ctx)
		return answer[T]{name, value, err}
	}
}

// Take takes a message that may be the poll's, and says what it did: an answer is kept, or, failed, the one before
// it, and the poll waits its while to ask again, unless it failed before any answered, which turns it off; a while
// that passed asks again, unless an answer came since the one it waited from.
func (p *Poll[T]) Take(env Env, msg tea.Msg) (tea.Cmd, Took) {
	switch msg := msg.(type) {
	case answer[T]:
		if msg.name != p.name {
			return nil, NotMine
		}
		p.asking = false
		took := Answered
		switch {
		case msg.err == nil:
			p.Value, p.Got = msg.value, true
		case !p.Got:
			p.off = true
			return nil, Failed
		default:
			took = Failed
		}
		p.serial++
		return env.after(p.every, tick{p.name, p.serial}), took
	case tick:
		if msg.name != p.name {
			return nil, NotMine
		}
		if msg.serial != p.serial {
			return nil, Ticked
		}
		return p.Ask(env), Ticked
	}
	return nil, NotMine
}
