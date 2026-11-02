<!-- kit:generated -- kit welcome's third step, the toolchains, from the kit package -->
<!-- see: "Changing what kit adds" in .kit/1st.md, to replace it -->
# Toolchains

A toolchain is a tool manager. kit looks for the ones it can use here, and never assumes one.

A toolchain fills one of two roles, or both:

- The project toolchain pins this repository's tools, such as the linters. `kit tools install` installs them.
- The system toolchain supplies what kit runs on: bash, git, jq, and yq. kit names its command for a missing tool,
  and installs nothing there on its own.

A toolchain is usable when its tool is on PATH and its file is in this repository. For each role, kit uses:

- the one you are already inside, such as a shell it opened
- else the one you chose, while it is still usable
- else the only usable one
- else none: the role is ambient, and kit uses what is on PATH

With several usable and none chosen, choose one: `kit toolchains choose <name>`, with `--system` for the system
role. Your choice stays on this machine. It is never committed, so each person chooses their own.

`kit toolchains`
