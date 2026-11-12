<!-- kit:generated -- kit welcome's second step, the toolchains, from the kit package -->
<!-- see: "Changing what kit adds" in .kit/1st.md, to replace it -->
# Toolchains — kit checks the four tools it runs on, and the toolchain that installs them on this machine

A [toolchain](kittui:welcome/terms/toolchain) is the set of tools a repository's work runs with, such as Go and its
linters. Two kinds of manager
install tools:

- A [tool manager](kittui:welcome/terms/tool%20manager), such as mise, installs and pins the versions of this
  repository's toolchain. `kit tools install`
  installs it.
- A [package manager](kittui:welcome/terms/package%20manager), such as brew or apt, installs software on the machine.
  kit uses one for the tools kit runs on:
  bash, git, jq, and yq. kit names its command for a missing tool, and runs it only when you say so.

kit looks for the managers it can use here, and never assumes one. Each fills a role: the project, for this
repository's toolchain, or the system, for kit's own tools. brew can fill both.

A manager is usable when its tool is on PATH. For the project, its file must also be in this repository. For each
role, kit uses:

- the one you are already inside, such as a shell it opened
- else, for the project, the one named for the run: `--toolchain=<name>` or `KIT_TOOLCHAIN=<name>`
- else the one you chose, while it is still usable
- else the only usable one
- else none: the role is [ambient](kittui:welcome/terms/ambient), and kit uses what is on PATH

With several usable and none chosen, choose one: `kit toolchains use <name>`, with `--system` for the system
role. Your choice stays on this machine. It is never committed, so each person chooses their own.

This step checks the system's toolchain, once for the machine. Each branch can have files of its own, so setup
checks the project's toolchain in each folder: the [desk](kittui:welcome/terms/desk), and each
[worktree](kittui:welcome/terms/worktree).

kit runs on four tools:

- bash 4.4 or later, which runs kit.
- git, which kit works through.
- jq 1.7 or later, which reads kit's JSON.
- yq 4, mikefarah's, which reads `kit.yml` and the other YAML files.

kit checks bash each time it starts, before it runs anything else. It finds a missing git, jq, or yq only when it
first runs that tool. This step checks all four, bash again among them, before any other check.

A missing one's fix is the system's package manager's command. Without one, the check says where to get the tool.

brew counts only when it is on PATH. If brew is installed but off PATH, a check shows the line that fixes it.
Run that line in your shell, and add it to your profile. kit cannot change your shell for you.

`kit toolchains`
