package mcp

import "github.com/boykush/adr/adi/internal/decision"

// --- list_decisions ---

type listDecisionsInput struct{}

type listDecisionsOutput struct {
	Decisions []decisionSummaryJSON `json:"decisions"`
}

// decisionSummaryJSON is one row of the listing: enough to judge whether a
// decision bears on the work, and the id to read it by. The body is left out
// so the listing stays small enough to read at the start of every session.
type decisionSummaryJSON struct {
	ADRID  string `json:"adr_id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	// Path is relative to the model directory, so it reads the same to a client
	// with the repository checked out as it does here.
	Path string `json:"path"`
}

// --- get_decision ---

type getDecisionInput struct {
	ADRID string `json:"adr_id" jsonschema:"The decision's id, written either \"0001\" or \"AD0001\"."`
}

type getDecisionOutput struct {
	ADRID  string `json:"adr_id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Path   string `json:"path"`
	// Body is the decision's Markdown as written, sections and all.
	Body string `json:"body"`
}

func toSummary(d decision.Decision) decisionSummaryJSON {
	return decisionSummaryJSON{ADRID: "AD" + d.ID, Title: d.Title, Status: d.Status, Path: d.Path}
}
