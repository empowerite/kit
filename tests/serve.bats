#!/usr/bin/env bats
# serve.bats - kit serve answers the Model Context Protocol on stdio: the handshake of either era, tools, and resources
# ----------------------------------------------------------------------------------------------------------------------
#
# Each test feeds request lines to kit serve in a repository with the kit installed, and reads the answers with jq.
# Stdout must hold one JSON-RPC message per line and nothing else, so every test reads it apart from stderr.

bats_require_minimum_version 1.5.0

setup() {
  export kit_checkout NO_COLOR=1
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}"
  repo="${BATS_TEST_TMPDIR}/repo"
  git init --quiet --initial-branch=main "${repo}"
  cd "${repo}"
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
}

# serve <line>… runs kit serve over the lines, with stdout in ${BATS_TEST_TMPDIR}/out and stderr in .../err.
serve() {
  printf '%s\n' "$@" | "${kit_checkout}/bin/kit" serve >"${BATS_TEST_TMPDIR}/out" 2>"${BATS_TEST_TMPDIR}/err"
}

# answer <id> prints the answer to request <id>, an id given as JSON.
answer() {
  jq --compact-output --argjson id "$1" 'select(.id == $id)' "${BATS_TEST_TMPDIR}/out"
}

# modern <id> <method> [<params>] prints a request of the 2026-07-28 revision, its version in _meta.
modern() {
  jq --null-input --compact-output --argjson id "$1" --arg method "$2" --argjson params "${3:-{\}}" '{
    jsonrpc: "2.0", id: $id, method: $method,
    params: ($params + {_meta: {"io.modelcontextprotocol/protocolVersion": "2026-07-28"}})
  }'
}

# call <id> <tool> <arg>… prints a tools/call request of the 2026-07-28 revision.
call() {
  local id=$1 tool=$2
  shift 2
  modern "${id}" tools/call "$(jq --null-input --compact-output --arg name "${tool}" \
    '{name: $name, arguments: {args: $ARGS.positional}}' --args "$@")"
}

@test "an initialize opens a session of the revision the client asks for; a notification gets no answer, and ping does" {
  serve '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"ping"}' \
    '{"jsonrpc":"2.0","id":3,"method":"tools/list"}'
  [ "$(wc -l < "${BATS_TEST_TMPDIR}/out")" -eq 3 ]
  answer 1 | jq --exit-status '.result | .protocolVersion == "2025-06-18" and .serverInfo.name == "kit"
    and .capabilities == {tools: {}, resources: {}} and (.instructions | contains("kit://1st.md"))'
  answer 2 | jq --exit-status '.result == {}'
  answer 3 | jq --exit-status '.result.tools | length == 10'
  serve '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1999-01-01","capabilities":{}}}'
  answer 1 | jq --exit-status '.result.protocolVersion == "2025-11-25"'
}

@test "without an initialize, a request names its revision in _meta, and server/discover lists the revisions" {
  serve '{"jsonrpc":"2.0","id":"d","method":"server/discover"}' \
    '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"1900-01-01"}}}' \
    "$(modern 3 resources/list)"
  answer '"d"' | jq --exit-status '.result | .resultType == "complete" and .supportedVersions[0] == "2026-07-28"
    and ._meta["io.modelcontextprotocol/serverInfo"].name == "kit" and .ttlMs == 0 and .cacheScope == "private"'
  answer 1 | jq --exit-status '.error.code == -32602'
  answer 2 | jq --exit-status '.error | .code == -32022 and .data.requested == "1900-01-01"
    and (.data.supported | index("2026-07-28"))'
  answer 3 | jq --exit-status '.result | .resultType == "complete" and (.resources | length > 0)'
}

@test "a line that is not JSON, an array, an unknown method, and an unknown tool each get their error, and the server goes on" {
  serve 'not json' '[1,2]' "$(modern 1 nope)" "$(call 2 rm -rf /)" '{"jsonrpc":"2.0","method":"nope"}' \
    "$(modern 3 tools/call '{"name":"version","arguments":{"args":[1]}}')" "$(call 4 version)"
  [ "$(wc -l < "${BATS_TEST_TMPDIR}/out")" -eq 6 ]
  [ "$(sed -n 1p "${BATS_TEST_TMPDIR}/out" | jq '.error.code')" -eq -32700 ]
  [ "$(sed -n 2p "${BATS_TEST_TMPDIR}/out" | jq '.error.code')" -eq -32600 ]
  answer 1 | jq --exit-status '.error.code == -32601'
  answer 2 | jq --exit-status '.error | .code == -32602 and (.message | contains("context"))'
  answer 3 | jq --exit-status '.error.code == -32602'
  answer 4 | jq --exit-status '.result.isError == false'
}

@test "tools/list names every tool, each described by its verb's help, with args as its only input" {
  serve "$(modern 1 tools/list)"
  answer 1 | jq --exit-status '.result | .resultType == "complete" and .cacheScope == "private"'
  [ "$(answer 1 | jq --raw-output '[.result.tools[].name] | join(" ")')" = \
    "backlog changes check context help lint package-list status syntax version" ]
  local verb
  for verb in backlog changes check context help lint status syntax version
  do
    answer 1 | jq --exit-status --arg verb "${verb}" --arg purpose "kit ${verb} $(cd "${kit_checkout}" && \
      sed -n '2{s/^# kit-[a-z-]* - //; s/ (kit verb)$//; p;}' "bin/kit-${verb}")" \
      '.result.tools[] | select(.name == $verb) | (.description | startswith($purpose))
        and (.inputSchema.properties | keys == ["args"]) and .annotations.readOnlyHint'
  done
  answer 1 | jq --exit-status '.result.tools[] | select(.name == "package-list")
    | (.description | startswith("kit package list prints every package in reach"))'
  answer 1 | jq --exit-status '[.result.tools[] | select(.annotations.openWorldHint) | .name] == ["backlog", "changes", "status"]'
}

@test "a tool runs its verb: context names a bash file's rule, and lint's finding comes back as an error" {
  command -v shellcheck >/dev/null || skip "shellcheck, the bash rule's linter, is not on PATH"
  printf '#!/usr/bin/env bash\nset -o errexit -o nounset -o pipefail\necho $1\n' > tool.sh
  serve "$(call 1 context tool.sh)" "$(call 2 lint tool.sh)" "$(call 3 package-list)"
  answer 1 | jq --exit-status '.result | .isError == false and (.content[0].text | split("\n") | index(".kit/rules/bash.md"))'
  answer 2 | jq --exit-status '.result | .isError and (.content[0].text | contains("SC2086"))
    and (.content[0].text | contains("kit lint exited with status 1"))'
  answer 3 | jq --exit-status '.result | .isError == false and (.content[0].text | contains("kit"))'
}

@test "a verb reads end of file on stdin, so the requests after it reach the server, and nothing but answers reach stdout" {
  serve "$(call 1 context --claude)" "$(call 2 help 'two
lines')" "$(call 3 version)"
  [ "$(wc -l < "${BATS_TEST_TMPDIR}/out")" -eq 3 ]
  jq --exit-status --slurp 'all(.jsonrpc == "2.0") and (map(.id) == [1, 2, 3])' "${BATS_TEST_TMPDIR}/out"
  answer 2 | jq --exit-status '.result.isError'
  answer 3 | jq --exit-status '.result.content[0].text == "unreleased\n"'
}

@test "resources list the kit's files on disk, a read returns one, and a URI that names none is refused" {
  serve "$(modern 1 resources/list)" "$(modern 2 resources/read '{"uri":"kit://rules/bash"}')" \
    "$(modern 3 resources/read '{"uri":"kit://rules/../../etc/passwd"}')" "$(modern 4 resources/read '{"uri":"kit://1st.md"}')"
  local expected
  expected=$( { printf 'kit://1st.md\n'
    for file in .kit/rules/*.md; do printf 'kit://rules/%s\n' "$(basename "${file}" .md)"; done
    for file in .kit/kinds/*.yml; do printf 'kit://kinds/%s\n' "$(basename "${file}" .yml)"; done; } )
  [ "$(answer 1 | jq --raw-output '.result.resources[].uri')" = "${expected}" ]
  [ "$(answer 2 | jq --raw-output '.result.contents[0].text')" = "$(cat .kit/rules/bash.md)" ]
  answer 2 | jq --exit-status '.result.contents[0].mimeType == "text/markdown"'
  answer 3 | jq --exit-status '.error.code == -32602'
  answer 4 | jq --exit-status '.result.contents[0].text | startswith("<!--")'
}

@test "kit serve takes no arguments" {
  run "${kit_checkout}/bin/kit" serve --stdio
  [ "$status" -eq 2 ]
  [[ "$output" == *"run: kit help serve"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
