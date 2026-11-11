<!-- kit:generated -- kit welcome's third step, setup, from the kit package -->
<!-- see: "Changing what kit adds" in .kit/1st.md, to replace it -->
# Setup

With kit's own tools in place and a toolchain settled, this step checks the rest, in order:

- Tools: the git hooks' runner, where the hooks package is active, and the linters the rules run.
- Code forges: for each remote's code forge, its command-line tool, installed and logged in.
- This repository: kit installed, `.kit/` in place, the git hooks installed, and `kit check` clean.
- Where you are: at the desk or in a worktree.

A missing tool's fix is `kit tools install`, which installs what the project toolchain pins. With no project
toolchain, the fix is the system toolchain's command. With neither, the check says where to get the tool.

A failed check shows the command that fixes it. Nothing runs until you say so. Run the command yourself, or let kit
run it and check again.

`kit welcome --checks`
