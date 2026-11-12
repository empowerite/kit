# Using kit as the toolchain

The kit toolchain is kit itself: `tools:` in `kit.yml` pins each tool by version and the URL of its download, and
`tools.lock` records what each download is, and its hash, for every platform, never where it came from. It needs only
curl and tar, and unzip for a zip. For how kit finds and chooses a toolchain, see [toolchains.md](../toolchains.md).

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

### Saying what a tool is, in place of its URL

A URL names a place. A tool may instead say what it is, so any source on the pull path that speaks its protocol
serves it, a mirror as well as the open internet. Each takes the place of `url`, and is filled in as `url` is:

| Keys | What the tool is | Served by |
| --- | --- | --- |
| `release`, `asset`, `tag` | A file of a GitHub release: `release: cli/cli`, `asset: gh_{version}_{os}_{arch}.zip`; `tag` is `v{version}` by default. | `github-releases` sources, and `direct` at github.com |
| `module`, `module_version` | A Go module's zip, unpacked: `module: golang.org/toolchain`, `module_version: v0.0.1-go{version}.{os}-{arch}`; `module_version` is `v{version}` by default. | `goproxy` sources, and `direct` at proxy.golang.org |
| `npm` | An npm package's tarball, at the version: `npm: markdownlint-cli2`. | `npm` sources, and `direct` at registry.npmjs.org |
| `archive`, `tag` | A GitHub tag's source tarball: `archive: bats-core/bats-core`; `tag` is `v{version}` by default. | `github-releases` sources, and `direct` at github.com |
| `dist`, `file` | A file of a project's own download directory: `dist: node`, `file: v{version}/node-v{version}-{os}-{arch}.tar.gz`. The dists are `node` (nodejs.org/dist), `go` (go.dev/dl), and `gnu` (ftp.gnu.org/gnu). | `dist` sources, a mirror of one dist, and `direct` at its home |
| `image`, `tag` | An OCI image's first layer, as oras pushes a file: `image: ghcr.io/acme/tool`; `tag` is the version by default. | `oci` sources, a pull-through cache, and `direct` at the registry the image names |

Go's own toolchain is the module `golang.org/toolchain`, so a Go proxy supplies go:

```yaml
tools:
  go:
    version: "1.27.3"
    module: golang.org/toolchain
    module_version: v0.0.1-go{version}.{os}-{arch}
    os: {macos: darwin}
    arch: {x64: amd64}
    bin: golang.org/toolchain@v0.0.1-go{version}.{os}-{arch}/bin
```

A module or an npm package keeps no file's mode, so kit makes the commands in its `bin` executable, and a module's
tools under `pkg/tool/` too, as go and npm do. An image's layer is unpacked by tar where it is a tarball, and is the
command itself otherwise. A tool with none of `url`, `release`, `module`, `npm`, `archive`, `dist`, and `image`, or
with two, a `release` without its `asset`, or a `dist` without its `file`, is refused by name.

## What kit reads

- **Usable**: curl and tar are on `PATH`, and `kit.yml` has `tools:` or `tools.lock` is here.
- **Active**: each directory `kit tools path` names is on `PATH`.
- **Pins**: each tool's version in `kit.yml`. `tools.lock` holds one line per tool and platform: the name, version,
  platform, profile, artifact, hash, and bin, separated by tabs. The artifact is what the download is: its URL's last
  part, as `file:gh_2.103.2_macOS_arm64.zip`, or what the tool says it is, as
  `release:cli/cli@v2.103.2/gh_2.103.2_macOS_arm64.zip`. The hash is its ecosystem's own, with its prefix:
  `sha256:<hex>` for a file, Go's `h1:` for a module, as `go.sum` holds it, and npm's `sha512-` integrity. A lock
  written before artifacts, with each URL and a bare sha256, reads as the same lock, and the next lock rewrites it.

`kit tools path` and the active check read `tools.lock` alone, so a shell hook that puts the tools on `PATH` stays
fast.

| `kit tools` | What kit does |
| --- | --- |
| `install` | Locks again each tool whose entry changed, downloading it for every platform, then installs each tool this platform and profile take that is not installed at its locked digest. |
| `update` | Moves each tool behind to its latest version in `kit.yml`, then locks and installs it. |
| `--outdated` | For a release, a module, or an npm package, asks the pull path: the newest version the first source that answers serves, a pre-release passed by, so `update` never moves to one the path does not serve yet. For a URL, asks the latest tag of each GitHub release or source archive, through gh where it is installed; Node.js's latest LTS; Go's latest release; and the newest file a GNU mirror's directory lists. It names each tool whose source it cannot ask. |

## Where things are kept

| What | Where |
| --- | --- |
| A tool | `~/.kit/tools/<name>/<version>/<platform>/`, under `KIT_HOME` when it is set, shared by every checkout |
| A download | `~/.cache/kit/downloads/<hex>`, its sha256's, under `XDG_CACHE_HOME` when it is set |

## Where a download comes from

Each download is found on the pull path, the artifact sources kit searches in order, the first that has it winning;
`kit tools sources` prints it, and [packages.md](../packages.md#kinds-and-the-pull-path) says how it is layered. A
`url:` names a place, so `direct`, the open internet, alone serves it: a path that ends at `off` before `direct`
installs none, and kit says so, naming the download and what each source said.

A source whose download has another hash than the one `tools.lock` pins is named and passed by, and the next one is
asked, as is one that cannot be reached; when none has the pinned bytes, nothing is installed. A source's credentials
come from netrc, or from `KIT_SOURCE_TOKEN_<NAME>`, the source's name upper-cased with each dash as `_`, sent as a
bearer token; never from YAML. A tool is unpacked apart and moved into place only once its `bin` is there.

kit fetches the tools. A language's packages it leaves to the language's manager, and `kit tools sources --env` writes
the same path into the manager's settings, a `NAME=value` line each, as a runner's `GITHUB_ENV` takes them: `GOPROXY`,
`GOPRIVATE`, and `GONOSUMDB` for go, from each `goproxy` source and its `private` patterns, and npm's registry,
`npm_config_registry`, or a scope's, `npm_config_@<scope>:registry`. `kit tools run` runs its command with them. A
path of `direct` alone, where no layer names a source, writes nothing, so each manager reads its own settings.

## A desk and a runner

- A desk locks: when an entry in `kit.yml` changes, `kit tools install` downloads the tool for every platform and
  rewrites its lines in `tools.lock`.
- A runner never locks. A `tools.lock` behind `kit.yml` fails the install, naming each tool, so a pin is never
  resolved on a runner that a desk has not checked.
- A tool with `profile: desk` is installed on a desk alone, and one with `profile: ci` on a runner alone.
