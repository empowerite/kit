# shellcheck shell=bash
# leases.sh - the paths each issue's work claims, in refs every worktree of the clone sees (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
#
# A lease is a JSON blob at refs/kit/leases/<key>, the key the issue's reference starts a branch name with, as
# tracker:key prints it: 73 for #73, ab4521 for AB#4521. Every worktree of a clone shares refs/kit/, so every session
# in the clone reads the same leases. A lease says which paths the issue's work will change, who took it, on which
# branch, and until when; modules/jq/leases.jq reads one, and judges which leases a claim overlaps:
#
#   {"lease": 1, "issue": "#73", "paths": ["docs/**"], "holder": "ada", "host": "desk", "branch": "73-leases",
#    "worktree": "/src/kit/worktrees/73-leases", "taken": 1762950000, "renewed": 1762950000, "expires": 1762953600}
#
# Every change to a lease moves refs/kit/leases-epoch too, in the one git update-ref --stdin transaction that moves the
# lease from the value it read: a session that changed any lease meanwhile moved the epoch, so the transaction fails,
# and the change is read and judged again, 50 times at most. Two claims at once are judged one after the other.
#
# A lease is not a lock, not an identity, and not proof that its holder is alive. It expires, an hour after it is
# taken or renewed by default. kit commit refuses a staged path another issue's lease on the branch holds, and kit
# policy refuses a path a branch changed outside the branch's own lease.
#
# see: modules/jq/leases.jq, bin/kit-leases, docs/superpowers/specs/2026-11-12-leases-design.md

use tracker forge file

# The refs the leases live in, the epoch every change moves, how many times a change is judged again, and the term a
# lease is taken for without --for, in seconds.
declare -gr leases_refs=refs/kit/leases leases_epoch=refs/kit/leases-epoch leases_tries=50 leases_term=3600

# leases:now prints the time, in seconds since the epoch.
leases:now() { date +%s; }

# leases:read prints every lease in the clone as one JSON object, {epoch, leases}: epoch the object name the epoch ref
# holds, or null before the first lease; leases each as leases.jq's read gives it, with its oid, by key. A ref that
# names no blob does not read as a lease. The epoch is read first: a lease that changes after it moves it, so a swap
# from what was read fails, and a change judged on leases older than the epoch never lands.
leases:read() {
  local oid ref epoch
  local -a words=() rows=()
  epoch=$(git rev-parse --quiet --verify "${leases_epoch}") || epoch=''
  list:read rows git for-each-ref --format='%(objectname) %(refname)' "${leases_refs}/" || return 1
  for ref in "${rows[@]}"
  do
    oid=${ref%% *} ref=${ref#* }
    words+=("${ref#"${leases_refs}/"}" "${oid}" "$(git cat-file blob "${oid}" 2>/dev/null)")
  done
  jq --null-input --compact-output -L "${kit_jq}" --arg epoch "${epoch}" 'import "leases" as leases;
    {epoch: (if $epoch == "" then null else $epoch end),
     leases: [$ARGS.positional as $w | range(0; $w | length; 3) | $w[.:. + 3]
       | leases::read(.[0]; .[2]) + {oid: .[1]}] | sort_by(.key)}' \
    --args "${words[@]}"
}

# leases:key <word> prints the key of a lease: a key a lease here already has, as kit leases lists one that does not
# read, or else the key of the issue the word refers to. It returns 2 for a word that is neither.
leases:key() {
  if [[ "$1" =~ ^[a-z0-9]+$ ]] && git show-ref --verify --quiet "${leases_refs}/$1"
  then
    printf '%s\n' "$1"
    return 0
  fi
  tracker:key "$1"
}

# leases:issue <word> prints the issue's reference in its tracker's form, #7 for 7, or the word as it is when no
# tracker reads it, as for a key kit leases lists.
leases:issue() { tracker:normal "$1" 2>/dev/null || printf '%s\n' "$1"; }

# leases:term <duration> prints a duration in seconds: a number and a unit, s, m, h, or d, as 90s, 45m, 2h, or 1d. It
# returns 2, saying so, for anything else.
leases:term() {
  local -A seconds=([s]=1 [m]=60 [h]=3600 [d]=86400)
  if [[ ! "$1" =~ ^([1-9][0-9]{0,6})([smhd])$ ]]
  then
    error "'$1' is not a duration; --for takes a number and a unit, s, m, h, or d: --for=45m"
    return 2
  fi
  printf '%s\n' "$(( BASH_REMATCH[1] * seconds[${BASH_REMATCH[2]}] ))"
}

# leases:paths <path>… prints the paths as a claim keeps them, as one JSON array: ** for none, ./ and a trailing /
# taken off. It returns 2, saying why, for a path that is empty, starts with /, or holds a .. segment.
leases:paths() {
  local refusal
  refusal=$(jq --null-input --raw-output -L "${kit_jq}" 'import "leases" as leases;
    [$ARGS.positional[] | leases::refused] | first // empty' --args "$@") || return 1
  if [[ -n "${refusal}" ]]
  then
    error "${refusal}; kit help leases says how a path is written"
    return 2
  fi
  jq --null-input --compact-output -L "${kit_jq}" 'import "leases" as leases;
    [$ARGS.positional[] | leases::tidy] | if length == 0 then ["**"] else unique end' --args "$@"
}

# leases:swap <epoch> <change> <line>… moves the leases as the git update-ref --stdin lines say, and the epoch from
# <epoch>, empty before the first, to a blob that names the change, in one transaction. It returns 1, quietly, when
# the transaction fails: a lease or the epoch is no longer what was read.
leases:swap() {
  local epoch=$1 change=$2 blob
  shift 2
  blob=$(jq --null-input --compact-output --arg after "${epoch}" --arg change "${change}" \
    --argjson at "$(leases:now)" --arg nonce "${BASHPID}.${RANDOM}" \
    '{after: (if $after == "" then null else $after end), change: $change, at: $at, nonce: $nonce}' \
    | git hash-object -w --stdin) || return 1
  if [[ -n "${epoch}" ]]
  then
    set -- "$@" "update ${leases_epoch} ${blob} ${epoch}"
  else
    set -- "$@" "create ${leases_epoch} ${blob}"
  fi
  printf '%s\n' "$@" | git update-ref --stdin >/dev/null 2>&1
}

# leases:retry sleeps a tenth of a second at most, so sessions that lost the same race do not race again in step.
leases:retry() { sleep "0.0$(( RANDOM % 9 + 1 ))"; }

# leases:here prints where a lease taken here is: the branch checked out, empty on a detached HEAD, the worktree's
# root, and the host's short name, separated by \x1f, which keeps an empty field where a tab would not.
leases:here() {
  local branch root host
  branch=$(git symbolic-ref --quiet --short HEAD) || branch=''
  root=$(git rev-parse --show-toplevel) || return 1
  host=$(hostname -s 2>/dev/null || hostname)
  printf '%s\x1f%s\x1f%s\n' "${branch}" "${root}" "${host%%.*}"
}

# leases:holder prints who takes a lease when --holder names no one: git's user.name, else the login.
leases:holder() { git config user.name 2>/dev/null || id -un; }

# leases:body-paths <body> prints the paths an issue's body lists under a Paths heading, one a line, as leases.jq's
# body_paths reads them, and nothing for a body that lists none.
leases:body-paths() {
  jq --null-input --raw-output -L "${kit_jq}" --arg body "$1" 'import "leases" as leases; $body | leases::body_paths[]'
}

# leases:refused <state> <claim> prints, one a line, why each lease of the state refuses the claim, as leases.jq's
# conflicts judges it, and nothing when none does.
leases:refused() {
  jq --raw-output -L "${kit_jq}" --argjson claim "$2" --argjson now "$(leases:now)" 'import "leases" as leases;
    .leases | leases::conflicts($claim; $now) | leases::refusal($now)' <<<"$1"
}

# leases:take <reference> <term> <holder> <path>… claims the paths for the issue, ** for none, for <term> seconds, an
# hour when it is empty, <holder> naming who takes it, git's user.name or the login when it is empty. A live lease of
# the issue keeps its paths, and the new ones join them; an expired one is replaced. The claim is refused, saying who
# holds what, and nothing is written, when it may overlap a live lease it competes with. It returns 1 on a refusal,
# and 2 for a reference no tracker reads or a path no claim takes.
leases:take() {
  local ref=$1 term=${2:-${leases_term}} holder=${3:-$(leases:holder)} key issue paths state branch root host claim
  local refusals doc blob old now try line
  shift 3
  key=$(tracker:key "${ref}") || return $?
  issue=$(tracker:normal "${ref}") || return $?
  paths=$(leases:paths "$@") || return $?
  IFS=$'\x1f' read -r branch root host < <(leases:here) || return 1
  for (( try = 0; try < leases_tries; try++ ))
  do
    state=$(leases:read) || return 1
    now=$(leases:now)
    claim=$(jq --compact-output --arg key "${key}" --argjson paths "${paths}" --arg branch "${branch}" \
      --argjson now "${now}" -L "${kit_jq}" 'import "leases" as leases;
      ([.leases[] | select(.key == $key and .readable and leases::live($now)) | .paths[]]) as $kept
      | {key: $key, branch: (if $branch == "" then null else $branch end), paths: ($kept + $paths | unique)}' \
      <<<"${state}") \
      || return 1
    refusals=$(leases:refused "${state}" "${claim}") || return 1
    if [[ -n "${refusals}" ]]
    then
      while IFS= read -r line
      do
        error "refusing to lease $(jq --raw-output '.paths | join(" ")' <<<"${claim}") to ${issue}: ${line}"
      done <<<"${refusals}"
      step "ask who holds it, or claim other paths: kit leases take ${issue} <path>…"
      return 1
    fi
    doc=$(jq --null-input --compact-output --arg issue "${issue}" --argjson claim "${claim}" --arg holder "${holder}" \
      --arg host "${host}" --arg root "${root}" --argjson now "${now}" --argjson term "${term}" \
      '{lease: 1, issue: $issue, paths: $claim.paths, holder: $holder, host: $host, branch: $claim.branch,
        worktree: $root, taken: $now, renewed: $now, expires: ($now + $term)}') || return 1
    blob=$(git hash-object -w --stdin <<<"${doc}") || return 1
    old=$(jq --raw-output --arg key "${key}" '.leases[] | select(.key == $key) | .oid' <<<"${state}")
    if [[ -n "${old}" ]]
    then
      line="update ${leases_refs}/${key} ${blob} ${old}"
    else
      line="create ${leases_refs}/${key} ${blob}"
    fi
    if leases:swap "$(jq --raw-output '.epoch // empty' <<<"${state}")" "take ${issue}" "${line}"
    then
      ok "${issue} holds $(jq --raw-output --argjson now "${now}" -L "${kit_jq}" 'import "leases" as leases;
        "\(.paths | join(" ")), \(leases::when($now))"' <<<"${doc}")"
      return 0
    fi
    leases:retry
  done
  error "the leases changed ${leases_tries} times while ${issue}'s claim was judged; run it again"
  return 1
}

# leases:renew <term> <reference>… moves each issue's lease to expire <term> seconds from now, its own term when <term>
# is empty. A lease that expired is judged again first, as a new claim is. It returns 1 when an issue holds no lease,
# its lease does not read, or another lease now refuses it.
leases:renew() {
  local term=$1 ref key issue state lease now refusals doc blob try line status=0
  shift
  for ref in "$@"
  do
    key=$(leases:key "${ref}") || return $?
    issue=$(leases:issue "${ref}")
    for (( try = 0; try < leases_tries; try++ ))
    do
      state=$(leases:read) || return 1
      now=$(leases:now)
      lease=$(jq --compact-output --arg key "${key}" '.leases[] | select(.key == $key)' <<<"${state}")
      if [[ -z "${lease}" ]]
      then
        error "${issue} holds no lease to renew; take one: kit leases take ${issue} [<path>…]"
        status=1
        break
      fi
      if [[ "$(jq --raw-output .readable <<<"${lease}")" != true ]]
      then
        error "refs/kit/leases/${key} does not read as a lease, so it cannot be renewed; release it, then take it" \
          "again: kit leases release ${key}"
        status=1
        break
      fi
      refusals=''
      if [[ "$(jq --raw-output --argjson now "${now}" '.expires > $now' <<<"${lease}")" != true ]]
      then
        refusals=$(leases:refused "${state}" "${lease}") || return 1
      fi
      if [[ -n "${refusals}" ]]
      then
        while IFS= read -r line
        do
          error "refusing to renew ${issue}'s lease, which expired: ${line}"
        done <<<"${refusals}"
        status=1
        break
      fi
      doc=$(jq --compact-output --arg key "${key}" --argjson now "${now}" --arg term "${term}" '.leases[]
        | select(.key == $key)
        | {lease: 1, issue, paths, holder, host, branch, worktree, taken, renewed: $now,
           expires: ($now + (if $term == "" then .expires - .renewed else ($term | tonumber) end))}' <<<"${state}") \
        || return 1
      blob=$(git hash-object -w --stdin <<<"${doc}") || return 1
      if leases:swap "$(jq --raw-output '.epoch // empty' <<<"${state}")" "renew ${issue}" \
        "update ${leases_refs}/${key} ${blob} $(jq --raw-output .oid <<<"${lease}")"
      then
        ok "${issue} holds $(jq --raw-output --argjson now "${now}" -L "${kit_jq}" 'import "leases" as leases;
          "\(.paths | join(" ")), \(leases::when($now))"' <<<"${doc}")"
        break
      fi
      leases:retry
    done
    if (( try == leases_tries ))
    then
      error "the leases changed ${leases_tries} times while ${issue}'s lease was renewed; run it again"
      status=1
    fi
  done
  return "${status}"
}

# leases:release <reference>… deletes each issue's lease, from the value it read, and the push remote's copy when it
# has one. An issue that holds none is said to, and is no failure.
leases:release() {
  local ref key issue state oid try status=0
  for ref in "$@"
  do
    key=$(leases:key "${ref}") || return $?
    issue=$(leases:issue "${ref}")
    for (( try = 0; try < leases_tries; try++ ))
    do
      state=$(leases:read) || return 1
      oid=$(jq --raw-output --arg key "${key}" '.leases[] | select(.key == $key) | .oid' <<<"${state}")
      if [[ -z "${oid}" ]]
      then
        info "${issue} holds no lease here"
        break
      fi
      if leases:swap "$(jq --raw-output '.epoch // empty' <<<"${state}")" "release ${issue}" \
        "delete ${leases_refs}/${key} ${oid}"
      then
        ok "${issue}'s lease is released"
        break
      fi
      leases:retry
    done
    if (( try == leases_tries ))
    then
      error "the leases changed ${leases_tries} times while ${issue}'s lease was released; run it again"
      status=1
      continue
    fi
    leases:unmirror "${key}" || status=1
  done
  return "${status}"
}

# leases:list <form> prints every lease, the live first, as text, or as one JSON object with json: {now, leases}, each
# lease as leases.jq's read gives it, with live.
leases:list() {
  local state
  state=$(leases:read) || return 1
  jq --raw-output -L "${kit_jq}" --arg form "$1" --argjson now "$(leases:now)" 'import "leases" as leases;
    [.leases[] | . + {live: leases::live($now)}] | sort_by(.live | not) as $leases
    | if $form == "json" then {now: $now, leases: $leases}
      elif ($leases | length) == 0 then
        "No lease is held in this clone; take one: kit leases take <reference> [<path>…]"
      else $leases[] | leases::line($now) end' <<<"${state}"
}

# leases:remote prints the push remote, the remote a branch goes to (forge:push-remote: the push setting in kit.yml,
# as a fork's origin, else the primary), when it is a remote of this clone, and nothing otherwise. A lease goes where
# its branch goes, so kit work and kit submit push both to one remote.
leases:remote() {
  local remote
  remote=$(forge:push-remote 2>/dev/null) || return 0
  git remote 2>/dev/null | grep --line-regexp --fixed-strings -- "${remote}" || true  # no such remote prints none
}

# leases:mirror <branch> pushes the branch's own lease, the one of the issue its name starts with, to the push
# remote, forced, since a blob has no history to fast-forward, so CI reads the claim that judges the branch. A branch
# with no lease here pushes nothing. A push that fails is warned of.
leases:mirror() {
  local ref key remote
  ref=$(tracker:branch-ref "$1" 2>/dev/null) || return 0
  [[ -n "${ref}" ]] || return 0
  key=$(tracker:key "${ref}" 2>/dev/null) || return 0
  git show-ref --verify --quiet "${leases_refs}/${key}" || return 0
  remote=$(leases:remote)
  [[ -n "${remote}" ]] || return 0
  if git push --quiet --force "${remote}" "${leases_refs}/${key}:${leases_refs}/${key}" 2>/dev/null
  then
    step "${ref}'s lease is on ${remote}, for CI"
  else
    warn "${ref}'s lease could not be pushed to ${remote}, so CI judges the branch by the lease ${remote} had; run:" \
      "kit submit"
  fi
}

# leases:unmirror <key> deletes the push remote's copy of the lease, when it has one. A remote that cannot be read
# or that refuses the deletion is warned of, and returns 1.
leases:unmirror() {
  local remote held
  remote=$(leases:remote)
  [[ -n "${remote}" ]] || return 0
  if ! held=$(git ls-remote --quiet "${remote}" "${leases_refs}/$1" 2>/dev/null)
  then
    warn "${remote} could not be read, so its copy of the lease, if any, is left; run: kit leases release $1"
    return 1
  fi
  [[ -n "${held}" ]] || return 0
  if ! git push --quiet "${remote}" ":${leases_refs}/$1" 2>/dev/null
  then
    warn "${remote} kept its copy of the lease; run: kit leases release $1"
    return 1
  fi
}

# leases:keys <subject> <trailers> prints, one a line, the key of each issue a commit names: the reference in front of
# its subject, and each Refs: trailer's, as git interpret-trailers --parse prints the trailers, a value that is a list
# joined by commas. A reference no tracker reads is left out.
leases:keys() {
  local ref
  local -a refs=()
  [[ ! "$1" =~ ^(([A-Z]+)?#[0-9]+)\  ]] || refs+=("${BASH_REMATCH[1]}")
  while IFS= read -r ref
  do
    ref=${ref#"${ref%%[! ]*}"}
    ref=${ref%"${ref##*[! ]}"}
    [[ -z "${ref}" ]] || refs+=("${ref}")
  done < <(sed -n 's/^[Rr][Ee][Ff][Ss]: *//p' <<<"$2" | tr ',' '\n')
  for ref in "${refs[@]}"
  do
    tracker:key "${ref}" 2>/dev/null || true  # a reference no tracker reads names no lease
  done
}

# leases:commit-held <branch> <message> <path>… refuses a staged path held by a live lease on the branch of an issue the
# commit, its message in the file <message>, does not name, unless it is the branch's own lease, which holds every path
# of the branch. A lease that does not read has no branch, so it is warned of, not refused. It returns 1 on a refusal.
leases:commit-held() {
  local branch=$1 message=$2 keys state held kind path line issue refused=0
  shift 2
  [[ -n "$(git for-each-ref --count=1 --format=x "${leases_refs}/")" ]] || return 0
  keys=$(leases:keys "$(head -n 1 "${message}")" "$(git interpret-trailers --parse < "${message}")") || return 1
  state=$(leases:read) || return 1
  held=$(jq --raw-output -L "${kit_jq}" --arg keys "${keys}" --arg branch "${branch}" --argjson now "$(leases:now)" \
    'import "leases" as leases;
    [$ARGS.positional[]] as $paths | ($keys | split("\n")) as $named
    | .leases[] | select(leases::live($now))
    | if .readable | not then "warn\t\(leases::refusal($now))"
      elif .branch == $branch and (.key | IN($named[]) | not) and (leases::own | not) then
        . as $lease | ($paths - [$paths | leases::outside($lease.paths)])[]
        | "held\t\(.)\t\($lease | leases::refusal($now))\t\($lease.issue)"
      else empty end' --args "$@" <<<"${state}") || return 1
  while IFS=$'\t' read -r kind path line issue
  do
    case "${kind}" in
      (warn) warn "${path}" ;;
      (held)
        error "refusing to commit ${path}: ${line}; name ${issue} in the commit, or ask its holder: kit leases"
        refused=1
    esac
  done <<<"${held}"
  return "${refused}"
}

# leases:commit-renew <branch> <commit> renews the lease of each issue the commit names, by the lease's own term, when
# it is live and on the branch, so a lease lives while its work goes on. It only warns: the commit exists already.
leases:commit-renew() {
  local keys key state
  [[ -n "$(git for-each-ref --count=1 --format=x "${leases_refs}/")" ]] || return 0
  keys=$(leases:keys "$(git log -1 --format=%s "$2")" \
    "$(git log -1 --format=%B "$2" | git interpret-trailers --parse)") || return 0
  state=$(leases:read) || return 0
  while IFS= read -r key
  do
    [[ -n "${key}" ]] || continue
    jq --exit-status --arg key "${key}" --arg branch "$1" --argjson now "$(leases:now)" -L "${kit_jq}" \
      'import "leases" as leases;
      any(.leases[]; .key == $key and .readable and .branch == $branch and leases::live($now))' \
      <<<"${state}" >/dev/null || continue
    leases:renew '' "${key}" >/dev/null 2>&1 \
      || warn "the lease of ${key}, an issue the commit names, could not be renewed; run: kit leases"
  done <<<"${keys}"
}

# leases:check <branch> <base> <runner> holds the paths the branch changed since its merge base with <base> to the
# branch's own lease, the one of the issue its name starts with: each path outside it is refused. The lease is the
# clone's; with <runner> true, on a CI runner whose clone holds none, it is the push remote's, fetched as an object
# alone, into no ref and no FETCH_HEAD. A branch that names no issue, or whose issue holds no lease, is judged by none,
# and says so. It returns 1 on a refusal, a lease that does not read, or one that cannot be fetched.
leases:check() {
  local branch=$1 base=$2 runner=$3 ref key issue oid remote held text lease fork outside count path
  local -a changed=()
  ref=$(tracker:branch-ref "${branch}" 2>/dev/null) || ref=''
  if [[ -z "${ref}" ]]
  then
    info "${branch} names no issue, so no lease holds it to its paths"
    return 0
  fi
  key=$(tracker:key "${ref}") || return 1
  issue=$(leases:issue "${ref}")
  oid=$(git rev-parse --quiet --verify "${leases_refs}/${key}") || oid=''
  if [[ -z "${oid}" && "${runner}" == true ]]
  then
    remote=$(leases:remote)
    if [[ -n "${remote}" ]]
    then
      if ! held=$(git ls-remote --quiet "${remote}" "${leases_refs}/${key}" 2>/dev/null)
      then
        error "run: git ls-remote ${remote} (${issue}'s lease on ${remote} could not be read, so ${branch}'s paths" \
          "are unchecked)"
        return 1
      fi
      if [[ -n "${held}" ]]
      then
        if ! git fetch --quiet --no-write-fetch-head "${remote}" "${leases_refs}/${key}" 2>/dev/null
        then
          error "run: git fetch ${remote} ${leases_refs}/${key} (${issue}'s lease could not be fetched, so" \
            "${branch}'s paths are unchecked)"
          return 1
        fi
        oid=${held%%$'\t'*}
      fi
    fi
  fi
  if [[ -z "${oid}" ]]
  then
    info "${issue} holds no lease, so ${branch}'s paths are held to none"
    return 0
  fi
  text=$(git cat-file blob "${oid}" 2>/dev/null) || text=''
  lease=$(jq --null-input --compact-output -L "${kit_jq}" --arg key "${key}" --arg text "${text}" \
    'import "leases" as leases; leases::read($key; $text)') || return 1
  if [[ "$(jq --raw-output .readable <<<"${lease}")" != true ]]
  then
    error "${leases_refs}/${key} does not read as a lease, so ${branch}'s paths cannot be judged; take it again: kit" \
      "leases release ${key}, then kit leases take ${issue} [<path>…]"
    summary "### ❌ ${issue}'s lease does not read" "" "Take it again, then push it with kit submit."
    return 1
  fi
  if ! fork=$(git merge-base "${base}" HEAD 2>/dev/null)
  then
    error "${branch} and its base share no commit here, so its paths cannot be judged; fetch it with full history"
    return 1
  fi
  list:read0 changed git diff --name-only --no-renames -z "${fork}" HEAD -- || return 1
  outside=$(jq --null-input --raw-output -L "${kit_jq}" --argjson lease "${lease}" 'import "leases" as leases;
    [$ARGS.positional[]] | leases::outside($lease.paths)' --args "${changed[@]}") || return 1
  if [[ -z "${outside}" ]]
  then
    ok "the ${#changed[@]} path(s) ${branch} changed are inside ${issue}'s lease"
    summary "✅ The ${#changed[@]} path(s) the branch changed are inside ${issue}'s lease."
    return 0
  fi
  count=0
  while IFS= read -r path
  do
    error "${path} is outside ${issue}'s lease, $(jq --raw-output '.paths | join(" ")' <<<"${lease}"); claim it: kit" \
      "leases take ${issue} ${path}, then kit submit"
    count=$(( count + 1 ))
  done <<<"${outside}"
  summary "### ❌ ${count} path(s) outside ${issue}'s lease" "" "- ${outside//$'\n'/$'\n'- }"
  return 1
}

# ----------------------------------------------------------------------------------------------------------------------
