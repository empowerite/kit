# Packages

A package is a directory under `packages/` in a release, and everything the kit delivers comes from one. A repository
activates the packages it wants, and `kit install`, `kit update`, and `kit check` work on that active set.

## Activation

- `kit install [<package>…]` activates `kit`, the one default package, plus the named packages, the packages they
  depend on, and the members they include. It declares the set under `packages:` in `kit.yml`.
- `kit packages` lists every package in reach, which are active, and which are excluded, filtered by state with
  `--active` or `--inactive`, by name or glob, and as JSON with `--json`. On a terminal, `kit packages` alone opens
  Packages in kittui: a row per package in reach, what it depends on and includes, and the active packages that
  need it, which adds, removes, and selects the packages selected by running the verbs below. `--text` lists them as
  text there; see [apps.md](apps.md).
- `kit packages add <package>…` activates more packages, placing only the paths the new set adds to the old one.
- `kit packages select <package>…` makes exactly these packages, plus what they depend on and include, the active
  set. It adds what is missing, removes the rest, and says what it kept for another package's sake. kit tui's
  package view runs it.
- A glob names every package it matches, in `kit install`, `kit packages add`, and `kit packages remove`:
  `kit packages add 'style/*'`. A glob that matches nothing is refused.
- `kit packages remove <package>…` deactivates packages. It removes the paths only the removed packages delivered,
  and takes the kit's entries back out of any merged `.json` file. A copied file the repository edited is named and
  left. A package that an active package depends on is refused. Before it takes anything out, it judges every
  region it would take out: a region whose markers are out of order, doubled, or never closed is named, nothing is
  taken out, and it exits 1.
- A bundle is a package that includes others, such as `empowerite`. Adding it activates its members, and a member
  can be removed on its own: `kit packages remove` lists it under `excludes:` in `kit.yml`, and
  `kit update` leaves it out. Naming it in `kit packages add` ends the exclusion.
- `kit update` resolves the active set again from the new release, so a new member of a bundle, a new dependency,
  or a new default package arrives with the release. An excluded member stays out, unless a package depends on it,
  which the update says.
- `kit uninstall` deactivates every package, the `kit` package included: it takes out what they delivered, the
  manifests, and the declaration, and removes `kit.yml` when nothing else is in it. A copied file the repository
  edited is named and left; `--dry-run` only names what would go. Before it takes anything out, `--dry-run` too, it
  judges every region it would take out: a region whose markers are out of order, doubled, or never closed is
  named, nothing is taken out, and it exits 1. What kit keeps in git's own directory stays: the context log and the
  cached kinds programs.

## Layout

```text
packages/<name>/
  package.yml          metadata
  files/<base>/       files mirrored at the base's root
  templates/<base>/   files rendered from the repository's answers, then mirrored the same way
  regions/<base>/     each file a region the kit owns inside a file the repository owns
  hosts/<base>/       for a region, the template of the file that holds it, at the region's path
```

A package must have `package.yml`. `files/`, `templates/`, `regions/`, and `hosts/` are optional, and the kit does
not read anything else in the package's directory. The base sets where a file lands:

| Base | Root | Record |
| --- | --- | --- |
| `repo` | the repository root | the declaration in `kit.yml`, committed; `.kit/manifest`, written beside what kit placed under `.kit/` and ignored like it |
| `home` | the user's home directory, `~` | `~/.kit/manifest`, on the user's machine; a CI runner has none |

- `files/home/.kit/<name>` is a user-level setting. Each machine has one copy, from whichever repository installed
  or updated last.
- `kit check` compares home paths with the release where `~/.kit/manifest` exists. Where it does not, as on a
  runner, which does not install anything there, `kit check` reports that it did not check them.
- A package that names any other base is refused at install, and the error lists the known bases.
- `KIT_HOME` replaces the home directory. The tests use it to stay out of the real one.

Every file a package delivers under `.kit/` opens with two lines in its own comment syntax, after a frontmatter's
opening `---` where it has one. The first says what the file is and which package it comes from. The second says
how to stop using it or replace it.

```yaml
# kit:generated -- the bash kind, from the kit package
# see: "Changing what kit adds" in .kit/1st.md, to replace it
```

A JSON file, which has no comments, carries both in a `"$comment"` key. A template is the one exception: its text is
the shape of the files `kit new file` makes from it, and the lines would pass into each of them. The gate fails a
delivered file without them.

## Metadata

`package.yml`, a rule's frontmatter, and `kit.yml` are YAML, read through `yq` 4 by `kit/modules/bash/yaml.sh`;
see [modules.md](modules.md#yaml).

```yaml
description: "One GitHub Actions job that runs kit ci"
default: true
depends:
  - kit
answers:
  ci_runner: ubuntu-latest
```

| Key | Meaning |
| --- | --- |
| `description` | Required. One line, shown by `kit packages`. |
| `default` | `true` makes the package active unless removed. A removed default package is listed under `excludes:`, so `kit update` leaves it out. In this release only `kit` sets it. |
| `depends` | Packages activated with this one. `kit packages remove` refuses a package another active package depends on. |
| `includes` | The members of a bundle: packages activated with this one, each of which `kit packages remove` can take out on its own. |
| `answers` | The default value of each template key. A repository overrides one under `answers:` in `kit.yml`. |
| `expects` | Lines the package expects in files the repository owns, each a `path`, a `pattern`, and a `remedy`; see below. |
| `notices` | What happens next, told when the kit places a path afresh or merges a new entry into it, each a `path` and a `text`; see below. |
| `policy` | Policy rules, with the keys of `kit.yml`'s `policy:`: each a pattern, or a map of a `pattern` and a `remedy`, and `commit_reference`. The kit package's are the kit's defaults; an active package's win over them, and `kit.yml`'s over all; see [policy.md](policy.md#packages-that-set-rules). |

[package.schema.json](../kit/packages/package.schema.json) is the schema, in JSON Schema 2020-12. This repository's gate
(its own CI command) validates every `package.yml` against it with `ajv`. Without a validator, `kit install` applies
the same contract and refuses, naming the problem:

- a key it does not know
- a missing `description`
- a `default` that is not `true` or `false`
- a dependency, or a member, the release does not have
- a `policy:` map with a subject the kit does not know, a rule that is neither a pattern nor a map of `pattern`
  and `remedy`, or a `commit_reference` that is neither `prefix` nor `trailer`

## Delivery

Each kind of source file is placed differently:

| Source | How it is placed |
| --- | --- |
| `files/<base>/<path>` | At `<path>` under the base's root: merged by key for a `.json` file, copied otherwise. |
| `templates/<base>/<path>` | Rendered, then placed like a `files/` file. |
| `regions/<base>/<path>` | Rendered, then written as a region inside the file at `<path>`. |

### Copies and merges

- A destination ending in `.json` is merged by key, except a schema (`*.schema.json`). The kit sets its keys and
  adds its array entries. On update or removal it takes out only the previous release's entries, so the
  repository's own keys and entries stay.
- Every other destination under `.kit/` is a link to its source when the source is inside the worktree, as the
  repository's own packages are, and otherwise a copy without the write bit. `.kit/.gitignore`, which kit writes,
  names each one, so git never sees them; it and `.kit/manifest` have no write bit either. `kit hydrate` brings them
  to that state from the declaration, and every verb that reads a delivered file does the same first.
- Every other destination outside `.kit/` is a copy.
- Where two active packages deliver the same path, the later package in dependency order wins.

### Includes

Where a file format supports includes, a package delivers a whole file of its own under `.kit/`. It then expects
the repository's file to include it with one line:

- lefthook's `extends`
- just's `import`
- git's `include`
- markdownlint's `extends`

The package declares each such line under `expects:`. An entry has a `path`, a `pattern` (an extended regular
expression that some line of that file must match), and a `remedy`.

```yaml
expects:
  - path: lefthook.yml
    pattern: '\.kit/lefthook\.yml'
    remedy: "put extends: [.kit/lefthook.yml] in lefthook.yml, so the kit's pre-commit commands run"
```

- `kit install` and `kit packages add` list each unmet expectation as left to do.
- `kit check` fails until every expectation is met.
- The kit does not write anything into the file. It belongs entirely to the repository.

A single-quoted value is taken literally, backslashes included. Write patterns that way.

### Notices

Some files change what a harness does only after the person answers it, such as a server a harness asks once
whether to start. The package says so under `notices:`. An entry has a `path`, a file the package delivers, and a
`text`, the line printed after it.

```yaml
notices:
  - path: .mcp.json
    text: "Claude Code asks once, at the start of the next session, whether to use the kit server this file names"
```

`kit install`, `kit packages add`, and `kit update` print a notice when they place its path afresh, or merge into it
an entry that was not there. When the repository already holds a path as the release has it, they
stay quiet.

The kit changes a file it touches in one of four ways:

| When the file | The kit uses |
| --- | --- |
| has a format that supports includes | an include line the repository adds |
| has a format with comments | a region |
| supports neither | a merge by key |
| belongs entirely to the kit | a link or a readonly copy under `.kit/`, a copy elsewhere |

### Regions

A file under `regions/<base>/` is the content of a region inside the file at the same path. It is rendered first,
like a template. The kit writes three things in the file's comment syntax: the bare directive `kit:generate <path>`,
the content, and the marker `kit:end <path>`. [generate.md](generate.md) describes directives. The comment syntax
follows the file's extension:

- `<!-- … -->` for markdown, HTML, XML, and SVG
- `//` for JavaScript, TypeScript, Go, Rust, C, C++, Java, Kotlin, Swift, C#, and Scala
- `--` for SQL, Lua, and Haskell
- `#` for the other kinds the kit knows: shell, YAML, TOML, Python, Ruby, Perl, just, make, and plain text
- a file with no extension takes `#`; a file with any other extension cannot carry a region

The block is placed like this:

- The file has the directive: the content between the directive and the end marker becomes what `kit region`
  prints.
- The file exists without the directive: the block is appended after a blank line.
- The file does not exist: the block becomes the whole file.

A package can state the shape of the file that holds its region, with a host template at the region's path under
`hosts/<base>/`. It is a template in the language [templates.md](templates.md) describes, whose holes stand for the
repository's own content, with the region's directive and end marker where the region belongs. `agents/generic`
gives `AGENTS.md` its title, a blank line, the region, then the repository's own instructions:

```text
# «title=AGENTS.md»

<!-- kit:generate AGENTS.md -->
<!-- kit:end AGENTS.md -->
«instructions»
```

With a host template, the block is placed like this instead:

- The file does not exist: the host template is made, each hole at its default, and the region is filled.
- The file exists without the directive: the block goes where the host template puts it, with a blank line on
  either side. That is after the fewest of the file's first lines that match the template's part before the
  region. When no run of first lines matches, the kit refuses, names the package, and asks for the directive where
  it belongs.
- `kit check` also holds the file to the host template, and names the line where it leaves it.

After that:

- The repository can write anything in the host template's holes, and, without a host template, can move the block
  anywhere in the file and write anything around it.
- `kit check` checks the region against its command.
- `kit update` regenerates the region and never merges it.
- Removal takes the block out, leaves the rest of the file, and removes the file if that leaves it empty.
- A region whose markers are out of order, doubled, or never closed is never written or taken out: the file is
  named and left as it is. `kit update`, `kit uninstall`, and `kit packages remove` judge every region before they
  write or take out anything, and with one refused they change nothing and exit 1.
- A file holds at most one kit region. Two packages that deliver a region to the same path are refused by name.
- A kind of file whose comment syntax the kit does not know, such as JSON, is refused.

### Templates

Files under `templates/<base>/` and `regions/<base>/` are rendered before they are placed. Each `@key@`
(autoconf's form) is replaced by the key's value. A template's result is then placed, digested, and merged exactly
like a `files/` file. Bash's `${var}` and GitHub's `${{ }}` cannot be mistaken for `@key@`, so a template can
contain both unchanged.

The values come from the whole of `kit.yml`:

- `@require_signing@` is a top-level setting.
- `@answers.ci_runner@` is an entry in a map.
- `@ci_runner@` is the entry under `answers:`.

A key with no value in `kit.yml` takes the package's default from its own `answers`. A key with neither
fails the install, and the error names the key and the package.

## Settings

`kit.yml` belongs to the repository. The kit writes three keys in it, `version`, `packages`, and `excludes`,
which declare the pinned release and the active packages, and reads the rest.

- Its schema is [kit.schema.json](../kit/packages/kit/files/repo/.kit/kit.schema.json), which the kit package
  delivers as `.kit/kit.schema.json`.
- An editor validates the file through the modeline `# yaml-language-server: $schema=.kit/kit.schema.json`.
- Before every install, update, and check, the kit checks the file against the schema's main points, and
  names any unknown key.

```yaml
answers:
  ci_runner: ubuntu-latest
```

| Key | Meaning |
| --- | --- |
| `version` | The kit release the repository is pinned to, written by `kit install` and `kit update`. A checkout of the kit itself declares `unreleased`. A manifest that still carries `version` and `packages` lines, from an earlier release, is read as the declaration until the next kit verb that places files moves it here. |
| `packages` | The active packages, in dependency order, written by `kit install`, `kit update`, and the `kit packages` verbs select, add, and remove. |
| `excludes` | Members of an active bundle, and default packages, removed with `kit packages remove`, which `kit update` leaves out. Written by the `kit packages` verbs and `kit update`. |
| `forge` | The code forge provider the workflow commands use, `ado`, `github`, or `gitlab`: the primary remote's. The code forge is the service that hosts the repository. When absent, it is detected from the primary's host: `github.com`, `gitlab.com`, or `dev.azure.com`, a CI runner's own server (`GITHUB_SERVER_URL`, `CI_SERVER_HOST`, `SYSTEM_COLLECTIONURI`), or a host the `gh` or `glab` CLI has logged in to. A primary on a host none of these names is refused where its pull request is read, naming this setting. `none` says the primary has no forge the kit knows, as on Bitbucket or a plain git server: nothing is asked of a forge, and it wins over the host and a runner's server. |
| `forges` | The code forge of each other remote, by name: `forges: {upstream: gitlab}`. A remote not named takes the code forge its host names, or none. |
| `primary` | The primary remote. Ahead and behind, the default branch, pull requests, builds, releases, and issues are read from it, and `kit rebase` and `kit release` read it. Default: `origin`. |
| `push` | The remote a branch goes to, as `push: origin` in a fork whose `primary` is `upstream`. `kit work` and `kit submit` fetch from it, look a branch up on it, fast-forward from it, and push to it, the branch's lease too, and the pull request opens on the primary from the branch there; on Azure DevOps, which opens none from a fork through `az`, `kit work` names where to open it. It must name a remote. Default: the primary. |
| `forge_settings` | Each code forge's own settings, by forge, read only when that forge is the primary remote's: `forge_settings: {ado: {type: Bug}}`, and `stand_ins` for any forge; see [architecture.md](architecture.md#code-forges-seams-and-providers). |
| `bot` | The name of this repository's bot: the slug of the GitHub App that acts for you here, as `kit bots` lists it. Without it, the bot is the only one in your bots file on the primary remote's host. |
| `gate` | What `kit gate` runs. When absent, the kit detects a justfile `ci` recipe, a Makefile `ci` target, or a `package.json` `ci` script. |
| `require_signing` | `true` makes `kit commit`, `kit work`, and `kit rebase` refuse before writing anything when git has no `user.signingkey`, makes `kit commit`, `kit work`'s start commit, and `kit rebase` sign every commit they make, refusing the commit when git cannot sign it, and makes `kit release` sign its tag. |
| `log_days` | How many days kit keeps its logs under `.kit/logs/`, 14 when absent; `0` turns them off. See [logs.md](logs.md). |
| `linked_exempt` | Logins, one per line, whose pull requests `kit linked` passes without an issue and `kit policy` passes unchecked, each saying so by name: a bot such as `dependabot[bot]`. On a branch, the list on its base counts; see [policy.md](policy.md#what-judges-a-branch). |
| `answers` | The values a package's templates use, overriding the package's own defaults. |
| `policy` | The patterns a branch, a commit subject, and a change title must match; see [policy.md](policy.md). |
| `policy_engine` | `patterns` or `opa`; see [policy.md](policy.md). |
| `facts` | `facts.read`: the globs of the files whose text the policy's facts carry; see [policy.md](policy.md#reading-a-files-text). |
| `publish` | The repository's own artifacts: what the runner built, what each is on the pull path, and the push targets `kit publish` sends a release and a pre-release to; see [features.md](features.md#publishing-and-pull-behind-push). |
| `sources` | The artifact sources: the releases a repository takes packages from, each with a `name`, a `url`, and a `digest`, and the pull path's default; see below. |
| `trackers` | Trackers other than the code forge's own whose issues the workflow accepts, each with a prefix and a provider; see [trackers.md](trackers.md). |
| `bash_path` | The bash module path: the directories `use` sources a bash module from, in order, starting with `kit:modules/bash/`. A directory that does not exist is skipped; see [modules.md](modules.md#the-module-path). |
| `tools` | The tools kit pins as the tool manager, by name: each a version and the url of its download, which `tools.lock` records for every platform; see [using-kit.md](toolchains/using-kit.md). |
| `flow` | The stages `kit flow` draws, in order: `issues`, `changes`, `builds`, `incoming`, `releases`. A stage left out is not drawn; `kit flow <stage>` still prints it. `issues` lists every open issue, in the backlog or underway, so `backlog` and `underway` are no stages. Default: all five, in that order. |

A provider setting, `forge`, a value of `forges`, `policy_engine`, or the `provider` of a tracker, takes one of the
kit's providers. It also takes one the repository adds on its bash module path: `forge: bitbucket` with a
`forge/bitbucket.sh` in a directory of that path. See [modules.md](modules.md#writing-a-bash-module).

## Sources

A repository takes packages from the kit's release, from its own `.kit/packages/`, and from the sources it names
under `sources:` in `kit.yml`. A source is an organization's rules, such as its own bundle: published once, taken by
every repository, always a release and never a copy.

```yaml
sources:
  - name: team
    url: git:empowerite/kit-team@v1.2.0
    digest: sha256:3b1f…
```

- A `url` is `git:<owner>/<repo>@<tag>`, the code forge's tarball of that tag, on GitHub, or on another host named
  first, as `git:gitlab.com/<owner>/<repo>@<tag>`; or any URL `curl` fetches, such as `https://…/team-1.2.0.tar.gz`.
- A source is pinned by the `sha256` digest of the tarball's bytes. It is fetched once into the kit's cache, under its
  digest, and refused when its bytes have any other digest, before anything of it is read. Moving a source to a new
  tag means pinning its new digest.
- A source's packages are under `packages/` in the tarball, or `kit/packages/` for a repository laid out as the kit's
  own is.
- A package name is looked up in the sources, the last first, then in the release, then in `.kit/packages/`, so a
  later source's package replaces one of the same name. `kit packages` shows where each comes from.
- A `bash_path` entry `<source>:<dir>` names a directory in a source, so a source can carry bash modules as well.
- A meta-package, one with only `depends` and `includes`, publishes a bundle: a source can carry a repository's
  `empowerite` as the release carries the kit's.

A file a source delivers is held to the source by `kit check`, like any other. `kit update` takes the source as the
pin names it. A file outside `.kit/` that came from a source and was edited keeps the edit, since the kit does not keep
an earlier copy of a source to merge against.

### Kinds and the pull path

Each entry under `sources:` has a `kind`, the protocol it speaks: `kit-packages`, the package sources above, when it
is left out. Every other kind is on the pull path, the sources kit searches, in order, for what it fetches, the first
hit winning. Besides a source, an entry may be one of two words: `direct`, the open internet, each kind at its public
home, and `off`, which ends the path.

```yaml
sources:
  - name: team
    url: git:empowerite/kit-team@v1.2.0
    digest: sha256:3b1f…
  - name: corp-go
    kind: goproxy
    url: https://proxy.corp/go
  - direct
```

- A kind is a provider of the `sources` seam, `modules/bash/sources/<kind>.sh`, from the kit or from the
  repository's own bash module path. Each takes the keys its header names, beyond `name`, `kind`, `url`, and
  `expect`, how long a pushed version usually takes to show there, as `30m`, `12h`, or `1d`.
- The kit's kinds: `kit-packages`; `github-releases`, a release's files at `<url>/<owner>/<repo>/releases/download/`,
  github.com, a GitHub Enterprise Server, or a mirror laid out the same; `goproxy`, a Go module proxy, which takes
  `private:`, the module patterns with no checksum database; and `npm`, a registry, which takes `scope:`, as
  `"@corp"`, for a registry that serves that scope alone; `dist`, a project's own download directory or a mirror of
  one, which takes `dist:`, the one it mirrors; `oci`, an OCI registry or a pull-through cache, which takes
  `prefix:`, the path the cache serves a registry's repositories under; and `artifactory`, an Artifactory
  repository asked as its `type:`, github-releases, goproxy, npm, or dist, at its `repository:`, a download it
  refuses with 403, as Xray's block, being blocked. What each serves kit as the tool manager, and how, is in
  [using kit as the tool manager](toolchains/using-kit.md#saying-what-a-tool-is-in-place-of-its-url).
- The pull path comes in layers, as `PATH` does, the first first: `KIT_SOURCES`, for a runner, entries apart by
  commas, each `direct`, `off`, or `<name>=<kind>:<url>`; `sources:` at the top of your `workspaces.yml` (see
  [workspaces.md](workspaces.md)), which applies to every repository you work in; and `kit.yml`'s, whose default,
  with none, is `direct`.
- A layer's entries go before the ones below it, and a name or `direct` met again is the first one's. `off` ends the
  path, so a layer that holds it replaces the layers below: a desk behind a mirror writes the mirror and `off` in its
  workspace, and never reaches the open internet.
- Package sources are the repository's: a workspace or `KIT_SOURCES` that names one is refused.
- `kit tools sources` prints the path, each entry with its kind, its url, and the layer it came from.
- A clone's `kit.yml` that names a source is what `kit trust` asks about, each source by its kind and url, before kit
  reads anything there.

The design, and what the kinds beyond `kit-packages` fetch, is in
[the artifact sources design](superpowers/specs/2026-11-12-artifact-sources-design.md).

## Packages a repository defines

A directory `.kit/packages/<name>/` in the repository is a package in every respect. It has the same parts:
`package.yml`, `files/<base>/`, `templates/<base>/`, `regions/<base>/`, and `hosts/<base>/`.

- The kit looks up a package name in the release first, then in `.kit/packages/`.
- It is validated like a delivered package, and it may depend on delivered packages.
- Activate it with `kit install <name>` or `kit packages add <name>`.
- `kit packages` shows it with its origin.
- Its paths are recorded in the manifest with their digests, so `kit check` can tell an edited installed copy
  from its source.
- On update, its files are compared with its own current source. An edit under `.kit/packages/` is never merged
  against a release.
- `kit context` finds a rule file it places under `.kit/rules/`, or a style file under `.kit/styles/`, like any
  other.

## This release's packages

| Package | Default | Depends on | Delivers |
| --- | --- | --- | --- |
| [`kit`](packages/kit.md) | yes | | `.kit/1st.md`, `.kit/README.md`, `.kit/welcome/*.md`, `.kit/kinds/<kind>.yml`, `.kit/rules/<kind>.md`, `.kit/kit.schema.json`; a region in `.gitignore` |
| [`workflow`](packages/workflow.md) | no | `kit` | `.kit/workflow.md`: the workflow from an issue to a merged pull request, plans as prose, and the worker contract |
| [`style/bash`](packages/style.md) | no | `kit` | `.kit/styles/bash.md` and the `script` and `module` templates |
| [`style/markdown`](packages/style.md) | no | `kit` | `.kit/styles/markdown.md`, its three templates, and `.kit/markdownlint.jsonc`, the settings of its linter |
| [`style/just`](packages/style.md) | no | `kit` | `.kit/styles/just.md` and the `root` and `module` templates |
| [`style/workflows`](packages/style.md) | no | `kit` | `.kit/styles/workflows.md` and the `workflow` template |
| [`style/yaml`](packages/style.md) | no | `kit` | `.kit/styles/yaml.md` and the `document` template |
| [`style/json`](packages/style.md) | no | `kit` | `.kit/styles/json.md` |
| [`style/gitignore`](packages/style.md) | no | `kit` | `.kit/styles/gitignore.md` and the `file` and `region` templates |
| [`style/prose`](packages/style.md) | no | `kit` | `.kit/styles/prose.md`, which governs every file of kind `text` |
| [`hooks/lefthook`](packages/hooks/lefthook.md) | no | `kit` | `.kit/lefthook.yml`, a pre-commit hook running `kit lint` and `kit generate --check` |
| [`github/ci`](packages/github/ci.md) | no | `kit` | `.github/workflows/kit-ci.yml`, from a template, running `kit ci`; answers `ci_runner`, `ci_timeout_minutes`, `ci_toolchain` |
| [`agents/generic`](packages/agents/generic.md) | no | `kit` | a region in `AGENTS.md` that sends any agent to `.kit/1st.md` |
| [`agents/claude`](packages/agents/claude.md) | no | `kit`, `agents/generic` | a region in `.claude/CLAUDE.md` importing `AGENTS.md`; `.claude/settings.json`, merged: the deny list and the `kit context --claude` hook; `.mcp.json`, merged: the kit's MCP server |
| [`agents/cursor`](packages/agents/cursor.md) | no | `kit`, `agents/generic` | `.cursor/rules/kit-<kind>.mdc`, one per kit rule, attached by its kind's paths, pointing at `.kit/rules/<kind>.md`; `.cursor/mcp.json`, merged: the kit's MCP server |
| [`agents/copilot`](packages/agents/copilot.md) | no | `kit`, `agents/generic` | `.github/instructions/kit-<kind>.instructions.md`, one per kit rule, applied to the rule's paths, pointing at the rule; `.vscode/mcp.json`, merged: the kit's MCP server |
| [`agents/gemini`](packages/agents/gemini.md) | no | `kit`, `agents/generic` | `.gemini/settings.json`, merged: `AGENTS.md` among the context files, the deny list as `run_shell_command` exclusions, and the kit's MCP server |
| [`policy/opa`](packages/policy/opa.md) | no | `kit` | `.kit/policy/kit.rego`, the naming policy in Rego, so `kit policy` runs on Open Policy Agent |
| [`policy/conventional`](packages/policy/conventional.md) | no | `kit` | no file: policy rules that hold commit subjects and pull request titles to Conventional Commits, the issue in a `Refs:` trailer |
| [`empowerite`](packages/empowerite.md) | no | `kit`; includes every `style/*`, `github/ci`, `agents/claude`, `hooks/lefthook`, and `workflow` | nothing of its own: it is the bundle Empowerite's repositories take |

No package in this release delivers to `home`. Each package's page, linked above, describes what it places and why.
