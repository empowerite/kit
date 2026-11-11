# Policy

A policy is the set of rules that a branch name, a commit subject, a change's title, and the tree of files must
meet. A change is a pull request on GitHub or a merge request on GitLab. Run `kit policy` to check the current
branch: its name, its commits, its open change, and the tree at HEAD. It prints each refusal and fails, and it
prints each warning too.

```sh
kit policy
```

- `kit policy` gathers the facts once, then the policy engine reports every refusal and every warning.
- `kit ci` runs it, after `kit linked` and before the gate (the repository's own CI command) and `kit check`, so a
  name that breaks the policy fails the delivered check like a failing test. That check blocks a merge only when a
  branch ruleset requires it.
- On a runner, the refusals and the warnings also appear in the step summary.

## The facts

`kit facts` prints the JSON document every engine reads. `kit policy --input` prints the same document.

`kit facts --moment=<moment>` chooses the point the facts are gathered for:

- `ci`, the default: the commits past the fork, the open change, and the tree at HEAD.
- `commit`: the tree in the index, `commits` as `[]`, and `change` as `null`. `kit commit` fills in the commit it makes.

```json
{
  "moment": "ci",
  "branch": "7-fix-the-thing",
  "default_branch": "main",
  "commits": [{"sha": "56d4f4b…", "subject": "#7 cli: fix the thing"}],
  "change": {"number": 3, "title": "Fix the thing", "body": "Closes #7", "author": "ada"},
  "tree": {
    "kit/bin/kit": {"type": "file", "executable": true, "size": 2048, "blob": "9f2c1a…", "kinds": ["bash"]}
  },
  "policy": {"branch": "…", "commit_subject": "…", "change_title": "…"}
}
```

| Field | Contents |
| --- | --- |
| `moment` | `commit` or `ci`: the point the facts are judged at. |
| `branch` | On a runner, the change's head branch. Elsewhere, at the desk or in any worktree, the current branch. On the default branch or a detached checkout it is `null`, and nothing is checked. |
| `default_branch` | The repository's default branch. |
| `commits` | Each commit past the fork from the branch's base, merges excluded: the default branch at origin, or on a runner the commit the change's event names as its base. The delivered workflow fetches full history for this. When the fork is not found, as in a shallow clone, `commits` is `null`, the commits cannot be checked, and `kit policy` fails. |
| `change` | The open change, read through the code forge (the service that hosts the repository): an object with its number, title, body, and author's login. `null` when the repository has no code forge, or the code forge answered that the branch has no open change, and `null` at commit. A `null` change leaves `change_title` unchecked. When the code forge cannot answer, for which change is open or for its title, body, or author, `change` is `{"error": "<line>"}`, the line kit says for the read that failed, such as `run: gh auth status (the open pull request on 7-fix-the-thing could not be read)`. `kit policy` refuses such facts before either engine runs, and so refuses a document given with `--facts` that holds one. |
| `tree` | The files git holds, keyed by path, read from git: the index at commit, and HEAD at ci. At either moment, a kind matched by its first line, such as a script's shebang, reads that line from the disk. |
| `policy` | The patterns, with the kit's defaults filled in. |

Each entry in `tree` has these fields:

| Field | Contents |
| --- | --- |
| `type` | `file`, `link`, or `submodule`. |
| `executable` | `true` for a file of mode 100755, and `false` for everything else. |
| `size` | The blob's bytes, and 0 for a submodule. |
| `blob` | The blob's object name, or the commit a submodule is at. |
| `kinds` | The kinds of file the path is, such as `bash`, or `[]` for none. On a branch, the kinds are the base's; see below. |
| `text` | The blob's content. Only a file matching a `facts.read` glob has it. |

### What kit commit judges

`kit commit` runs the policy before the commit exists. A plain `git commit` does not: until CI, only a harness's
deny list stops one.

- It judges the tree its own commit will make: HEAD plus the named paths, or the index when no paths are named.
- Its moment is `commit`, and `commits` holds the one commit it makes.
- When the engine errors, because `opa` is missing or a `.rego` file is broken, it refuses the commit. It fails closed.
- The engine and the patterns are read from the tree being judged, so an unstaged edit to `kit.yml` or
  `.kit/policy/` does not change the verdict.
- Warnings print, and the commit still goes through.

### What judges a branch

On a branch past the default one, `kit policy` and `kit linked` judge by the policy on the branch's base, the commit
the commits are counted from. A branch cannot loosen the rule that judges it: a pattern it blanks, a login it adds
under `linked_exempt`, an engine it names, a file it stops a rule from reading, a `forge` it drops or a `forges` entry
it changes, a kind it redefines, and a file it removes, adds, or edits under `.kit/policy/` change nothing.

- The base is `origin/<default>` at a desk. On a runner, it is the base the change's event names, which no command of
  the branch can move, as the gate could move `origin/<default>`. On GitHub that is the pull request's base commit,
  `pull_request.base.sha` in the event at `GITHUB_EVENT_PATH`, or a merge group's, `merge_group.base_sha`. On GitLab
  it is the target branch's tip: `CI_MERGE_REQUEST_TARGET_BRANCH_SHA` in a merged-results pipeline, else
  `origin/<target>` for the branch `CI_MERGE_REQUEST_TARGET_BRANCH_NAME` names, fetched from origin when it is not
  here; never the merge base, `CI_MERGE_REQUEST_DIFF_BASE_SHA`, which a branch chooses by where it forks. A runner
  that names a change and no base it can have fails both verbs, naming what is missing.
- In a merge queue, the branch is the merge group's head, `gh-readonly-queue/<base>/pr-<n>-<sha>`, which the kit's
  default `branch` pattern refuses. A repository that runs `kit ci` on `merge_group` events sets `policy.branch` to
  allow it.
- `kit ci` runs `kit linked` and `kit policy` first, before the gate and `kit check`, which run commands of the
  branch's own: the gate itself, and the commands the `kit:generate` directives name.
- `policy`, `policy_engine`, `linked_exempt`, `facts`, `forge`, and `forges` come from the base's `kit.yml`. Where
  the base has none of one, it is absent, and the kit's default applies, not the branch's: for `forge` and `forges`,
  the provider the remote's host names.
- `forges`, the map from remotes to providers, is the base's as `forge` is, so a branch cannot point the primary at
  a forge that reads no change.
- Every other setting is the checkout's. `primary` among them names the remote the base is read from, and a `primary`
  that names no remote fails.
- A branch that adds a file to `facts.read`, for a rule of its own, changes nothing until it is merged, as the rule
  itself does not.
- What `opa` loads is exactly the base's, and nothing on the disk under `.kit/policy/` is read: every file the base
  holds under `.kit/policy/`, in a subdirectory too, a data file beside the Rego included, a link resolved in the
  base's tree; and every file a package in the base's `packages:` delivers there, a template rendered with the base's
  answers.
- The `kinds` in the facts are the base's too: the kinds files the base holds under `.kit/kinds/` and those its
  packages deliver there, compiled into a matcher of their own, apart from the disk's. A kind the branch adds or
  redefines, to take a path out of a rule keyed on it, classifies nothing until it is merged.
- A package is looked up as delivery looks it up: in the sources the base's `kit.yml` names, the last first, fetched
  where the cache lacks them; then in the release; then in the repository's own `.kit/packages/`. A package under the
  repository's root, the repository's own or the release in a checkout of the kit, is read as the base holds it.
- A package the base lists that is in none of those places fails both verbs, naming it, so none of its rules is
  dropped.
- The packages are the ones `packages:` lists, as `kit hydrate` delivers them. `kit install` writes every package the
  set needs there; a bundle listed alone, without its members, delivers none of their Rego.
- A change to the policy is judged by the policy before it, and takes effect once it is on the default branch.
- On a runner, the branch is the change's head branch, whatever its name, so the detached checkout of a pull request
  is judged the same way. On the default branch, or a detached checkout elsewhere, both read the disk.
- A fork's pull request from its own `main` has `main` as its branch, which the default `branch` pattern refuses.
  A `policy.branch` on the default branch that matches `main` allows it.
- `kit policy --input` prints the facts as the base's policy has them, the base's kinds included. `kit facts` prints
  the disk's. `kit policy --facts=<file>` judges by the disk's, since it tests a rule.

A base with no `kit.yml` is one the kit is not installed on. There, both read the checkout's own settings and Rego,
as on the default branch, so the pull request that installs the kit sets its first policy.

A base that is not here, as in a clone that lacks `origin/<default>` or the event's commit, fails both verbs wherever
they run. They say `the base, <base>, is not here, so its policy cannot be read; fetch it with full history`, naming
the ref or the commit; a merge request's target that is not here and cannot be fetched is named the same way. A read
of the base's files, or of a source it names, that fails, for any reason other than the file not being there, fails
them too.

### Reading a file's text

A rule that judges content needs `text`. Name the files in `kit.yml` by glob:

```yaml
facts:
  read:
    - README.md
```

- A file's `text` is capped at 65536 bytes.
- A binary file or an oversized file has no `text`, and kit warns, naming it.
- A rule should treat a missing `text` as unknown, not as empty.

### Testing a rule

`kit policy --facts=<file>` judges a saved facts document, and does not gather any facts.

```sh
kit facts > fixture.json
kit policy --facts=fixture.json
```

Edit the fixture to test a rule against facts that the repository does not have.

- A relative `<file>` starts from the directory the command was typed in.
- A file that is not a facts document, a JSON object with a `policy` key, is refused.

## The rules

`policy:` in `kit.yml` holds one extended regular expression per subject.

- A subject that is absent takes the kit's default: the workflow's own naming from `.kit/1st.md`.
- An empty pattern turns that rule off.
- A subject the kit does not know is refused.
- A change whose author is a login under `linked_exempt` in `kit.yml` passes with no rule checked, the tree rules
  included, and `kit policy` says so by name. Its branch, its commits, and its pull request title are a bot's, such as
  `dependabot[bot]`'s. The same setting passes the change through `kit linked`; see
  [packages.md](packages.md#settings).

| Subject | Checks | Default |
| --- | --- | --- |
| `branch` | the branch's name | `^[a-z0-9]+-[a-z0-9][a-z0-9-]*$`, the form `kit work` creates |
| `commit_subject` | every commit's subject past the fork | `^([A-Z]+)?#[0-9]+ [^ ]`, starting with its issue's reference |
| `change_title` | the open change's title | `^[^[:cntrl:]]{1,120}$`, one line of at most 120 characters |

```yaml
policy:
  branch: '^(feature|fix)/[a-z0-9-]+$'
  commit_subject: ''
```

A single-quoted pattern is taken literally. Write patterns in the subset that bash's extended regular expressions
and OPA's RE2 share, so both engines read them the same way:

- character classes
- anchors
- alternation
- counted repetition

## The engines

| Engine | How it checks | Chosen by |
| --- | --- | --- |
| `patterns` | Matches each rule in bash. Needs no other tool. | Default. |
| `opa` | Open Policy Agent evaluates every `.rego` file under `.kit/policy/`, package `kit.policy`. The set `deny` of strings is the refusals. | `policy_engine: opa`, or any `.rego` file in `.kit/policy/` when `policy_engine` is not set. |

The `opa` engine needs `opa`. Pin it in the repository's toolchain, then run `kit tools install`.

The `policy/opa` package delivers `.kit/policy/kit.rego`: the patterns engine's rules written in Rego, with the same
refusals. Because it is a `.rego` file, adding the package selects the `opa` engine, and the results stay the same.

Rules over `tree` need `opa`. The `patterns` engine ignores `tree`.

### Refusals and warnings

The engine reads two sets of strings from package `kit.policy`:

- `deny` holds refusals. Any one fails `kit policy`. Rego that does not define `deny` at all, as in a misspelled
  package, is refused too, since nothing was judged.
- `warn` holds warnings. They are printed and never fail anything.

### A rule of your own, through a local package

Rego that belongs to one repository ships through a local package, as templates do. Put the file in the package's
source, and kit delivers it as a link:

```text
.kit/packages/house/files/repo/.kit/policy/house.rego   the source
.kit/policy/house.rego                                  delivered by kit, as a link
```

- The package must list `policy/opa` under `depends:`.
- Without it, `kit.rego` is missing. Any `.rego` file selects `opa`, so the naming rules would silently stop.
- `kit packages add` brings in what a package depends on. `kit hydrate` does not: it reads only `packages:` in
  `kit.yml`.
- So a package that is already active, and gains the line, needs one more step. Run `kit packages add policy/opa`,
  or list `policy/opa` under `packages:` in `kit.yml`.

```yaml
description: "This repository's own tree rules"
depends:
  - policy/opa
```

This repository's `house` package holds a rule over `tree`. It refuses a file in `kit/bin/` that is not executable:

```rego
package kit.policy

import rego.v1

bin := "kit/bin/"

# The name after kit/bin/, for a file directly in it.
bin_name(path) := name if {
    startswith(path, bin)
    name := substring(path, count(bin), -1)
    not contains(name, "/")
}

deny contains sprintf("%s is not executable", [path]) if {
    some path, f in input.tree
    bin_name(path)
    not f.executable
}
```

A rule can also judge the commits:

```rego
package kit.policy

import rego.v1

deny contains msg if {
    count(input.commits) > 20
    msg := sprintf("%d commits; split the change", [count(input.commits)])
}
```

`tests/policy.bats` checks that both engines give the same refusals for the same facts.
