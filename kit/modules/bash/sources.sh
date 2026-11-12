# shellcheck shell=bash
# sources.sh - the seam over artifact sources: the kinds, the pull path, and the package sources (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A seam is a contract plus the point that picks who answers it; docs/architecture.md says why not "interface".
#
# An artifact source is a place that serves artifacts by a protocol, its kind: kit's package tarballs, a Go module
# proxy, an npm registry, GitHub's release downloads. Each kind is a provider: sources/<kind>.sh beside it, whose
# functions are named sources/<kind>:<call>. sources: in kit.yml lists them, each a map with a name, a kind, and a
# url, or one of the words direct and off:
#
#   sources:
#     - name: team
#       url: git:empowerite/kit-team@v1.2.0
#       digest: sha256:3b1f…
#     - name: corp-go
#       kind: goproxy
#       url: https://proxy.corp/go
#     - direct
#
# A kind left out is kit-packages, so a kit.yml written before kinds reads as it did. A kit-packages source is a
# tarball pinned by the sha256 digest of its bytes: a url is git:<owner>/<repo>@<tag>, the forge's tarball of that
# tag, or any URL curl fetches. It is fetched once into the kit's cache, under its digest, and refused when its bytes
# have any other digest; nothing from a source runs before that check. Its packages are under packages/ in the
# tarball, or under kit/packages/ for a repository laid out as the kit's own is. A package name is looked up in the
# package sources, the last first, then in the release, then in the repository's own .kit/packages/, so a later
# source replaces a package of the same name. A bash_path entry <name>:<dir> names a directory in a source.
#
# Every other kind is on the pull path, the sources searched for an artifact in order, the first hit winning. It
# comes in layers, the first first: KIT_SOURCES, entries apart by commas, each direct, off, or <name>=<kind>:<url>,
# for a runner; sources: at the top of the person's workspaces.yml; and kit.yml's, whose default is direct. A layer's
# entries go before the ones below, a name or direct met again is the first one's, and off ends the path, so a layer
# that holds it replaces the layers below. direct is the open internet: each kind at its home. Package sources are the
# repository's, so a layer other than kit.yml that names one is refused.
#
# Every provider implements the calls below.
#
#   serves <coordinate>     succeeds when the kind serves an artifact of the coordinate's form, as release: or module:
#   home [<coordinate>]     the address direct stands for, for this kind, or for the coordinate where it says, as an
#                           image names its registry; nothing for a kind with no public home
#   keys                    the keys an entry of this kind takes beyond name, kind, url, and expect, one per line
#   fetch <url> <coordinate> <file>
#                           writes the artifact's bytes at <file>; 4 when the source does not serve it, 5 when it
#                           says it never will, as a scan's block, and 1 when it cannot be reached
#   hash <file> <coordinate>
#                           the ecosystem's own hash of the bytes, with its prefix, as a lock records it
#   versions <url> <coordinate>
#                           the versions the source serves of the coordinate's artifact, one per line; 4 when it
#                           serves none, 1 when it cannot be reached
#   settings <url> <entry>  NAME=value lines a language's package manager reads to search the source, from the
#                           entry's JSON; nothing for a kind no manager reads
#
# see: docs/packages.md, docs/superpowers/specs/2026-11-12-artifact-sources-design.md, issue #182

use yaml file

# sources_http is the HTTP status the last sources:get into a file was answered with, as 200 or 404.
declare -g sources_http=''

# sources_name_pattern is what a source's name must match.
declare -gr sources_name_pattern='^[a-z][a-z0-9-]*$'

# sources:kinds prints the kinds on the module path, one per line.
sources:kinds() {
  use:modules 'sources/*' | sed 's|^sources/||'
}

# sources:call <kind> <call> <argument>… runs the kind's provider's function for the call.
sources:call() {
  local kind=$1 call=$2
  shift 2
  use "sources/${kind}" || return 1
  "sources/${kind}:${call}" "$@"
}

# sources:list [<settings>] prints "<name>\x1f<url>\x1f<digest>" for each package source in the settings file,
# kit.yml by default, in order.
sources:list() {
  local -a rows
  sources:rows rows "$@" || return 1
  (( ${#rows[@]} == 0 )) || printf '%s\n' "${rows[@]}"
}

# sources:rows <array> [<settings>] fills the array with the lines sources:list prints, in this shell, so it forks
# nothing: the package sources alone, those whose kind is kit-packages or left out. It returns 1 when the settings
# file cannot be read. Its own variables start sources_rows_, so the name of the array does not.
sources:rows() {
  local sources_rows_file=${2:-kit.yml} sources_rows_row
  local -a sources_rows_all=() sources_rows_kept=()
  if [[ ! -f "${sources_rows_file}" ]]
  then
    mapfile -t "$1" < /dev/null
    return 0
  fi
  yaml:lines sources_rows_all records "${sources_rows_file}" sources name url digest kind || return 1
  for sources_rows_row in "${sources_rows_all[@]}"
  do
    case "${sources_rows_row##*$'\x1f'}" in
      ('' | kit-packages) sources_rows_kept+=("${sources_rows_row%$'\x1f'*}") ;;
      (*) ;;
    esac
  done
  if (( ${#sources_rows_kept[@]} ))
  then mapfile -t "$1" < <(printf '%s\n' "${sources_rows_kept[@]}")
  else mapfile -t "$1" < /dev/null
  fi
}

# sources:cache <digest> [<name>] prints the directory a package source with that digest is unpacked into. With a
# name, it sets that variable to the directory instead, and forks nothing.
sources:cache() {
  if (( $# > 1 ))
  then
    printf -v "$2" '%s/kit/sources/%s' "${XDG_CACHE_HOME:-${HOME}/.cache}" "${1#sha256:}"
  else
    printf '%s/kit/sources/%s\n' "${XDG_CACHE_HOME:-${HOME}/.cache}" "${1#sha256:}"
  fi
}

# sources:fetch <name> <url> <digest> unpacks the package source into its cache directory, unless it is there
# already, and prints the directory. The kit-packages provider fetches its bytes and hashes them. It refuses a source
# whose bytes have another digest, naming the digest they have, and a record without a sha256 digest.
sources:fetch() {
  local name=$1 url=$2 digest=$3 cache fresh have
  if [[ ! "${digest}" =~ ^sha256:[0-9a-f]{64}$ ]]
  then
    error "the source ${name} has no digest: pin it as digest: sha256:<hex> under sources: in kit.yml"
    return 1
  fi
  sources:cache "${digest}" cache
  if [[ -d "${cache}" ]]
  then
    printf '%s\n' "${cache}"
    return 0
  fi
  file:scratch fresh || return 1
  step "Fetching the source ${name} from ${url}"
  if ! sources:call kit-packages fetch "${url}" package: "${fresh}/source.tar.gz"
  then
    rm -rf "${fresh}"
    error "the source ${name} could not be fetched from ${url}"
    return 1
  fi
  have=$(sources:call kit-packages hash "${fresh}/source.tar.gz" package:) || { rm -rf "${fresh}"; return 1; }
  if [[ "${have}" != "${digest}" ]]
  then
    rm -rf "${fresh}"
    error "the source ${name} has digest ${have}, not the ${digest} kit.yml pins; check it, then pin it"
    return 1
  fi
  mkdir -p "${fresh}/tree"
  if ! tar --extract --gzip --file "${fresh}/source.tar.gz" --directory "${fresh}/tree" --strip-components 1
  then
    rm -rf "${fresh}"
    error "the source ${name} is not a gzipped tarball: ${url}"
    return 1
  fi
  mkdir -p "$(dirname "${cache}")"
  mv "${fresh}/tree" "${cache}"
  rm -rf "${fresh}"
  printf '%s\n' "${cache}"
}

# sources:fetch-all [<settings>] fetches every package source the settings file, kit.yml by default, names that is
# not in the cache yet. It returns 1 when one fails.
sources:fetch-all() {
  local name url digest row
  local -a rows
  sources:rows rows "$@" || return 1
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r name url digest <<<"${row}"
    [[ -n "${name}" ]] || continue
    sources:fetch "${name}" "${url}" "${digest}" >/dev/null || return 1
  done
}

# sources:roots [<settings>] prints "<name>\t<root>" for each package source in the cache, in the order of the
# settings file, kit.yml by default: <root> is the directory that holds its packages/. A source not fetched yet is
# left out; sources:fetch-all fetches it.
sources:roots() {
  local name url digest cache row
  local -a rows
  sources:rows rows "$@" || return 1
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r name url digest <<<"${row}"
    [[ -n "${name}" ]] || continue
    sources:cache "${digest}" cache
    if [[ -d "${cache}/packages" ]]
    then
      printf '%s\t%s\n' "${name}" "${cache}"
    elif [[ -d "${cache}/kit/packages" ]]
    then
      printf '%s\t%s\n' "${name}" "${cache}/kit"
    fi
  done
}

# sources:entries <file> <layer> prints each entry of sources: in the YAML file as one line of JSON, in order: a word
# as {word, layer}, and a map as {name, kind, url, layer, entry}, its kind kit-packages where it is left out, and
# entry the map as written. A file that does not exist has none.
sources:entries() {
  local file=$1 layer=$2 doc
  [[ -f "${file}" ]] || return 0
  doc=$(yaml:eval "${file}" 'sources entries' '.sources // []' --output-format=json --indent=0) || return 1
  jq --compact-output --arg layer "${layer}" '
    if type != "array" then {wrong: "sources is a list of sources", layer: $layer}
    else .[] | if type == "string" then {word: ., layer: $layer}
      elif type == "object" then {name: (.name // "" | tostring), kind: (.kind // "kit-packages" | tostring),
        url: (.url // "" | tostring), layer: $layer, entry: .}
      else {wrong: "an entry of sources is a map or a word, direct or off, not \(tojson)", layer: $layer} end
    end' <<<"${doc:-[]}"
}

# sources:environment prints each entry of KIT_SOURCES as one line of JSON, as sources:entries prints them, its
# layer env. An entry that is not direct, off, or <name>=<kind>:<url> is printed as {wrong}.
sources:environment() {
  local item name kind url
  local -a items
  [[ -n "${KIT_SOURCES:-}" ]] || return 0
  IFS=, read -r -a items <<<"${KIT_SOURCES}"
  for item in "${items[@]}"
  do
    item=${item#"${item%%[![:space:]]*}"}
    item=${item%"${item##*[![:space:]]}"}
    [[ -n "${item}" ]] || continue
    if [[ "${item}" == direct || "${item}" == off ]]
    then
      jq --null-input --compact-output --arg word "${item}" '{word: $word, layer: "env"}'
    elif [[ "${item}" =~ ^([^=]+)=([^:]+):(.+)$ ]]
    then
      name=${BASH_REMATCH[1]} kind=${BASH_REMATCH[2]} url=${BASH_REMATCH[3]}
      jq --null-input --compact-output --arg name "${name}" --arg kind "${kind}" --arg url "${url}" \
        '{name: $name, kind: $kind, url: $url, layer: "env", entry: {name: $name, kind: $kind, url: $url}}'
    else
      jq --null-input --compact-output --arg item "${item}" \
        '{wrong: "\($item) is not direct, off, or <name>=<kind>:<url>", layer: "env"}'
    fi
  done
}

# sources:check <entry>… checks the entries, each a line sources:entries prints, and says what is wrong with the
# first that is wrong, naming its layer's file, and returns 1: a word other than direct and off, a name off
# sources_name_pattern or used twice in one layer, a kind no provider answers, an entry with no url, a key its kind
# does not take, a package source outside kit.yml, and an expect that is no duration.
sources:check() {
  local line kinds kind keys allowed wrong file name
  local -A taken=()
  kinds=$(sources:kinds | paste -s -d ' ' -)
  for line in "$@"
  do
    [[ -n "${line}" ]] || continue
    kind=$(jq --raw-output '.kind // ""' <<<"${line}")
    allowed=''
    if [[ -n "${kind}" && " ${kinds} " == *" ${kind} "* ]]
    then
      keys=$(sources:call "${kind}" keys | paste -s -d ' ' -) || return 1
      allowed="name kind url expect ${keys}"
    fi
    wrong=$(jq --raw-output --arg kinds "${kinds}" --arg allowed "${allowed}" --arg pattern "${sources_name_pattern}" '
      if .wrong then .wrong
      elif .word then (if .word == "direct" or .word == "off" then empty
        else "\(.word) is no word of sources; a word is direct or off" end)
      elif (.name | test($pattern) | not) then
        "the source \(.name | tojson) needs a name of lowercase letters, digits, and dashes, starting with a letter"
      elif (.kind as $kind | $kinds | split(" ") | index([$kind]) | not) then
        "the source \(.name) has the kind \(.kind), which no provider answers; the kinds are \($kinds)"
      elif .url == "" then "the source \(.name) needs a url"
      elif .kind == "kit-packages" and .layer != "kit.yml" then
        "the source \(.name) is a package source, which kit.yml alone names; the pull path takes the other kinds"
      elif (.entry.expect // "" | tostring | test("^([0-9]+[smhd])?$") | not) then
        "the source \(.name) has expect: \(.entry.expect), which is no duration, as 30m, 12h, or 1d"
      else first(.entry | keys[] | select(. as $key | $allowed | split(" ") | index([$key]) | not)) as $key
        | "the source \(.name) has the key \($key), which a \(.kind) source does not take"
      end' <<<"${line}") || return 1
    if [[ -z "${wrong}" ]]
    then
      name=$(jq --raw-output '(.layer) + "\u001f" + (.name // .word)' <<<"${line}")
      if [[ -n "${taken[${name}]+x}" ]]
      then wrong="the source ${name#*$'\x1f'} is named twice"
      fi
      taken[${name}]=1
    fi
    [[ -n "${wrong}" ]] || continue
    case "$(jq --raw-output .layer <<<"${line}")" in
      (env) file=KIT_SOURCES ;;
      (workspace) file=$(sources:workspace-file) ;;
      (*) file=kit.yml ;;
    esac
    error "${file}: ${wrong}"
    return 1
  done
}

# sources:workspace-file prints the person's workspace file, whose sources: is the pull path's middle layer.
sources:workspace-file() {
  printf '%s/kit/workspaces.yml\n' "${XDG_CONFIG_HOME:-${HOME}/.config}"
}

# sources:layers prints every entry of the three layers, as sources:entries prints them, KIT_SOURCES first, then the
# workspace's, then kit.yml's, after checking them. kit.yml's package sources are left out, and with no other entry,
# kit.yml's layer is direct, its entry {"implied": true}.
sources:layers() {
  local line
  local -a env=() workspace=() repository=() artifacts=()
  list:read env sources:environment || return 1
  list:read workspace sources:entries "$(sources:workspace-file)" workspace || return 1
  list:read repository sources:entries kit.yml kit.yml || return 1
  sources:check "${env[@]}" "${workspace[@]}" "${repository[@]}" || return 1
  for line in "${repository[@]}"
  do
    [[ -z "${line}" || "${line}" == *'"kind":"kit-packages"'* ]] || artifacts+=("${line}")
  done
  (( ${#artifacts[@]} )) || artifacts=('{"word":"direct","layer":"kit.yml","entry":{"implied":true}}')
  printf '%s\n' "${env[@]}" "${workspace[@]}" "${artifacts[@]}" | sed '/^$/d'
}

# sources:path prints the pull path, one entry per line, apart by tabs: its name, its kind, its url, its layer (env,
# workspace, or kit.yml), and its entry's JSON. direct and off are their own names, with no kind or url. A name or
# direct met again is left out, and the path ends at the first off, which it prints.
sources:path() {
  local layers
  layers=$(sources:layers) || return 1
  jq --raw-output --slurp '
    reduce .[] as $e ({seen: {}, out: [], done: false};
      if .done then .
      else (($e.name // $e.word)) as $key
        | if .seen[$key] then .
          else .seen[$key] = true | .out += [$e] | (if $e.word == "off" then .done = true else . end) end
      end)
    | .out[] | [(.name // .word), (.kind // ""), (.url // ""), .layer, (.entry // {} | tojson)] | join("\t")' \
    <<<"${layers}"
}

# sources:direct <coordinate> prints where direct fetches the coordinate from, "<kind>\t<home>": the first kind, in
# name order, that serves the coordinate and has a home. A url: coordinate is its own address, with no kind. It
# returns 4 when no kind serves the coordinate at a home.
sources:direct() {
  local kind home
  local -a kinds
  if [[ "$1" == url:* ]]
  then
    printf '\t%s\n' "${1#url:}"
    return 0
  fi
  list:read kinds sources:kinds || return 1
  for kind in "${kinds[@]}"
  do
    sources:call "${kind}" serves "$1" || continue
    home=$(sources:call "${kind}" home "$1") || continue
    [[ -n "${home}" ]] || continue
    printf '%s\t%s\n' "${kind}" "${home}"
    return 0
  done
  return 4
}

# sources:artifact <coordinate> prints what the coordinate names, as a lock records it, apart from where it is: a
# url: coordinate as file:<its last part>, and any other as it is, since it names no place.
sources:artifact() {
  local url
  case "$1" in
    (url:*)
      url=${1#url:}
      url=${url%%[?#]*}
      printf 'file:%s\n' "${url##*/}"
      ;;
    (*) printf '%s\n' "$1"
  esac
}

# sources:find <coordinate> <hash> <file> [<path>] walks the pull path for the coordinate, writes the first hit whose
# hash is <hash> at <file>, and prints the hit's hash. With no hash, the first hit wins. A source whose kind does not
# serve the coordinate's form is passed by, as is one that does not have it. One that cannot be reached, or that
# serves other bytes, is named, and the walk goes on, so a mirror's other bytes are never installed. When nothing on
# the path served it, it says what each source said, and that the path ended at off where it did, and returns 1.
# <path> is what sources:path prints, read once for every artifact of a run; without it, the path is read.
sources:find() {
  local coordinate=$1 want=$2 file=$3 path name kind url entry line status have where artifact sources_name
  local sources_entry
  local -a said=()
  if (( $# > 3 ))
  then path=$4
  else path=$(sources:path) || return 1
  fi
  artifact=$(sources:artifact "${coordinate}")
  # a kind or a url can be empty, so each tab becomes \x1f, which read never runs together with the next
  while IFS=$'\x1f' read -r name kind url _ entry
  do
    [[ -n "${name}" ]] || continue
    sources_name=${name} sources_entry=${entry}
    case "${name}" in
      (off)
        said+=("the path ends at off")
        break
        ;;
      (direct)
        line=$(sources:direct "${coordinate}") || continue
        IFS=$'\x1f' read -r kind url <<<"${line//$'\t'/$'\x1f'}"
        where="direct (${url})"
        sources_name='' sources_entry='{}'
        ;;
      (*)
        [[ "${coordinate}" != url:* ]] || continue
        sources:call "${kind}" serves "${coordinate}" || continue
        where=${name}
    esac
    status=0
    rm -f "${file}"
    if [[ -z "${kind}" ]]
    then sources:get "${url}" "${file}" || status=$?
    else sources:call "${kind}" fetch "${url}" "${coordinate}" "${file}" || status=$?
    fi
    case "${status}" in
      (0) ;;
      (4)
        said+=("${where} does not have it")
        continue
        ;;
      (5)
        said+=("${where} blocks it")
        continue
        ;;
      (*)
        said+=("${where} could not be reached")
        continue
    esac
    if [[ -z "${kind}" ]]
    then have="sha256:$(file:digest "${file}")"
    else have=$(sources:call "${kind}" hash "${file}" "${coordinate}") || return 1
    fi
    if [[ -n "${want}" && "${have}" != "${want}" ]]
    then
      error "${where} serves ${artifact#file:} with the hash ${have}, not the ${want} the lock pins; kit passed" \
        "it by"
      said+=("${where} served other bytes")
      rm -f "${file}"
      continue
    fi
    printf '%s\n' "${have}"
    return 0
  done <<<"${path//$'\t'/$'\x1f'}"
  rm -f "${file}"
  if (( ${#said[@]} == 0 ))
  then said=("no source on it serves ${coordinate%%:*}: artifacts")
  fi
  error "no source on the pull path served ${artifact#file:}: $(IFS=';'; printf '%s' "${said[*]}" | sed 's/;/; /g');" \
    "see: kit tools sources"
  return 1
}

# sources:get <url> <file> fetches the url into the file, or prints it for a file of -, with the source's credentials:
# netrc's, and a bearer token from KIT_SOURCE_TOKEN_<NAME>, the name of the source being asked, sources_name, upper
# cased with each dash as _. It returns 4 when the source answers that it has no such thing, as an HTTP error, and 1
# when it cannot be reached. Fetching into a file, it sets sources_http to the HTTP status the source answered, so a
# provider can tell a refusal, as 403, from a miss.
sources:get() {
  local status=0 variable code
  local -a credentials=(--netrc-optional)
  if [[ -n "${sources_name:-}" ]]
  then
    variable=KIT_SOURCE_TOKEN_${sources_name//-/_}
    variable=${variable^^}
    [[ -z "${!variable:-}" ]] || credentials+=(--header "Authorization: Bearer ${!variable}")
  fi
  sources_http=''
  if [[ "$2" == - ]]
  then curl --fail --silent --location "${credentials[@]}" "$1" || status=$?
  else
    code=$(curl --fail --silent --location "${credentials[@]}" --output "$2" --write-out '%{http_code}' "$1") \
      || status=$?
    # shellcheck disable=SC2034  # read by a provider, as artifactory tells a 403 from a miss
    sources_http=${code}
  fi
  case "${status}" in
    (0) return 0 ;;
    (22) return 4 ;;
    (*) return 1
  esac
}

# sources:versions <coordinate> [<path>] prints the versions the first source on the pull path that serves the
# coordinate's form lists, one per line, as that source writes them: a release's tags, a module's versions. A source
# that serves none is passed by, as one that cannot be reached is, after it is named. It fails when none answered.
sources:versions() {
  local coordinate=$1 path name kind url entry line status listed sources_name sources_entry
  local -a said=()
  if (( $# > 1 ))
  then path=$2
  else path=$(sources:path) || return 1
  fi
  while IFS=$'\x1f' read -r name kind url _ entry
  do
    [[ -n "${name}" ]] || continue
    sources_name=${name} sources_entry=${entry}
    case "${name}" in
      (off) break ;;
      (direct)
        [[ "${coordinate}" != url:* ]] || continue
        line=$(sources:direct "${coordinate}") || continue
        IFS=$'\x1f' read -r kind url <<<"${line//$'\t'/$'\x1f'}"
        # shellcheck disable=SC2034  # read by the provider, as npm reads its scope
        sources_name='' sources_entry='{}'
        ;;
      (*) sources:call "${kind}" serves "${coordinate}" || continue
    esac
    status=0
    listed=$(sources:call "${kind}" versions "${url}" "${coordinate}") || status=$?
    case "${status}" in
      (0)
        printf '%s\n' "${listed}"
        return 0
        ;;
      (4) said+=("${name} serves none") ;;
      (*) said+=("${name} could not be reached")
    esac
  done <<<"${path//$'\t'/$'\x1f'}"
  (( ${#said[@]} )) || said=("no source on it serves ${coordinate%%:*}: artifacts")
  error "the versions of $(sources:artifact "${coordinate}") could not be read from the pull path:" \
    "$(IFS=';'; printf '%s' "${said[*]}" | sed 's/;/; /g'); see: kit tools sources"
  return 1
}

# sources:settings [<path>] prints the NAME=value lines each source on the pull path writes for a language's package
# manager, in the path's order: direct as each kind at its home, with {"direct": true} as its entry, and off as each
# kind with the url off and {"off": true}. GOPROXY, GOPRIVATE, and GONOSUMDB join their values with commas, as Go reads
# them; any other name is the first source's. A path that is direct alone, because no layer names a source, writes
# nothing, so a manager reads what its own settings say.
sources:settings() {
  local path name kind url entry line key value
  local -a order=() lines got kinds
  local -A values=()
  if (( $# ))
  then path=$1
  else path=$(sources:path) || return 1
  fi
  list:read kinds sources:kinds || return 1
  while IFS=$'\x1f' read -r name kind url _ entry
  do
    [[ -n "${name}" ]] || continue
    [[ "${entry}" != *'"implied":true'* ]] || continue
    lines=()
    case "${name}" in
      (direct | off)
        for kind in "${kinds[@]}"
        do
          if [[ "${name}" == off ]]
          then url=off
          else url=$(sources:call "${kind}" home) || continue
          fi
          [[ -n "${url}" ]] || continue
          list:read got sources:call "${kind}" settings "${url}" "{\"${name}\": true}" || return 1
          lines+=("${got[@]}")
        done
        ;;
      (*) list:read lines sources:call "${kind}" settings "${url}" "${entry:-{\}}" || return 1
    esac
    for line in "${lines[@]}"
    do
      [[ "${line}" == ?*=* ]] || continue
      key=${line%%=*} value=${line#*=}
      if [[ -z "${values[${key}]+x}" ]]
      then
        order+=("${key}")
        values[${key}]=${value}
      else
        case "${key}" in
          (GOPROXY | GOPRIVATE | GONOSUMDB) values[${key}]+=",${value}" ;;
          (*) ;;
        esac
      fi
    done
    [[ "${name}" != off ]] || break
  done <<<"${path//$'\t'/$'\x1f'}"
  for key in "${order[@]}"
  do
    printf '%s=%s\n' "${key}" "${values[${key}]}"
  done
}

# ----------------------------------------------------------------------------------------------------------------------
