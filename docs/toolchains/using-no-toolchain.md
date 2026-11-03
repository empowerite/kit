# Using no toolchain

With no project toolchain usable, kit runs **ambient**: it uses the tools already on `PATH`. A repository can choose
it on purpose with `kit toolchains choose ambient`. For how kit finds and chooses a toolchain, see
[toolchains.md](../toolchains.md).

Ambient, no toolchain pins anything. `kit tools` lists the tools the rules' linters need, and where each is found on
`PATH`, or `missing`.

| `kit tools` | What it does ambient |
| --- | --- |
| `install` | Does not install anything. With every tool on `PATH`, it says so and exits 0. Otherwise it fails, naming each missing tool with the system toolchain's command that installs it. |
| `update` | Updates what npm or uv pins. With neither usable, it refuses: there is nothing pinned to update. |
| `--outdated` | Finds no tool behind. |
| `path` | Prints only the dependency providers' directories, such as `node_modules/.bin`. |
| `run` | Runs the command with those directories first on `PATH`, or as is when there are none. |

The dependency providers still work ambient; see [dependencies](../toolchains.md#dependencies).

## On a runner

`ambient` is the CI package's default `ci_toolchain`. The workflow then installs nothing. It fails at the `toolchain`
step, naming each tool the rules need that is missing, unless the runner already has them; see
[packages/github/ci.md](../packages/github/ci.md).
