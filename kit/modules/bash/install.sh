# shellcheck shell=bash
# install.sh - what kit install offers: the packages, as kit package lists them (sourced module)
# ----------------------------------------------------------------------------------------------------------------------

use package

# install:choices <key> [<answer>…] answers kit syntax --choices as package:choices does: the same closed set of
# packages, with the same states and notes.
install:choices() {
  package:choices "$@"
}

# ----------------------------------------------------------------------------------------------------------------------
