# Every Planned Tab Reads kit: Design

kittui has 32 tabs across its seven apps, and 20 of them print "has no data source yet"
(`projects/tui/internal/flow/apps.go:26-35`). This design gives each one a kit read, through a seam, from what the
forge provides today, and makes each refresh and go stale the way backlog and changes do.

Issue #138 holds the apps; [the plan for its tabs](../plans/2026-11-10-apps-tabs-read-kit.md) makes a tab one catalog
entry naming its command. This design is the kit side that plan's "Tabs that need a kit stage first" leaves open, and
it widens that plan to every planned tab.

## Intent

A person who opens Issues, Changes, Delivery, People, Trust, or Platform sees real rows in every tab: the milestones,
the reviews waiting, the environments and what is deployed there, who has access, the attestations and alerts, the
rules the forge enforces, and the services and settings kit runs with. The owner's direction, 2026-11-10:

- Every planned tab gets a source, including the ones that need a seam kit does not have.
- The rows come from the forge now: gh's teams, GitHub's attestations, and whatever else the forge can provide. A
  command provider over an organization's own system is the spec's later step, and stays out.
- Each tab refreshes, and goes stale by its verb's `--marks`, as backlog and changes do.

## Goals

1. Each of the 20 tabs reads a kit command that prints its rows as JSON, and kittui draws them from one catalog entry
   in `projects/tui/internal/flow/fields.json`, as the #138 plan's Task 1 makes possible.
2. Every forge read goes through a seam call, answered by `forge/github.sh`, `forge/gitlab.sh`, and `forge/ado.sh`, or
   declared lacking by the provider, under the contract's rules in `kit/modules/bash/forge.sh`'s header.
3. Each tab's verb prints `--stages` and `--marks`. A tab is stale when its mark changes; a tab with no cheap mark has
   a null mark and reloads on Refresh and Watch alone.
4. `kit --workspace=<group>` draws each new stage across the group's members, as kit workspace does.

## Non-goals

- Writes: approving a deployment, dismissing an alert, editing a ruleset, adding a person to a team.
- The command provider, `kit verify`, the evidence store, signing, and controls from an outside standard. Trust's
  tabs show the forge's own concepts, named below, until the spec's evidence seam exists.
- The identity directory: a person is the forge's account, not a handle that joins accounts.
- Jira, Linear, and the other trackers of `docs/integrations/trackers.md`: Issues' new tabs read the forge's tracker.

## Anti-goals

- A tab that fails where the forge lacks the thing. A provider that lacks a call declares it, the call exits 3, and
  the tab draws its empty line, as releases does on Azure DevOps.
- A read per row, or a read whose cost grows with the repository's history. Each stage costs a bounded number of
  requests, as the contract's list limit says.
- A tab that shows a value no forge returned: no guessed state, no default filled in.

## The verbs

Each app's new tabs are the stages of one verb named for the app, shaped as `kit workspace` is: `<verb> <stage>
--json` prints `{rows, errors}`, each row `{key, …, links}`, and under a group each row gains `member`;
`<verb> --stages` and `<verb> --marks` hold for the picture; on a terminal `<verb> <stage>` opens kittui on that tab.

| App | Verb | Stages, in the tab bar's order after the existing ones |
| --- | --- | --- |
| Issues | `kit issues` | `milestones`, `iterations` |
| Changes | `kit changes` | `reviews`, `checks`, `merged` |
| Delivery | `kit delivery` | `artifacts`, `environments`, `deployments` |
| People | `kit people` | `people`, `teams`, `agents` |
| Trust | `kit audit` | `evidence`, `findings`, `controls`, `waivers`, `policy` |
| Platform | `kit platform` | `services`, `toolchains`, `tools`, `settings` |

`kit trust` already trusts a clone (#153), so Trust's verb is `kit audit`, the owner's choice; it changes nothing.
Issues', Changes', and Delivery's existing tabs stay kit flow's stages: the new verb's `--json`, `--stages`, and
`--marks` hold only the new ones. The six verbs also answer the #138 plan's first decision: the word that opens an app
is its verb. On a terminal, the verb alone opens its app on its first tab, and the verb with any of its app's tabs,
kit flow's included, opens it there maximized: `kit issues backlog` opens Issues on backlog, and with `--json`
refuses with 2, naming `kit flow backlog`. *Builder's call.*

### The #138 plan

This design revises the #138 plan. Its Tasks 1 to 4 and 8 stand. Tasks 5 to 7 become this design's Platform and
Trust stages: staged, marked, `{rows, errors}`, taking the workspace, Task 7's `kit policy --rules` kept as the
source. Task 9's planned line and People's hiding go, as no tab stays planned; its rules for a tab that is never
stale stay, for a stage whose mark is null. Task 10's `kit tui --app` goes; the six verbs replace it, and its README
rewrite moves to the last task. *Builder's call.*

## The forge seam: contract version 4

Each forge read below is a new call in `kit/modules/bash/forge.sh`, a feature in its table with `since 4`, and a
function in each provider. A provider that implements version 3 lacks every one of them, so the seam answers them
unsupported until the provider says 4. Each call prints one JSON object a line, whole or failing, as `open-issues`
does.

| Call | Feature | Fields it prints | GitHub | GitLab | Azure DevOps |
| --- | --- | --- | --- | --- | --- |
| `milestones` | milestones | `title, state, due, open, closed, url` | `repos/{o}/{r}/milestones?state=all` | `projects/:id/milestones` | lacks |
| `iterations` | iterations | `title, start, end, current, url` | the iteration fields of the repository's linked Projects | `projects/:id/iterations` | `az boards iteration team list` |
| `reviews` | reviews | `change, reviewer, state, unresolved, age, url` | each open change's `reviews` and `reviewThreads`, from the open-prs snapshot's read widened | `merge_requests/:n/approvals` and discussions | `az repos pr reviewer list` |
| `merged-prs` | merged | a change's open-prs fields and `merged, mergedBy` | `pullRequests(states: MERGED, last: 50)` | `merge_requests?state=merged` | `az repos pr list --status completed` |
| `artifacts` | artifacts | `name, digest, size, run, age, expires, url` | `repos/{o}/{r}/actions/artifacts`, the newest 100 | `projects/:id/jobs?scope=success` with `artifacts` | `az pipelines runs artifact list`, the newest 20 runs' |
| `environments` | environments | `name, protection, approvers, branches, url` | `repos/{o}/{r}/environments` | `projects/:id/environments` | `distributedtask/environments` |
| `deployments` | deployments | `environment, ref, sha, state, actor, age, url` | `repos/{o}/{r}/deployments` and each one's latest status | `projects/:id/deployments` | `environmentdeploymentrecords` |
| `collaborators` | people | `login, name, role, kind` | `repos/{o}/{r}/collaborators` | `projects/:id/members/all` | `az devops user list`, the project's |
| `teams` | teams | `team, members, role, url` | `repos/{o}/{r}/teams`, each one's members | the project's shared groups | `az devops team list` |
| `apps` | agents | `login, kind, last, url` | bots among recent commits', changes', and reviews' authors | bot users among the members | lacks |
| `attestations` | evidence | `subject, digest, kind, signer, age, url` | `repos/{o}/{r}/attestations` by each release asset's and artifact's digest | lacks | lacks |
| `*-alerts`, three calls | findings | `tool, rule, severity, state, path, age, url` | `code-scanning/alerts`, `dependabot/alerts`, `secret-scanning/alerts` | `vulnerability_findings`, off below the tier that has it | lacks |
| `rulesets` | controls | `control, target, enforcement, rules, source, url` | `repos/{o}/{r}/rulesets` and branch protection on the default branch | protected branches and push rules | branch policies (`az repos policy list`) |
| `bypasses` | waivers | `waives, for, kind, reason, by, until, url` | each ruleset's bypass actors, and dismissed alerts with their reason | lacks | lacks |

Decided after the builder's check, each the builder's call:

- **A feature off for one repository.** `lacks` stays the provider's static declaration. A call may also exit 3 at run
  time when the forge says this repository has the feature off: secret scanning disabled answers 404 with GitHub's
  "disabled" message, a GitLab tier without vulnerability findings. The seam passes 3 on, and the stage prints no
  rows and an `errors` entry `{feature, off}`; the tab draws its `empty` line and, dim, `<feature> is off here`.
  `forge.sh`'s contract text gains the case, and the conformance test holds each provider to it.
- **Findings are three calls.** `alerts` is `code-scanning-alerts`, `dependabot-alerts`, and `secret-scanning-alerts`,
  each its own feature. The `findings` stage merges them, `tool` saying which; a call that fails or is off goes under
  `errors` with its line, and the others draw. Each call stays whole or failing.
- **Iterations need no mapping.** GitHub reads the iteration fields of the Projects linked to the repository,
  `repository.projectsV2 { fields { ... on ProjectV2IterationField } }`, in one GraphQL request, the first 20
  projects; a repository with none answers none. The entities spec's `iteration` mapping stays later work.
- **Fan-out is capped, newest first, and says so.** A call that reads a list per row reads the newest N rows' lists and
  no more, as `runs` reads 20; the stage's document holds `limited: N` when it stopped there, and the tab's count says
  `N+`. GraphQL is used wherever it answers in one request. The caps: deployments, the 50 newest, their latest status
  in one GraphQL request; teams, 20 teams, each team's first 100 members; GitLab reviews, the open changes' approvals
  and discussions, already read per change by open-prs, reused; Azure DevOps deployment records, 20 environments;
  attestations, the digests of the 5 newest releases' assets and the 20 newest artifacts, each one request.
- **Agents are bots that acted.** `apps` lists the bot accounts, `__typename Bot` or a login ending `[bot]`, among the
  authors of the 100 newest default-branch commits and of the open and merged changes and their reviews, read in one
  GraphQL request. No installation list: it needs admin or app credentials.
- **Reviews hold both.** A row per change and reviewer: `requested` while the reviewer is asked and has not answered,
  else the reviewer's latest review's state, `approved`, `changes`, or `commented`; `unresolved` is the change's
  unresolved threads, on each of its rows. GitLab maps approvals to `approved` and a reviewer without one to
  `requested`; Azure DevOps maps votes 10, 5, -5, -10, and 0 to approved, approved, changes, changes, and requested.

The marks call gains the kinds these stages are stale by: `milestones` on GitHub reads one milestone sorted by
`updated`, and the rest reuse the marks the forge already reads, as the next table says. A call that answers a
permission refusal, as an alerts call does for a caller without security access, fails with its one line naming the scope
it needs; the tab shows the line, and the other tabs draw.

## kit's own reads

| Stage | Source |
| --- | --- |
| `policy` | `kit policy --rules --json`, the #138 plan's Task 7 |
| `toolchains`, `tools` | `kit toolchains --json` and `kit tools --json`, wrapped into `{rows, errors}` |
| `services` | the capabilities of the entities spec's Platform table that kit fills today: hosting, review, ci, releases (the forge), tracking (each tracker prefix), toolchain, dependencies, policy; each with its provider, whether it was inferred or set, and whether its tool is signed in |
| `settings` | every key kit reads from `kit.yml`, its value, and whether it was set or defaulted |

## Marks

| Stages | Mark |
| --- | --- |
| `milestones` | the forge's new milestones mark |
| `iterations`, `people`, `teams`, `agents`, `controls`, `waivers` | null: they reload on Refresh and Watch alone |
| `reviews`, `checks`, `merged` | the issues mark and the runs mark, as changes' |
| `artifacts`, `deployments`, `environments`, `evidence` | the runs mark, as a run makes each |
| `findings` | the runs mark, as a scan is a run |
| `policy`, `settings`, `services` | a hash of `kit.yml` and `.kit/policy/` |
| `toolchains`, `tools` | a hash of the toolchain's pin files and kit's chosen toolchain |

## kittui

Each stage is a catalog entry in `fields.json` with its `command`, `record` `.rows[]`, `key`, `empty`, and fields,
`staged` so the app reads `--stages` and `--marks`, and `takes: [workspace]`. `apps.go` names `<verb>/<stage>` where
it named `planned/<tab>`. `checks` reads the open-prs snapshot and needs no new forge call. A tab whose provider lacks
its feature draws the entry's `empty` line, as an unsupported call answers none. No tab stays planned, so the
`planned/` machinery and the #138 plan's Task 9 decision about hiding People go.

## Constraints

- The #138 plan's Tasks 1 to 4 come first: a tab must be one catalog entry before twenty are added.
- Peers share the worktree; commits name their paths, as the plan's Global Constraints say. Subjects start with the
  new issue's number for kit's seam and verbs, and `#138` for kittui's tabs.
- `tests/forge-conformance.bats` runs each new call against each provider's fake tool. Each provider page under
  `docs/integrations/` gains the call in its read table, and the schema of what it reads where one is published.
- Everything under `kit/` is written for a consumer, every option with a value is `--opt=value`, and kittui's rules
  hold: plain clicks, no circled glyphs, two-cell emoji in menus.
- Coverage stays at 98.0 or more in `projects/tui/`.

## Left to the builder

- Each stage's fields beyond the ones the table names, its presets, renderings, and glyphs in
  `kit/modules/jq/glyphs.json`.
- The order of the work across apps, and how the six verbs share code: one module of stages per verb, or one
  `entities.sh` the verbs call.
- How `reviews` widens the open-prs read without a second request per change.
