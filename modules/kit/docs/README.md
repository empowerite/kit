# The kit module

The core, active in every repository that takes the kit. It places:

- `.kit/1st.md`, the entry every agent reads, imported by the repository's `AGENTS.md`.
- `.kit/rules/<kind>.md`, one rule file per kind of file, each declaring in its frontmatter which paths and which
  first lines it governs.

The rules are discovered through `kit context <path>`, described in [docs/context.md](../../../docs/context.md),
and copied into no harness's directory. Adding a kind is adding one rule file here and one row to the table in
`1st.md`; every harness and every person finds it the same way.
