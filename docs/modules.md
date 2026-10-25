# Modules

Everything the kit delivers is a module: a directory under `modules/` in the release. A repository activates the
modules it wants, and `kit install`, `kit update`, and `kit check` work over the active set.

## Layout

```text
modules/<name>/
  module        metadata, one keyword per line
  tree/         files mirrored at the repository root
  templates/    files rendered from the repository's answers, then mirrored the same way
  docs/         the module's documentation, for a reader of this repository; not delivered
```

A module may have any of `tree/`, `templates/`, and `docs/`, and must have `module`.

## Metadata

The `module` file is lines of one keyword each. A line starting with `#` is a comment.

| Keyword | Meaning |
| --- | --- |
| `description <text>` | One line, shown by `kit module list`. |
| `default` | Active unless removed. `kit install` with no names activates every default module. |
| `depends <name>…` | Activated with this module. `kit module remove` refuses a module another active module depends on. |
| `answer <key> <value>` | The default value of a template key. A repository overrides it in `.kit/answers`. |

## Delivery

A file under `tree/` is placed at the same path under the repository root. A destination ending in `.json` is
merged by key: the kit's keys are set, its array entries added, and on update or removal only the previous
release's entries are taken out, so a repository's own keys and entries survive. Every other destination is a copy.
Where two active modules deliver the same path, the later module in dependency order wins.

A file under `templates/` is rendered first. Every `@key@` in it, autoconf's form, is replaced by the key's
answer, then the result is placed, digested, and merged exactly as a `tree/` file is. The form is one neither
bash's `${var}` nor GitHub's `${{ }}` can be mistaken for, so a template may carry both untouched. The repository's
answers live in `.kit/answers`, a file of `key value` lines the repository owns and the kit never writes; a key
with no line there takes the module's `answer` default, and a key with neither refuses the install, naming the key
and the module.

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
| `claude` | no | `kit` | `.claude/settings.json`, merged: the deny list and the `kit context --claude` hook |

Each module's `docs/README.md` says what it places and why.
