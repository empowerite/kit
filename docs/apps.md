# App verbs

An app verb prints the tabs of one of kittui's apps, each a stage, in the shape `kit workspace` prints its own. Each
app's verb is named for it, and `modules/bash/apps.sh` runs every one of them: its stages, its marks, its forms, and
its hand-off to kittui. A verb's own stages live in `modules/bash/apps/<verb>.sh`.

## The shape every app verb shares

- `kit <verb> --stages` prints the stages, one per line, in the app's tab order.
- `kit <verb> --marks` prints a mark per stage as one JSON object. A mark is a text that changes when what its stage
  draws does, and `null` for a stage with none, which kittui reads again only on Refresh, as it offers no
  Watch for a tab kit never marks.
- `kit <verb> <stage> --json` prints the stage's document, `{rows, errors}`, and `limited: N` where a capped read
  stopped at N, the newest. `--jq` prints it formatted, and `--text`, or no form off a terminal, as a table.
- `kit <verb> --json`, with no stage, prints `{stages, <stage>: document…}`, a stage that could not be read `null`.
- On a terminal, with no form, `kit <verb>` opens the app in kittui, and `kit <verb> <stage>` opens it on that tab.
- `kit --workspace=<group> <verb>` runs each stage in every cloned member's checkout at once. Each row, and each
  error, gains `member`, its host and path, and a member that could not be read is `{member, error}` under `errors`.

A row is `{key, …, links}`: `key` is the row's own within its stage, and `links` gives where a value leads, by its
field. A stage that reads the forge goes through the forge seam, `modules/bash/forge.sh`. A primary with no forge, or
a forge that lacks the stage's feature, gives no rows and no errors. A forge that says the feature is off for this
repository gives no rows and the error `{feature, off: true}`. Neither is a failure. A read that fails prints its one
line and the verb exits 1.

A tab of the app that another command prints opens on a terminal as the verb's own do; with `--json`, the verb
refuses with 2 and names that command.

Each verb exits 0 when its stages are drawn, 1 when one could not be read, after what was, and 2 on a usage error,
or outside a repository's working tree with no workspace.

## Platform: kit platform

`kit platform` prints what kit runs with here. It asks the forge only whether its tool is signed in.

| Stage | A row per | Fields | Mark |
| --- | --- | --- | --- |
| `services` | capability kit fills | `capability, service, provider, chosen, ready` | a hash of `kit.yml`, the git configuration, `.kit/policy`, and what `toolchains` hashes |
| `toolchains` | toolchain and role, as `kit toolchains --json` lists them | `provider, role, usable, active, chosen, fills` | a hash of the pin files and kit's choice |
| `tools` | tool pinned, as `kit tools --json` lists them | `name, pinned, installed, from` | as `toolchains` |
| `settings` | key the settings schema names | `setting, value, from` | a hash of `kit.yml` |

- `services` holds `hosting`, `review`, `ci`, and `releases`, the primary's forge; `tracking <prefix>`, each tracker
  prefix; `toolchain`, the project toolchain; `dependencies <provider>`, each dependency provider whose manifest and
  lock are here, or `dependencies` alone for none; and `policy`, the policy engine. `service` is what answers it, as
  `github`, `mise`, or `patterns`, `null` for none: the forge's with no remote, or a `ci` the forge lacks. `provider`
  is kit's module that reaches it, as `forge/github`. `chosen` is `set` where `kit.yml` or a recorded choice names
  it, else `inferred`. `ready` is whether its tool is there and signed in: a forge whose tool is not signed in is not
  ready, and the stage still draws.
- `toolchains` keys each row by provider and role, `mise project`, and `tools` by where the tool is pinned and its
  name. Where `kit tools` refuses, as with several toolchains usable and none chosen, `tools` is its line and the verb
  exits 1; `toolchains` still draws.
- `settings` lists every key of `.kit/kit.schema.json`, in its order: the value `kit.yml` sets, `from: set`, else the
  schema's default, `null` where it names none, `from: default`.
- The toolchains' mark hashes every file a toolchain or a dependency provider pins from, and the files that keep
  kit's choice of toolchain: this checkout's, the clone's, and yours under `~/.kit/settings.yml`.

## Issues: kit issues

`kit issues` prints the planning the forge keeps. Its other tabs in kittui, the backlog and what is underway, are
`kit flow`'s: `kit issues backlog` and `kit issues underway` open on a terminal, and with a form refuse with 2, naming
`kit flow backlog --json` or `kit flow underway --json`.

| Stage | A row per | Fields | Mark |
| --- | --- | --- | --- |
| `milestones` | milestone, open and closed, by number | `number, title, state, due, open, closed` | the forge's mark of its milestones |
| `iterations` | iteration, by start | `id, title, start, end, current` | `null` |

- `milestones` keys each row by its number. `state` is `open` or `closed`, `due` seconds since the epoch, `null` for
  none, and `open` and `closed` count the milestone's issues. The title links to the milestone's page. Azure DevOps
  has no milestones, so its stage has no rows and no errors. Its mark is GitHub's ETag of the milestones list's
  first 100, GitLab's newest milestone and newest issue by `updated_at`, and none on Azure DevOps.
- `iterations` keys each row by its id. `start` and `end` are seconds since the epoch, `end` the last second of the
  last day, both `null` for an iteration with no dates. `current` is true for the one that holds now. The title links
  to the page of the project, the iteration, or the team's sprints. GitHub's come from the projects linked to the
  repository, none when none is linked, and need `gh`'s `read:project` scope; GitLab's include its groups', and a tier
  without iterations is the error `{feature: iterations, off: true}`; Azure DevOps' are its team's, the team
  `forge_settings.ado.team` in `kit.yml` names, else the project's default team. It has no mark, so kittui reads it
  again only on Refresh.

## Changes: kit changes

`kit changes` prints what the forge says of the repository's changes. Its other tab in kittui, the open changes, is
`kit flow`'s: `kit changes changes` opens on a terminal, and with a form refuses with 2, naming
`kit flow changes --json`.

| Stage | A row per | Fields | Mark |
| --- | --- | --- | --- |
| `reviews` | open change and reviewer | `change, title, login, state, at, unresolved` | the forge's issues and runs |
| `checks` | open change and check, its latest run | `change, title, name, verdict, at, took` | as `reviews` |
| `merged` | change merged last, 50 at most | `number, title, author, branch, merged, by` | as `reviews` |

- `reviews` and `checks` read the open changes as `kit flow changes` does, from the forge's snapshot of them, so they
  cost no request more: one GraphQL request a minute on GitHub. Azure DevOps names the reviewers in its list of pull
  requests. GitLab reads each open merge request's approvals beside its discussions, a request each, in every read of
  the open changes: `kit flow changes`, `underway`, and `backlog` pay it as well. On GitHub, a token without
  `read:org` is refused a team asked to review; the read asks again without teams, and draws no team among them.
  With `--fresh`, each reads the open changes afresh, as `kit flow changes --fresh` does, so kittui's Refresh on
  either tab shows a review given within the minute.
- `reviews` keys each row by the change's number and the reviewer's login, `3 bob`. `state` is `requested` while the
  reviewer is asked and has not answered, else the reviewer's latest review: `approved`, `changes`, or `commented`.
  `at` is when they answered, in seconds since the epoch, `null` while asked. `unresolved` is the change's unresolved
  threads, on each of its rows. The author's own comments are no review. GitHub names a team asked by its slug; a
  reviewer's latest approval or request for changes stands though they commented after it, and a dismissed review
  stands as a comment. GitLab's reviewer is `approved` with an approval, else `requested`. Azure
  DevOps' votes 10 and 5 are `approved`, -5 and -10 `changes`, 0 `requested`, and it keeps no time of a vote.
- `checks` keys each row by the change's number and the check's name, `3 a`. Of the runs of one name on the head
  commit, the latest by its start counts, so a rerun that passed supersedes a run that failed. `verdict` is `passed`,
  `failed`, or `pending`; `at` is when the run started and `took` how long it ran, in seconds. A forge whose provider
  lacks `check-runs`, as one written to a contract before 4, prints no runs on its changes: the stage has no rows,
  and `{feature: check-runs, off: true, lacked: true}` under `errors` says why, though `kit flow changes` counts
  their checks.
- `merged` keys each row by its number, and its title links to its page. `merged` is when, in seconds since the
  epoch, and `by` who merged it. GitHub reads the 50 pull requests updated last, as a merge updates one, and orders
  them by merge; GitLab orders by `merged_at`, from GitLab 17.2, and an older GitLab is read by update, as GitHub is;
  Azure DevOps lists its completed pull requests in an order no parameter sets, newest first in its reference's
  examples. Where the forge holds more than it read, the document is `limited`. A list read by update could miss a
  change merged before the oldest update in it, so it keeps only those merged since, and `limited` counts them.
- Each stage's mark is the forge's marks of its issues, which move with its changes, and of its runs, joined, as
  `kit flow changes`' are; Azure DevOps has no marks, so its are `null`.

## Delivery: kit delivery

`kit delivery` prints what the forge holds of the repository's delivery. Its other tabs in kittui, the releases and
the builds, are `kit flow`'s: `kit delivery releases` and `kit delivery builds` open on a terminal, and with a form
refuse with 2, naming `kit flow releases --json` or `kit flow builds --json`.

| Stage | A row per | Fields | Mark |
| --- | --- | --- | --- |
| `artifacts` | artifact a run kept, the newest 100 at most | `id, name, digest, size, run, age, expires` | the forge's runs |
| `environments` | deployment environment | `name, protection, approvers, branches` | the forge's environments and runs |
| `deployments` | deployment, the newest 50 | `id, environment, ref, sha, state, actor, age` | the forge's deployments and runs |

- `artifacts` keys each row by its id, `<run>/<id>` on Azure DevOps, which numbers an artifact within its run, and its
  name links to its page. `digest` is the forge's, `sha256:…`, and `null` on GitLab and Azure DevOps, which show none;
  `size` is in bytes, `run` the run that kept it, `age` seconds since it was made, and `expires` seconds since the
  epoch, `null` for none. GitHub reads its 100 newest in one request and leaves out those expired, which no longer
  download; GitLab the archives of its 100 newest successful jobs, in one, each named for its job and its file, as
  `package: artifacts.zip`; Azure DevOps the artifacts of its 20 newest runs, a request each, 100 at most, and keeps
  no expiry. Where the forge holds more, the document is `limited`, at the rows drawn.
- `environments` keys each row by its name, and its name links to its page. `protection` is the kinds of rule that
  guard it, sorted: `reviewers`, `wait`, `branches`, `deployers`, or the forge's own name for another. `approvers` are
  who may approve, by login, or the name the forge gives a team or a role. `branches` are the refs that may deploy
  there, a tag's as `refs/tags/…`, `protected branches` for GitHub's rule of that name, and `null` for any. GitHub
  reads the environments, then the branch policies of each one with its own, a request each, for 20 environments at
  most: the list stops before a 21st, `limited`. GitLab reads its protected environments beside, on Premium and
  Ultimate; a tier or a caller that cannot read them leaves `protection` and `approvers` `null`, and GitLab restricts
  no branch, so `branches` is `null`. Azure DevOps reads its environments, 100 at most, and their checks in one
  request: an Approval's approvers, and a branch control's branches.
- `deployments` keys each row by its id. `environment` links to its row of `environments`, `row:environments/<name>`,
  `sha` to the commit's page, and `state` to its log or page. `state` is `success`, `failure`, `running`, `waiting`,
  `cancelled`, or `inactive`; `actor` who made it; `age` seconds since. GitHub reads the 50 newest, each with its latest
  status, in one GraphQL request; GitLab its 51 newest, in one. Azure DevOps keeps them as each environment's deployment
  records, so it reads those of the 20 environments changed last, 51 each, and names each one's ref, commit, and actor
  from its run among the project's 100 newest, `null` for a run past them. Past 50, or past 20 environments, the
  document is `limited`.
- Each stage's mark holds the forge's mark of its runs, as a run makes each. `environments`' also holds the forge's mark
  of its environments, GitHub's ETag of the first 100, GitLab's ids, states, and updates of them, as a person may edit
  one; `deployments`' the forge's mark of its deployments, GitHub's ETag of the newest 50 and GitLab's deployment
  updated last, as another tool may deploy and a status may change. GitLab's protected environments move no mark.
  Azure DevOps has no marks, so its are `null`.

## People: kit people

`kit people` prints who works on the repository, as the forge says. Every tab of People is its own.

| Stage | A row per | Fields | Mark |
| --- | --- | --- | --- |
| `people` | person or bot with access | `login, name, role, kind` | none |
| `teams` | team with access, 20 at most | `team, name, role, members` | none |
| `agents` | bot that acted | `login, acted, last` | none |

- `people` keys each row by its login. `name` is `null` where the forge shows none, as GitHub's list mostly does.
  `role` is the forge's own: GitHub's `admin`, `maintain`, `write`, `triage`, `read`, or a custom role, the highest
  any source gives; GitLab's `owner`, `maintainer`, `developer`, `reporter`, `planner`, or `guest`, the highest of a
  member's memberships, its own, its groups', and those of the groups the project is shared with or invited to; and
  on Azure DevOps `admin` for an administrator of one of the project's teams, else `member`. `kind` is `user`, `bot`,
  or, on Azure DevOps, `group`, for a group a team holds. GitHub reads every page of its collaborators, and shows them
  only to a caller who may push; any other is told to ask an admin for write access. GitLab reads its members through
  GraphQL, which says which is a bot, 100 a page. Azure DevOps lists its users only for the whole organization, so
  its people are the members of the project's teams, as `teams` reads them, and the document is `limited` where that
  is.
- `teams` keys each row by the team, GitHub's slug, GitLab's group path, Azure DevOps' name, and the team links to its
  page. `role` is the access the team has, GitHub's as its collaborators' are named, GitLab's the share's, and `null`
  on Azure DevOps, whose teams hold none. `members` are their logins, the first 100, and `null` for a team the caller
  cannot see. GitHub reads the teams with access, one request, then the members of all of them in one GraphQL
  request, which needs the `read:org` scope: without it, the line names `gh auth refresh --scopes read:org`. A
  repository a user owns has none. GitLab's teams are the groups the project is shared with, read the same way;
  Azure DevOps reads the project's teams, then each one's members, a request each. Past 20 teams, or past 100 members
  of one, the document is `limited`, at the teams drawn.
- `agents` keys each row by its login, which links to its page where the forge gives one; GitHub gives none for a
  bot seen only in commits, whose authors carry no page. `acted` is what it did, `commit`, `change`, or
  `review`, and `last` when it last did, in seconds since the epoch; the last to act comes first. GitHub reads, in one
  GraphQL request, the authors of the 100 newest commits on the default branch, and of the 100 open changes and the 50
  merged changes updated last, with each one's latest reviews: a bot is a `Bot` account or a login that ends `[bot]`,
  as `dependabot[bot]`, and a commit is also a bot's when its author names itself one, as an agent's commit under a
  person's account does. A bot that acted only before those is not among them, and past 100 open changes, or 20
  reviewers of one, the document is `limited`, at the bots drawn. GitLab's are its bot users among the
  members, as a project's or a group's access token makes, `acted` `null`, as GitLab does not say, and `last` the day
  each last acted. Azure DevOps names no bots, and draws none.
- No stage has a mark: the forge's marks are read for every app at once, and a mark of who has access would cost each
  a request more, one a caller who may not push is refused. Each is read again on Refresh only.

## Trust: kit audit

`kit audit` prints what the repository can be trusted for: what the forge attests and finds, what guards the default
branch and who may get past it, and the rules kit's policy judges by. Every tab of Trust is its own. Its verb is not
`kit trust`, which trusts a clone.

| Stage | A row per | Fields | Mark |
| --- | --- | --- | --- |
| `evidence` | attestation of the newest releases' assets and artifacts | `subject, digest, kind, signer, age` | the forge's runs and releases |
| `findings` | code scanning, Dependabot, or secret scanning alert, open or dismissed | `tool, number, rule, severity, state, path, age` | the forge's runs and alerts |
| `controls` | ruleset or protection that guards the default branch | `id, control, target, enforcement, rules, source` | none |
| `waivers` | bypass of a control, and alert dismissed | `waives, actor, kind, reason, by, until, hidden` | none |
| `policy` | rule `kit policy` judges by, where it is in force | `source, organization, repository, branch, scope, rule, pattern, from, engine` | a hash of each checkout's `kit.yml` and `.kit/policy/` |

- `evidence` keys each row by its digest and its kind, and the subject links to the repository's attestations.
  GitHub reads the 5 newest releases' assets and the 20 newest artifacts, each digest once, 50 at most, and asks for
  each digest's attestations, a request each; past those, the document is `limited`. `kind` is the in-toto
  statement's predicate type, as `https://slsa.dev/provenance/v1`, and `signer` the workflow its provenance names,
  `<owner>/<repo>/<path>@<ref>`, or `github` for one GitHub made itself, as an immutable release's. GitHub may keep a
  bundle apart, compressed, which kit does not read: its kind is `null`. GitLab and Azure DevOps keep none kit reads.
- `findings` merges three reads, one per scanner, and keys each row by its `tool`, `code-scanning`, `dependabot`, or
  `secret-scanning`, and its number; the rule links to the alert's page. `rule` is what found it: a rule's id, an
  advisory's GHSA id, or a secret's type. `state` is `open` or `dismissed`, a secret resolved as a false positive,
  as won't fix, or as used in tests being dismissed. A scanner the forge has off for the repository, as secret
  scanning disabled, is `{feature, off: true}` under `errors`, and the tab says it is off here; one that refuses the
  caller is `{feature, error}`, its line, and the others still draw, the stage exiting 1. GitHub's line names
  `gh auth refresh --scopes security_events` for a login without the scope, else an admin of the repository. GitLab
  reads its vulnerability report, SAST for code scanning, dependency scanning for Dependabot's kind, and secret
  detection; a tier without it has each off. GitLab leaves out a vulnerability the caller may not see, refusing
  nothing, so the tab reads empty for such a caller. Azure DevOps keeps none kit reads.
- `controls` keys each row by its id. `target` is the refs it names, `enforcement` `active`, `evaluate`, or
  `disabled`, and `rules` its kinds of rule, sorted, or `null` where only an admin may see them, as GitHub's branch
  protection. `source` is `ruleset` or `protection`. GitHub reads its rulesets of branches, its own and its
  organization's, in one GraphQL request, those whose patterns name the default branch, then the branch's own
  protection where it has one; GitLab its protected branches that name it, and its push rules; Azure DevOps the
  default branch's policies, an optional one `evaluate`.
- `waivers` keys each row by what it waives and the actor. `waives` is a control's id, which links to its row of
  `controls`, or an alert, `<tool> <number>`, which links to its row of `findings`; the reason links to the waiver's
  page. A `bypass` names who may bypass and its mode, `always`, `pull_request`, or `exempt`; those GitHub hides from
  who may not edit the ruleset, and teams kit cannot name, as for a login without `read:org`, are one row of the
  control, keyed `<id> hidden`, its actor `null` and `hidden` how many. A `dismissal` names who dismissed it, and
  the reason with its comment. The bypasses are the forge's own: GitHub's rulesets' bypass actors and its
  protection's pull request allowances. The dismissals are the dismissed alerts `findings` reads, each scanner read
  once a process for both stages; a scanner off here, or one that refuses the caller, is under `errors` as in
  `findings`, and the rest still draw, the stage exiting 1 for a refusal. GitLab's are its dismissed
  vulnerabilities; Azure DevOps keeps none kit reads.
- `policy` is `kit policy --rules --json` in each checkout of the clone with a branch: the one here, its branch the
  one `kit policy` judges, and each worktree on a branch. The policy in force depends on the branch: on the default
  branch it is the checkout's own, and on a branch past the default it is the base's. A row says where it is in
  force: `source`, the primary's forge, `github`, `gitlab`, or `ado`; `organization`, the path before the repository,
  a group and its subgroups on GitLab; `repository`; and `branch`. A clone with no remote on a forge is its main
  checkout's directory. Another checkout is
  read as it is, nothing placed in it, and one that fails is `{branch, error}` under `errors` while the rest draw.
- `policy` merges the rows of a rule that agree, its pattern, from, and engine the same. From the widest place down,
  a place whose rows all agree is one row, at the narrowest place that holds them all: `scope` names it, `branch`,
  `repository`, `organization`, `source`, or `workspace` where even the sources differ, and the fields finer than it are
  `null`. So a rule every branch of a repository agrees on is one row at `repository`, and over a workspace a rule
  every member agrees on is one row, wider than any of them. Each row is keyed by the rule and the place. The mark
  holds each checkout's `kit.yml` and `.kit/policy/`, and on a branch past the default the base's too, which a fetch
  moves.
- `evidence` and `findings` move with the forge's runs, as a run makes an attestation and a scan. `evidence` moves
  with its releases too, and `findings` with its alerts: GitHub's mark of each scanner's alert updated last, which a
  dismissal and a new advisory move without a run, `off` or `refused` for a scanner that is either, so the others'
  still move it; and GitLab's count of open and dismissed vulnerabilities.
  `controls` and `waivers` have none, and are read again on Refresh only.

## Packages: kit packages

`kit packages` keeps its own forms beside the app's, as [packages.md](packages.md) says: alone, with a filter, or
with `--text`, it lists the packages as text, and `--json` with no stage prints one object a package, of the
repository here; `select`, `add`, and `remove` change the active set. Off a terminal, `kit packages` alone still
lists them. On a terminal, `kit packages` alone opens Packages in kittui, and the app's forms hold as for every app
verb: `--stages`, `--marks`, and its one stage, `kit packages packages`.

| Stage | A row per | Fields | Mark |
| --- | --- | --- | --- |
| `packages` | package in reach, as `kit packages` lists them | `name, state, origin, depends, includes, dependents, bundles, description` | a hash of `kit.yml` and `.kit/packages/` |

- `state` is `active`, `inactive`, or `excluded`, and `origin` is `release`, `repository`, or a source's name.
  `depends` and `includes` are the package's own; `dependents` are the active packages that depend on it, which
  `kit packages remove` refuses it for, and `bundles` the active bundles that include it. Where kit was never
  installed, the stage refuses with kit packages' own line.
- Packages' tab menu acts on the packages selected, each action a kit verb kittui runs: Add runs `kit packages add`
  on the selected packages that are not active; Remove… lists the active ones and asks before it runs
  `kit packages remove`, which takes their files out; Make Only These Active… lists what it keeps, adds, and takes
  out, and asks before it runs `kit packages select`. Over a workspace, each runs in its member's checkout.

see: [toolchains.md](toolchains.md), [workspaces.md](workspaces.md), [integrations/gh.md](integrations/gh.md),
[integrations/glab.md](integrations/glab.md), [integrations/az.md](integrations/az.md)
