# shellcheck shell=bash
# super.sh - override a function another module defined, and still call the one it replaced (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# use sources every <module>.sh on the module path, in order, so a later file can redefine a function an earlier
# one defined. A plain redefinition replaces the function outright. A module that wants to wrap it instead says so
# before its own definition:
#
#   use super
#   super:override foo:bar
#   foo:bar() { …; super "$@"; …; }
#
# super:override keeps the definition being replaced as foo:bar^. When a third file overrides foo:bar again, the
# chain moves down one (foo:bar^ to foo:bar^^) and the new copy takes foo:bar^, so each definition's super reaches
# the one it replaced. Calls are bound when they run: every caller of foo:bar, in any module, gets the override.
#
# A copy made with eval reports this file as its source. super_origin keeps where each copy was defined
# ("<line> <file>"), so a trace can name the real file.

declare -p super_origin &>/dev/null || declare -Ag super_origin=()

# super:override <function> keeps <function> as <function>^, moving any earlier copies down the chain, so the
# definition that follows can call it with super. It returns 1 when <function> is not defined.
super:override() {
  local fn=$1 top=$1 extdebug=0 where
  if ! declare -F "${fn}" >/dev/null
  then
    error "super:override: ${fn} is not defined; use the module that defines it first"
    return 1
  fi
  while declare -F "${top}^" >/dev/null
  do
    top+='^'
  done
  while [[ "${top}" != "${fn}" ]]
  do
    super:copy "${top}" "${top}^"
    top=${top%^}
  done
  # declare -F names a function's file and line only under extdebug, which is restored at once.
  shopt -q extdebug && extdebug=1
  shopt -s extdebug
  where=$(declare -F "${fn}")
  (( extdebug )) || shopt -u extdebug
  super_origin[${fn}]=${where#* }
  super:copy "${fn}" "${fn}^"
  unset 'super_origin[${fn}]'
}

# super:copy <from> <to> defines <to> with the body of <from>, and carries <from>'s recorded origin.
super:copy() {
  local def
  def=$(declare -f "$1")
  eval "$2${def#"$1"}"
  super_origin[$2]=${super_origin[$1]:-}
}

# super <argument>… calls the definition its caller replaced, with the arguments given.
super() {
  "${FUNCNAME[1]}^" "$@"
}

# ----------------------------------------------------------------------------------------------------------------------
