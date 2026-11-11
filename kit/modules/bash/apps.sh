# shellcheck shell=bash
# apps.sh - the app verbs' runner: their stages, marks, and hand-off to kittui (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq and kit_root are set by modules/bash/1st.sh, the syntax_ arrays by guard:arguments
#
# An app verb draws the tabs of one of kittui's apps, each a stage, as kit workspace draws its own. A module of its
# own, apps/<verb>.sh, declares what is the verb's:
#
#   <verb>_stages           the stages, in the app's tab order: platform_stages=(services toolchains tools settings)
#   <verb>_elsewhere        optional, declared with declare -gA: each tab of the app that another kit command prints,
#                           to that command: ([backlog]='kit flow backlog')
#   apps/<verb>:<stage>     prints the stage's document, {rows, errors}, and limited when a capped call stopped; it
#                           returns 1, after its ❌ line, when nothing could be read
#   apps/<verb>:mark-<stage>  prints the stage's mark, a text that changes when what the stage draws does, or nothing
#                           for none; a stage without one has none, null, and is read again only when asked
#
# A row is {key, …, links}: key is the row's own in its stage, and links gives where a value leads, by its field. Over
# a workspace, each stage runs in every cloned member's checkout at once, each row gains member, its host and path, and
# so does each of its errors; a member that could not be read is {member, error} under errors.
#
# see: modules/jq/apps.jq, bin/kit-workspace, modules/bash/across.sh
use workspaces across tui forge file

# apps_forge_marks holds the forge's marks once apps:forge-mark has read them in this process, so the stages that
# share them read the forge once.
apps_forge_marks=''

# apps:main <verb> <word>… is an app verb's body, after guard:arguments: --stages prints the stages; --marks a mark per
# stage, as one JSON object, a stage with none null; <stage> with --json its document, with --jq formatted, and with
# --text, or off a terminal, as text; with no stage, every stage, as one JSON object {stages, <stage>: document…}, or
# as text under each name. On a terminal, with no form chosen, it runs kittui on the verb and the stage named. A tab
# another command prints is refused with 2, naming that command. It returns 2 outside a repository with no workspace,
# and 1 when a stage could not be read, after what was.
apps:main() {
  local verb=$1 workspace stage form command status=0 doc each first=true
  local -a named=()
  shift
  use "apps/${verb}" || return 2
  local -n apps_main_stages="${verb}_stages"
  stage=${syntax_literals[0]:-}
  if [[ -n "${syntax_options[--stages]+x}" ]]
  then
    printf '%s\n' "${apps_main_stages[@]}"
    return 0
  fi
  form=${syntax_options[--text]+--text}${syntax_options[--json]+--json}${syntax_options[--jq]+--jq}
  form+=${syntax_options[--marks]+--marks}
  # The workspace is the kit option's, kit --workspace=<workspace> <verb>, which bin/kit, the loader, hands on; outside
  # a repository, the last group used, as kit workspaces --current says; --workspace=none names none.
  workspace=$(workspaces:across "${KIT_WORKSPACE:-}") || return $?
  # On a terminal, with no form chosen, the app is kittui's, which runs this kit, the stage named maximized.
  if [[ -z "${form}" && -t 0 && -t 1 && -t 2 ]] && { [[ -n "${workspace}" ]] || workspaces:in-repository >/dev/null; } \
    && tui:binary --probe >/dev/null 2>&1
  then
    command=$(tui:binary) || return 1
    export KIT="${kit_root}/bin/kit"
    exec "${command}" "${verb}" ${workspace:+--workspace="${workspace}"} "$@"
  fi
  command=$(apps:elsewhere "${verb}" "${stage}")
  if [[ -n "${command}" ]]
  then
    error "kit ${verb} ${stage} opens the tab in kittui, on a terminal; ${stage} is printed by another command; run:" \
      "${command} --json"
    return 2
  fi
  # A bare repository, or a .git directory, has no working tree to read, so it is outside a repository too.
  if [[ -z "${workspace}" ]] && ! workspaces:in-repository >/dev/null
  then
    error "kit ${verb} reads a repository: run it in one, or name a workspace: kit --workspace=<workspace> ${verb};" \
      "see: kit workspaces"
    return 2
  fi
  # Over a workspace, each member's checkout runs the verb, and the workspace is resolved once.
  # shellcheck disable=SC2034  # across_verb and across_resolved are read by modules/bash/across.sh
  across_verb=${verb}
  if [[ -n "${workspace}" ]]
  then
    # shellcheck disable=SC2034  # read by modules/bash/across.sh
    across_resolved=$(workspaces:resolve "${workspace}") || return $?
  fi
  if [[ -n "${syntax_options[--marks]+x}" ]]
  then
    apps:marks "${verb}" "${workspace}"
    return
  fi
  if [[ -n "${stage}" ]]
  then
    doc=$(apps:document "${verb}" "${stage}" "${workspace}") || status=1
    [[ -n "${doc}" ]] || return 1
    apps:shown "${form}" "${doc}"
    return "${status}"
  fi
  # A stage that could not be read is null in the picture, its line said, and the others still draw.
  for each in "${apps_main_stages[@]}"
  do
    doc=$(apps:document "${verb}" "${each}" "${workspace}") || status=1
    if [[ "${form}" == --json || "${form}" == --jq ]]
    then
      named+=(--argjson "${each}" "${doc:-null}")
      continue
    fi
    [[ "${first}" == true ]] || printf '\n'
    first=false
    printf '%s:\n' "${each}"
    [[ -z "${doc}" ]] || apps:text "${doc}"
  done
  if [[ "${form}" == --json || "${form}" == --jq ]]
  then
    doc=$(jq --null-input --compact-output --argjson stages "$(printf '%s\n' "${apps_main_stages[@]}" \
      | workspaces:lines-json)" "${named[@]}" '{stages: $stages} + $ARGS.named') || return 1
    apps:shown "${form}" "${doc}"
  fi
  return "${status}"
}

# apps:elsewhere <verb> <stage> prints the command that prints <stage>, where it is a tab of the verb's app that
# another command prints, as <verb>_elsewhere names it, or nothing.
apps:elsewhere() {
  local verb=$1 stage=$2
  [[ -n "${stage}" ]] && declare -p "${verb}_elsewhere" >/dev/null 2>&1 || return 0
  local -n apps_elsewhere_of="${verb}_elsewhere"
  printf '%s' "${apps_elsewhere_of[${stage}]:-}"
}

# apps:document <verb> <stage> <workspace> prints the stage's document: the repository here's, or with a workspace,
# every member's, merged. It returns 1 when it could not be read, or a member could not, after what was read.
apps:document() {
  local verb=$1 stage=$2 workspace=$3
  if [[ -n "${workspace}" ]]
  then
    apps:across "${workspace}" "${stage}"
    return
  fi
  "apps/${verb}:${stage}"
}

# apps:across <workspace> <stage> runs the stage, with --json, in every cloned member's checkout at once, as
# across:start runs kit's across_verb, and prints one document of what they printed: each row and each error tagged
# with its member, and limited the least. A member whose document does not parse is {member, error}, its line, under
# errors. It returns 1, after the document, when a member did not exit 0, and what across:start returns for a
# workspace it refuses.
apps:across() {
  local workspace=$1 stage=$2 dir n=0 member doc failed=false
  local -a docs=()
  file:scratch dir
  across:start "${workspace}" "${dir}" "${stage}" --json >/dev/null || return $?
  while [[ -f "${dir}/${n}.member" ]]
  do
    member=$(<"${dir}/${n}.member")
    across:exited-0 "${dir}/${n}" || failed=true
    if [[ -s "${dir}/${n}.out" ]] && doc=$(jq --compact-output --arg member "${member}" '
      if type == "object" and (.rows | type) == "array" then
        {rows: (.rows | map(. + {member: $member})), errors: ((.errors // []) | map(. + {member: $member}))}
        + (if .limited == null then {} else {limited} end)
      else error("not a document") end' "${dir}/${n}.out" 2>/dev/null)
    then
      docs+=("${doc}")
    else
      doc=$(across:error "${dir}/${n}" "${member}" "${stage}")
      docs+=("$(jq --null-input --compact-output --argjson error "${doc}" '{rows: [], errors: [$error]}')")
      failed=true
    fi
    n=$(( n + 1 ))
  done
  apps:merge "${docs[@]}" || return 1
  [[ "${failed}" == false ]]
}

# apps:shown <form> <document> prints a document in the form asked for: JSON, formatted by jq, or text.
apps:shown() {
  case "$1" in
    (--json) printf '%s\n' "$2" ;;
    (--jq) jq . <<<"$2" ;;
    (*) apps:text "$2"
  esac
}

# apps:text <document> prints a stage's document as text: its rows, under each member's name where they have one, then
# each of its errors, one line each: a feature off here, or a member that could not be read.
apps:text() {
  jq --raw-output -L "${kit_jq}" '
    import "apps" as apps;
    (.rows | group_by(.member // "") | .[]
     | (if .[0].member then "📚 \(.[0].member)" else empty end), apps::text),
    (.errors[] | if .off then "\(.feature) is off here\(if .member then " in \(.member)" else "" end)"
                 else "❌ \(if .member then "\(.member): " else "" end)\(.error)" end),
    (if .limited then "the newest \(.limited), and no more" else empty end)' <<<"$1"
}

# apps:marks <verb> <workspace> prints a mark per stage as one JSON object: the repository here's, each a stage's
# apps/<verb>:mark-<stage>, null for none or one that is not there; or every member's, joined, as across:marks-merge
# joins them. It returns 1, after the object, when a mark could not be read, its line said.
apps:marks() {
  local verb=$1 workspace=$2 stage file unread=0 dir
  local -a args=()
  local -n apps_marks_stages="${verb}_stages"
  if [[ -n "${workspace}" ]]
  then
    file:scratch dir
    across:start "${workspace}" "${dir}" --marks >/dev/null || return $?
    across:marks-merge "${dir}"
    return
  fi
  # Each mark is read in this shell, through a file, so the forge's marks are read once for every stage.
  file:temp file mark || return 1
  for stage in "${apps_marks_stages[@]}"
  do
    : > "${file}"
    if declare -F "apps/${verb}:mark-${stage}" >/dev/null
    then
      "apps/${verb}:mark-${stage}" > "${file}" || unread=1
    fi
    args+=(--arg "${stage}" "$(<"${file}")")
  done
  jq --null-input --compact-output "${args[@]}" '$ARGS.named | map_values(if . == "" then null else . end)' || return 1
  return "${unread}"
}

# apps:forge <feature> <call> <def> [<argument>…] prints the document of forge:<call> with the arguments: each object
# it prints through apps::<def>, which gives the row, and limited from the {"limited": N} it prints last when capped.
# A primary with no forge, or a forge that lacks the feature, gives no rows and no errors; a call that says the feature
# is off here gives no rows and the error {feature, off: true}. Each is no failure. A call that fails returns 1, after
# its line, printing nothing, and so does a declaration that cannot be read. A def that is not a name, or that apps.jq
# does not have, returns 2, jq's line said.
apps:forge() {
  local feature=$1 call=$2 def=$3 out status=0
  shift 3
  if [[ ! "${def}" =~ ^[a-z][a-z0-9_]*$ ]]
  then
    error "apps:forge: '${def}' is not the name of a def in modules/jq/apps.jq"
    return 2
  fi
  # A def apps.jq does not have is jq's own line, which names it, before the forge is asked.
  if ! jq --null-input -L "${kit_jq}" "import \"apps\" as apps; if false then apps::${def} else empty end"
  then
    error "apps:forge: modules/jq/apps.jq has no def ${def}, as jq says above"
    return 2
  fi
  if forge:none
  then
    printf '{"rows":[],"errors":[]}\n'
    return 0
  fi
  forge:lacks >/dev/null || return 1
  if ! forge:has "${feature}"
  then
    printf '{"rows":[],"errors":[]}\n'
    return 0
  fi
  out=$("forge:${call}" "$@") || status=$?
  if (( status == 3 ))
  then
    jq --null-input --compact-output --arg feature "${feature}" '{rows: [], errors: [{feature: $feature, off: true}]}'
    return 0
  fi
  (( status == 0 )) || return 1
  if ! jq --slurp --compact-output -L "${kit_jq}" "import \"apps\" as apps; apps::fold(apps::${def})" <<<"${out}" \
    2>/dev/null
  then
    error "run: kit flow --capabilities (the forge's ${call} is not the JSON kit reads)"
    return 1
  fi
}

# apps:merge <document>… prints one document of several: the rows joined in order, the errors joined, and limited the
# least of those that hold one; of none, no rows and no errors.
apps:merge() {
  printf '%s\n' "$@" | jq --slurp --compact-output -L "${kit_jq}" 'import "apps" as apps; apps::merge'
}

# apps:hash <path>… prints the sha256 of what the paths hold, each that exists: a file's contents, or every file
# under a directory, by its path; or nothing when none exists, so the mark is null.
apps:hash() {
  local path any=false
  for path in "$@"
  do
    [[ ! -e "${path}" ]] || any=true
  done
  [[ "${any}" == true ]] || return 0
  for path in "$@"
  do
    if [[ -d "${path}" ]]
    then
      find "${path}" -type f | LC_ALL=C sort | while IFS= read -r each
      do
        printf '%s\0' "${each}"
        cat "${each}"
      done
    elif [[ -f "${path}" ]]
    then
      printf '%s\0' "${path}"
      cat "${path}"
    fi
  done | file:digest
}

# apps:forge-mark <kind>… prints the forge's marks of the kinds, issues, runs, releases, or milestones, apart by a
# space, or nothing when any is null, the primary has no forge, or the forge serves no marks. It returns 1, after the
# forge's line, when the marks cannot be read.
apps:forge-mark() {
  local kind
  if [[ -z "${apps_forge_marks}" ]]
  then
    if forge:none || ! forge:serves marks 2>/dev/null
    then
      apps_forge_marks='{}'
    else
      apps_forge_marks=$(forge:marks) || { apps_forge_marks=''; return 1; }
    fi
  fi
  for kind in "$@"
  do
    jq --exit-status --arg kind "${kind}" '.[$kind] | strings' <<<"${apps_forge_marks}" >/dev/null 2>&1 || return 0
  done
  jq --raw-output --args '[.[$ARGS.positional[]]] | join(" ")' "$@" <<<"${apps_forge_marks}"
}

# ----------------------------------------------------------------------------------------------------------------------
