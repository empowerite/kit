# underway.jq - the rows the underway stage lists: each issue an open change closes, with that change (jq module)
#
# import "underway" as underway; with jq -L "${kit_jq}".
#
# see: modules/bash/underway.sh

# row($pr; $worktrees; $branches) reads an issue, as forge.jq's issue documents or a change's closes hold one, and gives
# its row beside the change $pr: the issue's number, title, assignees, labels, age, source, organization, and
# repository, and any other field it holds; the change as pr, {number, source, title, author, state, branch, fork,
# pushed}, source the forge it is on; here, true when $worktrees names a checkout for the change's branch; local, true
# when $branches names the branch and no checkout has it; and worktree, that checkout's path, or null. $worktrees maps
# a branch to the path of the checkout that has it, and $branches lists the desk's local branches.
def row($pr; $worktrees; $branches):
  ($worktrees[$pr.branch] != null) as $here
  | . + { assignees: (.assignees // []), labels: (.labels // []), source: (.source // null),
          organization: (.organization // null), repository: (.repository // null),
          pr: { number: $pr.number, source: ($pr.source // null), title: $pr.title, author: $pr.author,
                state: $pr.state, branch: $pr.branch, fork: $pr.fork, pushed: $pr.pushed },
          here: $here, local: (($here | not) and ($pr.branch | IN($branches[]))),
          worktree: ($worktrees[$pr.branch] // null) };

# rows($worktrees; $branches) reads the open changes, as an array of forge.jq's change documents, and gives one row per
# issue and change, as row gives it, for each issue a change closes: its number, title, assignees, labels, age, source,
# organization, and repository. The oldest issue comes first; one issue that two changes close gives two rows, by the
# changes' numbers.
def rows($worktrees; $branches):
  [ .[] as $pr
    | $pr.closes[]?
    | {number, title, assignees, labels, age, source, organization, repository} | row($pr; $worktrees; $branches) ]
  | sort_by(-(.age // 0), .number, .pr.number);

# by_owners($logins; $except) reads rows and keeps those whose issue one of $logins is assigned; with $except, the
# others, an issue no one is assigned among them. Who opened the change does not count: its author has a column of its
# own. With no login, it keeps every row.
def by_owners($logins; $except):
  if ($logins | length) == 0 then .
  else map(select((.assignees | any(. as $x | $logins | index($x) != null)) != $except)) end;
