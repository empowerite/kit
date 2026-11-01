# shellcheck shell=bash
# region.sh - the kit's region inside a file the repository owns: marked, hosted, written, and removed (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
#
# A region is the part of a repository's file that the kit owns. It sits between a bare kit:generate line and an
# end line, both naming the path, in the file's own comment syntax. The rest of the file belongs to the repository.
# kit generate replaces the region's content with what kit region prints. A package may state the shape of the file
# that holds its region, under hosts/<base>/, and the region is then placed where that host template puts it.
#
# see: docs/generate.md, docs/packages.md

use release resolve render file

# region:name <dest> prints the region's name: the path relative to the base's root.
region:name() {
  local home
  home=$(release:base-root home)
  if [[ "$1" == "${home}"/* ]]
  then printf '%s\n' "${1#"${home}"/}"
  else printf '%s\n' "$1"
  fi
}

# region:command <dest> prints the command that generates the region. Under the repository it prints
# nothing, because the bare directive means `kit region <name>`. Under the home directory it prints
# `kit region --home <name>`, which the bare form cannot express.
region:command() {
  local home
  home=$(release:base-root home)
  if [[ "$1" == "${home}"/* ]]
  then printf 'kit region --home %s\n' "${1#"${home}"/}"
  fi
}

# region:markers <dest> prints the region's two marker lines in the file's comment syntax, chosen by
# extension: an HTML comment, //, --, or #. The first line is the directive, `kit:generate <name>`, followed by
# the command in backticks when there is one. The second is the end marker, `kit:end <name>`. A kind of file with
# no comment syntax the kit knows is rejected earlier, by render:all, when it reads the package.
region:markers() {
  local open close name command
  case "${1##*.}" in
    (md | markdown | html | htm | xml | svg) open='<!-- ' close=' -->' ;;
    (js | mjs | cjs | ts | tsx | jsx | go | rs | c | h | cc | cpp | hpp | java | kt | swift | cs | scala)
      open='// ' close='' ;;
    (sql | lua | hs) open='-- ' close='' ;;
    (*) open='# ' close='' ;;
  esac
  name=$(region:name "$1")
  command=$(region:command "$1")
  printf '%skit:generate %s%s%s\n%skit:end %s%s\n' "${open}" "${name}" "${command:+ \`${command}\`}" "${close}" \
    "${open}" "${name}" "${close}"
}

# region:known <path> succeeds when the kit knows the file's comment syntax, so a region can be marked in
# it.
region:known() {
  case "${1##*/}" in
    (*.json | *.jsonc | *.json5) return 1 ;;
    (*.*) case "${1##*.}" in
      (md | markdown | html | htm | xml | svg | js | mjs | cjs | ts | tsx | jsx | go | rs | c | h | cc | cpp | hpp \
        | java | kt | swift | cs | scala | sql | lua | hs | sh | bash | zsh | yml | yaml | toml | ini | cfg | conf \
        | py | rb | pl | txt | gitignore | gitattributes | editorconfig | env | just | mk | Makefile) return 0 ;;
      (*) return 1 ;;
    esac ;;
    (*) return 0 ;;
  esac
}

# region:has <dest> succeeds when the file contains the region's directive. A directive with no end
# marker fails (status 2) and names the file, because a write would drop everything after the directive.
region:has() {
  local path=$1 begin end
  [[ -f "${path}" ]] || return 1
  { read -r begin; read -r end; } < <(region:markers "$1")
  grep --quiet --line-regexp --fixed-strings "${begin}" "${path}" || return 1
  if ! grep --quiet --line-regexp --fixed-strings "${end}" "${path}"
  then
    error "${path}: the kit's region opens with '${begin}' and never closes; add '${end}' where it ends"
    return 2
  fi
}

# region:read <dest> prints the region's content, between its directive and its end marker. It fails
# when the file or the region is missing.
region:read() {
  local path=$1 begin end
  region:has "$1" || return $?
  { read -r begin; read -r end; } < <(region:markers "$1")
  awk -v b="${begin}" -v e="${end}" '$0 == e { on = 0 } on { print } $0 == b { on = 1 }' "${path}"
}

# region:host <content-file> prints the host template of the region whose rendered source is <content-file>: the file
# at the same path under the package's hosts/<base>/, which states the shape of the file that holds the region. It
# prints nothing when the package has none.
region:host() {
  local rest package base path host
  rest=${1##*/packages/}
  package=${rest%%/regions/*}
  rest=${rest#*/regions/}
  base=${rest%%/*}
  path=${rest#*/}
  host="$(resolve:home "${kit_root}" "${package}")/packages/${package}/hosts/${base}/${path}"
  [[ -f "${host}" ]] && printf '%s\n' "${host}"
  return 0
}

# region:host-insert <dest> <host> <begin> <end> writes <dest> with the region's directive and end marker placed where
# the host template puts them: after the fewest of the file's first lines that match the part of the template before
# the region, its trailing blank lines aside, then a blank line, and a blank line before what follows. It refuses, and
# names the host and the line the file leaves it at, when no run of first lines matches.
region:host-insert() {
  local dest=$1 host=$2 begin=$3 end=$4 prefix lines k rest
  # render:tmp makes its directory in this shell, so that a subshell's exit does not remove it.
  render:tmp >/dev/null
  prefix=${render_tmp}/host-prefix
  awk 'index($0, "kit:generate") { exit } { print }' "${host}" \
    | awk '{ line[NR] = $0 } END { n = NR; while (n > 0 && line[n] == "") n--; for (i = 1; i <= n; i++) print line[i] }' \
    > "${prefix}"
  lines=$(wc -l < "${dest}")
  for (( k = 0; k <= lines; k++ ))
  do
    # BSD head refuses -n 0, so the first k lines are read with awk.
    awk -v k="${k}" 'NR <= k' "${dest}" > "${prefix}.file"
    if [[ ! -s "${prefix}" && "${k}" -eq 0 ]] || template:unapply "${prefix}" "${prefix}.file" >/dev/null 2>&1
    then
      file:replace "${dest}" region:host-inserted "${dest}" "${k}" "${begin}" "${end}" || return 1
      return 0
    fi
  done
  rest=${host##*/packages/}
  error "${dest} does not open as the ${rest%%/hosts/*} package's host template has it, so the kit cannot place" \
    "its region; put ${begin} and ${end} where it belongs, then run: kit generate ${dest}"
  return 1
}

# region:host-inserted <dest> <k> <begin> <end> prints the file with an empty region, its markers alone, after its
# first <k> lines: a blank line before them unless k is 0, and one after them unless the rest starts with one.
region:host-inserted() {
  local dest=$1 k=$2 begin=$3 end=$4 rest
  rest=$(tail -n "+$(( k + 1 ))" "${dest}")
  awk -v k="${k}" 'NR <= k' "${dest}"
  (( k == 0 )) || printf '\n'
  printf '%s\n%s\n' "${begin}" "${end}"
  if [[ -n "${rest}" ]]
  then
    [[ -z "$(head -n 1 <<<"${rest}")" ]] || printf '\n'
    printf '%s\n' "${rest}"
  fi
}

# region:write <dest> <content-file> sets the region's content to <content-file>, the output of the
# region's command. When the package states the shape of the region's host file, under hosts/<base>/, it makes a
# missing file from that host template, and places the region in a file without one where the template puts it.
# Then it writes the content:
#
#   - between the directive and the end marker, when the file has them
#   - after a blank line at the end of the file, when the file has no region and the package states no host
#   - as the whole file, when there is no file and no host
region:write() {
  local path content=$2 begin end status=0 host
  path=$(release:target "$1")
  { read -r begin; read -r end; } < <(region:markers "$1")
  host=$(region:host "${content}")
  region:has "${path}" || status=$?
  (( status == 2 )) && return 1
  if [[ -n "${host}" ]] && (( status == 1 ))
  then
    use template
    if [[ -e "${path}" ]]
    then
      region:host-insert "${path}" "${host}" "${begin}" "${end}" || return 1
    else
      file:replace "${path}" template:render "${host}" '{}' || return 1
    fi
    status=0
  fi
  file:replace "${path}" region:filled "${path}" "${content}" "${begin}" "${end}" "${status}"
}

# region:filled <path> <content-file> <begin> <end> <has> prints the file with the region's content in place:
# between its markers when <has> is 0, the file has them; otherwise after the file's text and a blank line, or as the
# whole file when there is no text.
region:filled() {
  local path=$1 content=$2 begin=$3 end=$4 has=$5
  if (( has == 0 ))
  then
    awk -v b="${begin}" -v e="${end}" -v c="${content}" '
      $0 == b { print; while ((getline line < c) > 0) print line; skip = 1; next }
      $0 == e { skip = 0 }
      !skip { print }' "${path}"
    return 0
  fi
  if [[ -s "${path}" ]]
  then
    cat "${path}"
    [[ "$(tail -c 1 "${path}")" == "" ]] || printf '\n'
    printf '\n'
  fi
  printf '%s\n' "${begin}"
  cat "${content}"
  printf '%s\n' "${end}"
}

# region:remove <dest> [<content-file>] removes the region, its directive, its end marker, and the blank line
# before it. A file left empty is deleted, and so is one left as the package's host template made it, holes at their
# defaults, when <content-file> names the region's source and its package states a host.
region:remove() {
  local path begin end host made stripped
  path=$(release:target "$1")
  [[ -f "${path}" ]] || return 0
  { read -r begin; read -r end; } < <(region:markers "$1")
  # The x keeps the trailing newlines a command substitution would drop.
  stripped=$(region:strip "${begin}" "${end}" < "${path}" && printf x) || return 1
  stripped=${stripped%x}
  host=$( [[ -n "${2:-}" ]] && region:host "$2" )
  if [[ -n "${host}" ]]
  then
    use template
    made=$(template:render "${host}" '{}' | region:strip "${begin}" "${end}")
    [[ "$(printf '%s' "${stripped}")" != "${made}" ]] || stripped=''
  fi
  if [[ -n "${stripped}" ]]
  then file:replace "${path}" printf '%s' "${stripped}"
  else rm -f "${path}"
  fi
}

# region:strip <begin> <end> reads a file and prints it without the region between those marker lines, the
# markers included, and without the blank line before the region.
region:strip() {
  awk -v b="$1" -v e="$2" '
    $0 == b { skip = 1; held = 0; next }
    $0 == e { skip = 0; next }
    skip { next }
    held && printed { print ""; held = 0 }
    $0 == "" { held = 1; next }
    { print; printed = 1 }'
}

# ----------------------------------------------------------------------------------------------------------------------
