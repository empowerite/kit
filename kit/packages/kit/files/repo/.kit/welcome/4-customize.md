<!-- kit:generated -- kit welcome's fourth step, customize, from the kit package -->
<!-- see: "Changing what kit adds" in .kit/1st.md, to replace it -->
# Customize — kit is a starting point, and these are the parts teams change first

Each part ends with where to change it, the file first, then the doc that explains it, and with the command that
shows it.

## Packages

`packages:` in `kit.yml` lists the active packages. Change the list with `kit packages add` and
`kit packages remove`, never by hand. Each package delivers its files, and takes them back out when it goes.

→ [kit.yml](kit.yml) · [docs/packages.md](https://github.com/empowerite/kit/blob/main/docs/packages.md)

`kit packages`

## Policy

`policy:` in `kit.yml` sets the patterns that branch names, worktrees, commit subjects, and pull request titles must
match, each with a remedy a refusal names. `kit work` makes names that pass, and `kit commit` and CI hold names to it.

→ [kit.yml](kit.yml) · [docs/policy.md](https://github.com/empowerite/kit/blob/main/docs/policy.md)

`kit policy`

## The stages of kit flow

`flow:` in `kit.yml` names the stages `kit flow` draws, in order. Leave out the ones your team does not use.

→ [kit.yml](kit.yml) · [docs/features.md](https://github.com/empowerite/kit/blob/main/docs/features.md)

`kit flow --stages`

## Primary and forges

`primary:` names the remote kit reads pull requests, builds, and issues from; the default is `origin`. `kit work`
and `kit submit` push to `origin` either way. `forges:` names the code forge of a remote whose host does not say
it, such as a GitLab you host yourself.

→ [kit.yml](kit.yml) · [forges.md](https://github.com/empowerite/kit/blob/main/docs/integrations/forges.md)

`kit flow incoming --no-fetch`

## A house package

A house package holds your team's own rules, styles, and templates. It lives in the repository, under
`.kit/packages/<name>/`, in the same shape as the kit's packages.

- To add a file, place it under the package's `files/repo/`.
- To replace a file kit delivers, depend on the package that delivers it, and place yours at the same path. The
  later package wins.
- These pages are files too, under `.kit/welcome/`. A house package replaces one by its name, or adds one, such as
  `7-our-team.md`.

→ [.kit/packages/](.kit/packages/) · [docs/packages.md](https://github.com/empowerite/kit/blob/main/docs/packages.md)

`kit packages --active`

## The docs

`kit help` lists every command, and `kit help <verb>` prints one command's page. `.kit/README.md` explains the
`.kit/` directory. The kit's own README and its `docs/` cover the rest, at <https://github.com/empowerite/kit>.

→ [.kit/README.md](.kit/README.md) · [README.md](https://github.com/empowerite/kit/blob/main/README.md)

`kit help`
