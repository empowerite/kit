# shellcheck shell=bash
# uv.sh - the uv dependency provider: pyproject.toml names the packages, uv.lock pins them (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# uv is usable where uv is on PATH, and pyproject.toml and uv.lock are at the repository root. It pins every package
# in the lock but the project itself. The lock is TOML, read with yq; what is installed is read with uv pip list,
# from .venv.
#
# A package's name is compared in its normal form, lowercase with dashes: the lock has pyyaml, and pip lists PyYAML.

dependencies/uv:file() { printf 'pyproject.toml\n'; }

dependencies/uv:install() { uv sync --frozen; }

dependencies/uv:lock() { printf 'uv.lock\n'; }

# dependencies/uv:outdated asks the index which installed packages are behind, each by its normal name.
dependencies/uv:outdated() {
  local doc
  if ! doc=$(uv pip list --outdated --format json --python .venv/bin/python 2>/dev/null)
  then
    error "uv pip list --outdated could not ask the index; run kit tools install first"
    return 1
  fi
  jq --raw-output 'def normal: ascii_downcase | gsub("[-_.]+"; "-");
    .[] | [(.name | normal), .version, .latest_version] | @tsv' <<<"${doc}"
}

dependencies/uv:path() { printf '%s/.venv/bin\n' "${PWD}"; }

# dependencies/uv:pins never reaches the network. With no .venv, nothing is installed.
dependencies/uv:pins() {
  local lock installed='[]'
  lock=$(yq --input-format toml --output-format json uv.lock) || return 1
  [[ ! -x .venv/bin/python ]] \
    || installed=$(uv pip list --format json --python .venv/bin/python 2>/dev/null) || installed='[]'
  jq --null-input --raw-output --argjson lock "${lock}" --argjson installed "${installed}" '
    def normal: ascii_downcase | gsub("[-_.]+"; "-");
    ($installed | map({key: (.name | normal), value: .version}) | from_entries) as $have
    | $lock.package[] | select(.source.virtual == null and .source.editable == null)
    | [(.name | normal), .version, ($have[.name | normal] // "")] | @tsv'
}

# dependencies/uv:update moves every package to the latest pyproject.toml allows, then installs it.
dependencies/uv:update() { uv lock --upgrade && uv sync --frozen; }

dependencies/uv:usable() {
  command -v uv >/dev/null 2>&1 && [[ -f pyproject.toml && -f uv.lock ]]
}

# The seam's calls end here; what follows is this provider's own.

# ----------------------------------------------------------------------------------------------------------------------
