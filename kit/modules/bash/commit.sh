# shellcheck shell=bash
# commit.sh - the commit and the push as kit commands: the checks their rules require, then git (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# kit commit and kit submit replace git commit and git push for an agent, as kit rebase replaces git rebase: the kit
# command adds the check the rule requires, and a harness's deny list refuses the raw command. Both run in a worktree
# only. The root checkout is the human's desk, and the default branch advances by a merged change, never by a commit
# made on it.
#
# Before a commit exists, kit commit:
#
#   - refuses a staged path the kit placed under .kit/, read from .kit/.gitignore, which names every one
#   - runs kit lint and kit generate --check over the staged paths, so the pre-commit checks hold with no hook manager
#   - supplies the issue's reference from the branch when the subject carries none, and judges the subject by the
#     repository's policy through the policy seam, as kit policy judges it in CI
#   - warns when commits here are not signed; it cannot make a key appear, so it does not refuse
#
# kit submit commits when there is something to commit, then pushes, setting the upstream on the first push. It never
# rebases: a push origin refuses as not fast-forward names kit rebase as the remedy, so one verb does one thing and a
# conflict with the default branch never blocks sharing work.
#
# see: kit/bin/kit-commit, kit/bin/kit-submit, modules/bash/policy.sh, docs/features.md

use git policy tracker context

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
    error "no branch is checked out here, so there is nothing to commit to; run: kit issue <N>"
    return 1
  fi
  if [[ "$(git rev-parse --show-toplevel)" == "$(git:main-root)" ]]
  then
    error "refusing to commit on ${branch}: this is the root checkout, the human's desk; work in a worktree:" \
      "kit issue <N>"
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

# commit:placed prints each path .kit/.gitignore names, one per line, as kit placed it: .kit/<entry>.
commit:placed() {
  [[ -f .kit/.gitignore ]] || return 0
  sed -n 's@^/@.kit/@p' .kit/.gitignore
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

# commit:judge <branch> <subject> refuses a subject, or a branch, the policy does not allow, with the engine kit policy
# uses and the facts it would gather for this one commit.
commit:judge() {
  local branch=$1 subject=$2 rules input refusals line
  rules=$(policy:rules) || return 1
  input=$(mktemp)
  jq --null-input --arg branch "${branch}" --arg default_branch "$(git:default-branch)" --arg subject "${subject}" \
    --argjson policy "${rules}" \
    '{branch: $branch, default_branch: $default_branch, commits: [{sha: "0000000", subject: $subject}], change: null,
      policy: $policy}' > "${input}"
  refusals=$(policy:deny "${input}")
  rm -f "${input}"
  [[ -n "${refusals}" ]] || return 0
  while IFS= read -r line
  do
    [[ -n "${line}" ]] || continue
    error "${line/#commit 0000000 /the subject }"
  done <<<"${refusals}"
  return 1
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

# commit:commit <argument>… makes the commit as the file header says, from the options commit:parse reads. The first
# --message, or the file's first line, is the subject; the rest are the body. <path>… commits those paths alone, as
# git commit -- <path>… does. It returns 2 on a usage error and 1 on a refusal.
commit:commit() {
  local branch subject path rest message
  local -a staged=()
  commit:parse "$@" || return 2
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
  branch=$(commit:branch) || return 1
  if (( ${#commit_paths[@]} > 0 ))
  then
    for path in "${commit_paths[@]}"
    do
      path=$(context:relative "${path}")
      if ! git ls-files --error-unmatch -- "${path}" >/dev/null 2>&1
      then
        error "${path} is not known to git; run: git add '${path}'"
        return 1
      fi
      staged+=("${path}")
    done
  else
    [[ "${commit_all}" == false ]] || git add --update
    readarray -d '' -t staged < <(git diff --cached --name-only --diff-filter=ACMRD -z)
    if (( ${#staged[@]} == 0 ))
    then
      error "nothing is staged; run: git add <path>…, or kit commit --all for every change to a tracked file"
      return 1
    fi
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
  commit:judge "${branch}" "${subject}" || return 1
  if [[ "$(git config --type=bool commit.gpgsign 2>/dev/null)" != true ]]
  then
    warn "commits here are not signed; set commit.gpgsign to true, with user.signingkey, so the work is signed"
  fi
  message=$(mktemp)
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
  if (( ${#commit_paths[@]} > 0 ))
  then
    git commit --quiet --file "${message}" -- "${staged[@]}" || { rm -f "${message}"; return 1; }
  else
    git commit --quiet --file "${message}" || { rm -f "${message}"; return 1; }
  fi
  rm -f "${message}"
  ok "committed $(git rev-parse --short HEAD): ${subject}"
}

# commit:submit <argument>… commits when something is staged, or when --all or paths say what to take, then pushes the
# branch, setting the upstream on the first push. It never forces: when origin refuses the push, the remedy is kit
# rebase.
commit:submit() {
  local branch upstream ahead refused
  local -a staged=()
  commit:parse "$@" || return 2
  branch=$(commit:branch) || return 1
  [[ "${commit_all}" == false ]] || git add --update
  readarray -d '' -t staged < <(git diff --cached --name-only -z)
  if (( ${#staged[@]} > 0 || ${#commit_paths[@]} > 0 ))
  then
    commit:commit "$@" || return $?
  elif (( ${#commit_messages[@]} > 0 )) || [[ -n "${commit_file}" ]]
  then
    step "nothing is staged, so there is nothing to commit; pushing ${branch}"
  fi
  upstream=$(git rev-parse --abbrev-ref --symbolic-full-name '@{upstream}' 2>/dev/null || true)
  if [[ -z "${upstream}" ]]
  then
    if ! refused=$(git push --quiet --set-upstream origin "${branch}" 2>&1)
    then
      commit:push-refused "${branch}" "${refused}"
      return 1
    fi
    ok "pushed ${branch} to origin, which is its upstream from now on"
    return 0
  fi
  ahead=$(git rev-list --count "${upstream}..HEAD" 2>/dev/null || echo 0)
  if (( ahead == 0 ))
  then
    ok "nothing to submit: ${branch} is at ${upstream}"
    return 0
  fi
  if ! refused=$(git push --quiet 2>&1)
  then
    commit:push-refused "${branch}" "${refused}"
    return 1
  fi
  ok "pushed ${branch}: ${ahead} commit(s)"
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

# ----------------------------------------------------------------------------------------------------------------------
