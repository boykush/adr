// Package decision reads the decision model: one MADR file per decision, named
// NNNN-title-with-dashes.md, and the ADE rule file beside it.
package decision

import "strings"

// Decision is one file in the model. Body is everything after the frontmatter,
// heading included, kept verbatim -- what a decision says is its own prose, and
// this package does not interpret its sections.
type Decision struct {
	// ID is the four digits MADR puts at the front of the filename. Surfaces
	// print it as ADR-NNNN, the form MADR itself uses to reference a decision.
	ID     string
	Title  string
	Status string
	Body   string
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
