# Modules

A module is a directory under `modules/` in a release, and everything the kit delivers comes from one. A repository
activates the modules it wants, and `kit install`, `kit update`, and `kit check` work on that active set.

## Activation

- `kit install [<module>…]` activates `kit`, the one default module, plus the named modules and their
  dependencies. It records the set on the `modules` line of `.kit/manifest`.
- `kit module list` shows every module the release has, and which are active.
- `kit module add <module>…` activates more modules, placing only the paths the new set adds to the old one.
- `kit module remove <module>…` deactivates modules. It removes the paths only the removed modules delivered,
  and takes the kit's entries back out of any merged `.json` file.

## Layout

```text
modules/<name>/
  module.yml          metadata
  files/<base>/       files mirrored at the base's root
  templates/<base>/   files rendered from the repository's answers, then mirrored the same way
  regions/<base>/     each file a region the kit owns inside a file the repository owns
  docs/               the module's documentation; packed with the release, placed in no repository
```

A module must have `module.yml`. `files/`, `templates/`, `regions/`, and `docs/` are optional. The base
sets where a file lands:

| Base | Root | Record |
| --- | --- | --- |
| `repo` | the repository root | `.kit/manifest`, committed with the repository |
| `home` | the user's home directory, `~` | `~/.kit/manifest`, on the user's machine; a CI runner has none |

- `files/home/.kit/<name>` is a user-level setting. Each machine has one copy, from whichever repository installed
  or updated last.
- `kit check` compares home paths with the release where `~/.kit/manifest` exists. Where it does not, as on a
  runner, which installs nothing there, `kit check` reports that it did not check them.
- A module that names any other base is refused at install, and the error lists the known bases.
- `KIT_HOME` replaces the home directory. The tests use it to stay out of the real one.

## Metadata

`module.yml`, a rule's frontmatter, and `.kit/settings.yml` all use the small subset of YAML that `bin/yaml.sh`
reads: scalars, lists of `- item`, and one level of map. A line starting with `#` is a comment.

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
| `description` | Required. One line, shown by `kit module list`. |
| `default` | `true` makes the module active unless removed. In this release only `kit` sets it. |
| `depends` | Modules activated with this one. `kit module remove` refuses a module another active module depends on. |
| `answers` | The default value of each template key. A repository overrides one under `answers:` in `.kit/settings.yml`. |
| `expects` | Lines the module expects in files the repository owns, each a `path`, a `pattern`, and a `remedy`; see below. |

[module.schema.json](../modules/module.schema.json) is the schema, in JSON Schema 2020-12. This repository's gate
(its own CI command) validates every `module.yml` against it with `ajv`. Without a validator, `kit install` applies
the same contract and refuses, naming the problem:

- a key it does not know
- a missing `description`
- a `default` that is not `true` or `false`
- a dependency the release does not have

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
- Every other destination is a copy.
- Where two active modules deliver the same path, the later module in dependency order wins.

### Includes

Where a file format supports includes, a module delivers a whole file of its own under `.kit/`. It then expects
the repository's file to include it with one line:

- lefthook's `extends`
- just's `import`
- git's `include`
- markdownlint's `extends`

The module declares each such line under `expects:`. An entry has a `path`, a `pattern` (an extended regular
expression that some line of that file must match), and a `remedy`.

```yaml
expects:
  - path: lefthook.yml
    pattern: '\.kit/lefthook\.yml'
    remedy: "put extends: [.kit/lefthook.yml] in lefthook.yml, so the kit's pre-commit commands run"
```

- `kit install` and `kit module add` list each unmet expectation as left to do.
- `kit check` fails until every expectation is met.
- The kit writes nothing into the file. It belongs entirely to the repository.

A single-quoted value is taken literally, backslashes included. Write patterns that way.

The kit changes a file it touches in one of four ways:

| When the file | The kit uses |
| --- | --- |
| has a format that supports includes | an include line the repository adds |
| has a format with comments | a region |
| supports neither | a merge by key |
| belongs entirely to the kit | a copy |

### Regions

A file under `regions/<base>/` is the content of a region inside the file at the same path. It is rendered first,
like a template. The kit writes three things in the file's comment syntax: the bare directive `kit:generate <path>`,
the content, and the marker `kit:end <path>`. [generate.md](generate.md) describes directives. The comment syntax
follows the file's extension:

- `<!-- … -->` for markdown, HTML, XML, and SVG
- `//` for JavaScript, TypeScript, Go, Rust, C, C++, Java, Kotlin, Swift, C#, and Scala
- `--` for SQL, Lua, and Haskell
- `#` for everything else

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
- A file holds at most one kit region. Two modules that deliver a region to the same path are refused by name.
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

A key with no value in `.kit/settings.yml` takes the module's default from its own `answers`. A key with neither
fails the install, and the error names the key and the module.

## Settings

`.kit/settings.yml` belongs to the repository. The kit reads it and never writes it.

- Its schema is [settings.schema.json](../modules/kit/files/repo/.kit/settings.schema.json), delivered beside it as
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
| `forge` | The forge provider the workflow commands use, `github` or `gitlab`. The forge is the service that hosts the repository. When absent, it is detected from origin's host. Set it for a self-hosted forge. |
| `gate` | What `kit gate` runs. When absent, the kit detects a justfile `ci` recipe, a Makefile `ci` target, or a `package.json` `ci` script. |
| `require_signing` | `true` makes `kit issue` refuse to start when git has no `user.signingkey`. |
| `answers` | The values a module's templates use, overriding the module's own defaults. |
| `toolchains` | The tool managers that put the gate's tools on `PATH`: `mise`, `brew`, or `docker`; see [toolchains.md](toolchains.md). |
| `policy` | The patterns a branch, a commit subject, and a change title must match; see [policy.md](policy.md). |
| `policy_engine` | `patterns` or `opa`; see [policy.md](policy.md). |
| `trackers` | Trackers other than the forge's own whose issues the workflow accepts, each with a prefix and a provider; see [trackers.md](trackers.md). |

## Modules a repository defines

A directory `.kit/modules/<name>/` in the repository is a module in every respect. It has the same parts:
`module.yml`, `files/<base>/`, `templates/<base>/`, `regions/<base>/`, and `docs/`.

- The kit looks up a module name in the release first, then in `.kit/modules/`.
- It is validated like a delivered module, and it may depend on delivered modules.
- Activate it with `kit install <name>` or `kit module add <name>`.
- `kit module list` shows it with its origin.
- Its paths are recorded in the manifest with their digests, so `kit check` can tell an edited installed copy
  from its source.
- On update, its files are compared with its own current source. An edit under `.kit/modules/` is never merged
  against a release.
- `kit context` finds a rule file it places under `.kit/rules/` like any other rule.

## This release's modules

| Module | Default | Depends on | Delivers |
| --- | --- | --- | --- |
| `kit` | yes | | `.kit/1st.md`, `.kit/rules/<kind>.md`, `.kit/templates/`, `.kit/settings.schema.json`, `.kit/markdownlint.jsonc`; regions in `AGENTS.md` and `.gitignore` |
| `lefthook` | no | `kit` | `.kit/lefthook.yml`, a pre-commit hook running `kit lint` and `kit generate --check` |
| `ci-github` | no | `kit` | `.github/workflows/kit-ci.yml`, from a template, running `kit ci`; answer `ci_runner` |
| `claude` | no | `kit` | a region in `.claude/CLAUDE.md` importing `AGENTS.md`; `.claude/settings.json`, merged: the deny list and the `kit context --claude` hook |
| `policy-opa` | no | `kit` | `.kit/policy/kit.rego`, the naming policy in Rego, so `kit policy` runs on Open Policy Agent |

No module in this release delivers to `home`. Each module's `docs/README.md` describes what it places and why.
