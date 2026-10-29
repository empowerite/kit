# The policy/opa package

Delivers `.kit/policy/kit.rego`, the kit's naming policy in Rego, in package `kit.policy`. When a `.rego` file is
present under `.kit/policy/`, `kit policy` evaluates the policy with Open Policy Agent instead of the default
patterns engine.

The rules in `kit.rego` refuse exactly what the patterns engine refuses. Taking the package changes nothing until
the repository adds a rule of its own. To add one, put a `.rego` file of the repository's own beside `kit.rego`,
in the same package, that adds to `deny`.

`kit policy --input` prints the input document a rule reads. The repository pins `opa` in its toolchain.
