# shellcheck shell=bash
# npm.sh - the npm dependency provider: package.json names the packages, package-lock.json pins them (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# npm is usable where npm is on PATH, and package.json and package-lock.json are at the repository root. It pins the
# packages package.json names under dependencies and devDependencies, at the versions the lock holds. What is
# installed is read from node_modules/.package-lock.json, which npm writes on every install.

dependencies/npm:file() { printf 'package.json\n'; }

dependencies/npm:install() { npm ci --no-audit --no-fund; }

dependencies/npm:lock() { printf 'package-lock.json\n'; }

# dependencies/npm:outdated asks the registry. npm outdated exits 1 when a package is behind, so its answer, not its
# status, says whether it could ask.
dependencies/npm:outdated() {
  local doc
  doc=$(npm outdated --json 2>/dev/null) || true  # its status is 1 when a package is behind; the answer says more
  if ! jq --exit-status 'type == "object" and (has("error") | not)' <<<"${doc:-{\}}" >/dev/null 2>&1
  then
    error "npm outdated could not ask the registry"
    return 1
  fi
  jq --raw-output 'to_entries[] | select(.value | type == "object")
    | [.key, (.value.current // ""), .value.latest] | @tsv' <<<"${doc:-{\}}"
}

dependencies/npm:path() { printf '%s/node_modules/.bin\n' "${PWD}"; }

# dependencies/npm:pins reads the three files, and never runs npm. A package not installed has no installed version.
dependencies/npm:pins() {
  local installed=node_modules/.package-lock.json
  [[ -f "${installed}" ]] || installed=/dev/null
  jq --null-input --raw-output --slurpfile manifest package.json --slurpfile lock package-lock.json \
    --slurpfile installed "${installed}" '
      ($lock[0].packages // {}) as $pinned | ($installed[0].packages // {}) as $have
      | ($manifest[0].dependencies // {}) + ($manifest[0].devDependencies // {}) | keys[]
      | [., ($pinned["node_modules/" + .].version // ""), ($have["node_modules/" + .].version // "")] | @tsv'
}

# dependencies/npm:update installs the latest of each package behind. npm keeps each where package.json has it, and
# rewrites both files.
dependencies/npm:update() {
  local behind
  behind=$(dependencies/npm:outdated) || return 1
  [[ -n "${behind}" ]] || return 0
  # shellcheck disable=SC2046  # one package per word
  npm install --no-audit --no-fund $(cut -f 1 <<<"${behind}" | sed 's/$/@latest/')
}

dependencies/npm:usable() {
  command -v npm >/dev/null 2>&1 && [[ -f package.json && -f package-lock.json ]]
}

# The seam's calls end here; what follows is this provider's own.

# ----------------------------------------------------------------------------------------------------------------------
