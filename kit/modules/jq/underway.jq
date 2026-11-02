# underway.jq - the rows the underway stage lists: each issue an open change closes, with that change (jq module)
#
# import "underway" as underway; with jq -L "${kit_jq}".
#
# see: modules/bash/underway.sh

# rows($worktrees) reads the open changes, as an array of forge.jq's change documents, and gives one row per issue
# and change: the issue's number, title, assignees, and age; the change as pr, {number, title, author, state,
# branch}; here, true when $worktrees names a checkout for the change's branch; and worktree, that checkout's path,
# or null. $worktrees maps a branch to the path of the checkout that has it. The oldest issue comes first; one issue
# that two changes close gives two rows, by the changes' numbers.
def rows($worktrees):
  [ .[] as $pr
    | $pr.closes[]?
    | { number, title, assignees: (.assignees // []), age,
        pr: { number: $pr.number, title: $pr.title, author: $pr.author, state: $pr.state, branch: $pr.branch },
        here: ($worktrees[$pr.branch] != null), worktree: ($worktrees[$pr.branch] // null) } ]
  | sort_by(-(.age // 0), .number, .pr.number);

# by_owners($logins; $except) reads rows and keeps those whose change one of $logins opened, or whose issue one of
# them is assigned; with $except, those neither. With no login, it keeps every row.
def by_owners($logins; $except):
  if ($logins | length) == 0 then .
  else map(select((((.pr.author as $a | $logins | index($a)) != null)
                   or (.assignees | any(. as $x | $logins | index($x) != null))) != $except)) end;
