// Each forge's rate limit on the caller, at the right end of the keys bar: the limit it has least of, as a share of
// what it allows, after the forge's glyph. Dim while plenty is left; under a fifth, yellow, with the time it resets;
// spent, red. The screen asks kit for them after a stage loads, since a load spends them, and once a minute after it
// first reads them; a kit that cannot read them is asked again only on Refresh All.
package flow

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/empowerite/kit/projects/tui/internal/kit"
	"github.com/empowerite/kit/projects/tui/internal/style"
)

// QuotaEvery is how often the screen reads the rate limits again, once it has read them, while nothing loads.
const QuotaEvery = time.Minute

// quotaEvery is how long the screen waits for QuotaEvery: QuotaEvery, as a test waits less.
var quotaEvery = QuotaEvery

// quotaMsg is kit's answer to the rate limits: each forge's, or why they could not be read.
type quotaMsg struct {
	quotas []kit.Quota
	err    error
}

// quotaTickMsg says QuotaEvery has passed since the read the serial names.
type quotaTickMsg struct{ serial int }

// askQuota asks kit for the rate limits, unless a read is under way, or kit could not read them, until Refresh All.
func (m *Model) askQuota() tea.Cmd {
	if m.quoting || m.quotaOff {
		return nil
	}
	m.quoting = true
	r, ctx, limit := m.runner, m.ctx, m.limit
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		quotas, err := r.Quota(ctx)
		return quotaMsg{quotas, err}
	}
}

// tookQuota keeps what kit said of the rate limits, and has the screen read them again in QuotaEvery. A read that
// fails keeps what the last said; the first to fail before any read stops the asking, until Refresh All.
func (m *Model) tookQuota(msg quotaMsg) tea.Cmd {
	m.quoting = false
	switch {
	case msg.err == nil:
		m.quotas = msg.quotas
	case m.quotas == nil:
		m.quotaOff = true
		return nil
	}
	m.quotaSerial++
	serial := m.quotaSerial
	return tea.Tick(quotaEvery, func(time.Time) tea.Msg { return quotaTickMsg{serial} })
}

// forgeGlyphs gives each forge's glyph, as the source column draws it.
var forgeGlyphs = map[string]string{"github": GitHub, "gitlab": GitLab}

// quotaText draws each forge's scarcest limit, the forges apart by two cells, at the time now: remaining/limit after
// the forge's glyph, or its name where it has none. A forge without a limit draws nothing.
func quotaText(quotas []kit.Quota, now time.Time) string {
	var out []string
	for _, q := range quotas {
		l, ok := scarcest(q.Limits)
		if !ok {
			continue
		}
		glyph := forgeGlyphs[q.Forge]
		if glyph == "" {
			glyph = q.Forge
		}
		text := fmt.Sprintf("%s %d/%d", glyph, l.Remaining, l.Limit)
		switch {
		case l.Remaining <= 0:
			text = style.Refusal(text + resets(l, now))
		case l.Remaining*5 < l.Limit:
			text = style.Warn(text + resets(l, now))
		default:
			text = style.Off(text)
		}
		out = append(out, text)
	}
	return strings.Join(out, "  ")
}

// scarcest gives the limit that has least left of what it allows; none for a forge without a limit that allows any.
func scarcest(limits []kit.Limit) (kit.Limit, bool) {
	var best kit.Limit
	found := false
	for _, l := range limits {
		if l.Limit <= 0 {
			continue
		}
		if !found || float64(l.Remaining)/float64(l.Limit) < float64(best.Remaining)/float64(best.Limit) {
			best, found = l, true
		}
	}
	return best, found
}

// resets says when a limit resets, in local time, after ↻; nothing for one whose forge does not say, or that has.
func resets(l kit.Limit, now time.Time) string {
	if l.Reset <= 0 || l.Reset <= now.Unix() {
		return ""
	}
	return " ↻" + time.Unix(l.Reset, 0).Format("15:04")
}
