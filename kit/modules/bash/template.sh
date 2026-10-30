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
# A template is the file itself, with holes written «name». There are two kinds of hole:
#
#   a hole within a line      making a file fills it with a value. Checking a file matches it against any text on
#                             that line. A hole used twice must match the same text each time.
#   a hole that is a line     making a file fills it with the value's lines, else one blank line. Checking a file
#                             matches it against any run of lines, including none.
#
# The values of path, name, and stem come from the path. The caller gives the value of every other hole within a
# line.
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

# template:regex reads a template on stdin and prints the regular expression that matches a file with its shape,
# in Oniguruma syntax, which jq uses.
template:regex() {
  jq -L "${kit_jq}" --raw-input --slurp --raw-output 'import "template" as template; template::regex'
}

# template:test <regex> <file> is true when the file matches template:regex's expression.
template:test() {
  [[ "$(jq --raw-input --slurp --arg re "$1" 'test($re)' < "$2")" == true ]]
}

# template:matches <template> <file> is true when the file matches the template.
template:matches() {
  template:test "$(template:regex < "$1")" "$2"
}

# template:diverges <template> <file> prints "<line>\t<text>" for the first template line the file does not match.
# It finds the longest run of the template's first lines that the file still starts with.
template:diverges() {
  local total n
  total=$(wc -l < "$1")
  for (( n = 1; n <= total; n++ ))
  do
    if ! template:test "$({ head -n "${n}" "$1"; printf '«rest»\n'; } | template:regex)" "$2"
    then
      printf '%s\t%s\n' "${n}" "$(sed -n "${n}p" "$1")"
      return 0
    fi
  done
}

# template:holes <template> prints the holes within a line (the ones a value must fill), once each, in order.
template:holes() {
  grep --invert-match --extended-regexp '^«[a-z_]+»$' "$1" | grep --only-matching --extended-regexp '«[a-z_]+»' \
    | tr -d '«»' | awk '!seen[$0]++'
  return 0
}

# template:render <template> <values-json> prints the template with every hole filled from the values object:
#
#   - a hole within a line gets its value
#   - a hole that is a line gets its value's lines, else one blank line. When it is the template's last line and
#     has no value, it gets nothing, so the file never ends in a blank line.
template:render() {
  jq -L "${kit_jq}" --raw-input --slurp --join-output --argjson v "$2" \
    'import "template" as template; template::render($v)' < "$1"
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
# each file that matches none, it prints an error naming the variants it was checked against, and the line where
# the file stops matching the variant it follows furthest. A rule with no templates passes every file. Each
# template is compiled once, and all the files are read and tested in one pass, so checking every tracked file
# stays fast.
template:check() {
  local rule=$1 path variant glob executable file line text best where failed=0
  shift
  local -A regex=() allowed=()
  while IFS=$'\x1f' read -r variant glob executable
  do
    [[ -n "${variant}" ]] || continue
    file=$(template:file "${rule}" "${variant}")
    if [[ ! -f "${file}" ]]
    then
      error "${rule} lists the template ${variant}, and ${file} is not here; run: kit update"
      return 1
    fi
    regex[${variant}]=$(template:regex < "${file}")
  done < <(template:variants "${rule}")
  (( ${#regex[@]} > 0 )) || return 0
  for path in "$@"
  do
    allowed[${path}]=$(template:applies "${rule}" "${path}" | cut -d $'\x1f' -f 1 | paste -s -d ' ' -)
  done
  while IFS=$'\t' read -r path variants
  do
    [[ -n "${path}" ]] || continue
    failed=$(( failed + 1 ))
    best=0
    where='the template at its end'
    for variant in ${variants}
    do
      file=$(template:file "${rule}" "${variant}")
      IFS=$'\t' read -r line text < <(template:diverges "${file}" "${path}")
      if [[ -n "${line}" ]] && (( line > best ))
      then
        best=${line}
        where="${file} line ${line}: ${text}"
      fi
    done
    error "${path} does not have the shape of the $(template:rule-name "${rule}") template (${variants// /, });" \
      "it leaves ${where}"
  done < <(jq -L "${kit_jq}" --null-input --raw-output \
      --rawfile files <(for path in "$@"; do printf '%s\t%s\n' "${path}" "${allowed[${path}]}"; done) \
      --rawfile regexes <(for variant in "${!regex[@]}"
                          do
                            printf '%s\x1f%s\x1e' "${variant}" "${regex[${variant}]}"
                          done) \
      --rawfile contents <(for path in "$@"; do cat "${path}"; printf '\x1e'; done) '
    import "template" as template;
    ($regexes | split("\u001e") | map(select(. != "") | split("\u001f") | {(.[0]): .[1]}) | add) as $regex
    | ($files | split("\n") | map(select(. != "") | split("\t") | {path: .[0], variants: (.[1] | split(" "))}))
    | template::unmatched($regex; $contents | split("\u001e"))
    | "\(.path)\t\(.variants | join(" "))"')
  (( failed == 0 ))
}

# ----------------------------------------------------------------------------------------------------------------------
