# shellcheck shell=bash
# new.sh - what kit new makes: a file from a template, a kind, rule, style, package, module, or verb (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root, kit_bash_path, and help_groups are set by modules/bash/1st.sh and help.sh

use template file settings

# new:choices <key> [<answer>…] answers kit syntax --choices for --rule, the rules that have templates, each with its
# variants as the description, and for --variant, the variants of the rule and the style the answers name with
# --rule=<rule>, or else of every rule with templates for the path they name with <path>=<path>, each with its glob.
# Both sets are closed. With neither a rule nor a path, --variant has no list yet, and it returns 1.
new:choices() {
  local key=$1 answer rule='' path='' file name glob variants row
  local -a rows
  local -A seen=()
  shift
  case "${key}" in
    (--rule | --variant) ;;
    (*) return 1
  esac
  if [[ "${1:-}" == --kind ]]
  then
    printf 'closed\n'
    return 0
  fi
  for answer in "$@"
  do
    case "${answer}" in
      (--rule=*) rule=${answer#--rule=} ;;
      ('<path>='*) path=${answer#<path>=}
    esac
  done
  if [[ "${key}" == --rule ]]
  then
    # A rule and a style of one name share it, and their templates the directory; both lists are its templates.
    for name in $(find .kit/styles .kit/rules -maxdepth 1 -name '*.md' 2>/dev/null | sed 's@^.*/@@; s/\.md$//' \
      | LC_ALL=C sort -u)
    do
      variants=$(for file in ".kit/styles/${name}.md" ".kit/rules/${name}.md"
        do
          [[ -f "${file}" ]] && template:variants "${file}" | cut -d $'\x1f' -f 1
        done | paste -s -d ' ' -)
      [[ -n "${variants}" ]] && printf '%s\ttemplates: %s\t\t\n' "${name}" "${variants}"
    done
    return 0
  fi
  # The variants are those of every rule and style named, or that governs the path, as kit new tries them in turn.
  local -a files
  if [[ -n "${rule}" ]]
  then
    readarray -t files < <(for file in ".kit/styles/${rule}.md" ".kit/rules/${rule}.md"
      do
        [[ -f "${file}" ]] && printf '%s\n' "${file}"
      done)
  elif [[ -n "${path}" ]]
  then
    list:read files template:rules "${path}" || return 1
  fi
  (( ${#files[@]} )) || return 1
  for file in "${files[@]}"
  do
    list:read rows template:variants "${file}" || return 1
    for row in "${rows[@]}"
    do
      IFS=$'\x1f' read -r name glob _ <<<"${row}"
      [[ -n "${name}" && -z "${seen[${name}]+x}" ]] || continue
      seen[${name}]=1
      printf '%s\t%s\t\t\n' "${name}" "${glob}"
    done
  done
}

# new:resolve <path> <rule> <variant> prints the template kit new makes the path from, as "<rule-file>\x1f<variant>\x1f
# <executable>\x1f<template>", the fields template:variants uses. <rule> names a style or rule, else the most
# specific rule with templates for the path is taken; <variant> names one of its templates, else the first that applies
# to the path. It prints the refusal and returns 2 when the rule or the variant does not exist, and 1 when the template
# is missing.
new:resolve() {
  local path=$1 rule_name=$2 variant=$3 rule file candidates chosen='' executable='' name exec_flag template
  local -a rules
  if [[ -n "${rule_name}" ]]
  then
    readarray -t rules < <(for file in ".kit/styles/${rule_name}.md" ".kit/rules/${rule_name}.md"
      do
        [[ -f "${file}" ]] && printf '%s\n' "${file}"
      done)
    if (( ${#rules[@]} == 0 ))
    then
      error "there is no rule ${rule_name}; the rules are: $(find .kit/rules .kit/styles -maxdepth 1 -name '*.md' \
        2>/dev/null | sed 's@^.*/@@; s/\.md$//' | LC_ALL=C sort -u | paste -s -d ' ' -)"
      return 2
    fi
  else
    list:read rules template:rules "${path}" || return 1
    if (( ${#rules[@]} == 0 ))
    then
      error "no template governs ${path}: a style package delivers them;" \
        "name one with --rule=<rule>, or write the file by hand"
      return 2
    fi
  fi
  # The first rule, most specific first, with a template the path allows, or with the variant named, gives it.
  for rule in "${rules[@]}"
  do
    if [[ -n "${variant}" ]]
    then
      candidates=$(template:variants "${rule}")
    else
      candidates=$(template:applies "${rule}" "${path}")
    fi
    while IFS=$'\x1f' read -r name _ exec_flag
    do
      [[ -n "${name}" ]] || continue
      if [[ -z "${variant}" || "${name}" == "${variant}" ]]
      then
        chosen=${name} executable=${exec_flag:-false}
        break 2
      fi
    done <<<"${candidates}"
  done
  if [[ -z "${chosen}" && -z "${rule_name}" && -z "${variant}" ]]
  then
    error "no template governs ${path}: a style package delivers them;" \
      "name one with --rule=<rule>, or write the file by hand"
    return 2
  fi
  if [[ -z "${chosen}" ]]
  then
    rule=${rules[0]}
    error "the $(template:rule-name "${rule}") rule has no template ${variant:-for ${path}}; its templates are:" \
      "$(for file in "${rules[@]}"; do template:variants "${file}" | cut -d $'\x1f' -f 1; done | paste -s -d ' ' -)"
    return 2
  fi
  template=$(template:file "${rule}" "${chosen}")
  if [[ ! -f "${template}" ]]
  then
    error "${rule} lists the template ${chosen}, and ${template} is not here; run: kit update"
    return 1
  fi
  printf '%s\x1f%s\x1f%s\x1f%s\n' "${rule}" "${chosen}" "${executable}" "${template}"
}

# new:extend [<answer>…] answers kit syntax --extend: once the answers give the path (<path>=<path>) or the rule
# (--rule=<rule>), and the variant (--variant=<variant>) when the first that applies is not meant, the template's holes
# within a line, other than path, name, and stem, which the path fills, as arguments to add: {"--<hole>":
# {"description": "", "extended": true}}. Before the template is known, it prints {}.
new:extend() {
  local answer path='' rule='' variant='' resolved template hole
  local -a holes
  for answer in "$@"
  do
    case "${answer}" in
      ('<path>='*) path=${answer#<path>=} ;;
      (--rule=*) rule=${answer#--rule=} ;;
      (--variant=*) variant=${answer#--variant=}
    esac
  done
  if ! resolved=$(new:resolve "${path}" "${rule}" "${variant}" 2>/dev/null)
  then
    printf '{}\n'
    return 0
  fi
  template=${resolved##*$'\x1f'}
  list:read holes template:holes "${template}" || return 1
  for hole in "${holes[@]}"
  do
    case "${hole}" in
      ('' | path | name | stem) continue
    esac
    printf -- '--%s\n' "${hole}"
  done |
    jq --raw-input --slurp 'split("\n") | map(select(. != "") | {key: ., value: {description: "", extended: true}})
      | from_entries'
}

# new:kind <name> [--path=<glob>…] [--content=<regex>…] [--kinds=<kind>…] [--embeds=<kind>…] [--package=<name>]
# writes a kind into a package of the repository's own, makes the package when there is none, and places the kind
# when the package is active, or names the command that activates it. It refuses a name already defined, a kind
# with nothing to match a path by, and a value that holds a double quote, which the kinds file could not hold.
new:kind() {
  local name='' package='' arg key value file dir
  local -a packages
  local -A lists=([paths]='' [content]='' [kinds]='' [embeds]='')
  for arg in "$@"
  do
    case "${arg}" in
      (--package=?*) package=${arg#--package=} ;;
      (--path=?* | --content=?* | --kinds=?* | --embeds=?*)
        key=${arg%%=*}
        key=${key#--}
        [[ "${key}" == path ]] && key=paths
        value=${arg#*=}
        if [[ "${value}" == *'"'* ]]
        then
          error "--${key%s}=${value}: a kind's value cannot hold a double quote"
          return 2
        fi
        lists[${key}]+=${value}$'\n'
        ;;
      (*) name=${arg}
    esac
  done
  if [[ ! "${name}" =~ ^[a-z][a-z0-9-]*$ ]]
  then
    error "a kind's name is a lowercase letter, then lowercase letters, digits, and dashes, not '${name}'"
    return 2
  fi
  if [[ -z "${lists[paths]}${lists[content]}${lists[kinds]}" ]]
  then
    error "kit new kind ${name} needs --path, --content, or --kinds, to say what makes a path of the kind"
    return 2
  fi
  use context
  context:load || return 1
  if [[ -n "${context_kind_paths[${name}]+x}" ]]
  then
    error "the kind ${name} is defined already; run: kit kinds ${name}"
    return 1
  fi
  package=$(new:package-for "${package}")
  file=.kit/packages/${package}/files/repo/.kit/kinds/${name}.yml
  new:clear "${file}" || return 1
  new:ensure-package "${package}"
  mkdir -p "$(dirname "${file}")"
  {
    printf '# kit:generated -- the %s kind, from the %s package\n' "${name}" "${package}"
    printf '# see: kit packages remove %s, or "Changing what kit adds" in .kit/1st.md\n' "${package}"
    printf 'name: %s\n' "${name}"
    for key in paths content kinds embeds
    do
      [[ -n "${lists[${key}]}" ]] || continue
      printf '%s:\n' "${key}"
      while IFS= read -r value
      do
        [[ -n "${value}" ]] || continue
        if [[ "${key}" == paths || "${key}" == content ]]
        then
          printf '  - "%s"\n' "${value}"
        else
          printf '  - %s\n' "${value}"
        fi
      done <<<"${lists[${key}]}"
    done
  } > "${file}"
  ok "made ${file}"
  new:offer "${package}" "the ${name} kind" "kit kinds ${name}"
}

# new:package-for [<package>] prints the package a made thing goes into: the one named, else the repository's one
# package under .kit/packages/, else local.
new:package-for() {
  local -a packages
  if [[ -n "$1" ]]
  then
    printf '%s\n' "$1"
    return 0
  fi
  readarray -t packages < <(find .kit/packages -mindepth 2 -maxdepth 3 -name package.yml 2>/dev/null \
    | sed 's|^\.kit/packages/||; s|/package\.yml$||' | LC_ALL=C sort)
  if (( ${#packages[@]} == 1 ))
  then
    printf '%s\n' "${packages[0]}"
  else
    printf 'local\n'
  fi
}

# new:clear <path> is true when nothing is at <path>; otherwise it refuses, since kit new never overwrites a file.
new:clear() {
  if [[ -e "$1" ]]
  then
    error "$1 is there already; kit new makes a file, it never overwrites one"
    return 1
  fi
}

# new:ensure-package <package> [<description>] makes .kit/packages/<package>/package.yml, depending on kit, when the
# package has none.
new:ensure-package() {
  local dir=.kit/packages/$1 description="This repository's own kinds, rules, and styles"
  [[ -f "${dir}/package.yml" ]] && return 0
  [[ -z "${2:-}" ]] || description=$2
  mkdir -p "${dir}"
  printf '%s\n' "# .kit/packages/$1/package.yml - the $1 package, of this repository's own" '' \
    "description: \"${description}\"" 'depends:' '  - kit' > "${dir}/package.yml"
  step "made ${dir}/package.yml"
}

# new:offer <package> <what> <command> says what to run next: the command that shows <what> when the package is
# active, which kit hydrate then places, or the command that activates it.
new:offer() {
  local -a packages
  list:read packages settings:packages || return 1
  if list:among "$1" "${packages[@]}"
  then
    run kit hydrate >/dev/null
    step "$2 is in place; run: $3"
  else
    step "the $1 package is not active; run: kit packages add $1"
  fi
}

# new:frontmatter <package> <what> <kind> [<lint>] prints the frontmatter of a rule or a style of the package that
# governs <kind>: the kit:generated and see: lines, kind:, and lint: when there is a linter.
new:frontmatter() {
  printf -- '---\n# kit:generated -- %s, from the %s package\n' "$2" "$1"
  printf '# see: kit packages remove %s, or "Changing what kit adds" in .kit/1st.md\n' "$1"
  printf 'kind:\n  - %s\n' "$3"
  [[ -z "${4:-}" ]] || printf 'lint: %s\n' "$4"
  printf -- '---\n'
}

# new:rule <rule|style> <kind> [--lint=<command>] [--package=<name>] writes a rule or a style for <kind> into a
# package of the repository's own, with the frontmatter that names the kind and its linter, and a heading and a first
# line to write under. It warns when no kind of that name is defined yet.
new:rule() {
  local which=$1 kind=$2 lint='' package='' arg file title
  shift 2
  for arg in "$@"
  do
    case "${arg}" in
      (--lint=?*) lint=${arg#--lint=} ;;
      (--package=?*) package=${arg#--package=}
    esac
  done
  new:name "${kind}" kind || return 2
  package=$(new:package-for "${package}")
  file=.kit/packages/${package}/files/repo/.kit/${which}s/${kind}.md
  new:clear "${file}" || return 1
  use context
  context:load || return 1
  [[ -n "${context_kind_paths[${kind}]+x}" ]] || warn "no kind ${kind} is defined yet; run: kit new kind ${kind}"
  new:ensure-package "${package}"
  mkdir -p "$(dirname "${file}")"
  title="${kind^}"
  [[ "${which}" == style ]] && title+=' style'
  {
    new:frontmatter "${package}" "the ${kind} ${which}" "${kind}" "${lint}"
    # shellcheck disable=SC2016  # the backquotes are markdown's code spans, printed as they are
    printf '# %s\n\nApplies to files of kind `%s`; `kit kinds %s` says which they are.\n' "${title}" "${kind}" "${kind}"
  } > "${file}"
  ok "made ${file}"
  new:offer "${package}" "the ${kind} ${which}" "kit context <path>"
}

# new:package <name> [--description=<text>] [--depends=<package>…] writes .kit/packages/<name>/package.yml, depending
# on kit and the packages named.
new:package() {
  local name=$1 description="This repository's own $1 package" arg file
  local -a depends=(kit)
  shift
  for arg in "$@"
  do
    case "${arg}" in
      (--description=?*) description=${arg#--description=} ;;
      (--depends=?*) depends+=("${arg#--depends=}")
    esac
  done
  if [[ ! "${name}" =~ ^[a-z][a-z0-9-]*(/[a-z][a-z0-9-]*)?$ ]]
  then
    error "a package's name is lowercase words with dashes, one / at most, as style/bash, not '${name}'"
    return 2
  fi
  if [[ "${description}" == *'"'* ]]
  then
    error "--description cannot hold a double quote"
    return 2
  fi
  file=.kit/packages/${name}/package.yml
  new:clear "${file}" || return 1
  mkdir -p "$(dirname "${file}")"
  {
    printf '# .kit/packages/%s/package.yml - the %s package, of this repository'"'"'s own\n\n' "${name}" "${name}"
    printf 'description: "%s"\ndepends:\n' "${description}"
    printf '  - %s\n' "${depends[@]}"
  } > "${file}"
  ok "made ${file}; put what it delivers under .kit/packages/${name}/files/repo/"
  step "run: kit packages add ${name}"
}

# new:template <rule> <variant> [--from=<path>] [--package=<name>] writes a template, the file at <path> or a single
# body hole, into a package of the repository's own, and lists the variant under templates: in that package's rule or
# style for <rule>, when it has one; otherwise it says where to list it.
new:template() {
  local rule=$1 variant=$2 from='' package='' arg file owner
  shift 2
  for arg in "$@"
  do
    case "${arg}" in
      (--from=?*) from=${arg#--from=} ;;
      (--package=?*) package=${arg#--package=}
    esac
  done
  new:name "${rule}" rule || return 2
  new:name "${variant}" variant || return 2
  if [[ -n "${from}" && ! -f "${from}" ]]
  then
    error "--from=${from}: no such file"
    return 2
  fi
  package=$(new:package-for "${package}")
  file=.kit/packages/${package}/files/repo/.kit/templates/${rule}/${variant}
  new:clear "${file}" || return 1
  new:ensure-package "${package}"
  mkdir -p "$(dirname "${file}")"
  if [[ -n "${from}" ]]
  then
    cp "${from}" "${file}"
  else
    printf '«body»\n' > "${file}"
  fi
  ok "made ${file}; write its holes as «name»"
  for owner in .kit/packages/"${package}"/files/repo/.kit/styles/"${rule}".md \
    .kit/packages/"${package}"/files/repo/.kit/rules/"${rule}".md
  do
    [[ -f "${owner}" ]] || continue
    # shellcheck disable=SC2016  # the awk program names its fields with $ on purpose
    file:replace "${owner}" awk -v variant="${variant}" '
      NR == 1 && $0 == "---" { front = 1; print; next }
      front && $0 == "templates:" { listed = 1; print; print "  - variant: " variant; next }
      front && $0 == "---" { if (!listed) { print "templates:"; print "  - variant: " variant }; front = 0 }
      { print }' "${owner}"
    step "listed the ${variant} variant under templates: in ${owner}"
    new:offer "${package}" "the ${variant} template" "kit new file <path> --rule=${rule} --variant=${variant}"
    return 0
  done
  warn "list it under templates: in a rule or style named ${rule}; run: kit new style ${rule} --package=${package}"
}

# new:module-dir prints the directory a new bash module goes in: kit/modules/bash/ in the kit's own source, else the
# first directory of the repository's own on the module path. It refuses when there is none.
new:module-dir() {
  local dir root
  root=$(pwd -P)
  if new:own-source
  then
    printf 'kit/modules/bash\n'
    return 0
  fi
  use:load-path || return 1
  for dir in "${kit_bash_path[@]}"
  do
    if [[ "${dir}" == "${root}"/* ]]
    then
      printf '%s\n' "${dir#"${root}"/}"
      return 0
    fi
  done
  error "the module path has no directory of this repository's own; add one under bash_path in kit.yml," \
    "such as bash/"
  return 1
}

# new:own-source is true when the repository is the kit's own source: its kit/ is the release that runs.
new:own-source() {
  [[ -d kit && "$(cd kit && pwd -P)" == "${kit_root}" ]]
}

# new:module <name> --purpose=<text> [--jq] writes a bash module into the repository's own directory on the module
# path, from the bash style's module template where the repository has one. With --jq it writes a jq module and its
# tests, which the kit's own source alone has.
new:module() {
  local name='' purpose='' jq=false arg file test
  for arg in "$@"
  do
    case "${arg}" in
      (--purpose=?*) purpose=${arg#--purpose=} ;;
      (--jq) jq=true ;;
      (*) name=${arg}
    esac
  done
  if [[ ! "${name}" =~ ^[a-z][a-z0-9-]*(/[a-z][a-z0-9-]*)*$ ]]
  then
    error "a module's name is lowercase words with dashes, under directories with /, as toolchain/brew, not '${name}'"
    return 2
  fi
  if [[ "${jq}" == true ]]
  then
    if ! new:own-source
    then
      error "a jq module comes from the release alone; kit new module --jq makes one in the kit's own source"
      return 1
    fi
    file=kit/modules/jq/${name}.jq test=tests/jq/${name}.test
    new:clear "${file}" && new:clear "${test}" || return 1
    mkdir -p "$(dirname "${file}")" "$(dirname "${test}")"
    printf '%s\n' "# ${name}.jq - ${purpose} (jq module)" '#' \
      "# import \"${name}\" as ${name}; with jq -L \"\${kit_jq}\"." '' \
      '# identity gives its input unchanged; replace it with the first def the module needs.' \
      'def identity: .;' > "${file}"
    printf '%s\n' "# ${name}.test - the ${name} jq module, run by jq -L kit/modules/jq --run-tests" '#' \
      '# Each case is a program, an input, and the outputs it gives, one JSON value per line, then a blank line.' '' \
      '# identity gives its input unchanged.' "import \"${name}\" as ${name}; ${name}::identity" '{"a": 1}' \
      '{"a":1}' > "${test}"
    ok "made ${file} and ${test}"
    return 0
  fi
  file=$(new:module-dir)/${name}.sh || return 1
  new:bash-file "${file}" module "${purpose} (sourced module)" ''
}

# new:bash-file <path> <variant> <purpose> <body> makes a bash file through kit new file, from the bash style's
# template of that variant, with the purpose and body filled in; where the repository has no such template, it writes
# the banner and the body alone.
new:bash-file() {
  local file=$1 variant=$2 purpose=$3 body=$4
  new:clear "${file}" || return 1
  if [[ -f ".kit/templates/bash/${variant}" ]]
  then
    run kit new file "${file}" --rule=bash --variant="${variant}" --purpose="${purpose}" --body="${body}"
    return
  fi
  mkdir -p "$(dirname "${file}")"
  if [[ "${variant}" == module ]]
  then
    printf '# shellcheck shell=bash\n# %s - %s\n\n%s\n' "${file##*/}" "${purpose}" "${body}" > "${file}"
  else
    printf '#!/usr/bin/env bash\n# %s - %s\n\nset -o errexit -o nounset -o pipefail\n%s\n' "${file##*/}" "${purpose}" \
      "${body}" > "${file}"
    chmod +x "${file}"
  fi
  ok "made ${file}"
}

# new:seam-calls <seam> prints the calls a seam makes of its providers, once each: the functions every provider of the
# seam on the module path defines, so a helper one provider keeps to itself is left out.
new:seam-calls() {
  local provider file call count=0
  local -a calls
  local -A defined=()
  while IFS= read -r provider
  do
    count=$(( count + 1 ))
    readarray -t calls < <(while IFS= read -r file
      do
        sed -n -E "s|^${provider}:([a-z][a-z-]*)\(\).*|\1|p" "${file}"
      done < <(use:find "${provider}") | LC_ALL=C sort --unique)
    for call in "${calls[@]}"
    do
      [[ -n "${call}" ]] && defined[${call}]=$(( ${defined[${call}]:-0} + 1 ))
    done
  done < <(use:modules "$1/*")
  for call in "${!defined[@]}"
  do
    (( defined[${call}] == count )) && printf '%s\n' "${call}"
  done | LC_ALL=C sort
}

# new:provider <seam> <name> writes a provider of the seam into the repository's own directory on the module path:
# <seam>/<name>.sh, with one function for each call the seam makes, each refusing until it is written.
new:provider() {
  local seam=$1 name=$2 call body='' module
  local -a calls
  new:name "${seam}" seam || return 2
  new:name "${name}" provider || return 2
  use:load-path || return 1
  if [[ -z "$(use:find "${seam}")" ]]
  then
    error "no seam ${seam} on the module path; the seams are dependencies, forge, policy, toolchain, and tracker"
    return 2
  fi
  readarray -t calls < <(new:seam-calls "${seam}")
  module=${seam}/${name}
  for call in "${calls[@]}"
  do
    body+="# ${module}:${call} answers ${seam}:${call}. It refuses until it is written."$'\n'
    body+="${module}:${call}() {"$'\n'"  error \"${module}: ${call} is not written yet\""$'\n'"  return 1"$'\n'"}"$'\n\n'
  done
  body+="# The seam's calls end here; what follows is this provider's own."
  new:bash-file "$(new:module-dir)/${module}.sh" module "the ${name} provider of the ${seam} seam (sourced module)" \
    "${body}"
}

# new:issue [<prefix>] [--print] opens the page a person files a new issue at, in the tracker the prefix names, the
# forge's own, #, by default, and prints its URL; --print prints the URL alone. new:issue --list [--json] lists every
# tracker instead, as new:trackers says. A title is not taken here, as kit new file takes --title for a template's
# hole; the seam's new-url takes one.
new:issue() {
  local prefix='#' print=false list=false json=false arg url
  for arg in "$@"
  do
    case "${arg}" in
      (--print) print=true ;;
      (--list) list=true ;;
      (--json) json=true ;;
      (*) prefix=${arg}
    esac
  done
  use tracker
  if [[ "${list}" == true ]]
  then
    new:trackers "${json}"
    return
  fi
  tracker:provider "${prefix}" >/dev/null || return 2
  url=$(tracker:at "${prefix}" new-url) || return 1
  printf '%s\n' "${url}"
  [[ "${print}" == true ]] || new:browse "${url}"
}

# new:trackers <json> lists every tracker a new issue can be filed in, the forge's own first: its prefix, a label
# that says where, and the page new:issue opens, a line each, tab-separated; with json true, an array of {prefix,
# provider, label, url, error}, url null and error the tracker's refusal where it gives no page.
new:trackers() {
  local prefix provider url error label out='[]' refusal row text
  local -a rows
  use file forge
  file:temp refusal new-issue || return 1
  list:read rows tracker:records || return 1
  printf -v text '%s\n' "${rows[@]}"
  list:read rows new:forge-first <<<"${text}" || return 1
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r prefix provider <<<"${row}"
    [[ -n "${prefix}" ]] || continue
    error='' label=${provider}
    url=$(NO_COLOR=1 tracker:at "${prefix}" new-url 2>"${refusal}") || url=''
    [[ -n "${url}" ]] || error=$(tail -n 1 "${refusal}" | sed 's/^❌ //')
    case "${provider}" in
      (forge)
        label="$(forge:provider 2>/dev/null) $(sed -E 's|^[a-z]+://[^/]+/||; s|(/-)?/issues/new.*$||' <<<"${url}")" ;;
      (ado) label="ado $(tracker:field "${prefix}" project)" ;;
      (*) label="${provider} $(sed -E 's|^[a-z]+://([^/]+).*$|\1|' <<<"${url}")"
    esac
    out=$(jq --compact-output --arg prefix "${prefix}" --arg provider "${provider}" --arg label "${label% }" \
      --arg url "${url}" --arg error "${error}" \
      '. + [{prefix: $prefix, provider: $provider, label: $label, url: ($url | select(. != "") // null),
        error: ($error | select(. != "") // null)}]' <<<"${out}")
  done
  if [[ "$1" == true ]]
  then
    jq . <<<"${out}"
  else
    jq --raw-output '.[] | [.prefix, .label, (.url // .error)] | @tsv' <<<"${out}"
  fi
}

# new:forge-first passes the tracker records through, the forge's own first.
new:forge-first() {
  local records
  records=$(cat)
  # grep's exit 1 is a list with no record of that kind.
  grep "$(printf '\x1fforge$')" <<<"${records}" || true
  grep --invert-match "$(printf '\x1fforge$')" <<<"${records}" || true  # as above
}

# new:browse <url> opens the page in the browser: open on macOS, xdg-open elsewhere; without either, it warns, and the
# URL printed is the way there.
new:browse() {
  local opener=xdg-open
  [[ "$(uname -s)" == Darwin ]] && opener=open
  if ! command -v "${opener}" >/dev/null
  then
    warn "no ${opener} to open the page with; open the URL above"
    return 0
  fi
  "${opener}" "$1" >/dev/null 2>&1 || warn "${opener} could not open the page; open the URL above"
}

# new:verb <name> --purpose=<text> --group=<group> writes a verb into the kit's own source: kit/bin/kit-<name>, with
# the header kit help reads, the loader, and the guard on its arguments.
new:verb() {
  local name=$1 purpose='' group='' arg body
  shift
  for arg in "$@"
  do
    case "${arg}" in
      (--purpose=?*) purpose=${arg#--purpose=} ;;
      (--group=?*) group=${arg#--group=}
    esac
  done
  if ! new:own-source
  then
    error "a verb is part of the kit's own source; kit new verb makes one there"
    return 1
  fi
  new:name "${name}" verb || return 2
  use help
  if [[ " ${help_groups[*]} " != *" ${group} "* ]]
  then
    error "--group=${group}: a verb's group is one of ${help_groups[*]}"
    return 2
  fi
  body=$'# shellcheck source=modules/bash/1st.sh\nsource "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")/../modules/bash/1st.sh"'
  body+=$'\nkit:enter\nguard:arguments '"${name}"' "$@" || exit $?'
  new:clear "kit/bin/kit-${name}" || return 1
  run kit new file "kit/bin/kit-${name}" --rule=bash --variant=script --purpose="${purpose} (kit verb: ${group})" \
    --about=$'#\n# kit '"${name}"$'\n#\n# kit '"${name}"' '"${purpose}"$'.\n#\n# Exits 0 on success, and 2 on a usage error.\n' \
    --body="${body}"
}

# new:name <word> <what> is true when <word> is a lowercase word with dashes; otherwise it refuses, naming <what>.
new:name() {
  [[ "$1" =~ ^[a-z][a-z0-9-]*$ ]] && return 0
  error "a $2's name is a lowercase letter, then lowercase letters, digits, and dashes, not '$1'"
  return 1
}

# ----------------------------------------------------------------------------------------------------------------------
