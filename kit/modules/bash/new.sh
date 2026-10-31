# shellcheck shell=bash
# new.sh - what kit new offers: the rules with templates, and a rule's variants (sourced module)
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

# ----------------------------------------------------------------------------------------------------------------------
