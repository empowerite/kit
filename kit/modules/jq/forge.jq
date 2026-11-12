# forge.jq - a forge's answers, folded into the pull request document kit flow renders (jq module)
# ----------------------------------------------------------------------------------------------------------------------
# import "forge" as forge; with jq -L "${kit_jq}". Each provider turns its forge's own fields into runs, and the
# rest is shared, so a pull request on GitHub and a merge request on GitLab fold into the same document:
#
#   number, ref, branch, title, author, createdAt
#   source      the forge the change is on: github, gitlab, or ado
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
#               so a change's issues are drawn as the issues stage draws an issue without a second call to the forge
#   named       the references a closing keyword names in its body, as named reads them: AB#4521 of Closes AB#4521,
#               how a change closes an issue of a tracker the forge does not link by itself
#   reviewers   [{login, state, at}] of the people and teams asked to review it, or who did: state is requested while
#               one is asked and has not answered, else approved, changes, or commented; at, in seconds since the
#               epoch, is when they answered, null while asked, or where the forge keeps no time
#   runs        [{name, verdict, at, took}] of every check run on its head commit, those a rerun superseded among
#               them, which checks counts: at in seconds since the epoch, or null
#
# see: modules/bash/forge.sh, modules/bash/forge/github.sh, modules/bash/forge/gitlab.sh

# epoch reads an ISO 8601 time, with or without fractional seconds, in UTC or at an offset, as az prints one,
# 2026-10-01T09:00:00.123000+00:00, and gives seconds since the epoch. A time it cannot read is an error.
def epoch:
  (capture("^(?<at>[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2})(\\.[0-9]+)?(?<zone>Z|(?<sign>[+-])(?<h>[0-9]{2}):?(?<m>[0-9]{2}))$")
    // error("not an ISO 8601 time: \(.)"))
  | (.at + "Z" | fromdateiso8601)
    - (if .zone == "Z" then 0 else (if .sign == "-" then -1 else 1 end) * ((.h | tonumber) * 3600 + (.m | tonumber) * 60) end);

# place($path) gives where an issue is, as a repository's path names it: the organization, the path up to its last
# segment, an owner on GitHub and a group, with its subgroups, on GitLab; and the repository, that last segment. Both
# are null without a path.
def place($path):
  if ($path // "") == "" then {organization: null, repository: null}
  else ($path | split("/")) as $p | {organization: ($p[:-1] | join("/")), repository: $p[-1]} end;

# from($here) gives a row, an object, with where it is from: its own organization, repository, and source, each that
# it has, even null, as an issue says the repository it is in; else that of $here, {organization, repository, source},
# the repository kit read it in and the forge or the service it was read from. Anything but an object is as it was.
def from($here):
  if type == "object" then . as $row | . + ($here | with_entries(select(.key as $k | $row | has($k) | not)))
  else . end;

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

# reviewers($author) reads a change's reviewers, each {login, state, at}, at an ISO 8601 time or null, in the order
# their states count, and gives each reviewer once, by login, as the first that names them says: a reviewer asked
# again is asked, whatever they said before. The change's author, whose own comments are no review, is none, and so is
# a reviewer the forge names no login for, as a deleted account. at comes out in seconds since the epoch.
def reviewers($author):
  reduce (.[] | select(.login != null and .login != $author)) as $r ([];
    if any(.[]; .login == $r.login) then . else . + [$r] end)
  | map({login, state, at: (if .at == null then null else .at | epoch end)}) | sort_by(.login);

# named reads a change's body, or null, and gives the references a closing keyword names in it, each once, in the order
# it names them: Closes AB#4521, fixes #7, and Resolves: LIN#12 name AB#4521, #7, and LIN#12, as kit work writes the
# line, and modules/bash/tracker.sh's tracker:body-refs reads it.
def named:
  [(. // "") | scan("(?i)(?:^|[^a-z0-9])(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?):? +([^\\s,;()]*[0-9]+)") | .[0]]
  | reduce .[] as $ref ([]; if index([$ref]) then . else . + [$ref] end);

# document($now) reads {number, ref, source, branch, fork, title, author, createdAt, pushed, draft, conflicted,
# approval, runs, unresolved, labels, closes, reviewers, body} and gives the document, the body as named reads it. pushed is {at, by}, or null, and comes
# out with its age in place of at. labels are the forge's, as labels reads them. Each of closes is {number, title,
# assignees, labels, createdAt, source, organization, repository}, and comes out with its age; an issue an older answer
# gives without a time has a null age, one without a source, an organization, or a repository a null one, and one
# without labels none. reviewers are as reviewers reads them, none where the answer names none; each run comes out with
# at in seconds since the epoch, null where it was empty, and without since.
def document($now):
  (.runs | checks($now)) as $checks
  | { number, ref, source: (.source // null), branch, fork: (.fork // null), title, author, createdAt,
      age: ($now - (.createdAt | epoch)),
      pushed: (if .pushed == null then null else {age: ($now - (.pushed.at | epoch)), by: (.pushed.by // null)} end),
      state: (. + {checks: $checks} | state), checks: $checks, unresolved, labels: (.labels | labels),
      closes: [.closes[] | { number, title, assignees: (.assignees // []), labels: (.labels | labels),
                             source: (.source // null),
                             organization: (.organization // null), repository: (.repository // null),
                             age: (if .createdAt == null then null else $now - (.createdAt | epoch) end) }],
      named: (.body | named),
      reviewers: (.author as $author | .reviewers // [] | reviewers($author)),
      runs: [.runs[] | {name, verdict, at: (if (.at // "") == "" then null else .at | epoch end), took}] };

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
# answer gives them under it. Its reviewers are those it asks, a person by login and a team by slug, then each
# reviewer's latest review that approves or asks for changes, which a comment after it leaves standing, as GitHub's
# latestOpinionatedReviews gives them, a dismissed review standing as a comment; then each other reviewer's latest
# review, as latestReviews gives them: a comment, or one of those where the answer holds no opinionated reviews; a
# pending review, which only its author sees, is none.
def github_prs($now):
  .data.repository as $repository
  | ([$repository.issues.nodes[]? | {key: "\(.number)", value: [.assignees.nodes[]?.login]}] | from_entries)
    as $assigned
  | $repository.pullRequests.nodes[]
  | { number, ref: "#\(.number)", source: "github", branch: .headRefName, title, author: (.author.login // ""),
      createdAt, body,
      fork: (if .isCrossRepository then .headRepositoryOwner.login // null else null end),
      pushed: (.commits.nodes[0].commit // {}
               | if .committedDate == null then null else {at: .committedDate, by: .author.user.login} end),
      draft: .isDraft, conflicted: (.mergeStateStatus == "DIRTY"),
      approval: ((.reviewDecision // "") | IN("REVIEW_REQUIRED", "CHANGES_REQUESTED")),
      runs: ([.commits.nodes[0].commit.statusCheckRollup.contexts.nodes[]?] | github_runs),
      unresolved: ([.reviewThreads.nodes[]? | select(.isResolved | not)] | length),
      labels: [.labels.nodes[]?],
      reviewers: ([.reviewRequests.nodes[]? | .requestedReviewer // {}
                   | {login: (.login // .slug), state: "requested", at: null}]
                  + ([.latestOpinionatedReviews.nodes[]?] as $opinions
                     | [$opinions[], (.latestReviews.nodes[]?
                                      | select(.author.login as $l | all($opinions[]; .author.login != $l)))]
                     | map(select(. != null and .state != "PENDING")
                           | { login: .author.login,
                               state: ({APPROVED: "approved", CHANGES_REQUESTED: "changes"}[.state]
                                       // "commented"),
                               at: (.submittedAt // null) }))),
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

# gitlab_mark($kind) reads the lists GitLab answers a kind of mark from, {issues, mrs, runs, releases, milestones,
# deployments, environments}, each those the kind needs: its newest issue and merge request by update, its ten newest
# pipelines, its newest release, its newest milestone by update, its deployment updated last, and its first 100
# environments, and gives the kind's mark: what each says of its items and when they changed, as one JSON text, so a
# mark changes when one of them does. A milestone's own update does not move when its issues close, so its mark holds
# the newest issue too.
def gitlab_mark($kind):
  if $kind == "issues" then [.issues[]?, .mrs[]? | [.iid, .updated_at]]
  elif $kind == "runs" then [.runs[]? | [.id, .status, .updated_at]]
  elif $kind == "releases" then [.releases[]? | [.tag_name, .released_at]]
  elif $kind == "milestones" then [.milestones[]?, .issues[]? | [.iid, .updated_at]]
  elif $kind == "deployments" then [.deployments[]? | [.id, .status, .updated_at]]
  else [.environments[]? | [.id, .name, .state, .updated_at]] end
  | tojson;

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

# gitlab_pr($now; $jobs; $discussions; $closes; $fork; $approvals) reads a merge request, with its pipeline's jobs,
# its discussions, the issues it closes, the owner of the fork its branch lives in, or null, and its approvals, and
# gives its document. GitLab names a label without its color here, which colored then gives it.
# GitLab tells no push's time without a further call, so pushed is the head pipeline's: when it was created, and who
# started it; null when the branch has no pipeline. Its reviewers are those who approved it, at the time each did, then
# each reviewer it names without an approval, asked.
def gitlab_pr($now; $jobs; $discussions; $closes; $fork; $approvals):
  { number: .iid, ref: "!\(.iid)", source: "gitlab", branch: .source_branch, title,
    author: (.author.username // ""), createdAt: .created_at, body: .description, draft, conflicted: .has_conflicts,
    fork: $fork,
    pushed: (.head_pipeline // {}
             | if .created_at == null then null else {at: .created_at, by: (.user.username // null)} end),
    approval: ((.detailed_merge_status // "") == "not_approved"), labels,
    runs: ($jobs | gitlab_runs),
    unresolved: ([$discussions[] | .notes[]? | select(.resolvable and (.resolved | not))] | length),
    reviewers: ([$approvals.approved_by[]? | {login: .user.username, state: "approved", at: (.approved_at // null)}]
                + [.reviewers[]? | {login: .username, state: "requested", at: null}]),
    closes: [$closes[] | { number: .iid, title, assignees: [.assignees[]?.username], labels, createdAt: .created_at,
                           source: "gitlab" } + place(gitlab_path)] }
  | document($now);

# issue($now; $source; $path) reads {number, title, assignees, labels, createdAt, milestone, iteration}, the assignees
# as logins, the milestone and the iteration each a title or null, and gives the issue document kit flow issues draws:
# those, the labels as labels reads them, ref, age in seconds, source, the tracker that holds it: the code forge's own,
# github or gitlab, for now; and the organization and the repository it is in, as place reads $path.
def issue($now; $source; $path):
  { number, ref: "#\(.number)", title, assignees, labels: (.labels | labels), createdAt,
    milestone: (.milestone // null), iteration: (.iteration // null),
    age: ($now - (.createdAt | epoch)), source: $source }
  + place($path);

# stood_in($choice) reads an issue document and gives the value a stand-in for one of its fields chooses: with
# label_prefix, the rest of the name of its first label that starts with the prefix, and holds more than it; else the
# default; else null.
def stood_in($choice):
  $choice.label_prefix as $prefix
  | first(if $prefix == null then empty
          else (.labels // [])[] | .name | select(startswith($prefix) and length > ($prefix | length))
               | ltrimstr($prefix) end)
    // $choice.default // null;

# issue_fields($choices; $lacks) reads an issue document and gives it its milestone, iteration, and swimlane, each the
# field of a feature: the stand-in's value where $choices, the repository's stand-ins by feature, holds one for it,
# null where $lacks, the features the forge lacks, names it, else the forge's own value, or null.
def issue_fields($choices; $lacks):
  reduce (["milestone", "milestones"], ["iteration", "iterations"], ["swimlane", "swimlanes"]) as [$field, $feature]
    (.;
     if $choices[$feature] != null then .[$field] = stood_in($choices[$feature])
     elif any($lacks[]; . == $feature) then .[$field] = null
     else .[$field] = (.[$field] // null) end);

# github_issues($now) reads gh issue list's JSON, the answer to forge/github:open-issues, and gives one document per
# open issue, oldest first. Each issue's iteration is the one forge/github:open-issues puts beside it, from the projects
# it is in, as gh issue list does not say it.
def github_issues($now):
  sort_by(.createdAt, .number)[] | github_path as $path
  | {number, title, assignees: [.assignees[].login], labels, createdAt, milestone: .milestone.title?, iteration}
  | issue($now; "github"; $path);

# github_issue_iterations reads the pages of GitHub's GraphQL answer to the open issues' project items, each a page of
# forge/github:issue-iterations, and gives one object of each issue's number, as a text, to the title of its
# iteration: the first value of an iteration field among the projects it is in. An issue in none is not in it.
def github_issue_iterations:
  [.[].data.repository.issues.nodes[]
   | {key: (.number | tostring),
      value: first((.projectItems.nodes // [])[] | (.fieldValues.nodes // [])[] | .title? // empty)}
  ]
  | from_entries;

# gitlab_issues($now) reads GitLab's issues, the answer to forge/gitlab:open-issues, and gives the same documents. The
# answer gives each label's details, its color among them, and each issue's milestone and iteration; an iteration of a
# cadence with no title is named by its dates, as gitlab_iterations names it.
def gitlab_issues($now):
  sort_by(.created_at, .iid)[] | gitlab_path as $path
  | {number: .iid, title, assignees: [.assignees[].username], labels, createdAt: .created_at,
     milestone: .milestone.title?,
     iteration: (.iteration | if . == null then null else .title // "\(.start_date) to \(.due_date)" end)}
  | issue($now; "gitlab"; $path);

# release($now) shapes a release found on either forge into the document the releases stage draws: version (the tag without
# release/), tag, name, kind (release, pre-release, or draft), latest, at (when it was published, or made, for a
# draft), age in seconds, url (its page), notes (its text, without carriage returns, empty for none), and assets (each
# file it offers, {name, url}, where it downloads). It is given an object with tag, name, kind, latest, at, url, notes,
# and assets.
def release($now):
  {version: (.tag | sub("^release/"; "")), tag, name, kind, latest, at, age: ($now - (.at | epoch)), url,
   notes: ((.notes // "") | gsub("\r"; "")), assets: (.assets // [])};

# github_releases($now) reads the pages of GitHub's releases, each an array, and gives one document per release, newest
# first. A release never published, a draft, is dated by its creation. GitHub's REST list does not mark the latest
# release; GitHub names the newest release that is neither a draft nor a pre-release by its creation, and so does this.
# A release's assets are the files uploaded to it, the sources GitHub makes of every tag apart.
def github_releases($now):
  add // []
  | map({tag: .tag_name, name, at: (.published_at // .created_at), made: .created_at, url: .html_url, notes: .body,
         kind: (if .draft then "draft" elif .prerelease then "pre-release" else "release" end),
         assets: [(.assets // [])[] | {name, url: .browser_download_url}]})
  | (map(select(.kind == "release")) | max_by(.made) | .tag) as $latest
  | sort_by(.at) | reverse[] | . + {latest: (.tag == $latest)} | del(.made) | release($now);

# gitlab_releases($now) reads GitLab's releases and gives the same documents. GitLab marks no pre-release, so a tag
# with a suffix is one, as kit release names it; a release not yet due is a draft; the latest is the newest release.
# A release's assets are its links, each where it downloads directly; the sources GitLab makes of every tag are apart.
def gitlab_releases($now):
  map({tag: .tag_name, name, at: .released_at, url: ._links.self, notes: .description,
      kind: (if .upcoming_release then "draft" elif (.tag_name | test("-")) then "pre-release" else "release" end),
      assets: [(.assets.links // [])[] | {name, url: (.direct_asset_url // .url)}]})
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

# ado_text reads the HTML Azure Boards keeps a work item's description in and gives the text a Markdown reader would
# take from it: a heading as ## and its text, a list item as - and its text, a paragraph, a division, or a line break
# as a new line, every other tag dropped, the common entities read, and the blank lines at either end trimmed.
def ado_text:
  gsub("\r"; "")
  | gsub("<h[1-6][^>]*>"; "\n## "; "i") | gsub("<li[^>]*>"; "\n- "; "i")
  | gsub("<br\\s*/?>|</(p|div|h[1-6]|li)>"; "\n"; "i") | gsub("<[^>]*>"; "")
  | gsub("&nbsp;"; " ") | gsub("&lt;"; "<") | gsub("&gt;"; ">") | gsub("&quot;"; "\"") | gsub("&#39;"; "'")
  | gsub("&amp;"; "&") | gsub("^\\s+|\\s+$"; "");

# ado_assignees reads a work item and gives its assignee's login as a list: Azure Boards has one assignee at most, named
# by the unique name az signs in as.
def ado_assignees: [.fields["System.AssignedTo"] | objects | .uniqueName // empty];

# ado_tags reads a work item's tags, Azure Boards' labels, one text split by "; ", and gives their names.
def ado_tags: (.fields["System.Tags"] // "") | split(";") | map(ltrimstr(" ") | rtrimstr(" ")) | map(select(. != ""));

# ado_iteration reads a work item and gives the title of its iteration: the last name of its iteration path, as az names
# the team's iterations, or null where the path is the project's root alone, which is no iteration.
def ado_iteration:
  (.fields["System.IterationPath"] // "") | split("\\") | if length > 1 then .[-1] else null end;

# ado_issues($now; $path) reads the work items of an Azure Boards query, the answer to forge/ado:open-issues, and gives
# one issue document per work item, oldest first, with no milestone, which Azure Boards has none of. $path is the
# repository's, project/repository, where place reads the project as the organization, as forge/ado:repo names its
# owner.
def ado_issues($now; $path):
  sort_by(.fields["System.CreatedDate"], .id)[]
  | {number: .id, title: .fields["System.Title"], assignees: ado_assignees, labels: ado_tags,
     createdAt: .fields["System.CreatedDate"], iteration: ado_iteration}
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

# ado_reviewers reads a pull request and gives its reviewers by their votes: 10 and 5 approved, -5 and -10 changes, 0
# requested. One who declined is none, and a team or group, which votes as its members do, is named by its display
# name. A vote has no time Azure DevOps keeps, so at is null.
def ado_reviewers:
  [.reviewers[]? | select((.hasDeclined // false) | not)
   | { login: (if .isContainer // false then .displayName else .uniqueName end),
       state: ((.vote // 0) | if . > 0 then "approved" elif . < 0 then "changes" else "requested" end), at: null }];

# ado_pr($now; $policies; $threads; $closes; $path) reads a pull request, its policy evaluations, its threads, and the
# work items linked to it, and gives its document. Azure DevOps tells no push's time without a further call, so pushed
# is null. A pull request from a fork names the fork's repository as its owner.
def ado_pr($now; $policies; $threads; $closes; $path):
  { number: .pullRequestId, ref: "!\(.pullRequestId)", source: "ado",
    branch: (.sourceRefName | ltrimstr("refs/heads/")), title, author: (.createdBy.uniqueName // ""),
    createdAt: .creationDate, body: .description, draft: (.isDraft // false),
    conflicted: (.mergeStatus == "conflicts"), fork: (.forkSource.repository.name // null), pushed: null,
    approval: ado_blocked($policies), labels: [.labels[]? | .name], reviewers: ado_reviewers,
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

# ado_builds($now; $default; $tags; $results) reads az pipelines runs list's answer and gives the builds stage's runs.
# Azure Pipelines names the event the reason: a continuous integration build is a push, a manual one
# workflow_dispatch. The run's number is its build number, which a pipeline formats as it likes, so it is null when it
# is no count. az drops a run's _links, so its page is $results, the project's build results, with the run's id.
def ado_builds($now; $default; $tags; $results):
  map({ id, number: (.buildNumber // "" | if test("^[0-9]+$") then tonumber else null end),
        workflow: (.definition.name // "pipeline"),
        branch: ((.sourceBranch // "") | sub("^refs/(heads|tags)/"; "")),
        tag: ((.sourceBranch // "") | startswith("refs/tags/")), sha: .sourceVersion,
        event: ({individualCI: "push", batchedCI: "push", schedule: "schedule", manual: "workflow_dispatch",
                 pullRequest: "pull_request"}[.reason] // .reason),
        title: (.triggerInfo["ci.message"] // ""), state: ado_build_state, createdAt: .queueTime,
        started: .startTime, ended: .finishTime, url: (._links.web.href // "\($results)?buildId=\(.id)") })
  | builds($default; $tags) | map(build($now));

# merged_last reads merged changes, each {number, title, author, branch, merged, url, body}, merged an ISO 8601 time or
# null, and gives each with merged in seconds since the epoch and named, as named reads its body, in place of the body,
# merged last first, one with no time last.
def merged_last:
  map(.merged |= (if . == null then null else epoch end) | .named = (.body | named) | del(.body))
  | sort_by(-(.merged // 0)) | .[];

# merged_window($more) reads merged changes a forge listed by their update, the latest first, each as merged_last reads
# one with updated, an ISO 8601 time, beside, and gives them as merged_last does. When $more, the forge holds more than
# it gave, and a change merged before the oldest update among them may be missing, as merging updates a change but a
# change merged earlier and not updated since is past them: only those merged since that update are kept, then
# {"limited": N} counts them.
def merged_window($more):
  if $more then
    ([.[].updated | strings | epoch] | min) as $since
    | map(select(.merged != null and $since != null and (.merged | epoch) >= $since))
  else . end
  | (map(del(.updated)) | merged_last), (if $more then {limited: length} else empty end);

# github_merged reads the answer to forge/github:merged-prs's GraphQL query, the merged pull requests updated last, and
# gives one document per merged pull request: {number, title, author, branch, merged, by, url}, merged last first,
# author and by null for a deleted account. When GitHub holds more than it read, they are those merged_window keeps.
def github_merged:
  .data.repository.pullRequests as $prs
  | [$prs.nodes[] | select(. != null)
     | { number, title, author: .author.login, branch: .headRefName, merged: .mergedAt, by: .mergedBy.login, url,
         body, updated: .updatedAt }]
  | merged_window(($prs.totalCount // 0) > length);

# gitlab_merged_row reads a GitLab merge request and gives it as merged_last reads one, updated beside: who merged
# it is merge_user, else the deprecated merged_by.
def gitlab_merged_row:
  { number: .iid, title, author: .author.username, branch: .source_branch, merged: .merged_at,
    by: ((.merge_user // .merged_by).username), url: .web_url, body: .description, updated: .updated_at };

# gitlab_merged($limit) reads GitLab's merged merge requests, newest merged first, and gives the same documents, the
# first $limit, then {"limited": $limit} when it read more.
def gitlab_merged($limit):
  (.[:$limit] | map(gitlab_merged_row | del(.updated)) | merged_last),
  (if length > $limit then {limited: $limit} else empty end);

# gitlab_merged_by_update($limit) reads GitLab's merged merge requests updated last, as a GitLab older than 17.2, which
# orders none by its merge, lists them, and gives the first $limit as merged_window does, more than $limit read
# meaning GitLab holds more.
def gitlab_merged_by_update($limit):
  (length > $limit) as $more | .[:$limit] | map(gitlab_merged_row) | merged_window($more);

# ado_merged($limit; $url) reads Azure DevOps' completed pull requests, as it lists them, and gives the same documents,
# the first $limit, then {"limited": $limit} when it read more: merged is when it closed, by who closed it, and its page
# is under $url, the repository's.
def ado_merged($limit; $url):
  (.[:$limit]
   | map({ number: .pullRequestId, title, author: .createdBy.uniqueName,
           branch: ((.sourceRefName // "") | ltrimstr("refs/heads/")), merged: .closedDate,
           by: .closedBy.uniqueName, url: "\($url)/pullrequest/\(.pullRequestId)", body: .description })
   | merged_last),
  (if length > $limit then {limited: $limit} else empty end);

# day reads a day, as GitLab and Azure DevOps give one, 2026-12-01, or an ISO 8601 time, and gives its seconds since
# the epoch, at the start of the day for a day; null for none.
def day:
  if . == null then null
  elif test("^[0-9]{4}-[0-9]{2}-[0-9]{2}$") then (. + "T00:00:00Z" | fromdateiso8601)
  else epoch end;

# github_milestones reads the pages of GitHub's milestones, each an array, and gives one document per milestone, by
# number: {number, title, state, due, open, closed, url}, due seconds since the epoch, or null for none.
def github_milestones:
  add // [] | sort_by(.number)[]
  | {number, title, state, due: (.due_on | day), open: .open_issues, closed: .closed_issues, url: .html_url};

# gitlab_milestones($host) reads the nodes of GitLab's GraphQL milestones and gives the same documents, by number, its
# iid. Its dueDate is a day, its state active or closed, active being open, and its issues are counted by the stats:
# those open are the total less those closed, a milestone with no stats, or counts, having none. $host, as
# https://gitlab.com, begins the page, which is its webPath.
def gitlab_milestones($host):
  sort_by(.iid | tonumber)[]
  | ((.stats.totalIssuesCount // 0) as $total | (.stats.closedIssuesCount // 0) as $closed
     | {number: (.iid | tonumber), title, state: (if .state == "closed" then "closed" else "open" end),
        due: (.dueDate | day), open: ($total - $closed), closed: $closed, url: ($host + .webPath)});

# iteration($start; $end; $url) is an iteration's document: {id, title, start, end, url}, its start and end seconds
# since the epoch, null for none. It is given an object with id and title.
def iteration($start; $end; $url): {id, title, start: $start, end: $end, url: $url};

# github_iterations reads the answer of GitHub's GraphQL to the projects linked to the repository and gives one document
# per iteration of the iteration fields of each, by start, as {id, title, start, end, url}, url the project's page.
# An iteration lasts its duration in days from its start, and ends the second before the next day begins. A project with
# no iteration field gives none.
def github_iterations:
  [(.data.repository.projectsV2.nodes // [])[] | .url as $url
   | (.fields.nodes // [])[] | (.configuration // empty) | (.iterations // []) + (.completedIterations // [])
   | .[] | (.startDate | day) as $start
   | iteration($start; $start + .duration * 86400 - 1; $url)]
  | unique_by(.id) | sort_by(.start)[];

# gitlab_iterations reads GitLab's iterations, whose due_date is the last day, and gives one document each, by start. An
# iteration of a cadence may have no title; it is then named by its dates.
def gitlab_iterations:
  sort_by(.start_date)[]
  | .title = (.title // "\(.start_date) to \(.due_date)")
  | iteration((.start_date | day); (.due_date | day) + 86399; .web_url);

# ado_iterations($url) reads az's team iterations and gives one document each, by start, an iteration with no dates, as
# the project's root has, first. Azure DevOps' finish date is the last day, as GitLab's due date is.
def ado_iterations($url):
  sort_by(.attributes.startDate // "")[]
  | {id, title: .name, start: (.attributes.startDate | day),
     end: (if .attributes.finishDate then (.attributes.finishDate | day) + 86399 else null end), url: $url};

# since($now) reads an ISO 8601 time, or null, and gives the seconds from it to $now, null for none.
def since($now): if . == null then null else $now - epoch end;

# artifact($now) is an artifact's document: {id, name, digest, size, run, age, expires, url}, digest the forge's own,
# as sha256:…, or null where it keeps none; size in bytes; run the id of the run that made it; age seconds since it
# was made, and expires seconds since the epoch, each null where the forge says nothing. It is given an object with
# id, name, digest, size, run, made and expires, each an ISO 8601 time or null, and url.
def artifact($now):
  {id, name, digest: (.digest // null), size: (.size // null), run: (.run // null), age: (.made | since($now)),
   expires: (.expires | if . == null then null else epoch end), url};

# github_artifacts($now; $url) reads GitHub's page of the repository's artifacts, the newest first, and gives one
# document per artifact that has not expired, as one that has can no longer be downloaded, its page under $url, the
# repository's, beside its run's; then {"limited": N} when GitHub holds more than the page, N those drawn. An artifact
# of a run GitHub no longer names is its download.
def github_artifacts($now; $url):
  (.artifacts // []) as $all
  | [$all[] | select(.expired != true)
     | { id, name, digest, size: .size_in_bytes, run: .workflow_run.id, made: .created_at, expires: .expires_at,
         url: (if .workflow_run.id then "\($url)/actions/runs/\(.workflow_run.id)/artifacts/\(.id)"
               else .archive_download_url end) }
     | artifact($now)] as $rows
  | $rows[], (if (.total_count // 0) > ($all | length) then {limited: ($rows | length)} else empty end);

# gitlab_artifacts($now; $limit) reads GitLab's newest successful jobs and gives one document per job that kept an
# archive, named for the job and the file, "package: artifacts.zip", as most archives share that file's name, the job's
# page its url: GitLab keeps no digest of it. A job that kept none is none. When the jobs read are $limit, GitLab may
# hold more, and {"limited": N} follows the N drawn.
def gitlab_artifacts($now; $limit):
  [.[] | select(.artifacts_file != null)
   | { id, name: (if .name then "\(.name): \(.artifacts_file.filename)" else .artifacts_file.filename end),
       digest: null, size: .artifacts_file.size, run: .pipeline.id,
       made: (.finished_at // .created_at), expires: .artifacts_expire_at, url: .web_url } | artifact($now)] as $rows
  | $rows[], (if length >= $limit then {limited: ($rows | length)} else empty end);

# ado_artifacts($now; $run; $results) reads az pipelines runs artifact list's answer for the run $run, as az pipelines
# runs list gives it, and gives one document per artifact: its id "<run>/<id>", as Azure DevOps numbers an artifact
# within its run; no digest and no expiry, which it keeps none of; its size only for a pipeline artifact; and its
# page the run's artifacts, under $results, the project's build results. It is as old as its run's end, or its
# queueing while it runs.
def ado_artifacts($now; $run; $results):
  .[] | { id: "\($run.id)/\(.id)", name, digest: null,
          size: (.resource.properties.artifactsize // null | if . == null then null else tonumber? // null end),
          run: $run.id, made: ($run.finishTime // $run.queueTime), expires: null,
          url: "\($results)?buildId=\($run.id)&view=artifacts&type=publishedArtifacts" }
  | artifact($now);

# environment is an environment's document, {name, protection, approvers, branches, url}, given one: protection the
# kinds of rule that guard it, sorted, each once, reviewers for an approval, wait for a timer, branches for a branch
# control, deployers for a list of who may deploy, else the forge's own name; approvers who may approve, by login, or
# by the name the forge gives a team or a role; branches the refs that may deploy there, a branch by its name, a tag as
# refs/tags/…, and "protected branches" for those a rule protects, or null for any. protection and approvers are null
# where the forge does not say.
def environment:
  {name, protection: (.protection | if . == null then null else unique end), approvers, branches, url};

# github_environments($branches; $limit) reads the pages of GitHub's environments, each an object, and gives one
# document each, in GitHub's order, $branches the names of each one's branch policies, by its name. An environment with
# its own branch policies costs a read, so the list stops before the first past $limit of them, and {"limited": N}
# follows the N drawn.
def github_environments($branches; $limit):
  [.[].environments[]?] as $all
  | ([$all | to_entries[] | select(.value.deployment_branch_policy.custom_branch_policies == true) | .key]
     | .[$limit] // null) as $stop
  | (($all | if $stop == null then . else .[:$stop] end)[]
  | { name,
      protection: [.protection_rules[]?.type
                   | {required_reviewers: "reviewers", wait_timer: "wait", branch_policy: "branches"}[.] // .],
      approvers: [.protection_rules[]? | select(.type == "required_reviewers") | .reviewers[]?
                  | .reviewer.login // .reviewer.slug // empty],
      branches: (.deployment_branch_policy as $p
                 | if $p == null then null
                   elif $p.protected_branches == true then ["protected branches"]
                   else $branches[.name] // [] end),
      url: .html_url }
  | environment),
  (if $stop == null then empty else {limited: $stop} end);

# github_branch_policies reads an environment's GitHub branch policies and gives their names, a tag's as refs/tags/….
def github_branch_policies:
  [.branch_policies[]? | if .type == "tag" then "refs/tags/\(.name)" else .name end];

# gitlab_environments($protected; $url) reads GitLab's environments and gives one document each, in GitLab's order, its
# page under $url, the project's. $protected is the project's protected environments, null where GitLab shows none
# here: then protection and approvers are null. GitLab's approvers are its approval rules', named as GitLab describes
# them, a user by name, or else, for a count of approvals alone, those who may deploy. GitLab restricts no branch.
def gitlab_environments($protected; $url):
  .[] | .name as $name
  | ($protected | if . == null then null else map(select(.name == $name)) | first end) as $rule
  | { name,
      protection: (if $protected == null then null elif $rule == null then []
                   else [ (if ($rule.approval_rules // []) != [] or ($rule.required_approval_count // 0) > 0
                           then "reviewers" else empty end),
                          (if ($rule.deploy_access_levels // []) != [] then "deployers" else empty end) ] end),
      approvers: (if $protected == null then null elif $rule == null then []
                  elif ($rule.approval_rules // []) != []
                  then [$rule.approval_rules[].access_level_description // empty]
                  elif ($rule.required_approval_count // 0) > 0
                  then [($rule.deploy_access_levels // [])[].access_level_description // empty]
                  else [] end),
      branches: null, url: "\($url)/-/environments/\(.id)" }
  | environment;

# ado_environments($checks; $url) reads Azure DevOps' environments, {count, value}, and its checks on them, and gives
# one document each, in its order, its page under $url, the project's: an Approval check's approvers by their unique
# names, and a branch control's allowed branches, a branch by its name. A check that is turned off guards nothing.
def ado_environments($checks; $url):
  .value[] | (.id | tostring) as $id
  | [$checks.value[]? | select((.resource.id | tostring) == $id and ((.isDisabled // false) | not))] as $mine
  | ([$mine[] | select(.settings.definitionRef.name? == "evaluatebranchProtection")
      | .settings.inputs.allowedBranches // "" | split(",")[] | gsub("^\\s+|\\s+$"; "") | select(. != "")
      | sub("^refs/heads/"; "")]) as $branches
  | { name,
      protection: [$mine[] | if .type.name == "Approval" then "reviewers"
                             elif .settings.definitionRef.name? == "evaluatebranchProtection" then "branches"
                             else (.settings.definitionRef.name? // .type.name // "check"
                                   | ascii_downcase | gsub(" "; "-")) end],
      approvers: [$mine[] | select(.type.name == "Approval") | .settings.approvers[]?
                  | .uniqueName // .displayName // empty],
      branches: (if $branches == [] then null else $branches end), url: "\($url)/_environments/\(.id)" }
  | environment;

# deployment_state reads a forge's state of a deployment and gives kit's: success, failure, running, waiting (for an
# approval), cancelled, or inactive (superseded), else the forge's own word, in lower case.
def deployment_state:
  ascii_downcase
  | { success: "success", active: "success", succeeded: "success", succeededwithissues: "success",
      failure: "failure", failed: "failure", error: "failure",
      in_progress: "running", running: "running", queued: "running", pending: "running", created: "running",
      waiting: "waiting", blocked: "waiting", inactive: "inactive", destroyed: "inactive",
      canceled: "cancelled", cancelled: "cancelled", skipped: "cancelled", abandoned: "cancelled" }[.] // .;

# deployment($now) is a deployment's document, {id, environment, ref, sha, state, actor, age, url, commit}: state as
# deployment_state gives it, actor who made it, age seconds since it was made, url its log or page, and commit the
# commit's page, null where the forge does not say. It is given an object with id, environment, ref, sha, state, actor,
# made, an ISO 8601 time, url, and commit.
def deployment($now):
  {id, environment, ref, sha, state: (.state | if . == null then null else deployment_state end), actor,
   age: (.made | since($now)), url, commit};

# github_deployments($now; $limit) reads the answer to forge/github:deployments's GraphQL query, the newest first, and
# gives the first $limit, each by its latest status, its log its url, then {"limited": N} when GitHub holds more.
def github_deployments($now; $limit):
  .data.repository as $repo
  | ([$repo.deployments.nodes[]? | select(. != null)] | .[:$limit]) as $nodes
  | ($nodes[]
     | { id: .databaseId, environment, ref: .ref.name, sha: .commitOid, state: (.latestStatus.state // .state),
         actor: .creator.login, made: .createdAt,
         url: (.latestStatus.logUrl // .latestStatus.environmentUrl
               // "\($repo.url)/deployments/\(.environment // "" | @uri)"),
         commit: (.commit.url // "\($repo.url)/commit/\(.commitOid)") }
     | deployment($now)),
    (if ($repo.deployments.totalCount // 0) > ($nodes | length) then {limited: ($nodes | length)} else empty end);

# gitlab_deployments($now; $limit; $url) reads GitLab's deployments, the newest first, and gives the first $limit, then
# {"limited": $limit} when it read more: each one's url its job's page, else its environment's, under $url, the
# project's, as is its commit's.
def gitlab_deployments($now; $limit; $url):
  (.[:$limit][]
   | { id, environment: .environment.name, ref, sha, state: .status, actor: .user.username, made: .created_at,
       url: (.deployable.web_url // "\($url)/-/environments/\(.environment.id)"),
       commit: (if .sha then "\($url)/-/commit/\(.sha)" else null end) }
   | deployment($now)),
  (if length > $limit then {limited: $limit} else empty end);

# ado_deployments($now; $environments; $runs; $limit; $cut; $results; $repo) reads Azure DevOps' deployment records of
# the environments read, joined into one array, and gives the newest $limit, by when each started: its environment
# named from $environments, its {count, value}, and its branch, commit, and who asked for it from its run among $runs,
# az pipelines runs list's, null for a run past them. A run of another repository's names no commit page here. Its url
# is its run's, under $results. When $cut, environments were left unread, and {"limited": N} follows the N drawn, as
# it does when there are more than $limit.
def ado_deployments($now; $environments; $runs; $limit; $cut; $results; $repo):
  ($environments.value | map({key: (.id | tostring), value: .name}) | from_entries) as $names
  | ($runs | map({key: (.id | tostring), value: .}) | from_entries) as $by
  | (sort_by(.startTime // .queueTime | if . == null then 0 else epoch end) | reverse) as $all
  | ($all[:$limit] | map(
      ($by[.owner.id | tostring] // {}) as $run
      | { id, environment: $names[.environmentId | tostring],
          ref: ($run.sourceBranch // null | if . == null then null else sub("^refs/heads/"; "") end),
          sha: ($run.sourceVersion // null),
          state: (.result // (if .finishTime == null then "running" else null end)),
          actor: ($run.requestedFor.uniqueName // null), made: (.startTime // .queueTime),
          url: "\($results)?buildId=\(.owner.id)",
          commit: (if $run.sourceVersion != null and ($run.repository.name // $repo.name) == $repo.name
                   then "\($repo.url)/commit/\($run.sourceVersion)" else null end) }
      | deployment($now))) as $rows
  | $rows[], (if $cut or ($all | length) > $limit then {limited: ($rows | length)} else empty end);

# github_role reads a GitHub team's permission, pull, triage, push, maintain, or admin, and gives the role a
# collaborator's role_name names: read, triage, write, maintain, or admin; a custom role is its own name.
def github_role: if . == null then null else {pull: "read", push: "write"}[.] // . end;

# github_collaborators reads the pages of GitHub's collaborators, each an array, and gives one document per person or
# bot with access, in GitHub's order: {login, name, role, kind}, role the highest GitHub gives them, from any source,
# name null where GitHub shows none, and kind bot for a Bot account, else user.
def github_collaborators:
  .[][]? | select(. != null)
  | {login, name: (.name // null), role: (.role_name // null), kind: (if .type == "Bot" then "bot" else "user" end)};

# cut_teams($limit) reads team documents, each with cut, true where the team's members went past those read, and gives
# them without it, the first $limit of them, then {"limited": N}: N $limit when there were more teams, else N those
# drawn when a team's members were cut.
def cut_teams($limit):
  .[:$limit] as $rows
  | ($rows[] | del(.cut)),
    (if length > $limit then {limited: $limit} elif any($rows[]; .cut) then {limited: ($rows | length)}
     else empty end);

# github_teams($members; $limit) reads GitHub's page of the teams with access to the repository, and gives one document
# per team: {team, name, role, members, url}, team its slug, members their logins from $members, the organization's
# answer, t<i> the i-th team's, null for a team the caller cannot see, url its page; then {"limited": N}, as cut_teams
# says, a team cut past the members its answer holds.
def github_teams($members; $limit):
  to_entries
  | map(($members["t\(.key)"]) as $team
        | { team: .value.slug, name: .value.name, role: (.value.permission | github_role),
            members: (if $team == null then null else [$team.members.nodes[]? | .login? // empty] end),
            url: (.value.html_url // null),
            cut: ($team != null and ($team.members.totalCount // 0) > ($team.members.nodes // [] | length)) })
  | cut_teams($limit);

# github_bot reads an actor, as GitHub's GraphQL gives one, and gives its login if it is a bot, as the REST API and a
# pull request's page name it, dependabot[bot], else nothing: a Bot, whose GraphQL login is dependabot, or a login that
# ends [bot].
def github_bot:
  select(. != null and (.__typename == "Bot" or (.login // "" | endswith("[bot]"))))
  | if .login | endswith("[bot]") then .login else .login + "[bot]" end;

# github_bots reads the answer to forge/github:bots's GraphQL query and gives one document per bot that acted among
# those commits, changes, and reviews: {login, acted, last, url}, acted what it did, in the order commit, change,
# review, last when it last did, in seconds since the epoch, url its page as GitHub gave it, null for one seen only in
# commits, whose authors carry none; the last to act first. A commit is a bot's when its author's account is one, or
# else when its author names itself as one, dependabot[bot], as an agent's commit names a person's account; a commit,
# a change, or a review of no account is no one's. Then {"limited": N}, N those drawn, when GitHub holds more open
# changes than those read, or a change more reviewers.
def github_bots:
  .data.repository as $repo
  | [ ( ($repo.defaultBranchRef.target.history.nodes // [])[] | select(. != null) | .committedDate as $at
        | (.author // {}) as $author
        | (($author.user | github_bot) // ($author.name // "" | select(endswith("[bot]"))))
        | {login: ., act: "commit", at: $at, url: null} ),
      ( (($repo.open.nodes // []) + ($repo.merged.nodes // []))[] | select(. != null)
        | (.createdAt as $at | .author | {login: github_bot, act: "change", at: $at, url: .url}),
          ((.latestReviews.nodes // [])[] | select(. != null) | .submittedAt as $at | .author
           | {login: github_bot, act: "review", at: $at, url: .url}) ) ]
  | (($repo.open.totalCount // 0) > ($repo.open.nodes // [] | length)
     or any(($repo.open.nodes // []) + ($repo.merged.nodes // []) | .[] | select(. != null) | .latestReviews;
            (.totalCount? // 0) > (.nodes? // [] | length))) as $cut
  | group_by(.login)
  | map(map(.act) as $acts
        | { login: .[0].login, acted: [("commit", "change", "review") | select(IN($acts[]))],
            last: ([.[].at | select(. != null) | epoch] | max),
            url: (first(.[].url | strings) // null) })
  | sort_by(-(.last // 0), .login)
  | .[], (if $cut then {limited: length} else empty end);

# gitlab_role reads a GitLab access level and gives the role GitLab names it, in lower case: guest, planner, reporter,
# developer, maintainer, owner, and the rest; a level it does not name is its number, and none null.
def gitlab_role:
  if . == null then null
  else {"5": "minimal access", "10": "guest", "15": "planner", "20": "reporter", "25": "security manager",
        "30": "developer", "40": "maintainer", "50": "owner", "60": "admin"}[tostring] // tostring end;

# gitlab_members reads the project's members, as forge/gitlab:members joins their pages, and gives each user once, at
# the highest access any of their memberships gives them, by username; a membership of no user, as an invitation not
# yet taken, is none.
def gitlab_members:
  [.[] | select(.user.username? != null)] | group_by(.user.username) | map(max_by(.accessLevel.integerValue? // 0));

# gitlab_collaborators reads the project's members and gives one document per person or bot with access: {login, name,
# role, kind}, role as gitlab_role names it, and kind bot for a bot user, as a project's access token makes, else user.
def gitlab_collaborators:
  gitlab_members[]
  | { login: .user.username, name: (.user.name // null), role: (.accessLevel.integerValue? | gitlab_role),
      kind: (if .user.bot == true then "bot" else "user" end) };

# gitlab_bots reads the project's members and gives one document per bot user among them: {login, acted, last, url},
# acted null, as GitLab says only when the bot last acted, last that day, in seconds since the epoch, or null where
# GitLab does not show it, url its page; the last to act first.
def gitlab_bots:
  [gitlab_members[] | select(.user.bot == true)
   | {login: .user.username, acted: null, last: (.user.lastActivityOn | day), url: (.user.webUrl // null)}]
  | sort_by(-(.last // 0), .login)[];

# gitlab_teams($members; $host; $limit) reads the project, as GitLab's REST API gives it, and gives one document per
# group it is shared with, in GitLab's order: {team, name, role, members, url}, team the group's full path, role the
# access the share gives, members the usernames from $members, the GraphQL answer, g<i> the i-th group's, null for a
# group the caller cannot see, and url its page; then {"limited": N}, as cut_teams says.
def gitlab_teams($members; $host; $limit):
  (.shared_with_groups // []) | to_entries
  | map(($members["g\(.key)"]) as $group
        | { team: .value.group_full_path, name: (.value.group_name // null),
            role: (.value.group_access_level | gitlab_role),
            members: (if $group == null then null else [$group.groupMembers.nodes[]? | .user.username? // empty] end),
            url: ($group.webUrl // "\($host)/groups/\(.value.group_full_path)"),
            cut: ($group != null and $group.groupMembers.pageInfo.hasNextPage == true) })
  | cut_teams($limit);

# ado_teams($members; $url; $limit) reads az devops team list's answer and gives one document per team, in its order:
# {team, name, role, members, url}, team and name its name, role null, as a team holds no role of its own, members
# the unique names of the first 100 of its members, $members[<team id>], and url the project's teams, under $url, the
# project's; then {"limited": N}, as cut_teams says, a team cut past 100 members.
def ado_teams($members; $url; $limit):
  map(($members[.id] // []) as $team
      | { team: .name, name: .name, role: null,
          members: [$team[:100][] | .identity.uniqueName // .identity.displayName // empty],
          url: "\($url)/_settings/teams", cut: (($team | length) > 100) })
  | cut_teams($limit);

# ado_collaborators($members; $limit) reads az devops team list's answer and gives one document per member of the first
# $limit teams, the first 100 members of each, $members[<team id>], once each, in the order first met: {login, name,
# role, kind}, login the unique name, role admin for an administrator of any of the teams, else member, and kind
# group for a group a team holds, else user; then {"limited": N}, N those drawn, when there were more teams, or a team
# had more members.
def ado_collaborators($members; $limit):
  .[:$limit] as $teams
  | reduce ($teams[] | ($members[.id] // [])[:100][] | select(.identity != null)) as $m
      ({order: [], by: {}};
       ($m.identity.uniqueName // $m.identity.id) as $k
       | if .by | has($k) then .by[$k].admin = (.by[$k].admin or ($m.isTeamAdmin == true))
         else .order += [$k] | .by[$k] = {identity: $m.identity, admin: ($m.isTeamAdmin == true)} end) as $met
  | [$met.order[] | $met.by[.]
     | { login: (.identity.uniqueName // .identity.id), name: (.identity.displayName // null),
         role: (if .admin then "admin" else "member" end),
         kind: (if .identity.isContainer == true then "group" else "user" end) }] as $rows
  | $rows[],
    (if length > $limit or any($teams[]; ($members[.id] // []) | length > 100) then {limited: ($rows | length)}
     else empty end);

# github_attestations($subject; $digest; $made; $now; $url) reads the pages of GitHub's attestations of one digest,
# each an object, and gives one document per attestation of a kind: {subject, digest, kind, signer, age, url}. kind is
# the in-toto statement's predicateType, read from the bundle's envelope, and null where GitHub keeps the bundle apart,
# compressed, behind bundle_url, which kit does not read; signer the workflow its SLSA provenance names,
# <owner>/<repo>/<path>@<ref>, else github for one GitHub made itself, as an immutable release's, else null; age from
# the transparency log's entry, else from $made, when the subject was made; url the repository's attestations page.
# Two of one kind are one.
def github_attestations($subject; $digest; $made; $now; $url):
  [ .[].attestations[]? | select(. != null) | . as $a
    | (try ($a.bundle.dsseEnvelope.payload | @base64d | fromjson) catch null) as $statement
    | ($statement.predicate.buildDefinition.externalParameters.workflow? // null) as $workflow
    | ($a.bundle.verificationMaterial.tlogEntries[0].integratedTime? // null) as $logged
    | { subject: $subject, digest: $digest, kind: ($statement.predicateType? // null),
        signer: (if ($workflow | type) == "object" and $workflow.path != null
                 then "\($workflow.repository // "" | sub("^https?://[^/]+/"; ""))/\($workflow.path)@\($workflow.ref)"
                 elif $a.initiator == "github" then "github" else null end),
        age: (if $logged != null then $now - ($logged | tonumber) else ($made | since($now)) end),
        url: $url } ]
  | unique_by(.kind)[];

# severity reads a forge's severity, in any case, and gives it in lower case, or null for none.
def severity: if . == null then null else ascii_downcase end;

# dismissal(r; c) gives a dismissal's reason and its comment, apart by a colon, those that are there, or null.
def dismissal(r; c): [r, c] | map(select(. != null and . != "")) | if length == 0 then null else join(": ") end;

# latest_by_number reads alert documents, as they were read, and gives each number once, the last read of it, as an
# alert updated or dismissed while its pages were read is on two of them; the newest number first.
def latest_by_number: group_by(.number) | map(last) | sort_by(-.number);

# github_code_scanning($now) reads the pages of GitHub's code scanning alerts, each an array, and gives one document
# per alert: {number, rule, severity, state, path, age, url, by, reason}, rule the rule's id, severity its security
# severity where it has one, else the rule's own, path where it was last found, age seconds since it was made, by
# who dismissed it, and reason the dismissal's reason and comment, each null for an open alert.
def github_code_scanning($now):
  [ .[][]? | select(. != null)
    | { number, rule: (.rule.id? // .rule.name? // null),
        severity: ((.rule.security_severity_level? // .rule.severity?) | severity), state,
        path: (.most_recent_instance.location.path? // null), age: (.created_at | since($now)), url: .html_url,
        by: (.dismissed_by.login? // null), reason: dismissal(.dismissed_reason; .dismissed_comment) } ]
  | latest_by_number[];

# github_dependabot($now) reads the pages of GitHub's Dependabot alerts and gives one document per alert, as
# github_code_scanning does: rule the advisory's GHSA id, severity the vulnerability's, path the manifest's.
def github_dependabot($now):
  [ .[][]? | select(. != null)
    | { number, rule: (.security_advisory.ghsa_id? // null),
        severity: ((.security_vulnerability.severity? // .security_advisory.severity?) | severity), state,
        path: (.dependency.manifest_path? // null), age: (.created_at | since($now)), url: .html_url,
        by: (.dismissed_by.login? // null), reason: dismissal(.dismissed_reason; .dismissed_comment) } ]
  | latest_by_number[];

# github_secret_scanning($now) reads the pages of GitHub's secret scanning alerts and gives one document per alert, as
# github_code_scanning does: rule the secret's type, severity null, as GitHub gives none, a resolved alert dismissed,
# by who resolved it, the resolution and its comment the reason, and path where the secret was first found.
def github_secret_scanning($now):
  [ .[][]? | select(. != null)
    | { number, rule: (.secret_type // null), severity: null,
        state: (if .state == "resolved" then "dismissed" else .state end),
        path: (.first_location_detected.path? // null), age: (.created_at | since($now)), url: .html_url,
        by: (.resolved_by.login? // null),
        reason: (if .state == "resolved" then dismissal(.resolution; .resolution_comment) else null end) } ]
  | latest_by_number[];

# fnmatch($s) reads a ruleset's pattern and says whether it names the ref $s, as fnmatch does: * any characters but /,
# ** any characters, ? any one but /; every other character is itself.
def fnmatch($s):
  (gsub("(?<c>[.+^$(){}|\\[\\]\\\\])"; "\\\(.c)") | gsub("\\*\\*"; "\u0001") | gsub("\\*"; "[^/]*")
   | gsub("\\?"; "[^/]") | gsub("\u0001"; ".*")) as $re
  | $s | test("^" + $re + "$");

# github_guards($branch) reads a ruleset, as GitHub's GraphQL gives one, and says whether it guards the branch: an
# include that names it, ~ALL or ~DEFAULT_BRANCH or a pattern, and no exclude that does; a ruleset of no ref names
# guards every branch.
def github_guards($branch):
  def names: . == "~ALL" or . == "~DEFAULT_BRANCH" or fnmatch("refs/heads/\($branch)");
  (.conditions.refName? // null) as $ref
  | $ref == null or (any(($ref.include // [])[]; names) and (any(($ref.exclude // [])[]; names) | not));

# github_whole reads the answer to forge/github:rulesets's GraphQL query and says whether it holds every ruleset, and
# every rule and bypass actor of each.
def github_whole:
  .data.repository.rulesets as $sets
  | ($sets.totalCount // 0) <= ($sets.nodes // [] | length)
    and all(($sets.nodes // [])[] | select(. != null);
            (.rules.totalCount // 0) <= (.rules.nodes // [] | length)
            and (.bypassActors.totalCount // 0) <= (.bypassActors.nodes // [] | length));

# github_protection_rules reads GitHub's protection of a branch and gives the kinds of rule it holds, sorted, named as
# a ruleset names its own: deletion, non_fast_forward, pull_request, required_status_checks, and the rest.
def github_protection_rules:
  [ (select(.allow_deletions.enabled? == false) | "deletion"),
    (select(.allow_force_pushes.enabled? == false) | "non_fast_forward"),
    (select(.required_pull_request_reviews != null) | "pull_request"),
    (select(.required_status_checks != null) | "required_status_checks"),
    (select(.required_linear_history.enabled? == true) | "required_linear_history"),
    (select(.required_signatures.enabled? == true) | "required_signatures"),
    (select(.required_conversation_resolution.enabled? == true) | "required_conversation_resolution"),
    (select(.block_creations.enabled? == true) | "creation"),
    (select(.lock_branch.enabled? == true) | "lock_branch"),
    (select(.restrictions != null) | "restrictions"),
    (select(.enforce_admins.enabled? == true) | "enforce_admins") ]
  | sort;

# github_protection_checks reads GitHub's protection of a branch and gives the status checks a merge needs, by name,
# sorted: its contexts, and the context of each of its checks; none where it requires none.
def github_protection_checks:
  if .required_status_checks == null then []
  else [(.required_status_checks.contexts // [])[], (.required_status_checks.checks // [])[]?.context]
       | map(strings) | unique end;

# github_protection_reviews reads GitHub's protection of a branch and gives the approving reviews a merge needs: 0
# where it requires no pull request, and null where it does and says not how many.
def github_protection_reviews:
  if .required_pull_request_reviews == null then 0
  else .required_pull_request_reviews.required_approving_review_count // null end;

# github_rule_checks reads a ruleset, as GitHub's GraphQL gives one, and gives the status checks its rules require, by
# context, sorted; none where it has no such rule, and null where a rule's parameters are not shown.
def github_rule_checks:
  [.rules.nodes[]? | select(.type == "REQUIRED_STATUS_CHECKS")] as $rules
  | if any($rules[]; .parameters == null) then null
    else [$rules[].parameters.requiredStatusChecks[]?.context | strings] | unique end;

# github_rule_reviews reads a ruleset, as GitHub's GraphQL gives one, and gives the approving reviews its pull request
# rules require, the most of them; 0 where it has none, and null where a rule's parameters are not shown.
def github_rule_reviews:
  [.rules.nodes[]? | select(.type == "PULL_REQUEST")] as $rules
  | if any($rules[]; .parameters.requiredApprovingReviewCount == null) then null
    else [$rules[].parameters.requiredApprovingReviewCount] | max // 0 end;

# github_rulesets($protection) reads the answer to forge/github:rulesets's GraphQL query and gives one document per
# control on the default branch: {id, control, target, enforcement, rules, checks, reviews, source, url}, each ruleset
# that guards it, in GitHub's order, its include patterns its target, its rules' types, sorted, and the checks and
# reviews they require; then the branch's own protection, $protection, {rules, checks, reviews}, each null where
# GitHub shows them only to an admin, or nothing where it has none.
def github_rulesets($protection):
  .data.repository as $repo
  | ($repo.defaultBranchRef.name? // null) as $branch
  | select($branch != null)
  | ( ($repo.rulesets.nodes // [])[] | select(. != null and github_guards($branch))
      | { id: .databaseId, control: .name, target: (.conditions.refName.include? // ["~ALL"]),
          enforcement: (.enforcement // "" | ascii_downcase),
          rules: ([.rules.nodes[]? | .type | ascii_downcase] | sort),
          checks: github_rule_checks, reviews: github_rule_reviews,
          source: "ruleset", url: "\($repo.url)/rules/\(.databaseId)" } ),
    ( $protection | select(. != null)
      | { id: "protection", control: $branch, target: [$branch], enforcement: "active", rules, checks, reviews,
          source: "protection", url: "\($repo.url)/settings/branches" } );

# github_bypass_actor reads a ruleset's bypass actor and names it: a team by its slug, a user by the login, an app by
# its slug, else what it stands for, organization admins, a role, deploy keys, or enterprise owners; null where GitHub
# hides it from who may not edit the ruleset.
def github_bypass_actor:
  if . == null then null
  elif .actor != null then (.actor.slug // .actor.login // null)
  elif .organizationAdmin == true then "organization admins"
  elif .repositoryRoleName != null then "\(.repositoryRoleName) role"
  elif .deployKey == true then "deploy keys"
  elif .enterpriseOwner == true then "enterprise owners"
  else null end;

# github_bypasses($protection) reads the answer to forge/github:rulesets's GraphQL query and gives one waiver per
# bypass actor of each ruleset that guards the default branch: {waives, actor, kind, reason, by, until, hidden, url},
# waives the ruleset's id, kind bypass, reason its bypass mode, always, pull_request, or exempt; those GitHub hides
# from who may not edit the ruleset, and those kit cannot name, as a team asked for without its slug, are one waiver,
# actor and reason null, hidden how many; hidden is null on every other. Then one per login, team, or app the branch's
# own protection, $protection, lets bypass its pull request rule, waives protection.
def github_bypasses($protection):
  .data.repository as $repo
  | ($repo.defaultBranchRef.name? // null) as $branch
  | select($branch != null)
  | ( ($repo.rulesets.nodes // [])[] | select(. != null and github_guards($branch)) | . as $set
      | "\($repo.url)/rules/\($set.databaseId)" as $url
      | [ .bypassActors.nodes[]?
          | {actor: github_bypass_actor,
             mode: (if . == null or .bypassMode == null then null else .bypassMode | ascii_downcase end)} ] as $actors
      | ( $actors[] | select(.actor != null)
          | {waives: $set.databaseId, actor, kind: "bypass", reason: .mode, by: null, until: null, hidden: null,
             url: $url} ),
        ( [$actors[] | select(.actor == null)] | length | select(. > 0)
          | {waives: $set.databaseId, actor: null, kind: "bypass", reason: null, by: null, until: null, hidden: .,
             url: $url} ) ),
    ( $protection | select(. != null) | .required_pull_request_reviews.bypass_pull_request_allowances? // {}
      | (.users[]?.login, .teams[]?.slug, .apps[]?.slug) | select(. != null)
      | { waives: "protection", actor: ., kind: "bypass", reason: "pull_request", by: null, until: null, hidden: null,
          url: "\($repo.url)/settings/branches" } );

# gitlab_vulnerabilities($now) reads the nodes of GitLab's vulnerability report and gives one document per
# vulnerability: {number, rule, severity, state, path, age, url, by, reason}, number its id, rule its primary
# identifier's name, or its title, state open for one detected or confirmed and dismissed for one dismissed, path the
# file it was found in, age seconds since it was detected, by who dismissed it, and reason the dismissal's reason, in
# lower case, and its comment, each null for an open one.
def gitlab_vulnerabilities($now):
  [ .[] | select(. != null) | (.state == "DISMISSED") as $dismissed
    | { number: (.id | tostring | sub("^.*/"; "") | tonumber),
        rule: (.primaryIdentifier.name? // .title // null), severity: (.severity | severity),
        state: (if $dismissed then "dismissed" else "open" end), path: (.location.file? // null),
        age: (.detectedAt | since($now)), url: (.webUrl // null),
        by: (if $dismissed then .dismissedBy.username? // null else null end),
        reason: (if $dismissed then dismissal(.dismissalReason | severity; .stateComment) else null end) } ]
  | latest_by_number[];

# gitlab_controls($branch; $push; $url) reads GitLab's protected branches and gives one document per control on the
# branch $branch: each protected branch whose name, or wildcard, * any characters, names it, {id, control, target,
# enforcement, rules, checks, reviews, source, url}, its rules who may push and who may merge, non_fast_forward where
# force pushes are not allowed, and code_owner_review where code owners approve, its checks and reviews null, as a
# merge's checks and approvals are the project's settings, which kit does not read; then the project's push rules,
# $push, or nothing for none, its rules those set, no check and no review, source ruleset; each page the project's
# repository settings under $url.
def gitlab_controls($branch; $push; $url):
  def levels: [.[]?.access_level_description // empty] | join(", ");
  def names: "^" + (gsub("(?<c>[.+^$(){}|?\\[\\]\\\\])"; "\\\(.c)") | gsub("\\*"; ".*")) + "$";
  ( .[] | select(.name as $name | $branch | test($name | names))
    | { id, control: .name, target: [.name], enforcement: "active",
        rules: ([ (if (.push_access_levels // []) == [] then empty else "push: \(.push_access_levels | levels)" end),
                  (if (.merge_access_levels // []) == [] then empty else "merge: \(.merge_access_levels | levels)" end),
                  (select(.allow_force_push != true) | "non_fast_forward"),
                  (select(.code_owner_approval_required == true) | "code_owner_review") ] | sort),
        checks: null, reviews: null, source: "protection", url: "\($url)/-/settings/repository" } ),
  ( $push | objects
    | { id: "push_rule", control: "push rules", target: ["*"], enforcement: "active",
        rules: ([to_entries[] | select(.key | IN("id", "project_id", "created_at") | not)
                 | select(.value != null and .value != false and .value != "" and .value != 0) | .key] | sort),
        checks: [], reviews: 0, source: "ruleset", url: "\($url)/-/settings/repository" } );

# ado_controls($url) reads az repos policy list's answer for the default branch and gives one document per policy:
# {id, control, target, enforcement, rules, checks, reviews, source, url}, control the policy's type, target each
# scope's ref, a prefix's with * after it, enforcement active for a blocking policy, evaluate for an optional one, and
# disabled for one turned off, rules its settings that are set, key=value, checks the build a build policy requires,
# by its display name, else build <definition>, or the status a status policy requires, <genre>/<name>, reviews the
# approvers a reviewers policy requires, null where it says not how many, and url the branch's policies, $url.
def ado_controls($url):
  .[] | select(. != null and .isDeleted != true)
  | (.type.displayName? // null) as $type | (.settings // {}) as $set
  | { id, control: $type,
      target: [(.settings.scope // [])[]? | if .refName == null then "*"
               elif .matchKind == "Prefix" then .refName + "*" else .refName end],
      enforcement: (if .isEnabled != true then "disabled" elif .isBlocking == true then "active" else "evaluate" end),
      rules: ([(.settings // {}) | to_entries[] | select(.key != "scope")
               | select((.value | type) as $t | $t == "number" or $t == "string" or $t == "boolean")
               | select(.value != false and .value != "" and .value != 0) | "\(.key)=\(.value)"] | sort),
      checks: (if $type == "Build"
               then [$set.displayName // "" | if . == "" then "build \($set.buildDefinitionId)" else . end]
               elif $type == "Status"
               then [[$set.statusGenre, $set.statusName] | map(strings | select(. != "")) | join("/")]
               else [] end),
      reviews: (if $type == "Minimum number of reviewers" or $type == "Required reviewers"
                then $set.minimumApproverCount // null else 0 end),
      source: "protection", url: $url };
