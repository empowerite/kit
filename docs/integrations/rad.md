# rad

This page says what `rad` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Radicle](#radicle) | [code forge](forges.md) | `none` |

## Radicle

Radicle is a peer-to-peer code forge: each node keeps the repositories it follows, and issues and patches are
collaborative objects stored in git and spread by seeds. `rad` is its own CLI, and the identity is a key, created with
`rad auth`. A patch is opened by a push to the `rad` remote, `git push rad HEAD:refs/patches`, with push options for
draft, base, and message. `rad patch` prints text; `rad cob show` prints a patch as JSON, by default. Reads are local,
so they cost no request and count against no limit, and `rad sync` fetches what the seeds have.

### Read from Radicle

| Capability | Command |
| --- | --- |
| Signed in | `rad self` |
| Owner and name | `rad inspect`: the repository's ID, `rad:…` |
| Browser URL | none; a seed's web view, such as [Radicle's web view](https://app.radicle.xyz), shows the repository |
| Caller's login | `rad self --did` |
| Rate limits | none |
| Marks | none; reads are local, after `rad sync` |
| Branch for an issue | none; a patch records its head commit, not a branch name |
| Title, body | `rad patch show "${id}"`, or `rad cob show --repo "${rid}" --type xyz.radicle.patch --object "${id}"` for JSON |
| Author | `rad cob show --repo "${rid}" --type xyz.radicle.patch --object "${id}"`: the author's DID |
| State and merge commit | `rad cob show` on the patch: its state (`draft`, `open`, `archived`, `merged`) and the merge's commit |
| Change on a branch | `rad patch list --all`, and the patch whose head is the branch's commit |
| Closing issues | none |
| Review threads | `rad cob show` on the patch: its reviews' comments and their resolution |
| Checks rollup | none in `rad`; a CI broker writes its results elsewhere (not verified) |
| Open changes | `rad patch list --open`, as text |

### Write to Radicle

| Capability | Command |
| --- | --- |
| Branch linked to an issue | none |
| Open a draft | `git push -o patch.draft -o patch.message="${title}" rad HEAD:refs/patches` |
| Mark ready | `rad patch ready "${id}"`; `--undo` makes it a draft again |
| Set the title | `rad patch edit "${id}" --message "${title}"` |
| Set the body | `rad patch edit "${id}" --message "${title}" --message "${body}"`; the messages are joined with newlines |

Gaps: a patch has no branch and no closing issues, and `rad patch` prints no JSON. A patch's ID is a hash, where the
contract's change is a number.

### Radicle schemas

- Radicle publishes no OpenAPI document for its node's HTTP API, and no JSON Schema for its collaborative objects; the
  types are in the Rust source of [radicle-dev/heartwood](https://github.com/radicle-dev/heartwood).
