# git

git is the one integration kit cannot work without: kit works in a git checkout, and needs none of the others, the
code forge, the tracker, CI, or CD, to install, update, check, lint, or run `kit context`. kit asks git for the
checkout and its root, its worktrees and their branches, the remotes and which of them is the primary, how far a
branch is ahead and behind the default branch, the commits and their subjects, the tags, the index and the trees that
`kit commit`, `kit lint`, and `kit policy` judge, the hooks lefthook installs, and a handful of config keys. The index
of these pages is [1st.md](1st.md).

kit parses only the formats git documents for scripts: the porcelain formats of `git status` and `git worktree list`,
`git for-each-ref --format`, `git log --format`, the NUL-separated path lists, and the plumbing of `rev-parse`,
`rev-list`, `ls-files`, `ls-tree`, `cat-file`, and the raw output of `diff-tree` and `diff-index`. It reads git's
human words in three places only: it shows the first line of a failed fetch, it tells a push refused as not
fast-forward by `non-fast-forward`, `fetch first`, or `stale info`, and it tells a commit git could not sign by
`failed to sign` or `failed to write commit object`. A remote name reaches git after `--end-of-options` and a path as
a `:(literal)` pathspec, so neither reads as an option or a pattern.

kit states no minimum git version: `kit welcome` checks only that `git --version` answers. It calls
`git rev-parse --path-format=absolute`, which git 2.31 added, and an older git prints the option back as a path. A
package source written `git:${owner}/${repo}@${tag}` is the forge's tarball of the tag, fetched with `curl`, not by
git.
see: [forges.md](forges.md), [../workspaces.md](../workspaces.md), `kit/modules/bash/git.sh`,
`kit/modules/bash/commit.sh`, `kit/modules/bash/workspace.sh`, `kit/modules/jq/workspace.jq`,
`kit/modules/bash/flow.sh`

## Scope

| Capability | Kind | Used by |
| --- | --- | --- |
| git is on PATH | read | `welcome:tool`: `kit welcome`; `guard:require tool:git`: `kit commit`, `kit submit`, `kit rebase`, `kit release`, `kit flow`, `kit workspace` |
| Inside a repository, and its root | read | `kit:enter`, which moves every verb to the root; `welcome:repository`: `kit welcome`; `workspaces:in-repository`: `kit flow`, `kit workspace`; `context:relative`, `policy:from`, `across:repos`, `materialize:all`: `kit install`, `kit update`, `kit check` |
| The git directory and the common one | read | `git:main-root`: `kit commit`, `kit rebase`, `kit work`, the desk stage, `kit workspace`; `flow:in-worktree`: the worktrees stage; `welcome:where`; the files kit keeps under `${common}/kit/`: `context:log-path` and `context:matcher` (`kit context`), `flow:kept` (`kit flow`), `forge:snapshots` (the forge seam), `toolchain:kept` and `toolchain:recorded` (the toolchain chosen here) |
| The branch checked out, and HEAD's commit | read | `commit:branch`: `kit commit`, `kit submit`; `git:rebase`: `kit rebase`; `policy:branch`: `kit policy`, `kit facts`; `issue:start`: `kit work`; `changes:documents`: the changes stage; each provider's `pr-here`: `kit linked`, `kit ready`, `kit settle`; the desk and worktrees stages; `welcome:where` |
| Uncommitted changes | read | `git:clean`: `kit tidy`, `kit rebase`, `kit work`; `flow:checkout`: the desk and worktrees stages; `commit:choices`: the paths `kit commit` completes |
| A checkout's status with its branch | read | `workspace:worktrees`: the worktrees stage of `kit workspace`; `marks:status`: the desk and worktrees marks of `kit flow --marks`; `workspace:marks`: `kit workspace --marks` |
| Every worktree, the desk first | read | `git:linked-worktrees`: the worktrees stage; `git:worktrees`: `kit tidy`; `issue:worktree`: `kit work`; `issues:checkouts`: the issues stage; `workspace:paths`, `workspace::worktrees`, `workspace::heads`: `kit workspace`; `across:branches`: `kit workspace` over a workspace; `marks:desk`, `marks:worktrees` |
| Add a worktree for a branch | write | `issue:worktree`: `kit work` |
| Remove a finished worktree, and prune git's records | write | `git:tidy-worktree`, `git:tidy`: `kit tidy`; `git:tidy-wait`: `kit tidy --wait`, once the change merges |
| The default branch | read | `git:default-branch`, the primary's HEAD, else `main`: `kit commit`, `kit rebase`, `kit tidy`, `kit work`, `kit release`, `kit facts`, `kit policy`, `kit flow`, `kit workspace`; `flow:remote-default`: the incoming stage; `workspace:remote-facts`: the remotes stage of `kit workspace`; `policy:base` says why one is missing |
| Local and remote branches, with their upstreams | read | `flow:incoming-rows`: the incoming stage, the local branches on each remote branch; `workspace:branches`: the branches stage of `kit workspace`; `issues:branches` and `issues:documents`: the issues stage; `across:branches`; `git:gone-branches`: `kit tidy`, the worktrees stage |
| A ref names a commit | read | `commit:submit`, `commit:index`, `commit:named`, `commit:moved`: `kit commit`, `kit submit`; `git:rebase`, `git:unpushed`; `issue:worktree`; `releases:release`; `policy:base`, `facts:document`; `workspace:base`; `flow:remote-default`, `flow:checkout` |
| A branch name is valid | read | `policy:target-here`: `kit policy` on a GitLab merge request |
| A branch's upstream | read | `git:pushes-to`: `kit submit`, `kit work`, `kit rebase` |
| Delete a branch whose upstream is gone | write | `git:tidy`: `kit tidy` |
| The remotes, the primary first | read | `forge:primary`, `forge:remotes`: the forge seam, `kit tidy`, the incoming stage, `kit workspace`, the incoming mark; `policy:base` |
| A remote's fetch and push URLs | read | `forge:host` and `forge:provider`, which pick the forge; `forge:snapshots`; `forge/gitlab:project`, `forge/ado:place`; `flow:picture`; `workspace:remote-facts`; `workspaces:member-of`, `workspaces:remotes-json`: `kit workspaces`, `kit workspace` over a workspace |
| What a remote has, without fetching | read | `releases:on-origin`, `releases:release`: `kit release`; `issue:branch`: `kit work`; `marks:incoming`: the incoming mark |
| Fetch a remote | write | `git:tidy`: `kit tidy`, every remote; `git:rebase`: `kit rebase`; `issue:start`, `issue:branch`: `kit work`; `releases:release`: `kit release`; `policy:target-here`: `kit policy`; `flow:incoming`: the incoming stage; `flow:picture`: `kit flow`, before the desk and worktrees stages |
| Push a branch or a tag | write | `git:push`: `kit submit`, `kit work`, and `kit rebase`, which forces with a lease; `issue:branch`: `kit work` makes the branch on the push remote; `releases:release`: `kit release` |
| Ahead and behind the default branch | read | `flow:incoming-rows`: the incoming stage, a local branch against its upstream; `flow:fork`, `flow:checkout`: the desk and worktrees stages; `workspace:worktrees`: `kit workspace` |
| Commits no remote has | read | `git:unpushed`: `kit tidy`; `commit:submit`: `kit submit`; `releases:release`: `kit release`; `issue:start-commit`: `kit work`; `workspace:commits`: `kit workspace` |
| Ancestry | read | `flow:incoming-rows`: the incoming stage, a branch force-pushed; `git:tidy-worktree`: `kit tidy`; `git:rebase`: `kit rebase`; `releases:release`: `kit release`; `facts:document`: `kit facts`, `kit policy` |
| Commits and their subjects | read | `facts:document`: `kit facts`, `kit policy`; `flow:fetch-json`: the incoming stage; `flow:incoming-rows`, `flow:tip`: the incoming stage's rows, each commit's author and age too; `flow:checkout`: the desk and worktrees stages; `workspace:commits`: the commits stage of `kit workspace`; `commit:behind`, `commit:moved`, `commit:commit`: `kit commit`; `releases:release`; `issue:sweep-push` |
| The lines a checkout changes | read | `flow:checkout`: the desk and worktrees stages |
| Make a commit | write | `git:commit`: `kit commit`, and `kit work`'s empty start commit |
| Rebase onto the default branch | write | `git:rebase`: `kit rebase` |
| The tags | read | `forge:tags`: each provider's runs, for the builds stage; `workspace:tags`: the tags stage of `kit workspace`; `flow:refs`: the incoming stage; `flow:incoming-rows`: the incoming stage's rows, a tag's commit, tagger, and subject; `workspace:marks` |
| Tag a release and push it | write | `releases:release`: `kit release` |
| The staged paths | read | `kit lint` with no paths; `commit:checks`, `commit:commit`: `kit commit`; `commit:submit`: `kit submit` says they stay staged |
| The tracked paths | read | `kit lint --all-files`; `generate:discover`: `kit generate`; `syntax:tracked`: the paths `kit context`, `kit lint`, and `kit generate` complete; `context:named`: `kit context`; `materialize:all`: `kit install`, `kit update`, `kit check`; `commit:named`, `commit:index`: `kit commit` |
| The files of the index or a tree, as facts | read | `facts:tree`: `kit facts`, `kit policy`, `kit commit` |
| A file as the index or a tree holds it | read | `policy:object`, `policy:blob`, `policy:base-held`, `policy:base-tree`: `kit policy` and `kit commit` read `kit.yml` and `.kit/policy/` as the base holds them; `commit:behind`, `commit:unchanged` |
| The tree a commit would make | write | `commit:index`, `commit:tree`, `commit:same-tree`: `kit commit` builds it in a scratch index and writes its tree objects |
| The shared index, brought up to a commit made from another | write | `commit:shared-index`, `commit:made`, `commit:plan`, `commit:share`: `kit commit` from the index `GIT_INDEX_FILE` names |
| Stage every change to a tracked file | write | `commit:commit`: `kit commit --all` |
| Sweep the desk's work into the worktree | write | `issue:sweep-push`, `issue:sweep-apply`, `issue:sweep-restore`: `kit work` |
| A path's attributes, and whether it is ignored | read | `template:exempt`: `kit lint`; `issue:start`: `kit work` warns when `worktrees/` is not ignored |
| Merge a delivered file the repository edited | write | `kit update`, three ways |
| A hash of what git printed | read | `workspace:marks`: `kit workspace --marks`; `across:repos-mark`: the repos mark over a workspace |
| The signing config | read | `guard:require signing-key`: `kit commit`, `kit work`, and `kit rebase` with `require_signing` true; `commit:commit` warns when `commit.gpgsign` is not true |
| The pre-commit hook | read | `welcome:judge`: `kit welcome`; `uninstall:left`: `kit uninstall` |

## Read

### Read the checkout

| Capability | Command |
| --- | --- |
| git is on PATH | `git --version`, its first line |
| Inside a repository | `git rev-parse --git-dir`; `git rev-parse --is-inside-work-tree` |
| The root | `git rev-parse --show-toplevel`; `git -C "${dir}" rev-parse --show-toplevel` |
| The git directory and the common one | `git rev-parse --path-format=absolute --git-dir`; `git rev-parse --path-format=absolute --git-common-dir`; both at once, `git rev-parse --path-format=absolute --git-dir --git-common-dir` |
| The main checkout | `git -C "${path}" rev-parse --path-format=absolute --git-common-dir`, its parent directory |
| The branch checked out | `git symbolic-ref --quiet --short HEAD`; `git -C "${path}" symbolic-ref --quiet --short HEAD`; `git -C "${root}" symbolic-ref --quiet HEAD` for the full ref; `git -C "${root}" rev-parse --abbrev-ref HEAD`; `git branch --show-current` |
| HEAD's commit | `git rev-parse --verify --quiet HEAD`; `git -C "${worktree}" rev-parse HEAD`; `git rev-parse --short HEAD` |
| The commit kit commit made, and its parent | `git rev-list --first-parent "${head}..HEAD"`, its last line; `git rev-list --first-parent HEAD` before the first commit; `git rev-parse --verify --quiet "${made}^"` |
| The shared index, and the index in use | `git rev-parse --path-format=absolute --git-path index`, with `GIT_INDEX_FILE` unset and as it is |
| Uncommitted changes | `git -C "${path}" status --porcelain`; empty is clean |
| The paths git status reports | `git status --porcelain=v1 -z` |
| A checkout's status with its branch | `git -C "${path}" status --porcelain=v2 --branch` |

### Read the worktrees

| Capability | Command |
| --- | --- |
| Every worktree, the desk first | `git worktree list --porcelain`; `git -C "${path}" worktree list --porcelain` |

### Read the branches and refs

| Capability | Command |
| --- | --- |
| The default branch | `git -C "${path}" symbolic-ref --quiet "refs/remotes/${primary}/HEAD"`; `git symbolic-ref --quiet --short "refs/remotes/${remote}/HEAD"` |
| A remote's branch of the primary's default name, when its HEAD is not recorded | `git show-ref --verify --quiet "refs/remotes/${remote}/${default}"` |
| A local branch exists | `git show-ref --verify --quiet "refs/heads/${branch}"` |
| A ref names a commit | `git rev-parse --quiet --verify "refs/remotes/${remote}/${branch}^{commit}"`; `git rev-parse --verify --quiet "${base}^{commit}"`; `git rev-parse --verify --quiet "refs/remotes/${primary}/${default}"`; `git rev-parse --quiet --verify "refs/tags/${tag}"`; `git cat-file -e "${head}^{commit}"` |
| Local branches whose upstream is gone | `git for-each-ref --format='%(refname:short) %(upstream:track)' refs/heads/` |
| The local branches' names | `git for-each-ref --format='%(refname:short)' refs/heads`; `git -C "${path}" for-each-ref --format='%(refname:short)' refs/heads`; `git for-each-ref --format='%(refname:lstrip=2)' refs/heads/` |
| The primary's branches' names | `git for-each-ref --format='%(refname:lstrip=3)' -- "refs/remotes/${primary}/"` |
| Local and remote branches, with their upstreams | `git for-each-ref refs/heads refs/remotes --format='%(refname)%1f%(symref)%1f%(upstream:remotename)%1f%(upstream:remoteref)%1f%(upstream:track)%1f%(worktreepath)%1f%(objectname:short)%1f%(committerdate:unix)%1f%(contents:subject)'` |
| How many branches a remote has | `git for-each-ref --format='%(symref)' "refs/remotes/${remote}/"`, its empty lines counted |
| The refs a fetch moves | `git for-each-ref --format='%(refname) %(objectname)' "refs/remotes/${remote}" refs/tags`; `git for-each-ref --format='%(objectname) %(refname)' refs/remotes` |
| A branch name is valid | `git check-ref-format --branch "${target}"` |
| A branch's upstream | `git -C "${path}" config --get "branch.${branch}.remote"`; `git -C "${path}" config --get "branch.${branch}.merge"` |

### Read the remotes

| Capability | Command |
| --- | --- |
| The remotes | `git remote`; `git remote -v`, hashed for a mark |
| A remote's fetch and push URLs | `git remote get-url --end-of-options "${remote}"`; `git remote get-url --push --end-of-options "${remote}"`; `git -C "${dir}" remote get-url --end-of-options "${primary}"` |
| The release tags on the primary | `git ls-remote --tags --end-of-options "${primary}" 'refs/tags/release/*'` |
| A tag on the primary | `git ls-remote --tags --end-of-options "${primary}" "refs/tags/${tag}"` |
| An issue's branch on origin | `git ls-remote --heads origin "${key}-*"` |
| Every remote's branches and tags, asking for no password | `GIT_TERMINAL_PROMPT=0 GIT_SSH_COMMAND="${GIT_SSH_COMMAND:-ssh -o BatchMode=yes}" git ls-remote --heads --tags --end-of-options "${remote}"` |

### Read ahead and behind

| Capability | Command |
| --- | --- |
| A worktree ahead and behind the default branch | `git -C "${path}" rev-list --left-right --count "${base}...HEAD"` |
| Where a branch left the default branch | `git merge-base --end-of-options "${primary}/${default}" "${branch}"`, else `git merge-base --end-of-options "${default}" "${branch}"` |
| Ahead and behind that fork | `git rev-list --count "${fork}..${branch}"`; `git rev-list --count "${fork}..${primary}/${default}"` |
| Ahead of the remote's branch | `git rev-list --count "refs/remotes/${remote}/${branch}..HEAD"` |
| The remote's commits a worktree lacks | `git -C "${worktree}" rev-list --count "HEAD..${theirs}"` |
| A branch's commits no remote has | `git rev-list --count "refs/heads/${branch}" --not --remotes`; then `git rev-list --count "refs/heads/${branch}" --not --remotes "${head}"` |
| The commits a tag push sends | `git rev-list --count HEAD --not --remotes="${primary}"` |
| The local commits no remote has | `git rev-list --max-count=1000 --branches --not --remotes` |
| Commits past the default branch | `git -C "${wt}" rev-list "origin/${default}..HEAD"`; empty is none |
| Ancestry | `git -C "${path}" merge-base --is-ancestor HEAD "${primary}/${default}"`; `git merge-base --is-ancestor HEAD "${primary}/${default}"`; `git -C "${worktree}" merge-base --is-ancestor "${theirs}" HEAD`; `git merge-base "${base}" HEAD` |

### Read the commits

| Capability | Command |
| --- | --- |
| The commits past a branch's base | `git log --no-merges --format='%H%x1f%s' "${base}..HEAD"` |
| The commits a fetch brought to the default branch | `git log --format='%h%x09%s' "${old}..${new}"` |
| A checkout's commits past its fork | `git -C "${path}" log --format='%h%x09%s' "${fork}..${branch}"` |
| The commits each local branch holds | `git rev-list --max-count="${workspace_commits}" "refs/heads/${branch}" --not "${base}"` |
| The local branches' commits | `git log --branches --not "${base}" --max-count="${workspace_commits}" --format='%H%x1f%h%x1f%ct%x1f%an%x1f%s'` |
| A commit's subject | `git log -1 --format=%s HEAD`; `git log -1 --format='%h %s'`; `git log -1 --format='%h %s' "${last}"`; `git -C "${root}" log -1 --format=%s "${after}"` |
| The last commit that changed a path, and the path before it | `git log -1 --format=%H HEAD -- ":(literal)${path}"`; `git rev-parse --verify --quiet "${last}^:${path}"` |
| The lines a checkout changes | `git -C "${path}" diff --numstat "${fork}"`, or `HEAD` with no fork |

### Read the tags

| Capability | Command |
| --- | --- |
| The tags' names | `git for-each-ref --format='%(refname:lstrip=2)' refs/tags` |
| The newest tags, with their commits | `git for-each-ref refs/tags --sort=-creatordate --count="${workspace_tags}" --format='%(refname:short)%1f%(objecttype)%1f%(*objectname:short)%1f%(objectname:short)%1f%(taggername)%1f%(creatordate:unix)%1f%(contents:subject)'` |
| The refs a mark hashes | `git for-each-ref --format='%(refname) %(objectname)' refs/tags`; the same of `refs/heads refs/remotes`; `git for-each-ref --format='%(refname) %(objectname) %(upstream) %(worktreepath)' refs/heads refs/remotes`; `git for-each-ref --format='%(refname) %(symref)' refs/remotes`; `git for-each-ref --format='%(refname)' refs/heads` |

### Read the index and trees

| Capability | Command |
| --- | --- |
| The staged paths | `git diff --cached --name-only --diff-filter=ACMR -z`; `git diff --cached --name-only --diff-filter=ACMRD -z`; `git diff --cached --name-only -z` |
| The tracked paths | `git ls-files -z`; `git ls-files -z --cached --others --exclude-standard`; `git -C "${kit_invoked_in}" ls-files`; `git ls-files -- ${paths} .kit/manifest .kit/.gitignore` |
| The tracked paths under a named path | `git ls-files -z -- ":(literal)${path}"`; `git ls-files -z --with-tree=HEAD -- ":(literal)${path}"`; `GIT_INDEX_FILE=${index} git ls-files -z -- ":(literal)${path}"` |
| The index's entries | `git ls-files -z --stage`; `git ls-files --stage -- "${path}"`; `git ls-files --stage -- ":(literal)${path}"` |
| A tree's entries | `git ls-tree -z -r --full-tree "${rev}"`; `git ls-tree -z -r --full-tree "${base}" -- "${path}/"`; `git ls-tree --full-tree "${rev}" -- "${path}"` |
| The blobs' sizes and contents | `git cat-file --batch-check='%(objectsize)'`, one object name a line on its input; `git cat-file -s "${blob}"`; `git cat-file blob "${blob}"` |
| Paths as HEAD or the index has them | `git diff --quiet HEAD -- ":(literal)${path}"…`; `git diff --quiet -- ":(literal)${path}"` |
| The paths a commit changed, and the shared index's paths that differ from its parent | `git diff-tree -r -z --no-renames "${base}" "${commit}"`; `GIT_INDEX_FILE=${shared} git diff-index --cached -z --no-renames "${base}"` |
| A path is generated | `git check-attr linguist-generated -- "${path}"` |
| `worktrees/` is ignored | `git -C "${main_root}" check-ignore --quiet worktrees` |
| The stash on top, and a stash's slot | `git -C "${root}" rev-parse --quiet --verify refs/stash`; `git -C "${root}" stash list --format='%gd %H'` |
| A hash of what git printed | `git hash-object --stdin`, which writes no object |

### Read the config and hooks

| Capability | Command |
| --- | --- |
| A signing key is set | `git config user.signingkey` |
| Commits here are signed | `git config --type=bool commit.gpgsign` |
| The pre-commit hook runs lefthook | `git rev-parse --git-path hooks/pre-commit`, the file read for `lefthook` |

## Write

| Capability | Command |
| --- | --- |
| Fetch every remote, pruned | `git fetch --quiet --prune --end-of-options "${remote}"`; `kit rebase` runs it with `git -C "${worktree}"` |
| Fetch origin | `git fetch --quiet --prune origin`; `git fetch --quiet origin` |
| Fetch the default branch | `git fetch --quiet --end-of-options "${primary}" "${default}"` |
| Fetch a merge request's target | `git fetch --quiet --no-tags "${primary}" "+refs/heads/${target}:refs/remotes/${primary}/${target}"` |
| Make an issue's branch on origin | `git push --quiet origin "refs/remotes/origin/${default}:refs/heads/${branch}"` |
| Push a branch to its own name | `git -C "${path}" push --quiet --end-of-options "${remote}" "refs/heads/${branch}:refs/heads/${branch}"`; `--set-upstream` before `--end-of-options` on a first push |
| Push a rebased branch | `git -C "${worktree}" push --quiet --force-with-lease="refs/heads/${branch}:${theirs}" --end-of-options "${remote}" "refs/heads/${branch}:refs/heads/${branch}"` |
| Push a release tag | `git push --quiet --end-of-options "${primary}" "refs/tags/${tag}"` |
| Add a worktree | `git worktree add --quiet "${wt}" "${branch}"`; `git worktree add --quiet "${wt}" -b "${branch}" "origin/${branch}"` for a branch only origin has |
| Remove a worktree | `git worktree remove "${path}"` |
| Prune git's worktree records | `git worktree prune --verbose`; `git worktree prune --verbose --dry-run` |
| Delete a branch | `git branch -D "${branch}"` |
| Make a commit | `git -C "${path}" commit --quiet --cleanup=whitespace --file "${message}"`, with `--gpg-sign` under `require_signing`, and `-- ":(literal)${path}"…` for named paths |
| Make the start commit | `git -C "${wt}" commit --quiet --allow-empty --only --cleanup=whitespace --message "${ref} start: ${title}"`, with `--gpg-sign` under `require_signing` |
| Stage every change to a tracked file | `git add --update` |
| Bring the shared index up to a commit | `GIT_INDEX_FILE=${shared} git read-tree -m -i "${base}" "${target}"`, the two-tree merge, under git's lock; `GIT_INDEX_FILE=${shared} git update-index -z --index-info` when it refuses; the target built with `GIT_INDEX_FILE=${work}/index git read-tree "${commit}"`, `git update-index -z --index-info`, and `git write-tree`; `git hash-object -t tree -w /dev/null`, the empty tree, as the parent of a first commit |
| Build the tree a commit would make | `GIT_INDEX_FILE=${index} git read-tree HEAD`, or `GIT_INDEX_FILE=${index} git read-tree --empty` before the first commit; `GIT_INDEX_FILE=${index} git update-index --add --remove --replace -z --stdin`; `GIT_INDEX_FILE=${index} git write-tree`; `git write-tree` for the index itself |
| Rebase | `git -C "${worktree}" rebase --quiet "${target}"`, with `--gpg-sign` under `require_signing` |
| Undo a rebase that conflicts | `git -C "${worktree}" rebase --abort`; `git -C "${worktree}" reset --quiet --hard "${before}"` |
| Sweep the desk's work | `git -C "${root}" stash push --include-untracked --message "${message}"`; `git -C "${wt}" stash apply "${stash}"`; `git -C "${root}" stash apply --index "${stash}"` to put it back; `git -C "${root}" stash drop "${slot}"` |
| Fast-forward the desk | `git -C "${root}" merge --ff-only "origin/${default}"` |
| Tag a release | `git tag --annotate --message "release ${version}" "${tag}"`, with `--sign` under `require_signing` |
| Delete a tag whose push failed | `git tag --delete "${tag}"` |
| Merge a delivered file three ways | `git merge-file --marker-size 7 "${dest}" "${previous_src}" "${src}"` |

kit never runs `git config` to write, `git remote add`, or `git remote set-head`; a push with `--set-upstream` and a
`git worktree add -b` set a branch's upstream. Where a remedy is git's, kit names the command for the person to run:
`git rm --cached`, `git restore`, `git restore --staged`, `git add`, `git fsck`, `git remote add ${primary} ${url}`,
`git remote set-head ${primary} --auto`, `git config user.signingkey ${key}`, `git fetch ${primary} ${head}`,
`git pull --ff-only`, and `git stash list` or `git stash apply ${stash}`. git runs the hooks: `git commit`, which
`kit commit` runs, runs the pre-commit hook lefthook installs, and kit only reads whether that hook names lefthook.

## Formats kit parses

### git worktree list --porcelain

One record per worktree, the main checkout first, each line `${label} ${value}` or a bare label, and a blank line
after each record. `git:linked-worktrees`, `git:worktrees`, `workspace::worktrees`, `workspace::heads`,
`issues:checkouts`, `issue:worktree`, and `across:branches` read it; the marks hash it whole.
[git's porcelain format](https://git-scm.com/docs/git-worktree#_porcelain_format).

| Field | What it holds | What kit does with it |
| --- | --- | --- |
| `worktree ${path}` | the worktree's absolute path, which starts the record | the row's key and path; the first record is the desk, which `git:linked-worktrees` leaves out; `git:worktrees` keeps only those under `worktrees/` |
| `HEAD ${sha}` | the commit checked out | `kit workspace` shows its first 8 characters, and maps each commit to the worktrees whose HEAD it is |
| `branch ${ref}` | the branch checked out, as `refs/heads/${branch}` | the branch, `refs/heads/` taken off; `kit tidy` keeps a branch checked out in a worktree it keeps |
| `detached` | HEAD names no branch | `kit tidy` removes such a worktree only when the default branch has its commit |
| `bare` | the repository has no working tree | not read |
| `locked [${reason}]` | `git worktree lock` holds it | `kit workspace` marks the row locked; `git worktree remove` refuses it, and `kit tidy` keeps it |
| `prunable [${reason}]` | its directory is gone | `kit workspace` marks the row prunable; the worktrees stage warns and names `kit tidy` |

### git status --porcelain=v2 --branch

Header lines that start with `#`, then one line per changed path. `workspace::status` reads it for the worktrees
stage of `kit workspace`, and `marks:status` hashes it whole for the desk and worktrees marks.
[git's porcelain format version 2](https://git-scm.com/docs/git-status#_porcelain_format_version_2).

| Field | What it holds | What kit does with it |
| --- | --- | --- |
| `# branch.oid ${sha}` | HEAD's commit, or `(initial)` | hashed for a mark |
| `# branch.head ${branch}` | the branch, or `(detached)` | hashed for a mark |
| `# branch.upstream ${remote}/${branch}` | the upstream, when one is set | the row's upstream, and its link to the branches stage |
| `# branch.ab +${ahead} -${behind}` | how far the branch is from its upstream | the row's track, `[ahead, behind]` |
| `1 ${xy} …` | an ordinary change, staged or not | counted as a change |
| `2 ${xy} …` | a rename or a copy | counted as a change |
| `u ${xy} …` | an unmerged path | counted as a change |
| `? ${path}` | an untracked path | counted as untracked |
| `! ${path}` | an ignored path, shown only with `--ignored` | not asked for |

### git status --porcelain

`${xy} ${path}`, one line per path, where `${xy}` is the index's and the working tree's status. With `-z` each record
ends in a NUL and a rename's or a copy's old name is the record after it.
[git's short format](https://git-scm.com/docs/git-status#_short_format) and
[porcelain format version 1](https://git-scm.com/docs/git-status#_porcelain_format_version_1).

| Field | What it holds | What kit does with it |
| --- | --- | --- |
| the output as a whole | nothing for a clean checkout | `git:clean` is true when it is empty: `kit tidy`, `kit rebase`, `kit work` |
| `${xy}`, characters 1 and 2 | the status in the index and in the working tree, `??` for untracked | `flow::checkout` keeps it as a change's status; `commit:choices` names it untracked, modified, deleted, added, staged, renamed, or changed, and marks a path staged already as active |
| `${path}`, from character 4 | the path, root-relative | a change's path; with `-z`, `commit:choices` skips the old name after a rename or a copy |

### git for-each-ref --format

One line per ref, in the format's fields. `kit workspace` separates its fields with `%1f`, the unit separator, which
no ref name and no subject holds; the other formats separate them with a space.
[git's field names](https://git-scm.com/docs/git-for-each-ref#_field_names).

| Field | What it holds | What kit does with it |
| --- | --- | --- |
| `%(refname)` | the full ref name | `workspace::branches` tells `refs/heads/` from `refs/remotes/`, and a remote's name by the longest that starts the ref; `flow:refs` diffs it before and after a fetch; the marks hash it |
| `%(refname:short)` | the name without `refs/heads/` or `refs/tags/` | `git:gone-branches`, `workspace:commits`, and `across:branches` take the branch; `workspace::tags` the tag |
| `%(refname:lstrip=2)` | the name past `refs/heads/` or `refs/tags/` | `issues:branches` takes the local branches; `forge:tags` the tags, as one JSON array |
| `%(refname:lstrip=3)` | the name past `refs/remotes/${primary}/` | `issues:documents` takes the primary's branches, `HEAD` left out |
| `%(symref)` | the ref a symbolic ref names, empty for any other | a remote's `HEAD` is left out of its branches, which are counted by their empty `symref` |
| `%(objectname)` | the object the ref names | `flow:fetch-diff` tells a ref `new`, `moved`, or `gone` between two `flow:refs`; the marks hash it |
| `%(objectname:short)` | the same, abbreviated | a branch's commit; a lightweight tag's commit |
| `%(*objectname:short)` | an annotated tag's commit, empty for any other ref | an annotated tag's commit |
| `%(objecttype)` | `tag` for an annotated tag, `commit` for a lightweight one | the tag's kind, annotated or lightweight |
| `%(upstream)` | the upstream's full ref | hashed for the branches mark |
| `%(upstream:remotename)` | the upstream's remote | joins a local branch to the remote branch it pulls from |
| `%(upstream:remoteref)` | the upstream's ref on the remote, `refs/heads/${branch}` | as `%(upstream:remotename)`; a local branch with an upstream no remote-tracking branch matches is gone |
| `%(upstream:track)` | `[ahead ${n}, behind ${m}]`, `[ahead ${n}]`, `[behind ${m}]`, `[gone]`, or nothing | `git:gone-branches` takes `[gone]`; `workspace::branches` reads `[ahead, behind]` |
| `%(worktreepath)` | the worktree a branch is checked out in, or nothing | the branch's worktrees, and its link to the worktrees stage |
| `%(committerdate:unix)` | the commit's time, in seconds since 1970 | the branch's age |
| `%(creatordate:unix)` | the tag's or the commit's time | the tag's age; `--sort=-creatordate` puts the newest first |
| `%(taggername)` | an annotated tag's tagger, empty for a lightweight one | the tag's tagger |
| `%(contents:subject)` | the subject of the commit or the tag | the row's subject |

### git rev-list --left-right --count

`git rev-list --left-right --count ${base}...HEAD` prints `${left}\t${right}`: `${left}` the commits `${base}` has
and HEAD does not, `${right}` the commits HEAD has and `${base}` does not. `git rev-list --count` prints one number,
and `git rev-list` alone one commit a line. [git rev-list](https://git-scm.com/docs/git-rev-list), whose
`--left-right` and `--count` say this.

| Field | What it holds | What kit does with it |
| --- | --- | --- |
| `${left}` | the commits the default branch has past the fork | `workspace::worktrees`'s base, its second value: behind |
| `${right}` | the commits HEAD has past the fork | the base's first value: ahead |
| the count, alone | the commits a range or a `--not` list leaves | ahead and behind in the desk and worktrees stages; `git:unpushed`; what `kit submit` and a tag push send |
| one commit a line | the commits a range holds | `workspace:commits` maps each to its branches and marks those no remote has; `issue:start-commit` adds the start commit when there is none |

### git ls-remote

`${sha}\t${ref}`, one line per ref the remote has. With `--tags`, an annotated tag has a second line,
`${ref}^{}`, for the commit it names. [git ls-remote](https://git-scm.com/docs/git-ls-remote).

| Field | What it holds | What kit does with it |
| --- | --- | --- |
| `${sha}` | the object the remote's ref names | hashed for the incoming mark |
| `${ref}` | the remote's ref, `refs/heads/${branch}` or `refs/tags/${tag}` | `releases:on-origin` takes each `release/${version}`, `^{}` taken off, and keeps the valid ones; `issue:branch` takes the first branch named `${key}-*`; `releases:release` refuses a tag the remote has |

### git log --format

One line per commit, in the format's placeholders. `%x1f` is the unit separator and `%x09` a tab, so a subject with
spaces stays whole. [git's pretty formats](https://git-scm.com/docs/pretty-formats).

| Field | What it holds | What kit does with it |
| --- | --- | --- |
| `%H` | the full commit hash | `facts:document`'s `commits[].sha`; `workspace::commits`'s key; the last commit that changed a path |
| `%h` | the abbreviated hash | the commit the incoming, desk, and worktrees stages show, and `kit workspace`'s commit |
| `%ct` | the committer's time, in seconds since 1970 | the commit's age |
| `%an` | the author's name | the commit's author |
| `%s` | the subject | `kit policy` judges it; `kit commit` and `kit release` show it; `kit work` tells its own stash by it; `kit workspace` reads the issue a subject starts with, as `#7` |
| `%gd` | a stash's reflog selector, `stash@{${n}}` | `git stash list --format='%gd %H'` gives the slot of the stash `kit work` pushed, by its hash, to drop |

### git ls-files --stage and git ls-tree

`git ls-files --stage` prints `${mode} ${object} ${stage}\t${path}` and `git ls-tree` prints
`${mode} ${type} ${object}\t${path}`; with `-z` each record ends in a NUL, so a path with a newline comes through
whole. `facts:tree` reads both for `kit facts`, and `policy:object` and `policy:base-held` read them for `kit policy`.
[git ls-files](https://git-scm.com/docs/git-ls-files#_output) and
[git ls-tree](https://git-scm.com/docs/git-ls-tree#_output_format).

| Field | What it holds | What kit does with it |
| --- | --- | --- |
| `${mode}` | `100644`, `100755`, `120000` for a link, `160000` for a submodule | a fact's `type`, file, link, or submodule, and `executable` for `100755`; `policy:base-held` reads files and links alone |
| `${type}` | `blob`, `tree`, or `commit`, in `ls-tree` alone | skipped |
| `${object}` | the blob, or the commit a submodule is at | a fact's `blob`; `cat-file` reads its size and its text |
| `${stage}` | `0`, or `1` to `3` for a conflict, in `ls-files` alone | kit keeps stage `0` |
| `${path}` | the path, root-relative | the fact's key |

### git diff --numstat

`${added}\t${removed}\t${path}` per path, `-` for each count of a binary file.
[git diff](https://git-scm.com/docs/git-diff), whose `--numstat` says this.

| Field | What it holds | What kit does with it |
| --- | --- | --- |
| `${added}` | the lines added | summed as the checkout's `added`; `-` counts nothing |
| `${removed}` | the lines removed | summed as the checkout's `removed`; `-` counts nothing |
| `${path}` | the path | not read |

### git cat-file

`--batch-check='%(objectsize)'` prints one size a line, in the order the object names came in; `-s` prints one size;
`blob` prints the blob as it is; `-e` exits 0 when the object is here.
[git cat-file's batch output](https://git-scm.com/docs/git-cat-file#_batch_output).

| Field | What it holds | What kit does with it |
| --- | --- | --- |
| `%(objectsize)` | the blob's size in bytes | a fact's `size`, matched to its entry by order |
| the blob | the file as the tree holds it | a fact's `text` when it has no NUL and fits in 65536 bytes; the `kit.yml` and the policy `kit policy` reads from the base |
| the exit status of `-e` | the commit is here | `git:unpushed` leaves out what the change's head reaches only when this clone has it |

### git check-attr

`${path}: ${attribute}: ${info}`, where `${info}` is `set`, `unset`, `unspecified`, or the value.
[git check-attr](https://git-scm.com/docs/git-check-attr#_output).

| Field | What it holds | What kit does with it |
| --- | --- | --- |
| `${info}` of `linguist-generated` | whether `.gitattributes` marks the path generated | `template:exempt` leaves a path marked `set` out of the templates' shape `kit lint` checks |

### NUL-separated path lists

`git ls-files -z` and `git diff --cached --name-only -z` print each path, root-relative, followed by a NUL, so a path
with a space, an accent, or a newline survives. `list:read0` reads them into an array.
[git ls-files](https://git-scm.com/docs/git-ls-files) and [git diff](https://git-scm.com/docs/git-diff).

| Field | What it holds | What kit does with it |
| --- | --- | --- |
| a path | a tracked or staged path; `--diff-filter=ACMR` keeps the added, copied, modified, and renamed, `ACMRD` the deleted too | the paths `kit lint` lints, `kit generate` reads, and `kit commit` checks; a staged path missing from the disk is refused |

### Single values

Each of these prints one value on one line, or nothing and a status of 1, which kit reads as the answer.
[git rev-parse](https://git-scm.com/docs/git-rev-parse),
[git symbolic-ref](https://git-scm.com/docs/git-symbolic-ref), [git config](https://git-scm.com/docs/git-config),
[git remote](https://git-scm.com/docs/git-remote), and [git merge-base](https://git-scm.com/docs/git-merge-base).

| Field | What it holds | What kit does with it |
| --- | --- | --- |
| `rev-parse --show-toplevel` | the checkout's root, as a physical path | every verb runs from it; a path kit names is relative to it |
| `rev-parse --git-dir`, `--git-common-dir` | the checkout's git directory, and the one every worktree shares | the main checkout is the common one's parent; they differ in a linked worktree; kit keeps its logs, caches, and choices under `${common}/kit/` |
| `rev-parse --git-path hooks/pre-commit` | the pre-commit hook's path | read for `lefthook` |
| `rev-parse --verify --quiet ${rev}` | the commit's hash, or nothing | whether a ref, a tag, or a stash exists, and HEAD's commit before and after the checks of `kit commit` |
| `symbolic-ref --quiet [--short] ${ref}` | the branch a symbolic ref names, or nothing | the branch checked out, nothing when detached; a remote's default branch from `refs/remotes/${remote}/HEAD` |
| `branch --show-current` | the branch checked out, or nothing | `kit welcome` names it |
| `config --get ${key}` | the value, or nothing and status 1 | a branch's upstream remote and ref; `user.signingkey` |
| `config --type=bool commit.gpgsign` | `true` or `false` | `kit commit` warns on anything but `true` |
| `remote` | each remote's name, one a line | the remotes, the primary first; `primary` in `kit.yml` must be one |
| `remote get-url [--push] ${remote}` | the remote's fetch or push URL | its host picks the forge, its path names the repository, and its sum names the forge's snapshots |
| `merge-base ${a} ${b}` | the best common ancestor | the fork the desk and worktrees stages count from |
| `hash-object --stdin` | the hash of what came in | a mark of `kit workspace` |

## Schemas

git publishes no JSON Schema and prints no JSON. Its formats for scripts are documented in prose, in the pages
linked above, and its manual promises that a porcelain format does not change in a way that breaks a script.

kit turns git's output into JSON in several places, and none of them is git's format alone: `workspace.jq` makes the
rows of `kit workspace` from several git commands joined with each remote's forge and links, `flow::checkout` makes
the desk and worktrees stages' documents from `git status --porcelain`, `git diff --numstat`, and `git log`,
`facts:tree` makes the `tree` of `kit facts` from `git ls-files` or `git ls-tree` with each path's kinds, and
`forge:tags` and `issues:checkouts` make a list and a map. A schema of any of them describes a verb's output, not
git's, so this page writes none; [../workspaces.md](../workspaces.md) says what `kit workspace --json` prints, and
[../policy.md](../policy.md) what the facts hold.
