# underway.jq - the rows the underway stage lists: each issue an open change closes, with that change (jq module)
#
# import "underway" as underway; with jq -L "${kit_jq}".
#
# see: modules/bash/underway.sh

# rows($worktrees; $branches) reads the open changes, as an array of forge.jq's change documents, and gives one row per
# issue and change: the issue's number, title, assignees, age, source, organization, and repository; the change as pr,
# {number, title, author, state, branch, fork, pushed}; here, true when $worktrees names a checkout for the change's
# branch; local, true when $branches names the branch and no checkout has it; and worktree, that checkout's path, or
# null. $worktrees maps a branch to the path of the checkout that has it, and $branches lists the desk's local branches.
# The oldest issue comes first; one issue that two changes close gives two rows, by the changes' numbers.
def rows($worktrees; $branches):
  [ .[] as $pr
    | ($worktrees[$pr.branch] != null) as $here
    | $pr.closes[]?
    | { number, title, assignees: (.assignees // []), age, source: (.source // null),
        organization: (.organization // null), repository: (.repository // null),
        pr: { number: $pr.number, title: $pr.title, author: $pr.author, state: $pr.state, branch: $pr.branch,
              fork: $pr.fork, pushed: $pr.pushed },
        here: $here, local: (($here | not) and ($pr.branch | IN($branches[]))),
        worktree: ($worktrees[$pr.branch] // null) } ]
  | sort_by(-(.age // 0), .number, .pr.number);

# by_owners($logins; $except) reads rows and keeps those whose issue one of $logins is assigned; with $except, the
# others, an issue no one is assigned among them. Who opened the change does not count: its author has a column of its
# own. With no login, it keeps every row.
def by_owners($logins; $except):
  if ($logins | length) == 0 then .
  else map(select((.assignees | any(. as $x | $logins | index($x) != null)) != $except)) end;
