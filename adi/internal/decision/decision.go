// Package decision reads the decision model: one Markdown file per decision,
// with YAML frontmatter carrying the fields the surfaces address it by.
package decision

// Decision is one file in the model. Body is everything after the frontmatter,
// kept verbatim -- what a decision says is its own prose, and this package does
// not interpret its sections.
type Decision struct {
	ID     string
	Title  string
	Status string
	Body   string
	// Path is relative to the model directory, so it reads the same to a client
	// that has the repository checked out as it does to the server.
	Path string
}
