package mcp

import "github.com/boykush/adr/adi/internal/decision"

// --- list_rules ---

type listRulesInput struct{}

type listRulesOutput struct {
	Rules []ruleJSON `json:"rules"`
}

// ruleJSON is one decision's rule file. It goes out under the decision's id and
// title, which is what a session reads the reasoning by.
type ruleJSON struct {
	ADRID string `json:"adr_id"`
	Title string `json:"title"`
	// Rule is the file as written, in ADE's rule DSL.
	Rule     string `json:"rule"`
	RulePath string `json:"rule_path"`
}

// --- list_decisions ---

type listDecisionsInput struct{}

type listDecisionsOutput struct {
	Decisions []decisionSummaryJSON `json:"decisions"`
}

// decisionSummaryJSON is one row of the listing: enough to judge whether a
// decision bears on the work, and the id to read it by. The body is left out
// so the listing stays small enough to read before every push.
type decisionSummaryJSON struct {
	ADRID  string   `json:"adr_id"`
	Title  string   `json:"title"`
	Status string   `json:"status"`
	Tags   []string `json:"tags,omitempty"`
	// Path is relative to the model directory, so it reads the same to a client
	// with the repository checked out as it does here.
	Path string `json:"path"`
}

// --- get_decision ---

type getDecisionInput struct {
	ADRID string `json:"adr_id" jsonschema:"The decision's id, written \"0001\" or \"ADR-0001\"."`
}

type getDecisionOutput struct {
	ADRID  string   `json:"adr_id"`
	Title  string   `json:"title"`
	Status string   `json:"status"`
	Tags   []string `json:"tags,omitempty"`
	Path   string   `json:"path"`
	// Body is the decision's Markdown as written, sections and all.
	Body string `json:"body"`
}

func toRule(d decision.Decision) ruleJSON {
	return ruleJSON{ADRID: "ADR-" + d.ID, Title: d.Title, Rule: d.Rule, RulePath: d.RulePath}
}

func toSummary(d decision.Decision) decisionSummaryJSON {
	return decisionSummaryJSON{ADRID: "ADR-" + d.ID, Title: d.Title, Status: d.Status, Tags: d.Tags, Path: d.Path}
}
