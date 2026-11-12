# kit:generated -- this repository's own tree rules in Rego, from the house package
# see: docs/packages.md, .kit/packages/house/package.yml
# Five rules over the facts tree: kit/bin/ holds executable files named kit or kit-*, with no subfolders; kit/ holds
# only bin/, modules/, and packages/; and README.md exists and is not a starter template.

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

deny contains sprintf("%s: a file in kit/bin/ is named kit or kit-*", [path]) if {
	some path, _ in input.tree
	not regex.match(`^kit(-[a-z0-9-]+)?$`, bin_name(path))
}

deny contains sprintf("%s: kit/bin/ has no subfolders", [path]) if {
	some path, _ in input.tree
	startswith(path, bin)
	contains(substring(path, count(bin), -1), "/")
}

kit_dirs := {"bin", "libexec", "modules", "packages"}

deny contains sprintf("kit/%s: kit/ holds only bin/, libexec/, modules/, and packages/", [part]) if {
	some path, _ in input.tree
	parts := split(path, "/")
	parts[0] == "kit"
	part := parts[1]
	not part in kit_dirs
}

deny contains "README.md is missing" if not input.tree["README.md"]

# template(text) is true when README.md is still a starter: GitHub's, GitLab's, or kit's own.
template(text) if count([l | some l in split(text, "\n"); trim_space(l) != ""]) <= 2  # GitHub
template(text) if contains(text, "## Suggestions for a good README")                    # GitLab
template(text) if contains(text, "«")                                                    # kit

deny contains "README.md is still a template" if {
	template(input.tree["README.md"].text)
}
