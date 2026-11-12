# shellcheck shell=bash
# git.sh - git helpers: the default branch, worktrees, kit tidy, and kit rebase (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# kit tidy (git:tidy) decides whether a worktree's work is finished by asking:
#
#   - the forge (GitHub, GitLab), through modules/bash/forge.sh: is the branch's pull request merged or closed?
#   - when the branch has no pull request, the issue's tracker, through modules/bash/tracker.sh: is the branch's issue
#     closed?

use forge

# git:desks <dir> <depth> prints every desk at or below <dir>, at most <depth> levels down, in path order: a
# directory that holds a .git directory. The walk stops at a desk, so a worktree or a nested repository inside one
# is the desk's own, and it skips hidden directories and symbolic links.
git:desks() {
  local dir=$1 depth=$2 sub
  if [[ -d "${dir}/.git" ]]
  then
    printf '%s\n' "${dir}"
    return 0
  fi
  (( depth > 0 )) || return 0
  for sub in "${dir}"/*/
  do
    sub=${sub%/}
    [[ -d "${sub}" && ! -L "${sub}" ]] || continue
    git:desks "${sub}" $(( depth - 1 ))
  done
}

# git:default-branch [<path>] prints the default branch of the repository at <path>: the primary remote's HEAD, else
# main.
git:default-branch() {
  local branch primary
  primary=$(forge:primary) || return $?
  # A primary whose HEAD git has not recorded falls back to main. The ref is read in full: its short name is
  # remotes/<primary>/<branch> where a tag or a branch is named <primary>/<branch>.
  branch=$(git -C "${1:-.}" symbolic-ref --quiet "refs/remotes/${primary}/HEAD" 2>/dev/null || true)
  branch=${branch#"refs/remotes/${primary}"/}
  [[ -n "${branch}" ]] || branch=main
  printf '%s\n' "${branch}"
}

# git:main-root [<path>] prints the root of the main checkout, which contains worktrees/. It works from the main
# checkout or from any linked worktree.
git:main-root() {
  dirname "$(git -C "${1:-.}" rev-parse --path-format=absolute --git-common-dir)"
}

# git:linked-worktrees [<path>] prints "<path>\t<detached>\t<branch>" for every linked worktree of the repository at
# <path>, or of the current one, wherever it lies: every one git worktree list names but the main checkout, which it
# names first. <detached> is true or false, and <branch> is empty for a detached checkout.
git:linked-worktrees() {
  local path="" branch="" detached=false line first=true
  local -a lines
  list:read lines git -C "${1:-.}" worktree list --porcelain || return 1
  # git ends each record with a blank line; one more after the last record prints it too.
  for line in "${lines[@]}" ''
  do
    case "${line}" in
      (worktree\ *)
        path=${line#worktree }
        branch=""
        detached=false
        ;;
      (branch\ refs/heads/*)
        branch=${line#branch refs/heads/}
        ;;
      (detached)
        detached=true
        ;;
      ("")
        [[ -n "${path}" && "${first}" == false ]] && printf '%s\t%s\t%s\n' "${path}" "${detached}" "${branch}"
        [[ -n "${path}" ]] && first=false
        path=""
    esac
  done
}

# git:worktrees prints "<path>\t<detached>\t<branch>" for every linked worktree under worktrees/. <detached> is
# true or false. <branch> is empty for a detached checkout. The empty field comes last so a tab-separated read
# keeps it.
git:worktrees() {
  local worktree_root path="" branch="" detached=false line
  local -a lines
  worktree_root="$(git:main-root)/worktrees"
  list:read lines git worktree list --porcelain || return 1
  for line in "${lines[@]}" ''
  do
    case "${line}" in
      (worktree\ *)
        path=${line#worktree }
        branch=""
        detached=false
        ;;
      (branch\ refs/heads/*)
        branch=${line#branch refs/heads/}
        ;;
      (detached)
        detached=true
        ;;
      ("")
        # git ends each record with a blank line. The loop adds one more so the last record is printed too.
        [[ "${path}" == "${worktree_root}"/* ]] && printf '%s\t%s\t%s\n' "${path}" "${detached}" "${branch}"
        path=""
    esac
  done
}

# git:gone-branches prints every local branch whose upstream no longer exists.
git:gone-branches() {
  git for-each-ref --format='%(refname:short) %(upstream:track)' refs/heads/ | awk '$2 == "[gone]" { print $1 }'
}

# git:clean <path> is true when the checkout at <path> has no uncommitted changes.
git:clean() {
  [[ -z "$(git -C "$1" status --porcelain 2>/dev/null)" ]]
}

# git:unpushed <branch> prints how many commits of the local <branch> no remote-tracking branch reaches. When some are
# left, it asks the forge for the head commit of the branch's latest pull request. When this clone has that commit, it
# leaves out what the head reaches too, so a squash-merged branch whose remote branch is gone counts 0 when its tip
# is the head. When the forge gives a head this clone does not have, it leaves nothing more out, and prints the head
# after the count and a tab: the commits counted may be ones the head reaches, which a fetch of the head would show.
# It returns 1 when the commits cannot be counted, or the change's head cannot be read.
git:unpushed() {
  local branch=$1 count head missing=''
  count=$(git rev-list --count "refs/heads/${branch}" --not --remotes) \
    || { error "the commits of ${branch} could not be counted"; return 1; }
  if (( count > 0 ))
  then
    head=$(forge:pr-head-for-branch "${branch}") || return 1
    if [[ "${head}" =~ ^[0-9a-f]+$ ]] && git cat-file -e "${head}^{commit}" 2>/dev/null
    then
      count=$(git rev-list --count "refs/heads/${branch}" --not --remotes "${head}") \
        || { error "the commits of ${branch} could not be counted"; return 1; }
    elif [[ "${head}" =~ ^[0-9a-f]+$ ]]
    then
      missing=${head}
    fi
  fi
  printf '%s\t%s\n' "${count}" "${missing}"
}

# git:unpushed-why <count> [<head>] prints why a branch with <count> commits git:unpushed counted is kept: commits no
# remote ever had, or, given the change's head this clone lacks, commits no remote-tracking branch here reaches, and
# how to fetch that head, which may reach them.
git:unpushed-why() {
  local primary
  if [[ -z "${2:-}" ]]
  then
    printf '%s commit(s) no remote ever had\n' "$1"
    return 0
  fi
  primary=$(forge:primary 2>/dev/null) || primary=origin
  printf "%s commit(s) no remote-tracking branch here reaches; this clone lacks the change's head %s; fetch it, then" \
    "$1" "$2"
  printf ' run kit tidy again: git fetch %s %s\n' "${primary}" "$2"
}

# git:tidy-settle <branch> settles the issues of the branch's latest pull request, as kit settle does, once kit tidy
# has removed its worktree: what the forge does not close itself is closed or put back here, in case nothing else did.
# A failure is said and does not stop the sweep.
git:tidy-settle() {
  local pr
  [[ -n "$1" ]] || return 0
  if ! pr=$(forge:pr-last-for-branch "$1")
  then
    warn "the change of $1 could not be read, so its issues are not settled; settle them with kit settle and the" \
      "change's number"
    return 0
  fi
  [[ -n "${pr}" ]] || return 0
  use issue
  issue:settle "${pr}" || warn "the issues of change ${pr} could not all be settled; run: kit settle ${pr}"
}

# git:tidy-worktree <path> <branch> <detached> <default-branch> <self-root> <dry-run> decides whether to remove
# one worktree and prints the reason. A worktree whose branch has commits no remote ever had is kept, whatever its
# change's state. It returns 0 when the worktree was removed (or would be, under a dry run) and 1 when it is kept,
# git's refusal to remove it included.
git:tidy-worktree() {
  local path=$1 branch=$2 detached=$3 default_branch=$4 self_root=$5 dry_run=$6 state="" ref primary unpushed
  local counted missing issue_state=''
  if [[ "${path}" == "${self_root}" ]]
  then
    step "keeping ${path} (this command runs in it)"
    return 1
  fi
  if ! git:clean "${path}"
  then
    step "keeping ${path} (${branch:-detached}, it has uncommitted changes)"
    return 1
  fi
  if [[ -n "${branch}" ]]
  then
    if ! state=$(forge:pr-state-for-branch "${branch}")
    then
      step "keeping ${path} (${branch}, its change could not be read)"
      return 1
    fi
    if [[ -z "${state}" ]]
    then
      use tracker
      ref=$(tracker:branch-ref "${branch}")
      if [[ -n "${ref}" ]] && ! issue_state=$(tracker:of "${ref}" state)
      then
        step "keeping ${path} (${branch}, no change, and ${ref}'s state could not be read)"
        return 1
      fi
      [[ -z "${ref}" || "${issue_state}" != CLOSED ]] || state=CLOSED
    fi
  elif [[ "${detached}" == true ]]
  then
    # Remove a detached checkout only when its commit is already on the default branch. Otherwise its HEAD may be
    # the only ref that reaches that commit.
    primary=$(forge:primary) || return 1
    if git -C "${path}" merge-base --is-ancestor HEAD "${primary}/${default_branch}" 2>/dev/null
    then
      state=DETACHED
    else
      step "keeping ${path} (detached at a commit ${primary}/${default_branch} does not reach)"
      return 1
    fi
  fi
  case "${state}" in
    (MERGED | CLOSED | DETACHED)
      if [[ -n "${branch}" ]]
      then
        if ! counted=$(git:unpushed "${branch}")
        then
          step "keeping ${path} (${branch}, change ${state}, its commits could not be counted)"
          return 1
        fi
        IFS=$'\t' read -r unpushed missing <<<"${counted}"
        if (( unpushed > 0 ))
        then
          step "keeping ${path} (${branch}, change ${state}, $(git:unpushed-why "${unpushed}" "${missing}"))"
          return 1
        fi
      fi
      if [[ "${dry_run}" == true ]]
      then
        step "would remove ${path} (${branch:-detached}, ${state})"
      else
        step "removing ${path} (${branch:-detached}, ${state})"
        if ! git worktree remove "${path}"
        then
          step "keeping ${path} (${branch:-detached}, ${state}, git would not remove it)"
          return 1
        fi
        git:tidy-settle "${branch}"
      fi
      return 0
      ;;
    (*)
      step "keeping ${path} (${branch:-detached}, change ${state:-open or unknown})"
      return 1
  esac
}

# git:tidy [--dry-run] cleans up finished work. It:
#
#   1. fetches every remote with --prune: the primary, and the one a branch's upstream is on, as in a fork
#   2. removes every clean worktree under worktrees/ whose pull request is merged or closed (or, with no pull
#      request, whose issue is closed), or that is detached at a commit the default branch already has
#   3. prunes git's worktree bookkeeping
#   4. deletes every local branch whose upstream is gone
#
# It never touches:
#
#   - the default branch
#   - a branch checked out in a worktree it keeps
#   - a branch with commits no remote ever had, and the worktree it is checked out in, whatever its change's state;
#     a commit the change's head reaches, as the forge saw it, was pushed. When this clone lacks that head, the
#     branch is kept, and kit tidy names the head and the fetch that brings it
#   - a worktree with uncommitted changes
#   - a detached worktree whose commit nothing else reaches
#   - a worktree git will not remove, as a locked one
#   - the worktree it runs in
#
# --dry-run prints what it would do and changes nothing. --only=<path> sweeps the one worktree at <path>, as
# git:tidy-wait does once its change merges: steps 2 and 4 take that worktree and its branch alone.
git:tidy() {
  local dry_run=false flag repo default_branch self_root found=false path branch detached remote remotes only=''
  local only_branch='' removed_worktrees=0 removed_branches=0 checked_out unpushed counted missing row
  local -a worktrees
  local -A freed=()
  for flag in "$@"
  do
    case "${flag}" in
      (--dry-run) dry_run=true ;;
      (--only=*) only=${flag#--only=}
    esac
  done
  forge:require || return 1
  repo=$(forge:repo) || return 1
  if [[ "${dry_run}" == true ]]
  then
    section "kit tidy, dry run: nothing is changed"
  else
    section "kit tidy: sweeping ${repo} back to pristine"
  fi

  default_branch=$(git:default-branch) || return $?
  self_root=$(git rev-parse --show-toplevel)
  remotes=$(forge:remotes) || return $?
  while IFS= read -r remote
  do
    [[ -n "${remote}" ]] || continue
    if [[ "${dry_run}" == true ]]
    then
      step "would run: git fetch --prune ${remote}"
    else
      step "git fetch --prune ${remote}"
      git fetch --quiet --prune --end-of-options "${remote}"
    fi
  done <<<"${remotes}"

  section "worktrees whose pull request is merged or closed"
  list:read worktrees git:worktrees || return 1
  for row in "${worktrees[@]}"
  do
    IFS=$'\t' read -r path detached branch <<<"${row}"
    if [[ -n "${only}" ]]
    then
      [[ "${path}" == "${only}" ]] || continue
      only_branch=${branch}
    fi
    found=true
    if git:tidy-worktree "${path}" "${branch}" "${detached}" "${default_branch}" "${self_root}" "${dry_run}"
    then
      (( ++removed_worktrees ))
      [[ -n "${branch}" ]] && freed[${branch}]=1
    fi
  done
  [[ "${found}" == true ]] || ok "none"

  section "worktree bookkeeping"
  local -a prune=(--verbose)
  [[ "${dry_run}" != true ]] || prune+=(--dry-run)
  if git worktree prune "${prune[@]}"
  then
    ok "pruned"
  else
    warn "git worktree prune failed, so the records of removed worktrees stay; see above"
  fi

  section "local branches whose upstream is gone"
  checked_out=$(git worktree list --porcelain | sed -n 's@^branch refs/heads/@@p')
  while IFS= read -r branch
  do
    [[ -n "${branch}" ]] || continue
    [[ -z "${only}" || "${branch}" == "${only_branch}" ]] || continue
    [[ "${branch}" == "${default_branch}" ]] && continue
    if [[ -z "${freed[${branch}]+x}" ]] && grep --quiet --line-regexp --fixed-strings "${branch}" <<<"${checked_out}"
    then
      step "keeping ${branch} (checked out in a worktree)"
      continue
    fi
    if ! counted=$(git:unpushed "${branch}")
    then
      step "keeping ${branch} (its commits could not be counted)"
      continue
    fi
    IFS=$'\t' read -r unpushed missing <<<"${counted}"
    if (( unpushed > 0 ))
    then
      step "keeping ${branch} ($(git:unpushed-why "${unpushed}" "${missing}"))"
      continue
    fi
    if [[ "${dry_run}" == true ]]
    then
      step "would delete ${branch}"
    else
      step "deleting ${branch}"
      if ! git branch -D "${branch}" >/dev/null
      then
        warn "${branch} could not be deleted; see above"
        continue
      fi
    fi
    (( ++removed_branches ))
  done <<<"$(git:gone-branches)"
  (( removed_branches > 0 )) || ok "none"

  if [[ "${dry_run}" == true ]]
  then
    ok "Dry run: would remove ${removed_worktrees} worktree(s) and ${removed_branches} branch(es)."
  else
    ok "Tidied ${repo}: removed ${removed_worktrees} worktree(s) and ${removed_branches} branch(es)."
  fi
}

# git:tidy-wait <reference> watches the change on the branch of the issue's worktree, as kit tidy --wait does, through
# the forge seam: every KIT_TIDY_WAIT_SECONDS, 30 when unset. When the change merges, it runs kit tidy's sweep for
# that worktree alone, git:tidy --only; it returns 0 when the worktree is removed, and 1 when the sweep keeps it, as
# for uncommitted work, saying why. When the change closes unmerged, it says so, keeps the worktree, and returns 1.
# It refuses, and returns 1: no worktree under worktrees/ on a branch for the issue, or several; the worktree it runs
# in, which the sweep would keep; a branch with no change to wait for; and a change three reads in a row cannot read.
# A reference no declared tracker reads, or an interval that is no positive number, is a usage error, 2. ctrl-c,
# SIGINT, stops the wait, removing nothing, and the command exits 130.
git:tidy-wait() {
  local ref key interval=${KIT_TIDY_WAIT_SECONDS:-30} row path detached branch desk state failures=0 kept
  local -a worktrees paths=() branches=()
  use tracker
  ref=$(tracker:normal "$1") || return $?
  key=$(tracker:key "${ref}") || return $?
  if [[ ! "${interval}" =~ ^([0-9]+\.?[0-9]*|\.[0-9]+)$ || ! "${interval}" =~ [1-9] ]]
  then
    error "KIT_TIDY_WAIT_SECONDS must be a number of seconds above 0, not '${interval}'"
    return 2
  fi
  forge:require || return 1
  list:read worktrees git:worktrees || return 1
  for row in "${worktrees[@]}"
  do
    IFS=$'\t' read -r path detached branch <<<"${row}"
    if [[ "${branch}" == "${key}-"* ]]
    then
      paths+=("${path}")
      branches+=("${branch}")
    fi
  done
  if (( ${#paths[@]} == 0 ))
  then
    error "no worktree under worktrees/ is on a branch for ${ref}, so there is nothing to wait for;" \
      "run: kit work ${ref}"
    return 1
  elif (( ${#paths[@]} > 1 ))
  then
    error "several worktrees are on branches for ${ref}, ${branches[*]}, so kit tidy cannot tell which to wait for"
    return 1
  fi
  path=${paths[0]}
  branch=${branches[0]}
  if [[ "${path}" == "$(git rev-parse --show-toplevel)" ]]
  then
    desk=$(out:folder "$(git:main-root)")
    error "run: cd ${desk} && kit tidy --wait=${ref} (kit tidy keeps the worktree it runs in, ${path})"
    return 1
  fi
  # shellcheck disable=SC2064  # the branch and the path are the wait's, fixed when it starts
  trap "git:tidy-wait-stopped '${branch}' '${path}'" INT
  section "kit tidy --wait=${ref}: waiting for the change on ${branch} to merge"
  step "asking every ${interval}s; ctrl-c stops the wait, and removes nothing"
  while :
  do
    if ! state=$(forge:pr-state-for-branch "${branch}")
    then
      if (( ++failures >= 3 ))
      then
        error "the change on ${branch} could not be read three times in a row, so the wait stops; ${path} is kept"
        trap - INT
        return 1
      fi
      warn "the change on ${branch} could not be read; asking again in ${interval}s"
    else
      failures=0
      case "${state}" in
        (MERGED)
          trap - INT
          ok "the change on ${branch} merged; sweeping its worktree"
          git:tidy --only="${path}" || return $?
          kept=$(git worktree list --porcelain) || return 1
          if grep --quiet --line-regexp --fixed-strings "worktree ${path}" <<<"${kept}"
          then
            error "the change on ${branch} merged, and kit tidy kept ${path}; see above"
            return 1
          fi
          return 0
          ;;
        (CLOSED)
          trap - INT
          warn "the change on ${branch} closed unmerged; ${path} is kept, and nothing is removed"
          return 1
          ;;
        ('')
          trap - INT
          error "${branch} has no change to wait for; ${path} is kept; run: kit work ${ref}"
          return 1
          ;;
      esac
    fi
    sleep "${interval}"
  done
}

# git:tidy-wait-stopped <branch> <path> is the INT trap of the wait: it says the wait stopped, with nothing removed,
# and exits 130, as a command stopped by ctrl-c does.
git:tidy-wait-stopped() {
  printf '\n' >&2
  warn "stopped waiting for the change on $1; nothing was removed, and $2 is kept"
  exit 130
}

# git:pushes-to <branch> [<path>] prints the remote that <branch>, in the repository at <path>, pushes to. When the
# branch's upstream is the branch of the same name, it prints branch.<branch>.remote and returns 0. Otherwise, with no
# upstream or one of another name, as a branch started from origin/main has, it prints the push remote, the push
# setting in kit.yml, else the primary, and returns 1. It returns 2, and prints nothing, when kit.yml's push or primary
# names no remote here.
git:pushes-to() {
  local branch=$1 path=${2:-.} remote merge
  # git config --get exits 1 for a key that is not set: a branch with no upstream.
  remote=$(git -C "${path}" config --get "branch.${branch}.remote" 2>/dev/null || true)
  merge=$(git -C "${path}" config --get "branch.${branch}.merge" 2>/dev/null || true)  # as above
  if [[ -n "${remote}" && "${merge}" == "refs/heads/${branch}" ]]
  then
    printf '%s\n' "${remote}"
    return 0
  fi
  remote=$(forge:push-remote) || return 2
  printf '%s\n' "${remote}"
  return 1
}

# git:commit <path> <what> <argument>… runs git commit with the arguments in the checkout at <path>, its standard
# output shown as git writes it and its errors once git ends, and returns git's status. When the commit is not made
# and --gpg-sign is among the arguments, it says why, naming the commit as <what>: refused as one git could not sign
# when git's words say the signing failed (the signing program failed to sign the data, or git could not write the
# commit object, which it writes once it is signed), and failed otherwise, as when a hook refuses it. git's own words
# above say more.
git:commit() {
  local path=$1 what=$2 status=0 said
  shift 2
  { said=$(git -C "${path}" commit "$@" 2>&1 1>&3 3>&-) || status=$?; } 3>&1
  [[ -z "${said}" ]] || printf '%s\n' "${said}" >&2
  (( status != 0 )) || return 0
  list:among --gpg-sign "$@" || return "${status}"
  if grep --quiet --ignore-case --regexp 'failed to sign' --regexp 'failed to write commit object' <<<"${said}"
  then
    error "${what} is refused: require_signing is true and git could not sign it; see above"
  else
    error "${what} failed; see above"
  fi
  return "${status}"
}

# git:push <path> <branch> [<option>…] pushes refs/heads/<branch> of the checkout at <path>, quietly and with the
# options, to the same name on the remote git:pushes-to prints, and sets that as the upstream when git:pushes-to
# returned 1. It names the refspec, so push.default never decides where the branch goes. It pushes nothing, and
# returns 1, when git:pushes-to cannot name the remote.
git:push() {
  local path=$1 branch=$2 remote status=0 upstream=()
  shift 2
  remote=$(git:pushes-to "${branch}" "${path}") || status=$?
  (( status != 2 )) || return 1
  (( status == 0 )) || upstream=(--set-upstream)
  git -C "${path}" push --quiet "${upstream[@]}" "$@" --end-of-options "${remote}" \
    "refs/heads/${branch}:refs/heads/${branch}"
}

# git:rebase [<path>] rebases the branch checked out at <path> (default: the current checkout) onto the default
# branch at the primary remote, then pushes it to its own name on the remote git:pushes-to prints. It refuses:
#
#   - a checkout with no branch checked out
#   - the root checkout (the desk, which belongs to the human), whatever branch it has
#   - the default branch, in any checkout
#   - a worktree with uncommitted changes
#   - with require_signing true, a git with no signing key
#
# When that remote's branch has commits this checkout lacks, another session's, it takes them first: it rebases onto
# that tip, then onto the default branch. With require_signing true, every commit the rebase rewrites is signed. The
# push forces only over the tip it read, so a commit pushed after the fetch makes the push fail rather than vanish. On
# a conflict it aborts the rebase and leaves the worktree as it was.
git:rebase() {
  local worktree=${1:-.} branch default_branch primary remote before theirs target targets=() sign=()
  branch=$(git -C "${worktree}" symbolic-ref --quiet --short HEAD 2>/dev/null || true)  # detached is refused below
  if [[ -z "${branch}" ]]
  then
    error "${worktree} has no branch checked out, so there is nothing to rebase"
    return 1
  fi
  if [[ "$(git -C "${worktree}" rev-parse --show-toplevel)" == "$(git:main-root "${worktree}")" ]]
  then
    error "refusing to rebase ${branch}: it is the root checkout, the human's desk; rebase in the branch's worktree"
    return 1
  fi
  default_branch=$(git:default-branch "${worktree}") || return 1
  if [[ "${branch}" == "${default_branch}" ]]
  then
    error "refusing to rebase ${branch}: it is the default branch; it advances by: git pull --ff-only"
    return 1
  fi
  if ! git:clean "${worktree}"
  then
    error "refusing to rebase ${branch}: the worktree has uncommitted changes; commit or stash them first"
    return 1
  fi
  if [[ "$(settings:value require_signing)" == true ]]
  then
    guard:require signing-key || return 1
    sign=(--gpg-sign)
  fi
  primary=$(forge:primary) || return $?
  # git:pushes-to returns 1 for an upstream of another name, and prints the remote to use all the same.
  remote=$(git:pushes-to "${branch}" "${worktree}") || (( $? == 1 )) || return 1
  section "kit rebase ${branch} onto ${primary}/${default_branch}"
  before=$(git -C "${worktree}" rev-parse HEAD)
  git -C "${worktree}" fetch --quiet --prune --end-of-options "${primary}"
  [[ "${remote}" == "${primary}" ]] || git -C "${worktree}" fetch --quiet --prune --end-of-options "${remote}"
  # A branch the remote does not have gives no tip, and there is nothing of theirs to take.
  theirs=$(git -C "${worktree}" rev-parse --quiet --verify "refs/remotes/${remote}/${branch}^{commit}" || true)
  if [[ -n "${theirs}" ]] && ! git -C "${worktree}" merge-base --is-ancestor "${theirs}" HEAD
  then
    step "taking $(git -C "${worktree}" rev-list --count "HEAD..${theirs}") commit(s)" \
      "${remote}/${branch} has and this checkout lacks"
    targets+=("${remote}/${branch}")
  fi
  targets+=("${primary}/${default_branch}")
  for target in "${targets[@]}"
  do
    if ! git -C "${worktree}" rebase --quiet "${sign[@]}" "${target}"
    then
      # A rebase that stopped before it began has nothing to abort; the reset puts the worktree back either way.
      git -C "${worktree}" rebase --abort 2>/dev/null || true
      git -C "${worktree}" reset --quiet --hard "${before}"
      error "${branch} conflicts with ${target}; the rebase was aborted and the worktree is as it was"
      return 1
    fi
  done
  if ! git:push "${worktree}" "${branch}" "--force-with-lease=refs/heads/${branch}:${theirs}"
  then
    error "${branch} is rebased here and was not pushed; see above, then run: kit rebase"
    return 1
  fi
  ok "${branch} is rebased onto ${primary}/${default_branch} and pushed."
}

# ----------------------------------------------------------------------------------------------------------------------
