# shellcheck shell=bash
# yaml.sh - the kit's YAML files, read through yq (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# The kit's files that are not JSON are YAML: the repository's kit.yml, a package's package.yml, the kinds files under
# .kit/kinds/, and the frontmatter of a rule or style, between the opening `---` and the next. This module reads them
# through yq 4, mikefarah's (https://github.com/mikefarah/yq), one yq process per read, with an expression in yq's own
# language. So every form YAML allows reads as YAML says: a block scalar, a flow list, an anchor and its alias, a
# comment after a value, and CRLF line endings. A single-quoted value is read literally, backslashes included, which
# is how a pattern is written; a double-quoted one has its escapes unescaped.
#
# The reads, and what each prints:
#
#   yaml:value <file> <key>             the scalar under <key>, unquoted; nothing for a key that is missing or empty,
#                                       or that holds a list or a map
#   yaml:list <file> <key>              the scalar items of the list under <key>, unquoted, one per line
#   yaml:map <file> <key>               "<name>\t<value>" for every scalar entry of the map under <key>, the value
#                                       unquoted, and empty when the entry is empty
#   yaml:records <file> <key> <field>…  one line per record of the list under <key>: the named fields' values, in
#                                       the order named, separated by the unit separator \x1f (not a tab, because a
#                                       pattern may contain a tab); a field the record does not have is empty
#   yaml:keys <file>                    the top-level keys, one per line, in order
#   yaml:json <file>                    the document as JSON, on one line, for a jq program
#   yaml:read <file>                    the document as YAML, as yq prints it
#
# Each reads the frontmatter alone when the file's first line is `---`, and otherwise the whole file, and reads a file
# that does not exist as nothing. yaml:yq runs yq over a file for a reader of its own, such as context:read-kinds,
# which reads a stream of documents.
#
# A command reads the same file the same way many times over: kit check reads kit.yml some eight hundred times.
# Nearly every read runs in a subshell, as $(yaml:value …) or < <(yaml:records …), so a result kept in a shell
# variable would die with the subshell that read it. The results are kept in files instead, one per read, under
# ${XDG_CACHE_HOME:-${HOME}/.cache}/kit/yaml/, where the kit keeps its caches. An entry is named for the read and the
# file's absolute path, and holds the file's text as it was read beside the result. A read replays the result, with no
# process, while the file's text is the text the entry holds, and runs yq again otherwise, so a changed file is never
# read stale, whatever its size and time say: bash compares two files' text with a builtin, where a size and time from
# stat would cost a process, as much as the yq run it saves. The directory is made with mode 700, and used only while
# it is a directory, not a link, that this user owns; an entry is replayed or replaced only while it is a regular file,
# not a link, that this user owns, and is written through a temporary file in the same directory, moved over it.
# Entries older than seven days are removed once per command.
#
# A file yq cannot parse, with tabs for indentation or an unclosed quote, is refused in one line that names the file
# and gives yq's first error line, and the read returns 1. Without yq on PATH, or with a yq that is not mikefarah's
# (the python yq answers to the same name), the module says so once, when it is loaded, in a line that names the
# remedy, and every read returns 2 without a word. Loaded inside a subshell, as the loader's bash_path probe loads it,
# it says nothing, keeps no results, and leaves the word to the process's own load.

# yaml_ready is true when mikefarah's yq is on PATH, which yaml:detect checks once, when the module is loaded.
# yaml_cache is the directory of kept results, which yaml:cache names when the module is loaded in the main shell,
# and empty where it was loaded in a subshell or the directory cannot be used.
declare -g yaml_ready=false yaml_cache=''

# yaml:detect looks for mikefarah's yq on PATH and sets yaml_ready. In the main shell it says what is missing, and the
# remedy; in a subshell it says nothing.
yaml:detect() {
  local version=''
  if command -v yq >/dev/null 2>&1
  then
    version=$(yq --version 2>/dev/null) || version=''
  fi
  if [[ "${version}" == *mikefarah* ]]
  then
    yaml_ready=true
    return 0
  fi
  (( BASH_SUBSHELL == 0 )) || return 0
  if [[ -z "${version}" ]]
  then
    error "yq is not on PATH, and the kit reads its YAML through yq 4 (mikefarah/yq);" \
      "run: mise install, or brew install yq"
  else
    error "the yq on PATH is '${version%%$'\n'*}', not yq 4 (mikefarah/yq), which the kit reads its YAML through;" \
      "run: mise install, or brew install yq"
  fi
}

# yaml:usable is true while yaml_cache names a directory, not a link, that this user owns.
yaml:usable() {
  [[ -n "${yaml_cache}" && -d "${yaml_cache}" && ! -L "${yaml_cache}" && -O "${yaml_cache}" ]]
}

# yaml:cache names the directory of kept results for yaml_cache, making it with mode 700 when it is not there, and
# leaves yaml_cache empty when it cannot be made or used. At the top of a command, the one with no kit_depth, it also
# removes the entries older than seven days.
yaml:cache() {
  yaml_cache=${XDG_CACHE_HOME:-${HOME:-}/.cache}/kit/yaml
  if [[ ! -e "${yaml_cache}" && ! -L "${yaml_cache}" ]]
  then
    mkdir -p "${yaml_cache}" 2>/dev/null && chmod 700 "${yaml_cache}" 2>/dev/null
  fi
  if ! yaml:usable
  then
    yaml_cache=''
    return 0
  fi
  (( ${kit_depth:-0} == 0 )) || return 0
  find "${yaml_cache}" -mindepth 1 -maxdepth 1 -type f -mtime +7 -delete 2>/dev/null
  return 0
}

# yaml:keep <store> <text> <result> writes the file's text and the result to the entry <store>, through a temporary
# file in the same directory moved over it, unless the entry is there and is a link or not a regular file this user
# owns.
yaml:keep() {
  local store=$1 part=$1.${BASHPID}.part
  [[ ! -L "${store}" && ( ! -e "${store}" || ( -f "${store}" && -O "${store}" ) ) ]] || return 0
  [[ ! -e "${part}" && ! -L "${part}" ]] || return 0
  printf '%s\0%s' "$2" "$3" > "${part}" 2>/dev/null || return 0
  mv -f "${part}" "${store}" 2>/dev/null || rm -f "${part}" 2>/dev/null
  return 0
}

# yaml:run <file> <expression> [<option>…] runs yq over the file with the expression and the options, without colors
# or document separators, and prints what yq prints. It returns 1 for a file yq cannot parse, after one line that
# names the file and gives yq's first error line. A warning yq prints beside a result is passed on as one.
yaml:run() {
  local file=$1 expression=$2 said
  shift 2
  if ! { said=$(yq --no-colors --no-doc --unwrapScalar "$@" "${expression}" "${file}" 2>&1 1>&3); } 3>&1
  then
    said=${said%%$'\n'*}
    said=${said#"Error: bad file '${file}': "}
    error "${file}: ${said#Error: }"
    return 1
  fi
  [[ -z "${said}" ]] || warn "${file}: ${said%%$'\n'*}"
}

# yaml:yq <file> <read> <expression> [<option>…] prints what yaml:run prints for the file, the expression, and the
# options, replaying the result kept for <read> and the file while the file's text is as it was read, and keeping
# the result of a run otherwise. <read> names the read for the entry: the reader and its arguments. It returns 2
# without yq, 0 for a file that does not exist, and otherwise what yaml:run returns.
yaml:yq() {
  local file=$1 read=$2 expression=$3 name store text kept out
  shift 3
  [[ "${yaml_ready}" == true ]] || return 2
  [[ -f "${file}" ]] || return 0
  name="${read} ${file}"
  [[ "${file}" == /* ]] || name="${read} ${PWD}/${file}"
  name=${name//%/%25}
  name=${name//\//%2F}
  if ! yaml:usable || (( ${#name} > 250 ))
  then
    yaml:run "${file}" "${expression}" "$@"
    return
  fi
  store=${yaml_cache}/${name}
  IFS= read -r -d '' text < "${file}" || true
  if [[ -f "${store}" && ! -L "${store}" && -O "${store}" ]]
  then
    { IFS= read -r -d '' kept || true; IFS= read -r -d '' out || true; } < "${store}"
    if [[ "${kept}" == "${text}" ]]
    then
      printf '%s' "${out}"
      return 0
    fi
  fi
  out=$(yaml:run "${file}" "${expression}" "$@") || return $?
  [[ -z "${out}" ]] || out+=$'\n'
  printf '%s' "${out}"
  yaml:keep "${store}" "${text}" "${out}"
}

# yaml:eval <file> <read> <expression> [<option>…] runs yq, as yaml:yq does, over the one document a reader reads:
# the frontmatter when the file's first line is ---, and otherwise the whole file.
yaml:eval() {
  local first=''
  [[ -f "$1" ]] || return 0
  IFS= read -r first < "$1" || true
  if [[ "${first%$'\r'}" == '---' ]]
  then
    yaml:yq "$@" --front-matter=extract
  else
    yaml:yq "$@"
  fi
}

# Each expression below resolves the aliases, then keeps one node per document until its last step, an array of the
# lines to print, and then takes the array apart: yq gives a literal a value even where nothing matched, so a select
# followed by a literal would print where the line should say nothing.

# yaml:value <file> <key> prints the scalar under <key>, unquoted, or nothing.
yaml:value() {
  kit_yaml_key=$2 yaml:eval "$1" "value $2" 'explode(.) | (if kind == "map" then [.[strenv(kit_yaml_key)]] else [] end)
    | map(select(kind == "scalar" and tag != "!!null")) | .[]'
}

# yaml:list <file> <key> prints the scalar items of the list under <key>, unquoted, one per line.
yaml:list() {
  kit_yaml_key=$2 yaml:eval "$1" "list $2" 'explode(.) | (if kind == "map" then .[strenv(kit_yaml_key)] else null end)
    | (if kind == "seq" then . else [] end) | map(select(kind == "scalar" and tag != "!!null")) | .[]'
}

# yaml:map <file> <key> prints "<name>\t<value>" for every scalar entry of the map under <key>, values unquoted, an
# empty value as empty.
yaml:map() {
  kit_yaml_key=$2 yaml:eval "$1" "map $2" 'explode(.) | (if kind == "map" then .[strenv(kit_yaml_key)] else null end)
    | (if kind == "map" then to_entries else [] end) | map(select(.value | kind == "scalar"))
    | map((.key | tostring) + "\t" + (.value | if tag == "!!null" then "" else tostring end)) | .[]'
}

# yaml:records <file> <key> <field>… prints one line per record of the list under <key>: the named fields' values,
# in the order named, separated by the unit separator \x1f (not a tab, because a pattern may contain a tab). A
# field the record does not have is empty.
yaml:records() {
  local file=$1 key=$2
  shift 2
  kit_yaml_key=${key} kit_yaml_fields="$*" kit_yaml_us=$'\x1f' yaml:eval "${file}" "records ${key} $*" "
    explode(.) | (if kind == \"map\" then .[strenv(kit_yaml_key)] else null end)
    | (if kind == \"seq\" then . else [] end) | map(select(kind == \"map\"))
    | map(. as \$record | strenv(kit_yaml_fields) | split(\" \")
        | map(. as \$field | \$record[\$field]
              | if kind == \"scalar\" and tag != \"!!null\" then tostring else \"\" end)
        | join(strenv(kit_yaml_us)))
    | .[]"
}

# yaml:keys <file> prints the top-level keys, one per line, in order.
yaml:keys() {
  yaml:eval "$1" keys 'explode(.) | (if kind == "map" then keys else [] end) | .[]'
}

# yaml:json <file> prints the document as JSON, on one line, for a jq program.
yaml:json() {
  yaml:eval "$1" json '.' --output-format=json --indent=0
}

# yaml:read <file> prints the document as YAML, as yq prints it.
yaml:read() {
  yaml:eval "$1" read '.' --no-doc=false
}

yaml:detect
if (( BASH_SUBSHELL == 0 ))
then
  yaml:cache
fi

# ----------------------------------------------------------------------------------------------------------------------
