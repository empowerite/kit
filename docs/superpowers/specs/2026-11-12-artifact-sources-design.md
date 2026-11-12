# Artifact sources: design

Status: designed, 2026-11-12. see: issue #182, which builds on #181

## Intent

A repository pins what it builds with: kit's packages, its tools, and its language's packages. Today each comes from
one place, written into `kit.yml`: a package source's tarball URL, and a tool's download URL. A desk behind a mirror,
a desk on the open internet, and a runner on a private network all reach for the same URL, and one of them cannot.

An artifact source is a place that serves artifacts by a protocol: kit's package tarballs, a Go module proxy, an npm
registry, GitHub's release downloads, an OCI registry, or Artifactory, which speaks several. kit searches a list of
them, the pull path, in order, and installs what the first one serves, after checking it is the artifact the lock
records. Releases go the other way, to push targets, which are never searched, and only from a runner.

## Terms

- **Artifact**: a file kit fetches or a release pushes: a package tarball, a tool's download, a Go module's zip, an
  npm package's tarball, an OCI blob.
- **Coordinate**: what an artifact is, apart from where it is: `release:cli/cli@v2.103.2/gh_2.103.2_macOS_arm64.zip`,
  `module:golang.org/toolchain@v0.0.1-go1.27.3.darwin-arm64`, `npm:markdownlint-cli2@0.18.1`. A tool pin is a name
  and a version, which kit turns into a coordinate for each platform.
- **Kind**: the protocol a source speaks: `kit-packages`, `goproxy`, `npm`, `github-releases`, `oci`, or
  `artifactory`. Each kind is one provider of the `sources` seam.
- **Pull path**: the sources kit searches for an artifact, in order, the first hit winning.
- **direct** and **off**: the two words a pull path may hold besides a source. `direct` is the open internet: each
  kind at its public home. `off` ends the path: past it, nothing is searched.
- **Push target**: where a release of the repository's own artifacts goes. A target has a kind of its own, and a
  connector, the command line tool its service ships.
- **Visibility**: whether a pull source serves a pushed version yet, and with which hash.

## Goals

- One `sources:` list in `kit.yml`, each entry with a `kind:`, kit-packages as today among them.
- One adapter for each protocol, not a template for each tool: a tool says what it is, never where it is.
- A pull path in layers, as `PATH` is: `KIT_SOURCES` over the person's workspace over `kit.yml`, where a layer may
  add sources or replace the ones below.
- A lock that records what each artifact is, in its ecosystem's own hash, and never where it came from, so every
  machine installs the same bytes from its own path.
- The same path written into each language manager's settings, so `go` and `npm` search where kit does.
- Push targets per artifact, a release and a pre-release each with their own, pushed only from a runner, only through
  a connector.
- Pull behind push shown where a view already reaches the network: kit flow's releases stage, `kit tools
  --outdated`, and `kit release --wait=<duration>`.

## Non-goals

- kit does not install a language's packages. npm, go, and uv do; kit gives them the path.
- kit does not speak a service's upload API. A push target runs its connector.
- kit does not build a tool from its source. A Go module on the path is installed as its zip unpacks, which serves
  `golang.org/toolchain`; a tool that ships as source alone stays a `url:` or a release file.
- No credentials in YAML. A source's credentials come from netrc, the keychain, `gh auth`, or the environment.

## Anti-goals

- A lock that names a URL, so that a mirror's bytes and the internet's could differ under one lock line unnoticed.
- A version overwritten on a target. Update is a new version; delete is retire.
- A publish from a desk. A pre-release is the one exception, at its user's own risk, as `bin/release` allows today.
- A workspace that moves where a project's releases go. It may give a target its credentials, never move it.

## Settings

### `sources:` in `kit.yml`

```yaml
sources:
  - name: team                    # kit's packages, as today: kind kit-packages is the default
    url: git:empowerite/kit-team@v1.2.0
    digest: sha256:3b1f…
  - name: corp-go
    kind: goproxy
    url: https://artifactory.corp/api/go/go-remote
    private: [github.com/corp/*]  # GOPRIVATE and GONOSUMDB: no checksum database for these
    expect: 1d                    # how long a push usually takes to show here
  - name: corp-npm
    kind: npm
    url: https://artifactory.corp/api/npm/npm-remote
    scope: "@corp"                # this registry serves the scope alone, as .npmrc's @corp:registry
  - name: corp-files
    kind: artifactory
    url: https://artifactory.corp
    repository: github-remote     # a remote of github.com, laid out as GitHub's release downloads are
    type: github-releases         # which protocol the repository speaks: github-releases, goproxy, or npm
  - direct
```

- An entry is a map with `name`, `kind`, and `url`, or one of the words `direct` and `off`.
- `kind` is the name of a provider of the `sources` seam on the module path: `kit-packages` when it is left out, so a
  `kit.yml` written before kinds reads as it did.
- A `kit-packages` entry also has `digest`, as today. Every other kind takes the keys its provider reads, named in
  the provider's header: `private` for goproxy, `scope` for npm, `repository` and `type` for artifactory.
- `expect: <duration>`, as `30m`, `12h`, or `1d`, is how long a version usually takes to show on that source after a
  push. Past it, a pending version is stuck.
- A name is lowercase letters, digits, and dashes, starting with a letter, and once in the file.

`settings:validate` refuses an entry with an unknown kind, a missing name or url, a name used twice, or a key its
kind does not read. `.kit/kit.schema.json` says the same for an editor.

### Package sources are the repository's

A `kit-packages` source is never on the pull path. Which packages a repository has is the repository's choice, pinned
by digest, so the package sources come from `kit.yml` alone, the last first, as today. A workspace or `KIT_SOURCES`
that names one is refused, naming the entry.

### The layers

The pull path is built from three layers, the first first:

1. `KIT_SOURCES`, for a runner: entries apart by commas, each `direct`, `off`, or `<name>=<kind>:<url>`, as
   `KIT_SOURCES=mirror=goproxy:https://proxy.corp/go,off`. A runner has no workspace, so this is how it says where
   it may reach.
2. The workspace: `sources:` at the top of the person's `workspaces.yml`, the same entries as `kit.yml`'s. A
   workspace is one person's, and applies to every repository they work in.
3. `kit.yml`'s artifact sources, the repository's default. With none, the default is `direct`.

A layer's entries go before the ones below it. A layer that holds `off` replaces the layers below: `off` ends the
path, so nothing after it is searched. A name or `direct` met a second time is the first one's. So a desk behind a
mirror writes its mirror and `off` in its workspace, a desk on the open internet writes nothing, and a runner writes
`KIT_SOURCES`; each installs the same bytes, which the lock checks.

`kit tools sources` prints the path: each entry, the layer it came from, its kind, and its url, with `--json`.

### `tools:`, as a coordinate

A tool pin says what the tool is. Its `version` and `os`, `arch`, `ext`, `bin`, `profile`, and `platforms` read as
they do in #181; its download is named by one of these, in place of `url`:

| Keys | Coordinate | Served by |
| --- | --- | --- |
| `release: <owner>/<repo>`, `asset: <file>`, `tag:` (`v{version}` by default) | `release:<owner>/<repo>@<tag>/<asset>` | github-releases, artifactory of type github-releases, direct |
| `module: <path>`, `module_version:` (`v{version}` by default) | `module:<path>@<module_version>` | goproxy, artifactory of type goproxy, direct |
| `npm: <package>` | `npm:<package>@<version>` | npm, artifactory of type npm, direct |
| `image: <repository>`, `tag:` (`{version}` by default) | `oci:<repository>:<tag>` | oci, direct |
| `url: <url>` | `url:<url>` | direct alone |

Each is filled in as #181 fills `url`: `{version}`, `{platform}`, `{os}`, `{arch}`, and `{ext}`. `url:` stays for a
download no protocol serves, as a GNU mirror's tarball; only `direct` serves it, so a desk with `off` before `direct`
cannot install it, and kit says so, naming the tool.

Go's own toolchain is a module on the proxy, so a Go proxy on the path supplies go:

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

## The `sources` seam

`kit/modules/bash/sources.sh` holds the seam; each kind is `kit/modules/bash/sources/<kind>.sh`, whose functions are
named `sources/<kind>:<call>`. `kit new provider sources <kind>` makes one from
`.kit/templates/bash/sources-provider`, and the bash rule holds `**/sources/*.sh` to that template's shape. The
seam's header lists its calls, and `tests/template.bats` checks the header and the template agree.

Every provider implements these calls:

| Call | What it does |
| --- | --- |
| `serves <coordinate>` | succeeds when the kind can serve an artifact of that coordinate's form, `release:` or `module:` |
| `home` | the address `direct` stands for, for this kind; nothing for a kind with no public home |
| `keys` | the keys an entry of this kind takes beyond `name`, `kind`, `url`, and `expect`, one per line, so a key it does not take is refused |
| `fetch <url> <coordinate> <file>` | writes the artifact's bytes at `<file>`; 4 when the source does not serve it, 1 when it cannot be reached |
| `hash <file> <coordinate>` | the ecosystem's own hash of the bytes, as the lock records it |
| `versions <url> <coordinate>` | the versions the source serves of the coordinate's artifact, one per line; 4 when it serves none, 1 when it cannot be reached |
| `settings <url> <entry>` | `NAME=value` lines a language's manager reads to search the source, from the entry's JSON; nothing for a kind no manager reads |

A coordinate's version is ignored by `versions`, which lists them all.

The seam's own functions pick the provider, as the toolchain seam does, by building `sources/<kind>:<call>` from the
entry's kind:

- `sources:path` prints the pull path, a line per entry, apart by tabs: name, kind, url, layer, and the entry's JSON.
- `sources:find <coordinate> <hash> <file>` walks the path: each entry whose kind serves the coordinate fetches it,
  and the first hit whose hash is the lock's wins. A miss goes on to the next entry. A source that cannot be reached
  is named, and the walk goes on. A hit with another hash is refused, naming the source, and the walk goes on, so a
  mirror serving other bytes never installs them. With no hash yet, as when the lock is written, the first hit wins
  and its hash is the one locked. At the end, it fails, naming each source it asked and what each said, and `off` if
  the path ended there.
- `sources:versions <coordinate>` prints the versions the first source serving the coordinate's form lists.
- `sources:settings` prints what each entry's `settings` call prints, the path's order kept: the goproxy entries
  joined into `GOPROXY`, with `direct` and `off` as Go writes them, and `GOPRIVATE` and `GONOSUMDB` from their
  `private`; npm's registry as `npm_config_registry`, and a scoped registry as `npm_config_@<scope>:registry`.
  `kit tools sources --env` prints them, and `kit tools run` runs with them. A path of `direct` alone, because no
  layer names a source, writes nothing, so each manager reads its own settings.

### The providers

| Kind | `serves` | `home` | `fetch` | `hash` | `versions` | `settings` |
| --- | --- | --- | --- | --- | --- | --- |
| kit-packages | `package:` | none | the source's tarball, `git:` as the forge's | `sha256:` | none | none |
| github-releases | `release:` | `https://github.com` | `<url>/<owner>/<repo>/releases/download/<tag>/<asset>` | `sha256:` | the releases' tags, through `gh api` for github.com where gh is signed in, else the API | none |
| goproxy | `module:` | `https://proxy.golang.org` | `<url>/<module>/@v/<version>.zip`, the path case-encoded | `h1:`, Go's dirhash of the zip | `<url>/<module>/@v/list` | `GOPROXY`, `GOPRIVATE`, `GONOSUMDB` |
| npm | `npm:` | `https://registry.npmjs.org` | the tarball the package document's `dist.tarball` names | `sha512-`, npm's integrity | the package document's `versions` | `npm_config_registry`, or the scope's |
| oci | `oci:` | the repository's own registry | the manifest's first layer, through the registry's v2 API | `sha256:` | the repository's tags | none |
| artifactory | as its `type` | none | as its `type`'s provider, at `<url>/api/<type's API>/<repository>` | as its `type` | as its `type`, and Xray's block as blocked | as its `type` |

A provider reaches the network with curl, and its credentials come from netrc (`curl --netrc-optional`), the
keychain or `gh auth token` for GitHub, or the environment: `KIT_SOURCE_TOKEN_<NAME>`, the name upper-cased with
dashes as underscores, sent as a bearer token. A clone's `kit.yml` that names a source is what `kit trust` asks about,
each source by name, kind, and url, before kit reads anything there.

## The lock

`tools.lock` loses its URL. A line is, apart by tabs (shown here as spaces): name, version, platform, profile,
artifact, hash, and bin.

- The artifact is what the download is, apart from where it is: the coordinate itself for every form but `url:`,
  which names a place, and for `url:` its last part, as `file:gh_2.103.2_macOS_arm64.zip`. A pin whose artifact
  changes is locked again; a pin that moves to a mirror of the same file is not, and the hash holds the mirror to the
  same bytes.
- The hash is the ecosystem's own, with its prefix: `sha256:<hex>` for a release file, an OCI blob, or a `url:`
  download; `h1:<base64>` for a Go module, as `go.sum` writes it; `sha512-<base64>` for an npm package, as its
  `integrity`.

```text
# tools.lock - what each tool's download is, for every platform, written by kit tools install from tools: in kit.yml
# name  version  platform  profile  artifact  hash  bin
gh  2.103.2  macos-arm64  both  release:cli/cli@v2.103.2/gh_2.103.2_macOS_arm64.zip  sha256:5416000f…  gh_2.103.2_macOS_arm64/bin
go  1.27.3  macos-arm64  both  module:golang.org/toolchain@v0.0.1-go1.27.3.darwin-arm64  h1:Jq3w…  golang.org/toolchain@v0.0.1-go1.27.3.darwin-arm64/bin
```

- A download is kept in the cache by its hash: a sha256 by its hex, as before, so a cache from #181 is still used,
  and any other hash with each `/` as `_`.
- A Go module has no checksum database when it is private, so its `h1:` lives in the lock, as `go.sum` holds it.
  kit checks every module against the lock alone; it never asks `sum.golang.org`.
- A lock line whose tool entry moved is locked again from the path, as #181 locks it from its URL. A runner still
  refuses a lock behind `kit.yml`.
- A lock written before this change has the URL and a bare sha256 in its fifth and sixth columns. kit reads it as
  the URL's last part, `file:<part>`, and `sha256:<hex>`, so it is not behind, and writes the new form the next time
  it locks.

## Push targets

```yaml
publish:
  kit:                                   # the artifact, by name
    files: ["dist/kit-{version}.tar.gz"] # what the runner built, {version} filled in, quoted for YAML's braces
    release: empowerite/kit              # its coordinate on the pull path, as a tool pin's
    asset: kit-{version}.tar.gz
    targets:
      release: [vcs-tag, github-release]
      prerelease: [github-release]
    verify: [corp-files]                 # the pull sources that check visibility; the whole path by default
```

- A target is a provider of the `targets` seam, `kit/modules/bash/targets/<kind>.sh`, made from
  `.kit/templates/bash/targets-provider`: `vcs-tag` (git push of the release tag), `github-release` (gh),
  `artifactory-local` (jfrog-cli), and `npm-publish` (npm). Its kind is its own, never the push side of a pull kind:
  a Go module is published by its tag, and the proxy fetches it.
- Every target implements `connector`, the tool it runs; `keys`, the keys it takes beyond `kind`; `pushed <version>
  <target>`, which succeeds when the target holds the version; `push <version> <target> <file>…`; and `retire
  <version> <target>`. The target is its JSON, the artifact's own under `artifact`. A target is written as its kind,
  or as a map of its kind and its keys.
- `kit publish <version>` pushes each artifact to each target its release or pre-release names. It refuses where
  `GITHUB_ACTIONS`, or the runner's own marker (`GITLAB_CI`, `TF_BUILD`), is not set, unless the version is a
  pre-release. It refuses a version a target already holds, before it pushes to any. It runs each connector with the
  runner's credentials; a workspace may name a target's credentials, never its kind or place.
- The push is done when the target takes it. `kit publish` then tags what it pushed as `release-hashes/<version>`, a
  tag of a blob of one line per artifact, `<artifact>\t<hash>`, the hash its pull kind takes, so anyone who can read
  the forge knows what was pushed. The release tag is made at a desk before the runner builds anything, so its
  annotation cannot hold the hashes, and a tag is never rewritten; a tag of its own, one per version, is never
  overwritten either. A Go module's zip is the proxy's own, so a module records no hash and is visible once served.

## Visibility

Visibility is a state kit watches, not a step a push waits for. For each release tag on the forge and each pull
source that serves the artifact's coordinate form (or the `verify` sources alone), it is one of:

- **visible**: served, with the hash `release-hashes/<version>` records, or with any where none is recorded;
- **pending**: tagged, not served yet, with its age since the tag; past the source's `expect`, it is **stuck**;
- **blocked**: the source says it never will serve it, as Artifactory's Xray block or a refused approval: its
  provider's `fetch` returns 5;
- **mismatch**: served, with another hash.

- kit flow's releases stage shows each release's states in the publishing repository, and says when one is stuck.
- In a consumer, `kit tools --outdated` says, for each tool behind, the newest version the path serves, and, where the
  path holds a mirror, the newest tagged upstream, read from `direct`, where it is newer: `tagged`.
- `kit update` and `kit tools update` move to the newest version the path serves, never only the newest tag. A pin
  to a version the path does not serve yet fails, naming it pending.
- It is read only where a view already reaches the network, and cached under `XDG_CACHE_HOME/kit/visibility/`: a
  visible or blocked answer is kept, and a pending one is asked again after five minutes.
- `kit release <version> --wait=<duration>` pushes the tag, or with the tag on the primary already only waits, until
  every checked source is visible, or one is blocked or a mismatch, or the duration passes, and exits 0, 1, or 3. It
  asks again every 30 seconds, `KIT_RELEASE_POLL` to change it, and never from what is kept.
- kit flow's releases stage reads the five newest releases that are no drafts, and a row carries `visibility`, a
  state per artifact and source, which no column shows.

The open question in the issue, whether a target names the pull source that checks it, is settled: the whole path
checks it by default, and `verify:` narrows it to the sources it names.

## Implementation slices

Each slice is its own #182 commit, with bats tests against fakes in `tests/fakes`, never the network, and its docs.

1. **Source kinds and the pull path.** The `sources` seam and its template; `sources/kit-packages.sh`, which the
   package sources move onto, their behaviour unchanged; `kind:`, `direct`, and `off` in `sources:`, validated, and
   in the schema; the layers, `KIT_SOURCES` over the workspace over `kit.yml`; `kit tools sources`; `kit trust`
   naming each source's kind and url.
2. **Tool pins through the path, a hash-only lock.** `tools.lock` with the artifact in place of its URL, the hash with
   its prefix; a lock of URLs and bare sha256s read as the new form; each download found through `sources:find`, a
   `url:` pin served by `direct` alone, a source with other bytes passed by; the cache keyed by hash.
3. **github-releases, goproxy, and npm.** The three providers, with the `release:`, `module:`, and `npm:` pins, the
   `h1:` and `sha512-` hashes, `versions` behind `kit tools --outdated` and `update`, and `sources:settings` behind
   `kit tools sources --env` and `kit tools run`. This repository's tools move from `url:` to their coordinates where a
   protocol serves them.
4. **Push targets, visibility, and `kit release --wait`.** The `targets` seam and its template, `vcs-tag`,
   `github-release`, `artifactory-local`, and `npm-publish`; `publish:` in `kit.yml`; `kit publish`, refused off a
   runner but for a pre-release; the tag's hash annotation; the four states, cached; kit flow's releases stage, `kit
   tools --outdated`'s upstream column, and `update` held to what the path serves; `kit release --wait=<duration>`.
5. **oci and artifactory.** The two providers: OCI's v2 API with its bearer token dance, and Artifactory's
   repositories by type, with Xray's block as blocked.
