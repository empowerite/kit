# hut

This page says what `hut` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [SourceHut](#sourcehut) | [code forge](forges.md) | `none` |

## SourceHut

SourceHut reviews changes by email: a contributor sends a patch series to a mailing list with `git send-email`, and
lists.sr.ht keeps it as a patchset with a status, one of `PROPOSED`, `NEEDS_REVISION`, `SUPERSEDED`, `APPROVED`,
`REJECTED`, or `APPLIED`. A patchset is not tied to a branch, has no draft, and has no review threads beyond the
replies on the list. `hut` is the community CLI: `hut init` stores an OAuth token per instance. Its commands print
text, and `hut graphql "${service}"` runs a GraphQL query on one service, `git`, `lists`, or `meta`, and prints the
JSON answer.

### Read from SourceHut

| Capability | Command |
| --- | --- |
| Signed in | `hut meta show` |
| Owner and name | `hut git show`, as text |
| Browser URL | none; a repository's is `https://git.sr.ht/~${owner}/${repo}` |
| Caller's login | `hut meta show`, or `hut graphql meta <<<'query { me { canonicalName } }'` |
| Rate limits | none documented |
| Marks | `hut graphql lists` on a list's `patches` and their `updated` (raw API) |
| Branch for an issue | none |
| Title, body | `hut lists patchset show "${id}"`: the subject and the cover letter |
| Author | `hut lists patchset show "${id}"`: the submitter |
| State and merge commit | `hut lists patchset show "${id}"`: the status; an applied patchset names no commit |
| Change on a branch | none |
| Closing issues | none; todo.sr.ht reads `Fixes:` and `Implements:` trailers in commits pushed to git.sr.ht (not verified) |
| Review threads | none; replies on the list have no resolved state |
| Checks rollup | none from `hut`; builds.sr.ht can test a patchset sent to a list set up for it (not verified) |
| Open changes | `hut lists patchset list "${list}"`, which shows the `PROPOSED` ones by default |

### Write to SourceHut

| Capability | Command |
| --- | --- |
| Branch linked to an issue | none |
| Open a draft | none; `git send-email --to="${list_address}" "${default}"` sends the series ready for review, and `--rfc` marks it a request for comments |
| Mark ready | none |
| Set the title | none; a new version of the series |
| Set the body | none; a new version of the series |
| Set the status | `hut lists patchset update "${id}" --status APPROVED` |

Gaps: the contract's change is a branch with a body that can change, and a SourceHut patchset is neither, so most of
the change calls have no answer. A provider would read patchsets alone, through `hut graphql`.

### SourceHut schemas

- [git.sr.ht GraphQL schema](https://git.sr.ht/~sircmpwn/git.sr.ht/blob/master/api/graph/schema.graphqls),
  [lists.sr.ht GraphQL schema](https://git.sr.ht/~sircmpwn/lists.sr.ht/blob/master/api/graph/schema.graphqls), and
  [meta.sr.ht GraphQL schema](https://git.sr.ht/~sircmpwn/meta.sr.ht/blob/master/api/graph/schema.graphqls), each in
  the public domain or CC0. todo.sr.ht's is on [trackers.md](trackers.md), and builds.sr.ht's manifest schema on
  [ci.md](ci.md).
