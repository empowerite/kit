<!-- kit:generated -- this directory, explained to the person at the desk, from the kit package -->
<!-- see: "Changing what kit adds" in .kit/1st.md, to replace it -->
# This directory

`.kit/` is where [empowerite/kit](https://github.com/empowerite/kit) keeps what it brings to this repository: the
rules and styles that agents and linters hold files to, the templates new files are made from, and the repository's
own packages. Which release and which packages the repository takes is declared in `kit.yml` at the root. This page
is for the person at the desk. Agents read [1st.md](1st.md) instead, and `AGENTS.md` at the root sends them there.

## What git tracks here

One thing is committed here, and only this: `packages/`, packages of the repository's own, in the release's shape.

The declaration, `kit.yml`, is committed at the root. `version:` names the kit release, `packages:` the active
packages, and `excludes:` the members of a bundle the repository leaves out. `answers:` fills in the values templates
ask for. Every other key is a setting of the repository's own. `kit.schema.json`, placed here, is its schema.

## What kit places here

Everything else is placed by `kit hydrate` from the declaration. `.gitignore` in this directory, which kit writes,
names every placed file, so git never sees one. Each placed file is one of:

- a symbolic link to its source, when the source is inside this worktree, as the repository's own packages are.
  A change at the source shows in `git status` and reaches a commit. A change through the link is that change.
- a copy without the write bit, when the source is the installed release, outside the worktree.

`manifest` records the digest of every placed file, so `kit update` can tell the release's file from one that was
edited. It and `.gitignore` are kit's too, and have no write bit either.

A fresh clone or worktree has none of these until something runs `kit hydrate`. Every kit command that reads a
rule runs it first, `kit issue` runs it in the worktree it makes, and a `mise.toml` with the kit's `enter` hook runs
it on `cd`. A file deleted, a link an editor's save replaced, or a copy given its write bit back is set right the
same way.

## Commands you run here

| Command | What it does |
| --- | --- |
| `kit hydrate` | Places what the declaration names, and says what it changed. |
| `kit check` | Reports every placed file that differs from the pinned release. It heals nothing. |
| `kit update` | Brings the repository to the release on `PATH`, after the pin in `mise.toml` changes. |
| `kit package add <package>…` | Takes a package; `kit package remove` gives one up; `kit packages` lists them. |
| `kit context <path>` | Prints the rules and the style that govern a file. |
| `kit lint [<path>…]` | Runs each rule's linter over the staged files, or over the files named. |
| `kit help` | Lists every command, and `kit help <verb>` prints one command's page. |

## Changing what kit adds

A placed file cannot be edited where it lies, and `kit check` fails when one differs from the release. To stop
taking a file, remove its package. To replace one, write a package under `packages/` that depends on the package
adding the file and places a file at the same path: the later package wins. To fill in a value a template asks for,
set it under `answers:` in `kit.yml` at the root. The steps are in [1st.md](1st.md), under "Changing what kit adds".

## Rules, styles, and templates

`kinds/` defines each kind of file: the paths and first lines that make a file of that kind, and the kinds found
inside its files, such as the bash in a justfile. `rules/` holds one rule per kind: what the kit holds a file of
that kind to, and the linter that checks it. `styles/` holds the styles the repository takes, one per kind, and
`templates/` the shapes `kit new file` makes a file from. A rule or a style names the kinds it governs in its
frontmatter, and `kit context <path>` prints the ones that match, which is how an agent, a hook, and `kit lint`
all find the same rules for the same file.

How the kit works, and how to install it, is in the kit's own README, at the link above.
