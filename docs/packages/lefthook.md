# The lefthook package

Delivers `.kit/lefthook.yml`, a pre-commit hook with two commands over the staged paths:

- `kit lint` runs each rule's own linter over the paths the rule governs, before the commit.
- `kit generate --check` refuses a stale generated region before it is committed.

The repository's own `lefthook.yml` includes the hook with `extends: [.kit/lefthook.yml]` and adds its own
commands. Lefthook merges commands by name. The kit writes nothing into `lefthook.yml`. The package requires the
`extends` line: `kit install` lists it as a step left to do, and `kit check` fails until it is there.

The repository pins `lefthook` itself, because only a developer's machine runs a hook. `kit lint` and
`kit generate` run the same way without any hook manager.
