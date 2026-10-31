# shellcheck shell=bash
# template.sh - a rule's templates: make a new file from one, and check a file still matches one (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
#
# A rule can have templates, one per variant of the file it governs, at .kit/templates/<rule>/<variant>. The rule's
# frontmatter lists them in the order they are tried. Each entry can limit its template to a path glob, and can
# mark the file as executable:
#
#   templates:
#     - variant: root
#       path: justfile
#     - variant: module
#
# A template is the file itself, with holes written «name», in the language modules/jq/template.jq describes:
#
#   «name»                 within a line: making a file fills it with a value; checking one matches any text there
#   «name…»                the same, across lines
#   «name» on its own line making a file fills it with the value's lines, else one blank line; checking one matches
#                          any run of whole lines
#   «name:-default»        any of these, with the text it makes when no value is given
#
# path, name, and stem are bound to the file's path, so a file whose banner names another file does not match. A
# name that appears more than once takes one value per appearance. A region in a template, between a kit:generate
# directive and its kit:end line, matches whatever its command wrote. Two holes with nothing between them are refused.
#
# The same template both makes a file (kit new file) and checks one (kit lint). A file passes when it matches any
# variant its path allows. To change a template, a repository adds a package of its own that places the same path.
# The later package in dependency order wins.
#
# see: docs/templates.md

use context yaml

# template_dir is the directory the templates are installed in.
declare -gr template_dir=.kit/templates

# template:exempt <path> is true for a path that no template checks:
#
#   - a template itself
#   - a package's region, which is a fragment of a file
#   - a path .gitattributes marks linguist-generated, because a tool writes it
template:exempt() {
  [[ "$1" =~ (^|/)\.kit/templates/ || "$1" =~ (^|/)packages/[^/]+(/[^/]+)?/regions/ ]] && return 0
  [[ "$(git check-attr linguist-generated -- "$1" 2>/dev/null)" == *': linguist-generated: set' ]]
}

# template:rule-name <rule-file> prints the rule's name: its file's name without .md.
template:rule-name() {
  local name=${1##*/}
  printf '%s\n' "${name%.md}"
}

# template:variants <rule-file> prints "<variant>\x1f<path-glob>\x1f<executable>" for each template the rule lists,
# in order.
template:variants() {
  yaml:records "$1" templates variant path executable
}

# template:file <rule-file> <variant> prints the template's path.
template:file() {
  printf '%s/%s/%s\n' "${template_dir}" "$(template:rule-name "$1")" "$2"
}

# template:applies <rule-file> <path> prints the variants whose path glob matches <path>, or that have no glob. It
# prints them in order, in the same format as template:variants.
template:applies() {
  local variant glob executable
  while IFS=$'\x1f' read -r variant glob executable
  do
    [[ -n "${variant}" ]] || continue
    if [[ -z "${glob}" ]] || context:glob-matches "$2" "${glob}"
    then
      printf '%s\x1f%s\x1f%s\n' "${variant}" "${glob}" "${executable}"
    fi
  done < <(template:variants "$1")
}

# template:unapply <template> <file> prints, as one JSON object, the values the file gives the template's holes, path,
# name, and stem bound to the file's path. It returns 1 when the file does not have the template's shape.
template:unapply() {
  local name=${2##*/}
  jq -L "${kit_jq}" --null-input --exit-status --rawfile template "$1" --rawfile text "$2" \
    --arg path "$2" --arg name "${name}" --arg stem "${name%.*}" \
    'import "template" as template; $template | template::unapply({path: $path, name: $name, stem: $stem}; $text)'
}

# template:ambiguous <template> prints the line of each hole that follows another hole with nothing between them.
template:ambiguous() {
  jq -L "${kit_jq}" --raw-input --slurp --raw-output 'import "template" as template; template::parts | template::ambiguous' \
    < "$1"
}

# template:holes <template> prints the holes a caller must fill to make a file, once each, in order: those within a
# line, or across lines, that have no default.
template:holes() {
  jq -L "${kit_jq}" --raw-input --slurp --raw-output 'import "template" as template; template::parts | template::holes[]' \
    < "$1"
}

# template:render <template> <values-json> prints the file the template makes with the values object: each hole takes
# its value, else its default; else a hole within a line stays a hole, and a hole that is a line gives one blank line,
# or nothing as the template's last line. A region gives the template's own lines, for kit generate to fill.
template:render() {
  jq -L "${kit_jq}" --raw-input --slurp --join-output --argjson v "$2" \
    'import "template" as template; template::parts | template::render($v)' < "$1"
}

# template:rules <path> prints the rules of the path's own kinds that have templates, most specific first. The most
# specific rule names the kind whose matching glob has the most non-wildcard characters. A rule whose kind matched
# only by the file's first line sorts last.
template:rules() {
  local path=$1 rule glob best literal kind
  context:load
  while IFS= read -r rule
  do
    [[ -n "${rule}" && -n "$(template:variants "${rule}")" ]] || continue
    best=0
    for kind in ${context_file_kinds[${rule}]-}
    do
      while IFS= read -r glob
      do
        if [[ -n "${glob}" ]] && context:glob-matches "${path}" "${glob}"
        then
          literal=${glob//\*/}
          (( ${#literal} > best )) && best=${#literal}
        fi
      done <<<"${context_kind_paths[${kind}]-}"
    done
    printf '%s\t%s\n' "${best}" "${rule}"
  done < <(context:rules "${path}" --direct) | LC_ALL=C sort -t $'\t' -k1,1nr -k2,2 | cut -f2
}

# template:check <rule-file> <path>… is true when every file matches a variant the rule allows at its path. For
# each file that matches none, it prints an error naming the variants it was checked against, and the line where the
# file leaves the variant it follows furthest. A rule with no templates passes every file, and a template with two
# holes and nothing between them is refused. Every file, and the variants its path allows, are judged in one jq pass.
template:check() {
  local rule=$1 variant glob executable file line failed=0 path variants where verdicts
  shift
  local -A files=()
  local -a globs=()
  while IFS=$'\x1f' read -r variant glob executable
  do
    [[ -n "${variant}" ]] || continue
    file=$(template:file "${rule}" "${variant}")
    if [[ ! -f "${file}" ]]
    then
      error "${rule} lists the template ${variant}, and ${file} is not here; run: kit update"
      return 1
    fi
    line=$(template:ambiguous "${file}" | head -n 1)
    if [[ -n "${line}" ]]
    then
      error "${file}:${line}: two holes with nothing between them; no file could be split between them"
      return 1
    fi
    files[${variant}]=${file}
    globs+=("${variant}" "$(if [[ -n "${glob}" ]]; then context:glob-ere "${glob}"; fi)")
  done < <(template:variants "${rule}")
  (( ${#files[@]} > 0 )) || return 0
  verdicts=$(jq -L "${kit_jq}" --null-input --raw-output \
      --rawfile templates <(for variant in "${!files[@]}"
                            do
                              printf '%s\x1f' "${variant}"
                              cat "${files[${variant}]}"
                              printf '\x1e'
                            done) \
      --rawfile contents <(for path in "$@"; do printf '%s\x1f' "${path}"; cat "${path}"; printf '\x1e'; done) \
      --args '
    import "template" as template;
    ($templates | split("\u001e") | map(select(. != "") | split("\u001f") | {(.[0]): .[1]}) | add) as $templates
    | [range(0; $ARGS.positional | length; 2) as $i | {variant: $ARGS.positional[$i], glob: $ARGS.positional[$i + 1]}]
        as $globs
    | [$contents | split("\u001e")[] | select(. != "") | split("\u001f") | .[0] as $path
        | {path: $path, text: (.[1:] | join("\u001f")),
           variants: [$globs[] | select(.glob as $g | $g == "" or ($path | test($g))) | .variant]}] as $files
    | template::check($templates; $files)
    | "\(.path)\t\(.variants | join(" "))\t\(.line // "")\t\(.variant // "")"' "${globs[@]}") || {
    error "the $(template:rule-name "${rule}") templates could not be checked; see above"
    return 1
  }
  while IFS=$'\t' read -r path variants line variant
  do
    [[ -n "${path}" ]] || continue
    failed=$(( failed + 1 ))
    where='the template at its end'
    if [[ -n "${line}" ]]
    then
      where="${files[${variant}]} line ${line}: $(sed -n "${line}p" "${files[${variant}]}")"
    fi
    error "${path} does not have the shape of the $(template:rule-name "${rule}") template (${variants// /, });" \
      "it leaves ${where}"
  done <<<"${verdicts}"
  (( failed == 0 ))
}

# ----------------------------------------------------------------------------------------------------------------------
