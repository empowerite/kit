# The agents/generic package

Adapts the kit for any agent that reads `AGENTS.md`, which most harnesses do on their own: Codex, Cursor, GitHub
Copilot, Gemini CLI when told to, and Claude Code through an import. The package delivers one region in `AGENTS.md`,
the sentence that sends the agent to `.kit/1st.md`. The region starts at a `kit:generate` directive, and
`kit generate` fills it. The rest of `AGENTS.md` belongs to the repository.

Take this package alone for an agent the kit has no package for. Every `agents/<harness>` package whose harness
reads `AGENTS.md` depends on it, so taking one of them takes this one as well:

- [`agents/claude`](claude.md)
- [`agents/cursor`](cursor.md)
- [`agents/copilot`](copilot.md)
- [`agents/gemini`](gemini.md)

A repository that takes no agents package has no `AGENTS.md` region. Its agents still find the rules through
`kit context`, as people do.
