# The lefthook module

Places `.kit/lefthook.yml`, which says which linter runs over which paths before a commit. The repository's own
`lefthook.yml` takes it with `extends: [.kit/lefthook.yml]` and adds commands of its own; Lefthook merges by name.
`kit lint` runs the same hook by hand. The repository pins `lefthook` itself, since only a desk runs a hook.
