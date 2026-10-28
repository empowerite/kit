# The policy-opa module

Places `.kit/policy/kit.rego`, the kit's naming policy in Rego, package `kit.policy`. With a `.rego` file under
`.kit/policy/`, `kit policy` evaluates the policy with Open Policy Agent instead of the default patterns engine.
The rules here are the patterns engine's, refusal for refusal, so taking the module changes nothing until the
repository adds a rule of its own: a `.rego` file of its own beside this one, in the same package, adding to `deny`.
`kit policy --input` prints the input document a rule reads. The repository pins `opa` in its toolchain.
