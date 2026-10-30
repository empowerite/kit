# forge.jq - a forge's answers, folded into the pull request document kit status renders (jq module)
# ----------------------------------------------------------------------------------------------------------------------
#
# import "forge" as forge; with jq -L "${kit_jq}". Each provider turns its forge's own fields into runs, and the
# rest is shared, so a pull request on GitHub and a merge request on GitLab fold into the same document:
#
#   number, ref, branch, title, author, createdAt
#   age         seconds since the change was created
#   state       draft, queued, conflicted, failing:gate, no-check, comments:open, blocked:approval, or ready
#   checks      {state, passed, total, failing, took}
#   unresolved  the number of unresolved review threads
#   closes      [{number, title}] of the issues it closes
#
# see: modules/bash/forge.sh, modules/bash/forge-github.sh, modules/bash/forge-gitlab.sh

# epoch reads an ISO 8601 time, with or without fractional seconds, and gives seconds since the epoch.
def epoch: sub("\\.[0-9]+"; "") | fromdateiso8601;

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

# document($now) reads {number, ref, branch, title, author, createdAt, draft, conflicted, approval, runs, unresolved,
# closes} and gives the document.
def document($now):
  (.runs | checks($now)) as $checks
  | { number, ref, branch, title, author, createdAt, age: ($now - (.createdAt | epoch)),
      state: (. + {checks: $checks} | state), checks: $checks, unresolved, closes };

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

# github_prs($now) reads the answer to forge-github:open-prs's GraphQL query and gives one document per pull request.
def github_prs($now):
  .data.repository.pullRequests.nodes[]
  | { number, ref: "#\(.number)", branch: .headRefName, title, author: (.author.login // ""), createdAt,
      draft: .isDraft, conflicted: (.mergeStateStatus == "DIRTY"),
      approval: ((.reviewDecision // "") | IN("REVIEW_REQUIRED", "CHANGES_REQUESTED")),
      runs: ([.commits.nodes[0].commit.statusCheckRollup.contexts.nodes[]?] | github_runs),
      unresolved: ([.reviewThreads.nodes[]? | select(.isResolved | not)] | length),
      closes: [.closingIssuesReferences.nodes[] | { number, title }] }
  | document($now);

# gitlab_runs reads a pipeline's jobs and gives its runs.
def gitlab_runs:
  map({ name,
        verdict: (if .status | IN("created", "pending", "running", "preparing", "waiting_for_resource", "scheduled")
                  then "pending" elif .status | IN("success", "skipped", "manual") then "passed" else "failed" end),
        at: (.started_at // .created_at // ""),
        since: (.started_at // .created_at // null | if . == null then null else epoch end),
        took: (if .started_at != null and .finished_at != null
               then (.finished_at | epoch) - (.started_at | epoch) else null end) });

# gitlab_pr($now; $jobs; $discussions; $closes) reads a merge request, with its pipeline's jobs, its discussions,
# and the issues it closes, and gives its document.
def gitlab_pr($now; $jobs; $discussions; $closes):
  { number: .iid, ref: "!\(.iid)", branch: .source_branch, title, author: (.author.username // ""),
    createdAt: .created_at, draft, conflicted: .has_conflicts,
    approval: ((.detailed_merge_status // "") == "not_approved"),
    runs: ($jobs | gitlab_runs),
    unresolved: ([$discussions[] | .notes[]? | select(.resolvable and (.resolved | not))] | length),
    closes: [$closes[] | { number: .iid, title }] }
  | document($now);
