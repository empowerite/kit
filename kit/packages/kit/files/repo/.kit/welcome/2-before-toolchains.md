<!-- kit:generated -- kit welcome's second step, before toolchains, from the kit package -->
<!-- see: "Changing what kit adds" in .kit/1st.md, to replace it -->
# Before toolchains

kit is written in bash. Before it can look for a toolchain, it needs four tools on PATH:

- bash 4.4 or later, which runs kit.
- git, which kit works through.
- jq 1.7 or later, which reads kit's JSON.
- yq 4, mikefarah's, which reads `kit.yml` and the other YAML files.

A missing tool's fix comes from the system toolchain: the tool manager that installs software on this machine.
Without one, the check says where to get the tool.

`kit welcome --checks`
