// Package source reads kit for a screen: each read is a source, which keeps its answer, and asks again as the screen
// says. Every wait goes through a Clock, so a test fires the timers it means to, and none of its commands blocks on
// one.
package source

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Clock gives the command that brings msg once d has passed.
type Clock interface {
	After(d time.Duration, msg tea.Msg) tea.Cmd
}

// Ticks is the program's clock: each wait is a tea.Tick.
type Ticks struct{}

// After gives a tea.Tick of d that brings msg.
func (Ticks) After(d time.Duration, msg tea.Msg) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return msg })
}

// Env is what every read of a screen shares: the context that ends with the screen, how long one read may take, and
// the clock its waits go through, Ticks where none is given.
type Env struct {
	Ctx   context.Context
	Limit time.Duration
	Clock Clock
}

// after gives the command that brings msg once d has passed, on the env's clock.
func (e Env) after(d time.Duration, msg tea.Msg) tea.Cmd {
	if e.Clock == nil {
		return Ticks{}.After(d, msg)
	}
	return e.Clock.After(d, msg)
}
