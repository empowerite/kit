# App verbs

An app verb prints the tabs of one of kittui's apps, each a stage, in the shape `kit workspace` prints its own. Each
app's verb is named for it, and `modules/bash/apps.sh` runs every one of them: its stages, its marks, its forms, and
its hand-off to kittui. A verb's own stages live in `modules/bash/apps/<verb>.sh`.

## The shape every app verb shares

- `kit <verb> --stages` prints the stages, one per line, in the app's tab order.
- `kit <verb> --marks` prints a mark per stage as one JSON object. A mark is a text that changes when what its stage
  draws does, and `null` for a stage with none, which kittui reads again only on Refresh and Watch.
- `kit <verb> <stage> --json` prints the stage's document, `{rows, errors}`, and `limited: N` where a capped read
  stopped at N, the newest. `--jq` prints it formatted, and `--text`, or no form off a terminal, as a table.
- `kit <verb> --json`, with no stage, prints `{stages, <stage>: document…}`, a stage that could not be read `null`.
- On a terminal, with no form, `kit <verb>` opens the app in kittui, and `kit <verb> <stage>` opens it on that tab.
- `kit --workspace=<group> <verb>` runs each stage in every cloned member's checkout at once. Each row, and each
  error, gains `member`, its host and path, and a member that could not be read is `{member, error}` under `errors`.

A row is `{key, …, links}`: `key` is the row's own within its stage, and `links` gives where a value leads, by its
field. A stage that reads the forge goes through the forge seam, `modules/bash/forge.sh`. A primary with no forge, or
a forge that lacks the stage's feature, gives no rows and no errors. A forge that says the feature is off for this
repository gives no rows and the error `{feature, off: true}`. Neither is a failure. A read that fails prints its one
line and the verb exits 1.

A tab of the app that another command prints opens on a terminal as the verb's own do; with `--json`, the verb
refuses with 2 and names that command.

Each verb exits 0 when its stages are drawn, 1 when one could not be read, after what was, and 2 on a usage error,
or outside a repository's working tree with no workspace.

## Platform: kit platform

`kit platform` prints what kit runs with here. It asks the forge only whether its tool is signed in.

| Stage | A row per | Fields | Mark |
| --- | --- | --- | --- |
| `services` | capability kit fills | `capability, service, provider, chosen, ready` | a hash of `kit.yml`, the git configuration, `.kit/policy`, and what `toolchains` hashes |
| `toolchains` | toolchain and role, as `kit toolchains --json` lists them | `provider, role, usable, active, chosen, fills` | a hash of the pin files and kit's choice |
| `tools` | tool pinned, as `kit tools --json` lists them | `name, pinned, installed, from` | as `toolchains` |
| `settings` | key the settings schema names | `setting, value, from` | a hash of `kit.yml` |

- `services` holds `hosting`, `review`, `ci`, and `releases`, the primary's forge; `tracking <prefix>`, each tracker
  prefix; `toolchain`, the project toolchain; `dependencies <provider>`, each dependency provider whose manifest and
  lock are here, or `dependencies` alone for none; and `policy`, the policy engine. `service` is what answers it, as
  `github`, `mise`, or `patterns`, `null` for none: the forge's with no remote, or a `ci` the forge lacks. `provider`
  is kit's module that reaches it, as `forge/github`. `chosen` is `set` where `kit.yml` or a recorded choice names
  it, else `inferred`. `ready` is whether its tool is there and signed in: a forge whose tool is not signed in is not
  ready, and the stage still draws.
- `toolchains` keys each row by provider and role, `mise project`, and `tools` by where the tool is pinned and its
  name. Where `kit tools` refuses, as with several toolchains usable and none chosen, `tools` is its line and the verb
  exits 1; `toolchains` still draws.
- `settings` lists every key of `.kit/kit.schema.json`, in its order: the value `kit.yml` sets, `from: set`, else the
  schema's default, `null` where it names none, `from: default`.
- The toolchains' mark hashes every file a toolchain or a dependency provider pins from, and the files that keep
  kit's choice of toolchain: this checkout's, the clone's, and yours under `~/.kit/settings.yml`.

see: [toolchains.md](toolchains.md), [workspaces.md](workspaces.md)
