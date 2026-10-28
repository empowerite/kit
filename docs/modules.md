# Modules

Everything the kit delivers is a module: a directory under `modules/` in the release. A repository activates the
modules it wants, and `kit install`, `kit update`, and `kit check` work over the active set.

## Layout

```text
modules/<name>/
  module.yml          metadata
  files/<base>/       files mirrored at the base's root
  templates/<base>/   files rendered from the repository's answers, then mirrored the same way
  regions/<base>/     each file a region the kit owns inside a file the repository owns
  docs/               the module's documentation; packed with the release, placed in no repository
```

A module may have any of `files/`, `templates/`, `regions/`, and `docs/`; `module.yml` it must have. A base names
where a file lands:

| Base | Root | Record |
| --- | --- | --- |
| `repo` | the repository root | `.kit/manifest`, committed with the repository |
| `home` | the user's home directory, `~` | `~/.kit/manifest`, the desk's own; a runner has none |

So `files/home/.kit/<name>` is a user-level setting, delivered once per desk by whichever repository installs or
updates last. `kit check` holds home paths to the release where `~/.kit/manifest` exists and says so where it does
not, since a runner installs nothing there. A module naming any other base is refused at install with the known
bases listed. `KIT_HOME` stands in for the home directory, which is how the tests keep out of the real one.

## Metadata

`module.yml` is the small subset of YAML `bin/yaml.sh` reads: scalars, lists of `- item`, and one level of map.
A line starting with `#` is a comment.

```yaml
description: "One GitHub Actions job that runs kit ci"
default: true
depends:
  - kit
answers:
  ci_runner: ubuntu-latest
```

| Key | Meaning |
| --- | --- |
| `description` | One line, shown by `kit module list`. |
| `default` | `true` for a module active unless removed: `kit`. |
| `depends` | Modules activated with this one. `kit module remove` refuses a module another active module depends on. |
| `answers` | The default value of each template key. A repository overrides one under `answers:` in `.kit/settings.yml`. |
| `expects` | Lines the module expects in files the repository owns, each a `path`, a `pattern`, and a `remedy`; see below. |

[module.schema.json](../modules/module.schema.json) is the schema, JSON Schema 2020-12. This repository's gate validates
every `module.yml` against it with `ajv`, and `kit install` holds a module to the same contract where no validator is: a
key it does not know, a `default` that is not `true` or `false`, or a dependency the release lacks is refused, naming
it.

## Delivery

A file under `files/<base>/` is placed at the same path under the base's root. A destination ending in `.json`, other
than a schema, `*.schema.json`, is merged by key: the kit's keys are set, its array entries added, and on update or
removal only the previous release's entries are taken out, so a repository's own keys and entries survive. Every other
destination is a copy. Where two active modules deliver the same path, the later module in dependency order wins.

Where a format composes, a module delivers a whole file of its own under `.kit/` and expects the repository's file to
take it with one include line: lefthook's `extends`, just's `import`, git's `include`, markdownlint's `extends`. The
module says so under `expects:`, a `path`, an extended regular expression some line of that file must match, and a
`remedy`. `kit install` and `kit module add` name each one not yet met as left to do, `kit check` refuses until it is,
and the kit writes nothing into the file, which stays the repository's entirely. The rule for any file the kit touches
is then: an include where the format composes, a region where it has comments, a merge by key where it has neither, and
a copy only where the file is the kit's own.

```yaml
expects:
  - path: lefthook.yml
    pattern: '\.kit/lefthook\.yml'
    remedy: "put extends: [.kit/lefthook.yml] in lefthook.yml, so the kit's pre-commit commands run"
```

A single-quoted value is taken literally, backslashes and all, which is how a pattern is written.

A file under `regions/<base>/` is the content of a region inside the file at the same path, rendered as a template
is. What is placed there is the bare directive `kit:generate <path>`, the content, and the marker `kit:end <path>`, in
the file's own comment syntax: `<!-- … -->` in markdown, `#` elsewhere, as [generate.md](generate.md) describes. Where
the file carries the directive already, the content between it and the end marker is made what `kit region` prints;
where it does not, the block is appended after a blank line; where there is no file, it is made of the block alone. The
repository may move the block anywhere in the file and write what it likes around it. `kit check` holds the region to
its command, `kit update` regenerates it, never merges it, and removal takes the block out and leaves the rest, removing
a file it leaves empty. One file holds one kit region, and two modules delivering a region to the same path are refused
by name. A kind of file the kit cannot mark is refused. The `kit` module places two: the sentence in `AGENTS.md` that
sends every agent to `.kit/1st.md`, and the `/worktrees/` entry in `.gitignore`.

A file under `templates/<base>/`, and one under `regions/<base>/`, is rendered first. Every `@key@` in it, autoconf's
form, is replaced by the key's value, then the result is placed, digested, and merged exactly as a `files/` file is.
The form is one neither bash's `${var}` nor GitHub's `${{ }}` can be mistaken for, so a template may carry both
untouched. The context is the whole of `.kit/settings.yml`: a top-level setting as `@require_signing@`, a map's entry
as `@answers.ci_runner@`, and a bare `@ci_runner@` meaning the entry under `answers:`. A key with no value there takes
the module's default under its own `answers`, and a key with neither refuses the install, naming the key and the module.

The same YAML subset is what a rule's frontmatter and `.kit/settings.yml` are written in, read by `bin/yaml.sh` in
all three places, so a rule's lists, a module's `depends:`, and the settings have one reader among them.

## Settings

`.kit/settings.yml` is the repository's, read by the kit and never written by it, in the same YAML subset.
[settings.schema.json](../modules/kit/files/repo/.kit/settings.schema.json) is its schema, delivered beside it as
`.kit/settings.schema.json`, so an editor validates the file through the modeline
`# yaml-language-server: $schema=settings.schema.json`; the kit holds the file to the schema's main points before
every install, update, and check, naming an unknown key.

```yaml
answers:
  ci_runner: ubuntu-latest
```

| Key | Meaning |
| --- | --- |
| `forge` | The provider the loop's verbs use, `github` or `gitlab`; detected from origin's host when absent, and named here for a self-hosted forge. |
| `gate` | What `kit gate` runs; a justfile `ci` recipe, a Makefile `ci` target, or a `package.json` `ci` script is detected when absent. |
| `require_signing` | `true` makes `kit issue` refuse a desk without a signing key. |
| `answers` | The values a module's templates take, overriding the module's own defaults. |
| `trackers` | The trackers besides the forge's own whose issues the loop takes, a prefix and a provider each; see [trackers.md](trackers.md). |

## Modules a repository defines

A directory `.kit/modules/<name>/` in the repository is a module in every respect: `module.yml`, `files/<base>/`,
`templates/<base>/`, `regions/<base>/`, and `docs/`. A name is looked up in the release first, then there. It is
validated like a delivered module, may depend on delivered ones, is activated with `kit install <name>` or `kit module
add <name>`, and `kit module list` shows it with its origin. Its paths are recorded in the manifest with their digests,
so `kit check` tells an edited installed copy from its source, and on update its files are compared with its own current
source, so an edit under `.kit/modules/` is never merged against a release. A rule file it places under `.kit/rules/` is
found by `kit context` like any other.

## Activation

- `kit install [<module>…]` activates `kit`, the one default module, and the named ones, with their dependencies,
  and records the set on the `modules` line of `.kit/manifest`.
- `kit module list` shows every module the release carries and which are active.
- `kit module add <module>…` activates more, placing only the paths the new set delivers beyond the old.
- `kit module remove <module>…` deactivates, removing the paths only the removed modules delivered and taking the
  kit's entries back out of a merged `.json`.

## This release's modules

| Module | Default | Depends on | Delivers |
| --- | --- | --- | --- |
| `kit` | yes | | `.kit/1st.md`, `.kit/rules/<kind>.md`, `.kit/settings.schema.json`; regions in `AGENTS.md` and `.gitignore` |
| `lefthook` | no | `kit` | `.kit/lefthook.yml`, a pre-commit hook running `kit lint` |
| `ci-github` | no | `kit` | `.github/workflows/kit-ci.yml`, from a template, running `kit ci`; answer `ci_runner` |
| `claude` | no | `kit` | a region in `.claude/CLAUDE.md` importing `AGENTS.md`; `.claude/settings.json`, merged: the deny list and the `kit context --claude` hook |

No module in this release delivers to `home`. Each module's `docs/README.md` says what it places and why.
