# The lefthook module

Places `.kit/lefthook.yml`, a pre-commit hook with two commands over the staged paths: `kit lint`, so each rule's own
linter runs over the paths the rule governs before a commit, and `kit generate --check`, so a stale generated region is
refused before it is committed. The repository's own `lefthook.yml` takes it with `extends: [.kit/lefthook.yml]` and
adds commands of its own; Lefthook merges by name. The repository pins `lefthook` itself, since only a desk runs a hook,
and `kit lint` and `kit generate` run the same way with no hook manager at all.
