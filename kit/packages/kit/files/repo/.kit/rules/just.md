---
# kit:generated -- the just rule, from the kit package
# see: "Changing what kit adds" in .kit/1st.md, to replace it
kind:
  - just
---
# Just

Applies to `justfile` and `*.just`. A `[script]` recipe body is bash, so the bash rule also applies to it.

1. A recipe runs the repository's pinned tools, the same ones its CI runs, never another copy that happens to be
   first on `PATH`.
2. A `[script]` body sets `errexit`, `nounset`, and `pipefail` first, so a failure stops the recipe.
3. Use no `{{...}}` interpolation in a body. A value interpolated into shell breaks on a space or a quote, and can
   run as a command. Pass a value that `just` computes as an exported variable, `export <name> := <expression>`,
   and refer to it in the body only as `${name}`. `set export := true` exports every parameter.
4. A recipe that changes a remote or a worktree names what it is about to do before doing it.
5. A bash recipe that needs the kit's modules loads the kit's loader through `BASH_ENV`, exported once at the top of
   the justfile: ``export BASH_ENV := `kit root` / "modules/bash/1st.sh"``. It then calls `use <module>` and the
   output verbs (`info`, `warn`, `error`) as a verb does, finds the repository's own modules on the `bash_path`
   setting, and crosses a process boundary with `run kit <verb>` or `run just <recipe>`, so the trace depth carries
   into a nested call. The loader sets no shell option, so the recipe's own stand.
