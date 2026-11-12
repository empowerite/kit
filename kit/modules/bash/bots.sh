# shellcheck shell=bash
# bots.sh - the bots file, choosing a bot, and a bot's key (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# The bots a person has set up are public facts, kept in bots.yml under XDG_CONFIG_HOME, else ~/.config, in kit/:
#
#   bots:
#     - name: empowerite-jig      # the App's slug; the bot's login is empowerite-jig[bot]
#       host: github.com
#       app: 4906610              # the App ID
#       owner: empowerite         # the account that registered the App
#       jobs: [work, review]      # what it is set up to do: work, review, workflows
#
# The file is only ever edited with yq --inplace, so a person's comments stay. The App's private key is never in it:
# it is an item of the secret store, service kit-bot, account <host>/<app>, as modules/bash/secrets.sh keeps it.
#
# A bot belongs to the host of the primary remote, as the forge seam reaches the one API of that host, so a bot on
# another host is never chosen. Outside a repository, or where the primary remote names no host, the host is github.com.
#
# see: modules/bash/secrets.sh, modules/bash/forge.sh
use file forge secrets settings yaml

# bots_jobs lists the jobs a bot can be set up for, in the order a person is told them.
declare -gra bots_jobs=(work review workflows)

# bots:file prints the path of the bots file, which may not exist.
bots:file() { printf '%s\n' "${XDG_CONFIG_HOME:-${HOME}/.config}/kit/bots.yml"; }

# bots:json prints {bots: [...]} as the file holds them, one line, a bot with no jobs having work and review, and no
# bots where the file is missing or empty. It returns 1, saying what is wrong, for a file that does not parse or whose
# bots are not a list of {name, host, app}.
bots:json() {
  local file doc
  file=$(bots:file)
  [[ -f "${file}" ]] || { printf '{"bots":[]}\n'; return 0; }
  doc=$(yaml:json "${file}") || return 1
  jq --compact-output '
    if type == "null" then {bots: []}
    elif type == "object" and ((.bots // []) | type == "array")
      and all((.bots // [])[]; type == "object" and (.name | type == "string") and (.host | type == "string")
        and (.app | type == "number"))
    then {bots: [(.bots // [])[] | .jobs //= ["work", "review"]]}
    else error("bad")
    end' <<<"${doc:-null}" 2>/dev/null \
    || {
      error "${file} is not {bots: [{name, host, app, owner, jobs}]}; fix it, or remove it and run: kit bots --add="
      return 1
    }
}

# bots:primary-host prints the host of the primary remote, or nothing outside a repository or for a remote on this
# machine. bots:host prints it, or github.com where there is none.
bots:primary-host() { forge:host "$(forge:primary 2>/dev/null)" 2>/dev/null || true; }
bots:host() {
  local host
  host=$(bots:primary-host)
  printf '%s\n' "${host:-github.com}"
}

# bots:repository prints owner/name, the path of the primary remote's URL less a trailing .git, or nothing where there
# is no primary remote with a URL.
bots:repository() {
  local url path
  url=$(git remote get-url --end-of-options "$(forge:primary 2>/dev/null)" 2>/dev/null) || return 0
  case "${url}" in
    (*://*) path=${url#*://}; path=${path#*/} ;;
    (*:*) path=${url#*:} ;;
    (*) return 0
  esac
  path=${path%/}
  printf '%s\n' "${path%.git}"
}

# bots:names <json> prints the names of the bots in {bots: […]}, comma-separated.
bots:names() { jq --raw-output '[.bots[].name] | join(", ")' <<<"$1"; }

# bots:choose [<name>] prints one bot's JSON, {name, host, app, owner, jobs}: the named one; else the one bot: in
# kit.yml names; else the only bot on the primary remote's host. It says why not, and returns 2, for a name no bot
# has, no bot, two on the host, and a bot on another host than the primary remote's, which the forge seam could not
# reach.
bots:choose() {
  local name=${1:-} doc host bot where
  local -a onhost
  doc=$(bots:json) || return 1
  host=$(bots:host)
  [[ -n "${name}" ]] || name=$(settings:value bot)
  if [[ -n "${name}" ]]
  then
    bot=$(jq --compact-output --arg name "${name}" '[.bots[] | select(.name == $name)] | first // empty' <<<"${doc}")
    if [[ -z "${bot}" ]]
    then
      error "no bot is named ${name}; the bots are: $(bots:names "${doc}")" \
        "(run: kit bots --add=${name} --app=<id> --key=<source>)"
      return 2
    fi
  else
    # shellcheck disable=SC2016  # $host is jq's
    list:read onhost jq --compact-output --arg host "${host}" '.bots[] | select(.host == $host)' <<<"${doc}" || return 1
    case "${#onhost[@]}" in
      (0)
        where=$(jq --raw-output '[.bots[].host] | unique | join(", ")' <<<"${doc}")
        error "no bot is on ${host}, the primary remote's host${where:+ (the bots are on ${where})};" \
          "run: kit bots --add=<name> --app=<id> --key=<source>"
        return 2
        ;;
      (1) bot=${onhost[0]} ;;
      (*)
        error "${#onhost[@]} bots are on ${host}: $(jq --raw-output --arg host "${host}" \
          '[.bots[] | select(.host == $host) | .name] | join(", ")' <<<"${doc}"); name one, or set bot: in kit.yml"
        return 2
    esac
  fi
  if [[ "$(jq --raw-output .host <<<"${bot}")" != "${host}" ]]
  then
    error "the primary remote is on ${host}, but $(jq --raw-output .name <<<"${bot}") is on" \
      "$(jq --raw-output .host <<<"${bot}"); name a bot on ${host}"
    return 2
  fi
  printf '%s\n' "${bot}"
}

# bots:key <bot-json> <file> writes the bot's PEM to <file>, mode 600, from the secret store. It says how to add the
# key, and returns 1, where the store has none.
bots:key() {
  local name host app backend
  name=$(jq --raw-output .name <<<"$1")
  host=$(jq --raw-output .host <<<"$1")
  app=$(jq --raw-output .app <<<"$1")
  backend=$(secrets:backend) || return 1
  : > "$2" && chmod 600 "$2" || return 1
  if ! secrets:get kit-bot "${host}/${app}" > "$2"
  then
    : > "$2"
    error "no key for ${name} in the ${backend}; run: kit bots --add=${name} --app=${app} --key=<file>"
    return 1
  fi
}

# bots:jobs-ok <job>… succeeds when every job is one a bot can be set up for. It says which is not, and returns 2.
bots:jobs-ok() {
  local job all=${bots_jobs[*]}
  for job in "$@"
  do
    [[ " ${bots_jobs[*]} " == *" ${job} "* ]] && continue
    error "'${job}' is not a job; the jobs are: ${all// /, }"
    return 2
  done
}

# bots:jobs-need <job>… prints, as one JSON object, the permissions the jobs need: metadata read always, and for each
# job what the design lists, write covering read.
bots:jobs-need() {
  bots:jobs-ok "$@" || return 2
  jq --null-input --compact-output --sort-keys '
    {work: {contents: "write", issues: "write", pull_requests: "write", checks: "read", statuses: "read",
            actions: "read"},
     review: {pull_requests: "write", contents: "read", checks: "read"},
     workflows: {workflows: "write"}} as $table
    | reduce ($ARGS.positional[] | $table[.] | to_entries[]) as $need ({metadata: "read"};
        if .[$need.key] == "write" then . else .[$need.key] = $need.value end)' --args "$@"
}

# bots:forge <call> <argument>… makes a bot-accounts call of the forge seam, as forge:bot-<call>. Where the primary
# remote names no host, outside a repository, the seam has no forge to ask, so GitHub's provider is called directly.
bots:forge() {
  local call=$1 status=0
  shift
  if [[ -z "$(bots:primary-host)" ]]
  then
    use forge/github
    "forge/github:bot-${call}" "$@"
    return
  fi
  "forge:bot-${call}" "$@" || status=$?
  if (( status == 3 ))
  then
    error "the forge at $(bots:host) has no bot accounts kit can use"
    return 1
  fi
  return "${status}"
}

# bots:list prints {file, bots: [{name, host, app, owner, jobs, key}]}, key being true where the secret store holds the
# bot's key. A bot's key is read only where there are bots, so a machine with no store lists none.
bots:list() {
  local doc bot file keyed_json
  local -a keyed=()
  doc=$(bots:json) || return 1
  file=$(bots:file)
  while IFS= read -r bot
  do
    [[ -n "${bot}" ]] || continue
    if secrets:get kit-bot "$(jq --raw-output '"\(.host)/\(.app)"' <<<"${bot}")" >/dev/null 2>&1
    then keyed+=(true)
    else keyed+=(false)
    fi
  done < <(jq --compact-output '.bots[]' <<<"${doc}")
  [[ "${#keyed[@]}" -eq 0 ]] || secrets:backend >/dev/null || return 1
  keyed_json=$(printf '%s\n' "${keyed[@]}" | jq --slurp --compact-output .)
  jq --compact-output --arg file "${file}" --argjson keyed "${keyed_json}" \
    '{file: $file, bots: [.bots | to_entries[] | .value + {owner: (.value.owner // null), key: $keyed[.key]} | {name, host, app, owner, jobs, key}]}' <<<"${doc}"
}

# bots:text <list> prints the document bots:list made as a table, one bot a line, or says there are none.
bots:text() {
  jq --raw-output '
    def pad($w): . + (" " * ($w - length));
    if (.bots | length) == 0
    then "no bots in \(.file); run: kit bots --add=<name> --app=<id> --key=<source>"
    else
      [.bots[] | [.name, .host, (.app | tostring), (.owner // "-"), (.jobs | join(",")),
        (if .key then "key kept" else "no key" end)]] as $rows
      | [range(0; 6) as $c | [$rows[][$c] | length] | max] as $w
      | $rows[] | [range(0; 6) as $c | .[$c] | pad($w[$c])] | join("  ") | rtrimstr(" ")
    end' <<<"$1"
}

# bots:read-key <source> <file> writes the PEM that <source> names to <file>, with LF line ends: <source> is a PEM
# file, or keychain:<service>/<account>, an item another tool stored. It says what is wrong, and returns 2, for a
# source that names nothing.
bots:read-key() {
  local source=$1 rest service account
  case "${source}" in
    (keychain:*)
      rest=${source#keychain:}
      service=${rest%%/*}
      account=${rest#*/}
      if [[ "${rest}" != */* || -z "${service}" || -z "${account}" ]]
      then
        error "'${source}' is no keychain item; write keychain:<service>/<account>"
        return 2
      fi
      if ! secrets:foreign "${service}" "${account}" | tr -d '\r' > "$2" || [[ ! -s "$2" ]]
      then
        error "the secret store holds no item ${service}/${account}; name a PEM file, or an item that is there"
        return 2
      fi
      ;;
    (*)
      if [[ ! -f "${source}" || ! -r "${source}" ]]
      then
        error "${source} is not a readable file; --key is a PEM file, or keychain:<service>/<account>"
        return 2
      fi
      tr -d '\r' < "${source}" > "$2"
  esac
}

# bots:add <name> <app> <source> [<job>…] adopts the GitHub App <app>, whose key <source> names, as the bot <name>,
# which is the App's slug. The key is proved before anything is stored: it signs for the App, and mints a token, which
# is revoked at once, from the installation on this repository, else the App's first installation. Then the key goes
# to the secret store and the bot to the bots file, replacing a bot of that name. The jobs are work and review without
# any. It returns 2 for a usage error, and 1 when the proof or a write failed, having stored nothing.
bots:add() {
  local name=$1 app=$2 source=$3 pem status=0
  local -a jobs
  shift 3
  jobs=("$@")
  (( ${#jobs[@]} )) || jobs=(work review)
  if [[ ! "${name}" =~ ^[A-Za-z0-9][A-Za-z0-9-]*$ ]]
  then
    error "'${name}' is not an App's slug: letters, digits, and -"
    return 2
  fi
  if [[ ! "${app}" =~ ^[0-9]+$ ]]
  then
    error "'${app}' is not an App ID: it is a number"
    return 2
  fi
  bots:jobs-ok "${jobs[@]}" || return 2
  secrets:backend >/dev/null || return 1
  bots:json >/dev/null || return 1
  file:temp pem bots-key || return 1
  chmod 600 "${pem}"
  bots:read-key "${source}" "${pem}" && bots:adopt "${name}" "${app}" "${pem}" "${jobs[@]}" || status=$?
  rm -f "${pem}"
  return "${status}"
}

# bots:adopt <name> <app> <pem> <job>… is bots:add after the key is read: the proof, then the key and the bot kept.
bots:adopt() {
  local name=$1 app=$2 pem=$3 host app_json slug owner html had=false
  shift 3
  host=$(bots:host)
  if ! app_json=$(bots:forge app "${app}" "${pem}")
  then
    error "App ${app} did not accept that key; nothing was stored"
    return 1
  fi
  slug=$(jq --raw-output .slug <<<"${app_json}")
  owner=$(jq --raw-output .owner <<<"${app_json}")
  html=$(jq --raw-output .html_url <<<"${app_json}")
  if [[ "${slug}" != "${name}" ]]
  then
    error "App ${app} is ${slug}, not ${name}, and the bot's name is its slug, so its login is right;" \
      "run: kit bots --add=${slug} --app=${app} --key=<source>"
    return 2
  fi
  bots:prove "${app}" "${pem}" "${html}" || return 1
  secrets:get kit-bot "${host}/${app}" >/dev/null 2>&1 && had=true
  secrets:put kit-bot "${host}/${app}" < "${pem}" || return 1
  if ! bots:write "${name}" "${host}" "${app}" "${owner}" "$@"
  then
    [[ "${had}" == true ]] || secrets:delete kit-bot "${host}/${app}" || true  # the write's error is what to report
    return 1
  fi
  printf 'added %s: App %s on %s, for %s; its key is in the %s\n' "${name}" "${app}" "${host}" "$*" \
    "$(secrets:backend)"
}

# bots:reaches <app> <pem> <owner/repo> prints the App's installation on the repository, {id, permissions,
# repository_selection, html_url}, or nothing where the App has none there, which is no failure. It returns 1, with the
# forge's reason on stderr, where the lookup itself failed.
bots:reaches() {
  local errors installation=''
  file:temp errors bots-errors || return 1
  installation=$(bots:forge installation "$1" "$2" "$3" 2> "${errors}") || true  # 1 and silent: none
  if [[ -s "${errors}" ]]
  then
    cat "${errors}" >&2
    return 1
  fi
  [[ -z "${installation}" ]] || printf '%s\n' "${installation}"
}

# bots:prove <app> <pem> <html_url> mints an installation token with the App's key, narrowed to this repository where
# the App is installed on it, else, with bot-proof's token of metadata alone, from the App's first installation, and
# revokes it. With no installation at all it names the App's install page and returns 1; a lookup that failed returns 1
# saying why.
bots:prove() {
  local app=$1 pem=$2 html=$3 repo installation='' installations reached='' token id
  repo=$(bots:repository)
  if [[ -n "${repo}" ]]
  then
    installation=$(bots:reaches "${app}" "${pem}" "${repo}") || return 1
    [[ -z "${installation}" ]] || reached=${repo}
  fi
  if [[ -z "${installation}" ]]
  then
    installations=$(bots:forge installations "${app}" "${pem}") || return 1
    installation=${installations%%$'\n'*}
  fi
  if [[ -z "${installation}" ]]
  then
    error "App ${app} is installed nowhere; install it at ${html}/installations/new, then run this again"
    return 1
  fi
  id=$(jq --raw-output .id <<<"${installation}")
  if [[ -n "${reached}" ]]
  then
    token=$(bots:forge token "${app}" "${pem}" "${id}" "${reached}") || return 1
  else
    token=$(bots:forge proof "${app}" "${pem}" "${id}") || return 1
  fi
  bots:forge revoke "$(jq --raw-output .token <<<"${token}")" || return 1
}

# bots:check <bot-json> prints {name, installation, page, repository, reached, lacks, extra} in one line: the bot's
# App against its jobs. reached is true where the App's installation reaches this repository, false where it does not,
# and null outside a repository, which judges none. The permissions judged are the installation's where it reaches the
# repository, else the App's own, and installation is the one reaching it, else the App's first, or null with none.
# page is the installation's settings page, else the App's page to install it. lacks maps each job to the permissions
# it needs that are not held, as "<permission>: <level>"; extra lists each permission held beyond what every job needs,
# metadata aside, a write held where read is needed being "<permission>: write". It returns 1, saying why, where the
# key or a call fails.
bots:check() {
  local pem app repo installation='' installations reached=null held app_json html job needs='{}' status=0
  local -a jobs
  app=$(jq --raw-output .app <<<"$1")
  list:read jobs jq --raw-output '.jobs[]' <<<"$1" || return 1
  file:temp pem bots-key || return 1
  bots:key "$1" "${pem}" || return 1
  app_json=$(bots:forge app "${app}" "${pem}") || status=$?
  if (( status == 0 ))
  then
    html=$(jq --raw-output .html_url <<<"${app_json}")
    held=$(jq --compact-output .permissions <<<"${app_json}")
    repo=$(bots:repository)
  fi
  if (( status == 0 )) && [[ -n "${repo}" ]]
  then
    installation=$(bots:reaches "${app}" "${pem}" "${repo}") || status=$?
    if [[ -n "${installation}" ]]
    then
      reached=true
      held=$(jq --compact-output .permissions <<<"${installation}")
    else reached=false
    fi
  fi
  if (( status == 0 )) && [[ "${reached}" != true ]]
  then
    installations=$(bots:forge installations "${app}" "${pem}") && installation=${installations%%$'\n'*} || status=$?
  fi
  rm -f "${pem}"
  (( status == 0 )) || return "${status}"
  for job in "${jobs[@]}"
  do
    needs=$(jq --compact-output --arg job "${job}" --argjson need "$(bots:jobs-need "${job}")" '.[$job] = $need' \
      <<<"${needs}") || return 1
  done
  # shellcheck disable=SC2016  # $bot, $held, $needs, and the rest are jq's
  jq --null-input --compact-output --argjson bot "$1" --argjson held "${held}" --argjson needs "${needs}" \
    --arg repo "${repo}" --arg html "${html}" --argjson reached "${reached}" \
    --argjson installation "${installation:-null}" '
    def rank: if . == null then 0 else {read: 1, write: 2, admin: 3}[.] // 0 end;
    ([$needs[] | to_entries[]] | group_by(.key) | map({key: .[0].key, value: (map(.value) | max_by(rank))})
      | from_entries) as $all
    | {name: $bot.name,
       installation: ($installation | if . then {id, repository_selection, html_url} else null end),
       page: ($installation.html_url // "\($html)/installations/new"),
       repository: ($repo | if . == "" then null else . end),
       reached: $reached,
       lacks: ($needs | to_entries
         | map({key, value: [.value | to_entries[] | select(($held[.key] | rank) < (.value | rank))
             | "\(.key): \(.value)"]})
         | map(select(.value | length > 0)) | from_entries),
       extra: [$held | to_entries | sort_by(.key)[] | select(.key != "metadata" and (.value | rank) > ($all[.key] | rank))
         | "\(.key): \(.value)"]}'
}

# bots:check-text <check> prints the document bots:check made as sentences, one thing to a line.
bots:check-text() {
  jq --raw-output '
    "\(.name): " + (
      if .reached == true then "its installation reaches \(.repository)"
      elif .reached == false then "its installation does not reach \(.repository); add the repository on \(.page)"
      elif .installation == null then "the App is installed nowhere; install it at \(.page)"
      else "no repository here, so its permissions are judged alone" end),
    (.lacks | to_entries[] | "the \(.key) job lacks: \(.value | join(", "))"),
    (if (.extra | length) > 0 then "no job needs: \(.extra | join(", "))" else empty end),
    (if (.lacks | length) == 0 and (.extra | length) == 0 then "its permissions match its jobs" else empty end)' <<<"$1"
}

# bots:token <bot-json> prints {token, expires_at}, an installation token narrowed to this repository, and revokes
# nothing: the caller has it to use. It returns 1, saying what to do, outside a repository, and where the
# installation does not reach this repository, which names kit bots --check.
bots:token() {
  local pem app repo installation='' status=0
  repo=$(bots:repository)
  if [[ -z "${repo}" ]]
  then
    error "a token is narrowed to one repository, and there is no repository here; run this inside one"
    return 1
  fi
  app=$(jq --raw-output .app <<<"$1")
  file:temp pem bots-key || return 1
  bots:key "$1" "${pem}" || return 1
  installation=$(bots:reaches "${app}" "${pem}" "${repo}") || status=$?
  if (( status == 0 )) && [[ -z "${installation}" ]]
  then
    error "$(jq --raw-output .name <<<"$1")'s installation does not reach ${repo}; run: kit bots --check"
    status=1
  fi
  (( status != 0 )) || bots:forge token "${app}" "${pem}" "$(jq --raw-output .id <<<"${installation}")" "${repo}" \
    || status=$?
  rm -f "${pem}"
  return "${status}"
}

# bots:write <name> <host> <app> <owner> <job>… sets the bot's entry in the bots file, in place if it is there, else
# at the end, and makes the file, with its directory, where there is none. yq keeps the file's comments.
bots:write() {
  local file expression
  file=$(bots:file)
  mkdir -p "$(dirname "${file}")" || return 1
  if [[ ! -f "${file}" ]]
  then
    printf '# bots.yml - the bots that act for you; kit reads and edits it, and keeps your comments\n' > "${file}" \
      || return 1
  fi
  # shellcheck disable=SC2016  # $entry is yq's
  expression='{"name": strenv(kit_bot_name), "host": strenv(kit_bot_host), "app": env(kit_bot_app),
    "owner": strenv(kit_bot_owner), "jobs": (strenv(kit_bot_jobs) | from_yaml)} as $entry | .bots = (.bots // [])'
  if [[ "$(bots:json | jq --arg name "$1" '[.bots[] | select(.name == $name)] | length')" == 0 ]]
  then
    # shellcheck disable=SC2016  # $entry is yq's
    expression+=' | .bots += [$entry]'
  else
    # shellcheck disable=SC2016  # $entry is yq's
    expression+=' | (.bots[] | select(.name == strenv(kit_bot_name))) = $entry'
  fi
  kit_bot_name=$1 kit_bot_host=$2 kit_bot_app=$3 kit_bot_owner=$4 kit_bot_jobs="[$(IFS=,; printf '%s' "${*:5}")]" \
    yq --inplace "${expression}" "${file}"
}

# bots:remove <name> forgets the bot, and deletes its key from the secret store; the App stays on the forge. It says
# which bots there are, and returns 2, for a name no bot has. The bot stays in the file where the key could not be
# deleted, so the command can be run again.
bots:remove() {
  local doc bot
  doc=$(bots:json) || return 1
  bot=$(jq --compact-output --arg name "$1" '[.bots[] | select(.name == $name)] | first // empty' <<<"${doc}")
  if [[ -z "${bot}" ]]
  then
    error "no bot is named $1; the bots are: $(bots:names "${doc}")"
    return 2
  fi
  secrets:delete kit-bot "$(jq --raw-output '"\(.host)/\(.app)"' <<<"${bot}")" || return 1
  kit_bot_name=$1 yq --inplace '.bots |= map(select(.name != strenv(kit_bot_name)))' "$(bots:file)"
  printf 'removed %s; its key is deleted, and its App stays on the forge\n' "$1"
}
# ----------------------------------------------------------------------------------------------------------------------
