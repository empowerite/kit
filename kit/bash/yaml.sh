# shellcheck shell=bash
# yaml.sh - the subset of YAML the kit's own files are written in (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# The kit reads three kinds of file with this module:
#
#   - a package's package.yml
#   - the repository's .kit/settings.yml
#   - a rule file's frontmatter, between the opening and closing `---`
#
# All three use the same small subset of YAML. One reader handles all of them, so their formats cannot drift
# apart:
#
#   key: value          a scalar, quoted or not
#   key:                a list, one `  - item` per line
#     - item
#   key:                a map, one `  name: value` per line
#     name: value
#   key:                a list of records, each opened by `  - field: value` and continued by `    field: value`
#     - field: value
#       field: value
#
# A value may be double-quoted or single-quoted. A single-quoted value is read literally, backslashes included,
# which is how a pattern is written.
#
# A line starting with # is a comment. Nothing nested more deeply is read; the kit writes no file that needs more.
# When the file starts with `---`, only the frontmatter up to the next `---` is read.

# yaml:value <file> <key> prints the scalar under <key>, unquoted, or nothing.
yaml:value() {
  yaml:read "$1" | awk -v key="$2" '
    $0 ~ "^" key ": " {
      v = $0; sub("^" key ": *", "", v)
      if (v ~ /^".*"$/ || v ~ /^\x27.*\x27$/) v = substr(v, 2, length(v) - 2)
      print v; exit
    }'
}

# yaml:list <file> <key> prints the items of the list under <key>, unquoted, one per line. A # after an unquoted
# item, with a space before it, starts a comment, as in YAML.
yaml:list() {
  yaml:read "$1" | awk -v key="$2" '
    $0 == key ":" { on = 1; next }
    /^[A-Za-z0-9_-]/ { on = 0 }
    on && /^  - / {
      v = $0; sub(/^  - /, "", v)
      if (v ~ /^"/) gsub(/^"|"$/, "", v); else sub(/[ \t]+#.*$/, "", v)
      print v
    }'
}

# yaml:map <file> <key> prints "<name>\t<value>" for every entry of the map under <key>, values unquoted.
yaml:map() {
  yaml:read "$1" | awk -v key="$2" '
    $0 == key ":" { on = 1; next }
    /^[A-Za-z0-9_-]/ { on = 0 }
    on && /^  [A-Za-z0-9_-]+: / {
      n = $0; sub(/^  /, "", n); sub(/: .*$/, "", n); v = $0; sub(/^  [A-Za-z0-9_-]+: */, "", v);
      if (v ~ /^".*"$/ || v ~ /^\x27.*\x27$/) v = substr(v, 2, length(v) - 2); print n "\t" v }'
}

# yaml:records <file> <key> <field>… prints one line per record of the list under <key>: the named fields' values,
# in the order named, separated by the unit separator \x1f (not a tab, because a pattern may contain a tab). A
# field the record does not have is empty.
yaml:records() {
  local file=$1 key=$2
  shift 2
  yaml:read "${file}" | awk -v key="${key}" -v fields="$*" '
    function unquote(v) {
      if (v ~ /^".*"$/ || v ~ /^\x27.*\x27$/) v = substr(v, 2, length(v) - 2)
      return v
    }
    function flush(   i, line) {
      if (!open) return
      line = ""
      for (i = 1; i <= n; i++) line = line (i > 1 ? "\x1f" : "") value[names[i]]
      print line
      delete value
      open = 0
    }
    BEGIN { n = split(fields, names, " ") }
    $0 == key ":" { on = 1; next }
    /^[A-Za-z0-9_-]/ { flush(); on = 0 }
    on && /^  - [A-Za-z0-9_-]+: / { flush(); open = 1; sub(/^  - /, "") }
    on && open && /^    [A-Za-z0-9_-]+: / { sub(/^    /, "") }
    on && open && /^[A-Za-z0-9_-]+: / {
      f = $0; sub(/: .*$/, "", f); v = $0; sub(/^[A-Za-z0-9_-]+: */, "", v); value[f] = unquote(v)
    }
    END { flush() }'
}

# yaml:keys <file> prints the top-level keys, one per line.
yaml:keys() {
  yaml:read "$1" | awk '/^[A-Za-z0-9_-]+:/ { k = $0; sub(/:.*$/, "", k); print k }'
}

# yaml:read <file> prints the file's YAML, without comment lines: the frontmatter when the file starts with ---,
# otherwise the whole file.
yaml:read() {
  [[ -f "$1" ]] || return 0
  awk '
    NR == 1 && $0 == "---" { front = 1; next }
    front && $0 == "---" { exit }
    /^#/ { next }
    { print }' "$1"
}

# ----------------------------------------------------------------------------------------------------------------------
