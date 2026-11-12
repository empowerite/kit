# Using apt, dnf, or pacman

A Linux system's package manager supplies what kit stands on: bash, git, jq, and yq. kit knows three: apt, dnf, and
pacman. For how kit finds and chooses a manager, see [toolchains.md](../toolchains.md).

- Each is usable where its command is on `PATH`: `apt-get`, `dnf`, or `pacman`.
- They fill the system role alone. They install for the whole machine, so they do not pin any repository's tools.
- None is ever active: none is a shell one enters.

## Installing a missing tool

- Each names the command that installs a missing tool, such as `sudo apt-get install shellcheck`. kit never runs it
  on its own, and `kit tools install` only names it.
- `kit welcome` runs it only when you say now, in your terminal, so `sudo` can ask for a password.
- A package's name can differ from the tool's. Fedora names shellcheck `ShellCheck`; Arch names gh `github-cli`.
- A tool with no package there does not get a command. kit names the tool's site instead.
- Debian's and Arch's `yq` are not mikefarah's, which kit needs. So on apt kit names the site, and on pacman it names
  `go-yq`.

## With brew too

With brew on `PATH` too, both are usable for the system. brew is offered first, and the choice is yours:
`kit toolchains use apt --system`.

## With no tool manager

They pin nothing for the project, so a repository with none of the others runs ambient; see
[using no toolchain](using-no-toolchain.md).
