# shellcheck shell=bash
# serve.sh - the Model Context Protocol server that kit serve runs on stdio (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# serve:main reads one JSON-RPC message per line on stdin and answers each in order, one line on stdout per answer.
# Nothing else reaches stdout: a verb a tool runs has its stdout and stderr captured, and every log line goes to
# stderr. A message without an id is a notification, and gets no answer, even when it fails.
#
# The server answers two eras of the protocol, as the specification lets a dual-era server do:
#
#   - 2026-07-28, which has no session. A client may ask server/discover, and every other request names its version
#     in params._meta["io.modelcontextprotocol/protocolVersion"]; a request that names none is refused with -32602.
#   - The revisions before it, which a client selects by opening with initialize. From then on a request needs no
#     _meta, and ping is answered.
#
# A tool is a verb that reads and reports, and changes nothing. Its description is the verb's help, read by the
# functions kit help uses, and its one argument is args, the verb's arguments as an array of strings. A call runs
# kit <verb> <args>… in a new process with stdin at end of file, so the verb cannot read the protocol stream.
#
# A resource is one of the kit's files under .kit/: kit://1st.md, and kit://rules/<kind>, kit://styles/<kind>, and
# kit://kinds/<kind> for each one the repository has.
#
# see: kit/bin/kit-serve, docs/serve.md, https://modelcontextprotocol.io/specification/2026-07-28

use help release syntax

# The protocol revisions the server answers, the newest first. The first is the one without a session.
declare -gra serve_versions=(2026-07-28 2025-11-25 2025-06-18 2025-03-26)

# The tools, each a verb that changes nothing. package-list runs kit package list.
declare -gra serve_tools=(backlog changes check context help lint package-list status syntax version)

# The tools that ask the forge, over the network, rather than reading the repository alone.
declare -gra serve_open_world=(backlog changes status)

# serve_era is legacy once a client opens with initialize, and empty until then. serve_tool_list holds the tools/list
# result's tools array once it is built.
serve_era=''
serve_tool_list=''

serve_instructions='kit holds the rules this repository asks every agent to follow. Read the resource kit://1st.md'
serve_instructions+=' first. Before you change a file, call the context tool with its path, and read every rule it names.'

# serve:main answers each message on stdin, in order, until stdin ends.
serve:main() {
  local line
  while IFS= read -r line || [[ -n "${line}" ]]
  do
    [[ -n "${line//[[:space:]]/}" ]] || continue
    serve:handle "${line}"
  done
}

# serve:handle <line> answers one message.
serve:handle() {
  local fields kind id method params
  if ! fields=$(jq --raw-output '
      if type != "object" then "invalid", "null", "", "{}"
      elif has("method") | not then "response", "null", "", "{}"
      else
        (if has("id") then "request" else "notification" end),
        (.id | tojson),
        (.method | if type == "string" then gsub("[\r\n]"; " ") else "" end),
        (.params // {} | tojson)
      end' <<<"$1" 2>/dev/null)
  then
    serve:error null -32700 'Parse error: the line is not JSON'
    return 0
  fi
  { read -r kind; read -r id; read -r method; read -r params; } <<<"${fields}"
  case "${kind}" in
    (invalid)
      serve:error null -32600 'Invalid Request: a message is one JSON object'
      return 0
      ;;
    (response | notification)
      return 0
  esac
  case "${method}" in
    (initialize) serve:initialize "${id}" "${params}" ;;
    (ping) serve:reply "${id}" '{}' ;;
    (server/discover) serve:version "${id}" "${params}" discover && serve:discover "${id}" ;;
    (tools/list) serve:version "${id}" "${params}" && serve:tools-list "${id}" ;;
    (tools/call) serve:version "${id}" "${params}" && serve:tools-call "${id}" "${params}" ;;
    (resources/list) serve:version "${id}" "${params}" && serve:resources-list "${id}" ;;
    (resources/read) serve:version "${id}" "${params}" && serve:resources-read "${id}" "${params}" ;;
    (*) serve:error "${id}" -32601 "Method not found: ${method}"
  esac
  return 0
}

# serve:reply <id> <result> writes the answer to request <id>, both as JSON, on one line.
serve:reply() {
  jq --null-input --compact-output --argjson id "$1" --argjson result "$2" '{jsonrpc: "2.0", id: $id, result: $result}'
}

# serve:error <id> <code> <message> [<data>] writes an error answer to request <id>; <id> and <data> are JSON.
serve:error() {
  jq --null-input --compact-output --argjson id "$1" --argjson code "$2" --arg message "$3" --argjson data "${4:-null}" \
    '{jsonrpc: "2.0", id: $id, error: ({code: $code, message: $message} + if $data then {data: $data} else {} end)}'
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
  requested=$(jq --raw-output '._meta["io.modelcontextprotocol/protocolVersion"] // "" | strings' <<<"$2")
  if [[ -z "${requested}" ]]
  then
    [[ "${serve_era}" == legacy || "${3:-}" == discover ]] && return 0
    serve:error "$1" -32602 'Invalid params: params._meta names no io.modelcontextprotocol/protocolVersion'
    return 1
  fi
  serve:known-version "${requested}" && return 0
  serve:error "$1" -32022 'Unsupported protocol version' "$(jq --null-input --compact-output \
    --arg requested "${requested}" '{supported: $ARGS.positional, requested: $requested}' --args "${serve_versions[@]}")"
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
      capabilities: {tools: {}, resources: {}},
      serverInfo: {name: "kit", version: $kit},
      instructions: $instructions
    }')"
}

# serve:discover <id> answers server/discover with the revisions, the capabilities, and the server's name.
serve:discover() {
  serve:complete "$1" "$(jq --null-input --compact-output --arg kit "$(release:version)" \
    --arg instructions "${serve_instructions}" '{
      supportedVersions: $ARGS.positional,
      capabilities: {tools: {}, resources: {}},
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

# serve:tool <tool> prints the tool's definition as JSON: its name, its description from the verb's help, args as
# its input, and the hints that it changes nothing.
serve:tool() {
  local tool=$1 verb=$1 description open=false world
  for world in "${serve_open_world[@]}"
  do
    [[ "${tool}" == "${world}" ]] && open=true
  done
  if [[ "${tool}" == package-list ]]
  then
    verb=package
    description="kit package list $(syntax:spec package | jq --raw-output '.arguments.list.description')"$'\n\n'
    description+=$'Usage:\n  kit package list'
  else
    description="kit ${verb} $(help:purpose "${verb}")"$'\n\nUsage:\n'
    description+=$(help:usage "${verb}" | sed 's/^/  /')$'\n\n'
    description+=$(help:description "${verb}")
  fi
  jq --null-input --compact-output --arg name "${tool}" --arg description "${description}" \
    --arg verb "${verb}" --argjson open "${open}" '{
      name: $name,
      title: ("kit " + ($name | sub("-"; " "))),
      description: $description,
      inputSchema: {
        type: "object",
        properties: {
          args: {
            type: "array",
            items: {type: "string"},
            description: ("the arguments that follow kit " + ($name | sub("-"; " ")) + ", as the usage lines write them")
          }
        },
        additionalProperties: false
      },
      annotations: {readOnlyHint: true, destructiveHint: false, openWorldHint: $open}
    }'
}

# serve:tools-call <id> <params> runs the tool's verb with its args and answers with what it printed. A verb that
# exits non-zero sets isError, and the text names its exit status.
serve:tools-call() {
  local name status=0 out err
  local -a command args
  name=$(jq --raw-output '.name // "" | strings' <<<"$2")
  if ! serve:is-tool "${name}"
  then
    serve:error "$1" -32602 "Unknown tool: ${name}; the tools are ${serve_tools[*]}"
    return 0
  fi
  if ! jq --exit-status '(.arguments.args // []) | type == "array" and all(type == "string")' <<<"$2" >/dev/null
  then
    serve:error "$1" -32602 'Invalid params: args is an array of strings'
    return 0
  fi
  readarray -d '' -t args < <(jq --raw-output0 '.arguments.args // [] | .[]' <<<"$2")
  command=("${name}")
  [[ "${name}" == package-list ]] && command=(package list)
  out=$(mktemp) err=$(mktemp)
  (
    export NO_COLOR=1
    unset CLICOLOR_FORCE
    run kit "${command[@]}" "${args[@]}"
  ) </dev/null >"${out}" 2>"${err}" || status=$?
  serve:complete "$1" "$(jq --null-input --compact-output --rawfile out "${out}" --rawfile err "${err}" \
    --arg command "kit ${command[*]}" --argjson status "${status}" '{
      content: [{type: "text", text: (
        [$out, $err, (if $status != 0 then "\($command) exited with status \($status)\n" else empty end)]
        | map(select(length > 0)) | join("\n")
      )}],
      isError: ($status != 0)
    }')"
  rm -f "${out}" "${err}"
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
# description, tab-separated.
serve:resources() {
  local file kind directory
  [[ -f .kit/1st.md ]] &&
    printf 'kit://1st.md\t.kit/1st.md\ttext/markdown\tthe entry every agent reads first: the workflow and the rules\n'
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
    kind=$(basename "${file}" .yml)
    printf 'kit://kinds/%s\t%s\tapplication/yaml\tthe paths and first lines that make a file of kind %s\n' \
      "${kind}" "${file}" "${kind}"
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

# serve:resources-read <id> <params> answers resources/read with the text of the file the URI names.
serve:resources-read() {
  local uri path mime
  uri=$(jq --raw-output '.uri // "" | strings' <<<"$2")
  IFS=$'\t' read -r path mime < <(serve:resources | awk -F '\t' -v uri="${uri}" '$1 == uri { print $2 "\t" $3 }') || true
  if [[ -z "${path:-}" ]]
  then
    serve:error "$1" -32602 "Resource not found: ${uri}" "$(jq --null-input --compact-output --arg uri "${uri}" '{uri: $uri}')"
    return 0
  fi
  serve:complete "$1" "$(jq --null-input --compact-output --arg uri "${uri}" --arg mime "${mime}" \
    --rawfile text "${path}" '{contents: [{uri: $uri, mimeType: $mime, text: $text}]}')" cacheable
}

# ----------------------------------------------------------------------------------------------------------------------
