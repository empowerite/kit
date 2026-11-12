<!-- kit:generated -- kit welcome's first step, the introduction, from the kit package -->
<!-- see: "Changing what kit adds" in .kit/1st.md, to replace it -->
# Introduction — what kit is, where it works, and who it is for

kit is one set of commands for the whole development cycle, the same for people and for coding agents: start work
from an issue, commit, open and update a pull request, run CI, and make a release.

## One set of commands across your tools

- Agent harnesses: Claude Code, Cursor, GitHub Copilot, and Gemini CLI each read the same rules, in `.kit/`. The
  rules are not copied into each harness's own files, so there is nothing to regenerate or keep in sync.
- Code forges: GitHub, GitLab, and Azure DevOps. The commands are the same on each.
- Trackers: the forge's own issues, Azure DevOps work items, and others such as Jira and Linear. One repository can
  use more than one.
- CI: `kit ci` runs the same checks on your machine and in any CI system.
- Repositories: kit shows and acts on a group of repositories as one workspace.

## Any repository, or installed in one

kit works in any repository as it is. Install it in a repository to do more there: kit adds the rules that every
person and agent follows, checks the work against them on each pull request, and updates them with each kit release.

## Who it is for

- People who work with coding agents and want an agent's work held to the same standard as their own.
- Teams that want the same rules and the same commands in every repository.
- Agents: kit tells an agent where the rules are, which command does each step, and why a check failed.
