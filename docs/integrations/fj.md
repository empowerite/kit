# fj

This page says what `fj` reads and writes where it meets kit's scope, for each service it reaches, and the
machine-readable descriptions each publishes. The index of these pages is [1st.md](1st.md).

| Service | Category | kit status |
| --- | --- | --- |
| [Forgejo and Codeberg](#forgejo-and-codeberg) | [code forge](forges.md) | `none` |

## Forgejo and Codeberg

Forgejo is a hard fork of Gitea, and Codeberg is the largest public instance of it. Its API is Gitea's, with
additions under `/api/forgejo/v1`, and it drifts from Gitea's release by release. `fj` is the community CLI: it
authenticates with `fj auth login` or `fj auth add-token`, per instance, and has `fj pr create`, `view`, `status`,
`edit`, `search`, and `merge`, all printing text; it has no JSON output and no raw API command. `tea` reaches a
Forgejo instance for the calls the two still share, which Forgejo does not test (not verified). The tables use `fj`
where it has the command, and the raw API through `tea api` or `curl` elsewhere.

### Read from Forgejo and Codeberg

| Capability | Command |
| --- | --- |
| Signed in | `fj whoami` |
| Owner and name | `fj repo view`, as text; `tea api 'repos/{owner}/{repo}'` for JSON (raw API) |
| Browser URL | `html_url` of `tea api 'repos/{owner}/{repo}'` (raw API) |
| Caller's login | `fj whoami` |
| Rate limits | the `RateLimit` and `RateLimit-Policy` headers Codeberg sends with every answer, read by `tea api --include` (raw API); a self-hosted Forgejo sets its own |
| Marks | `tea api 'repos/{owner}/{repo}/pulls?state=all&sort=recentupdate&limit=1'` (raw API) |
| Branch for an issue | `tea api 'repos/{owner}/{repo}/pulls?state=open'` and the head that starts with `${n}-` (raw API) |
| Title, body | `fj pr view "${n}"`, and `fj pr view "${n}" body` |
| Author | `fj pr view "${n}"`, as text |
| State and merge commit | `tea api "repos/{owner}/{repo}/pulls/${n}"` (raw API): `merged`, `merge_commit_sha` |
| Change on a branch | `tea api "repos/{owner}/{repo}/pulls/${default}/${branch}"` (raw API) |
| Closing issues | none, as on Gitea |
| Review threads | `tea api "repos/{owner}/{repo}/pulls/${n}/reviews"` and each review's `comments`, with `resolver` (raw API) |
| Checks rollup | `fj pr status "${n}"`, as text |
| Open changes | `fj pr search --state open`, as text |

### Write to Forgejo and Codeberg

| Capability | Command |
| --- | --- |
| Branch linked to an issue | `tea api -X POST 'repos/{owner}/{repo}/branches' -f new_branch_name="${n}-${slug}" -f old_ref_name="${default}"` (raw API), with no link to the issue |
| Open a draft | `fj pr create --base "${default}" --head "${branch}" "WIP: ${title}"`; Forgejo, like Gitea, marks a draft by the prefix |
| Mark ready | `fj pr edit "${n}" title "${title}"`, without the prefix |
| Set the title | `fj pr edit "${n}" title "${title}"` |
| Set the body | `fj pr edit "${n}" body "${body}"` |

Gaps: `fj` prints no JSON, so a kit provider would read the API for every read. A provider over `tea` covers more,
but rests on an API the two projects no longer keep the same.

### Forgejo and Codeberg schemas

- Every Forgejo instance serves `https://${host}/swagger.v1.json` (Swagger 2.0); Codeberg's is
  [Codeberg Swagger 2.0 description](https://codeberg.org/swagger.v1.json). Forgejo serves no OpenAPI 3.0 document.
- The source's is
  [Forgejo Swagger 2.0 description in the repository](https://codeberg.org/forgejo/forgejo/raw/branch/forgejo/templates/swagger/v1_json.tmpl),
  a template the server fills with its URL.
- No JSON Schema for webhook payloads or `fj` output is published.
