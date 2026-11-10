# shellcheck shell=bash
# workspace.sh - kit workspace's stages: each git entity of a repository as rows (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
#
# A stage reads one repository, the one whose checkout kit runs in, from its main checkout, so a linked worktree reads
# the same as the desk. It reads what git keeps here alone and asks no remote: what a remote has is what the last
# fetch left. Each prints its rows as JSON, as workspace.jq makes them from what git printed.
#
# see: bin/kit-workspace, modules/jq/workspace.jq
use forge git workspaces

# workspace_stages are kit workspace's stages, in its order.
# shellcheck disable=SC2034  # read by bin/kit-workspace
workspace_stages=(repos remotes branches worktrees commits tags)

# workspace_commits is the most commits the commits stage lists, and workspace_tags the most tags, the newest first.
workspace_commits=200
workspace_tags=100

# workspace:stage <stage> prints the stage's rows, as a JSON array, read from the main checkout. repos is no stage of
# one repository's: across:repos lists the workspace's members.
workspace:stage() {
  local stage=$1
  (
    cd "$(git:main-root)" || exit 1
    "workspace:${stage}"
  )
}

# workspace:format <field>… prints a git for-each-ref format of the fields, apart by the unit separator, as
# workspace.jq reads them.
workspace:format() {
  printf '%%(%s)' "$@" | sed 's/)%(/)%1f%(/g'
}

# workspace:paths prints the path of every worktree of the repository here, the main checkout first.
workspace:paths() {
  local line
  local -a lines
  list:read lines git worktree list --porcelain || return 1
  for line in "${lines[@]}"
  do
    [[ "${line}" != 'worktree '* ]] || printf '%s\n' "${line#worktree }"
  done
}

# workspace:remote-facts prints each remote of the repository here, the primary first, as a JSON array of
# {name, url, push, forge, member, web, default, branches}: its fetch and push URLs, its forge, the member its URL
# names, host and path, its page, its default branch, as its HEAD names it, and how many branches it has.
workspace:remote-facts() {
  local remote url push forge member default branches
  local -a remotes
  {
    list:read remotes forge:remotes || exit 1
    for remote in "${remotes[@]}"
    do
      [[ -n "${remote}" ]] || continue
      url=$(git remote get-url --end-of-options "${remote}" 2>/dev/null) || url=''
      push=$(git remote get-url --push --end-of-options "${remote}" 2>/dev/null) || push=${url}
      forge=$(forge:of "${remote}" 2>/dev/null) || forge=''
      member=$(workspaces:url-member "${url}")
      default=$(git symbolic-ref --quiet --short "refs/remotes/${remote}/HEAD" 2>/dev/null || true)
      default=${default#"${remote}"/}
      branches=$(git for-each-ref --format='%(symref)' "refs/remotes/${remote}/" | grep --count '^$' || true)
      jq --null-input --compact-output -L "${kit_jq}" --arg name "${remote}" --arg url "${url}" --arg push "${push}" \
        --arg forge "${forge}" --arg member "${member}" --arg default "${default}" --argjson branches "${branches}" '
        import "workspace" as workspace;
        def some: if . == "" then null else . end;
        {name: $name, url: ($url | some), push: ($push | some), forge: ($forge | some), member: ($member | some),
         default: ($default | some), branches: $branches}
        | . + {web: workspace::web(.forge; .member)}'
    done
  } | jq --slurp --compact-output .
}

# workspace:remotes prints a row per remote, as workspace::remotes gives them.
workspace:remotes() {
  local primary
  primary=$(forge:primary) || return $?
  workspace:remote-facts | jq --compact-output -L "${kit_jq}" --arg primary "${primary}" \
    'import "workspace" as workspace; workspace::remotes($primary)'
}

# workspace:branches prints a row per branch, local and on a remote, as workspace::branches gives them.
workspace:branches() {
  local facts
  facts=$(workspace:remote-facts) || return $?
  git for-each-ref refs/heads refs/remotes --format="$(workspace:format refname symref upstream:remotename \
    upstream:remoteref upstream:track worktreepath objectname:short committerdate:unix contents:subject)" \
    | jq --compact-output --raw-input --slurp -L "${kit_jq}" --argjson now "$(date +%s)" \
      --argjson remotes "${facts}" 'import "workspace" as workspace; workspace::branches($remotes; $now)'
}

# workspace:base prints the ref of the primary's default branch, as the last fetch left it, or nothing where there is
# none: no primary remote, or never fetched.
workspace:base() {
  local primary default
  primary=$(forge:primary) || return $?
  default=$(git:default-branch) || return $?
  git rev-parse --verify --quiet "refs/remotes/${primary}/${default}" >/dev/null || return 0
  printf 'refs/remotes/%s/%s\n' "${primary}" "${default}"
}

# workspace:worktrees prints a row per worktree, the desk first, as workspace::worktrees gives them: each one's status
# read in it, and what it is ahead and behind the primary's default branch.
workspace:worktrees() {
  local base path statuses='{}' bases='{}' status counts
  local -a paths
  base=$(workspace:base) || return $?
  list:read paths workspace:paths || return 1
  for path in "${paths[@]}"
  do
    [[ -d "${path}" ]] || continue
    status=$(git -C "${path}" status --porcelain=v2 --branch 2>/dev/null) || continue
    statuses=$(jq --compact-output --arg path "${path}" --arg status "${status}" '. + {($path): $status}' \
      <<<"${statuses}")
    [[ -n "${base}" ]] || continue
    counts=$(git -C "${path}" rev-list --left-right --count "${base}...HEAD" 2>/dev/null) || continue
    bases=$(jq --compact-output --arg path "${path}" --arg counts "${counts}" '. + {($path): $counts}' <<<"${bases}")
  done
  git worktree list --porcelain \
    | jq --compact-output --raw-input --slurp -L "${kit_jq}" --argjson statuses "${statuses}" \
      --argjson bases "${bases}" 'import "workspace" as workspace; workspace::worktrees($statuses; $bases)'
}

# workspace:commits prints a row per commit a local branch holds and the primary's default branch does not, the newest
# first, at most workspace_commits, as workspace::commits gives them; with no default branch to leave out, the newest
# the local branches hold. Each says the local branches that hold it, whether a remote has it, and the worktrees whose
# HEAD it is.
workspace:commits() {
  local base primary branch held='' unpushed heads facts
  local -a not=() names
  base=$(workspace:base) || return $?
  [[ -z "${base}" ]] || not=(--not "${base}")
  primary=$(forge:primary) || return $?
  list:read names git for-each-ref --format='%(refname:short)' refs/heads || return 1
  for branch in "${names[@]}"
  do
    [[ -n "${branch}" ]] || continue
    held+=$(git rev-list --max-count="${workspace_commits}" "refs/heads/${branch}" "${not[@]}" \
      | sed "s|\$|$(printf '\t')${branch}|")$'\n'
  done
  unpushed=$(git rev-list --max-count=1000 --branches --not --remotes \
    | jq --compact-output --raw-input --slurp -L "${kit_jq}" 'import "workspace" as workspace; workspace::hashes')
  heads=$(git worktree list --porcelain \
    | jq --compact-output --raw-input --slurp -L "${kit_jq}" 'import "workspace" as workspace; workspace::heads')
  held=$(jq --compact-output --raw-input --slurp -L "${kit_jq}" 'import "workspace" as workspace; workspace::held' \
    <<<"${held}")
  facts=$(workspace:remote-facts) || return $?
  git log --branches "${not[@]}" --max-count="${workspace_commits}" --format='%H%x1f%h%x1f%ct%x1f%an%x1f%s' \
    | jq --compact-output --raw-input --slurp -L "${kit_jq}" --argjson now "$(date +%s)" \
      --argjson branches "${held}" --argjson unpushed "${unpushed}" --argjson heads "${heads}" \
      --argjson primary "$(jq --compact-output --arg name "${primary}" \
        'map(select(.name == $name))[0] // {} | {forge, web}' <<<"${facts}")" \
      'import "workspace" as workspace; workspace::commits($now; $branches; $unpushed; $heads; $primary)'
}

# workspace:tags prints a row per tag, the newest first, at most workspace_tags, as workspace::tags gives them.
workspace:tags() {
  git for-each-ref refs/tags --sort=-creatordate --count="${workspace_tags}" \
    --format="$(workspace:format refname:short objecttype '*objectname:short' objectname:short taggername \
      creatordate:unix contents:subject)" \
    | jq --compact-output --raw-input --slurp -L "${kit_jq}" --argjson now "$(date +%s)" \
      'import "workspace" as workspace; workspace::tags($now)'
}

# workspace:marks prints a mark per stage, as one JSON object, read from the main checkout: each the hash of what git
# says of what the stage draws, so it changes when that does; repos', the checkout's worktrees and branches.
workspace:marks() {
  (
    cd "$(git:main-root)" || exit 1
    local path
    local -a paths
    jq --null-input --compact-output \
      --arg repos "$({ pwd; git worktree list --porcelain; git for-each-ref --format='%(refname)' refs/heads; } \
        | git hash-object --stdin)" \
      --arg remotes "$({ git remote -v; git for-each-ref --format='%(refname) %(symref)' refs/remotes; } \
        | git hash-object --stdin)" \
      --arg branches "$(git for-each-ref --format='%(refname) %(objectname) %(upstream) %(worktreepath)' \
        refs/heads refs/remotes | git hash-object --stdin)" \
      --arg worktrees "$({ git worktree list --porcelain
        # The mark only has to change when the worktrees do: a list git cannot give, or a worktree removed while it
        # is read, changes what is hashed, and the stage is drawn again.
        list:read paths workspace:paths || exit 1
        for path in "${paths[@]}"
        do
          [[ ! -d "${path}" ]] || git -C "${path}" status --porcelain=v2 --branch 2>/dev/null || true
        done; } | git hash-object --stdin)" \
      --arg commits "$(git for-each-ref --format='%(refname) %(objectname)' refs/heads refs/remotes \
        | git hash-object --stdin)" \
      --arg tags "$(git for-each-ref --format='%(refname) %(objectname)' refs/tags | git hash-object --stdin)" \
      '{repos: $repos, remotes: $remotes, branches: $branches, worktrees: $worktrees, commits: $commits, tags: $tags}'
  )
}

# workspace:text <stage> <document> prints a stage's {rows, errors} as text: its rows, under each member's name where
# they have one, then each member's error, one line each.
workspace:text() {
  local stage=$1 document=$2
  jq --raw-output -L "${kit_jq}" --arg stage "${stage}" '
    import "workspace" as workspace;
    (.rows | group_by(.member // "") | .[]
     | (if .[0].member then "📚 \(.[0].member)" else empty end), workspace::text($stage)),
    (.errors[] | "❌ \(.member): \(.error)")' <<<"${document}"
}

# ----------------------------------------------------------------------------------------------------------------------
