# Packages

A package is a directory under `packages/` in a release, and everything the kit delivers comes from one. A repository
activates the packages it wants, and `kit install`, `kit update`, and `kit check` work on that active set.

## Activation

- `kit install [<package>…]` activates `kit`, the one default package, plus the named packages, the packages they
  depend on, and the members they include. It declares the set under `packages:` in `.kit/settings.yml`.
- `kit package list` shows every package the release has, which are active, and which are excluded.
- `kit package add <package>…` activates more packages, placing only the paths the new set adds to the old one.
- A glob names every package it matches, in `kit install`, `kit package add`, and `kit package remove`:
  `kit package add 'style/*'`. A glob that matches nothing is refused.
- `kit package remove <package>…` deactivates packages. It removes the paths only the removed packages delivered,
  and takes the kit's entries back out of any merged `.json` file. A package that an active package depends on is
  refused.
- A bundle is a package that includes others, such as `empowerite`. Adding it activates its members, and a member
  can be removed on its own: `kit package remove` lists it under `excludes:` in `.kit/settings.yml`, and
  `kit update` leaves it out. Naming it in `kit package add` ends the exclusion.
- `kit update` resolves the active set again from the new release, so a new member of a bundle, a new dependency,
  or a new default package arrives with the release. An excluded member stays out, unless a package depends on it,
  which the update says.

## Layout

```text
packages/<name>/
  package.yml          metadata
  files/<base>/       files mirrored at the base's root
  templates/<base>/   files rendered from the repository's answers, then mirrored the same way
  regions/<base>/     each file a region the kit owns inside a file the repository owns
```

A package must have `package.yml`. `files/`, `templates/`, and `regions/` are optional, and the kit reads nothing
else in the package's directory. The base sets where a file lands:

| Base | Root | Record |
| --- | --- | --- |
| `repo` | the repository root | the declaration in `.kit/settings.yml`, committed; `.kit/manifest`, written beside what kit placed under `.kit/` and ignored like it |
| `home` | the user's home directory, `~` | `~/.kit/manifest`, on the user's machine; a CI runner has none |

- `files/home/.kit/<name>` is a user-level setting. Each machine has one copy, from whichever repository installed
  or updated last.
- `kit check` compares home paths with the release where `~/.kit/manifest` exists. Where it does not, as on a
  runner, which installs nothing there, `kit check` reports that it did not check them.
- A package that names any other base is refused at install, and the error lists the known bases.
- `KIT_HOME` replaces the home directory. The tests use it to stay out of the real one.

## Metadata

`package.yml`, a rule's frontmatter, and `.kit/settings.yml` all use the small subset of YAML that
`kit/modules/bash/yaml.sh` reads: scalars, lists of `- item`, one level of map, and lists of records. A line
starting with `#` is a comment.

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
| `description` | Required. One line, shown by `kit package list`. |
| `default` | `true` makes the package active unless removed. In this release only `kit` sets it. |
| `depends` | Packages activated with this one. `kit package remove` refuses a package another active package depends on. |
| `includes` | The members of a bundle: packages activated with this one, each of which `kit package remove` can take out on its own. |
| `answers` | The default value of each template key. A repository overrides one under `answers:` in `.kit/settings.yml`. |
| `expects` | Lines the package expects in files the repository owns, each a `path`, a `pattern`, and a `remedy`; see below. |

[package.schema.json](../kit/packages/package.schema.json) is the schema, in JSON Schema 2020-12. This repository's gate
(its own CI command) validates every `package.yml` against it with `ajv`. Without a validator, `kit install` applies
the same contract and refuses, naming the problem:

- a key it does not know
- a missing `description`
- a `default` that is not `true` or `false`
- a dependency, or a member, the release does not have

## Delivery

Each kind of source file is placed differently:

| Source | How it is placed |
| --- | --- |
| `files/<base>/<path>` | At `<path>` under the base's root: merged by key for a `.json` file, copied otherwise. |
| `templates/<base>/<path>` | Rendered, then placed like a `files/` file. |
| `regions/<base>/<path>` | Rendered, then written as a region inside the file at `<path>`. |

### Copies and merges

- A destination ending in `.json` is merged by key, except a schema (`*.schema.json`). The kit sets its keys and
  adds its array entries. On update or removal it takes out only the previous release's entries, so the
  repository's own keys and entries stay.
- Every other destination under `.kit/` is materialized: a link to its source when the source is inside the
  worktree, as the repository's own packages and this repository's own release are, and otherwise a copy without
  the write bit. `.kit/.gitignore`, which kit writes, names each one, so git never sees them; it and `.kit/manifest`
  have no write bit either. `kit hydrate` brings them to that state from the declaration, and every verb that reads a
  delivered file does the same first.
- Every other destination outside `.kit/` is a copy.
- Where two active packages deliver the same path, the later package in dependency order wins.

### Includes

Where a file format supports includes, a package delivers a whole file of its own under `.kit/`. It then expects
the repository's file to include it with one line:

- lefthook's `extends`
- just's `import`
- git's `include`
- markdownlint's `extends`

The package declares each such line under `expects:`. An entry has a `path`, a `pattern` (an extended regular
expression that some line of that file must match), and a `remedy`.

```yaml
expects:
  - path: lefthook.yml
    pattern: '\.kit/lefthook\.yml'
    remedy: "put extends: [.kit/lefthook.yml] in lefthook.yml, so the kit's pre-commit commands run"
```

- `kit install` and `kit package add` list each unmet expectation as left to do.
- `kit check` fails until every expectation is met.
- The kit writes nothing into the file. It belongs entirely to the repository.

A single-quoted value is taken literally, backslashes included. Write patterns that way.

The kit changes a file it touches in one of four ways:

| When the file | The kit uses |
| --- | --- |
| has a format that supports includes | an include line the repository adds |
| has a format with comments | a region |
| supports neither | a merge by key |
| belongs entirely to the kit | a link or a readonly copy under `.kit/`, a copy elsewhere |

### Regions

A file under `regions/<base>/` is the content of a region inside the file at the same path. It is rendered first,
like a template. The kit writes three things in the file's comment syntax: the bare directive `kit:generate <path>`,
the content, and the marker `kit:end <path>`. [generate.md](generate.md) describes directives. The comment syntax
follows the file's extension:

- `<!-- … -->` for markdown, HTML, XML, and SVG
- `//` for JavaScript, TypeScript, Go, Rust, C, C++, Java, Kotlin, Swift, C#, and Scala
- `--` for SQL, Lua, and Haskell
- `#` for the other kinds the kit knows: shell, YAML, TOML, Python, Ruby, Perl, just, make, and plain text
- a file with no extension takes `#`; a file with any other extension cannot carry a region

The block is placed like this:

- The file has the directive: the content between the directive and the end marker becomes what `kit region`
  prints.
- The file exists without the directive: the block is appended after a blank line.
- The file does not exist: the block becomes the whole file.

After that:

- The repository can move the block anywhere in the file and write anything around it.
- `kit check` checks the region against its command.
- `kit update` regenerates the region and never merges it.
- Removal takes the block out, leaves the rest of the file, and removes the file if that leaves it empty.
- A file holds at most one kit region. Two packages that deliver a region to the same path are refused by name.
- A kind of file whose comment syntax the kit does not know, such as JSON, is refused.

### Templates

Files under `templates/<base>/` and `regions/<base>/` are rendered before they are placed. Each `@key@`
(autoconf's form) is replaced by the key's value. A template's result is then placed, digested, and merged exactly
like a `files/` file. Bash's `${var}` and GitHub's `${{ }}` cannot be mistaken for `@key@`, so a template can
contain both unchanged.

The values come from the whole of `.kit/settings.yml`:

- `@require_signing@` is a top-level setting.
- `@answers.ci_runner@` is an entry in a map.
- `@ci_runner@` is the entry under `answers:`.

A key with no value in `.kit/settings.yml` takes the package's default from its own `answers`. A key with neither
fails the install, and the error names the key and the package.

## Settings

`.kit/settings.yml` belongs to the repository. The kit writes three keys in it, `version`, `packages`, and `excludes`,
which declare the pinned release and the active packages, and reads the rest.

- Its schema is [settings.schema.json](../kit/packages/kit/files/repo/.kit/settings.schema.json), delivered beside it as
  `.kit/settings.schema.json`.
- An editor validates the file through the modeline `# yaml-language-server: $schema=settings.schema.json`.
- Before every install, update, and check, the kit checks the file against the schema's main points, and
  names any unknown key.

```yaml
answers:
  ci_runner: ubuntu-latest
```

| Key | Meaning |
| --- | --- |
| `version` | The kit release the repository is pinned to, written by `kit install` and `kit update`. A checkout of the kit itself declares `unreleased`. A manifest that still carries `version` and `packages` lines, from an earlier release, is read as the declaration until `kit update` moves it here. |
| `packages` | The active packages, in dependency order, written by `kit install`, `kit update`, and `kit package`. |
| `excludes` | Members of an active bundle removed with `kit package remove`, which `kit update` leaves out. Written by `kit package` and `kit update`. |
| `forge` | The forge provider the workflow commands use, `github` or `gitlab`. The forge is the service that hosts the repository. When absent, it is detected from origin's host. Set it for a self-hosted forge. |
| `gate` | What `kit gate` runs. When absent, the kit detects a justfile `ci` recipe, a Makefile `ci` target, or a `package.json` `ci` script. |
| `require_signing` | `true` makes `kit issue` refuse to start when git has no `user.signingkey`. |
| `answers` | The values a package's templates use, overriding the package's own defaults. |
| `toolchains` | The tool managers that put the gate's tools on `PATH`: `mise`, `brew`, or `docker`; see [toolchains.md](toolchains.md). |
| `policy` | The patterns a branch, a commit subject, and a change title must match; see [policy.md](policy.md). |
| `policy_engine` | `patterns` or `opa`; see [policy.md](policy.md). |
| `trackers` | Trackers other than the forge's own whose issues the workflow accepts, each with a prefix and a provider; see [trackers.md](trackers.md). |
| `bash_path` | The bash module path: the directories `use` sources a bash module from, in order, starting with `kit:modules/bash/`. A directory that does not exist is skipped; see [modules.md](modules.md#the-module-path). |

A provider setting, `forge`, `policy_engine`, or the `provider` of a tracker or a toolchain, takes one of the kit's
providers, or one the repository adds on its bash module path: `forge: bitbucket` with a
`modules/bash/forge-bitbucket.sh`. See [modules.md](modules.md#writing-a-bash-module).

## Packages a repository defines

A directory `.kit/packages/<name>/` in the repository is a package in every respect. It has the same parts:
`package.yml`, `files/<base>/`, `templates/<base>/`, and `regions/<base>/`.

- The kit looks up a package name in the release first, then in `.kit/packages/`.
- It is validated like a delivered package, and it may depend on delivered packages.
- Activate it with `kit install <name>` or `kit package add <name>`.
- `kit package list` shows it with its origin.
- Its paths are recorded in the manifest with their digests, so `kit check` can tell an edited installed copy
  from its source.
- On update, its files are compared with its own current source. An edit under `.kit/packages/` is never merged
  against a release.
- `kit context` finds a rule file it places under `.kit/rules/`, or a style file under `.kit/styles/`, like any
  other.

## This release's packages

| Package | Default | Depends on | Delivers |
| --- | --- | --- | --- |
| [`kit`](packages/kit.md) | yes | | `.kit/1st.md`, `.kit/README.md`, `.kit/rules/<kind>.md`, `.kit/settings.schema.json`; regions in `AGENTS.md` and `.gitignore` |
| [`style/bash`](packages/style.md) | no | `kit` | `.kit/styles/bash.md` and the `script` and `module` templates |
| [`style/markdown`](packages/style.md) | no | `kit` | `.kit/styles/markdown.md`, its four templates, and `.kit/markdownlint.jsonc`, the settings of its linter |
| [`style/just`](packages/style.md) | no | `kit` | `.kit/styles/just.md` and the `root` and `module` templates |
| [`style/workflows`](packages/style.md) | no | `kit` | `.kit/styles/workflows.md` and the `workflow` template |
| [`style/yaml`](packages/style.md) | no | `kit` | `.kit/styles/yaml.md` and the `document` template |
| [`style/json`](packages/style.md) | no | `kit` | `.kit/styles/json.md` |
| [`style/gitignore`](packages/style.md) | no | `kit` | `.kit/styles/gitignore.md` and the `file` and `region` templates |
| [`style/prose`](packages/style.md) | no | `kit` | `.kit/styles/prose.md`, which governs every file |
| [`hooks/lefthook`](packages/hooks/lefthook.md) | no | `kit` | `.kit/lefthook.yml`, a pre-commit hook running `kit lint` and `kit generate --check` |
| [`github/ci`](packages/github/ci.md) | no | `kit` | `.github/workflows/kit-ci.yml`, from a template, running `kit ci`; answer `ci_runner` |
| [`agents/claude`](packages/agents/claude.md) | no | `kit` | a region in `.claude/CLAUDE.md` importing `AGENTS.md`; `.claude/settings.json`, merged: the deny list and the `kit context --claude` hook |
| [`agents/cursor`](packages/agents/cursor.md) | no | `kit` | `.cursor/rules/kit-<kind>.mdc`, a rule per kind attached by the kit rule's paths, pointing at `.kit/rules/<kind>.md` |
| [`agents/copilot`](packages/agents/copilot.md) | no | `kit` | `.github/instructions/kit-<kind>.instructions.md`, an instruction file per kind applied to the kit rule's paths, pointing at the rule |
| [`agents/gemini`](packages/agents/gemini.md) | no | `kit` | `.gemini/settings.json`, merged: `AGENTS.md` among the context files, and the deny list as `run_shell_command` exclusions |
| [`policy/opa`](packages/policy/opa.md) | no | `kit` | `.kit/policy/kit.rego`, the naming policy in Rego, so `kit policy` runs on Open Policy Agent |
| [`empowerite`](packages/empowerite.md) | no | `kit`; includes every `style/*`, `github/ci`, `agents/claude`, and `hooks/lefthook` | nothing of its own: it is the bundle Empowerite's repositories take |

No package in this release delivers to `home`. Each package's page, linked above, describes what it places and why.
