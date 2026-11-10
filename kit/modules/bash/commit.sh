# shellcheck shell=bash
# commit.sh - the commit and the push as kit commands: the checks their rules require, then git (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# kit commit and kit submit replace git commit and git push for an agent, as kit rebase replaces git rebase: the kit
# command adds the check the rule requires, and a harness's deny list refuses the raw command. Both run in a worktree
# only. The root checkout is the human's desk, and the default branch advances by a merged change, never by a commit
# made on it.
#
# Before a commit exists, kit commit:
#
#   - refuses a staged path the kit placed under .kit/, read from .kit/.gitignore, which names every one
#   - runs kit lint and kit generate --check over the staged paths, so the pre-commit checks hold with no hook manager
#   - supplies the issue's reference from the branch when the subject carries none
#   - judges the subject, and the tree the commit will make, by the repository's policy through the policy seam, as
#     kit policy judges them in CI
#   - with require_signing true in kit.yml, refuses when git has no user.signingkey; otherwise it warns when commits
#     here are not signed
#
# The commit keeps its message as kit commit judged it: --cleanup=whitespace, so a commit.cleanup of strip never drops
# a subject that starts with #. With require_signing true it is signed, and refused when git cannot sign it.
#
# A commit whose subject names an issue other than the branch's own attaches that issue to the branch's change, so
# kit commit then adds a `Closes <reference>` line to the change's body, and assigns the caller to the issue when no
# one is assigned. The commit exists by then, so a forge that fails only warns.
#
# kit submit commits when there is something to commit, then pushes the branch to the branch of its own name on its
# remote, setting the upstream on the first push. Its remote is origin, unless its upstream is a branch of its own
# name on another remote. It names the refspec, so push.default and an upstream of another name, such as the
# origin/main a branch started from, never decide where the branch goes. It never rebases: a push the remote refuses
# as not fast-forward names kit rebase as the remedy, so one verb does one thing and a conflict with the default
# branch never blocks sharing work.
#
# see: kit/bin/kit-commit, kit/bin/kit-submit, modules/bash/policy.sh, docs/features.md

use git policy facts tracker context settings

# commit:parse <argument>… reads the options both verbs share into commit_all, commit_messages, commit_file, and
# commit_paths. guard:arguments has judged the shape of the line already, so this reads what it let through.
commit:parse() {
  local arg
  commit_all=false
  commit_messages=()
  commit_file=''
  commit_paths=()
  while (( $# > 0 ))
  do
    arg=$1
    shift
    case "${arg}" in
      (--all) commit_all=true ;;
      (--message)
        commit_messages+=("$1")
        shift ;;
      (--message=*) commit_messages+=("${arg#--message=}") ;;
      (--file)
        commit_file=$1
        shift ;;
      (--file=*) commit_file=${arg#--file=} ;;
      (--) commit_paths+=("$@"); break ;;
      (*) commit_paths+=("${arg}")
    esac
  done
}

# commit:branch prints the branch checked out here, or refuses: no branch, the root checkout, or the default branch.
commit:branch() {
  local branch default_branch
  branch=$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)
  if [[ -z "${branch}" ]]
  then
    error "no branch is checked out here, so there is nothing to commit to; run: kit work <N>"
    return 1
  fi
  if [[ "$(git rev-parse --show-toplevel)" == "$(git:main-root)" ]]
  then
    error "refusing to commit on ${branch}: this is the root checkout, the human's desk; work in a worktree:" \
      "kit work <N>"
    return 1
  fi
  default_branch=$(git:default-branch)
  if [[ "${branch}" == "${default_branch}" ]]
  then
    error "refusing to commit on ${branch}: it is the default branch, which advances by a merged change"
    return 1
  fi
  printf '%s\n' "${branch}"
}

# commit:placed prints each path .kit/.gitignore names, one per line, as kit placed it: .kit/<entry>. A directory,
# such as /logs/, loses its trailing slash, so commit:guard holds every path under it.
commit:placed() {
  [[ -f .kit/.gitignore ]] || return 0
  sed -n -e '/^\//!d' -e 's@^/@.kit/@' -e 's@/*$@@' -e p .kit/.gitignore
}

# commit:guard <path>… refuses a path kit placed, or one under a directory it placed. The paths are root-relative.
commit:guard() {
  local path placed refused=0
  local -a placed_paths=()
  readarray -t placed_paths < <(commit:placed)
  for path in "$@"
  do
    for placed in "${placed_paths[@]}"
    do
      if [[ "${path}" == "${placed}" || "${path}" == "${placed}"/* ]]
      then
        error "${path} is kit's, placed from a package; run: git rm --cached '${path}' (a delivered file never" \
          "reaches a commit)"
        refused=1
        break
      fi
    done
  done
  (( refused == 0 ))
}

# commit:with-reference <branch> <subject> prints the subject with the branch's issue reference in front, when the
# subject carries no reference of its own and the branch was made for an issue.
commit:with-reference() {
  local branch=$1 subject=$2 reference
  if [[ "${subject}" =~ ^([A-Z]+)?#[0-9]+\  ]]
  then
    printf '%s\n' "${subject}"
    return 0
  fi
  reference=$(tracker:branch-ref "${branch}")
  if [[ -n "${reference}" ]]
  then
    step "subject: ${reference} supplied from the branch"
    printf '%s %s\n' "${reference}" "${subject}"
  else
    printf '%s\n' "${subject}"
  fi
}

# commit:attach <branch> <subject> closes the issue the subject names in the branch's open change, and assigns the
# caller to it when it has no assignee, when that issue is not the branch's own. It asks the forge nothing otherwise,
# nor in a repository with no forge, and it only warns: the commit exists already.
commit:attach() {
  local branch=$1 subject=$2 ref pr prefix me assignees
  [[ "${subject}" =~ ^(([A-Z]+)?#[0-9]+)\  ]] || return 0
  ref=$(tracker:normal "${BASH_REMATCH[1]}" 2>/dev/null) || return 0
  [[ "${ref}" != "$(tracker:branch-ref "${branch}")" ]] || return 0
  use issue
  forge:provider >/dev/null 2>&1 || return 0
  pr=$(forge:pr-for-branch "${branch}")
  if [[ -z "${pr}" ]]
  then
    warn "${ref} is not closed by a change: ${branch} has no open change; add Closes ${ref} to its body when it has one"
  else
    issue:link-in-body "${pr}" "${ref}" \
      || warn "${ref} could not be added to change ${pr}'s body; add Closes ${ref} by hand"
  fi
  prefix=$(tracker:parse "${ref}" | cut -f 1)
  assignees=$(tracker:of "${ref}" assignees) || { warn "${ref}'s assignees could not be read"; return 0; }
  [[ -z "${assignees}" ]] || return 0
  me=$(tracker:at "${prefix}" me) || return 0
  if tracker:of "${ref}" assign "${me}"
  then
    ok "${ref} assigned to ${me}, as no one was"
  else
    warn "${ref} could not be assigned to ${me}; assign it by hand"
  fi
}

# commit:judge <branch> <subject> [<path>…] refuses a subject, a branch, or a tree the policy does not allow. It prints
# warnings first, and they never refuse. It uses the engine kit policy uses, with the facts of the commit moment: this
# one commit, and the tree it will make. With
# <path>…, that tree is HEAD and those paths as the disk has them, as git commit --only makes it. It is built in an
# index of its own, so what another session staged never shows in it. With no path, the tree is the index.
commit:judge() {
  local branch=$1 subject=$2 work warnings refusals line
  shift 2
  file:scratch work || return 1
  if (( $# > 0 ))
  then
    if ! commit:index "${work}/index" "$@"
    then
      commit:unjudged "${work}" "the tree of the named paths could not be built"
      return 1
    fi
    # The policy reads kit.yml and the repository's Rego as the commit will have them, never as the disk does.
    GIT_INDEX_FILE=${work}/index policy:from '' "${work}/policy" || { commit:unjudged "${work}"; return 1; }
    GIT_INDEX_FILE=${work}/index facts:document commit > "${work}/facts" || { commit:unjudged "${work}"; return 1; }
  else
    policy:from '' "${work}/policy" || { commit:unjudged "${work}"; return 1; }
    facts:document commit > "${work}/facts" || { commit:unjudged "${work}"; return 1; }
  fi
  jq --arg branch "${branch}" --arg subject "${subject}" \
    '.branch = $branch | .commits = [{sha: "0000000", subject: $subject}]' "${work}/facts" > "${work}/input" \
    || { commit:unjudged "${work}"; return 1; }
  # An engine that cannot judge refuses the commit: unjudged is never allowed.
  if ! refusals=$(policy:deny "${work}/input")
  then
    commit:unjudged "${work}" "the policy could not be judged"
    return 1
  fi
  # A warning is printed before any refusal and never fails the commit; one that cannot be judged does.
  if ! warnings=$(policy:warn "${work}/input")
  then
    commit:unjudged "${work}" "the policy could not be judged"
    return 1
  fi
  policy:from-disk
  rm -rf "${work}"
  while IFS= read -r line
  do
    [[ -n "${line}" ]] || continue
    warn "${line/#commit 0000000 /the subject }"
  done <<<"${warnings}"
  [[ -n "${refusals}" ]] || return 0
  while IFS= read -r line
  do
    [[ -n "${line}" ]] || continue
    error "${line/#commit 0000000 /the subject }"
  done <<<"${refusals}"
  return 1
}

# commit:unjudged <work> [<why>] removes the judge's scratch directory, puts the policy back on the disk's settings, and
# says why the commit is refused: by default, the facts for the policy could not be gathered.
commit:unjudged() {
  policy:from-disk
  rm -rf "$1"
  error "the commit is refused: ${2:-the facts for the policy could not be gathered}"
}

# commit:index <index> <path>… writes, at <index>, the index git commit --only -- <path>… builds: HEAD, or nothing
# before the first commit, and each named path as the disk has it. A path the disk lacks is removed. The paths are
# those the real index or HEAD holds under <path>…, so a file git ignores or does not track never joins, as with git.
# The real index is only read.
commit:index() {
  local index=$1
  shift
  if git rev-parse --verify --quiet HEAD >/dev/null
  then
    GIT_INDEX_FILE=${index} git read-tree HEAD || return 1
  else
    GIT_INDEX_FILE=${index} git read-tree --empty || return 1
  fi
  { git ls-files -z -- "${@/#/:(literal)}" && GIT_INDEX_FILE=${index} git ls-files -z -- "${@/#/:(literal)}"; } \
    | LC_ALL=C sort --zero-terminated --unique \
    | GIT_INDEX_FILE=${index} git update-index --add --remove --replace -z --stdin
}

# commit:checks <path>… runs the pre-commit checks over the paths that exist: kit lint, then kit generate --check.
# The paths are root-relative; they are passed absolute, so the verbs resolve them the same from any directory.
commit:checks() {
  local path
  local -a present=()
  for path in "$@"
  do
    [[ -e "${path}" ]] && present+=("${PWD}/${path}")
  done
  (( ${#present[@]} > 0 )) || return 0
  if ! run kit lint "${present[@]}"
  then
    error "the commit is refused: a linter failed over the staged paths"
    return 1
  fi
  if ! run kit generate --check "${present[@]}"
  then
    error "the commit is refused: a region is stale; run: kit generate"
    return 1
  fi
}

# commit:behind <path>… refuses a staged path that would undo a commit the index never saw. Such a path is staged as it
# was before the last commit on HEAD that changed it, absent when that commit added it, while the disk holds something
# else: an index another session's commit passed by. A change put back on purpose matches the disk, and passes. It
# names every such path and returns 1.
commit:behind() {
  local path last before staged behind=false
  for path in "$@"
  do
    last=$(git log -1 --format=%H HEAD -- ":(literal)${path}")
    [[ -n "${last}" ]] || continue
    before=$(git rev-parse --verify --quiet "${last}^:${path}" || true)
    staged=$(git ls-files --stage -- ":(literal)${path}" | cut -d ' ' -f 2)
    [[ "${staged}" == "${before}" ]] || continue
    if [[ -n "${staged}" ]]
    then
      git diff --quiet -- ":(literal)${path}" && continue
    else
      [[ -e "${path}" || -L "${path}" ]] || continue
    fi
    error "${path} is staged as it was before $(git log -1 --format='%h %s' "${last}"); committing it would undo" \
      "that commit; run: git restore --staged -- '${path}'"
    behind=true
  done
  [[ "${behind}" == false ]]
}

# commit:named <path>… prints, NUL-separated, every file git knows under the named paths: in the index, or in HEAD,
# which holds what git mv or git rm took away. A named path is a path, never a pattern, and a directory names every
# file under it. It refuses a path under which git knows no file, naming it, and returns 1.
commit:named() {
  local path
  local -a known=() files=()
  # A path git mv or git rm took away is in HEAD alone, which git commit takes; a first commit has no HEAD.
  git rev-parse --verify --quiet HEAD >/dev/null && known=(--with-tree=HEAD)
  for path in "$@"
  do
    readarray -d '' -t files < <(git ls-files -z "${known[@]}" -- ":(literal)${path}")
    if (( ${#files[@]} == 0 ))
    then
      error "${path} is not known to git; run: git add '${path}'"
      return 1
    fi
    printf '%s\0' "${files[@]}"
  done | LC_ALL=C sort --zero-terminated --unique
  local -a statuses=("${PIPESTATUS[@]}")
  (( statuses[0] == 0 && statuses[1] == 0 ))
}

# commit:unchanged <file>… succeeds when every file is as HEAD has it on the disk, so committing them alone would
# commit nothing. Before the first commit, nothing is unchanged.
commit:unchanged() {
  git rev-parse --verify --quiet HEAD >/dev/null || return 1
  git diff --quiet HEAD -- "${@/#/:(literal)}"
}

# commit:tree [<file>…] prints the tree the commit would make: the index's, or, with files, HEAD's with those files as
# the disk has them, built in an index of its own. kit commit takes it before its checks and again before git
# commits, so what another session staged or wrote meanwhile is never committed unchecked.
commit:tree() {
  local work tree
  if (( $# == 0 ))
  then
    git write-tree
    return
  fi
  file:scratch work || return 1
  commit:index "${work}/index" "$@" || return 1
  tree=$(GIT_INDEX_FILE=${work}/index git write-tree) || return 1
  rm -rf "${work}"
  printf '%s\n' "${tree}"
}

# commit:moved <head> refuses when HEAD is no longer <head>, the commit kit commit started on: another session
# committed while the checks ran. Without paths, the index never saw that commit; with paths, the policy never judged
# the tree it makes. Git refuses a HEAD that moves during its own hook.
commit:moved() {
  local head
  head=$(git rev-parse --verify --quiet HEAD || true)
  [[ "${head}" != "$1" ]] || return 0
  error "HEAD moved to $(git log -1 --format='%h %s') while kit commit checked; nothing is committed; run kit commit again"
  return 1
}

# commit:same-tree <tree> [<file>…] refuses when the tree the commit would make is no longer <tree>, the one kit commit
# checked: another session staged or wrote a path meanwhile.
commit:same-tree() {
  local tree
  tree=$(commit:tree "${@:2}") || return 1
  [[ "${tree}" != "$1" ]] || return 0
  error "the index changed while kit commit checked, so what was checked is not what git would commit; nothing is" \
    "committed; run kit commit again"
  return 1
}

# commit:commit <argument>… makes the commit as the file header says, from the options commit:parse reads. The first
# --message, or the file's first line, is the subject; the rest are the body. <path>… commits those paths alone, as
# git commit -- <path>… does. It returns 2 on a usage error and 1 on a refusal.
commit:commit() {
  local branch subject path rest message head tree names
  local -a staged=() named=() sign=()
  commit:parse "$@" || return 2
  head=$(git rev-parse --verify --quiet HEAD || true)
  if (( ${#commit_messages[@]} == 0 )) && [[ -z "${commit_file}" ]]
  then
    error "say what the commit is: --message <text>, or --file <path>; kit commit opens no editor; run: kit help commit"
    return 2
  fi
  if [[ -n "${commit_file}" ]]
  then
    commit_file=$(context:relative "${commit_file}")
    if [[ ! -f "${commit_file}" ]]
    then
      error "no message file at ${commit_file}; run: kit help commit"
      return 2
    fi
  fi
  if [[ "${commit_all}" == true ]] && (( ${#commit_paths[@]} > 0 ))
  then
    error "--all takes every change to a tracked file, so it does not go with named paths; name the paths, or use" \
      "--all alone"
    return 2
  fi
  branch=$(commit:branch) || return 1
  if [[ "$(settings:value require_signing)" == true ]]
  then
    guard:require signing-key || return 1
    sign=(--gpg-sign)
  fi
  if (( ${#commit_paths[@]} > 0 ))
  then
    for path in "${commit_paths[@]}"
    do
      named+=("$(context:relative "${path}")")
    done
    names=$(commit:named "${named[@]}" | tr '\0' '\n') || return 1
    readarray -t staged <<<"${names}"
    if commit:unchanged "${staged[@]}"
    then
      if (( ${#named[@]} == 1 ))
      then
        error "nothing to commit: ${named[0]} is as HEAD has it"
      else
        error "nothing to commit: ${named[*]} are as HEAD has them"
      fi
      return 1
    fi
  else
    [[ "${commit_all}" == false ]] || git add --update
    readarray -d '' -t staged < <(git diff --cached --name-only --diff-filter=ACMRD -z)
    if (( ${#staged[@]} == 0 ))
    then
      error "nothing is staged; run: git add <path>…, or kit commit --all for every change to a tracked file"
      return 1
    fi
    commit:behind "${staged[@]}" || return 1
  fi
  if (( ${#commit_paths[@]} > 0 ))
  then
    tree=$(commit:tree "${staged[@]}") || return 1
  else
    tree=$(commit:tree) || return 1
  fi
  if [[ -n "${commit_file}" ]]
  then
    subject=$(head -n 1 "${commit_file}")
  else
    subject=${commit_messages[0]%%$'\n'*}
  fi
  subject=$(commit:with-reference "${branch}" "${subject}")
  commit:guard "${staged[@]}" || return 1
  commit:checks "${staged[@]}" || return 1
  # Named paths are committed alone, so the policy judges the tree they make; otherwise, the index is the tree.
  if (( ${#commit_paths[@]} > 0 ))
  then
    commit:judge "${branch}" "${subject}" "${staged[@]}" || return 1
  else
    commit:judge "${branch}" "${subject}" || return 1
  fi
  if (( ${#sign[@]} == 0 )) && [[ "$(git config --type=bool commit.gpgsign 2>/dev/null)" != true ]]
  then
    warn "commits here are not signed; set commit.gpgsign to true, with user.signingkey, so the work is signed"
  fi
  file:temp message message || return 1
  {
    printf '%s\n' "${subject}"
    if [[ -n "${commit_file}" ]]
    then
      tail -n +2 "${commit_file}"
    else
      rest=${commit_messages[0]#*$'\n'}
      [[ "${rest}" == "${commit_messages[0]}" ]] || printf '%s\n' "${rest}"
      for rest in "${commit_messages[@]:1}"
      do
        printf '\n%s\n' "${rest}"
      done
    fi
  } > "${message}"
  # A HEAD that moved during the checks was never judged, so the commit is refused either way, and so is a tree that
  # changed under the checks.
  commit:moved "${head}" || { rm -f "${message}"; return 1; }
  if (( ${#commit_paths[@]} > 0 ))
  then
    commit:same-tree "${tree}" "${staged[@]}" || { rm -f "${message}"; return 1; }
    commit:git "${message}" "${sign[@]}" -- "${staged[@]/#/:(literal)}" || return 1
  else
    commit:same-tree "${tree}" || { rm -f "${message}"; return 1; }
    commit:git "${message}" "${sign[@]}" || return 1
  fi
  subject=$(git log -1 --format=%s HEAD)
  ok "committed $(git rev-parse --short HEAD): ${subject}"
  commit:attach "${branch}" "${subject}"
}

# commit:git <message> [--gpg-sign] [-- <path>…] runs git commit with the message file as written: --cleanup=whitespace
# keeps a line that starts with #, whatever commit.cleanup says. It removes the message file either way. With
# --gpg-sign, a commit git fails to make is refused as one git could not sign, and git's own words above say why.
commit:git() {
  local message=$1 status=0
  shift
  git commit --quiet --cleanup=whitespace --file "${message}" "$@" || status=$?
  rm -f "${message}"
  (( status != 0 )) || return 0
  [[ "${1:-}" != --gpg-sign ]] \
    || error "the commit is refused: require_signing is true and git could not sign it; see above"
  return 1
}

# commit:submit <argument>… commits when something is staged, or when --all or paths say what to take, then pushes the
# branch with git:push, to its own name on the remote git:pushes-to prints, setting the upstream on the first push. It
# never forces: when the remote refuses the push, the remedy is kit rebase.
commit:submit() {
  local branch remote first=false ahead refused
  local -a staged=()
  commit:parse "$@" || return 2
  branch=$(commit:branch) || return 1
  [[ "${commit_all}" == false ]] || git add --update
  readarray -d '' -t staged < <(git diff --cached --name-only -z)
  if (( ${#commit_paths[@]} > 0 )) && commit:paths-unchanged "${commit_paths[@]}"
  then
    step "nothing to commit: the named paths are as HEAD has them; pushing ${branch}"
  elif (( ${#staged[@]} > 0 || ${#commit_paths[@]} > 0 ))
  then
    commit:commit "$@" || return $?
  elif (( ${#commit_messages[@]} > 0 )) || [[ -n "${commit_file}" ]]
  then
    step "nothing is staged, so there is nothing to commit; pushing ${branch}"
  fi
  # A branch with no upstream of its own name, such as one started from origin/main, has its first push here.
  remote=$(git:pushes-to "${branch}") || first=true
  if [[ "${first}" == true ]]
  then
    if ! refused=$(git:push . "${branch}" 2>&1)
    then
      commit:push-refused "${branch}" "${refused}"
      return 1
    fi
    ok "pushed ${branch} to ${remote}, which is its upstream from now on"
    return 0
  fi
  # A remote-tracking ref that is gone, as after a prune once the remote's branch was deleted, is no measure of what
  # the remote has: the push is a first push in effect.
  if ! git rev-parse --quiet --verify "refs/remotes/${remote}/${branch}^{commit}" >/dev/null
  then
    if ! refused=$(git:push . "${branch}" 2>&1)
    then
      commit:push-refused "${branch}" "${refused}"
      return 1
    fi
    ok "pushed ${branch} to ${remote}; this clone had no ${remote}/${branch}"
    return 0
  fi
  ahead=$(git rev-list --count "refs/remotes/${remote}/${branch}..HEAD" 2>/dev/null || echo 0)
  if (( ahead == 0 ))
  then
    ok "nothing to submit: ${branch} is at ${remote}/${branch}"
    return 0
  fi
  if ! refused=$(git:push . "${branch}" 2>&1)
  then
    commit:push-refused "${branch}" "${refused}"
    return 1
  fi
  ok "pushed ${branch}: ${ahead} commit(s)"
}

# commit:paths-unchanged <path>… succeeds when git knows every named path and every file under them is as HEAD has
# it. A path git does not know is left for kit commit to refuse.
commit:paths-unchanged() {
  local path names
  local -a named=() files=()
  for path in "$@"
  do
    named+=("$(context:relative "${path}")")
  done
  names=$(commit:named "${named[@]}" 2>/dev/null | tr '\0' '\n') || return 1
  readarray -t files <<<"${names}"
  commit:unchanged "${files[@]}"
}

# commit:push-refused <branch> <git-output> says why the push did not happen, with the remedy, and git's own words
# as the detail.
commit:push-refused() {
  local line
  if grep --quiet 'non-fast-forward\|fetch first\|stale info' <<<"$2"
  then
    error "origin refused the push of $1: it has commits this branch lacks; run: kit rebase"
  else
    error "origin refused the push of $1"
  fi
  while IFS= read -r line
  do
    [[ -n "${line}" ]] && step "${line}"
  done <<<"$2"
}

# commit:choices <key> [<answer>…] answers kit syntax --choices for <path>: an open set, every path git status reports,
# with what became of it as the description, and active for a path that is staged already. kit submit answers the same.
commit:choices() {
  local record xy path skip=false description state
  local -a records
  [[ "$1" == '<path>' ]] || return 1
  if [[ "${2:-}" == --kind ]]
  then
    printf 'open\n'
    return 0
  fi
  readarray -d '' -t records < <(git status --porcelain=v1 -z)
  for record in "${records[@]}"
  do
    # a rename's or copy's old name is the record after it
    if [[ "${skip}" == true ]]
    then
      skip=false
      continue
    fi
    xy=${record:0:2} path=${record:3}
    [[ "${xy:0:1}" != [RC] ]] || skip=true
    case "${xy}" in
      ('??') description=untracked ;;
      (' M' | MM | AM) description=modified ;;
      (' D') description=deleted ;;
      ('A ') description=added ;;
      ('M ') description=staged ;;
      ('D ') description='deleted, staged' ;;
      (R* | C*) description=renamed ;;
      (*) description=changed
    esac
    state=''
    [[ "${xy:0:1}" == [' ?'] ]] || state=active
    printf '%s\t%s\t%s\t\n' "${path}" "${description}" "${state}"
  done
  return 0
}

# ----------------------------------------------------------------------------------------------------------------------
