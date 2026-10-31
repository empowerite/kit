# shellcheck shell=bash
# submit.sh - what kit submit offers: the changed paths, as kit commit lists them (sourced module)
# ----------------------------------------------------------------------------------------------------------------------

use commit

# submit:choices <key> [<answer>…] answers kit syntax --choices as commit:choices does: the same open set of changed
# paths, with the staged ones active.
submit:choices() {
  commit:choices "$@"
}

# ----------------------------------------------------------------------------------------------------------------------
