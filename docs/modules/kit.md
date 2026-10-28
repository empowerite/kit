# The kit module

The core module. It is active in every repository that uses the kit. It delivers:

- `.kit/1st.md`, the entry file every agent reads. The repository's `AGENTS.md` points to it.
- `.kit/rules/<kind>.md`, one rule file per kind of file. Each rule file's frontmatter declares the paths and
  first lines it governs, and the linter `kit lint` runs over them.
- `.kit/templates/<rule>/<variant>`, the templates `kit new file` makes a file from and `kit lint` holds files to.
- `.kit/settings.schema.json`, the schema of the repository's `.kit/settings.yml`.
- `.kit/markdownlint.jsonc`, the markdown rule's linter settings.
- A region in `AGENTS.md`: the sentence that sends every agent to `.kit/1st.md`.
- A region in `.gitignore`: the `/worktrees/` entry the workflow needs.

Each region starts at a `kit:generate` directive, and `kit generate` fills it through `kit region`. The rest of
`AGENTS.md` and `.gitignore` belongs to the repository.

Agents and people find the rules through `kit context`, described in
[docs/context.md](../context.md).

To add a kind of file, add one rule file here and one row to the table in `1st.md`.
