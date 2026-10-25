# shellcheck shell=bash
# yaml.sh - the subset of YAML the kit's own files are written in (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# Two kinds of file are read here: a module's module.yml, and the frontmatter between the opening and closing `---`
# of a rule file. Both are written in the same small subset, and one reader serves both so they cannot drift:
#
#   key: value          a scalar, quoted or not
#   key:                a list, one `  - item` per line
#     - item
#   key:                a map, one `  name: value` per line
#     name: value
#
# A line starting with # is a comment. Nothing deeper is read, and a file that needs more is a file the kit does
# not write. When the file opens with `---`, only the frontmatter up to the next `---` is read.

# yaml:value <file> <key> prints the scalar under <key>, unquoted, or nothing.
yaml:value() {
  yaml:read "$1" | awk -v key="$2" '
    $0 ~ "^" key ": " { v = $0; sub("^" key ": *", "", v); gsub(/^"|"$/, "", v); print v; exit }'
}

# yaml:list <file> <key> prints the items of the list under <key>, unquoted, one per line.
yaml:list() {
  yaml:read "$1" | awk -v key="$2" '
    $0 == key ":" { on = 1; next }
    /^[A-Za-z]/ { on = 0 }
    on && /^  - / { v = $0; sub(/^  - /, "", v); gsub(/^"|"$/, "", v); print v }'
}

# yaml:map <file> <key> prints "<name>\t<value>" for every entry of the map under <key>, values unquoted.
yaml:map() {
  yaml:read "$1" | awk -v key="$2" '
    $0 == key ":" { on = 1; next }
    /^[A-Za-z]/ { on = 0 }
    on && /^  [A-Za-z_]+: / { n = $0; sub(/^  /, "", n); sub(/: .*$/, "", n); v = $0; sub(/^  [A-Za-z_]+: */, "", v);
      gsub(/^"|"$/, "", v); print n "\t" v }'
}

# yaml:keys <file> prints the top-level keys, one per line.
yaml:keys() {
  yaml:read "$1" | awk '/^[A-Za-z_]+:/ { k = $0; sub(/:.*$/, "", k); print k }'
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
