# The kit package

The core package, and the default: `kit install` with no names brings it. `kit packages remove kit` takes it out
once no other active package depends on it. It is then listed under `excludes:` in `kit.yml`, so `kit update` leaves
it out. It delivers:

- `.kit/1st.md`, the entry file every agent reads. The `agents/generic` package points the repository's
  `AGENTS.md` to it; see [agents/generic.md](agents/generic.md).
- `.kit/README.md`, the directory explained to the person at the desk: what is committed, what is placed, and how to
  change what kit adds.
- `.kit/welcome/`, the pages of `kit welcome`, one per step: the intro, before toolchains, toolchains, setup, the
  lay of the land, the terms, and how to make the kit yours. A house package replaces a page by placing one of the
  same name, or adds one.
- `.kit/kinds/<kind>.yml`, one file per kind, each with the kind's `name:`: the paths and first lines that make a
  file of that kind, and the kinds found inside its files, such as the bash in a justfile.
- `.kit/rules/<kind>.md`, a rule file for each kind that has one: bash, gitignore, jq, json, just, markdown, and
  workflows. `any`, `text`, and `yaml` have none of their own; `kit context` prints the rules of every kind a file
  is of. A rule says what the work must do to be trusted. Its frontmatter names the kind it governs, and the linter
  and checks `kit lint` runs over its files.
- `.kit/kit.schema.json`, the schema of the repository's `kit.yml`.
- A region in `.gitignore`: the `/worktrees/` entry the workflow needs.

The region starts at a `kit:generate` directive, and `kit generate` fills it through `kit region`. The rest of
`.gitignore` belongs to the repository.

The kit package does not deliver any style. How a file looks, and the templates that hold its shape, come from the style
packages a repository takes; see [style.md](style.md). The one exception is the five provider templates the bash
rule lists, `.kit/templates/bash/<seam>-provider`. They hold a seam's provider to its seam's calls in every
repository, whatever style it takes.

Agents and people find the rules through `kit context`, described in [docs/context.md](../context.md).

To add a kind of file:

- add a kind file here
- add a rule file when the kind has something the work must do to be trusted
- add one row to the table in `1st.md` when it has a rule
