# kit.rego - the kit's naming policy in Rego: the patterns engine's rules, word for word
# ----------------------------------------------------------------------------------------------------------------------
# Delivered by kit, not edited by hand. A change is a new release, then kit update. A repository's own rules go in
# a file of its own beside this one, in the same package, adding to deny; `kit policy --input` prints the input.

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
