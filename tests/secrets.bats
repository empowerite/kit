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
  ! grep -F MIIEow "${FAKE_DIR}/secret-tool.calls"
  PATH="${lean}" run kit-bash 'secrets:foreign kit-bot github.com/12 | base64 -d | cat; secrets:delete kit-bot github.com/12 && ! secrets:get kit-bot github.com/12'
  [ "$status" -eq 0 ]
}
