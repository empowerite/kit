# The policy/opa package

Delivers `.kit/policy/kit.rego`, the kit's naming policy in Rego, in package `kit.policy`. When a `.rego` file is
present under `.kit/policy/`, `kit policy` evaluates the policy with Open Policy Agent instead of the default
patterns engine. A `policy_engine` setting in `kit.yml` overrides that choice.

The rules in `kit.rego` refuse exactly what the patterns engine refuses. Taking the package does not change anything
until the repository adds a rule of its own. Such a rule is Rego in package `kit.policy` that adds to `deny` or
`warn`. It ships through a local package; see [a rule of your own, through a local
package](../../policy.md#a-rule-of-your-own-through-a-local-package).

`kit facts` prints the document a rule reads. The repository pins `opa` in its toolchain, and `kit tools install`
installs it.
