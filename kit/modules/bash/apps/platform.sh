# shellcheck shell=bash
# platform.sh - kit platform's stages: the services, toolchains, tools, and settings kit runs with (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq and kit_root are set by modules/bash/1st.sh
#
# Each stage reads what kit itself reads where it runs, and no forge but for whether the forge's tool is signed in:
#
#   services      a row per capability kit fills: hosting, review, ci, and releases, the primary's forge; tracking,
#                 each tracker prefix; the project toolchain; each dependency provider whose manifest and lock are
#                 here; and the policy engine. Each says its service, the provider that answers it, whether kit inferred
#                 it or kit.yml set it, and whether its tool is there and signed in
#   toolchains    kit toolchains --json, a row per toolchain and role
#   tools         kit tools --json, a row per tool the chosen toolchain and the dependencies pin
#   settings      a row per key the settings schema names: the value kit.yml sets, else the schema's default
#
# services' and settings' marks are kit.yml's hash; toolchains' and tools', the hash of every file a toolchain or a
# dependency provider pins from and of the files that keep kit's choice of toolchain.
#
# see: bin/kit-platform, modules/bash/apps.sh, docs/apps.md
use apps forge tracker toolchain dependencies policy settings yaml file

# platform_stages are kit platform's stages, in Platform's tab order.
# shellcheck disable=SC2034  # read by modules/bash/apps.sh
platform_stages=(services toolchains tools settings)

# apps/platform:services prints the services document, as the file header says. A forge or a policy engine kit.yml
# names wrong is its line under errors, and the rest still draws; it returns 1 after the document then.
apps/platform:services() {
  local forge='' chosen=inferred ready='' said line feature capability prefix provider service named engine
  local tracker_ready dirs toolchain='' kept found=false
  local -a rows=() errors=() records declared
  file:temp said platform || return 1
  # The forge: none, as kit.yml or the primary says; else the provider, whose line, when it names none kit knows, is
  # an error of the stage's.
  named=$(forge:named "$(forge:primary 2>/dev/null)" 2>/dev/null) || named=''
  [[ -z "${named}" ]] || chosen='set'
  if ! forge:none
  then
    if ! forge=$(forge:provider 2>"${said}")
    then
      forge=''
      line=$(apps/platform:said "${said}")
      errors+=("$(jq --null-input --compact-output --arg error "${line}" '{error: $error}')")
    fi
  fi
  if [[ -n "${forge}" ]]
  then
    ready=false
    ! forge:require >/dev/null 2>&1 || ready=true
  fi
  for capability in hosting review ci releases
  do
    service=${forge}
    case "${capability}" in
      (ci) feature=runs ;;
      (releases) feature=releases ;;
      (*) feature=''
    esac
    [[ -z "${service}" || -z "${feature}" ]] || forge:has "${feature}" || service=''
    rows+=("$(apps/platform:row "${capability}" "${service}" "${service:+forge/${service}}" "${chosen}" \
      "${service:+${ready}}")")
  done
  # Each tracker prefix: the forge's own, unless kit.yml claims it, and each kit.yml declares.
  list:read records tracker:records || return 1
  list:read declared yaml:records "$(settings:file)" trackers prefix || return 1
  for line in "${records[@]}"
  do
    IFS=$'\x1f' read -r prefix provider <<<"${line}"
    [[ -n "${prefix}" ]] || continue
    chosen=inferred
    [[ " ${declared[*]} " != *" ${prefix} "* ]] || chosen='set'
    if [[ "${provider}" == forge ]]
    then
      rows+=("$(apps/platform:row "tracking ${prefix}" "${forge}" tracker/forge "${chosen}" "${ready}")")
      continue
    fi
    tracker_ready=false
    ! tracker:at "${prefix}" me >/dev/null 2>&1 || tracker_ready=true
    rows+=("$(apps/platform:row "tracking ${prefix}" "${provider}" "tracker/${provider}" "${chosen}" \
      "${tracker_ready}")")
  done
  # The project toolchain, as kit toolchains reports it: none where several are usable and none is chosen.
  toolchain=$(toolchain:chosen project list 2>/dev/null) || toolchain=''
  chosen=''
  ready=''
  if [[ -n "${toolchain}" ]]
  then
    chosen=inferred
    kept=$(toolchain:recorded project 2>/dev/null | head -n 1) || kept=''
    [[ -z "${KIT_TOOLCHAIN:-}" && "${kept}" != "${toolchain}" ]] || chosen='set'
    if [[ "${toolchain}" != ambient ]]
    then
      ready=false
      ! toolchain:call "${toolchain}" require >/dev/null 2>&1 || ready=true
    fi
  fi
  provider=''
  [[ -z "${toolchain}" || "${toolchain}" == ambient ]] || provider=toolchain/${toolchain}
  rows+=("$(apps/platform:row toolchain "${toolchain}" "${provider}" "${chosen}" "${ready}")")
  # Each dependency provider whose manifest and lock are here, its tool looked for with the toolchain's first on PATH.
  dirs=$(dependencies:toolchain-dirs)
  for provider in $(dependencies:providers)
  do
    dependencies:locked "${provider}" || continue
    found=true
    ready=false
    ! PATH="${dirs:+${dirs}:}${PATH}" dependencies:call "${provider}" usable >/dev/null 2>&1 || ready=true
    rows+=("$(apps/platform:row "dependencies ${provider}" "${provider}" "dependencies/${provider}" inferred \
      "${ready}")")
  done
  [[ "${found}" == true ]] || rows+=("$(apps/platform:row dependencies '' '' '' '')")
  # The policy engine: the one kit.yml names, else opa where .kit/policy holds Rego, else patterns, matched in bash.
  if engine=$(policy:engine 2>"${said}")
  then
    chosen=inferred
    [[ -z "$(settings:value policy_engine)" ]] || chosen='set'
    ready=true
    [[ "${engine}" != opa ]] || command -v opa >/dev/null 2>&1 || ready=false
    rows+=("$(apps/platform:row policy "${engine}" "policy/${engine}" "${chosen}" "${ready}")")
  else
    line=$(apps/platform:said "${said}")
    errors+=("$(jq --null-input --compact-output --arg error "${line}" '{error: $error}')")
    rows+=("$(apps/platform:row policy '' '' set '')")
  fi
  jq --null-input --compact-output \
    --argjson rows "$(printf '%s\n' "${rows[@]}" | jq --slurp --compact-output .)" \
    --argjson errors "$(printf '%s\n' "${errors[@]}" | grep -v '^$' | jq --slurp --compact-output .)" \
    '{rows: $rows, errors: $errors}' || return 1
  (( ${#errors[@]} == 0 ))
}

# apps/platform:said <file> prints the last line a read left in <file>, less its leading ❌.
apps/platform:said() {
  local line
  line=$(grep -v '^[[:space:]]*$' "$1" | tail -n 1 || true)  # grep's exit 1: the read said nothing
  printf '%s\n' "${line#❌ }"
}

# apps/platform:row <capability> <service> <provider> <chosen> <ready> prints a services row; an empty value is null.
apps/platform:row() {
  jq --null-input --compact-output --arg capability "$1" --arg service "$2" --arg provider "$3" --arg chosen "$4" \
    --arg ready "$5" '
    def some: if . == "" then null else . end;
    {key: $capability, capability: $capability, service: ($service | some), provider: ($provider | some),
     chosen: ($chosen | some), ready: (if $ready == "" then null else $ready == "true" end), links: {}}'
}

# apps/platform:toolchains prints the toolchains document, a row per line kit toolchains --json prints.
apps/platform:toolchains() { apps/platform:listed toolchains toolchain; }

# apps/platform:tools prints the tools document, a row per line kit tools --json prints. It returns 1, after its line,
# where kit tools refuses, as with no toolchain chosen among several.
apps/platform:tools() { apps/platform:listed tools tool; }

# apps/platform:listed <verb> <def> prints the document of kit <verb> --json, each object it prints through apps::<def>.
# It returns 1 when the verb fails, its line said.
apps/platform:listed() {
  local out
  out=$(run kit "$1" --json) || return 1
  jq --slurp --compact-output -L "${kit_jq}" "import \"apps\" as apps; {rows: map(apps::$2), errors: []}" <<<"${out}"
}

# apps/platform:settings prints the settings document, as apps::settings gives it from the settings schema the release
# delivers and kit.yml. It returns 1, after its line, when either cannot be read.
apps/platform:settings() {
  local schema=${kit_root}/packages/kit/files/repo/.kit/kit.schema.json settings doc=''
  settings=$(settings:file)
  if [[ ! -f "${schema}" ]]
  then
    error "reinstall kit: kit tools install (${schema} is missing, so the settings cannot be listed)"
    return 1
  fi
  if [[ -f "${settings}" ]]
  then
    doc=$(yaml:json "${settings}") || return 1
  fi
  jq --compact-output -L "${kit_jq}" --slurpfile schema "${schema}" \
    'import "apps" as apps; {rows: apps::settings($schema[0]), errors: []}' <<<"${doc:-null}"
}

apps/platform:mark-services() { apps:hash "$(settings:file)"; }

apps/platform:mark-settings() { apps:hash "$(settings:file)"; }

apps/platform:mark-toolchains() {
  local -a paths
  list:read paths apps/platform:pins || return 1
  apps:hash "${paths[@]}"
}

apps/platform:mark-tools() { apps/platform:mark-toolchains; }

# apps/platform:pins prints every file a toolchain or a dependency provider pins from, one per line, and the files that
# keep kit's choice of toolchain: this checkout's, the clone's, and the person's.
apps/platform:pins() {
  local provider kind name row common
  local -a inputs
  for provider in $(toolchain:providers)
  do
    list:read inputs toolchain:call "${provider}" inputs || return 1
    for row in "${inputs[@]}"
    do
      IFS=$'\t' read -r kind name <<<"${row}"
      if [[ "${kind}" == file ]]
      then
        printf '%s\n' "${name}"
      fi
    done
  done
  for provider in $(dependencies:providers)
  do
    dependencies:call "${provider}" file
    dependencies:call "${provider}" lock
  done
  toolchain:kept 2>/dev/null || true  # outside a git repository, no choice is kept for the project
  if common=$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null)
  then
    printf '%s/kit/toolchain\n' "${common}"
  fi
  toolchain:settings
}

# ----------------------------------------------------------------------------------------------------------------------
