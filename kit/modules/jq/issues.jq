# issues.jq - the rows the issues stage lists: every open issue, in the backlog or underway (jq module)
# ----------------------------------------------------------------------------------------------------------------------
# import "issues" as issues; with jq -L "${kit_jq}".
#
# see: modules/bash/issues.sh, modules/jq/backlog.jq, modules/jq/underway.jq

import "backlog" as backlog;
import "underway" as underway;

# rows($changes; $heads; $worktrees; $branches) reads the open issues, as an array of forge.jq's issue documents, and
# gives every one a row. An issue no change in $changes closes is underway when a branch is named for it, its number
# and a dash, as 9-other: one of the desk's, $branches, a checked-out one first, else one of $heads, the primary's;
# its row has that branch, no pr, and here, local, and worktree as the branch has them. One no change closes and no
# branch names is in the backlog, its branch, pr, here, local, and worktree null. A row with status underway comes for
# each issue and change that closes it, as underway.jq gives it, with the change's branch as its branch, and the
# issue's milestone, iteration, and swimlane as the open issue of its number has them. Every row's keys start number,
# title, assignees, labels, age, source, organization, repository, milestone, iteration, swimlane, status, branch, pr,
# here, local, and worktree, in that order, any other the issue holds after them. The oldest issue comes first; one
# issue that two changes close gives two rows, by the changes' numbers.
def rows($changes; $heads; $worktrees; $branches):
  (map({key: (.number | tostring), value: .}) | from_entries) as $open
  | ([$branches[] | select($worktrees[.] != null)] + [$branches[] | select($worktrees[.] == null)] + $heads)
    as $started
  | (backlog::unclaimed($changes; $started; []; false)
     | map(if .branch == null then . + {status: "backlog", pr: null, here: null, local: null, worktree: null}
           else ($worktrees[.branch] != null) as $here
           | . + {status: "underway", pr: null, here: $here, local: (($here | not) and (.branch | IN($branches[]))),
                  worktree: ($worktrees[.branch] // null)} end))
    + ($changes | underway::rows($worktrees; $branches)
       | map(($open[.number | tostring] // {}) as $issue
             | . + {milestone: ($issue.milestone // null), iteration: ($issue.iteration // null),
                    swimlane: ($issue.swimlane // null), status: "underway", branch: .pr.branch}))
  | map({number, title, assignees, labels, age, source, organization, repository, milestone, iteration, swimlane,
         status, branch, pr, here, local, worktree} + .)
  | sort_by(-(.age // 0), .number, (.pr.number // 0));

# by_owners($logins; $except) reads rows and keeps those whose issue one of $logins is assigned; with $except, the
# others, an issue no one is assigned among them. With no login, it keeps every row.
def by_owners($logins; $except):
  if ($logins | length) == 0 then .
  else map(select(((.assignees // []) | any(. as $x | $logins | index($x) != null)) != $except)) end;
