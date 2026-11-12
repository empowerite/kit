#!/usr/bin/env bats
# secrets.bats - the secret store: a key kept in the macOS keychain or the Linux Secret Service, never in argv
# ----------------------------------------------------------------------------------------------------------------------
# tests/fakes/fake-secrets stands in for security and secret-tool, and keeps each item as a file under FAKE_DIR/secrets.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
  fresh-repo >/dev/null
  export FAKE_DIR="${BATS_TEST_TMPDIR}/fake"
  mkdir -p "${FAKE_DIR}/bin"
  ln -sf "${BATS_TEST_DIRNAME}/fakes/fake-secrets" "${FAKE_DIR}/bin/security"
  export PATH="${FAKE_DIR}/bin:${PATH}"
  kit_bash_modules=secrets
  PEM=$'-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA\n-----END RSA PRIVATE KEY-----'
  export PEM
}

# only-secret-tool makes a directory with the fake as secret-tool alone, among the tools the kit and the module use, and
# not security, and sets lean to it: a command run with PATH="${lean}" has no keychain.
only-secret-tool() {
  local bin=${BATS_TEST_TMPDIR}/only tool
  mkdir -p "${bin}"
  for tool in bash env dirname basename cat tr base64 mkdir rm od sed git uname readlink mktemp
  do
    ln -sf "$(command -v "${tool}")" "${bin}/${tool}"
  done
  ln -sf "${BATS_TEST_DIRNAME}/fakes/fake-secrets" "${bin}/secret-tool"
  lean=${bin}
}

@test "secrets:put then secrets:get gives back a PEM with its newlines" {
  run --separate-stderr kit-bash 'printf "%s\n" "${PEM}" | secrets:put kit-bot github.com/12 && secrets:get kit-bot github.com/12'
  [ "$status" -eq 0 ]
  [ "$output" = "${PEM}" ]
}

@test "the secret never reaches argv" {
  run kit-bash 'printf "%s\n" "${PEM}" | secrets:put kit-bot github.com/12'
  [ "$status" -eq 0 ]
  [ -s "${FAKE_DIR}/security.calls" ]
  ! grep -F -e MIIEow -e "$(printf '%s\n' "${PEM}" | base64 | tr -d '\n' | cut -c1-40)" "${FAKE_DIR}/security.calls"
}

@test "secrets:put replaces an item and secrets:delete removes it, and again with none" {
  run kit-bash 'printf one | secrets:put kit-bot h/1 && printf two | secrets:put kit-bot h/1 && secrets:get kit-bot h/1'
  [ "$status" -eq 0 ]
  [ "$output" = two ]
  run kit-bash 'secrets:delete kit-bot h/1 && secrets:delete kit-bot h/1 && ! secrets:get kit-bot h/1'
  [ "$status" -eq 0 ]
}

@test "secrets:get of no item returns 1 and prints nothing" {
  run --separate-stderr kit-bash 'secrets:get kit-bot github.com/99'
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [ -z "$stderr" ]
}

@test "secrets:get of a store that cannot be read says so and returns 2, apart from no item, which returns 1" {
  FAKE_SECRETS_FAIL_READ=1 run --separate-stderr kit-bash 'secrets:get kit-bot github.com/99'
  [ "$status" -eq 2 ]
  [ -z "$output" ]
  [[ "$stderr" == *'the keychain could not be read for kit-bot github.com/99 (security: User interaction is not allowed.)'* ]]
  only-secret-tool
  FAKE_SECRETS_FAIL_READ=1 PATH="${lean}" run --separate-stderr kit-bash 'secrets:get kit-bot github.com/99'
  [ "$status" -eq 2 ]
  [[ "$stderr" == *'the secret-service could not be read for kit-bot github.com/99 (secret-tool: Cannot autolaunch D-Bus'* ]]
  PATH="${lean}" run --separate-stderr kit-bash 'secrets:get kit-bot github.com/99'
  [ "$status" -eq 1 ]
  [ -z "$stderr" ]
}

@test "secrets:foreign un-hexes what security hex-encodes" {
  mkdir -p "${FAKE_DIR}/secrets"
  printf '%s\n' "${PEM}" > "${FAKE_DIR}/secrets/other@github.com_7"
  run --separate-stderr kit-bash 'secrets:foreign other github.com/7'
  [ "$status" -eq 0 ]
  [ "$output" = "${PEM}" ]
}

@test "with neither security nor secret-tool on PATH, secrets:backend names secret-tool and returns 1" {
  only-secret-tool
  rm "${BATS_TEST_TMPDIR}/only/secret-tool"
  PATH="${lean}" run --separate-stderr kit-bash 'secrets:backend'
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"install secret-tool (libsecret) so kit can keep a key; kit keeps none in a file"* ]]
}

@test "secrets:backend is the keychain where security is, and the secret service where only secret-tool is" {
  run kit-bash 'secrets:backend'
  [ "$output" = keychain ]
  only-secret-tool
  PATH="${lean}" run kit-bash 'secrets:backend'
  [ "$output" = secret-service ]
}

@test "secret-tool is the backend where security is missing" {
  only-secret-tool
  PATH="${lean}" run --separate-stderr kit-bash 'printf "%s\n" "${PEM}" | secrets:put kit-bot github.com/12 && secrets:get kit-bot github.com/12'
  [ "$status" -eq 0 ]
  [ "$output" = "${PEM}" ]
  grep -q 'store' "${FAKE_DIR}/secret-tool.calls"
  run ! grep -F -e MIIEow -e "$(printf '%s\n' "${PEM}" | base64 | tr -d '\n' | cut -c1-40)" "${FAKE_DIR}/secret-tool.calls"
  PATH="${lean}" run kit-bash 'secrets:foreign kit-bot github.com/12'
  [ "$output" = "$(printf '%s\n' "${PEM}" | base64 | tr -d '\n')" ]
  PATH="${lean}" run kit-bash 'secrets:delete kit-bot github.com/12 && ! secrets:get kit-bot github.com/12'
  [ "$status" -eq 0 ]
}

@test "secrets:get does not un-hex a stored value made only of hex digits" {
  mkdir -p "${FAKE_DIR}/secrets"
  printf 'ABCD' > "${FAKE_DIR}/secrets/kit-bot@h_1"
  run kit-bash 'secrets:get kit-bot h/1 | base64'
  [ "$status" -eq 0 ]
  [ "$output" = ABCD ]
}

@test "secrets:put that the backend does not keep is an error naming the backend, and returns 1" {
  FAKE_SECRETS_REFUSE=1 run --separate-stderr kit-bash 'printf x | secrets:put kit-bot h/1'
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"the keychain did not keep kit-bot h/1"* ]]
  only-secret-tool
  FAKE_SECRETS_REFUSE=1 PATH="${lean}" run --separate-stderr kit-bash 'printf x | secrets:put kit-bot h/1'
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"the secret-service did not keep kit-bot h/1"* ]]
}

@test "secrets:put whose backend fails is an error naming the backend, and returns 1" {
  rm "${FAKE_DIR}/bin/security"
  printf '#!/usr/bin/env bash\nexit 3\n' > "${FAKE_DIR}/bin/security"
  chmod +x "${FAKE_DIR}/bin/security"
  run --separate-stderr kit-bash 'printf x | secrets:put kit-bot h/1'
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"the keychain did not store kit-bot h/1"* ]]
}

@test "secrets:put whose base64 fails stores nothing and returns 1" {
  printf '#!/usr/bin/env bash\nexit 1\n' > "${FAKE_DIR}/bin/base64"
  chmod +x "${FAKE_DIR}/bin/base64"
  run kit-bash 'printf x | secrets:put kit-bot h/1'
  [ "$status" -eq 1 ]
  [ ! -e "${FAKE_DIR}/security.calls" ]
}

@test "secrets:delete of an item the backend will not delete is an error naming the backend, and returns 1" {
  run kit-bash 'printf x | secrets:put kit-bot h/1'
  FAKE_SECRETS_FAIL_DELETE=1 run --separate-stderr kit-bash 'secrets:delete kit-bot h/1'
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"the keychain did not delete kit-bot h/1"* ]]
  only-secret-tool
  PATH="${lean}" run kit-bash 'printf x | secrets:put kit-bot h/2'
  FAKE_SECRETS_FAIL_DELETE=1 PATH="${lean}" run --separate-stderr kit-bash 'secrets:delete kit-bot h/2'
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"the secret-service did not delete kit-bot h/2"* ]]
}

@test "secrets:delete of no item succeeds through secret-tool too" {
  only-secret-tool
  PATH="${lean}" run kit-bash 'secrets:delete kit-bot h/none'
  [ "$status" -eq 0 ]
}
