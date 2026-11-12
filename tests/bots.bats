#!/usr/bin/env bats
# bots.bats - kit bots: the bots file, choosing a bot, and --add, --remove, and the list
# ----------------------------------------------------------------------------------------------------------------------
# curl is a stub, routed by URL, as in forge-bots.bats; tests/fakes/fake-secrets stands in for the keychain. Each test
# runs with an XDG_CONFIG_HOME of its own, so the bots file starts missing.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
  fake-tools gh
  curl-stub
  ln -sf "${BATS_TEST_DIRNAME}/fakes/fake-secrets" "${FAKE_DIR}/bin/security"
  fresh-repo >/dev/null
  git remote add origin https://github.com/empowerite/kit.git
  openssl genrsa -out "${BATS_TEST_TMPDIR}/app.pem" 2048 2>/dev/null
  pem="${BATS_TEST_TMPDIR}/app.pem"
  kit_bash_modules=bots
  bots_file="${XDG_CONFIG_HOME}/kit/bots.yml"
}

# curl-stub, curl-route: as forge-bots.bats has them.
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
  [[ -z "${dump}" ]] || : > "${dump}"
  [[ -z "${body}" ]] || cat "${FAKE_DIR}/${body}"
  printf '%s' "${write/'%{http_code}'/${code}}"
  exit "${status}"
done < "${FAKE_DIR}/curl.routes"
echo "curl: unrouted call: $*" >&2
exit 99
STUB
  chmod +x "${FAKE_DIR}/bin/curl"
}

curl-route() {
  printf '%s\t%s\t%s\t%s\t%s\n' "$1" "$2" "$3" "${4:-}" "${5:-}" >> "${FAKE_DIR}/curl.routes"
}

# github-routes <host> <app> <slug> routes what --add asks of a GitHub that knows the App and its installation on the
# repository, with a token it may mint and revoke.
github-routes() {
  local api=https://api.github.com
  [[ "$1" == github.com ]] || api=https://$1/api/v3
  fixture app.json <<<"{\"slug\":\"$3\",\"owner\":{\"login\":\"empowerite\"},\"permissions\":{},
    \"html_url\":\"https://$1/apps/$3\"}"
  fixture inst.json <<<'{"id":99,"permissions":{},"repository_selection":"selected","html_url":"https://github.com/settings/installations/99"}'
  fixture token.json <<<'{"token":"ghs_abc","expires_at":"2026-11-12T10:00:00Z"}'
  fixture none.json <<<''
  curl-route "*${api}/app" 0 200 app.json
  curl-route "*/repos/empowerite/kit/installation" 0 200 inst.json
  curl-route "*/app/installations/99/access_tokens" 0 201 token.json
  curl-route "*/installation/token" 0 204 none.json
}

# called <pattern> succeeds when a curl call matches the pattern: its --write-out puts a newline inside the line.
called() { tr '\n' ' ' < "${FAKE_DIR}/curl.calls" | grep --quiet -- "$1"; }

# stored <service> <account> prints what the fake keychain holds, as kit stored it.
stored() { base64 -d < "${FAKE_DIR}/secrets/$1@${2//\//_}"; }

write-bots() {
  mkdir -p "$(dirname "${bots_file}")"
  cat > "${bots_file}"
}

two-bots() {
  write-bots <<'YAML'
# my bots
bots:
  - name: kit-bot
    host: github.com
    app: 12345
    owner: empowerite
    jobs: [work, review]
  - name: other-bot
    host: github.com
    app: 777
    owner: empowerite
    jobs: [review]
YAML
}

@test "kit bots lists none where the file is missing, as text and JSON" {
  run --separate-stderr kit bots
  [ "$status" -eq 0 ]
  [[ "$output" == *'no bots'*'kit bots --add='* ]]
  run kit bots --json
  [ "$status" -eq 0 ]
  [ "$output" = "{\"file\":\"${bots_file}\",\"bots\":[]}" ]
}

@test "kit bots lists each bot, with whether the store holds its key" {
  two-bots
  printf 'k' | kit-bash 'secrets:put kit-bot github.com/12345'
  run kit bots --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.bots[] | {name, key}]' <<<"$output")" = '[{"name":"kit-bot","key":true},{"name":"other-bot","key":false}]' ]
  [ "$(jq -c '.bots[0]' <<<"$output")" = '{"name":"kit-bot","host":"github.com","app":12345,"owner":"empowerite","jobs":["work","review"],"key":true}' ]
  run kit bots
  [ "$status" -eq 0 ]
  [[ "$output" == *kit-bot*github.com*12345*empowerite*work,review* ]]
  [[ "$output" == *other-bot*777*'no key'* ]]
  run kit bots --jq
  [ "$status" -eq 0 ]
  [[ "$output" == *'"bots": ['* ]]
}

@test "--add proves the key before it stores anything" {
  github-routes github.com 12345 kit-bot
  run --separate-stderr kit bots --add=kit-bot --app=12345 --key="${pem}"
  [ "$status" -eq 0 ]
  [ "$(yq -o=json -I=0 '.bots[0]' "${bots_file}")" = '{"name":"kit-bot","host":"github.com","app":12345,"owner":"empowerite","jobs":["work","review"]}' ]
  [ "$(stored kit-bot github.com/12345)" = "$(cat "${pem}")" ]
  called '--request POST.*/app/installations/99/access_tokens'
  called '--request DELETE.*/installation/token'
  slice=$(base64 < "${pem}" | tr -d '\n' | cut -c1-40)
  ! grep --quiet -F -e BEGIN -e "${slice}" "${FAKE_DIR}/security.calls" "${FAKE_DIR}/curl.calls"
  [[ "$output" == *'kit-bot'* ]]
}

@test "--add keeps the comments of a file that is there, and says which jobs it sets up" {
  two-bots
  github-routes github.com 31 new-bot
  run kit bots --add=new-bot --app=31 --key="${pem}" --jobs=work:workflows
  [ "$status" -eq 0 ]
  head -n 1 "${bots_file}" | grep --quiet '^# my bots'
  [ "$(yq -o=json -I=0 '.bots | map(.name)' "${bots_file}")" = '["kit-bot","other-bot","new-bot"]' ]
  [ "$(yq -o=json -I=0 '.bots[2].jobs' "${bots_file}")" = '["work","workflows"]' ]
}

@test "--add of a bot again replaces its entry and its key" {
  github-routes github.com 12345 kit-bot
  run kit bots --add=kit-bot --app=12345 --key="${pem}"
  [ "$status" -eq 0 ]
  run kit bots --add=kit-bot --app=12345 --key="${pem}" --jobs=review --jobs=workflows
  [ "$status" -eq 0 ]
  [ "$(yq -o=json -I=0 '.bots | length' "${bots_file}")" = 1 ]
  [ "$(yq -o=json -I=0 '.bots[0].jobs' "${bots_file}")" = '["review","workflows"]' ]
}

@test "--add with an App ID the key does not sign for stores nothing" {
  fixture bad.json <<<'{"message":"A JSON web token could not be decoded"}'
  curl-route '*/app' 22 401 bad.json
  run --separate-stderr kit bots --add=kit-bot --app=12345 --key="${pem}"
  [ "$status" -eq 1 ]
  [[ "$stderr" == *'12345'* ]]
  [ ! -e "${bots_file}" ]
  [ ! -e "${FAKE_DIR}/secrets" ] || [ -z "$(ls "${FAKE_DIR}/secrets")" ]
}

@test "--add reads keychain:<service>/<account>, as security gives it in hex" {
  github-routes github.com 4906610 kit-bot
  mkdir -p "${FAKE_DIR}/secrets"
  cp "${pem}" "${FAKE_DIR}/secrets/jig-app@4906610"
  run kit bots --add=kit-bot --app=4906610 --key=keychain:jig-app/4906610
  [ "$status" -eq 0 ]
  [ "$(stored kit-bot github.com/4906610)" = "$(cat "${pem}")" ]
}

@test "--add refuses a keychain item that is not there" {
  run --separate-stderr kit bots --add=kit-bot --app=1 --key=keychain:jig-app/1
  [ "$status" -eq 2 ]
  [[ "$stderr" == *'jig-app/1'* ]]
  [ ! -e "${bots_file}" ]
  run --separate-stderr kit bots --add=kit-bot --app=1 --key=keychain:jig-app
  [ "$status" -eq 2 ]
  [[ "$stderr" == *'keychain:<service>/<account>'* ]]
}

@test "--add takes a PKCS#8 key with CRLF line ends, and keeps it with LF" {
  github-routes github.com 12345 kit-bot
  openssl pkcs8 -topk8 -nocrypt -in "${pem}" | sed 's/$/\r/' > "${BATS_TEST_TMPDIR}/crlf.pem"
  grep --quiet 'BEGIN PRIVATE KEY' "${BATS_TEST_TMPDIR}/crlf.pem"
  run kit bots --add=kit-bot --app=12345 --key="${BATS_TEST_TMPDIR}/crlf.pem"
  [ "$status" -eq 0 ]
  [[ "$(stored kit-bot github.com/12345)" != *$'\r'* ]]
  [[ "$(stored kit-bot github.com/12345)" == *'BEGIN PRIVATE KEY'* ]]
}

@test "--add refuses a slug that differs from the name, and stores nothing" {
  github-routes github.com 12345 real-slug
  run --separate-stderr kit bots --add=kit-bot --app=12345 --key="${pem}"
  [ "$status" -eq 2 ]
  [[ "$stderr" == *'real-slug'*'--add=real-slug'* ]]
  [ ! -e "${bots_file}" ]
  [ ! -e "${FAKE_DIR}/secrets" ] || [ -z "$(ls "${FAKE_DIR}/secrets")" ]
}

@test "--add with this repository out of the App's reach proves with its first installation, narrowing nothing" {
  fixture missing.json <<<'{"message":"Not Found"}'
  fixture list.json <<<'[{"id":99,"account":{"login":"empowerite"},"repository_selection":"selected","html_url":"https://github.com/settings/installations/99"}]'
  curl-route '*/repos/empowerite/kit/installation' 22 404 missing.json
  curl-route '*/app/installations?per_page*' 0 200 list.json
  github-routes github.com 12345 kit-bot
  run kit bots --add=kit-bot --app=12345 --key="${pem}"
  [ "$status" -eq 0 ]
  called '--request POST.*/app/installations/99/access_tokens'
  ! grep --quiet 'repositories' "${FAKE_DIR}/curl.calls"
  called '--data {"permissions":{"metadata":"read"}} '
  called '--request DELETE'
}

@test "--add with no installation at all names the install page and stores nothing" {
  fixture missing.json <<<'{"message":"Not Found"}'
  fixture empty.json <<<'[]'
  curl-route '*/repos/empowerite/kit/installation' 22 404 missing.json
  curl-route '*/app/installations?per_page*' 0 200 empty.json
  github-routes github.com 12345 kit-bot
  run --separate-stderr kit bots --add=kit-bot --app=12345 --key="${pem}"
  [ "$status" -eq 1 ]
  [[ "$stderr" == *'https://github.com/apps/kit-bot/installations/new'* ]]
  [ ! -e "${bots_file}" ]
  [ ! -e "${FAKE_DIR}/secrets" ] || [ -z "$(ls "${FAKE_DIR}/secrets")" ]
}

@test "--add reports an installation lookup that failed, not a missing installation" {
  fixture oops.json <<<'{"message":"Server Error"}'
  curl-route '*/repos/empowerite/kit/installation' 22 500 oops.json
  github-routes github.com 12345 kit-bot
  run --separate-stderr kit bots --add=kit-bot --app=12345 --key="${pem}"
  [ "$status" -eq 1 ]
  [[ "$stderr" == *'HTTP 500'* ]]
  [ ! -e "${bots_file}" ]
}

@test "--add records the primary remote's host, and asks that host's API" {
  git remote set-url origin https://ghe.example.com/empowerite/kit.git
  printf 'forge: github\n' > kit.yml
  github-routes ghe.example.com 12345 kit-bot
  run kit bots --add=kit-bot --app=12345 --key="${pem}"
  [ "$status" -eq 0 ]
  [ "$(yq '.bots[0].host' "${bots_file}")" = ghe.example.com ]
  [ -f "${FAKE_DIR}/secrets/kit-bot@ghe.example.com_12345" ]
}

@test "--add works outside a repository, for github.com" {
  github-routes github.com 12345 kit-bot
  fixture list.json <<<'[{"id":99,"account":{"login":"empowerite"},"repository_selection":"all","html_url":"https://github.com/settings/installations/99"}]'
  curl-route '*/app/installations?per_page*' 0 200 list.json
  mkdir "${BATS_TEST_TMPDIR}/outside"
  cd "${BATS_TEST_TMPDIR}/outside"
  run kit bots --add=kit-bot --app=12345 --key="${pem}"
  [ "$status" -eq 0 ]
  [ "$(yq '.bots[0].host' "${bots_file}")" = github.com ]
  [ -f "${FAKE_DIR}/secrets/kit-bot@github.com_12345" ]
}

@test "--add refuses what is not a name, an App ID, a job, or a key, and an option it lacks" {
  run --separate-stderr kit bots --add='bad name' --app=1 --key="${pem}"
  [ "$status" -eq 2 ]
  run --separate-stderr kit bots --add=kit-bot --app=x1 --key="${pem}"
  [ "$status" -eq 2 ]
  run --separate-stderr kit bots --add=kit-bot --app=1 --key="${pem}" --jobs=sweep
  [ "$status" -eq 2 ]
  [[ "$stderr" == *'sweep'*'work, review, workflows'* ]]
  run --separate-stderr kit bots --add=kit-bot --app=1 --key="${BATS_TEST_TMPDIR}/none.pem"
  [ "$status" -eq 2 ]
  [[ "$stderr" == *none.pem* ]]
  run --separate-stderr kit bots --add=kit-bot --key="${pem}"
  [ "$status" -eq 2 ]
  [[ "$stderr" == *'--app'* ]]
  run --separate-stderr kit bots --add=kit-bot --app=1
  [ "$status" -eq 2 ]
  [[ "$stderr" == *'--key'* ]]
  run --separate-stderr kit bots --app=1
  [ "$status" -eq 2 ]
  run --separate-stderr kit bots --add=kit-bot --remove=kit-bot
  [ "$status" -eq 2 ]
  [ ! -e "${bots_file}" ]
  [ ! -e "${FAKE_DIR}/curl.calls" ]
}

@test "--add with no secret store refuses before it asks the forge" {
  run --separate-stderr kit-bash 'PATH=/nonexistent; bots:add kit-bot 1 /dev/null'
  [ "$status" -eq 1 ]
  [[ "$stderr" == *'install secret-tool'* ]]
}

@test "--remove forgets the bot and deletes its key" {
  two-bots
  printf 'k' | kit-bash 'secrets:put kit-bot github.com/12345'
  run kit bots --remove=kit-bot
  [ "$status" -eq 0 ]
  [ "$(yq -o=json -I=0 '.bots | map(.name)' "${bots_file}")" = '["other-bot"]' ]
  head -n 1 "${bots_file}" | grep --quiet '^# my bots'
  [ ! -e "${FAKE_DIR}/secrets/kit-bot@github.com_12345" ]
}

@test "--remove of a bot that is not there is a usage error naming the bots" {
  run --separate-stderr kit bots --remove=ghost
  [ "$status" -eq 2 ]
  [[ "$stderr" == *ghost* ]]
  two-bots
  run --separate-stderr kit bots --remove=ghost
  [ "$status" -eq 2 ]
  [[ "$stderr" == *'kit-bot'*'other-bot'* ]]
}

@test "--remove keeps the bot when the store will not delete its key" {
  two-bots
  printf 'k' | kit-bash 'secrets:put kit-bot github.com/12345'
  FAKE_SECRETS_FAIL_DELETE=1 run --separate-stderr kit bots --remove=kit-bot
  [ "$status" -eq 1 ]
  [ "$(yq '.bots | length' "${bots_file}")" = 2 ]
}

@test "bots:choose: the named bot, else bot: in kit.yml, else the only one on the host" {
  two-bots
  run kit-bash 'bots:choose other-bot | jq -r .name'
  [ "$output" = other-bot ]
  printf 'bot: other-bot\n' > kit.yml
  run kit-bash 'bots:choose | jq -r .name'
  [ "$output" = other-bot ]
  run kit-bash 'bots:choose kit-bot | jq -r .name'
  [ "$output" = kit-bot ]
  rm kit.yml
  yq --inplace 'del(.bots[1])' "${bots_file}"
  run kit-bash 'bots:choose | jq -r .name'
  [ "$output" = kit-bot ]
}

@test "two bots on one host and no choice: exit 2, naming both" {
  two-bots
  run --separate-stderr kit-bash 'bots:choose'
  [ "$status" -eq 2 ]
  [ -z "$output" ]
  [[ "$stderr" == *'kit-bot'*'other-bot'*'bot: in kit.yml'* ]]
}

@test "bots:choose: no bots, an unknown name, and a bot: that names none, exit 2" {
  run --separate-stderr kit-bash 'bots:choose'
  [ "$status" -eq 2 ]
  [[ "$stderr" == *'kit bots --add='* ]]
  two-bots
  run --separate-stderr kit-bash 'bots:choose ghost'
  [ "$status" -eq 2 ]
  [[ "$stderr" == *ghost*kit-bot*other-bot* ]]
  printf 'bot: ghost\n' > kit.yml
  run --separate-stderr kit-bash 'bots:choose'
  [ "$status" -eq 2 ]
  [[ "$stderr" == *ghost* ]]
}

@test "bots:choose with bots only on another host says so, naming both hosts" {
  git remote set-url origin https://gitlab.example.org/g/kit.git
  two-bots
  run --separate-stderr kit-bash 'bots:choose'
  [ "$status" -eq 2 ]
  [[ "$stderr" == *'gitlab.example.org'*'github.com'* ]]
  run --separate-stderr kit-bash 'bots:choose kit-bot'
  [ "$status" -eq 2 ]
  [[ "$stderr" == *'gitlab.example.org'*'github.com'* ]]
}

@test "bots:key writes the PEM, mode 600, and says how to add a key that is not there" {
  github-routes github.com 12345 kit-bot
  kit bots --add=kit-bot --app=12345 --key="${pem}"
  run kit-bash 'bots:key "$(bots:choose)" "${BATS_TEST_TMPDIR}/out.pem"'
  [ "$status" -eq 0 ]
  cmp "${pem}" "${BATS_TEST_TMPDIR}/out.pem"
  [ "$(stat -c %a "${BATS_TEST_TMPDIR}/out.pem" 2>/dev/null || stat -f %Lp "${BATS_TEST_TMPDIR}/out.pem")" = 600 ]
  rm "${FAKE_DIR}/secrets/kit-bot@github.com_12345"
  rm "${BATS_TEST_TMPDIR}/out.pem"
  run --separate-stderr kit-bash 'bots:key "$(bots:choose)" "${BATS_TEST_TMPDIR}/out.pem"'
  [ "$status" -eq 1 ]
  [[ "$stderr" == *'no key for kit-bot in the keychain; run: '*'kit bots --add=kit-bot --app=12345 --key=<file>'* ]]
  [ ! -s "${BATS_TEST_TMPDIR}/out.pem" ]
}

@test "bots:jobs-need prints the permissions the jobs need, write covering read" {
  run kit-bash 'bots:jobs-need work review'
  [ "$status" -eq 0 ]
  [ "$output" = '{"actions":"read","checks":"read","contents":"write","issues":"write","metadata":"read","pull_requests":"write","statuses":"read"}' ]
  run kit-bash 'bots:jobs-need review'
  [ "$output" = '{"checks":"read","contents":"read","metadata":"read","pull_requests":"write"}' ]
  run kit-bash 'bots:jobs-need workflows'
  [ "$output" = '{"metadata":"read","workflows":"write"}' ]
  run kit-bash 'bots:jobs-need'
  [ "$output" = '{"metadata":"read"}' ]
  run --separate-stderr kit-bash 'bots:jobs-need sweep'
  [ "$status" -eq 2 ]
}

@test "a bots file that is no list of bots is refused" {
  write-bots <<<'bots: 3'
  run --separate-stderr kit bots
  [ "$status" -eq 1 ]
  [[ "$stderr" == *'bots.yml'* ]]
}

@test "bot: in kit.yml is a key settings:validate accepts" {
  kit_bash_modules=settings
  printf 'bot: kit-bot\n' > kit.yml
  run kit-bash 'settings:validate'
  [ "$status" -eq 0 ]
}

@test "a bot with no jobs in the file has work and review, in the list and in bots:choose" {
  write-bots <<'YAML'
bots:
  - name: kit-bot
    host: github.com
    app: 12345
YAML
  run kit bots --json
  [ "$(jq -c '.bots[0].jobs' <<<"$output")" = '["work","review"]' ]
  run kit-bash 'bots:choose | jq -c .jobs'
  [ "$output" = '["work","review"]' ]
}

@test "a comment inside an entry survives --add of another bot and --remove" {
  write-bots <<'YAML'
bots:
  - name: kit-bot
    host: github.com
    app: 12345
    # the App I made for work
    owner: empowerite
    jobs: [work]
YAML
  github-routes github.com 31 new-bot
  run kit bots --add=new-bot --app=31 --key="${pem}"
  [ "$status" -eq 0 ]
  grep --quiet '# the App I made for work' "${bots_file}"
  run kit bots --remove=new-bot
  [ "$status" -eq 0 ]
  grep --quiet '# the App I made for work' "${bots_file}"
}

@test "--add writes a plain one-line comment into a new file, not the repository's banner" {
  github-routes github.com 12345 kit-bot
  run kit bots --add=kit-bot --app=12345 --key="${pem}"
  [ "$status" -eq 0 ]
  [ "$(head -n 1 "${bots_file}")" = '# bots.yml - the bots that act for you; kit reads and edits it, and keeps your comments' ]
  [ "$(sed -n 2p "${bots_file}")" = 'bots:' ]
}

@test "--add refuses a bots file that is malformed before it asks the forge or stores anything" {
  github-routes github.com 12345 kit-bot
  write-bots <<<'bots: 3'
  run --separate-stderr kit bots --add=kit-bot --app=12345 --key="${pem}"
  [ "$status" -eq 1 ]
  [[ "$stderr" == *'bots.yml'* ]]
  [ ! -e "${FAKE_DIR}/curl.calls" ]
  [ ! -e "${FAKE_DIR}/secrets" ] || [ -z "$(ls "${FAKE_DIR}/secrets")" ]
}

@test "--add refuses a list form and an empty --jobs, and --remove a list form" {
  run --separate-stderr kit bots --add=kit-bot --app=1 --key="${pem}" --text
  [ "$status" -eq 2 ]
  [[ "$stderr" == *'--add'* ]]
  run --separate-stderr kit bots --add=kit-bot --app=1 --key="${pem}" --jobs=
  [ "$status" -eq 2 ]
  [[ "$stderr" == *'--jobs'* ]]
  run --separate-stderr kit bots --add=kit-bot --app=1 --key="${pem}" --jobs=work::review
  [ "$status" -eq 2 ]
  run --separate-stderr kit bots --remove=kit-bot --json
  [ "$status" -eq 2 ]
  [ ! -e "${FAKE_DIR}/curl.calls" ]
}

# one-bot <jobs> sets up the one bot, kit-bot, with its key kept and a GitHub that knows it.
one-bot() {
  write-bots <<YAML
bots:
  - name: kit-bot
    host: github.com
    app: 12345
    owner: empowerite
    jobs: [$1]
YAML
  kit-bash 'secrets:put kit-bot github.com/12345' < "${pem}"
  github-routes github.com 12345 kit-bot
}

@test "--check names what a job lacks and what no job needs" {
  one-bot work
  fixture inst.json <<<'{"id":99,"permissions":{"metadata":"read","contents":"write","issues":"write","pull_requests":"write","statuses":"read","actions":"write"},"repository_selection":"selected","html_url":"https://github.com/settings/installations/99"}'
  run --separate-stderr kit bots --check --json
  [ "$status" -eq 1 ]
  [ "$(jq -c '{name, repository, reached, lacks, extra}' <<<"$output")" = '{"name":"kit-bot","repository":"empowerite/kit","reached":true,"lacks":{"work":["checks: read"]},"extra":["actions: write"]}' ]
  run --separate-stderr kit bots --check
  [ "$status" -eq 1 ]
  [[ "$output" == *'reaches empowerite/kit'* ]]
  [[ "$output" == *'work'*'checks: read'* ]]
  [[ "$output" == *'actions: write'* ]]
  run --separate-stderr kit bots --check --bot=kit-bot --jq
  [ "$status" -eq 1 ]
  [[ "$output" == *'"reached": true'* ]]
}

@test "--check of an installation that fits its jobs says so" {
  one-bot review
  fixture inst.json <<<'{"id":99,"permissions":{"metadata":"read","pull_requests":"write","contents":"read","checks":"read"},"repository_selection":"all","html_url":"https://github.com/settings/installations/99"}'
  run --separate-stderr kit bots --check
  [ "$status" -eq 0 ]
  [[ "$output" == *'match its jobs'* ]]
  run --separate-stderr kit bots --check --json
  [ "$(jq -c '[.lacks, .extra]' <<<"$output")" = '[{},[]]' ]
}

@test "--check says where the installation does not reach this repository" {
  one-bot work
  fixture none.json <<<'{"message":"Not Found"}'
  : > "${FAKE_DIR}/curl.routes"
  curl-route "*api.github.com/app" 0 200 app.json
  curl-route "*/repos/empowerite/kit/installation" 22 404 none.json
  fixture list.json <<<'[{"id":99,"account":{"login":"empowerite"},"repository_selection":"selected","html_url":"https://github.com/settings/installations/99"}]'
  curl-route '*/app/installations?per_page*' 0 200 list.json
  run --separate-stderr kit bots --check
  [ "$status" -eq 1 ]
  [[ "$output" == *'does not reach empowerite/kit'*'https://github.com/settings/installations/99'* ]]
  run --separate-stderr kit bots --check --json
  [ "$status" -eq 1 ]
  [ "$(jq -c '{reached, page}' <<<"$output")" = '{"reached":false,"page":"https://github.com/settings/installations/99"}' ]
}

@test "--check outside a repository judges the App's permissions and says it judged no repository" {
  one-bot work
  fixture app.json <<<'{"slug":"kit-bot","owner":{"login":"empowerite"},"permissions":{"metadata":"read","contents":"write","issues":"write","pull_requests":"write","checks":"read","statuses":"read","actions":"read"},"html_url":"https://github.com/apps/kit-bot"}'
  fixture list.json <<<'[{"id":99,"account":{"login":"empowerite"},"repository_selection":"all","html_url":"https://github.com/settings/installations/99"}]'
  curl-route '*/app/installations?per_page*' 0 200 list.json
  mkdir "${BATS_TEST_TMPDIR}/outside"
  cd "${BATS_TEST_TMPDIR}/outside"
  run --separate-stderr kit bots --check
  [ "$status" -eq 0 ]
  [[ "$output" == *'no repository'* ]]
  [[ "$output" == *'match its jobs'* ]]
  run --separate-stderr kit bots --check --json
  [ "$(jq -c '{repository, reached, installation: .installation.id}' <<<"$output")" = '{"repository":null,"reached":null,"installation":99}' ]
}

@test "--check outside a repository of an App installed nowhere says so, names the page, exit 1" {
  one-bot work
  fixture empty.json <<<'[]'
  curl-route '*/app/installations?per_page*' 0 200 empty.json
  mkdir "${BATS_TEST_TMPDIR}/outside"
  cd "${BATS_TEST_TMPDIR}/outside"
  run --separate-stderr kit bots --check
  [ "$status" -eq 1 ]
  [[ "$output" == *'installed nowhere'*'https://github.com/apps/kit-bot/installations/new'* ]]
}

@test "--check of an App installed nowhere names the page to install it" {
  one-bot work
  : > "${FAKE_DIR}/curl.routes"
  curl-route "*api.github.com/app" 0 200 app.json
  curl-route "*/repos/empowerite/kit/installation" 22 404 none.json
  fixture empty.json <<<'[]'
  curl-route '*/app/installations?per_page*' 0 200 empty.json
  run --separate-stderr kit bots --check
  [ "$status" -eq 1 ]
  [[ "$output" == *'https://github.com/apps/kit-bot/installations/new'* ]]
}

@test "--check with no key says how to add it" {
  one-bot work
  rm "${FAKE_DIR}"/secrets/*
  run --separate-stderr kit bots --check
  [ "$status" -eq 1 ]
  [[ "$stderr" == *'kit bots --add=kit-bot'* ]]
}

@test "--check refuses a bot it does not know, and a form it lacks" {
  one-bot work
  run --separate-stderr kit bots --check --bot=ghost
  [ "$status" -eq 2 ]
  [[ "$stderr" == *ghost* ]]
  run --separate-stderr kit bots --bot=kit-bot
  [ "$status" -eq 2 ]
  run --separate-stderr kit bots --check --bot=
  [ "$status" -eq 2 ]
  run --separate-stderr kit bots --check token
  [ "$status" -eq 2 ]
  run --separate-stderr kit bots token --json
  [ "$status" -eq 2 ]
  run --separate-stderr kit bots --check --add=x --app=1 --key=k
  [ "$status" -eq 2 ]
}

@test "token prints only the token" {
  one-bot work
  run --separate-stderr kit bots token
  [ "$status" -eq 0 ]
  [ "$output" = ghs_abc ]
  called '--request POST.*/app/installations/99/access_tokens'
  grep --quiet -F '"repositories":["kit"]' "${FAKE_DIR}/curl.calls"
  ! called '--request DELETE.*/installation/token'
  run --separate-stderr kit bots --bot=kit-bot token
  [ "$output" = ghs_abc ]
}

@test "token outside a repository refuses, exit 1" {
  one-bot work
  mkdir "${BATS_TEST_TMPDIR}/outside"
  cd "${BATS_TEST_TMPDIR}/outside"
  run --separate-stderr kit bots token
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *'repository'* ]]
}

@test "token where the installation does not reach names kit bots --check" {
  one-bot work
  fixture none.json <<<'{"message":"Not Found"}'
  : > "${FAKE_DIR}/curl.routes"
  curl-route "*/repos/empowerite/kit/installation" 22 404 none.json
  run --separate-stderr kit bots token
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *'kit bots --check'* ]]
}

@test "--check judges every job, write covering read, and an admin held is extra" {
  one-bot 'work, review, workflows'
  fixture inst.json <<<'{"id":99,"permissions":{"metadata":"read","contents":"admin","pull_requests":"write","workflows":"read","checks":"write"},"repository_selection":"selected","html_url":"https://github.com/settings/installations/99"}'
  run --separate-stderr kit bots --check --json
  [ "$(jq -c '[.lacks | keys[]]' <<<"$output")" = '["work","workflows"]' ]
  [ "$(jq -c '.lacks.workflows' <<<"$output")" = '["workflows: write"]' ]
  [ "$(jq -c '.extra' <<<"$output")" = '["checks: write","contents: admin"]' ]
}

# sent <pattern> counts the curl calls that match the pattern, a call's --write-out putting a newline inside it.
sent() { tr '\n' ' ' < "${FAKE_DIR}/curl.calls" | grep --only-matching -- "$1" | wc -l | tr -d ' '; }

# last-sent prints the request of the last curl call.
last-sent() { tr '\n' ' ' < "${FAKE_DIR}/curl.calls" | grep --only-matching -- '--request [A-Z]*' | tail -n 1; }

# run-gh makes the gh run's shim execs: it records its arguments and the GH_TOKEN it was given, and answers a login.
run-gh() {
  rm -f "${FAKE_DIR}/bin/gh"
  cat > "${FAKE_DIR}/bin/gh" <<'GH'
#!/usr/bin/env bash
printf '%s GH_TOKEN=%s\n' "$*" "${GH_TOKEN-}" >> "${FAKE_DIR}/gh.calls"
[[ "$*" == 'api user'* ]] && echo octocat
exit 0
GH
  chmod +x "${FAKE_DIR}/bin/gh"
}

# run-setup is one-bot work, whose token is good for an hour, a gh of run-gh's kind, and a runtime directory of the
# test's own, which kit bots run makes its session directory in.
run-setup() {
  one-bot work
  fixture token.json <<<"{\"token\":\"ghs_abc\",\"expires_at\":\"$(jq --null-input --raw-output 'now + 3600 | floor | todate')\"}"
  run-gh
  export XDG_RUNTIME_DIR="${BATS_TEST_TMPDIR}/runtime"
  mkdir -p "${XDG_RUNTIME_DIR}"
}

@test "run gives the command GH_TOKEN, KIT_BOT, KIT_BOT_LOGIN, and KIT_BOT_FOR" {
  run-setup
  run --separate-stderr kit bots run bash -c 'printf "%s|%s|%s|%s\n" "${GH_TOKEN}" "${KIT_BOT}" "${KIT_BOT_LOGIN}" "${KIT_BOT_FOR}"'
  [ "$status" -eq 0 ]
  [ "$output" = 'ghs_abc|kit-bot|kit-bot[bot]|octocat' ]
  [[ "$output$stderr" != *'BEGIN'* ]]
  run --separate-stderr kit bots --bot=kit-bot run bash -c 'echo "${KIT_BOT}"'
  [ "$output" = kit-bot ]
}

@test "run keeps the person a nested run was given, and the words after run are the command's" {
  run-setup
  run --separate-stderr env KIT_BOT_FOR=someone kit bots run bash -c 'echo "${KIT_BOT_FOR} $*"' x --other -v
  [ "$status" -eq 0 ]
  [ "$output" = 'someone --other -v' ]
  run --separate-stderr kit bots run echo --json
  [ "$status" -eq 2 ]
  ! grep --quiet '^api user' "${FAKE_DIR}/gh.calls" 2> /dev/null
}

@test "run passes stdin to the command" {
  run-setup
  run --separate-stderr bash -c 'echo piped | kit bots run cat'
  [ "$status" -eq 0 ]
  [ "$output" = piped ]
}

@test "run's gh mints a new token when the old is within ten minutes of expiry" {
  run-setup
  fixture token.json <<<"{\"token\":\"ghs_one\",\"expires_at\":\"$(jq --null-input --raw-output 'now + 300 | floor | todate')\"}"
  cat > "${BATS_TEST_TMPDIR}/command" <<'COMMAND'
#!/usr/bin/env bash
echo "first ${GH_TOKEN}"
printf '{"token":"ghs_two","expires_at":"%s"}' "$(jq --null-input --raw-output 'now + 3600 | floor | todate')" \
  > "${FAKE_DIR}/token.json"
gh api user
gh api user
COMMAND
  chmod +x "${BATS_TEST_TMPDIR}/command"
  run --separate-stderr kit bots run "${BATS_TEST_TMPDIR}/command"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = 'first ghs_one' ]
  # both calls had the second token: the first minted it, the second used it unexpired
  [ "$(grep --count 'GH_TOKEN=ghs_two' "${FAKE_DIR}/gh.calls")" -eq 2 ]
  [ "$(sent '--request POST')" -eq 2 ]
  # both tokens were good, so both were revoked
  [ "$(sent '--request DELETE')" -eq 2 ]
}

@test "run's gh keeps a token that has more than ten minutes left" {
  run-setup
  fixture token.json <<<"{\"token\":\"ghs_one\",\"expires_at\":\"$(jq --null-input --raw-output 'now + 3000 | floor | todate')\"}"
  run --separate-stderr kit bots run gh api user
  [ "$status" -eq 0 ]
  [ "$(grep --count 'GH_TOKEN=ghs_one' "${FAKE_DIR}/gh.calls")" -eq 1 ]
  [ "$(sent '--request POST')" -eq 1 ]
}

@test "run's gh re-mints in the checkout run started in, whichever folder the command is in" {
  run-setup
  fixture token.json <<<"{\"token\":\"ghs_one\",\"expires_at\":\"$(jq --null-input --raw-output 'now + 60 | floor | todate')\"}"
  mkdir "${BATS_TEST_TMPDIR}/elsewhere"
  run --separate-stderr kit bots run bash -c "cd '${BATS_TEST_TMPDIR}/elsewhere' && gh api user"
  [ "$status" -eq 0 ]
  grep --quiet -F '"repositories":["kit"]' "${FAKE_DIR}/curl.calls"
  [ "$(sent '--request POST')" -eq 2 ]
}

# without-gh puts in PATH every directory of it that has a gh as a directory of links to all but the gh, so no gh is
# found, and every other tool is.
without-gh() {
  local dir tool kept='' IFS=:
  for dir in ${PATH}
  do
    if [[ -x "${dir}/gh" ]]
    then
      mkdir -p "${BATS_TEST_TMPDIR}/nogh${dir}"
      for tool in "${dir}"/*
      do
        [[ "${tool##*/}" == gh ]] || ln -sf "${tool}" "${BATS_TEST_TMPDIR}/nogh${dir}/"
      done
      dir=${BATS_TEST_TMPDIR}/nogh${dir}
    fi
    kept+=${kept:+:}${dir}
  done
  PATH=${kept}
}

@test "run's gh says how to install gh where there is none, and run itself still runs" {
  one-bot work
  rm -f "${FAKE_DIR}/bin/gh"
  without-gh
  ! command -v gh
  export XDG_RUNTIME_DIR="${BATS_TEST_TMPDIR}/runtime" KIT_BOT_FOR=someone
  mkdir -p "${XDG_RUNTIME_DIR}"
  run --separate-stderr kit bots run bash -c 'gh api user; echo "gh exited $?"'
  [ "$status" -eq 0 ]
  [[ "$output" == *'gh exited 127'* ]]
  [[ "$stderr" == *'gh is not installed'*'https://cli.github.com'* ]]
}

@test "git under run asks the helper, which answers with the token" {
  run-setup
  run --separate-stderr kit bots run bash -c 'printf "protocol=https\nhost=github.com\n\n" | git credential fill'
  [ "$status" -eq 0 ]
  [[ "$output" == *$'username=x-access-token\npassword=ghs_abc'* ]]
  # a host that is not the forge's gets no answer from the helper, so git has none to give
  run --separate-stderr kit bots run bash -c 'printf "protocol=https\nhost=example.com\nusername=u\npassword=p\n\n" | git credential fill'
  [[ "$output" != *ghs_abc* ]]
  # and storing or erasing asks nothing of it
  run --separate-stderr kit bots run bash -c 'git credential-kit-bot erase </dev/null; echo "erase $?"'
  [[ "$output" == *'erase 0'* ]]
}

@test "run keeps the git configuration the caller exported, and the helper comes after it" {
  run-setup
  run --separate-stderr env GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=user.name GIT_CONFIG_VALUE_0=Someone \
    kit bots run bash -c 'git config user.name; echo "${GIT_CONFIG_COUNT}"; git config --get-all credential.https://github.com.helper | wc -l'
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = Someone ]
  [ "${lines[1]}" = 3 ]
  [ "${lines[2]//[[:space:]]/}" = 2 ]
}

@test "run's session directory is the person's alone, and holds the token only there" {
  run-setup
  run --separate-stderr kit bots run bash -c 'dir=$(dirname "$(dirname "$(command -v gh)")"); stat -f %Lp "${dir}" 2>/dev/null || stat -c %a "${dir}"; ls "${dir}"; ls "${dir}/bin"'
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = 700 ]
  [[ "$output" == *token.json* ]]
  [[ "$output" == *git-credential-kit-bot* ]]
  # the token is in no argv: curl is told it on stdin
  ! grep --quiet 'ghs_abc' "${FAKE_DIR}/curl.calls"
}

@test "run passes the command's exit status through, revokes the token, and removes the directory" {
  run-setup
  run --separate-stderr kit bots run false
  [ "$status" -eq 1 ]
  [ "$(last-sent)" = '--request DELETE' ]
  [ -z "$(ls -A "${XDG_RUNTIME_DIR}")" ]
  run --separate-stderr kit bots run bash -c 'exit 7'
  [ "$status" -eq 7 ]
  run -127 --separate-stderr kit bots run kit-bot-no-such-command
  [ -z "$(ls -A "${XDG_RUNTIME_DIR}")" ]
}

@test "run revokes when the command is interrupted by TERM or INT, and the command is signalled too" {
  run-setup
  run --separate-stderr kit bots run bash -c 'trap "echo got-term >&2; exit 0" TERM; kill -TERM "${PPID}"; sleep 1; echo late'
  [ "$status" -eq 143 ]
  [[ "$stderr" == *got-term* ]]
  [[ "$output" != *late* ]]
  [ "$(last-sent)" = '--request DELETE' ]
  [ -z "$(ls -A "${XDG_RUNTIME_DIR}")" ]
  : > "${FAKE_DIR}/curl.calls"
  run --separate-stderr kit bots run bash -c 'trap "echo got-int >&2; exit 0" INT; kill -INT "${PPID}"; sleep 1; echo late'
  [ "$status" -eq 130 ]
  [[ "$stderr" == *got-int* ]]
  [ "$(last-sent)" = '--request DELETE' ]
  [ -z "$(ls -A "${XDG_RUNTIME_DIR}")" ]
}

@test "run warns where the revoke fails, still removes the directory, and passes the status through" {
  run-setup
  fixture gone.json <<<'{"message":"Bad credentials"}'
  : > "${FAKE_DIR}/curl.routes"
  curl-route "*/repos/empowerite/kit/installation" 0 200 inst.json
  curl-route "*/app/installations/99/access_tokens" 0 201 token.json
  curl-route "*/installation/token" 22 401 gone.json
  curl-route "*api.github.com/app" 0 200 app.json
  run --separate-stderr kit bots run bash -c 'exit 3'
  [ "$status" -eq 3 ]
  [[ "$stderr" == *'revoke'* ]]
  [[ "$stderr" != *ghs_abc* ]]
  [ -z "$(ls -A "${XDG_RUNTIME_DIR}")" ]
}

@test "run refuses, minting nothing and leaving no directory, outside a repository and for an unknown bot" {
  run-setup
  mkdir "${BATS_TEST_TMPDIR}/outside"
  cd "${BATS_TEST_TMPDIR}/outside"
  run --separate-stderr env KIT_BOT_FOR=someone kit bots run true
  [ "$status" -eq 1 ]
  [[ "$stderr" == *repository* ]]
  [ -z "$(ls -A "${XDG_RUNTIME_DIR}")" ]
  cd "${repo}"
  run --separate-stderr kit bots --bot=ghost run true
  [ "$status" -eq 2 ]
  [[ "$stderr" == *ghost* ]]
  run --separate-stderr kit bots run
  [ "$status" -eq 2 ]
  run --separate-stderr kit bots --check run true
  [ "$status" -eq 2 ]
}

@test "run says why where the person's login cannot be read" {
  run-setup
  cat > "${FAKE_DIR}/bin/gh" <<'GH'
#!/usr/bin/env bash
echo "not logged in" >&2
exit 1
GH
  run --separate-stderr kit bots run true
  [ "$status" -eq 1 ]
  [[ "$stderr" == *'gh auth login'* ]]
  [ -z "$(ls -A "${XDG_RUNTIME_DIR}")" ]
}

@test "run makes its directory under TMPDIR where there is no XDG_RUNTIME_DIR" {
  run-setup
  unset XDG_RUNTIME_DIR
  mkdir "${BATS_TEST_TMPDIR}/tmpdir"
  run --separate-stderr env TMPDIR="${BATS_TEST_TMPDIR}/tmpdir" kit bots run bash -c 'command -v gh'
  [ "$status" -eq 0 ]
  [[ "$output" == "${BATS_TEST_TMPDIR}/tmpdir/kit-bot."*/bin/gh ]]
  [ -z "$(ls -A "${BATS_TEST_TMPDIR}/tmpdir")" ]
}
