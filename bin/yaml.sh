# shellcheck shell=bash
# yaml.sh - the subset of YAML the kit's own files are written in (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# Three kinds of file are read here: a module's module.yml, the repository's .kit/settings.yml, and the
# frontmatter between the opening and closing `---` of a rule file. All are written in the same small subset,
# and one reader serves them so they cannot drift:
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
# A value may be double-quoted or single-quoted, and a single-quoted one is taken literally, backslashes and all,
# which is how a pattern is written.
#
# A line starting with # is a comment. Nothing deeper is read, and a file that needs more is a file the kit does
# not write. When the file opens with `---`, only the frontmatter up to the next `---` is read.

# yaml:value <file> <key> prints the scalar under <key>, unquoted, or nothing.
yaml:value() {
  yaml:read "$1" | awk -v key="$2" '
    $0 ~ "^" key ": " {
      v = $0; sub("^" key ": *", "", v)
      if (v ~ /^".*"$/ || v ~ /^\x27.*\x27$/) v = substr(v, 2, length(v) - 2)
      print v; exit
    }'
}

# yaml:list <file> <key> prints the items of the list under <key>, unquoted, one per line.
yaml:list() {
  yaml:read "$1" | awk -v key="$2" '
    $0 == key ":" { on = 1; next }
    /^[A-Za-z0-9_-]/ { on = 0 }
    on && /^  - / { v = $0; sub(/^  - /, "", v); gsub(/^"|"$/, "", v); print v }'
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
# in the order named, separated by the unit separator \x1f, since a pattern may hold a tab. A field a record
# lacks is empty.
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

# yaml:read <file> prints the file's YAML: the frontmatter when the file opens with ---, the whole file otherwise.
yaml:read() {
  [[ -f "$1" ]] || return 0
  awk '
    NR == 1 && $0 == "---" { front = 1; next }
    front && $0 == "---" { exit }
    /^#/ { next }
    { print }' "$1"
}

# ----------------------------------------------------------------------------------------------------------------------
