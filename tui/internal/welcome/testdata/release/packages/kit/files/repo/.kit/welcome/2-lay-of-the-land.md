# The lay of the land

<!-- the stages of kit status, in order -->

Work moves through the stages that `kit status` draws. Each part ends with the command that shows it live.

## The backlog

An issue waits in the backlog until someone takes it. `kit status backlog` lists them.

## Underway

`kit issue <N>` makes a worktree for issue N, on a branch of its own, and the issue is underway. You commit there
with `kit commit`, which runs the checks a commit's rule requires.

- `kit submit` pushes the branch and opens the pull request.
- The build runs the gates; `kit status builds` shows it.
- A release tags what the build passed.

## The desk

The desk is the root checkout. People read and review there; agents never work there, so two never edit one tree.

```sh
kit status desk
```
