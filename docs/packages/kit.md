# The kit package

The core package. It is active in every repository that uses the kit. It delivers:

- `.kit/1st.md`, the entry file every agent reads. The `agents/generic` package points the repository's
  `AGENTS.md` to it; see [agents/generic.md](agents/generic.md).
- `.kit/README.md`, the directory explained to the person at the desk: what is committed, what is placed, and how to
  change what kit adds.
- `.kit/kinds/<kind>.yml`, one file per kind, each with the kind's `name:`: the paths and first lines that make a
  file of that kind, and the kinds found inside its files, such as the bash in a justfile.
- `.kit/rules/<kind>.md`, one rule file per kind of file. A rule says what the work must do to be trusted. Each
  rule file's frontmatter names the kind it governs, and the linter and checks `kit lint` runs over its files.
- `.kit/settings.schema.json`, the schema of the repository's `.kit/settings.yml`.
- A region in `.gitignore`: the `/worktrees/` entry the workflow needs.

The region starts at a `kit:generate` directive, and `kit generate` fills it through `kit region`. The rest of
`.gitignore` belongs to the repository.

The kit package delivers no style. How a file looks, and the templates that hold its shape, come from the style
packages a repository takes; see [style.md](style.md).

Agents and people find the rules through `kit context`, described in
[docs/context.md](../context.md).

To add a kind of file, add one kind file and one rule file here, and one row to the table in `1st.md`.
