<!-- kit:generated -- kit welcome's first step, setup, from the kit package -->
<!-- see: "Changing what kit adds" in .kit/1st.md, to replace it -->
# Setup

kit needs a few tools, a login to each forge, and a repository set up for it. This step checks each one, in order:

- Tools: bash, git, jq, yq, and the linters the rules run. kit finds the tool manager the repository pins its
  tools with, or uses what is on PATH.
- Forges: for each remote's forge, its command-line tool, installed and logged in.
- This repository: kit installed, `.kit/` in place, the git hooks installed, and `kit check` clean.
- Where you are: at the desk or in a worktree.

A failed check shows the command that fixes it. Nothing runs until you say so. Run the command yourself, or let kit
run it and check again. A check with no known fix says where to read.

`kit welcome --checks`
