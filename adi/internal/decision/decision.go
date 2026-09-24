// Package decision reads the decision model: one MADR file per decision, named
// NNNN-title-with-dashes.md, and the ADE rule file beside it.
package decision

import (
	"slices"
	"strings"
)

// Decision is one file in the model. Body is everything after the frontmatter,
// heading included, kept verbatim -- what a decision says is its own prose, and
// this package does not interpret its sections.
type Decision struct {
	// ID is the four digits MADR puts at the front of the filename. Surfaces
	// print it as ADR-NNNN, the form MADR itself uses to reference a decision.
	ID     string
	Title  string
	Status string
	// Tags say which repositories the decision bears on: those that declare at
	// least one of them. What a tag stands for is the model's to define.
	Tags []string
	Body string
	// Path is relative to the model directory, so it reads the same to a client
	// that has the repository checked out as it does to the server.
	Path string
	// Rule is the ADE rule file sitting beside the record, when there is one:
	// the part of the decision ADE's DSL can state. Prose has to name the
	// repository it speaks about; a rule speaks about whichever one it is read
	// in, so it survives the trip to a consumer unchanged.
	Rule     string
	RulePath string
}

// Binds reports whether the decision binds the work. Only an accepted one
// does: a proposed decision is still being argued, and the other statuses say
// where the decision went.
func (d Decision) Binds() bool {
	return strings.EqualFold(strings.TrimSpace(d.Status), "accepted")
}

// AppliesTo reports whether the decision bears on a repository that declares
// the given tags. A decision without tags bears on every repository, so no
// declaration leaves it out.
func (d Decision) AppliesTo(tags []string) bool {
	if len(d.Tags) == 0 {
		return true
	}
	for _, tag := range tags {
		if slices.Contains(d.Tags, normalizeTag(tag)) {
			return true
		}
	}
	return false
}
