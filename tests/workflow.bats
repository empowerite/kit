#!/usr/bin/env bats
# workflow.bats - the workflow package: the workflow, plans as prose, and the worker contract, apart from the entry file
# ----------------------------------------------------------------------------------------------------------------------
# Each test gets an empty git repository under a temporary directory and runs the kit from this checkout against
# it. The kit package delivers the rules every repository takes; the workflow package delivers the taste a repository
# may leave out, and the entry file points to it only as a file that may be there.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fresh-repo
}

@test "install's last line names the file the workflow is in: .kit/workflow.md with the package, else none" {
  run "${kit_checkout}/bin/kit" install workflow
  [ "$status" -eq 0 ]
  [[ "${lines[${#lines[@]} - 1]}" == *"The workflow is in .kit/workflow.md." ]]
  run "${kit_checkout}/bin/kit" uninstall
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  [[ "$output" != *"The workflow is in"* ]]
  [[ "${lines[${#lines[@]} - 1]}" == *"Every agent reads .kit/1st.md first." ]]
}

@test "kit install alone delivers the entry file without the workflow, the worker contract, or a forge's command" {
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  [ -f .kit/1st.md ]
  [ ! -e .kit/workflow.md ]
  grep --quiet '^## Integrity$' .kit/1st.md
  grep --quiet '^## Kit commands instead of git commands$' .kit/1st.md
  run ! grep --quiet 'The worker contract' .kit/1st.md
  run ! grep --quiet 'Plans are prose' .kit/1st.md
  run ! grep --quiet 'gh pr ready' .kit/1st.md
  # the entry file names the file the workflow package delivers, so an agent knows where to look when it is there
  grep --quiet '^## The workflow$' .kit/1st.md
  grep --quiet 'workflow.md' .kit/1st.md
}

@test "kit install workflow delivers .kit/workflow.md: the workflow, plans as prose, and the worker contract, forge-neutral" {
  run "${kit_checkout}/bin/kit" install workflow
  [ "$status" -eq 0 ]
  [ -f .kit/workflow.md ]
  [ "$(head -n 1 .kit/workflow.md)" = '<!-- kit:generated -- the workflow from an issue to a merged pull request, from the workflow package -->' ]
  grep --quiet '^## The worker contract$' .kit/workflow.md
  grep --quiet '^## Plans are prose$' .kit/workflow.md
  grep --quiet 'what was left out, and why' .kit/workflow.md
  grep --quiet 'glab mr update --ready' .kit/workflow.md
  run "${kit_checkout}/bin/kit" packages remove workflow
  [ "$status" -eq 0 ]
  [ ! -e .kit/workflow.md ]
  [ -f .kit/1st.md ]
}

# ----------------------------------------------------------------------------------------------------------------------
