<!-- kit:generated -- kit welcome's third step, setup, from the kit package -->
<!-- see: "Changing what kit adds" in .kit/1st.md, to replace it -->
# Setup — kit checks this machine once, then each folder of the workspace on its own

With kit's own tools in place and the system toolchain settled, this step checks the rest. Each folder is checked on
its own, since each [branch](kittui:welcome/terms/branch) can have a `kit.yml` or none, and pin its own kit:

- This machine: for each [code forge](kittui:welcome/terms/code%20forge) among the folders' remotes, its command-line
  tool, installed and logged in, once
  per host.
- Each folder, by [repository](kittui:welcome/terms/repository), its [desk](kittui:welcome/terms/desk) first, then
  each [worktree](kittui:welcome/terms/worktree): kit installed, with the version its `kit.yml`
  declares; `.kit/` in place; the git hooks installed; its project toolchain; the hooks' runner and the linters the
  rules run; and `kit check` clean, run by the kit that folder pins.

The [workspace](kittui:welcome/terms/workspace) is the one `kit workspaces --current` names, or the one `kit
--workspace=<workspace> welcome` names: a
repository, its desk and its worktrees; a group, every member's. With no workspace, only the machine is checked.

A missing tool's fix in a folder is `kit tools install`, which installs what its project toolchain pins. With no
project toolchain, the fix is the system toolchain's command. With neither, the check says where to get the tool.

A failed check shows the command that fixes it. Nothing runs until you say so. Run the command yourself, or let kit
run it and check again. A folder's fix runs in that folder.

`kit welcome --checks`
