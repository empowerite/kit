#!/usr/bin/env bats
# forge-bots.bats - the forge seam's bot-accounts: a GitHub App's JWT, its installations, and a token narrowed to one repository
# ----------------------------------------------------------------------------------------------------------------------
# curl is a stub, routed by URL, that answers as curl does with --dump-header and --write-out, so what is proven is what
# the provider asks of GitHub and what it makes of the answer. gh is a fake no call of the App may reach.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fake-tools gh
  curl-stub
  fresh-repo
  git remote add origin https://github.com/empowerite/kit.git
  openssl genrsa -out "${BATS_TEST_TMPDIR}/app.pem" 2048 2>/dev/null
  pem="${BATS_TEST_TMPDIR}/app.pem"
}

# call <call> <argument>… runs one call of the seam, as a verb would.
call() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && "forge:$0" "$@"' "$@"
}

# github <function> <argument>… runs one function of the GitHub provider.
github() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && use forge/github && "forge/github:$0" "$@"' "$@"
}

# curl-stub makes curl a stub that answers the routes curl-route adds: "<glob>\t<exit>\t<http code>\t<body>\t<headers>",
# the body and the headers being files in FAKE_DIR. As curl does, it dumps the headers to the --dump-header file and
# prints the --write-out text, with the http code, after the body; it ends with the exit status the route names.
curl-stub() {
  : > "${FAKE_DIR}/curl.routes"
  cat > "${FAKE_DIR}/bin/curl" <<'STUB'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "${FAKE_DIR}/curl.calls"
dump='' write='' args=("$@")
for (( i = 0; i < ${#args[@]}; i++ ))
do
  [[ "${args[i]}" == --dump-header ]] && dump=${args[i + 1]}
  [[ "${args[i]}" == --write-out ]] && write=${args[i + 1]}
done
cat > /dev/null
while IFS=$'\t' read -r glob status code body headers
do
  # shellcheck disable=SC2053  # the route is a glob on purpose
  [[ "$*" == ${glob} ]] || continue
  [[ -z "${dump}" ]] || { [[ -z "${headers}" ]] && : > "${dump}" || cp "${FAKE_DIR}/${headers}" "${dump}"; }
  [[ -z "${body}" ]] || cat "${FAKE_DIR}/${body}"
  printf '%s' "${write/'%{http_code}'/${code}}"
  exit "${status}"
done < "${FAKE_DIR}/curl.routes"
echo "curl: unrouted call: $*" >&2
exit 99
STUB
  chmod +x "${FAKE_DIR}/bin/curl"
}

# curl-route <glob> <exit> <http code> [<body> [<headers>]] adds a route for curl.
curl-route() {
  printf '%s\t%s\t%s\t%s\t%s\n' "$1" "$2" "$3" "${4:-}" "${5:-}" >> "${FAKE_DIR}/curl.routes"
}

# without <tool>… makes a PATH of the tools these tests need except <tool>…, in a folder of its own.
without() {
  local tool
  rm -rf "${BATS_TEST_TMPDIR}/bin"
  mkdir "${BATS_TEST_TMPDIR}/bin"
  for tool in bash git jq yq openssl curl date tr cat mktemp rm mkdir sed grep dirname basename sort head tail cp awk
  do
    [[ " $* " == *" ${tool} "* ]] || ln -sf "$(command -v "${tool}")" "${BATS_TEST_TMPDIR}/bin/${tool}"
  done
}

@test "forge/github:jwt signs what the key's public half verifies" {
  run github jwt 12345 "${pem}"
  [ "$status" -eq 0 ]
  IFS=. read -r head body signature <<<"$output"
  b64() { local text=$1; while (( ${#text} % 4 )); do text+==; done; tr '_-' '/+' <<<"${text}" | openssl base64 -d -A; }
  openssl rsa -in "${pem}" -pubout -out "${BATS_TEST_TMPDIR}/pub.pem" 2>/dev/null
  b64 "${signature}" > "${BATS_TEST_TMPDIR}/sig"
  printf '%s.%s' "${head}" "${body}" > "${BATS_TEST_TMPDIR}/signed"
  openssl dgst -sha256 -verify "${BATS_TEST_TMPDIR}/pub.pem" -signature "${BATS_TEST_TMPDIR}/sig" \
    "${BATS_TEST_TMPDIR}/signed"
  [ "$(b64 "${head}")" = '{"alg":"RS256","typ":"JWT"}' ]
  payload=$(b64 "${body}")
  [ "$(jq -r .iss <<<"${payload}")" = 12345 ]
  [ "$(jq '.exp - .iat' <<<"${payload}")" = 600 ]
  back=$(( $(date +%s) - $(jq .iat <<<"${payload}") ))
  [ "${back}" -ge 60 ] && [ "${back}" -le 62 ]
}

@test "forge/github:jwt fails, and prints no key, for a file that is no key or is missing" {
  printf 'not a key\n' > "${BATS_TEST_TMPDIR}/bad.pem"
  run --separate-stderr github jwt 1 "${BATS_TEST_TMPDIR}/bad.pem"
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *'no RSA private key'* ]]
  run --separate-stderr github jwt 1 "${BATS_TEST_TMPDIR}/none.pem"
  [ "$status" -eq 1 ]
  [[ "$stderr" == *'not a readable file'* ]]
}

@test "a machine without openssl or curl is told to install it" {
  without openssl
  PATH="${BATS_TEST_TMPDIR}/bin" run --separate-stderr "${BATS_TEST_TMPDIR}/bin/bash" -c \
    'source "${kit_root}/modules/bash/1st.sh" && use forge && use forge/github && forge/github:bot-app 1 "$1"' _ "${pem}"
  [ "$status" -eq 1 ]
  [[ "$stderr" == "❌ run: "*"kit tools install (openssl is not on PATH)" ]]
  without curl
  PATH="${BATS_TEST_TMPDIR}/bin" run --separate-stderr "${BATS_TEST_TMPDIR}/bin/bash" -c \
    'source "${kit_root}/modules/bash/1st.sh" && use forge && use forge/github && forge/github:bot-app 1 "$1"' _ "${pem}"
  [ "$status" -eq 1 ]
  [[ "$stderr" == "❌ run: "*"kit tools install (curl is not on PATH)" ]]
}

@test "forge:bot-app prints the App, and the JWT reaches curl on stdin, never in argv" {
  fixture app.json <<<'{"slug":"kit-bot","owner":{"login":"empowerite"},"permissions":{"issues":"write"},"html_url":"https://github.com/apps/kit-bot","id":1}'
  curl-route '*https://api.github.com/app' 0 200 app.json
  run call bot-app 12345 "${pem}"
  [ "$status" -eq 0 ]
  [ "$output" = '{"slug":"kit-bot","owner":"empowerite","permissions":{"issues":"write"},"html_url":"https://github.com/apps/kit-bot"}' ]
  ! grep --quiet 'Bearer\|eyJ' "${FAKE_DIR}/curl.calls"
  [ ! -e "${FAKE_DIR}/gh.calls" ]
}

@test "an answer that is no App is a read that failed, and says so in the App's words" {
  fixture odd.json <<<'[1]'
  curl-route '*/app' 0 200 odd.json
  run --separate-stderr call bot-app 1 "${pem}"
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"check the App's ID and key (the GitHub App could not be read)"* ]]
}

@test "forge:bot-installation prints the installation, and returns 1 on a 404 with no output" {
  fixture inst.json <<<'{"id":99,"permissions":{"issues":"write"},"repository_selection":"selected","html_url":"https://github.com/settings/installations/99","account":{}}'
  fixture missing.json <<<'{"message":"Not Found"}'
  curl-route '*/repos/empowerite/missing/installation' 22 404 missing.json
  curl-route '*/repos/empowerite/kit/installation' 0 200 inst.json
  run call bot-installation 1 "${pem}" empowerite/kit
  [ "$status" -eq 0 ]
  [ "$output" = '{"id":99,"permissions":{"issues":"write"},"repository_selection":"selected","html_url":"https://github.com/settings/installations/99"}' ]
  run --separate-stderr call bot-installation 1 "${pem}" empowerite/missing
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [ -z "$stderr" ]
}

@test "a 401 or 403 names the App's ID and key, and GitHub's reason" {
  fixture bad.json <<<'{"message":"A JSON web token could not be decoded"}'
  curl-route '*/app' 22 401 bad.json
  run --separate-stderr call bot-app 1 "${pem}"
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [ "$stderr" = "❌ check the App's ID and key (GitHub refused GET /app, HTTP 401: A JSON web token could not be decoded)" ]
  : > "${FAKE_DIR}/curl.routes"
  curl-route '*/app' 22 403 bad.json
  run --separate-stderr call bot-app 1 "${pem}"
  [[ "$stderr" == *"check the App's ID and key (GitHub refused GET /app, HTTP 403"* ]]
}

@test "a 404 that is not the installation's is said as GitHub's, and does not blame the key" {
  fixture missing.json <<<'{"message":"Not Found"}'
  curl-route '*/app/installations/9/access_tokens' 22 404 missing.json
  run --separate-stderr call bot-token 1 "${pem}" 9 kit
  [ "$status" -eq 1 ]
  [ "$stderr" = "❌ GitHub refused POST /app/installations/9/access_tokens, HTTP 404: Not Found" ]
}

@test "a host that cannot be reached is said so, with curl's exit, and does not blame the key" {
  for exit in 6 7 28 35
  do
    : > "${FAKE_DIR}/curl.routes"
    curl-route '*/app' "${exit}" 000
    run --separate-stderr call bot-app 1 "${pem}"
    [ "$status" -eq 1 ]
    [ "$stderr" = "❌ check the network (https://api.github.com could not be reached: curl exit ${exit}; GET /app was not made)" ]
  done
}

@test "a 5xx says GitHub failed, with its status, and does not blame the key" {
  fixture down.json <<<'{"message":"Server Error"}'
  curl-route '*/app' 22 503 down.json
  run --separate-stderr call bot-app 1 "${pem}"
  [ "$status" -eq 1 ]
  [ "$stderr" = "❌ try again later (GitHub failed on GET /app, HTTP 503: Server Error)" ]
  : > "${FAKE_DIR}/curl.routes"
  fixture html.html <<<'<html>bad gateway</html>'
  curl-route '*/app' 22 502 html.html
  run --separate-stderr call bot-app 1 "${pem}"
  [ "$stderr" = "❌ try again later (GitHub failed on GET /app, HTTP 502: )" ]
}

@test "a spent rate limit says when it resets from the response's headers, and gh is never asked" {
  fixture limited.json <<<'{"message":"API rate limit exceeded"}'
  printf 'HTTP/2 403\r\nX-RateLimit-Remaining: 0\r\nX-RateLimit-Reset: 4102444800\r\n\r\n' > "${FAKE_DIR}/limited.headers"
  curl-route '*/app' 22 403 limited.json limited.headers
  run --separate-stderr call bot-app 1 "${pem}"
  [ "$status" -eq 1 ]
  [[ "$stderr" == "❌ wait "*" seconds, until 2100-01-01T00:00:00Z: the App's rate limit is spent (GET /app was not made)" ]]
  printf 'HTTP/2 429\r\nx-ratelimit-remaining: 0\r\n\r\n' > "${FAKE_DIR}/limited.headers"
  run --separate-stderr call bot-app 1 "${pem}"
  [ "$stderr" = "❌ wait for the App's rate limit to reset (GET /app was not made)" ]
  fixture secondary.json <<<'{"message":"You have exceeded a secondary rate limit."}'
  : > "${FAKE_DIR}/curl.routes"
  curl-route '*/app' 22 403 secondary.json
  run --separate-stderr call bot-app 1 "${pem}"
  [[ "$stderr" == "❌ wait a few minutes: GitHub's secondary rate limit refused GET /app (HTTP 403)" ]]
  [ ! -e "${FAKE_DIR}/gh.calls" ]
}

@test "a reset already past waits 0 seconds" {
  fixture limited.json <<<'{"message":"API rate limit exceeded"}'
  printf 'x-ratelimit-remaining: 0\nx-ratelimit-reset: 1\n' > "${FAKE_DIR}/limited.headers"
  curl-route '*/app' 22 403 limited.json limited.headers
  run --separate-stderr call bot-app 1 "${pem}"
  [[ "$stderr" == "❌ wait 0 seconds, until 1970-01-01T00:00:01Z:"* ]]
}

@test "forge:bot-installations prints one object per installation, over every page" {
  fixture page1.json <<<"$(jq -c -n '[range(100) | {id: ., account: {login: "org\(.)"}, repository_selection: "all", html_url: "u"}]')"
  fixture page2.json <<<'[{"id":500,"account":{"login":"last"},"repository_selection":"selected","html_url":"v"}]'
  curl-route '*/app/installations?per_page=100&page=2' 0 200 page2.json
  curl-route '*/app/installations?per_page=100&page=1' 0 200 page1.json
  run call bot-installations 1 "${pem}"
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 101 ]
  [ "${lines[0]}" = '{"id":0,"account":"org0","repository_selection":"all","html_url":"u"}' ]
  [ "${lines[100]}" = '{"id":500,"account":"last","repository_selection":"selected","html_url":"v"}' ]
}

@test "a list that reaches the limit fails, and prints none of it" {
  fixture page1.json <<<"$(jq -c -n '[range(100) | {id: ., account: {login: "o"}}]')"
  curl-route '*/app/installations?*' 0 200 page1.json
  run --separate-stderr bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && forge_list_limit=100 \
    && forge:bot-installations 1 "$1"' _ "${pem}"
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *'kit reads at most 100 items of a list'* ]]
}

@test "a list answer that is no list is a read that failed, and says so" {
  fixture odd.json <<<'{"message":"odd"}'
  curl-route '*/app/installations?*' 0 200 odd.json
  run --separate-stderr call bot-installations 1 "${pem}"
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"check the App's ID and key (the answer of /app/installations could not be read)"* ]]
}

@test "forge:bot-token asks for the one repository and prints the token and its expiry" {
  fixture token.json <<<'{"token":"ghs_abc","expires_at":"2026-11-12T10:00:00Z","permissions":{}}'
  curl-route '*/app/installations/99/access_tokens' 0 201 token.json
  run call bot-token 1 "${pem}" 99 empowerite/kit
  [ "$status" -eq 0 ]
  [ "$output" = '{"token":"ghs_abc","expires_at":"2026-11-12T10:00:00Z"}' ]
  grep --quiet -- '--request POST' "${FAKE_DIR}/curl.calls"
  grep --quiet -F '"repositories":["kit"]' "${FAKE_DIR}/curl.calls"
}

@test "an App, an installation, or a repository that cannot go in a URL is refused before any request" {
  run --separate-stderr call bot-app 12x "${pem}"
  [ "$status" -eq 1 ]
  [ "$stderr" = "❌ '12x' is not the ID of a GitHub App: it is a number" ]
  run --separate-stderr call bot-token 1 "${pem}" '9/../x' kit
  [ "$stderr" = "❌ '9/../x' is not the ID of an installation: it is a number" ]
  for repo in 'a/b?x' 'a/b#x' 'a/../b' 'a/b/c' 'ab' '/b' 'a/' 'a/.' './b' '-a/b' 'a/-b' '../b'
  do
    run --separate-stderr call bot-installation 1 "${pem}" "${repo}"
    [ "$status" -eq 1 ]
    [ "$stderr" = "❌ '${repo}' is not a repository: it is owner/name" ]
  done
  for repo in 'a?x' '..' '.' '-a' 'a#'
  do
    run --separate-stderr call bot-token 1 "${pem}" 9 "${repo}"
    [ "$status" -eq 1 ]
    [ "$stderr" = "❌ '${repo}' is not the name of a repository" ]
  done
  for repo in 'a/b/c' '-a/b' 'a/.' './b' 'a?/b'
  do
    run --separate-stderr call bot-token 1 "${pem}" 9 "${repo}"
    [ "$status" -eq 1 ]
    [ "$stderr" = "❌ '${repo}' is not a repository: it is owner/name" ]
  done
  run --separate-stderr call bot-installations x "${pem}"
  [ "$status" -eq 1 ]
  [ ! -e "${FAKE_DIR}/curl.calls" ]
}

@test "forge:bot-repositories prints the full names the token reaches, and forge:bot-revoke deletes the token" {
  fixture repos.json <<<'{"total_count":2,"repositories":[{"full_name":"empowerite/kit"},{"full_name":"empowerite/other"}]}'
  fixture none.json <<<''
  curl-route '*/installation/repositories?*' 0 200 repos.json
  curl-route '*/installation/token' 0 204 none.json
  run call bot-repositories ghs_abc
  [ "$status" -eq 0 ]
  [ "$output" = $'empowerite/kit\nempowerite/other' ]
  run call bot-revoke ghs_abc
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  grep --quiet -- '--request DELETE' "${FAKE_DIR}/curl.calls"
  grep --quiet 'https://api.github.com/installation/token$' "${FAKE_DIR}/curl.calls"
  ! grep --quiet ghs_abc "${FAKE_DIR}/curl.calls"
}

@test "a token that reaches no repository prints nothing" {
  fixture empty.json <<<'{"total_count":0,"repositories":[]}'
  curl-route '*/installation/repositories?*' 0 200 empty.json
  run call bot-repositories ghs_abc
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "the API base is api.github.com for github.com and /api/v3 for another host" {
  run github api-base
  [ "$output" = https://api.github.com ]
  git remote set-url origin git@ghe.example.com:empowerite/kit.git
  run github api-base
  [ "$output" = https://ghe.example.com/api/v3 ]
}

@test "the API base is api.github.com when the primary remote names no host" {
  git remote remove origin
  run github api-base
  [ "$status" -eq 0 ]
  [ "$output" = https://api.github.com ]
}

@test "a repository whose name starts with a dot, as .github does, is narrowed to by its name, with or without its owner" {
  fixture token.json <<<'{"token":"t","expires_at":"2026-11-12T10:00:00Z"}'
  curl-route '*/app/installations/9/access_tokens' 0 201 token.json
  run call bot-token 1 "${pem}" 9 empowerite/.github
  [ "$status" -eq 0 ]
  grep --quiet -F '"repositories":[".github"]' "${FAKE_DIR}/curl.calls"
  run call bot-token 1 "${pem}" 9 .github
  [ "$status" -eq 0 ]
}
