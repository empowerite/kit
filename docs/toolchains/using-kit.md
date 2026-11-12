# Using kit as the toolchain

The kit toolchain is kit itself: `tools:` in `kit.yml` pins each tool by version and the URL of its download, and
`tools.lock` records each download's sha256 for every platform. It needs only curl and tar, and unzip for a zip. For
how kit finds and chooses a toolchain, see [toolchains.md](../toolchains.md).

## Set up

1. Pin each tool under `tools:` in `kit.yml`, with its version quoted and the URL of its download:

   ```yaml
   tools:
     actionlint:
       version: "1.7.12"
       url: https://github.com/rhysd/actionlint/releases/download/v{version}/actionlint_{version}_{os}_{arch}.tar.gz
       os: {macos: darwin}
       arch: {x64: amd64}
     lefthook:
       version: "2.2.0"
       url: https://github.com/evilmartians/lefthook/releases/download/v{version}/lefthook_{version}_{os}_{arch}
       os: {linux: Linux, macos: MacOS}
       arch: {x64: x86_64}
       profile: desk
   ```

2. Run `kit tools install --toolchain=kit`. It downloads each tool for every platform, writes `tools.lock`, and
   installs this platform's.
3. Commit `kit.yml` and `tools.lock`.

## A tool's entry

| Key | What it holds |
| --- | --- |
| `version` | The exact version, quoted, so YAML keeps `"1.20"` as written rather than as `1.2`. |
| `url` | The download, with `{version}`, `{platform}`, `{os}`, `{arch}`, and `{ext}` filled in. |
| `os`, `arch` | The tool's names for kit's own: `linux` and `macos`, `x64` and `arm64`. A platform is `<os>-<arch>`. |
| `ext` | Each os's file ending for `{ext}`, as `{linux: tar.gz, macos: zip}`, for a tool that ships a zip to one and a tarball to the other. |
| `bin` | The directory, in what the URL unpacks to, that holds the tool's commands, one of them named as the tool, filled in as the URL is. `.` by default. |
| `profile` | `desk` or `ci` installs the tool on one of them alone. A run where `CI` is set is a runner's. |
| `platforms` | The platforms it is locked for, when not all four: `linux-x64`, `linux-arm64`, `macos-x64`, `macos-arm64`. |

A URL that ends in `.tar.gz`, `.tgz`, `.tar.xz`, `.txz`, `.tar.bz2`, or `.tar` is unpacked by tar, and one that ends
in `.zip` by unzip. Any other URL is the command itself, saved under the tool's name.

## What kit reads

- **Usable**: curl and tar are on `PATH`, and `kit.yml` has `tools:` or `tools.lock` is here.
- **Active**: each directory `kit tools path` names is on `PATH`.
- **Pins**: each tool's version in `kit.yml`. `tools.lock` holds one line per tool and platform: the name, version,
  platform, profile, URL, sha256, and bin, separated by tabs.

`kit tools path` and the active check read `tools.lock` alone, so a shell hook that puts the tools on `PATH` stays
fast.

| `kit tools` | What kit does |
| --- | --- |
| `install` | Locks again each tool whose entry changed, downloading it for every platform, then installs each tool this platform and profile take that is not installed at its locked digest. |
| `update` | Moves each tool behind to its latest version in `kit.yml`, then locks and installs it. |
| `--outdated` | Asks the latest tag of each GitHub release or source archive, through gh where it is installed; Node.js's latest LTS; Go's latest release; and the newest file a GNU mirror's directory lists. It names each tool whose source it cannot ask. |

## Where things are kept

| What | Where |
| --- | --- |
| A tool | `~/.kit/tools/<name>/<version>/<platform>/`, under `KIT_HOME` when it is set, shared by every checkout |
| A download | `~/.cache/kit/downloads/<sha256>`, under `XDG_CACHE_HOME` when it is set |

A download whose bytes have another digest than the one `tools.lock` pins is refused, and nothing is installed from
it. A tool is unpacked apart and moved into place only once its `bin` is there.

## A desk and a runner

- A desk locks: when an entry in `kit.yml` changes, `kit tools install` downloads the tool for every platform and
  rewrites its lines in `tools.lock`.
- A runner never locks. A `tools.lock` behind `kit.yml` fails the install, naming each tool, so a pin is never
  resolved on a runner that a desk has not checked.
- A tool with `profile: desk` is installed on a desk alone, and one with `profile: ci` on a runner alone.
