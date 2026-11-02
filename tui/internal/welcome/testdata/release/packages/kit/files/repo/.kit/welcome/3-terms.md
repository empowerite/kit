# Terms

- **desk**: the root checkout, where people read and review. `kit status desk`
- **worktree**: a checkout of its own for one issue, where the work happens. `kit status worktrees`
- **forge**: where a remote lives, GitHub or GitLab, with its command-line tool. `kit status incoming`
- **remote** and **primary**: a remote is a git remote; the primary is the one issues and pull requests live on.
- **package**: a set of rules, styles, and templates a repository takes. `kit packages`
- **kind**: a sort of file kit knows, by its name. `kit kinds`
- **rule**: what a kind of file must hold, checked on commit. `kit context <path>`
- **style**: how a kind of file is written, for people and agents. `kit context <path>`
- **template**: the start of a new file of a kind. `kit new`
- **region**: a part of a file kit keeps, between markers. `kit region`
- **gate**: a check the build runs before a change may land. `kit gate`
- **stage**: one part of the picture `kit status` draws. `kit status --stages`
