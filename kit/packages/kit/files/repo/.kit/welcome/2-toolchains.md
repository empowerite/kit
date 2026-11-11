<!-- kit:generated -- kit welcome's second step, the toolchains, from the kit package -->
<!-- see: "Changing what kit adds" in .kit/1st.md, to replace it -->
# Toolchains

A toolchain is a tool manager. kit looks for the ones it can use here, and never assumes one.

A toolchain fills one of two roles, or both:

- The project toolchain pins this repository's tools, such as the linters. `kit tools install` installs them.
- The system toolchain supplies what kit runs on: bash, git, jq, and yq. kit names its command for a missing tool,
  and installs nothing there on its own.

A toolchain is usable when its tool is on PATH. For the project role, its file must also be in this repository.
For each role, kit uses:

- the one you are already inside, such as a shell it opened
- else, for the project, the one named for the run: `--toolchain <name>` or `KIT_TOOLCHAIN=<name>`
- else the one you chose, while it is still usable
- else the only usable one
- else none: the role is ambient, and kit uses what is on PATH

With several usable and none chosen, choose one: `kit toolchains choose <name>`, with `--system` for the system
role. Your choice stays on this machine. It is never committed, so each person chooses their own.

kit runs on four tools, which this step checks first:

- bash 4.4 or later, which runs kit.
- git, which kit works through.
- jq 1.7 or later, which reads kit's JSON.
- yq 4, mikefarah's, which reads `kit.yml` and the other YAML files.

A missing one's fix is the system toolchain's command. Without one, the check says where to get the tool.

brew counts only when it is on PATH. If brew is installed but off PATH, a check shows the line that fixes it.
Run that line in your shell, and add it to your profile. kit cannot change your shell for you.

`kit toolchains`
