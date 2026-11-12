# tea

`tea` is Gitea's own CLI. It authenticates with `tea login add`, a token per instance, and finds the login from the
checkout's remote. `--output json` (`-o json`) prints JSON, and `--fields` picks the fields of a list. A list takes
`--page` and `--limit`, and reads one page. `tea api "${endpoint}"` calls any endpoint under `/api/v1/`, fills
`{owner}` and `{repo}` from the checkout, and prints headers on standard error with `--include`; it has no
`--paginate`. Gitea marks a pull request as a draft by a `WIP:` prefix on its title.

This page says what `tea` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Gitea pull requests](#gitea-pull-requests) | [code forge](forges.md) | `none` |
| [Gitea and Forgejo issues](#gitea-and-forgejo-issues) | [tracker](trackers.md) | none |
| [Forgejo and Gitea Actions](#forgejo-and-gitea-actions) | [CI](ci.md) | `none` |

## Gitea pull requests

### Read from Gitea pull requests

| Capability | Command |
| --- | --- |
| Signed in | `tea whoami` |
| Owner and name | `tea api 'repos/{owner}/{repo}'`: `full_name` (raw API) |
| Browser URL | `tea api 'repos/{owner}/{repo}'`: `html_url` (raw API); a pull request's is `${url}/pulls/${n}` |
| Caller's login | `tea whoami`, or `tea api user`: `login` |
| Rate limits | none found; gitea.com sent no rate-limit header |
| Marks | `tea api 'repos/{owner}/{repo}/pulls?state=all&sort=recentupdate&limit=1'` (raw API); no ETag was seen |
| Branch for an issue | `tea pulls list --state open --fields index,head --output json` and the head that starts with `${n}-` |
| Title, body | `tea pulls list --state all --fields index,title,body --output json`, or `tea api "repos/{owner}/{repo}/pulls/${n}"` |
| Author | the `author` field of `tea pulls list --fields` |
| State and merge commit | `tea api "repos/{owner}/{repo}/pulls/${n}"` (raw API): `state`, `merged`, and `merge_commit_sha` |
| Change on a branch | `tea pulls list --state all --fields index,state,head,base-commit --output json`, or `tea api "repos/{owner}/{repo}/pulls/${default}/${branch}"` (raw API) |
| Closing issues | none; Gitea closes the issues a merged pull request's body or commits name with a closing keyword, and lists no such link |
| Review threads | `tea pulls review-comments "${n}" --fields id,path,line,resolver --output json`; a comment with a `resolver` is resolved |
| Checks rollup | the `ci` field of `tea pulls list --fields`, or `tea api "repos/{owner}/{repo}/commits/${sha}/status"` (raw API) |
| Open changes | `tea pulls list --state open --output json --limit "${limit}"` |

### Write to Gitea pull requests

| Capability | Command |
| --- | --- |
| Branch linked to an issue | `tea api -X POST 'repos/{owner}/{repo}/branches' -f new_branch_name="${n}-${slug}" -f old_ref_name="${default}"` (raw API), with no link to the issue |
| Open a draft | `tea pulls create --draft --head "${branch}" --base "${default}" --title "${title}" --description ""` |
| Mark ready | `tea pulls edit "${n}" --ready`, which strips the `WIP:` prefix |
| Set the title | `tea pulls edit "${n}" --title "${title}"` |
| Set the body | `tea pulls edit "${n}" --description "${body}"`, or `--description-file -` |

Gaps: `tea api` reads one page, so a whole list is read page by page. Gitea keeps no link from a pull request to the
issues it closes until the merge closes them.

### Gitea pull requests schemas

- Every Gitea instance serves its own description at `https://${host}/swagger.v1.json` (Swagger 2.0) and
  `https://${host}/openapi3.v1.json` (OpenAPI 3.0); gitea.com's are
  [gitea.com Swagger 2.0 description](https://gitea.com/swagger.v1.json) and
  [gitea.com OpenAPI 3.0 description](https://gitea.com/openapi3.v1.json).
- The source's are
  [Gitea Swagger 2.0 description in the repository](https://raw.githubusercontent.com/go-gitea/gitea/main/templates/swagger/v1-swagger.generated.json)
  and
  [Gitea OpenAPI 3.0 description in the repository](https://raw.githubusercontent.com/go-gitea/gitea/main/templates/swagger/v1-openapi3.generated.json),
  for the next release.
- No JSON Schema for webhook payloads or `tea` output is published. SchemaStore's Gitea entries are for issue
  templates, under [Gitea and Forgejo issues schemas](#gitea-and-forgejo-issues-schemas).

## Gitea and Forgejo issues

Gitea and Forgejo are self-hosted code forges with issues; Forgejo is a fork of Gitea and keeps its API. Gitea is
MIT and Forgejo GPL-3.0; Gitea Cloud hosts Gitea and Codeberg hosts Forgejo. An issue has assignees, labels, a
milestone, and dependencies; it has no parent and no close reason.

`tea` is Gitea's own CLI. `tea login add` stores a token per instance. `--output json` prints JSON from lists, and
`tea api` reaches any REST route, with `{owner}` and `{repo}` standing for the current repository. `--page` and
`--limit` page a list. It talks to Forgejo too, which keeps Gitea's routes, though a Forgejo-only route may differ.
`fj` is a community CLI for Forgejo ([forgejo-contrib/forgejo-cli](https://codeberg.org/forgejo-contrib/forgejo-cli))
that prints text only.

### Read from Gitea and Forgejo issues

| Capability | Command |
| --- | --- |
| Title, state, and assignees | `tea api "/repos/{owner}/{repo}/issues/${n}"` |
| Caller | `tea whoami` |
| Issue page | `tea api "/repos/{owner}/{repo}/issues/${n}"`, its `html_url` |
| New-issue page | none; `${repo_url}/issues/new?title=${title}` |
| Open issues | `tea issues list --state open --fields index,title,assignees,labels,created,url --output json --limit 50 --page "${page}"` |
| Dependencies | `tea api "/repos/{owner}/{repo}/issues/${n}/dependencies"` |
| Milestones | `tea milestones list --output json` |

### Write to Gitea and Forgejo issues

| Capability | Command |
| --- | --- |
| Assign | `tea issues edit "${n}" --add-assignees "${login}"` |
| Unassign | `tea issues edit "${n}" --remove-assignees "${login}"` |
| Close | `tea issues close "${n}"`; no reason is kept |
| Comment | `tea comments add "${n}" "${text}"` |
| Create | `tea issues create --title "${title}" --description "${body}"` |
| Put under a parent | none; `tea api -X POST "/repos/{owner}/{repo}/issues/${n}/blocks"` makes a dependency instead |
| Link a change | `Closes #${n}` in the pull request's description |
| Set the milestone | `tea issues edit "${n}" --milestone "${milestone}"` |

Gaps: no sub-issues, no duplicate state, and no iterations.

see: [the code forges](#gitea-pull-requests) for `tea` and `fj` against pull requests

### Gitea and Forgejo issues schemas

- [Gitea's Swagger 2.0 from gitea.com](https://gitea.com/swagger.v1.json) and
  [Forgejo's Swagger 2.0 from Codeberg](https://codeberg.org/swagger.v1.json); every instance serves its own at
  `/swagger.v1.json`
- [Gitea Issue Template forms](https://www.schemastore.org/gitea-issue-forms.json) and
  [Gitea Issue Template configuration](https://www.schemastore.org/gitea-issue-config.json), SchemaStore entries
  for `.gitea/ISSUE_TEMPLATE/` and `.forgejo/ISSUE_TEMPLATE/`

## Forgejo and Gitea Actions

Forgejo and Gitea run GitHub Actions' workflow syntax on runners of their own, and post commit statuses as their own
CI. Both are also code forges, so a provider for either would come with a forge provider; see [Gitea pull
requests](#gitea-pull-requests) and [fj.md](fj.md). Gitea's `tea` signs in with `tea login add` and prints `-o json`.
Forgejo's `fj` is a community CLI with no JSON output and only `fj actions tasks` and `fj actions dispatch` for
Actions, so for Forgejo kit's capabilities are mostly REST. Both REST APIs live under
`/api/v1/repos/${owner}/${repo}/actions/` and page with `page` and `limit`.

### Read from Forgejo and Gitea Actions

| Capability | Gitea | Forgejo |
| --- | --- | --- |
| Recent runs | `tea actions runs ls --event push --branch "${branch}" -o json` | `GET runs?event=push&ref=${ref}` (REST); `fj actions tasks` lists jobs |
| A run's jobs | `tea actions runs view "${run_id}" --jobs` | `GET runs/${run_id}/jobs` (REST) |
| A job's log | `tea actions runs logs "${run_id}" --job "${job_id}"` | `GET jobs/${job_id}/logs` (REST) |
| Artifacts | `GET runs/${run_id}/artifacts`, then `GET artifacts/${artifact_id}/zip` (REST) | the same routes (REST) |

### Write to Forgejo and Gitea Actions

| Capability | Gitea | Forgejo |
| --- | --- | --- |
| Rerun | `POST runs/${run_id}/rerun` or `rerun-failed-jobs` (REST) | `POST runs/${run_id}/rerun` (REST) |
| Cancel | `tea actions runs cancel "${run_id}"` | `POST runs/${run_id}/cancel` (REST) |
| Start a run | `tea actions workflows dispatch "${workflow}" --ref "${ref}"` | `fj actions dispatch "${workflow}" "${ref}"` |
| Report a status | the code forge's `POST /api/v1/repos/${owner}/${repo}/statuses/${sha}` (REST) | the same route (REST) |

### Forgejo and Gitea Actions schemas

- The [GitHub workflow JSON Schema](https://www.schemastore.org/github-workflow.json) covers both; SchemaStore matches
  it to `.gitea/workflows/` and `.forgejo/workflows/`.
- [Gitea's Swagger](https://gitea.com/swagger.v1.json), served by gitea.com, and
  [Forgejo's Swagger](https://codeberg.org/swagger.v1.json), served by Codeberg; Forgejo keeps its source in
  [the Forgejo repository](https://codeberg.org/forgejo/forgejo/raw/branch/forgejo/templates/swagger/v1_json.tmpl)
