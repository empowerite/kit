# forge.jq - a forge's answers, folded into the pull request document kit flow renders (jq module)
# ----------------------------------------------------------------------------------------------------------------------
# import "forge" as forge; with jq -L "${kit_jq}". Each provider turns its forge's own fields into runs, and the
# rest is shared, so a pull request on GitHub and a merge request on GitLab fold into the same document:
#
#   number, ref, branch, title, author, createdAt
#   fork        the owner of the repository the change's branch lives in, when that is not the base repository: a
#               login on GitHub, a namespace's path on GitLab; null for a branch in the base repository, or a fork
#               whose owner the forge does not name
#   age         seconds since the change was created
#   pushed      {age, by} of the latest push to the branch, as far as the forge tells it cheaply, or null when it
#               does not: age in seconds, and by a login, or null
#   state       draft, queued, conflicted, failing:gate, no-check, comments:open, blocked:approval, or ready
#   checks      {state, passed, total, failing, took}
#   unresolved  the number of unresolved review threads
#   labels      [{name, color}] of the labels on it, as labels gives them
#   closes      [{number, title, assignees, labels, age, source}] of the issues it closes, the assignees as logins,
#               so a change's issues are drawn as the backlog stage draws an issue without a second call to the forge
#
# see: modules/bash/forge.sh, modules/bash/forge/github.sh, modules/bash/forge/gitlab.sh

# epoch reads an ISO 8601 time, with or without fractional seconds, and gives seconds since the epoch.
def epoch: sub("\\.[0-9]+"; "") | fromdateiso8601;

# place($path) gives where an issue is, as a repository's path names it: the organization, the path up to its last
# segment, an owner on GitHub and a group, with its subgroups, on GitLab; and the repository, that last segment. Both
# are null without a path.
def place($path):
  if ($path // "") == "" then {organization: null, repository: null}
  else ($path | split("/")) as $p | {organization: ($p[:-1] | join("/")), repository: $p[-1]} end;

# hex reads a color as a forge writes it, with or without a #, in either case, and gives it as #rrggbb in lower case;
# null for none.
def hex: if (. // "") == "" then null else "#" + (ascii_downcase | ltrimstr("#")) end;

# labels reads a forge's labels, each a name alone, as GitLab gives one without its details, or an object with a name
# and a color, as GitHub gives one, and gives each as {name, color}: the color as hex gives it, or null where the
# forge gives none.
def labels:
  [.[]? | if type == "string" then {name: ., color: null} else {name, color: (.color | hex)} end];

# colored($colors) reads a change's document and gives each label on it, and on the issues it closes, that has no
# color the one $colors gives its name, as a project's label list does on GitLab, as hex gives it; a name it does not
# give stays null.
def colored($colors):
  def paint: map(.color //= ($colors[.name] | hex));
  .labels |= paint | .closes |= map(.labels |= paint);

# github_path reads a GitHub issue's url and gives its repository's owner/name, or null without one.
def github_path: (.url // "") | capture("^https?://[^/]+/(?<path>[^/]+/[^/]+)/issues/").path // null;

# gitlab_path reads a GitLab issue's full reference, group/…/project#7, and gives its project's path, or null without
# one.
def gitlab_path: (.references.full // "") | sub("#[0-9]+$"; "") | if . == "" then null else . end;

# checks($now) reads runs, [{name, verdict, at, since, took}], and gives {state, passed, total, failing, took}. A
# run's verdict is pending, passed, or failed; at orders the runs of one name, and the latest counts, as after a
# rerun; since is when it started or was queued, in epoch seconds; took is its own duration, or null. took is the
# slowest run's duration, or the wait so far while one is pending. failing names the first failed check.
def checks($now):
  (group_by(.name) | map(sort_by(.at) | last)) as $latest
  | ($latest | length) as $n
  | ([$latest[] | select(.verdict == "pending")] | length) as $running
  | ([$latest[] | select(.verdict == "passed")] | length) as $ok
  | ([$latest[] | select(.verdict == "failed") | .name] | first // "") as $failing
  | (if $n == 0 then "NONE" elif $failing != "" then "FAILURE" elif $running > 0 then "PENDING" else "SUCCESS" end)
    as $state
  | ([$latest[] | select(.verdict == "pending") | .since // empty] | min) as $waiting
  | ([$latest[] | .took // empty] | max) as $longest
  | { state: $state, passed: $ok, total: $n, failing: $failing,
      took: (if $state == "PENDING" then (if $waiting == null then null else $now - $waiting end) else $longest end) };

# state reads {draft, checks, conflicted, unresolved, approval} and gives the change's state: the first that holds,
# in the order the document lists them. approval is true while a review blocks the merge.
def state:
  if .draft then "draft"
  elif .checks.state == "PENDING" then "queued"
  elif .conflicted then "conflicted"
  elif .checks.state == "FAILURE" then "failing:gate"
  elif .checks.state == "NONE" then "no-check"
  elif .unresolved > 0 then "comments:open"
  elif .approval then "blocked:approval"
  else "ready" end;

# document($now) reads {number, ref, branch, fork, title, author, createdAt, pushed, draft, conflicted, approval, runs,
# unresolved, labels, closes} and gives the document. pushed is {at, by}, or null, and comes out with its age in place
# of at. labels are the forge's, as labels reads them. Each of closes is {number, title, assignees, labels, createdAt,
# source, organization, repository}, and comes out with its age; an issue an older answer gives without a time has a
# null age, one without a source, an organization, or a repository a null one, and one without labels none.
def document($now):
  (.runs | checks($now)) as $checks
  | { number, ref, branch, fork: (.fork // null), title, author, createdAt, age: ($now - (.createdAt | epoch)),
      pushed: (if .pushed == null then null else {age: ($now - (.pushed.at | epoch)), by: (.pushed.by // null)} end),
      state: (. + {checks: $checks} | state), checks: $checks, unresolved, labels: (.labels | labels),
      closes: [.closes[] | { number, title, assignees: (.assignees // []), labels: (.labels | labels),
                             source: (.source // null),
                             organization: (.organization // null), repository: (.repository // null),
                             age: (if .createdAt == null then null else $now - (.createdAt | epoch) end) }] };

# github_runs reads a commit's status check contexts, CheckRun and StatusContext alike, and gives its runs.
def github_runs:
  map((if .__typename == "CheckRun" then (if .status == "COMPLETED" then .conclusion else "PENDING" end)
       else .state end) as $verdict
      | { name: (.name // .context),
          verdict: (if $verdict | IN("PENDING", "EXPECTED") then "pending"
                    elif $verdict | IN("SUCCESS", "NEUTRAL", "SKIPPED") then "passed" else "failed" end),
          at: (.startedAt // .createdAt // ""),
          since: (.startedAt // .createdAt // null | if . == null then null else epoch end),
          took: (if .startedAt != null and .completedAt != null
                 then (.completedAt | epoch) - (.startedAt | epoch) else null end) });

# github_prs($now) reads the answer to forge/github:open-prs's GraphQL query and gives one document per pull request.
# GitHub tells no push's time without a further call, so pushed is the head commit's: when it was committed, and the
# login of its author, or null when the author's email belongs to no account. A closed issue's assignees are those
# the answer's open issues with an assignee give it; an issue they do not name has none: one with no assignee, one
# closed, one in another repository, or one past the hundred most recently updated. Its labels are its own, as the
# answer gives them under it.
def github_prs($now):
  .data.repository as $repository
  | ([$repository.issues.nodes[]? | {key: "\(.number)", value: [.assignees.nodes[]?.login]}] | from_entries)
    as $assigned
  | $repository.pullRequests.nodes[]
  | { number, ref: "#\(.number)", branch: .headRefName, title, author: (.author.login // ""), createdAt,
      fork: (if .isCrossRepository then .headRepositoryOwner.login // null else null end),
      pushed: (.commits.nodes[0].commit // {}
               | if .committedDate == null then null else {at: .committedDate, by: .author.user.login} end),
      draft: .isDraft, conflicted: (.mergeStateStatus == "DIRTY"),
      approval: ((.reviewDecision // "") | IN("REVIEW_REQUIRED", "CHANGES_REQUESTED")),
      runs: ([.commits.nodes[0].commit.statusCheckRollup.contexts.nodes[]?] | github_runs),
      unresolved: ([.reviewThreads.nodes[]? | select(.isResolved | not)] | length),
      labels: [.labels.nodes[]?],
      closes: [.closingIssuesReferences.nodes[]
               | { number, title, createdAt, source: "github", labels: [.labels.nodes[]?],
                   assignees: (if .repository.nameWithOwner == $repository.nameWithOwner
                               then $assigned["\(.number)"] // [] else [] end) }
                 + place(.repository.nameWithOwner)] }
  | document($now);

# github_partial reads the answer to forge/github:open-prs's GraphQL query and gives a line for each list the answer
# does not hold whole, as its page says it has a next one: a pull request's review threads, or the checks on its head
# commit. The line's fields, split by tabs, are the pull request's number, threads or checks, the cursor the page
# ended on, and the head commit's oid; nothing when the answer holds every list whole.
def github_partial:
  .data.repository.pullRequests.nodes[]
  | .number as $number
  | (.commits.nodes[0].commit // {}) as $commit
  | (.reviewThreads.pageInfo // {} | select(.hasNextPage == true)
     | "\($number)\tthreads\t\(.endCursor)\t\($commit.oid)"),
    ($commit.statusCheckRollup.contexts.pageInfo // {} | select(.hasNextPage == true)
     | "\($number)\tchecks\t\(.endCursor)\t\($commit.oid)");

# github_spent($now) reads the answer of GitHub's rate_limit endpoint and gives the wait for the limit that is spent,
# GraphQL's or REST's, the later to reset when both are: when it resets, in local time, and how long that is from
# $now. It gives nothing when neither is spent.
def github_spent($now):
  [.resources // {} | to_entries[] | select(.key | IN("graphql", "core")) | select(.value.remaining == 0)]
  | max_by(.value.reset) // empty
  | (((.value.reset - $now) / 60) | ceil | if . < 1 then 1 else . end) as $minutes
  | "wait until \(.value.reset | strflocaltime("%H:%M")), in \($minutes)m, when GitHub's"
    + " \({graphql: "GraphQL", core: "REST"}[.key]) rate limit resets";

# github_quota reads the answer of GitHub's rate_limit endpoint and gives the caller's limits that kit spends: GraphQL's
# points, which the open changes and issues take, first, then REST's requests, which the builds take, as rest.
def github_quota:
  { forge: "github",
    limits: ( [ .resources // {} | to_entries[] | select(.key | IN("graphql", "core"))
                | {name: (if .key == "core" then "rest" else .key end), limit: .value.limit,
                   remaining: .value.remaining, reset: .value.reset} ]
              | sort_by(.name != "graphql") ) };

# gitlab_quota reads an answer of GitLab's API with its headers, as glab api --include prints it, and gives the limit
# its RateLimit headers name: none when it sends none, as a GitLab without rate limits does, or a limit or what is left
# of it that is no number. A reset that is no number is null.
def gitlab_quota:
  def number: (tonumber? // null);
  ( [ split("\n")[] | capture("^ratelimit-(?<key>[a-z]+):\\s*(?<value>\\S+)"; "i")
      | {key: (.key | ascii_downcase), value} ] | from_entries ) as $h
  | ($h.limit // "" | number) as $limit | ($h.remaining // "" | number) as $remaining
  | { forge: "gitlab",
      limits: (if $limit == null or $remaining == null then []
               else [{name: ($h.name // "api"), limit: $limit, remaining: $remaining,
                      reset: ($h.reset // "" | number)}] end) };

# gitlab_marks($issues; $mrs; $runs; $releases) reads GitLab's newest issue and merge request by update, its ten newest
# pipelines, and its newest release, and gives a mark of each kind: what each says of its items and when they changed,
# as one text, so a mark changes when one of them does.
def gitlab_marks($issues; $mrs; $runs; $releases):
  { issues: ([$issues[]?, $mrs[]? | [.iid, .updated_at]] | tojson),
    runs: ([$runs[]? | [.id, .status, .updated_at]] | tojson),
    releases: ([$releases[]? | [.tag_name, .released_at]] | tojson) };

# gitlab_runs reads a pipeline's jobs and gives its runs.
def gitlab_runs:
  map({ name,
        verdict: (if .status | IN("created", "pending", "running", "preparing", "waiting_for_resource", "scheduled")
                  then "pending" elif .status | IN("success", "skipped", "manual") then "passed" else "failed" end),
        at: (.started_at // .created_at // ""),
        since: (.started_at // .created_at // null | if . == null then null else epoch end),
        took: (if .started_at != null and .finished_at != null
               then (.finished_at | epoch) - (.started_at | epoch) else null end) });

# gitlab_pipeline_jobs reads a merge request and gives its head pipeline as one job, named pipeline, with the
# pipeline's own status and times: the checks of a pipeline whose jobs cannot be read, as a fork's may not be.
def gitlab_pipeline_jobs:
  [.head_pipeline | {name: "pipeline", status, created_at, started_at, finished_at}];

# gitlab_pr($now; $jobs; $discussions; $closes; $fork) reads a merge request, with its pipeline's jobs, its
# discussions, the issues it closes, and the owner of the fork its branch lives in, or null, and gives its document.
# GitLab names a label without its color here, which colored then gives it.
# GitLab tells no push's time without a further call, so pushed is the head pipeline's: when it was created, and who
# started it; null when the branch has no pipeline.
def gitlab_pr($now; $jobs; $discussions; $closes; $fork):
  { number: .iid, ref: "!\(.iid)", branch: .source_branch, title, author: (.author.username // ""),
    createdAt: .created_at, draft, conflicted: .has_conflicts, fork: $fork,
    pushed: (.head_pipeline // {}
             | if .created_at == null then null else {at: .created_at, by: (.user.username // null)} end),
    approval: ((.detailed_merge_status // "") == "not_approved"), labels,
    runs: ($jobs | gitlab_runs),
    unresolved: ([$discussions[] | .notes[]? | select(.resolvable and (.resolved | not))] | length),
    closes: [$closes[] | { number: .iid, title, assignees: [.assignees[]?.username], labels, createdAt: .created_at,
                           source: "gitlab" } + place(gitlab_path)] }
  | document($now);

# issue($now; $source; $path) reads {number, title, assignees, labels, createdAt}, the assignees as logins, and gives
# the issue document kit backlog draws: those, the labels as labels reads them, ref, age in seconds, source, the
# tracker that holds it: the code forge's own, github or gitlab, for now; and the organization and the repository it
# is in, as place reads $path.
def issue($now; $source; $path):
  { number, ref: "#\(.number)", title, assignees, labels: (.labels | labels), createdAt,
    age: ($now - (.createdAt | epoch)), source: $source }
  + place($path);


# github_issues($now) reads gh issue list's JSON, the answer to forge/github:open-issues, and gives one document per
# open issue, oldest first.
def github_issues($now):
  sort_by(.createdAt, .number)[] | github_path as $path
  | {number, title, assignees: [.assignees[].login], labels, createdAt}
  | issue($now; "github"; $path);

# gitlab_issues($now) reads GitLab's issues, the answer to forge/gitlab:open-issues, and gives the same documents. The
# answer gives each label's details, its color among them.
def gitlab_issues($now):
  sort_by(.created_at, .iid)[] | gitlab_path as $path
  | {number: .iid, title, assignees: [.assignees[].username], labels, createdAt: .created_at}
  | issue($now; "gitlab"; $path);

# release($now) shapes a release found on either forge into the document the releases stage draws: version (the tag without
# release/), tag, name, kind (release, pre-release, or draft), latest, at (when it was published, or made, for a
# draft), and age in seconds. It is given an object with tag, name, kind, latest, and at.
def release($now):
  {version: (.tag | sub("^release/"; "")), tag, name, kind, latest, at, age: ($now - (.at | epoch))};

# github_releases($now) reads gh release list's answer and gives one document per release, newest first.
def github_releases($now):
  map({tag: .tagName, name, latest: .isLatest, at: (.publishedAt // .createdAt),
      kind: (if .isDraft then "draft" elif .isPrerelease then "pre-release" else "release" end)})
  | sort_by(.at) | reverse[] | release($now);

# gitlab_releases($now) reads GitLab's releases and gives the same documents. GitLab marks no pre-release, so a tag
# with a suffix is one, as kit release names it; a release not yet due is a draft; the latest is the newest release.
def gitlab_releases($now):
  map({tag: .tag_name, name, at: .released_at,
      kind: (if .upcoming_release then "draft" elif (.tag_name | test("-")) then "pre-release" else "release" end)})
  | sort_by(.at) | reverse
  | (map(select(.kind == "release")) | first | .tag) as $latest
  | .[] | . + {latest: (.tag == $latest)} | release($now);

# build($now) shapes a run found on either forge into the document the builds stage draws. It is given id, number,
# attempt, workflow, branch, tag, event, sha, title, state, createdAt, and ended, when the run finished, or null while
# it runs; and started, when it began. It gives id; number, the forge's count of the workflow's runs, or null; attempt,
# 1 for a first run; workflow; branch; tag, true when the branch is a tag; event; sha, the commit's short hash, and
# title, the forge's headline for the run, each empty when unknown; state; age and took in seconds, took null while
# it runs; url; and jobs, empty here.
def build($now):
  { id, number, attempt: (.attempt // 1), workflow, branch, tag, event, sha: ((.sha // "")[:7]),
    title: (.title // ""), state, age: ($now - (.createdAt | epoch)),
    took: (if .state == "running" or .ended == null then null
           else (.ended | epoch) - ((.started // .createdAt) | epoch) end),
    url, jobs: [] };

# builds($default; $tags) keeps the runs not owned by any pull request, newest first, the 20 newest at most: a push to
# the default branch or to a tag, and a scheduled or manually started run on any branch. Each is marked as on a tag or
# not.
def builds($default; $tags):
  map(.tag = (.tag == true or (.branch | IN($tags[]))))
  | map(select((.event == "push" and (.branch == $default or .tag))
               or (.event | IN("schedule", "workflow_dispatch"))))
  | sort_by(.createdAt) | reverse | .[:20];

# github_build_state reads a GitHub run or job, its status and conclusion, and gives running, passed, failed, or
# cancelled.
def github_build_state:
  if .status != "completed" then "running"
  elif .conclusion | IN("success", "neutral") then "passed"
  elif .conclusion | IN("cancelled", "skipped", "stale") then "cancelled"
  else "failed" end;

# github_builds($now; $default; $tags) reads gh run list's answer and gives the builds stage's runs, as one array.
def github_builds($now; $default; $tags):
  map({ id: .databaseId, number, attempt, workflow: .workflowName, branch: .headBranch, event, sha: .headSha,
        title: .displayTitle, state: github_build_state, createdAt, started: .startedAt, ended: .updatedAt, url })
  | builds($default; $tags) | map(build($now));

# github_jobs reads gh run view's jobs and gives each as {name, state, took, step}: took null while it runs, and step
# the name of the first step that failed, or null.
def github_jobs:
  [.jobs[]? | { name, state: github_build_state,
                took: (if .status == "completed" and .startedAt != null and .completedAt != null
                       then (.completedAt | epoch) - (.startedAt | epoch) else null end),
                step: ([.steps[]? | select(.conclusion == "failure") | .name] | first) }];

# gitlab_build_state reads a GitLab pipeline's status and gives running, passed, failed, or cancelled. A pipeline
# that waits for a person, manual, is cancelled: it does not run on its own.
def gitlab_build_state:
  if .status | IN("created", "waiting_for_resource", "preparing", "pending", "running", "scheduled") then "running"
  elif .status == "success" then "passed"
  elif .status == "failed" then "failed"
  else "cancelled" end;

# gitlab_builds($now; $default; $tags) reads GitLab's pipelines and gives the same runs. GitLab names the event the
# source, and a pipeline started from the web page is workflow_dispatch here; a pipeline has a name only when the
# project gives one. A pipeline marked as a tag's counts as one, as does a ref named as a local tag. The list gives no
# start, so took counts from creation to the end, or the last update; nor a title, so a pipeline has none; its number
# is the project's, iid.
def gitlab_builds($now; $default; $tags):
  map({ id, number: .iid, workflow: (.name // "pipeline"), branch: .ref, sha,
        event: ({push: "push", schedule: "schedule", web: "workflow_dispatch"}[.source] // .source),
        state: gitlab_build_state, createdAt: .created_at, started: .started_at,
        ended: (.finished_at // .updated_at), url: .web_url,
        tag: (.tag // false) })
  | builds($default; $tags + [.[] | select(.tag) | .branch]) | map(build($now));

# gitlab_branch_for_issue($n) reads GitLab's open merge requests and gives the first source branch that starts with
# the issue's number and a dash, or nothing.
def gitlab_branch_for_issue($n):
  [.[] | select(.source_branch | startswith("\($n)-"))] | .[0].source_branch // empty;

# gitlab_pr_state reads GitLab's merge requests for a branch, newest first, and gives the newest one's state in
# GitHub's words: MERGED, CLOSED for a closed or locked one, else OPEN; nothing when there is none.
def gitlab_pr_state:
  .[0].state // empty | {merged: "MERGED", closed: "CLOSED", locked: "CLOSED"}[.] // "OPEN";

# ado_closed reads a work item and gives true when its state is a closing one of Azure Boards' process templates:
# Closed, Done, or Removed, as the ado tracker reads them.
def ado_closed: (.fields["System.State"] // "") | IN("Closed", "Done", "Removed");

# ado_assignees reads a work item and gives its assignee's login as a list: Azure Boards has one assignee at most, named
# by the unique name az signs in as.
def ado_assignees: [.fields["System.AssignedTo"] | objects | .uniqueName // empty];

# ado_tags reads a work item's tags, Azure Boards' labels, one text split by "; ", and gives their names.
def ado_tags: (.fields["System.Tags"] // "") | split(";") | map(ltrimstr(" ") | rtrimstr(" ")) | map(select(. != ""));

# ado_issues($now; $path) reads the work items of an Azure Boards query, the answer to forge/ado:open-issues, and gives
# one issue document per work item, oldest first. $path is the repository's, project/repository, where place reads
# the project as the organization, as forge/ado:repo names its owner.
def ado_issues($now; $path):
  sort_by(.fields["System.CreatedDate"], .id)[]
  | {number: .id, title: .fields["System.Title"], assignees: ado_assignees, labels: ado_tags,
     createdAt: .fields["System.CreatedDate"]}
  | issue($now; "ado"; $path);

# ado_verdict reads a policy evaluation's status and gives the check's verdict: approved is passed, rejected and broken
# are failed, and queued and running are pending.
def ado_verdict:
  if .status | IN("approved") then "passed" elif .status | IN("rejected", "broken") then "failed" else "pending" end;

# ado_runs($policies) reads a pull request's policy evaluations and gives its runs: each build policy's evaluation,
# named for its pipeline, and each status policy's, named for its status. A policy that does not apply is no check.
def ado_runs($policies):
  [$policies[] | select(.status != "notApplicable")
   | select((.configuration.type.displayName // "") | IN("Build", "Status"))
   | { name: (.context.buildDefinitionName // .configuration.settings.statusName // .configuration.settings.displayName
              // .configuration.type.displayName),
       verdict: ado_verdict,
       at: (.startedDate // .completedDate // ""),
       since: (.startedDate // null | if . == null then null else epoch end),
       took: (if .startedDate != null and .completedDate != null
              then (.completedDate | epoch) - (.startedDate | epoch) else null end) }];

# ado_blocked($policies) reads a pull request's policy evaluations and gives true while a blocking policy that is no
# build or status, such as a minimum number of reviewers, has not approved it.
def ado_blocked($policies):
  any($policies[]; (.configuration.isBlocking // false)
      and ((.configuration.type.displayName // "") | IN("Build", "Status") | not)
      and (.status | IN("approved", "notApplicable") | not));

# ado_unresolved($threads) reads a pull request's threads, as the REST API gives them, and counts those a person opened
# and nobody has resolved: active or pending, not deleted, and not one Azure DevOps writes itself, such as a vote.
def ado_unresolved($threads):
  [$threads.value[]? | select((.isDeleted // false) | not) | select(.status | IN("active", "pending"))
   | select(((.comments // [])[0].commentType // "text") == "text")] | length;

# ado_pr($now; $policies; $threads; $closes; $path) reads a pull request, its policy evaluations, its threads, and the
# work items linked to it, and gives its document. Azure DevOps tells no push's time without a further call, so pushed
# is null. A pull request from a fork names the fork's repository as its owner.
def ado_pr($now; $policies; $threads; $closes; $path):
  { number: .pullRequestId, ref: "!\(.pullRequestId)", branch: (.sourceRefName | ltrimstr("refs/heads/")), title,
    author: (.createdBy.uniqueName // ""), createdAt: .creationDate, draft: (.isDraft // false),
    conflicted: (.mergeStatus == "conflicts"), fork: (.forkSource.repository.name // null), pushed: null,
    approval: ado_blocked($policies), labels: [.labels[]? | .name],
    runs: ado_runs($policies), unresolved: ado_unresolved($threads),
    closes: [$closes[] | { number: .id, title: .fields["System.Title"], assignees: ado_assignees, labels: ado_tags,
                           createdAt: .fields["System.CreatedDate"], source: "ado" } + place($path)] }
  | document($now);

# ado_pr_state reads a pull request and gives its state in GitHub's words: completed is MERGED, abandoned is CLOSED, and
# active is OPEN.
def ado_pr_state: {completed: "MERGED", abandoned: "CLOSED"}[.status] // "OPEN";

# ado_latest reads Azure DevOps' pull requests for a branch and gives the newest, by its number, or nothing.
def ado_latest: max_by(.pullRequestId) // empty;

# ado_build_state reads a pipeline run and gives running, passed, failed, or cancelled. A run that partly succeeded
# passed, as its pipeline lets it.
def ado_build_state:
  if .status | IN("inProgress", "notStarted", "postponed", "cancelling") then "running"
  elif .result | IN("succeeded", "partiallySucceeded") then "passed"
  elif .result == "failed" then "failed"
  else "cancelled" end;

# ado_builds($now; $default; $tags) reads az pipelines runs list's answer and gives the builds stage's runs. Azure
# Pipelines names the event the reason: a continuous integration build is a push, a manual one workflow_dispatch.
# The run's number is its build number, which a pipeline formats as it likes, so it is null when it is no count.
def ado_builds($now; $default; $tags):
  map({ id, number: (.buildNumber // "" | if test("^[0-9]+$") then tonumber else null end),
        workflow: (.definition.name // "pipeline"),
        branch: ((.sourceBranch // "") | sub("^refs/(heads|tags)/"; "")),
        tag: ((.sourceBranch // "") | startswith("refs/tags/")), sha: .sourceVersion,
        event: ({individualCI: "push", batchedCI: "push", schedule: "schedule", manual: "workflow_dispatch",
                 pullRequest: "pull_request"}[.reason] // .reason),
        title: (.triggerInfo["ci.message"] // ""), state: ado_build_state, createdAt: .queueTime,
        started: .startTime, ended: .finishTime, url: (._links.web.href // null) })
  | builds($default; $tags) | map(build($now));
