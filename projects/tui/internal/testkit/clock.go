// A test's clock keeps each wait it is asked for and brings none until the test fires it, so no command a test
// drives blocks on a wait, and a test sees a timer was set without waiting it out.
package testkit

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// Timer is a wait a clock was asked for: how long, and what it brings.
type Timer struct {
	After time.Duration
	Msg   tea.Msg
}

// Clock is a test's clock: the timers set and not yet fired, in the order they were set. Its methods take a pointer,
// so every copy of a model that holds it shares one.
type Clock struct{ timers []Timer }

// After keeps the wait, and gives no command: nothing comes until Fire.
func (c *Clock) After(d time.Duration, msg tea.Msg) tea.Cmd {
	c.timers = append(c.timers, Timer{d, msg})
	return nil
}

// Timers gives the timers set and not yet fired.
func (c *Clock) Timers() []Timer { return c.timers }

// Fire gives what every timer set brings, in the order they were set, and forgets them.
func (c *Clock) Fire() []tea.Msg {
	var msgs []tea.Msg
	for _, t := range c.timers {
		msgs = append(msgs, t.Msg)
	}
	c.timers = nil
	return msgs
}
