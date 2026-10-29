# kit.rego - the kit's naming policy in Rego: the same rules as the patterns engine
# ----------------------------------------------------------------------------------------------------------------------
# Added by kit's policy-opa package. To remove it, run kit package remove policy-opa.
# To change it, do not edit it here: see "Changing what kit adds" in .kit/1st.md.
# Put the repository's own rules in a separate file beside this one, in the same package, adding to deny.
# `kit policy --input` prints the input.

package kit.policy

import rego.v1

deny contains msg if {
	input.branch != null
	pattern := input.policy.branch
	pattern != ""
	not regex.match(pattern, input.branch)
	msg := sprintf("branch '%s' does not match policy.branch: %s", [input.branch, pattern])
}

deny contains msg if {
	pattern := input.policy.commit_subject
	pattern != ""
	some commit in input.commits
	not regex.match(pattern, commit.subject)
	msg := sprintf("commit %s '%s' does not match policy.commit_subject: %s",
		[substring(commit.sha, 0, 7), commit.subject, pattern])
}

deny contains msg if {
	input.change != null
	pattern := input.policy.change_title
	pattern != ""
	not regex.match(pattern, input.change.title)
	msg := sprintf("change title '%s' does not match policy.change_title: %s", [input.change.title, pattern])
}
