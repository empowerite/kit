# shellcheck shell=bash
# issue.sh - work on issues: start them, assign them, gather them under a parent, and close them (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# issue:start [<reference>…] starts or resumes work on one or more issues. Each reference names its tracker, as
# modules/bash/tracker.sh describes:
#
#   - `7` or `#7`: an issue in the forge's own tracker (GitHub or GitLab)
#   - `AB#4521`: a work item in a declared Azure DevOps tracker
#
# The first reference is the primary issue. The others are secondary. Each issue is closed on merge by:
#
#   - a primary issue in the forge's tracker: the forge links it to the branch the forge creates
#   - any other primary issue, and every secondary one: a `Closes <reference>` line in the pull request's body
#
# Every listed issue is assigned to the caller, as each tracker identifies the caller.
#
# issue:start never switches the checkout it runs from. It checks the branch out in worktrees/<key>-<slug> under
# the main checkout, and hydrates it: what the kit delivers under .kit/ is in place before the first command runs
# there. The key is the reference as it starts a branch name: 7 or ab4521. An existing branch, worktree, or pull
# request is reused, so running it again fills in whatever is missing.
#
# Given --title <title> instead of a reference, it creates the issue first; it never asks. Given --title and
# references, it gathers the references under a new parent issue with that title, as issue:gather does, and starts the
# parent with the references as its secondaries, so the one pull request closes them all. Given --pr <change> and
# references, it adds the references to the caller's own open pull request instead, as issue:join does.
#
# Uncommitted work is handled by where it is:
#
#   - in the root checkout (the desk, which belongs to the human): moved into the new worktree, still uncommitted.
#     The desk is left clean, and is fast-forwarded when it is on the default branch. If the work cannot be
#     applied in the worktree, it stays in the stash and the command stops. When the command stops before the
#     worktree has the work, by a refusal, a failed command, or an interrupt, the work goes back to the desk. Work a
#     stash does not take, such as work inside a submodule, stays on the desk, and no other stash is touched.
#   - in any other checkout: left in place, since it belongs to that checkout's issue
#
# The desk never holds the issue's branch while the command works: when the desk has the branch checked out,
# issue:start refuses before it assigns, stashes, or commits anything.
#
# Every forge call goes through modules/bash/forge.sh, and every issue call through modules/bash/tracker.sh. Each helper below
# prints its one result on stdout and its messages on stderr, so issue:start can capture the results in order.

use git forge tracker settings yaml file

# issue_swept is the hash of the stash issue:start swept the desk's work into, from the push until the stash is
# applied in the worktree; empty, nothing swept is waiting. issue:sweep-restore reads it at exit.
declare -g issue_swept=''

# issue:create <title> creates an issue with the title in the forge's tracker, and prints its reference.
issue:create() {
  local title=$1 number
  if [[ -z "${title}" ]]
  then
    error "an issue title is required"
    return 2
  fi
  number=$(tracker:at '#' create "${title}") || return 1
  ok "Created #${number}: ${title}"
  printf '#%s\n' "${number}"
}

# issue:assign <reference>… adds the caller (as each issue's tracker identifies the caller) to each issue's
# assignees. It warns when someone else is already assigned, and when the assignment fails.
issue:assign() {
  local ref prefix me assignees
  local -A me_in=()
  for ref in "$@"
  do
    prefix=$(tracker:parse "${ref}" | cut -f 1)
    [[ -n "${me_in[${prefix}]:-}" ]] || me_in[${prefix}]=$(tracker:at "${prefix}" me) || return 1
    me=${me_in[${prefix}]}
    assignees=$(tracker:of "${ref}" assignees)
    [[ " ${assignees} " == *" ${me} "* ]] && continue
    [[ -z "${assignees}" ]] \
      || warn "${ref} is already assigned to ${assignees}; adding you as well. Coordinate before continuing."
    tracker:of "${ref}" assign "${me}" || warn "${ref} could not be assigned to ${me}; assign it by hand."
  done
}

# issue:assign-to <login> <reference>… adds the login to each issue's assignees, as kit assign --to does: one login
# for every tracker, which is the caller's to get right. It warns when someone else is already assigned.
issue:assign-to() {
  local login=$1 ref assignees failed=0
  shift
  for ref in "$@"
  do
    assignees=$(tracker:of "${ref}" assignees) || { failed=1; continue; }
    [[ " ${assignees} " == *" ${login} "* ]] && { info "${ref} is assigned to ${login} already"; continue; }
    [[ -z "${assignees}" ]] || warn "${ref} is already assigned to ${assignees}; adding ${login} as well."
    if tracker:of "${ref}" assign "${login}"
    then ok "${ref} assigned to ${login}"
    else error "${ref} could not be assigned to ${login}"; failed=1
    fi
  done
  return "${failed}"
}

# issue:unassign [--all] <reference>… takes the caller, as each issue's tracker identifies the caller, off each
# issue; with --all, every assignee.
issue:unassign() {
  local all=false ref prefix assignees login failed=0
  local -A me_in=()
  [[ "${1:-}" != --all ]] || { all=true; shift; }
  for ref in "$@"
  do
    assignees=$(tracker:of "${ref}" assignees) || { failed=1; continue; }
    if [[ "${all}" == false ]]
    then
      prefix=$(tracker:parse "${ref}" | cut -f 1)
      [[ -n "${me_in[${prefix}]:-}" ]] || me_in[${prefix}]=$(tracker:at "${prefix}" me) || return 1
      [[ " ${assignees} " == *" ${me_in[${prefix}]} "* ]] || { info "${ref} is not assigned to you"; continue; }
      assignees=${me_in[${prefix}]}
    fi
    for login in ${assignees}
    do
      if tracker:of "${ref}" unassign "${login}"
      then ok "${ref} no longer assigned to ${login}"
      else error "${ref} could not be unassigned from ${login}"; failed=1
      fi
    done
  done
  return "${failed}"
}

# issue:close (--not-planned | --duplicate-of <reference>) <reference>… closes each open issue. A duplicate of an
# issue in the same tracker is closed as its tracker closes a duplicate; of one in another tracker, it gets a comment
# naming the original and is closed as not planned. An issue already closed is left as it is.
issue:close() {
  local reason original='' ref same failed=0
  case ${1:-} in
    --not-planned) reason=not-planned; shift ;;
    --duplicate-of) reason=duplicate; original=$(tracker:normal "${2:-}") || return 2; shift 2 ;;
    *) error "issue:close needs --not-planned or --duplicate-of <reference>"; return 2 ;;
  esac
  for ref in "$@"
  do
    ref=$(tracker:normal "${ref}") || return 2
    if [[ "${ref}" == "${original}" ]]
    then
      error "${ref} cannot be a duplicate of itself"
      failed=1
      continue
    fi
    [[ "$(tracker:of "${ref}" state)" != CLOSED ]] || { info "${ref} is closed already"; continue; }
    same=''
    [[ -z "${original}" || "$(tracker:parse "${ref}" | cut -f 1)" != "$(tracker:parse "${original}" | cut -f 1)" ]] \
      || same=$(tracker:parse "${original}" | cut -f 2)
    if [[ "${reason}" == duplicate && -n "${same}" ]]
    then
      tracker:of "${ref}" close duplicate "${same}" || { error "${ref} could not be closed"; failed=1; continue; }
    elif [[ "${reason}" == duplicate ]]
    then
      if ! tracker:of "${ref}" comment "Duplicate of ${original}" || ! tracker:of "${ref}" close not-planned
      then
        error "${ref} could not be closed"
        failed=1
        continue
      fi
    else
      tracker:of "${ref}" close not-planned || { error "${ref} could not be closed"; failed=1; continue; }
    fi
    ok "${ref} closed as ${reason/-/ }${original:+ of ${original}}"
  done
  return "${failed}"
}

# issue:gather <title> <reference>… creates a parent issue with the title in the forge's tracker and puts each issue
# under it: an issue in the forge's tracker as the forge nests one (a sub-issue on GitHub, a linked item on GitLab),
# and an issue in another tracker as a task line in the parent's body. It prints the parent's reference, and starts
# no work.
issue:gather() {
  local title=$1 ref body='' number failed=0
  local -a refs=()
  shift
  if [[ -z "${title}" || $# -eq 0 ]]
  then
    error "a title and at least one issue to gather are required"
    return 2
  fi
  for ref in "$@"
  do
    ref=$(tracker:normal "${ref}") || return 2
    refs+=("${ref}")
    tracker:is-forge "${ref}" || body+="- [ ] ${ref}"$'\n'
  done
  number=$(tracker:at '#' create "${title}" "${body}") || return 1
  ok "Created #${number}: ${title}"
  for ref in "${refs[@]}"
  do
    tracker:is-forge "${ref}" || continue
    forge:issue-add-sub "${number}" "${ref#\#}" || { warn "${ref} could not be put under #${number}; do it by hand"; failed=1; }
  done
  (( failed == 0 )) && ok "Gathered ${refs[*]} under #${number}"
  printf '#%s\n' "${number}"
}

# issue:join <change> <reference>… adds the issues to one of the caller's own open pull requests: each is assigned to
# the caller and closed by the pull request's body. A pull request someone else opened is refused, with the way to
# hand the issues to its author instead.
issue:join() {
  local pr=${1#[#!]} state author me sigil noun ref
  local -a refs=()
  shift
  forge:require || return 1
  for ref in "$@"
  do
    ref=$(tracker:normal "${ref}") || return 2
    refs+=("${ref}")
  done
  IFS=$'\t' read -r sigil noun < <(forge:noun)
  state=$(forge:pr-state "${pr}") || return 1
  state=${state%%$'\t'*}
  if [[ "${state}" != OPEN ]]
  then
    error "${noun} ${sigil}${pr} is ${state,,}; add the issues to an open one"
    return 1
  fi
  author=$(forge:pr-author "${pr}") || return 1
  me=$(forge:me) || return 1
  if [[ "${author}" != "${me}" ]]
  then
    error "${noun} ${sigil}${pr} is ${author}'s; hand the issues over instead: kit assign --to ${author} ${refs[*]}"
    return 1
  fi
  issue:assign "${refs[@]}" || return
  issue:link-in-body "${pr}" "${refs[@]}"
  ok "Added ${refs[*]} to ${noun} ${sigil}${pr}"
}

# issue:settle <change> settles the issues a closed pull request names, as kit settle describes. It is safe to run
# again: an issue already closed, or no longer assigned, is left as it is.
issue:settle() {
  local pr=${1#[#!]} line state sha sigil noun url body author repo ref login number istate owner failed=0
  local -a refs=()
  IFS=$'\t' read -r sigil noun < <(forge:noun)
  line=$(forge:pr-state "${pr}") || return 1
  state=${line%%$'\t'*}
  sha=${line#*$'\t'}
  url=$(forge:pr-url "${pr}")
  body=$(forge:pr-body "${pr}") || return 1
  case ${state} in
    MERGED)
      while IFS= read -r ref
      do
        [[ -n "${ref}" ]] || continue
        [[ "$(tracker:of "${ref}" state)" != CLOSED ]] || continue
        if tracker:of "${ref}" comment "Closed by ${noun} ${sigil}${pr}, merged as ${sha:0:12}: ${url}" \
          && tracker:of "${ref}" close completed
        then
          ok "${ref} closed: ${noun} ${sigil}${pr} merged"
        else
          error "${ref} could not be closed"
          failed=1
        fi
      done < <(tracker:body-refs "${body}")
      ;;
    CLOSED)
      author=$(forge:pr-author "${pr}") || return 1
      repo=$(forge:repo) || return 1
      while read -r number istate owner
      do
        [[ "${owner}" == "${repo}" && "${istate}" == OPEN ]] && refs+=("#${number}")
      done < <(forge:pr-closing-issues "${pr}")
      while IFS= read -r ref
      do
        [[ -z "${ref}" ]] || refs+=("${ref}")
      done < <(tracker:body-refs "${body}")
      for ref in "${refs[@]}"
      do
        if tracker:is-forge "${ref}"
        then login=${author}
        else login=$(tracker:at "$(tracker:parse "${ref}" | cut -f 1)" me 2>/dev/null) || continue
        fi
        [[ " $(tracker:of "${ref}" assignees) " == *" ${login} "* ]] || continue
        if tracker:of "${ref}" comment "${noun} ${sigil}${pr} closed without merging; this is backlog again: ${url}" \
          && tracker:of "${ref}" unassign "${login}"
        then
          ok "${ref} is backlog again: ${noun} ${sigil}${pr} closed unmerged, ${login} unassigned"
        else
          error "${ref} could not be put back"
          failed=1
        fi
      done
      ;;
    *)
      info "${noun} ${sigil}${pr} is open; nothing to settle"
      ;;
  esac
  return "${failed}"
}

# issue:branch <reference> <default-branch> <title> prints the issue's branch. It reuses an existing branch, so a
# second run never creates a duplicate. It looks, in order, for:
#
#   1. the branch of an open pull request for the issue
#   2. a branch on origin named <key>-*
#
# When neither exists, it creates the branch:
#
#   - for an issue in the forge's tracker, the forge creates a branch linked to the issue
#   - for any other issue, kit pushes <key>-<slug of the title> to origin from the default branch
issue:branch() {
  local ref=$1 default_branch=$2 title=$3 key branch slug
  key=$(tracker:key "${ref}") || return 2
  branch=$(forge:branch-for-issue "${key}")
  [[ -n "${branch}" ]] \
    || branch=$(git ls-remote --heads origin "${key}-*" 2>/dev/null | sed -n '1s@.*refs/heads/@@p')
  if [[ -z "${branch}" ]] && tracker:is-forge "${ref}"
  then
    step "Linking a branch to ${ref}"
    forge:link-branch "${key}" "${default_branch}" || return 1
    git fetch --quiet origin
    branch=$(git ls-remote --heads origin "${key}-*" 2>/dev/null | sed -n '1s@.*refs/heads/@@p')
    if [[ -z "${branch}" ]]
    then
      error "the branch the forge made for ${ref} could not be found on origin; link one by hand and re-run"
      return 1
    fi
  elif [[ -z "${branch}" ]]
  then
    slug=$(tr '[:upper:]' '[:lower:]' <<<"${title}" | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//' | cut -c 1-60)
    branch="${key}-${slug%-}"
    step "Making ${branch} for ${ref} from ${default_branch}"
    git push --quiet origin "refs/remotes/origin/${default_branch}:refs/heads/${branch}" || return 1
    git fetch --quiet origin
  fi
  printf '%s\n' "${branch}"
}

# issue:worktree <main-root> <branch> prints the worktree the branch is checked out in. If there is none, it
# creates <main-root>/worktrees/<branch>. It refuses, and returns 1, when that path exists and is not a worktree.
issue:worktree() {
  local main_root=$1 branch=$2 wt
  wt=$(git worktree list --porcelain 2>/dev/null \
    | awk -v b="branch refs/heads/${branch}" '/^worktree / { w = substr($0, 10) } $0 == b { print w; exit }')
  if [[ -n "${wt}" ]]
  then
    step "Reusing worktree ${wt}"
  else
    wt="${main_root}/worktrees/${branch}"
    if [[ -e "${wt}" ]]
    then
      error "${wt} exists but is not a worktree of ${branch}; remove it (or run: kit tidy) and re-run."
      return 1
    fi
    step "Creating worktree ${wt}"
    mkdir -p "${main_root}/worktrees" || return 1
    if git show-ref --verify --quiet "refs/heads/${branch}"
    then
      git worktree add --quiet "${wt}" "${branch}" || return 1
    else
      git worktree add --quiet "${wt}" -b "${branch}" "origin/${branch}" || return 1
    fi
  fi
  printf '%s\n' "${wt}"
}

# issue:hydrate <worktree> places what the active packages deliver under .kit/ in the worktree, with the kit that
# runs there: the worktree's own kit/bin/kit where the repository is the kit, which declares `unreleased` and carries
# kit/modules/bash/1st.sh beside it, so the links lead into that worktree's sources, and otherwise this release's kit.
# A worktree whose kit.yml declares no kit version, or has none, is left alone. The version is read as YAML, so
# spaces and a comment beside it do not matter. What hydrate placed is said in one line; its own lines are shown only
# when it fails. Under kit serve, which placed .kit/ in its own checkout only, the new worktree is still placed.
issue:hydrate() {
  local wt=$1 out version
  version=$(yaml:value "${wt}/kit.yml" version) || return 1
  [[ -n "${version}" ]] || return 0
  step "Hydrating ${wt}"
  if [[ "${version}" == unreleased && -x "${wt}/kit/bin/kit" && -f "${wt}/kit/modules/bash/1st.sh" ]]
  then
    out=$(unset kit_serve_hydrated; cd "${wt}" && kit_depth=$(( ${kit_depth:-0} + 1 )) kit/bin/kit hydrate 2>&1) \
      && return 0
  else
    out=$(unset kit_serve_hydrated; cd "${wt}" && run kit hydrate 2>&1) && return 0
  fi
  printf '%s\n' "${out}" >&2
  return 1
}

# issue:start-commit <worktree> <default-branch> <reference> <title> adds an empty first commit when the branch has
# no commits beyond the default branch, because a forge cannot open a pull request with nothing to compare. --only
# keeps the commit empty, so work already staged in a reused worktree stays in the index. The subject starts with
# the reference, as the workflow's commit subjects do: "<reference> start: <title>", kept whatever commit.cleanup
# says. With require_signing true, the commit is signed, and refused when git cannot sign it; a commit that fails for
# another reason, as when a hook refuses it, is said to have failed.
issue:start-commit() {
  local wt=$1 default_branch=$2 ref=$3 title=$4
  local -a sign=()
  [[ -z "$(git -C "${wt}" rev-list "origin/${default_branch}..HEAD" 2>/dev/null)" ]] || return 0
  if [[ "$(settings:value require_signing)" == true ]]
  then
    guard:require signing-key || return 1
    sign=(--gpg-sign)
  fi
  step "Adding the start commit"
  git:commit "${wt}" "the start commit" --quiet --allow-empty --only --cleanup=whitespace "${sign[@]}" \
    --message "${ref} start: ${title}" || return 1
}

# issue:sweep-push <root> <message> stashes all uncommitted work at <root>, including untracked files, and prints
# the new stash's commit hash. The hash stays valid even when other entries are pushed onto or dropped from the
# stash. It prints nothing when refs/stash is the same before and after the push: the work, such as work inside a
# submodule, is not work a stash takes, and the stash on top is one that was there before.
issue:sweep-push() {
  local root=$1 message=$2 before after
  before=$(git -C "${root}" rev-parse --quiet --verify refs/stash) || before=''
  if ! git -C "${root}" stash push --include-untracked --message "${message}" >/dev/null
  then
    error "the uncommitted work at ${root} could not be stashed; resolve it by hand and re-run."
    return 1
  fi
  after=$(git -C "${root}" rev-parse --quiet --verify refs/stash) || after=''
  [[ "${after}" == "${before}" ]] || printf '%s\n' "${after}"
}

# issue:sweep-restore <root> puts the stash in issue_swept back on the desk at <root>, staged work staged again, and
# drops it once it is applied. It does nothing when issue_swept is empty. issue:start registers it to run at exit,
# so work swept from the desk goes back there when the command stops before the worktree has it.
issue:sweep-restore() {
  local root=$1 slot branch
  [[ -n "${issue_swept}" ]] || return 0
  if ! git -C "${root}" stash apply --index "${issue_swept}" >/dev/null 2>&1
  then
    error "the desk's uncommitted work is in the stash: git -C '${root}' stash apply ${issue_swept}"
    return 1
  fi
  slot=$(git -C "${root}" stash list --format='%gd %H' | awk -v h="${issue_swept}" '$2 == h { print $1; exit }')
  if [[ -n "${slot}" ]] && ! git -C "${root}" stash drop "${slot}" >/dev/null
  then
    warn "the swept work is back, and its stash ${issue_swept} is left; drop it by hand: git -C '${root}' stash list"
  fi
  issue_swept=''
  branch=$(git -C "${root}" rev-parse --abbrev-ref HEAD) || branch=HEAD
  ok "The swept work is back on ${branch}."
}

# issue:sweep-apply <root> <worktree> <stash> <default-branch> applies the stash in the worktree, uncommitted, and
# drops it. Then, if the desk is on the default branch, it fast-forwards the now-clean desk. On a conflict it keeps
# the stash and returns non-zero, so the caller stops.
issue:sweep-apply() {
  local root=$1 wt=$2 stash=$3 default_branch=$4 slot
  if ! git -C "${wt}" stash apply "${stash}" >/dev/null 2>&1
  then
    error "the swept work conflicts in ${wt}. It is kept in the stash: git stash apply ${stash}"
    return 1
  fi
  slot=$(git -C "${root}" stash list --format='%gd %H' | awk -v h="${stash}" '$2 == h { print $1; exit }')
  [[ -n "${slot}" ]] && git -C "${root}" stash drop "${slot}" >/dev/null
  ok "Swept the work into ${wt}, uncommitted; commit it there with a real message."
  [[ "$(git -C "${root}" rev-parse --abbrev-ref HEAD)" == "${default_branch}" ]] || return 0
  git -C "${root}" merge --ff-only "origin/${default_branch}" >/dev/null 2>&1 \
    || warn "${default_branch} could not fast-forward to origin; it is left as it was."
}

# issue:pull-request <branch> <default-branch> <title> prints the number of the branch's open pull request,
# opening a draft if there is none.
issue:pull-request() {
  local branch=$1 default_branch=$2 title=$3 pr
  pr=$(forge:pr-for-branch "${branch}")
  if [[ -z "${pr}" ]]
  then
    step "Opening a draft change"
    pr=$(forge:pr-create "${branch}" "${default_branch}" "${title}") || return 1
  fi
  printf '%s\n' "${pr}"
}

# issue:link-in-body <change> <reference>… adds a `Closes <reference>` line to the pull request's body for each
# reference the body does not already close, fix, or resolve. The references are the secondary issues, and a
# primary issue the forge cannot link by itself.
issue:link-in-body() {
  local pr=$1 ref escaped body changed=false
  shift
  body=$(forge:pr-body "${pr}")
  for ref in "$@"
  do
    escaped=$(ere:quote "${ref}")
    if ! grep --quiet --ignore-case --extended-regexp \
      "(close[sd]?|fix(e[sd])?|resolve[sd]?):? ${escaped}([^0-9]|\$)" <<<"${body}"
    then
      body="${body}"$'\n'"Closes ${ref}"
      changed=true
    fi
  done
  if [[ "${changed}" == true ]]
  then
    step "Linking the issues in the change's body"
    forge:pr-set-body "${pr}" "${body}"
  fi
}

# issue:start (<reference>… | --title <title> [<reference>…] | --pr <change> <reference>…) does what the file header
# describes; with --title alone, it creates the issue first. When the require_signing setting is true, it fails before writing anything if git has no signing key
# configured.
issue:start() {
  local n primary title default_branch main_root desk desk_branch branch wt stash="" pr created sigil noun new_title=''
  local -a refs=() in_body=()
  if [[ "${1:-}" == --pr ]]
  then
    shift
    issue:join "$@"
    return
  fi
  if [[ "${1:-}" == --title ]]
  then
    new_title=${2:-}
    shift 2 || shift
  fi
  forge:require || return 1
  if [[ "$(settings:value require_signing)" == true ]]
  then
    guard:require signing-key || return 1
  fi
  if (( $# == 0 ))
  then
    created=$(issue:create "${new_title}") || return
    set -- "${created}"
  elif [[ -n "${new_title}" ]]
  then
    created=$(issue:gather "${new_title}" "$@") || return
    set -- "${created}" "$@"
  fi
  for n in "$@"
  do
    n=$(tracker:normal "${n}") || return 2
    refs+=("${n}")
  done
  set -- "${refs[@]}"
  primary=$1
  tracker:is-forge "${primary}" || in_body+=("${primary}")
  in_body+=("${@:2}")

  git fetch --quiet --prune origin
  default_branch=$(git:default-branch)
  main_root=$(git:main-root)
  desk=$(git rev-parse --show-toplevel)
  desk_branch=$(git -C "${desk}" rev-parse --abbrev-ref HEAD)
  title=$(tracker:of "${primary}" title) || return 1
  section "Starting ${primary}: ${title}"

  branch=$(issue:branch "${primary}" "${default_branch}" "${title}") || return
  if [[ "$(git -C "${main_root}" symbolic-ref --quiet --short HEAD)" == "${branch}" ]]
  then
    error "refusing to start ${primary}: its branch ${branch} is checked out on the desk, ${main_root}, which is the" \
      "human's; switch the desk to another branch, then run kit work again"
    return 1
  fi
  issue:assign "$@" || return
  if [[ "${desk}" == "${main_root}" ]] && ! git:clean "${desk}"
  then
    step "Sweeping the uncommitted work on ${desk_branch} into the worktree"
    stash=$(issue:sweep-push "${desk}" "kit work ${primary} → ${branch}") || return
    if [[ -z "${stash}" ]]
    then
      warn "Nothing to sweep: what is uncommitted on ${desk_branch} is not work a stash takes"
    else
      issue_swept=${stash}
      file:at-exit issue:sweep-restore "${desk}"
    fi
  fi
  wt=$(issue:worktree "${main_root}" "${branch}") || return
  git -C "${main_root}" check-ignore --quiet worktrees 2>/dev/null \
    || warn "worktrees/ is not ignored; add /worktrees/ to .gitignore so a worktree never appears in a commit"
  issue:hydrate "${wt}" || return 1
  issue:start-commit "${wt}" "${default_branch}" "${primary}" "${title}" || return 1
  git:push "${wt}" "${branch}" || return 1
  if [[ -n "${stash}" ]]
  then
    issue_swept=''
    issue:sweep-apply "${desk}" "${wt}" "${stash}" "${default_branch}" || return
  fi
  pr=$(issue:pull-request "${branch}" "${default_branch}" "${title}") || return
  (( ${#in_body[@]} > 0 )) && issue:link-in-body "${pr}" "${in_body[@]}"

  IFS=$'\t' read -r sigil noun < <(forge:noun)
  ok "Draft ${noun} ${sigil}${pr} on ${branch}, primary ${primary}${2:+, secondaries: ${*:2}}"
  ok "Worktree ready: ${wt}"
  step "cd '${wt}'    # do all work there; this checkout stays on ${desk_branch}"
}

# ----------------------------------------------------------------------------------------------------------------------
