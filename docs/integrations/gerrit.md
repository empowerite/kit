# gerrit

This page says what `gerrit` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Gerrit](#gerrit) | [code forge](forges.md) | `none` |

## Gerrit

Gerrit reviews changes rather than branches: each commit is a change, named by the `Change-Id` trailer in its
message, and a new push of the commit is a new patch set of the same change. A change's title is its commit's
subject, and its body the rest of the message. There is no draft; a change pushed as work in progress waits there.
Gerrit has two command lines. Its SSH daemon answers `ssh -p 29418 "${host}" gerrit …`, and
`gerrit query --format=JSON` prints one JSON object per line. `git-review`, the OpenStack project's community tool,
pushes a branch to `refs/for/${branch}`. Everything else is the REST API at `https://${host}/a/`, with HTTP
credentials from the user's settings; each JSON answer starts with the line `)]}'`, which a reader drops.

### Read from Gerrit

| Capability | Command |
| --- | --- |
| Signed in | `ssh -p 29418 "${host}" gerrit version`; `curl --user "${user}:${password}" "https://${host}/a/accounts/self"` (raw API) |
| Owner and name | the project's name, from the remote's path; `GET /a/projects/${project}` (raw API) |
| Browser URL | none; a change's is `https://${host}/c/${project}/+/${n}` |
| Caller's login | `GET /a/accounts/self` (raw API): `username` |
| Rate limits | none in Gerrit itself; the quota plugin adds some |
| Marks | `GET /a/changes/?q=project:${project}&n=1` (raw API); Gerrit lists the change updated last first. No ETag was seen |
| Branch for an issue | `ssh -p 29418 "${host}" gerrit query --format=JSON "status:open project:${project} topic:${n}"`, when the change's topic names the issue |
| Title, body | `gerrit query --format=JSON "change:${n}"`: `subject`; `GET /a/changes/${n}/message` (raw API) for the whole message |
| Author | `gerrit query --format=JSON "change:${n}"`: `owner.username` |
| State and merge commit | `gerrit query --format=JSON --current-patch-set "change:${n}"`: `status` (`NEW`, `MERGED`, `ABANDONED`) and the current patch set's `revision`, which is the merged commit unless the submit rebased it |
| Change on a branch | `gerrit query --format=JSON "change:${change_id}"`, with the `Change-Id` of the branch's head commit |
| Closing issues | none; an its-* plugin links an issue tracker by a footer in the message |
| Review threads | `GET /a/changes/${n}/comments` (raw API): each comment's `unresolved`, a thread's state being its last comment's |
| Checks rollup | `gerrit query --format=JSON --submit-records "change:${n}"`, the labels and submit requirements; `GET /a/changes/${n}/detail` (raw API). CI results are votes on a label such as `Verified` |
| Open changes | `gerrit query --format=JSON "status:open project:${project}"` |

### Write to Gerrit

| Capability | Command |
| --- | --- |
| Branch linked to an issue | none; a local branch, and `git review --topic "${n}"` |
| Open a draft | `git review --wip "${default}"`, or `git push origin "HEAD:refs/for/${default}%wip"` |
| Mark ready | `git review --ready` on the next push, or `POST /a/changes/${n}/ready` (raw API) |
| Set the title | `PUT /a/changes/${n}/message` (raw API), which makes a new patch set; or amend the commit and push again |
| Set the body | `PUT /a/changes/${n}/message` (raw API) |

Gaps: Gerrit's change is a commit, not a branch, so the contract's branch lookups go through the `Change-Id`, and
`kit work`'s `Closes` lines would go in the commit message. `gerrit query` prints no unresolved comment state.

### Gerrit schemas

- Gerrit publishes no OpenAPI document or GraphQL schema. The REST API is described in prose at
  [Gerrit REST API](https://gerrit-review.googlesource.com/Documentation/rest-api.html), and the query command at
  [gerrit query](https://gerrit-review.googlesource.com/Documentation/cmd-query.html).
