# backlog.jq - the backlog's issues: those not closed by any open change, with the branch that claims each (jq module)
#
# import "backlog" as backlog; with jq -L "${kit_jq}".
#
# see: modules/bash/backlog.sh

# unclaimed($changes; $heads; $logins; $except) reads the open issues, as an array of forge.jq's issue documents, and
# gives those no change in $changes closes, each with branch: the first of $heads, the primary's branches, that
# starts with its number and a dash, or null. With $logins, it keeps the issues assigned to one of them, or with
# $except, those assigned to none of them.
def unclaimed($changes; $heads; $logins; $except):
  ($changes | map(.closes[]?.number)) as $closed
  | map(select(.number as $n | $closed | index($n) | not)
        | . + {branch: (. as $issue | $heads | map(select(startswith("\($issue.number)-"))) | first)})
  | if ($logins | length) == 0 then .
    else map(select((.assignees | any(. as $a | $logins | index($a))) != $except)) end;
