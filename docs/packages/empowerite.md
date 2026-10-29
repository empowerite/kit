# The empowerite package

The bundle of packages Empowerite's repositories take. It delivers no file of its own: adding it activates its
members, and that is all it does.

| Member | Brings |
| --- | --- |
| the seven `style/<kind>` packages | the style for each kind of file, with its templates; see [style.md](style.md) |
| `github/ci` | the GitHub Actions job that runs `kit ci` |
| `agents/claude` | Claude Code's deny list, and the hook that runs `kit context` before every write |
| `hooks/lefthook` | the pre-commit hook that runs `kit lint` and `kit generate --check` |

`kit install empowerite` takes all of them. A member a repository disagrees with leaves on its own:
`kit package remove style/just` records it as excluded, so `kit update` does not bring it back as a member, and
`kit package add style/just` ends the exclusion. Removing the bundle itself leaves its members active.

The bundle is in the release until a package can come from a source of its own, such as a repository of
Empowerite's; see [#20](https://github.com/empowerite/kit/issues/20). This repository's `house` package depends on
it, and adds the taste that applies here only.
