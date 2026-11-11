# shellcheck shell=bash
# serve.sh - the Model Context Protocol server that kit serve runs on stdio (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported, and kit_bash_path set, by modules/bash/1st.sh
#
# serve:main reads one JSON-RPC message per line on stdin and answers each in order, one line on stdout per answer.
# Nothing else reaches stdout: a verb a tool runs has its stdout and stderr captured, and every log line goes to
# stderr. A message without an id is a notification, and gets no answer, even when it fails. A request whose params is
# there and is neither an object nor null is refused with -32602 before any handler runs; params null is no params. A
# line that holds a NUL byte is not JSON, and is answered with -32700. Nothing a client sent, beyond the verb's own
# words a call's arguments become, and nothing a verb printed, is a word of a command line or a variable of the
# environment: each reaches jq on stdin or from a file, and a value an error quotes is cut to 200 characters.
#
# The server answers two eras of the protocol, as the specification lets a dual-era server do:
#
#   - 2026-07-28, which has no session. A client may ask server/discover, and every other request names its version
#     in params._meta["io.modelcontextprotocol/protocolVersion"]; a request that names none is refused with -32602.
#   - The revisions before it, which a client selects by opening with initialize. From then on a request needs no
#     _meta, and ping is answered.
#
# A tool is a verb, in one of three groups, and its annotations say which, so a client can ask before it calls one:
#
#   - context, flow, help, kinds, packages, syntax, and version read only
#   - check, lint, toolchains, and tools run what the repository's files name: check each generated region's command,
#     lint each rule's linter, and toolchains and tools the toolchain's own definition, such as mise.toml or a flake
#   - assign, close, gather, settle, unassign, and work act on issues and pull requests
#
# A tool's description is the verb's purpose and the usage lines it offers. The verb's whole page is a resource,
# kit://help/<verb>. Its input is typed, built from the verb's syntax by modules/jq/serve.jq: a property for each
# flag, option, and operand, and one for a line's leading word. A call's arguments become the verb's words, and a name
# the tool does not know is refused with -32602, as is a request with a NUL escape anywhere, which no command line
# carries.
# A call runs kit <verb> <word>… in a new process with stdin at end of file, so the verb cannot read the protocol
# stream.
#
# A tool that reads only runs no command the repository's files name, takes no word that acts, reads no path outside
# the repository, changes no file git sees, and never fetches. Of the workflow tools, only work writes the repository,
# as it makes a branch, a worktree, and a pull request:
#
#   - The server places .kit/ as it starts, and again between messages when kit.yml changed. A call does not place
#     anything; kit_serve_hydrated tells the verb so. While a changed kit.yml cannot be placed, every call and read
#     is refused with -32603, which names kit hydrate, and the server places again before each message.
#   - kit flow is always given --no-fetch, so incoming reads the refs as they are.
#   - A word that acts is never offered: packages, toolchains, and tools are listings only, and context's hook mode,
#     --claude, is the harness's own. An operand's value that is a word of the verb's syntax is refused, offered or
#     not, and so is a value off its property's enum.
#   - A path stays inside the repository: each word given to context, kinds, or lint that is not a flag is refused
#     when it leads outside the root, or when the root cannot be read, and flow does not offer --workspace, which
#     reads other clones.
#
# A tool's verb still writes the kit's log, and a verb's cache under .kit/cache/, which git never sees.
#
# Before each message, the server looks for a file in its release's bin/ and modules/, or in a module directory of
# the repository's own, changed since it started. When one has, it starts itself again over the same stdin, from the
# changed code, and that process answers the message: an edit to a verb, a module, or the server itself takes effect
# on the next message, with no reconnect. A changed bash file that does not parse keeps the running code, and says
# so on stderr, until the next change. The new process tells the client that the tools and the resources may have
# changed, with notifications/tools/list_changed and notifications/resources/list_changed, so it asks for them again.
# A .kit/.gitignore changed since, which kit hydrate rewrites when what it places changes, sends the second alone.
#
# A resource is one of the kit's files under .kit/: kit://1st.md, kit://workflow.md with the workflow package,
# kit://rules/<kind> and kit://styles/<kind> for each rule and style, and kit://kinds/<file> for each kinds file, which
# may define several kinds. Each tool's whole help page is a resource too, kit://help/<verb>, as kit help <verb>
# prints it.
#
# Each message it answers is a line in .kit/logs/mcp-YYYY-MMDD.log, as modules/bash/log.sh writes it: the request's
# id=, its method=, and its outcome=, which is ok, error=<code> for a refusal, status=<status> for a tool whose verb
# failed, or none for a notification. A resource's line ends with its uri=, and a tool's with the command it ran,
# kit <verb> <arg>…, quoted so it can be pasted back. kit serve adds a line when it starts and one when stdin ends.
#
# see: kit/bin/kit-serve, docs/serve.md, https://modelcontextprotocol.io/specification/2026-07-28

use context help release syntax log file materialize

# The protocol revisions the server answers, the newest first. The first is the one without a session.
declare -gra serve_versions=(2026-07-28 2025-11-25 2025-06-18 2025-03-26)

# The tools, in three groups: the seven verbs that read only and the four that run what the repository's files name,
# check, lint, toolchains, and tools, together by name; then the six workflow verbs, which act on issues and pull
# requests. serve_acts holds the annotations of every tool that does not read only.
declare -gra serve_tools=(check context flow help kinds lint packages syntax toolchains tools version
  assign close gather settle unassign work)

# The tools that ask the forge, over the network, rather than reading the repository alone.
declare -gra serve_open_world=(flow assign close gather settle unassign work)

# The annotations of the tools that do not read only. check, lint, toolchains, and tools run what the repository's
# files name; each workflow tool changes something. A destructive one closes issues or takes people off them; an
# idempotent one called again with the same arguments changes nothing more.
declare -grA serve_acts=(
  [check]='{"readOnlyHint": false, "destructiveHint": false, "idempotentHint": true}'
  [lint]='{"readOnlyHint": false, "destructiveHint": false, "idempotentHint": true}'
  [toolchains]='{"readOnlyHint": false, "destructiveHint": false, "idempotentHint": true}'
  [tools]='{"readOnlyHint": false, "destructiveHint": false, "idempotentHint": true}'
  [assign]='{"readOnlyHint": false, "destructiveHint": false, "idempotentHint": true}'
  [close]='{"readOnlyHint": false, "destructiveHint": true, "idempotentHint": true}'
  [gather]='{"readOnlyHint": false, "destructiveHint": false, "idempotentHint": false}'
  [settle]='{"readOnlyHint": false, "destructiveHint": true, "idempotentHint": true}'
  [unassign]='{"readOnlyHint": false, "destructiveHint": true, "idempotentHint": true}'
  [work]='{"readOnlyHint": false, "destructiveHint": false, "idempotentHint": true}'
)

# The tools offered as a listing only. Each word would act, or reach the network, so a tool does not offer it: a line
# that needs one is not offered, and an optional part that holds one is taken out. Each word is a literal or --flag.
declare -grA serve_listing_only=(
  [packages]='select add remove'
  [toolchains]='choose'
  [tools]='install update path run --outdated'
)

# What each listing-only tool's words do, as the refusal of a call that names one says.
declare -grA serve_listing_acts=(
  [packages]='select, add, or remove packages'
  [toolchains]='choose a toolchain'
  [tools]='install, update, or run the tools, or ask which are outdated'
)

# The words a tool does not offer for another reason: context's hook mode is the harness's own. A workspace, which
# reads other clones, is a kit option, before the verb, so no tool offers one.
declare -grA serve_hidden=(
  [context]='--claude'
)

# The tools whose operands are paths. A call to one is refused when a word it makes, but a flag, leads outside the
# repository.
declare -gra serve_paths=(context kinds lint)

# The flags the server gives a tool's verb itself, on any line that takes them, and never offers: flow never fetches.
declare -grA serve_implied=(
  [flow]='no-fetch'
)

# The name of the property that holds a line's leading word, where the verb has such lines.
declare -grA serve_leading=(
  [flow]='stage'
)

# serve_era is legacy once a client opens with initialize, and empty until then. serve_tool_list holds the tools/list
# result's tools array once it is built. serve_specs holds each tool's spec and serve_opts what the server says of it,
# as serve.jq reads them, once each is built.
serve_era=''
serve_tool_list=''
declare -gA serve_specs=() serve_opts=()

# serve_log_fields holds the id and method of the message being answered, for its log line, serve_outcome how it
# ended, and serve_log_last what the line ends with: a resource's uri, or a tool's command. Each message starts
# with all three empty.
declare -ga serve_log_fields=()
serve_outcome=''
serve_log_last=''

serve_instructions='kit holds the rules this repository asks every agent to follow. Read the resource kit://1st.md'
serve_instructions+=' first. Before you change a file, call the context tool with its path, and read every rule it names.'

# serve:main answers each message on stdin, in order, until stdin ends, and logs each one.
serve:main() {
  local line
  serve:watch
  serve:resume
  serve:guard-nul
  while IFS= read -r line || [[ -n "${line}" ]]
  do
    [[ -n "${line//[[:space:]]/}" ]] || continue
    serve:reload "${line}"
    serve:answer "${line}"
  done
}

# serve:guard-nul puts jq between stdin and the server, once for the stream. bash's read drops a 0x00 byte without a
# word, so a line that holds one would be served as the line without it. jq reads each line as it arrives and passes
# it on, and passes a line that holds a NUL byte as a line of the one byte 0x01, which serve:handle answers as a
# parse error. kit_serve_guarded says the filter is there, so the server started again over the same stdin keeps
# reading from it.
serve:guard-nul() {
  [[ -z "${kit_serve_guarded:-}" ]] || return 0
  export kit_serve_guarded=1
  exec < <(jq --raw-input --raw-output --unbuffered 'if explode | any(. == 0) then "\u0001" else . end')
}

# serve:answer <line> answers one message, and logs it.
serve:answer() {
  local started
  log:now started
  serve_log_fields=() serve_outcome='' serve_log_last=''
  serve:handle "$1"
  log:call mcp "${started}" "pid=$$" "${serve_log_fields[@]}" "outcome=${serve_outcome:-none}" \
    ${serve_log_last:+"${serve_log_last}"}
}

# serve_marker is a file whose modification time is when the server started, or last looked for a change and found
# one it would not take; serve_scratch is the directory that holds it.
serve_marker='' serve_scratch=''

# serve:watch makes the marker the server compares the files it runs with against.
serve:watch() {
  file:scratch serve_scratch
  serve_marker=${serve_scratch}/started
  : > "${serve_marker}"
}

# serve:watched prints the directories whose files the server runs: its release's bin/ and modules/, and each
# directory of the module path outside the release, which holds the repository's own modules.
serve:watched() {
  local dir release
  release=$(cd "${kit_root}" && pwd -P)
  printf '%s\n' "${release}/bin" "${release}/modules"
  for dir in "${kit_bash_path[@]}"
  do
    [[ "${dir}" == "${release}" || "${dir}" == "${release}"/* ]] || printf '%s\n' "${dir}"
  done
}

# serve:changed prints each file in the watched directories changed since the marker, one per line.
serve:changed() {
  local -a dirs
  readarray -t dirs < <(serve:watched)
  # A watched directory that is gone has nothing changed in it; find says so with its status.
  find "${dirs[@]}" -type f -newer "${serve_marker}" 2>/dev/null || true
}

# serve:reload <line> starts the server again from the changed code, to answer <line>, when a file it runs has
# changed and every changed bash file parses. It passes the era a client chose with initialize in kit_serve_era, and
# in kit_serve_pending a file that holds the line, since a line may be longer than the environment carries.
# Otherwise it returns, and the running server answers <line>. Before that, it places .kit/ again when kit.yml
# changed, since a call does not place anything; a placing that fails sets serve_unplaced until one succeeds. It sends
# notifications/resources/list_changed when .kit/.gitignore changed, and says so on stderr when a file did not parse.
serve:reload() {
  local file now pending
  local -a changed broken=()
  readarray -t changed < <(serve:changed)
  if (( ${#changed[@]} == 0 ))
  then
    local placed=false
    if [[ kit.yml -nt "${serve_marker}" ]]
    then
      # Placing writes its steps to stderr. One that fails leaves the marker as it is, so the next message places
      # again, and until one places, serve:placed refuses every call and every read the old .kit/ would answer.
      if ! materialize:hydrate >&2
      then
        serve_unplaced=true
        return 0
      fi
      serve_unplaced=false
      placed=true
    fi
    if [[ .kit/.gitignore -nt "${serve_marker}" ]]
    then
      : > "${serve_marker}"
      serve:notify notifications/resources/list_changed
    elif [[ "${placed}" == true ]]
    then
      : > "${serve_marker}"
    fi
    return 0
  fi
  for file in "${changed[@]}"
  do
    case "${file}" in
      (*.sh | */bin/kit | */bin/kit-*) bash -n "${file}" 2>/dev/null || broken+=("${file}")
    esac
  done
  if (( ${#broken[@]} > 0 ))
  then
    : > "${serve_marker}"
    warn "kit serve: ${broken[*]} does not parse; the server keeps the code it runs until the next change"
    log:now now
    log:call mcp "${now}" "pid=$$" event=reload-refused "${broken[@]}"
    return 0
  fi
  # The file outlives the exec, which runs no sweep; the new process's serve:resume removes it.
  if ! { file:temp pending pending && printf '%s\n' "$1" >"${pending}"; }
  then
    : > "${serve_marker}"
    warn "kit serve: the line could not be kept for a new process; the server keeps the code it runs"
    return 0
  fi
  log:now now
  log:call mcp "${now}" "pid=$$" event=reload
  rm -rf "${serve_scratch}"
  export kit_serve_era=${serve_era} kit_serve_pending=${pending}
  exec "${kit_root}/bin/kit-serve"
}

# serve:resume picks up where the server it replaced left off, when kit_serve_pending names the file that holds the
# line that one read: it takes the era, reads the line, removes the file and its directory, tells the client the
# tools and the resources may have changed, and answers the line. A client that opens with initialize hears nothing
# before its answer, as the earlier revisions require.
serve:resume() {
  local file=${kit_serve_pending:-} pending=''
  [[ -n "${file}" ]] || return 0
  serve_era=${kit_serve_era:-}
  unset kit_serve_era kit_serve_pending
  [[ -f "${file}" ]] && pending=$(<"${file}")
  rm -f "${file}"
  rmdir "${file%/*}" 2>/dev/null || true  # a directory left behind costs only tidiness
  [[ -n "${pending}" ]] || return 0
  if [[ "${pending}" != *'"initialize"'* ]]
  then
    serve:notify notifications/tools/list_changed
    serve:notify notifications/resources/list_changed
  fi
  serve:answer "${pending}"
}

# serve_unplaced is true while kit.yml has changed and .kit/ could not be placed from it (serve:reload).
serve_unplaced=false

# serve:placed <id> succeeds while .kit/ is placed from kit.yml as it is. Otherwise it answers request <id> with an
# internal error, so no call or read is answered from the .kit/ placed before kit.yml changed, and returns 1.
serve:placed() {
  [[ "${serve_unplaced}" == true ]] || return 0
  local message='Internal error: run: kit hydrate (.kit/ could not be placed after kit.yml changed, so the server'
  message+=' answers no call or read from the .kit/ placed before; the steps that failed are on its stderr)'
  serve:error "$1" -32603 "${message}"
  return 1
}

# serve:notify <method> sends a notification, a message the client does not answer.
serve:notify() {
  printf '{"jsonrpc":"2.0","method":"%s"}\n' "$1"
}

# serve:handle <line> answers one message. The line serve:guard-nul passes for one that held a NUL byte is a parse
# error. A request that holds a NUL escape, \u0000, in any string, a key or a value at any depth, is refused before
# any field of it reaches the shell, which would drop the NUL and serve another name than the one sent; the method it
# logs has a space for each NUL.
serve:handle() {
  local fields kind id method params nul quoted_id quoted_method
  if [[ "$1" == $'\x01' ]]
  then
    serve:error null -32700 'Parse error: the line holds a NUL byte, which JSON text never does'
    return 0
  fi
  if ! fields=$(jq --raw-output '
      if type != "object" then "invalid", "null", "", "{}", false
      elif has("method") | not then "response", "null", "", "{}", false
      else
        (if has("id") then "request" else "notification" end),
        (.id | tojson),
        (.method | if type == "string" then gsub("[\r\n]"; " ") | explode | map(if . == 0 then 32 else . end)
          | implode else "" end),
        (.params | if . == null then {} elif type == "object" then . else null end | tojson),
        any(.. | strings, (objects | keys[]); explode | any(. == 0))
      end' <<<"$1" 2>/dev/null)
  then
    serve:error null -32700 'Parse error: the line is not JSON'
    return 0
  fi
  { read -r kind; read -r id; read -r method; read -r params; read -r nul; } <<<"${fields}"
  printf -v quoted_id '%q' "${id}"
  printf -v quoted_method '%q' "${method:-none}"
  serve_log_fields=("method=${quoted_method}")
  [[ "${kind}" != request ]] || serve_log_fields=("id=${quoted_id}" "${serve_log_fields[@]}")
  case "${kind}" in
    (invalid)
      serve:error null -32600 'Invalid Request: a message is one JSON object'
      return 0
      ;;
    (response | notification)
      return 0
  esac
  if [[ "${nul}" == true ]]
  then
    serve:error "${id}" -32602 'Invalid params: the request holds a NUL, which no command line carries'
    return 0
  fi
  # params null is no params; any other value that is not an object reaches no handler.
  if [[ "${params}" == null ]]
  then
    serve:error "${id}" -32602 'Invalid params: params is an object'
    return 0
  fi
  case "${method}" in
    (initialize) serve:initialize "${id}" "${params}" ;;
    (ping) serve:reply "${id}" '{}' ;;
    (server/discover) serve:version "${id}" "${params}" discover && serve:discover "${id}" ;;
    (tools/list) serve:version "${id}" "${params}" && serve:tools-list "${id}" ;;
    (tools/call) serve:version "${id}" "${params}" && serve:placed "${id}" && serve:tools-call "${id}" "${params}" ;;
    (resources/list) serve:version "${id}" "${params}" && serve:resources-list "${id}" ;;
    (resources/read)
      serve:version "${id}" "${params}" && serve:placed "${id}" && serve:resources-read "${id}" "${params}" ;;
    (*) serve:error "${id}" -32601 "Method not found: ${method:0:200}"
  esac
  return 0
}

# serve:reply <id> <result> writes the answer to request <id>, both as JSON, on one line. Both reach jq on stdin,
# since either may be longer than a command line carries.
serve:reply() {
  serve_outcome=${serve_outcome:-ok}
  printf '%s\n' "$1" "$2" | jq --null-input --compact-output \
    'input as $id | input as $result | {jsonrpc: "2.0", id: $id, result: $result}'
}

# serve:error <id> <code> <message> [<data>] writes an error answer to request <id>; <id> and <data> are JSON. The
# id, the data, and the message reach jq on stdin, the message as a JSON string.
serve:error() {
  serve_outcome="error=$2"
  { printf '%s\n' "$1" "${4:-null}"; printf '%s' "$3" | jq --raw-input --slurp .; } \
    | jq --null-input --compact-output --argjson code "$2" 'input as $id | input as $data | input as $message
      | {jsonrpc: "2.0", id: $id, error: ({code: $code, message: $message} + if $data then {data: $data} else {} end)}'
}

# serve:complete <id> <result> [cacheable] answers with a complete result, the resultType 2026-07-28 requires, and,
# for a list or a read, its caching hints. ttlMs is 0 because the files and the verbs' answers change as the agent
# works, and the scope is private because they are the repository's own.
serve:complete() {
  local hints='{}'
  [[ "${3:-}" == cacheable ]] && hints='{"ttlMs": 0, "cacheScope": "private"}'
  serve:reply "$1" "$(jq --compact-output --argjson hints "${hints}" '{resultType: "complete"} + . + $hints' <<<"$2")"
}

# serve:version <id> <params> [discover] is true when the request may be served: it names a version the server
# answers, or names none after an initialize, or is server/discover. Otherwise it writes the error and is false.
serve:version() {
  local requested
  requested=$(jq --raw-output '(._meta | objects | .["io.modelcontextprotocol/protocolVersion"]) // "" | strings' \
    <<<"$2")
  if [[ -z "${requested}" ]]
  then
    [[ "${serve_era}" == legacy || "${3:-}" == discover ]] && return 0
    serve:error "$1" -32602 'Invalid params: params._meta names no io.modelcontextprotocol/protocolVersion'
    return 1
  fi
  serve:known-version "${requested}" && return 0
  serve:error "$1" -32022 'Unsupported protocol version' "$(printf '%s' "${requested}" | jq --raw-input --slurp \
    --compact-output '{supported: $ARGS.positional, requested: .[:200]}' --args "${serve_versions[@]}")"
  return 1
}

# serve:known-version <version> is true when the server answers that revision.
serve:known-version() {
  local version
  for version in "${serve_versions[@]}"
  do
    [[ "$1" == "${version}" ]] && return 0
  done
  return 1
}

# serve:initialize <id> <params> opens a session of an earlier revision: the one the client asks for when the server
# answers it, and otherwise the newest of them.
serve:initialize() {
  local requested version=${serve_versions[1]}
  requested=$(jq --raw-output '.protocolVersion // "" | strings' <<<"$2")
  if [[ -n "${requested}" && "${requested}" != "${serve_versions[0]}" ]] && serve:known-version "${requested}"
  then
    version=${requested}
  fi
  serve_era=legacy
  serve:reply "$1" "$(jq --null-input --compact-output --arg version "${version}" --arg kit "$(release:version)" \
    --arg instructions "${serve_instructions}" '{
      protocolVersion: $version,
      capabilities: {tools: {listChanged: true}, resources: {listChanged: true}},
      serverInfo: {name: "kit", version: $kit},
      instructions: $instructions
    }')"
}

# serve:discover <id> answers server/discover with the revisions, the capabilities, and the server's name.
serve:discover() {
  serve:complete "$1" "$(jq --null-input --compact-output --arg kit "$(release:version)" \
    --arg instructions "${serve_instructions}" '{
      supportedVersions: $ARGS.positional,
      capabilities: {tools: {listChanged: true}, resources: {listChanged: true}},
      instructions: $instructions,
      _meta: {"io.modelcontextprotocol/serverInfo": {name: "kit", version: $kit}}
    }' --args "${serve_versions[@]}")" cacheable
}

# serve:tools-list <id> answers tools/list with every tool, built once per process.
serve:tools-list() {
  if [[ -z "${serve_tool_list}" ]]
  then
    local tool
    serve_tool_list=$(for tool in "${serve_tools[@]}"
    do
      serve:tool "${tool}"
    done | jq --slurp --compact-output .)
  fi
  serve:complete "$1" "{\"tools\": ${serve_tool_list}}" cacheable
}

# serve:spec <tool> fills serve_specs[<tool>] with the verb's syntax, as syntax:spec prints it, and serve_opts[<tool>]
# with what the server says of the tool, as serve.jq reads it: the words it does not offer, the flags it gives itself,
# the name of the property for a line's leading word, and the values of each argument whose choices are closed.
# Each is built once per process. It assigns rather than prints, so call it in the server's own shell.
serve:spec() {
  local tool=$1 spec key choices='{}'
  [[ -z "${serve_specs[${tool}]+x}" ]] || return 0
  spec=$(syntax:spec "${tool}") || return 1
  while IFS= read -r key
  do
    [[ -n "${key}" ]] || continue
    choices=$(syntax:choices "${tool}" "${key}" | cut -f 1 \
      | jq --raw-input --slurp --compact-output --argjson choices "${choices}" --arg key "${key}" \
        '$choices + {($key): (split("\n") | map(select(. != "")))}')
  done < <(jq --raw-output '.arguments | to_entries[] | select(.value.choices == "closed") | .key' <<<"${spec}")
  serve_specs[${tool}]=${spec}
  serve_opts[${tool}]=$(jq --null-input --compact-output --argjson choices "${choices}" \
    --arg refused "${serve_listing_only[${tool}]:-} ${serve_hidden[${tool}]:-}" --arg implied "${serve_implied[${tool}]:-}" \
    --arg word "${serve_leading[${tool}]:-}" \
    '{refused: ($refused | split(" ") | map(select(. != ""))), implied: ($implied | split(" ") | map(select(. != ""))),
      choices: $choices} + (if $word != "" then {word: $word} else {} end)')
}

# serve:tool <tool> prints the tool's definition as JSON: its name, a description of its purpose and the usage lines
# it offers, its typed input, and its hints: that it reads only, or, for a workflow tool, what it changes.
serve:tool() {
  local tool=$1 open=false world acts='{"readOnlyHint": true, "destructiveHint": false}'
  for world in "${serve_open_world[@]}"
  do
    [[ "${tool}" == "${world}" ]] && open=true
  done
  [[ -z "${serve_acts[${tool}]+x}" ]] || acts=${serve_acts[${tool}]}
  serve:spec "${tool}" || return 1
  jq --compact-output -L "${kit_jq}" --arg purpose "$(help:purpose "${tool}")" --argjson opts "${serve_opts[${tool}]}" \
    --argjson open "${open}" --argjson acts "${acts}" 'import "serve" as serve;
    {
      name: .verb,
      title: ("kit " + .verb),
      description: ("kit \(.verb) \($purpose)\n\nUsage:\n" + ([serve::lines($opts)[].text | "  " + .] | join("\n"))
        + "\n\nIts whole page is the resource kit://help/\(.verb)."),
      inputSchema: serve::schema($opts),
      annotations: ($acts + {openWorldHint: $open})
    }' <<<"${serve_specs[${tool}]}"
}

# serve:tools-call <id> <params> runs the tool's verb with the words its arguments make, and answers with what it
# printed. Arguments the tool does not take are refused with -32602, and the verb never runs; a listing-only tool's
# refusal says what it does not offer. A tool whose operands are paths refuses, the same way, a word outside the
# repository. A verb that exits non-zero sets isError, and the text names its exit status; one the system will not
# run, as when its words are longer than a command line carries, has status 126. The arguments reach jq on stdin, and
# the words, the verb's output, and the command reach the server through files in its scratch directory, which each
# call writes over, so no call leaves a file behind.
serve:tools-call() {
  local name status=0 quoted made problem tool word
  local words_file=${serve_scratch}/words out=${serve_scratch}/out err=${serve_scratch}/err
  local command=${serve_scratch}/command
  local -a words
  name=$(jq --raw-output '.name // "" | strings' <<<"$2")
  if ! serve:is-tool "${name}"
  then
    serve:error "$1" -32602 "Unknown tool: ${name:0:200}; the tools are ${serve_tools[*]}"
    return 0
  fi
  if ! jq --exit-status '(.arguments // {}) | type == "object"' <<<"$2" >/dev/null
  then
    serve:error "$1" -32602 'Invalid params: arguments is an object, one property per argument'
    return 0
  fi
  serve:spec "${name}" || { serve:error "$1" -32603 "Internal error: kit ${name}'s usage does not parse"; return 0; }
  if ! made=$(jq --compact-output -L "${kit_jq}" --argjson spec "${serve_specs[${name}]}" \
    --argjson opts "${serve_opts[${name}]}" \
    'import "serve" as serve; (.arguments // {}) as $args | $spec | serve::argv($args; $opts)' <<<"$2")
  then
    serve:error "$1" -32603 "Internal error: the verb's words could not be read"
    return 0
  fi
  problem=$(jq --raw-output '.error // empty' <<<"${made}")
  if [[ -n "${problem}" ]]
  then
    if [[ -n "${serve_listing_only[${name}]+x}" \
      && ( "${problem}" == 'unknown argument'* || "${problem}" == *" is a word kit ${name} acts on" ) ]]
    then
      problem+="; kit serve offers kit ${name} as a listing only; run it in a terminal to ${serve_listing_acts[${name}]}"
    fi
    serve:error "$1" -32602 "Invalid params: ${problem}"
    return 0
  fi
  # A word jq cannot write, or a file it cannot make, would leave the verb with fewer words than the call gave.
  if ! jq --raw-output0 '.argv[]' <<<"${made}" >"${words_file}" 2>/dev/null
  then
    serve:error "$1" -32603 "Internal error: the verb's words could not be read"
    return 0
  fi
  readarray -d '' -t words <"${words_file}"
  for tool in "${serve_paths[@]}"
  do
    [[ "${tool}" == "${name}" ]] || continue
    for word in "${words[@]}"
    do
      if [[ "${word}" != -* ]] && ! serve:inside "${word}"
      then
        serve:error "$1" -32602 "Invalid params: ${word:0:200} is outside the repository kit serve runs in"
        return 0
      fi
    done
  done
  printf -v quoted ' %q' "${name}" "${words[@]}"
  serve_log_last="kit${quoted}"
  printf '%s' "kit ${name}${words[*]:+ ${words[*]}}" >"${command}"
  (
    export NO_COLOR=1 kit_serve_hydrated=1
    unset CLICOLOR_FORCE
    run kit "${name}" "${words[@]}"
  ) </dev/null >"${out}" 2>"${err}" || status=$?
  (( status == 0 )) || serve_outcome="status=${status}"
  serve:complete "$1" "$(jq --null-input --compact-output --rawfile out "${out}" --rawfile err "${err}" \
    --rawfile command "${command}" --argjson status "${status}" '{
      content: [{type: "text", text: (
        [$out, $err, (if $status != 0 then "\($command) exited with status \($status)\n" else empty end)]
        | map(select(length > 0)) | join("\n")
      )}],
      isError: ($status != 0)
    }')"
}

# serve_path_max is the longest path the system resolves, PATH_MAX, once serve:inside has asked for it.
serve_path_max=''

# serve:inside <path> succeeds when the path, read from the root, is the root or under it, as the kernel reads it and
# as a shell does. The two read a .. after a symbolic link differently, and a verb may use either: the kernel applies
# .. to where the link leads, and a shell's cd removes the part before .. as text. A path with a .. is judged both
# ways, and is inside only when both readings are; a .. that leaves nothing before it to remove leads out of /. A
# path whose last part is .. is judged a third way, as context:relative reads it: the parts before it as a shell
# reads them, then the .. as the kernel applies it. A path without a .. reads the same every way, and is judged
# once. A . or an empty part never changes where a path leads, so no reading counts it. A path whose length from /,
# the root's own length included, is PATH_MAX bytes or more without them is outside, since the system answers "too
# long" for it whether or not it exists, and the guard cannot judge it. A root that cannot be read, as when the
# server's directory is gone, leaves every path outside. Each part is taken once, since a pattern that cuts a part off
# the end takes time that grows with the square of a path's length, and a call's path may be megabytes long.
serve:inside() {
  local LC_ALL=C root path=$1 part length=0 up=false last='' popped=''
  local -a parts kernel=() shell=()
  root=$(pwd -P 2>/dev/null) && [[ "${root}" == /* && -d "${root}" ]] || return 1
  if [[ -z "${serve_path_max}" ]]
  then
    serve_path_max=$(getconf PATH_MAX / 2>/dev/null) || true  # with no answer, the default below
    [[ "${serve_path_max}" =~ ^[0-9]+$ ]] || serve_path_max=1024
  fi
  [[ "${path}" == /* ]] || path=${root}/${path}
  readarray -d / -t parts < <(printf '%s' "${path}")
  for part in "${parts[@]}"
  do
    [[ -n "${part}" && "${part}" != . ]] || continue
    length=$(( length + ${#part} + 1 ))
    (( length < serve_path_max )) || return 1
    kernel+=("${part}")
    last=${part}
    if [[ "${part}" == .. ]]
    then
      up=true
      (( ${#shell[@]} > 0 )) || return 1
      popped=${shell[-1]}
      unset 'shell[-1]'
    else
      shell+=("${part}")
    fi
  done
  serve:resolves-inside "${root}" "${kernel[@]}" || return 1
  [[ "${up}" == false ]] || serve:resolves-inside "${root}" "${shell[@]}" || return 1
  [[ "${last}" != .. ]] || serve:resolves-inside "${root}" "${shell[@]}" "${popped}" ..
}

# serve:resolves-inside <root> [<part>…] succeeds when the path the parts make, from /, resolved by the kernel with
# every symbolic link followed, is <root> or under it. A path that does not exist is judged by its nearest directory
# that does, and is outside when a .. is left after that. A dangling link, one whose target does not exist, is
# outside, since where it leads cannot be judged.
serve:resolves-inside() {
  local root=$1 part path='' found=''
  shift
  for part in "$@"
  do
    if [[ -z "${found}" && -L "${path}/${part}" && ! -e "${path}/${part}" ]]
    then
      return 1
    elif [[ -z "${found}" && -e "${path}/${part}" ]]
    then
      path+=/${part}
    else
      found=${path:-/}
      [[ "${part}" != .. ]] || return 1
    fi
  done
  found=$(readlink -f "${found:-${path:-/}}") || return 1
  [[ "${found}" == "${root}" || "${found}" == "${root}"/* ]]
}

# serve:is-tool <name> is true when the server has a tool of that name.
serve:is-tool() {
  local tool
  for tool in "${serve_tools[@]}"
  do
    [[ "$1" == "${tool}" ]] && return 0
  done
  return 1
}

# serve:resources prints each resource the repository has, one per line: its URI, its path, its MIME type, and its
# description, tab-separated. A tool's help page has no file; its path is help:<verb>.
serve:resources() {
  local file kind directory tool
  [[ -f .kit/1st.md ]] &&
    printf 'kit://1st.md\t.kit/1st.md\ttext/markdown\tthe entry every agent reads first: integrity, the desk, and the rules\n'
  [[ -f .kit/workflow.md ]] &&
    printf 'kit://workflow.md\t.kit/workflow.md\ttext/markdown\tthe workflow from an issue to a merged change, and the worker contract\n'
  for directory in rules styles
  do
    for file in .kit/"${directory}"/*.md
    do
      [[ -f "${file}" ]] || continue
      kind=$(basename "${file}" .md)
      printf 'kit://%s/%s\t%s\ttext/markdown\tthe %s for files of kind %s\n' "${directory}" "${kind}" "${file}" \
        "${directory%s}" "${kind}"
    done
  done
  for file in .kit/kinds/*.yml
  do
    [[ -f "${file}" ]] || continue
    kind=$(context:read-kinds "${file}" | awk -F '\x1f' '$3 == "name" { printf "%s%s", sep, $2; sep = ", " }')
    printf 'kit://kinds/%s\t%s\tapplication/yaml\tthe paths and first lines that make a file of kind %s\n' \
      "$(basename "${file}" .yml)" "${file}" "${kind}"
  done
  for tool in "${serve_tools[@]}"
  do
    printf 'kit://help/%s\thelp:%s\ttext/plain\tthe whole help page of kit %s, the %s tool\n' "${tool}" "${tool}" \
      "${tool}" "${tool}"
  done
}

# serve:resources-list <id> answers resources/list with every resource the repository has.
serve:resources-list() {
  serve:complete "$1" "$(serve:resources | jq --raw-input --slurp --compact-output '{
    resources: [split("\n")[] | select(length > 0) | split("\t") | {
      uri: .[0], name: (.[0] | ltrimstr("kit://")), description: .[3], mimeType: .[2]
    }]
  }')" cacheable
}

# serve:resources-read <id> <params> answers resources/read with the text of the file the URI names, or of the help
# page, without color, when it names a tool's page. The URI is matched in the shell, and reaches jq only once it
# names a resource, or on stdin, cut to 200 characters, when it names none.
serve:resources-read() {
  local uri path='' mime='' quoted candidate file type text
  uri=$(jq --raw-output '.uri // "" | strings' <<<"$2")
  printf -v quoted '%q' "${uri}"
  serve_log_last="uri=${quoted}"
  while IFS=$'\t' read -r candidate file type _
  do
    if [[ -z "${path}" && "${candidate}" == "${uri}" ]]
    then
      path=${file} mime=${type}
    fi
  done < <(serve:resources)
  if [[ -z "${path}" ]]
  then
    serve:error "$1" -32602 "Resource not found: ${uri:0:200}" \
      "$(printf '%s' "${uri}" | jq --raw-input --slurp --compact-output '{uri: .[:200]}')"
    return 0
  fi
  if [[ "${path}" == help:* ]]
  then
    if ! text=$(NO_COLOR=1 help:page "${path#help:}")
    then
      serve:error "$1" -32603 "Internal error: kit help ${path#help:} could not be read"
      return 0
    fi
    serve:complete "$1" "$(printf '%s\n' "${text}" | jq --raw-input --slurp --compact-output --arg uri "${uri}" \
      --arg mime "${mime}" '{contents: [{uri: $uri, mimeType: $mime, text: .}]}')" cacheable
    return 0
  fi
  serve:complete "$1" "$(jq --null-input --compact-output --arg uri "${uri}" --arg mime "${mime}" \
    --rawfile text "${path}" '{contents: [{uri: $uri, mimeType: $mime, text: $text}]}')" cacheable
}

# ----------------------------------------------------------------------------------------------------------------------
