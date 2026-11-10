#!/usr/bin/env bats
# serve.bats - kit serve answers the Model Context Protocol on stdio: the handshake of either era, tools, and resources
# ----------------------------------------------------------------------------------------------------------------------
# Each test feeds request lines to kit serve in a repository with the kit installed, and reads the answers with jq.
# Stdout must hold one JSON-RPC message per line and nothing else, so every test reads it apart from stderr.

bats_require_minimum_version 1.5.0

load helper

setup_file() {
  world-repo installed
}

setup() {
  world-copy-repo installed
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

# call <id> <tool> [<arguments>] prints a tools/call request of the 2026-07-28 revision, its arguments a JSON object.
call() {
  modern "$1" tools/call "$(jq --null-input --compact-output --arg name "$2" --argjson arguments "${3:-{\}}" \
    '{name: $name, arguments: $arguments}')"
}

@test "an initialize opens a session of the revision the client asks for; a notification gets no answer, and ping does" {
  serve '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"ping"}' \
    '{"jsonrpc":"2.0","id":3,"method":"tools/list"}'
  [ "$(wc -l < "${BATS_TEST_TMPDIR}/out")" -eq 3 ]
  answer 1 | jq --exit-status '.result | .protocolVersion == "2025-06-18" and .serverInfo.name == "kit"
    and .capabilities == {tools: {listChanged: true}, resources: {listChanged: true}} and (.instructions | contains("kit://1st.md"))'
  answer 2 | jq --exit-status '.result == {}'
  answer 3 | jq --exit-status '.result.tools | length == 17'
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
  serve 'not json' '[1,2]' "$(modern 1 nope)" "$(call 2 rm)" '{"jsonrpc":"2.0","method":"nope"}' \
    "$(call 3 version '{"args":["1"]}')" "$(call 4 version)" "$(call 5 context '{"file":"x.sh"}')" \
    "$(call 6 context '{"path":"x.sh"}')" "$(call 7 context '{"path":["-x"]}')" "$(call 8 context '{"json":true}')"
  [ "$(wc -l < "${BATS_TEST_TMPDIR}/out")" -eq 10 ]
  [ "$(sed -n 1p "${BATS_TEST_TMPDIR}/out" | jq '.error.code')" -eq -32700 ]
  [ "$(sed -n 2p "${BATS_TEST_TMPDIR}/out" | jq '.error.code')" -eq -32600 ]
  answer 1 | jq --exit-status '.error.code == -32601'
  answer 2 | jq --exit-status '.error | .code == -32602 and (.message | contains("context"))'
  answer 3 | jq --exit-status '.error | .code == -32602 and (.message | contains("unknown argument args; kit version takes: "))'
  answer 4 | jq --exit-status '.result.isError == false'
  answer 5 | jq --exit-status '.error | .code == -32602 and (.message | contains("unknown argument file; kit context takes: path, json, jq"))'
  answer 6 | jq --exit-status '.error | .code == -32602 and (.message | contains("path is a list of strings"))'
  answer 7 | jq --exit-status '.error | .code == -32602 and (.message | contains("starts with -"))'
  answer 8 | jq --exit-status '.error | .code == -32602 and (.message | contains("fit no usage of kit context: kit context <path>… [--json | --jq]"))'
}

@test "tools/list names every tool, with its purpose, its usage, its page's resource, and typed inputs from its syntax" {
  serve "$(modern 1 tools/list)"
  answer 1 | jq --exit-status '.result | .resultType == "complete" and .cacheScope == "private"'
  [ "$(answer 1 | jq --raw-output '[.result.tools[].name] | join(" ")')" = \
    "check context flow help kinds lint packages syntax toolchains tools version assign close gather settle unassign work" ]
  local verb
  for verb in check context flow help kinds lint packages syntax toolchains tools version \
    assign close gather settle unassign work
  do
    answer 1 | jq --exit-status --arg verb "${verb}" --arg purpose "kit ${verb} $(cd "${kit_checkout}" && \
      sed -n '2{s/^# kit-[a-z-]* - //; s/ (kit verb[^)]*)$//; p;}' "bin/kit-${verb}")" \
      '.result.tools[] | select(.name == $verb) | (.description | startswith($purpose)
        and endswith("Its whole page is the resource kit://help/\($verb).")) and .inputSchema.additionalProperties == false'
  done
  answer 1 | jq --exit-status '.result.tools[] | select(.name == "context") | .inputSchema
    | (.properties | keys_unsorted) == ["path", "json", "jq"] and .required == ["path"]
      and .properties.path.type == "array" and .properties.json.type == "boolean"
      and (.properties.path.description | startswith("the files"))'
  answer 1 | jq --exit-status '.result.tools[] | select(.name == "flow") | .inputSchema.properties
    | (.stage.enum | index("backlog") and index("releases")) and (has("no-fetch") | not) and .owner.type == "array"'
  answer 1 | jq --exit-status '.result.tools[] | select(.name == "packages") | .inputSchema.properties.package.items.enum
    | index("kit") and index("workflow")'
  # check, lint, toolchains, and tools run what the repository's files name, and the workflow tools act: none of them
  # reads only; close, unassign, and settle are destructive, and each workflow tool asks the forge
  answer 1 | jq --exit-status '[.result.tools[] | select(.annotations.readOnlyHint | not) | .name]
    == ["check", "lint", "toolchains", "tools", "assign", "close", "gather", "settle", "unassign", "work"]'
  answer 1 | jq --exit-status '[.result.tools[] | select(.annotations.destructiveHint) | .name]
    == ["close", "settle", "unassign"]'
  answer 1 | jq --exit-status '[.result.tools[] | select(.name | IN("assign", "close", "gather", "settle", "unassign",
    "work")) | .annotations.openWorldHint] | length == 6 and all'
  answer 1 | jq --exit-status '.result.tools[] | select(.name == "close") | .inputSchema.properties
    | has("not-planned") and has("duplicate-of") and .reference.type == "array"'
  # the whole list fits in well under half the size it had when each description was the verb's whole page
  [ "$(answer 1 | wc -c)" -lt 20000 ]
}

@test "a tool runs its verb from typed arguments: context names a bash file's rule, and lint's finding comes back as an error" {
  command -v shellcheck >/dev/null || skip "shellcheck, the bash rule's linter, is not on PATH"
  printf '#!/usr/bin/env bash\nset -o errexit -o nounset -o pipefail\necho $1\n' > tool.sh
  serve "$(call 1 context '{"path":["tool.sh"]}')" "$(call 2 lint '{"path":["tool.sh"]}')" "$(call 3 packages)" \
    "$(call 4 context '{"path":["tool.sh"],"json":true}')" "$(call 5 kinds '{"path":["tool.sh"],"json":false}')"
  answer 1 | jq --exit-status '.result | .isError == false and (.content[0].text | split("\n") | index(".kit/rules/bash.md"))'
  answer 2 | jq --exit-status '.result | .isError and (.content[0].text | contains("SC2086"))
    and (.content[0].text | contains("kit lint tool.sh exited with status 1"))'
  answer 3 | jq --exit-status '.result | .isError == false and (.content[0].text | contains("kit"))'
  answer 4 | jq --exit-status '.result.content[0].text | fromjson | .path == "tool.sh" and (.rules | length > 0)'
  answer 5 | jq --exit-status '.result | .isError == false and (.content[0].text | contains("bash"))'
}

@test "packages, toolchains, and tools are offered as listings only; an action cannot be said, and is refused unrun" {
  fake-tools mise
  route mise '--version' '!0'
  fixture ls.json <<<'{"jq": [{"version": "1.8.0", "requested_version": "1.8", "installed": true}]}'
  route mise 'ls --json --current' ls.json
  touch mise.toml
  local before
  before=$(cksum kit.yml)
  serve "$(call 1 packages '{"subcommand":"add","package":["style/yaml"]}')" "$(call 2 packages '{"add":true}')" \
    "$(call 3 toolchains '{"subcommand":"choose","name":"mise"}')" "$(call 4 tools '{"subcommand":"install"}')" \
    "$(call 5 tools '{"outdated":true}')" "$(call 6 tools '{"command":["touch","ran"]}')" \
    "$(call 7 tools)" "$(call 8 toolchains)" "$(call 9 packages)" "$(modern 10 tools/list)"
  for id in 1 2
  do
    answer "${id}" | jq --exit-status '.error | .code == -32602
      and (.message | contains("kit serve offers kit packages as a listing only; run it in a terminal to"))'
  done
  answer 3 | jq --exit-status '.error.message | contains("kit serve offers kit toolchains as a listing only")'
  for id in 4 5 6
  do
    answer "${id}" | jq --exit-status '.error.message | contains("kit serve offers kit tools as a listing only")'
  done
  # nothing ran: kit.yml, the choice, and the command's file are as they were, and mise was only asked to list
  [ "$(cksum kit.yml)" = "${before}" ]
  [ ! -e .git/kit/toolchain ] && [ ! -e ran ]
  if grep -E 'install|upgrade|outdated' "${FAKE_DIR}/mise.calls"; then false; fi
  answer 7 | jq --exit-status '.result | .isError == false and (.content[0].text | contains("jq"))'
  answer 8 | jq --exit-status '.result | .isError == false and (.content[0].text | contains("mise"))'
  answer 9 | jq --exit-status '.result.isError == false'
  answer 10 | jq --exit-status '[.result.tools[] | select(.name == "packages" or .name == "toolchains" or .name == "tools")
    | .inputSchema.properties | keys[]] | any(IN("subcommand", "add", "outdated", "check", "command", "name")) | not'
}

@test "a listing-only tool's description shows only its listing usage, without the parts it refuses" {
  serve "$(modern 1 tools/list)"
  answer 1 | jq --exit-status '.result.tools[] | select(.name == "packages") | .description | split("\n\n")[1]
    == "Usage:\n  kit packages [--active | --inactive] [--json | --jq] [<package>…]"'
  answer 1 | jq --exit-status '.result.tools[] | select(.name == "tools") | .description | split("\n\n")[1]
    == "Usage:\n  kit tools [--toolchain <name>] [--json | --jq]"'
  answer 1 | jq --exit-status '.result.tools[] | select(.name == "toolchains") | .description | split("\n\n")[1]
    == "Usage:\n  kit toolchains [--json | --jq]"'
  answer 1 | jq --exit-status '.result.tools[] | select(.name == "context") | .description | split("\n\n")[1]
    == "Usage:\n  kit context <path>… [--json | --jq]"'
}

@test "a verb reads end of file on stdin, so the requests after it reach the server, and nothing but answers reach stdout" {
  serve "$(call 1 context '{"claude":true}')" "$(call 2 help '{"verb":["two\nlines"]}')" "$(call 3 version)"
  [ "$(wc -l < "${BATS_TEST_TMPDIR}/out")" -eq 3 ]
  jq --exit-status --slurp 'all(.jsonrpc == "2.0") and (map(.id) == [1, 2, 3])' "${BATS_TEST_TMPDIR}/out"
  answer 1 | jq --exit-status '.error | .code == -32602 and (.message | contains("unknown argument claude"))'
  answer 2 | jq --exit-status '.result.isError'
  answer 3 | jq --exit-status '.result.content[0].text == "unreleased\n"'
}

@test "a tool that reads only changes nothing: a call never hydrates, flow never fetches, and each says it reads only" {
  git init --quiet --bare --initial-branch=main "${BATS_TEST_TMPDIR}/origin.git"
  git remote add origin "${BATS_TEST_TMPDIR}/origin.git"
  git add --all && git commit --quiet --message 'start' && git push --quiet origin HEAD:main 2>/dev/null
  git clone --quiet "${BATS_TEST_TMPDIR}/origin.git" "${BATS_TEST_TMPDIR}/other" 2>/dev/null
  (cd "${BATS_TEST_TMPDIR}/other" && git commit --quiet --allow-empty --message 'theirs' && git push --quiet origin main)
  local seen
  seen=$(git rev-parse origin/main)
  mkfifo "${BATS_TEST_TMPDIR}/in"
  "${kit_checkout}/bin/kit" serve <"${BATS_TEST_TMPDIR}/in" >"${BATS_TEST_TMPDIR}/out" 2>"${BATS_TEST_TMPDIR}/err" &
  server=$!
  exec 7>"${BATS_TEST_TMPDIR}/in"
  modern 1 tools/list >&7
  await 1
  # the server placed .kit/ as it started; a call after a delivered file went missing leaves it missing
  rm -f .kit/rules/json.md
  call 2 kinds >&7
  call 3 flow '{"stage":"incoming","json":true}' >&7
  call 4 flow '{"json":true}' >&7
  await 4
  exec 7>&-
  wait "${server}"
  [ ! -e .kit/rules/json.md ]
  [ "$(git rev-parse origin/main)" = "${seen}" ]
  answer 3 | jq --exit-status '.result.content[0].text | fromjson | .fetched == false'
  answer 1 | jq --exit-status '[.result.tools[] | select(.name | IN("check", "lint", "toolchains", "tools", "assign",
    "close", "gather", "settle", "unassign", "work") | not) | .annotations]
    | length == 7 and all(.readOnlyHint and (.destructiveHint | not))'
}

@test "a listing-only tool refuses an action given as an operand's value, and a value off its enum, unrun" {
  local before
  before=$(cksum kit.yml)
  serve "$(call 1 packages '{"package":["add","style/yaml"]}')" "$(call 2 packages '{"package":["nosuch"]}')"
  answer 1 | jq --exit-status '.error | .code == -32602 and (.message | contains("add is a word kit packages acts on"))
    and (.message | contains("kit serve offers kit packages as a listing only"))'
  answer 2 | jq --exit-status '.error.message | contains("value nosuch is not one of: ")'
  [ "$(cksum kit.yml)" = "${before}" ]
  [ ! -e .kit/styles/yaml.md ]
}

@test "check, lint, toolchains, and tools say they run the repository's commands; a tool that reads only leaves it" {
  printf '# notes\n\n<!-- kit:generate x `touch ran; echo hi` -->\nhi\n<!-- kit:end x -->\n' > notes.md
  local before
  before=$(git status --porcelain; cksum kit.yml)
  serve "$(modern 1 tools/list)" "$(call 2 context '{"path":["notes.md"]}')" "$(call 3 kinds)" "$(call 4 packages)" \
    "$(call 5 help)" "$(call 6 version)"
  answer 1 | jq --exit-status '[.result.tools[] | select(.annotations.readOnlyHint | not) | .name]
    == ["check", "lint", "toolchains", "tools", "assign", "close", "gather", "settle", "unassign", "work"]'
  jq --exit-status --slurp '[.[] | select(.id >= 2) | .result.isError] == [false, false, false, false, false]' \
    "${BATS_TEST_TMPDIR}/out"
  [ ! -e ran ]
  [ "$(git status --porcelain; cksum kit.yml)" = "${before}" ]
}

@test "a path outside the repository is refused however it is spelled, and flow offers no workspace" {
  local outside=${BATS_TEST_TMPDIR}/outside.sh id
  printf '#!/usr/bin/env bash\necho $1\n' > "${outside}"
  ln -s "${outside}" link.sh
  serve "$(call 1 lint "$(jq --null-input --compact-output --arg path "${outside}" '{path: [$path]}')")" \
    "$(call 2 context '{"path":["../outside.sh"]}')" "$(call 3 kinds '{"path":["link.sh"]}')" \
    "$(call 4 context '{"path":["nosuch/../../outside.sh"]}')" \
    "$(call 5 kinds "$(jq --null-input --compact-output --arg path "${outside}" '{kind: [$path]}')")" \
    "$(call 6 flow '{"workspace":"x","stage":"desk"}')" "$(call 7 context '{"path":["kit.yml"]}')"
  for id in 1 2 3 4 5
  do
    answer "${id}" | jq --exit-status '.error | .code == -32602 and (.message | contains("is outside the repository"))'
  done
  answer 6 | jq --exit-status '.error | .code == -32602 and (.message | contains("unknown argument workspace"))'
  answer 7 | jq --exit-status '.result.isError == false'
}

@test "a path is refused when the server's directory is gone, so the root cannot be read" {
  local outside=${BATS_TEST_TMPDIR}/outside.sh arguments id
  printf '#!/usr/bin/env bash\necho $1\n' > "${outside}"
  arguments=$(jq --null-input --compact-output --arg path "${outside}" '{path: [$path]}')
  mkfifo "${BATS_TEST_TMPDIR}/in"
  "${kit_checkout}/bin/kit" serve <"${BATS_TEST_TMPDIR}/in" >"${BATS_TEST_TMPDIR}/out" 2>"${BATS_TEST_TMPDIR}/err" &
  server=$!
  exec 7>"${BATS_TEST_TMPDIR}/in"
  modern 1 server/discover >&7
  await 1
  cd "${BATS_TEST_TMPDIR}"
  rm -rf "${repo}"
  call 2 context "${arguments}" >&7
  call 3 kinds "${arguments}" >&7
  call 4 lint "${arguments}" >&7
  await 4
  exec 7>&-
  wait "${server}" || true
  for id in 2 3 4
  do
    answer "${id}" | jq --exit-status '.error | .code == -32602 and (.message | contains("is outside the repository"))'
  done
}

@test "kit.yml changed between messages is placed by the server, which tells the client the resources changed" {
  mkfifo "${BATS_TEST_TMPDIR}/in"
  "${kit_checkout}/bin/kit" serve <"${BATS_TEST_TMPDIR}/in" >"${BATS_TEST_TMPDIR}/out" 2>"${BATS_TEST_TMPDIR}/err" &
  server=$!
  exec 7>"${BATS_TEST_TMPDIR}/in"
  modern 1 server/discover >&7
  await 1
  [ ! -e .kit/workflow.md ]
  yq --inplace '.packages += ["workflow"]' kit.yml
  touch -t 209901010000 kit.yml
  modern 2 resources/list >&7
  await 2
  exec 7>&-
  wait "${server}"
  [ -e .kit/workflow.md ]
  answer 2 | jq --exit-status '[.result.resources[].uri] | index("kit://workflow.md")'
  grep --quiet '"method":"notifications/resources/list_changed"' "${BATS_TEST_TMPDIR}/out"
}

@test "each tool's whole help page is a resource" {
  serve "$(modern 1 resources/list)" "$(modern 2 resources/read '{"uri":"kit://help/context"}')" \
    "$(modern 3 resources/read '{"uri":"kit://help/commit"}')"
  [ "$(answer 1 | jq --raw-output '[.result.resources[].uri | select(startswith("kit://help/"))] | join(" ")')" = \
    "$(printf 'kit://help/%s ' check context flow help kinds lint packages syntax toolchains tools version \
      assign close gather settle unassign work | sed 's/ $//')" ]
  answer 2 | jq --exit-status '.result.contents[0] | .mimeType == "text/plain"
    and (.text | startswith("kit context - prints the rules that govern a file")) and (.text | contains("Description"))
    and (.text | contains("\u001b") | not)'
  answer 3 | jq --exit-status '.error.code == -32602'
}

@test "resources list the kit's files on disk, a read returns one, and a URI that names none is refused" {
  serve "$(modern 1 resources/list)" "$(modern 2 resources/read '{"uri":"kit://rules/bash"}')" \
    "$(modern 3 resources/read '{"uri":"kit://rules/../../etc/passwd"}')" "$(modern 4 resources/read '{"uri":"kit://1st.md"}')"
  local expected
  expected=$( { printf 'kit://1st.md\n'
    [[ ! -f .kit/workflow.md ]] || printf 'kit://workflow.md\n'
    for file in .kit/rules/*.md; do printf 'kit://rules/%s\n' "$(basename "${file}" .md)"; done
    for file in .kit/kinds/*.yml; do printf 'kit://kinds/%s\n' "$(basename "${file}" .yml)"; done
    printf 'kit://help/%s\n' check context flow help kinds lint packages syntax toolchains tools version \
      assign close gather settle unassign work; } )
  [ "$(answer 1 | jq --raw-output '.result.resources[].uri')" = "${expected}" ]
  [ "$(answer 2 | jq --raw-output '.result.contents[0].text')" = "$(cat .kit/rules/bash.md)" ]
  answer 2 | jq --exit-status '.result.contents[0].mimeType == "text/markdown"'
  answer 3 | jq --exit-status '.error.code == -32602'
  answer 4 | jq --exit-status '.result.contents[0].text | startswith("<!--")'
}

@test "the workflow package's file is a resource beside the entry file when the package is active, and not before" {
  serve "$(modern 1 resources/list)"
  run ! grep --quiet 'kit://workflow.md' "${BATS_TEST_TMPDIR}/out"
  "${kit_checkout}/bin/kit" packages add workflow >/dev/null 2>&1
  serve "$(modern 1 resources/list)" "$(modern 2 resources/read '{"uri":"kit://workflow.md"}')"
  [ "$(answer 1 | jq --raw-output '.result.resources[1].uri')" = "kit://workflow.md" ]
  [ "$(answer 2 | jq --raw-output '.result.contents[0].text')" = "$(cat .kit/workflow.md)" ]
}

@test "kit serve takes no arguments" {
  run "${kit_checkout}/bin/kit" serve --stdio
  [ "$status" -eq 2 ]
  [[ "$output" == *"run: kit help serve"* ]]
}

# await <id> waits up to ten seconds for the answer to request <id> in ${BATS_TEST_TMPDIR}/out.
await() {
  local tries=100
  until [[ -n "$(answer "$1" 2>/dev/null)" ]]
  do
    (( tries-- > 0 )) || return 1
    sleep 0.1
  done
}

@test "in a checkout of the kit, the checkout's own kit serves it, whichever kit started the server" {
  checkout-copy
  sed -i.bak "s/^serve_instructions='kit holds/serve_instructions='own: kit holds/" kit/modules/bash/serve.sh
  serve "$(modern 1 server/discover)"
  answer 1 | jq --exit-status '.result.instructions | startswith("own: ")'
  grep --quiet "this checkout's own kit serves it" "${BATS_TEST_TMPDIR}/err"
}

@test "between messages, a changed module starts the server again from it, which says the lists changed; one that does not parse is kept out" {
  checkout-copy
  mkfifo "${BATS_TEST_TMPDIR}/in"
  kit/bin/kit serve <"${BATS_TEST_TMPDIR}/in" >"${BATS_TEST_TMPDIR}/out" 2>"${BATS_TEST_TMPDIR}/err" &
  server=$!
  exec 7>"${BATS_TEST_TMPDIR}/in"
  modern 1 server/discover >&7
  await 1
  sed -i.bak "s/^serve_instructions='kit holds/serve_instructions='edited: kit holds/" kit/modules/bash/serve.sh
  modern 2 server/discover >&7
  await 2
  answer 2 | jq --exit-status '.result.instructions | startswith("edited: ")'
  [ "$(grep --count '"method":"notifications/tools/list_changed"' "${BATS_TEST_TMPDIR}/out")" -eq 1 ]
  [ "$(grep --count '"method":"notifications/resources/list_changed"' "${BATS_TEST_TMPDIR}/out")" -eq 1 ]
  printf 'serve:broken() {\n' >> kit/modules/bash/serve.sh
  modern 3 server/discover >&7
  await 3
  answer 3 | jq --exit-status '.result.instructions | startswith("edited: ")'
  grep --quiet 'does not parse; the server keeps the code it runs until the next change' "${BATS_TEST_TMPDIR}/err"
  exec 7>&-
  wait "${server}"
  grep --quiet ' event=reload$' .kit/logs/mcp-*.log
  grep --quiet ' event=reload-refused .*/kit/modules/bash/serve.sh$' .kit/logs/mcp-*.log
}

@test "a .kit/.gitignore that changed between messages tells the client the resources changed" {
  mkfifo "${BATS_TEST_TMPDIR}/in"
  "${kit_checkout}/bin/kit" serve <"${BATS_TEST_TMPDIR}/in" >"${BATS_TEST_TMPDIR}/out" 2>"${BATS_TEST_TMPDIR}/err" &
  server=$!
  exec 7>"${BATS_TEST_TMPDIR}/in"
  modern 1 server/discover >&7
  await 1
  touch .kit/.gitignore
  modern 2 server/discover >&7
  await 2
  exec 7>&-
  wait "${server}"
  grep --quiet '"method":"notifications/resources/list_changed"' "${BATS_TEST_TMPDIR}/out"
  run ! grep --quiet 'notifications/tools/list_changed' "${BATS_TEST_TMPDIR}/out"
}

# ----------------------------------------------------------------------------------------------------------------------
