# shellcheck shell=bash
# new.sh - what kit new makes a file from, and what it offers: the rules, the variants, and the holes (sourced module)
# ----------------------------------------------------------------------------------------------------------------------

use template

# new:rule-file <rule> prints the file of the rule by that name, a style of the name before a rule of it, as kit new
# reads --rule. It returns 1 when neither exists.
new:rule-file() {
  if [[ -f ".kit/styles/$1.md" ]]
  then
    printf '.kit/styles/%s.md\n' "$1"
  elif [[ -f ".kit/rules/$1.md" ]]
  then
    printf '.kit/rules/%s.md\n' "$1"
  else
    return 1
  fi
}

# new:choices <key> [<answer>…] answers kit syntax --choices for --rule, the rules that have templates, each with its
# variants as the description, and for --variant, the variants of the rule the answers name with --rule=<rule>, or
# else of the most specific rule with templates for the path they name with <path>=<path>, each with its path glob.
# Both sets are closed. With neither a rule nor a path, --variant has no list yet, and it returns 1.
new:choices() {
  local key=$1 answer rule='' path='' file name glob variants
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
    for file in .kit/styles/*.md .kit/rules/*.md
    do
      [[ -f "${file}" ]] || continue
      name=$(template:rule-name "${file}")
      [[ -z "${seen[${name}]+x}" ]] || continue
      variants=$(template:variants "${file}" | cut -d $'\x1f' -f 1 | paste -s -d ' ' -)
      [[ -n "${variants}" ]] || continue
      seen[${name}]=1
      printf '%s\ttemplates: %s\t\t\n' "${name}" "${variants}"
    done
    return 0
  fi
  if [[ -n "${rule}" ]]
  then
    file=$(new:rule-file "${rule}") || return 1
  elif [[ -n "${path}" ]]
  then
    file=$(template:rules "${path}" | head -n 1)
    [[ -n "${file}" ]] || return 1
  else
    return 1
  fi
  while IFS=$'\x1f' read -r name glob _
  do
    [[ -n "${name}" ]] || continue
    printf '%s\t%s\t\t\n' "${name}" "${glob}"
  done < <(template:variants "${file}")
}

# new:resolve <path> <rule> <variant> prints the template kit new makes the path from, as "<rule-file>\x1f<variant>\x1f
# <executable>\x1f<template>", the fields template:variants uses. <rule> names a style or rule, else the most
# specific rule with templates for the path is taken; <variant> names one of its templates, else the first that applies
# to the path. It prints the refusal and returns 2 when the rule or the variant does not exist, and 1 when the template
# is missing.
new:resolve() {
  local path=$1 rule_name=$2 variant=$3 rule candidates chosen='' executable='' name exec_flag template
  if [[ -n "${rule_name}" ]]
  then
    if ! rule=$(new:rule-file "${rule_name}")
    then
      error "there is no rule ${rule_name}; the rules are: $(find .kit/rules .kit/styles -maxdepth 1 -name '*.md' \
        2>/dev/null | sed 's@^.*/@@; s/\.md$//' | LC_ALL=C sort -u | paste -s -d ' ' -)"
      return 2
    fi
  else
    rule=$(template:rules "${path}" | head -n 1)
    if [[ -z "${rule}" ]]
    then
      error "no template governs ${path}: a style package delivers them;" \
        "name one with --rule=<rule>, or write the file by hand"
      return 2
    fi
  fi
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
      break
    fi
  done <<<"${candidates}"
  if [[ -z "${chosen}" ]]
  then
    error "the $(template:rule-name "${rule}") rule has no template ${variant:-for ${path}}; its templates are:" \
      "$(template:variants "${rule}" | cut -d $'\x1f' -f 1 | paste -s -d ' ' -)"
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
  while IFS= read -r hole
  do
    case "${hole}" in
      ('' | path | name | stem) continue
    esac
    printf -- '--%s\n' "${hole}"
  done < <(template:holes "${template}") |
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
  if [[ -z "${package}" ]]
  then
    readarray -t packages < <(find .kit/packages -mindepth 2 -maxdepth 2 -name package.yml 2>/dev/null \
      | sed 's|^\.kit/packages/||; s|/package\.yml$||' | LC_ALL=C sort)
    package=local
    (( ${#packages[@]} == 1 )) && package=${packages[0]}
  fi
  dir=.kit/packages/${package}
  file=${dir}/files/repo/.kit/kinds/${name}.yml
  if [[ -e "${file}" ]]
  then
    error "${file} is there already; kit new makes a file, it never overwrites one"
    return 1
  fi
  if [[ ! -f "${dir}/package.yml" ]]
  then
    mkdir -p "${dir}"
    printf '%s\n' "# .kit/packages/${package}/package.yml - the ${package} package: this repository's own kinds" \
      '' "description: \"This repository's own kinds, rules, and styles\"" 'depends:' '  - kit' > "${dir}/package.yml"
    step "made ${dir}/package.yml"
  fi
  mkdir -p "$(dirname "${file}")"
  {
    printf '# kit:generated -- the %s kind, from the %s package\n' "${name}" "${package}"
    printf '# see: kit package remove %s, or "Changing what kit adds" in .kit/1st.md\n' "${package}"
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
  readarray -t packages < <(release:declared-packages)
  if release:among "${package}" "${packages[@]}"
  then
    run kit hydrate >/dev/null
    step "the ${name} kind is in place; run: kit kinds ${name}"
  else
    step "the ${package} package is not active; run: kit package add ${package}"
  fi
}

# ----------------------------------------------------------------------------------------------------------------------
