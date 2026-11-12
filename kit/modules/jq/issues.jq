# issues.jq - the rows the issues stage lists: every open issue, in the backlog or underway (jq module)
# ----------------------------------------------------------------------------------------------------------------------
# import "issues" as issues; with jq -L "${kit_jq}".
#
# see: modules/bash/issues.sh, modules/jq/backlog.jq, modules/jq/underway.jq

import "backlog" as backlog;
import "underway" as underway;

# names($issue) reads a change, open or merged, and says whether it names $issue, one of a tracker the forge does not
# link by itself: its branch starts with the issue's key and a dash, as kit work names the branch, ab4521-fix; or a
# closing keyword in its body names the issue's reference, in either case, as forge.jq's named reads them.
def names($issue):
  ((.branch // "") | startswith("\($issue.key)-"))
  or ((.named // []) | map(ascii_upcase) | index($issue.ref | ascii_upcase) != null);

# started($started; $worktrees; $branches) reads an issue's row with no change and gives it its status: underway when
# one of $started, the branches named for an issue, starts with its key and a dash, as 9-other, its branch that one,
# and here, local, and worktree as the branch has them; else backlog, its branch, here, local, and worktree null.
def started($started; $worktrees; $branches):
  (.key // "\(.number)") as $key
  | (first($started[] | select(startswith("\($key)-"))) // null) as $branch
  | if $branch == null
    then . + {status: "backlog", branch: null, pr: null, here: null, local: null, worktree: null}
    else ($worktrees[$branch] != null) as $here
    | . + {status: "underway", branch: $branch, pr: null, here: $here,
           local: (($here | not) and ($branch | IN($branches[]))), worktree: ($worktrees[$branch] // null)} end;

# rows($changes; $heads; $worktrees; $branches; $merged) reads the open issues, as an array of forge.jq's issue
# documents, and gives every one a row. An issue of the forge's own, one without a key, is underway in a row for each
# change in $changes that closes it, as underway.jq gives it, with the change's branch as its branch, and the issue's
# milestone, iteration, and swimlane as the open issue of its number has them. One of another tracker, which holds its
# key, as ab4521, is underway in a row for each change that names it, as names says: the forge closes it on no merge,
# so kit does, and one a change in $merged names is done, and has no row, whether or not its tracker closed it. An issue
# no change closes or names is underway when a branch is named for it, its key and a dash, as 9-other or ab4521-fix:
# one of the desk's, $branches, a checked-out one first, else one of $heads, the primary's; its row has that branch, no
# pr, and here, local, and worktree as the branch has them. One no change closes and no branch names is in the backlog,
# its branch, pr, here, local, and worktree null. Every row's keys start number, ref, tracker, title, assignees,
# labels, age, source, organization, repository, milestone, iteration, swimlane, status, branch, pr, here, local, and
# worktree, in that order, any other the issue holds after them, but its key. ref is the issue's reference, #7 where
# the issue names none. The oldest issue comes first; one issue that two changes close gives two rows, by the changes'
# numbers.
def rows($changes; $heads; $worktrees; $branches; $merged):
  map(select(.key == null)) as $forge
  | (map(select(.key != null) | . as $issue | select(any($merged[]; names($issue)) | not))) as $others
  | ($forge | map({key: (.number | tostring), value: .}) | from_entries) as $open
  | ([$branches[] | select($worktrees[.] != null)] + [$branches[] | select($worktrees[.] == null)] + $heads)
    as $started
  | ($forge | backlog::unclaimed($changes; []; []; false) | map(started($started; $worktrees; $branches)))
    + ($changes | underway::rows($worktrees; $branches)
       | map(($open[.number | tostring] // {}) as $issue
             | . + {ref: ($issue.ref // null), tracker: ($issue.tracker // null),
                    milestone: ($issue.milestone // null), iteration: ($issue.iteration // null),
                    swimlane: ($issue.swimlane // null), status: "underway", branch: .pr.branch}))
    + [$others[] | . as $issue | [$changes[] | select(names($issue))] as $by
       | if $by == [] then started($started; $worktrees; $branches)
         else $by[] as $pr | $issue | underway::row($pr; $worktrees; $branches)
              | . + {status: "underway", branch: $pr.branch} end]
  | map({number, ref, tracker, title, assignees, labels, age, source, organization, repository, milestone, iteration,
         swimlane, status, branch, pr, here, local, worktree} + . | .ref //= "#\(.number)" | del(.key))
  | sort_by(-(.age // 0), .number, .ref, (.pr.number // 0));

# rows($changes; $heads; $worktrees; $branches) gives the rows of the open issues as rows does, with no change merged.
def rows($changes; $heads; $worktrees; $branches): rows($changes; $heads; $worktrees; $branches; []);

# by_owners($logins; $except) reads rows and keeps those whose issue one of $logins is assigned; with $except, the
# others, an issue no one is assigned among them. With no login, it keeps every row.
def by_owners($logins; $except):
  if ($logins | length) == 0 then .
  else map(select(((.assignees // []) | any(. as $x | $logins | index($x) != null)) != $except)) end;
