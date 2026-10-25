# Modules

Everything the kit delivers is a module: a directory under `modules/` in the release. A repository activates the
modules it wants, and `kit install`, `kit update`, and `kit check` work over the active set.

## Layout

```text
modules/<name>/
  module.yml          metadata
  files/<base>/       files mirrored at the base's root
  templates/<base>/   files rendered from the repository's answers, then mirrored the same way
  docs/               the module's documentation, for a reader of this repository; not delivered
```

A module may have any of `files/`, `templates/`, and `docs/`, and must have `module.yml`. A base names where a
file lands:

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
description: One CI job: the repository's gate, kit check, and the linked-open-issue refusal
default: true
depends:
  - kit
answers:
  ci_runner: ubuntu-latest
```

| Key | Meaning |
| --- | --- |
| `description` | One line, shown by `kit module list`. |
| `default` | `true` for a module active unless removed. `kit install` with no names activates every default module. |
| `depends` | Modules activated with this one. `kit module remove` refuses a module another active module depends on. |
| `answers` | The default value of each template key. A repository overrides one in `.kit/answers`. |

[module.schema.json](../modules/module.schema.json) is the schema, JSON Schema 2020-12. The gate validates every
`module.yml` against it with `ajv`, and `kit install` holds a module to the same contract where no validator is:
a key it does not know, a `default` that is not `true` or `false`, or a dependency the release lacks is refused,
naming it.

## Delivery

A file under `files/<base>/` is placed at the same path under the base's root. A destination ending in `.json` is
merged by key: the kit's keys are set, its array entries added, and on update or removal only the previous
release's entries are taken out, so a repository's own keys and entries survive. Every other destination is a copy.
Where two active modules deliver the same path, the later module in dependency order wins.

A file under `templates/<base>/` is rendered first. Every `@key@` in it, autoconf's form, is replaced by the key's
answer, then the result is placed, digested, and merged exactly as a `tree/` file is. The form is one neither
bash's `${var}` nor GitHub's `${{ }}` can be mistaken for, so a template may carry both untouched. The repository's
answers live in `.kit/answers`, a file of `key value` lines the repository owns and the kit never writes; a key
with no line there takes the module's `answer` default, and a key with neither refuses the install, naming the key
and the module.

The same file is also what a rule's frontmatter is written in, read by the same functions, so a rule's `paths:` and
`content:` lists and a module's `depends:` list have one reader between them.

## Activation

- `kit install [<module>…]` activates the default modules and the named ones, with their dependencies, and records
  the set on the `modules` line of `.kit/manifest`.
- `kit module list` shows every module the release carries and which are active.
- `kit module add <module>…` activates more, placing only the paths the new set delivers beyond the old.
- `kit module remove <module>…` deactivates, removing the paths only the removed modules delivered and taking the
  kit's entries back out of a merged `.json`.

## This release's modules

| Module | Default | Depends on | Delivers |
| --- | --- | --- | --- |
| `kit` | yes | | `.kit/1st.md`, `.kit/rules/<kind>.md` |
| `lefthook` | yes | `kit` | `.kit/lefthook.yml` |
| `ci` | yes | `kit` | `.github/workflows/kit-ci.yml`, from a template; answer `ci_runner` |
| `claude` | no | `kit` | `.claude/CLAUDE.md` importing `AGENTS.md`; `.claude/settings.json`, merged: the deny list and the `kit context --claude` hook |

No module delivers to `home` yet. A module's `files/repo/` and `templates/repo/` carry the paths above; its
`docs/README.md` says why.

Each module's `docs/README.md` says what it places and why.
