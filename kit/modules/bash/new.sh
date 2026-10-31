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

# ----------------------------------------------------------------------------------------------------------------------
