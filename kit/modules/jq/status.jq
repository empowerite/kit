# status.jq - the checkout documents kit status draws, and what of each fits on the screen (jq module)
# ----------------------------------------------------------------------------------------------------------------------
#
# import "status" as status; with jq -L "${kit_jq}".
#
# see: modules/bash/status.sh

# checkout($facts; $porcelain) reads a checkout's commits since its fork, one "<sha>\t<subject>" line each, and
# gives its document: $facts, {path, relative, branch, ahead, behind, added, removed}, with detached, uncommitted,
# changes, and commits. $porcelain is git status --porcelain: a two-character status, a space, and the path.
def checkout($facts; $porcelain):
  [$porcelain | split("\n")[] | select(. != "") | { status: .[0:2], path: .[3:] }] as $changes
  | $facts + { detached: ($facts.branch == ""), uncommitted: ($changes | length), changes: $changes,
               commits: [split("\n")[] | select(. != "") | split("\t") | { sha: .[0], subject: (.[1:] | join("\t")) }] };

# widths reads checkout documents and gives the widths of the widest "↑a", "↓b", "+x", and "-y", space-separated.
def widths:
  [ (map(.ahead) | max), (map(.behind) | max), (map(.added) | max), (map(.removed) | max) ]
  | map(tostring | length + 1) | join(" ");

# changes_shown($cap) reads a checkout document and gives its uncommitted paths as [status, path]. Past $cap and
# one more, it gives the first $cap and then ["…", "and <n> more"], so a count never stands in for a single path.
def changes_shown($cap):
  (.changes // []) as $c | ($c | length) as $n
  | if $n > $cap + 1
    then ($c[:$cap][] | [.status, .path]), ["…", "and \($n - $cap) more"]
    else $c[] | [.status, .path] end;

# commits_shown($cap) reads a checkout document and gives its commits, newest first, as [sha, subject]. Past $cap
# and one more, it gives the newest $cap - 1, then ["…", "and <n> more"], then the oldest, the one at the fork.
def commits_shown($cap):
  (.commits | length) as $n
  | if $n > $cap + 1
    then (.commits[:$cap - 1][] | [.sha, .subject]), ["…", "and \($n - $cap) more"], (.commits[-1] | [.sha, .subject])
    else .commits[] | [.sha, .subject] end;
