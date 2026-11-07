# shellcheck shell=bash
# packages.sh - the choices kit packages offers, answered by package.sh (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# kit syntax reads a verb's choices from the module named after the verb. kit packages lists and acts through
# package.sh, which keeps its name, so this module hands the question on to package:choices.

use package

# packages:choices <key> [<answer>…] answers kit syntax packages --choices as package:choices does.
packages:choices() {
  package:choices "$@"
}

# ----------------------------------------------------------------------------------------------------------------------
