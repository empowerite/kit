#!/usr/bin/env bats
# forge-bots.bats - the forge seam's bot-accounts: a GitHub App's JWT, its installations, and a token narrowed to one repository
# ----------------------------------------------------------------------------------------------------------------------
# curl is the fake, routed by URL, so what is proven is what the provider asks of GitHub and what it makes of the answer.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fake-tools gh curl
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
  [ "$(( $(date +%s) - $(jq .iat <<<"${payload}") ))" -ge 60 ]
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

@test "forge:bot-app prints the App, and the JWT reaches curl on stdin, never in argv" {
  fixture app.json <<<'{"slug":"kit-bot","owner":{"login":"empowerite"},"permissions":{"issues":"write"},"html_url":"https://github.com/apps/kit-bot","id":1}'
  route curl '*https://api.github.com/app' app.json
  run call bot-app 12345 "${pem}"
  [ "$status" -eq 0 ]
  [ "$output" = '{"slug":"kit-bot","owner":"empowerite","permissions":{"issues":"write"},"html_url":"https://github.com/apps/kit-bot"}' ]
  ! grep --quiet 'Bearer\|eyJ' "${FAKE_DIR}/curl.calls"
}

@test "forge:bot-installation prints the installation, and returns 1 on a 404 with no output" {
  fixture inst.json <<<'{"id":99,"permissions":{"issues":"write"},"repository_selection":"selected","html_url":"https://github.com/settings/installations/99","account":{}}'
  fixture missing.json <<<'{"message":"Not Found"}'
  route curl '*/repos/empowerite/missing/installation' '!22 >missing.json'
  route curl '*/repos/empowerite/kit/installation' inst.json
  run call bot-installation 1 "${pem}" empowerite/kit
  [ "$status" -eq 0 ]
  [ "$output" = '{"id":99,"permissions":{"issues":"write"},"repository_selection":"selected","html_url":"https://github.com/settings/installations/99"}' ]
  run --separate-stderr call bot-installation 1 "${pem}" empowerite/missing
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [ -z "$stderr" ]
}

@test "forge:bot-installation says what GitHub refused when it is no 404" {
  fixture bad.json <<<'{"message":"A JSON web token could not be decoded"}'
  route curl '*/repos/empowerite/kit/installation' '!22 >bad.json'
  run --separate-stderr call bot-installation 1 "${pem}" empowerite/kit
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"check the App's ID and key"*'A JSON web token could not be decoded'* ]]
}

@test "forge:bot-installations prints one object per installation, over every page" {
  fixture page1.json <<<"$(jq -c -n '[range(100) | {id: ., account: {login: "org\(.)"}, repository_selection: "all", html_url: "u"}]')"
  fixture page2.json <<<'[{"id":500,"account":{"login":"last"},"repository_selection":"selected","html_url":"v"}]'
  route curl '*/app/installations?per_page=100&page=2' page2.json
  route curl '*/app/installations?per_page=100&page=1' page1.json
  run call bot-installations 1 "${pem}"
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 101 ]
  [ "${lines[0]}" = '{"id":0,"account":"org0","repository_selection":"all","html_url":"u"}' ]
  [ "${lines[100]}" = '{"id":500,"account":"last","repository_selection":"selected","html_url":"v"}' ]
}

@test "forge:bot-token asks for the one repository and prints the token and its expiry" {
  fixture token.json <<<'{"token":"ghs_abc","expires_at":"2026-11-12T10:00:00Z","permissions":{}}'
  route curl '*/app/installations/99/access_tokens' token.json
  run call bot-token 1 "${pem}" 99 empowerite/kit
  [ "$status" -eq 0 ]
  [ "$output" = '{"token":"ghs_abc","expires_at":"2026-11-12T10:00:00Z"}' ]
  grep --quiet -- '--request POST' "${FAKE_DIR}/curl.calls"
  grep --quiet -F '"repositories":["kit"]' "${FAKE_DIR}/curl.calls"
}

@test "forge:bot-repositories prints the full names the token reaches, and forge:bot-revoke deletes the token" {
  fixture repos.json <<<'{"total_count":2,"repositories":[{"full_name":"empowerite/kit"},{"full_name":"empowerite/other"}]}'
  fixture none.json <<<''
  route curl '*/installation/repositories?*' repos.json
  route curl '*/installation/token' none.json
  run call bot-repositories ghs_abc
  [ "$status" -eq 0 ]
  [ "$output" = $'empowerite/kit\nempowerite/other' ]
  run call bot-revoke ghs_abc
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  grep --quiet -- '--request DELETE .*/installation/token' "${FAKE_DIR}/curl.calls"
  ! grep --quiet ghs_abc "${FAKE_DIR}/curl.calls"
}

@test "the API base is api.github.com for github.com and /api/v3 for another host" {
  run github api-base
  [ "$output" = https://api.github.com ]
  git remote set-url origin git@ghe.example.com:empowerite/kit.git
  run github api-base
  [ "$output" = https://ghe.example.com/api/v3 ]
}

@test "a rate limit is told as the provider tells it, and not as a refusal of the key" {
  fixture limited.json <<<'{"message":"API rate limit exceeded"}'
  fixture limits.json <<<'{"resources":{"core":{"limit":60,"remaining":0,"reset":1800000060}}}'
  route curl '*/app' '!22 >limited.json'
  route gh 'api rate_limit*' limits.json
  run --separate-stderr call bot-app 1 "${pem}"
  [ "$status" -eq 1 ]
  [[ "$stderr" == *'could not be read'* ]]
  [[ "$stderr" != *"the App's ID and key"* ]]
}

@test "a machine without curl is told to install it" {
  local tool
  mkdir "${BATS_TEST_TMPDIR}/bin"
  for tool in bash git jq yq openssl date tr cat mktemp rm mkdir sed grep dirname basename sort head tail
  do
    ln -s "$(command -v "${tool}")" "${BATS_TEST_TMPDIR}/bin/${tool}" 2>/dev/null || true
  done
  PATH="${BATS_TEST_TMPDIR}/bin" run --separate-stderr "${BATS_TEST_TMPDIR}/bin/bash" -c \
    'source "${kit_root}/modules/bash/1st.sh" && use forge && use forge/github && forge/github:bot-app 1 "$1"' _ "${pem}"
  [ "$status" -eq 1 ]
  [[ "$stderr" == *curl* ]]
}
