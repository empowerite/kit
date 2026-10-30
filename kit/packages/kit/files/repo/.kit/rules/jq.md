---
# Added by the kit package, which kit always installs.
# To change it, do not edit it here: see "Changing what kit adds" in .kit/1st.md.
kind:
  - jq
checks:
  - jq-compiles
---
# jq

Applies to `*.jq`, a jq module: a file of `def`s that a program imports with `jq -L <dir>`.

1. Every function compiles, and every name it calls is defined. `kit lint` checks both, because importing a module
   does not check a function nothing calls.
2. A function takes what varies between runs as an argument, such as the current time, so a test can fix it.
3. A module has tests, in the format of `jq --run-tests`: a program, an input, and the outputs it gives.
