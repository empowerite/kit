# shellcheck shell=bash
# yaml.sh - the kit's YAML files, read through yq (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# The kit's files that are not JSON are YAML: the repository's kit.yml, a package's package.yml, the kinds files under
# .kit/kinds/, and the frontmatter of a rule or style, between the opening `---` and the next. This module reads them
# through yq 4, mikefarah's (https://github.com/mikefarah/yq), with expressions in yq's own language: one yq run
# answers every read below of a file but yaml:json and yaml:read, which run yq for themselves. So every form YAML
# allows reads as YAML says: a block scalar, a flow list, an anchor and its alias, a comment after a value, and CRLF
# line endings. A single-quoted value is read literally, backslashes included, which
# is how a pattern is written; a double-quoted one has its escapes unescaped.
#
# The reads, and what each prints:
#
#   yaml:value <file> <key>             the scalar under <key>, unquoted; nothing for a key that is missing or empty,
#                                       or that holds a list or a map
#   yaml:type <file> <key>              the tag of the value under <key>, as yq names it: !!seq, !!map, !!str, and
#                                       so on, or !!null for a key that is missing or empty
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
# A read prints, and a caller that takes what it prints as $(yaml:value …) or list:read forks a subshell for it. Two
# calls take it in the caller's own shell instead, and fork nothing:
#
#   yaml:get <name> <reader> <argument>…     sets the variable <name> to what yaml:<reader> prints, as $(…) would
#   yaml:lines <array> <reader> <argument>…  fills the array with the lines yaml:<reader> prints, as list:read would
#
# A command reads the same file the same way many times over: kit check reads kit.yml some eight hundred times.
# Many reads run in a subshell, as $(yaml:value …) or < <(yaml:records …), so a result kept in a shell variable
# would die with the subshell that read it. The results are kept in files instead, under
# ${XDG_CACHE_HOME:-${HOME}/.cache}/kit/yaml/, where the kit keeps its caches: for each file, one entry holds its facts,
# every top-level key's value, type, list, map, and records, which the readers but yaml:json and yaml:read pick from,
# and every other read has an entry of its own. An entry is named for the read, facts for the facts, and the file's
# absolute path, and holds the file's text as it was read beside the result. A read replays the result, with no
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

# What a read hands to the function that called it, set in the reader's own shell, so it forks none: yaml:entry sets
# yaml_entry, yaml_text, and yaml_out, yaml:facts sets yaml_facts, yaml:head sets yaml_head, and yaml:pick sets
# yaml_picked.
declare -g yaml_entry='' yaml_text='' yaml_out='' yaml_head=''
# yaml_capture is true inside yaml:get and yaml:lines, which declare it, and yaml_printed, local to them as well, holds
# what the reader would have printed.
declare -g yaml_capture=false yaml_printed=''
declare -ga yaml_facts=() yaml_picked=()

# yaml:detect looks for mikefarah's yq on PATH and sets yaml_ready. A yq on PATH that is the one kit_yq names, which
# the loader exports once it has seen it is mikefarah's, is trusted without starting it. In the main shell it says
# what is missing, and the remedy; in a subshell it says nothing.
yaml:detect() {
  local version='' yq=''
  hash yq 2>/dev/null && yq=${BASH_CMDS[yq]:-}
  if [[ -n "${yq}" && "${yq}" == "${kit_yq:-}" ]]
  then
    yaml_ready=true
    return 0
  fi
  if [[ -n "${yq}" ]]
  then
    version=$("${yq}" --version 2>/dev/null) || version=''
  fi
  if [[ "${version}" == *mikefarah* ]]
  then
    yaml_ready=true
    return 0
  fi
  (( BASH_SUBSHELL == 0 )) || return 0
  if [[ -z "${version}" ]]
  then
    guard:missing yq "yq is not on PATH, and the kit reads its YAML through yq 4, mikefarah/yq"
  else
    guard:missing yq \
      "the yq on PATH is '${version%%$'\n'*}', not yq 4, mikefarah/yq, which the kit reads its YAML through"
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

# yaml:entry <file> <read> sets yaml_entry to the entry that keeps <read> of the file, and yaml_text to the file's
# text, and returns 0 when the entry holds a result read from that text, with the result in yaml_out. Otherwise it
# returns 1, with yaml_out empty, and with yaml_entry empty where no entry can be kept: the cache cannot be used, or
# the entry's name, with its temporary file's suffix, would be longer than a file name may be.
yaml:entry() {
  local file=$1 name kept LC_ALL=C  # so the name's length is counted in bytes, as a file name's is
  yaml_entry='' yaml_text='' yaml_out=''
  name="$2 ${file}"
  [[ "${file}" == /* ]] || name="$2 ${PWD}/${file}"
  name=${name//%/%25}
  name=${name//\//%2F}
  if ! yaml:usable || (( ${#name} > 240 ))
  then
    return 1
  fi
  yaml_entry=${yaml_cache}/${name}
  # read -d '' meets the end of a file before any NUL, so it returns 1 on every file, the whole of it read.
  IFS= read -r -d '' yaml_text < "${file}" || true
  [[ -f "${yaml_entry}" && ! -L "${yaml_entry}" && -O "${yaml_entry}" ]] || return 1
  # Each read stops at a NUL, and the last at the end of the file, where it returns 1 with its text read.
  { IFS= read -r -d '' kept || true; IFS= read -r -d '' yaml_out || true; } < "${yaml_entry}"
  [[ "${kept}" == "${yaml_text}" ]] && return 0
  yaml_out=''
  return 1
}

# yaml:writable <entry> is true when the entry can be written: it is not there, or is a regular file, not a link,
# that this user owns, and its temporary file, <entry>.<pid>.part, is not there.
yaml:writable() {
  [[ ! -L "$1" && ( ! -e "$1" || ( -f "$1" && -O "$1" ) ) ]] || return 1
  [[ ! -e "$1.${BASHPID}.part" && ! -L "$1.${BASHPID}.part" ]]
}

# yaml:keep <entry> <text> <result> writes the file's text and the result to the entry, through a temporary file in
# the same directory moved over it, unless yaml:writable says the entry cannot be written.
yaml:keep() {
  local part=$1.${BASHPID}.part
  yaml:writable "$1" || return 0
  printf '%s\0%s' "$2" "$3" 2>/dev/null > "${part}" || return 0
  mv -f "${part}" "$1" 2>/dev/null || rm -f "${part}" 2>/dev/null
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
    yaml:refuse "${file}" "${said}"
    return 1
  fi
  [[ -z "${said}" ]] || warn "${file}: ${said%%$'\n'*}"
}

# yaml:refuse <file> <said> says, in one line, that yq cannot parse the file, with the first line of what yq said.
yaml:refuse() {
  local said=${2%%$'\n'*}
  said=${said#"Error: bad file '$1': "}
  error "$1: ${said#Error: }"
}

# yaml:yq <file> <read> <expression> [<option>…] prints what yaml:run prints for the file, the expression, and the
# options, replaying the result kept for <read> and the file while the file's text is as it was read, and keeping
# the result of a run otherwise. <read> names the read for the entry: the reader and its arguments. It returns 2
# without yq, 0 for a file that does not exist, and otherwise what yaml:run returns.
yaml:yq() {
  local file=$1 read=$2 expression=$3
  shift 3
  [[ "${yaml_ready}" == true ]] || return 2
  [[ -f "${file}" ]] || return 0
  if ! yaml:entry "${file}" "${read}"
  then
    yaml_out=$(yaml:run "${file}" "${expression}" "$@") || return $?
    [[ -z "${yaml_out}" ]] || yaml_out+=$'\n'
    [[ -z "${yaml_entry}" ]] || yaml:keep "${yaml_entry}" "${yaml_text}" "${yaml_out}"
  fi
  if [[ "${yaml_capture}" == true ]]
  then
    yaml_printed=${yaml_out%$'\n'}
  else
    printf '%s' "${yaml_out}"
  fi
}

# yaml:eval <file> <read> <expression> [<option>…] runs yq, as yaml:yq does, over the one document a reader reads:
# the frontmatter when the file's first line is ---, and otherwise the whole file.
yaml:eval() {
  local first=''
  [[ -f "$1" ]] || return 0
  IFS= read -r first < "$1" || true  # a first line with no newline is read, and returns 1
  if [[ "${first%$'\r'}" == '---' ]]
  then
    yaml:yq "$@" --front-matter=extract
  else
    yaml:yq "$@"
  fi
}

# The facts of a document are what yaml:value, yaml:type, yaml:list, yaml:map, yaml:records, and yaml:keys read from
# it, every top-level key's at once, so one yq run answers every read of the file. Each fact is one line: its kind,
# the key, and its fields, separated by the unit separator \x1f, each field with its backslashes doubled, and its
# newlines and unit separators written \n and \x1f, so printf %b gives it back; a NUL is dropped, as bash drops one
# from what a command prints. Each document's facts start with D, and then come each key's:
#
#   K <key>                             the key, in the document's order
#   T <key> <tag>                       the tag of its value
#   V <key> <value>                     its value, when that is a scalar, not null
#   L <key> <item>                      each scalar item of its list, not null
#   M <key> <name>\t<value>             each scalar entry of its map, a null value as empty
#   R <key> [<field> <value>]…          each record of its list, a map, with its scalar fields, not null
#
# The expression resolves the aliases, then builds an array of facts for each key, each fact an array of its fields:
# yq gives a literal a value even where nothing matched, so a select followed by a literal would make a fact where
# there is none.
# shellcheck disable=SC2016  # $key is the expression's own variable, which yq binds, not the shell's
declare -gr yaml_facts_expression='explode(.) | (if kind == "map" then to_entries else [] end)
  | map((.key | tostring) as $key | .value
      | [["K", $key], ["T", $key, type]]
        + (if kind == "scalar" and tag != "!!null" then [["V", $key, tostring]] else [] end)
        + (if kind == "seq" then map(select(kind == "scalar" and tag != "!!null") | ["L", $key, tostring]) else [] end)
        + (if kind == "map" then to_entries | map(select(.value | kind == "scalar")
            | ["M", $key, (.key | tostring) + "\t" + (.value | if tag == "!!null" then "" else tostring end)])
          else [] end)
        + (if kind == "seq" then map(select(kind == "map") | ["R", $key] + (to_entries
            | map(select(.value | (kind == "scalar" and tag != "!!null")) | [(.key | tostring), (.value | tostring)])
            | flatten)) else [] end))
  | [["D"]] + flatten(1)
  | map(map(split(strenv(kit_yaml_bs)) | join(strenv(kit_yaml_bs) + strenv(kit_yaml_bs))
      | split(strenv(kit_yaml_nl)) | join("\\n") | split(strenv(kit_yaml_us)) | join("\\x1f")
      | sub(strenv(kit_yaml_nul), "")) | join(strenv(kit_yaml_us)))
  | .[]'

# yaml:facts <file> sets the array yaml_facts to the facts of the one document a reader reads, one per element: the
# frontmatter when the file's first line is ---, and otherwise the whole file. It replays the facts kept for the file
# while its text is as it was read. Otherwise yq appends them to the entry's temporary file, after the text, and the
# file is moved over the entry, so the read forks no shell; what yq says goes to the same file, told apart from the
# facts by its shape, a line that is not D and does not start with a kind and a unit separator. Where no entry can be
# written, yq runs as yaml:run runs it. It returns 0 for a file that does not exist, with no facts, 2 without yq, and 1
# for a file yq cannot parse, after saying so in one line.
yaml:facts() {
  local file=$1 first='' part said='' fact status=0
  local -a options=()
  local -x kit_yaml_bs=\\ kit_yaml_nl=$'\n' kit_yaml_us=$'\x1f' kit_yaml_nul='\x00'
  yaml_facts=()
  [[ -f "${file}" ]] || return 0
  [[ "${yaml_ready}" == true ]] || return 2
  IFS= read -r first < "${file}" || true  # a first line with no newline is read, and returns 1
  [[ "${first%$'\r'}" != '---' ]] || options=(--front-matter=extract)
  if yaml:entry "${file}" facts
  then
    # the entry opens with the text its facts are of, to its NUL, and one cut short returns 1
    { IFS= read -r -d '' _ || true; mapfile -t yaml_facts; } < "${yaml_entry}"
    return 0
  fi
  part=${yaml_entry}.${BASHPID}.part
  if [[ -z "${yaml_entry}" ]] || ! yaml:writable "${yaml_entry}" \
    || ! printf '%s\0' "${yaml_text}" 2>/dev/null > "${part}"
  then
    yaml_out=$(yaml:run "${file}" "${yaml_facts_expression}" "${options[@]}") || return $?
    mapfile -t yaml_facts <<<"${yaml_out}"
    return 0
  fi
  yq --no-colors --no-doc --unwrapScalar "${options[@]}" "${yaml_facts_expression}" "${file}" >> "${part}" 2>&1 \
    || status=$?
  # the entry opens with the text its facts are of, to its NUL, and one cut short returns 1
  { IFS= read -r -d '' _ || true; mapfile -t yaml_facts; } < "${part}"
  for fact in "${yaml_facts[@]}"
  do
    if [[ "${fact}" != D && "${fact}" != [KTVLMR]$'\x1f'* ]]
    then
      said=${fact}
      break
    fi
  done
  if (( status != 0 ))
  then
    rm -f "${part}"
    yaml_facts=()
    yaml:refuse "${file}" "${said}"
    return 1
  fi
  mv -f "${part}" "${yaml_entry}" 2>/dev/null || rm -f "${part}" 2>/dev/null
  [[ -z "${said}" ]] || warn "${file}: ${said}"
  return 0
}

# yaml:head <kind> <key> sets yaml_head to the start of the facts of <kind> under <key>, as they are written: the
# kind, a unit separator, and the key, escaped as the facts' fields are.
yaml:head() {
  local key bs=\\
  key=${2//"${bs}"/"${bs}${bs}"}
  key=${key//$'\n'/'\n'}
  yaml_head=$1$'\x1f'${key//$'\x1f'/'\x1f'}
}

# yaml:pick <kind> [<key>] sets the array yaml_picked to the fields of every fact of <kind> in yaml_facts, under
# <key> when one is named, as they are written: the rest of the fact's line after its kind and its key.
yaml:pick() {
  local fact
  yaml_picked=()
  if (( $# == 1 ))
  then
    for fact in "${yaml_facts[@]}"
    do
      [[ "${fact}" != "$1"$'\x1f'* ]] || yaml_picked+=("${fact#"$1"$'\x1f'}")
    done
    return 0
  fi
  yaml:head "$1" "$2"
  for fact in "${yaml_facts[@]}"
  do
    if [[ "${fact}" == "${yaml_head}" ]]
    then
      yaml_picked+=('')
    elif [[ "${fact}" == "${yaml_head}"$'\x1f'* ]]
    then
      yaml_picked+=("${fact#"${yaml_head}"$'\x1f'}")
    fi
  done
}

# yaml:print <field>… prints the fields, their escapes undone, one per line, as yq prints them and a command
# substitution keeps them: the newlines at the end of the last are dropped. Under yaml:get or yaml:lines it sets
# yaml_printed to the lines instead, without the last newline.
yaml:print() {
  local out='' field
  (( $# )) || return 0
  for field in "$@"
  do
    printf -v field '%b' "${field}"
    out+=${field}$'\n'
  done
  while [[ "${out}" == *$'\n' ]]
  do
    out=${out%$'\n'}
  done
  if [[ "${yaml_capture}" == true ]]
  then
    yaml_printed=${out}
  elif [[ -n "${out}" ]]
  then
    printf '%s\n' "${out}"
  fi
}

# yaml:value <file> <key> prints the scalar under <key>, unquoted, or nothing.
yaml:value() {
  yaml:facts "$1" || return
  yaml:pick V "$2"
  yaml:print "${yaml_picked[@]}"
}

# yaml:type <file> <key> prints the tag of the value under <key>: !!seq, !!map, !!str, !!int, !!bool, and !!null for
# a key that is missing or empty, one line for each document. A file that does not exist prints nothing, as every
# read does.
yaml:type() {
  local fact tag='' document=false
  local -a tags=()
  [[ -f "$1" ]] || return 0
  yaml:facts "$1" || return
  yaml:head T "$2"
  # the D after the facts closes the last document, as each D closes the one before it
  for fact in "${yaml_facts[@]}" D
  do
    if [[ "${fact}" == D ]]
    then
      [[ "${document}" == false ]] || tags+=("${tag:-!!null}")
      document=true tag=''
    elif [[ "${fact}" == "${yaml_head}"$'\x1f'* ]]
    then
      tag=${fact#"${yaml_head}"$'\x1f'}
    fi
  done
  yaml:print "${tags[@]}"
}

# yaml:list <file> <key> prints the scalar items of the list under <key>, unquoted, one per line.
yaml:list() {
  yaml:facts "$1" || return
  yaml:pick L "$2"
  yaml:print "${yaml_picked[@]}"
}

# yaml:map <file> <key> prints "<name>\t<value>" for every scalar entry of the map under <key>, values unquoted, an
# empty value as empty.
yaml:map() {
  yaml:facts "$1" || return
  yaml:pick M "$2"
  yaml:print "${yaml_picked[@]}"
}

# yaml:records <file> <key> <field>… prints one line per record of the list under <key>: the named fields' values,
# in the order named, separated by the unit separator \x1f (not a tab, because a pattern may contain a tab). A
# field the record does not have is empty.
yaml:records() {
  local file=$1 key=$2 record field name line i
  local -a pairs lines=()
  local -A values
  shift 2
  yaml:facts "${file}" || return
  yaml:pick R "${key}"
  for record in "${yaml_picked[@]}"
  do
    IFS=$'\x1f' read -r -a pairs <<<"${record}"
    values=()
    for (( i = 0; i < ${#pairs[@]}; i += 2 ))
    do
      printf -v name '%b' "${pairs[i]}"
      [[ -z "${name}" ]] || values[${name}]=${pairs[i + 1]:-}
    done
    line=''
    for field in "$@"
    do
      [[ -z "${field}" ]] || line+=${values[${field}]:-}
      line+=$'\x1f'
    done
    lines+=("${line%$'\x1f'}")
  done
  yaml:print "${lines[@]}"
}

# yaml:keys <file> prints the top-level keys, one per line, in order.
yaml:keys() {
  yaml:facts "$1" || return
  yaml:pick K
  yaml:print "${yaml_picked[@]}"
}

# yaml:json <file> prints the document as JSON, on one line, for a jq program.
yaml:json() {
  yaml:eval "$1" json '.' --output-format=json --indent=0
}

# yaml:read <file> prints the document as YAML, as yq prints it.
yaml:read() {
  yaml:eval "$1" read '.' --no-doc=false
}

# yaml:get <name> <reader> <argument>… sets the variable <name> to what yaml:<reader> prints with the arguments, as
# <name>=$(yaml:<reader> …) would, but in this shell, so it forks none. It returns what the reader returns, and 2,
# after a line that says so, for a name that starts yaml_, which would name one of the reader's own.
yaml:get() {
  local yaml_capture=true yaml_printed='' yaml_status=0
  if [[ "$1" == yaml_* ]]
  then
    error "yaml:get: '$1' names a variable of the module's own; name another"
    return 2
  fi
  local -n yaml_into=$1
  "yaml:$2" "${@:3}" || yaml_status=$?
  # shellcheck disable=SC2034  # yaml_into is a nameref: the caller's variable, which the caller reads
  yaml_into=${yaml_printed}
  return "${yaml_status}"
}

# yaml:lines <array> <reader> <argument>… fills the array with the lines yaml:<reader> prints with the arguments, as
# list:read <array> yaml:<reader> … would, but in this shell, so it forks none. It returns what the reader returns,
# and 2, after a line that says so, for a name that starts yaml_.
yaml:lines() {
  local yaml_capture=true yaml_printed='' yaml_status=0
  if [[ "$1" == yaml_* ]]
  then
    error "yaml:lines: '$1' names a variable of the module's own; name another"
    return 2
  fi
  "yaml:$2" "${@:3}" || yaml_status=$?
  if [[ -n "${yaml_printed}" ]]
  then
    mapfile -t "$1" <<<"${yaml_printed}"
  else
    mapfile -t "$1" < /dev/null
  fi
  return "${yaml_status}"
}

yaml:detect
if (( BASH_SUBSHELL == 0 ))
then
  yaml:cache
fi

# ----------------------------------------------------------------------------------------------------------------------
