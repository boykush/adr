package mcp

import (
	"context"

	"github.com/boykush/adr/adi/internal/decision"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerTools wires the read surface. There is no writing tool and no tool
// that narrows the list for the caller: an agent judges relevance from a title,
// and the server holds no opinion about which paths a decision touches.
func (s *Server) registerTools(srv *mcpsdk.Server) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "list_rules",
		Description: "List every rule that binds this session, in full: the rule files of the accepted decisions, written in ADE's rule DSL. A rule names paths and assertions about whichever repository it is read in, so it applies as written to the one you are working in. Nothing runs the rules: check the work against them yourself. Call this at the start of a session. A rule holds only the part of its decision that ADE's DSL can state.",
	}, s.listRules)
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "list_decisions",
		Description: "List every architectural decision in the model as its id, title and status. Call this at the start of a session too: a decision can bind the work beyond what its rule states, or have no rule at all. Read any of them in full with get_decision.",
	}, s.listDecisions)
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "get_decision",
		Description: "Read one decision in full, addressed by the id list_rules or list_decisions gave: what was decided, the options weighed and the reasoning recorded alongside. Check the status before following it: a decision that is still proposed binds nothing.",
	}, s.getDecision)
}

func (s *Server) listRules(_ context.Context, _ *mcpsdk.CallToolRequest, _ listRulesInput) (*mcpsdk.CallToolResult, listRulesOutput, error) {
	decisions, err := decision.Load(s.cfg.ModelDir)
	if err != nil {
		return nil, listRulesOutput{}, err
	}
	rules := make([]ruleJSON, 0, len(decisions))
	for _, d := range decisions {
		// Beside any decision but an accepted one, a rule is a draft or a
		// leftover, and a session handed it would obey it all the same.
		if d.Binds() && d.RulePath != "" {
			rules = append(rules, toRule(d))
		}
	}
	return nil, listRulesOutput{Rules: rules}, nil
}

func (s *Server) listDecisions(_ context.Context, _ *mcpsdk.CallToolRequest, _ listDecisionsInput) (*mcpsdk.CallToolResult, listDecisionsOutput, error) {
	decisions, err := decision.Load(s.cfg.ModelDir)
	if err != nil {
		return nil, listDecisionsOutput{}, err
	}
	summaries := make([]decisionSummaryJSON, 0, len(decisions))
	for _, d := range decisions {
		summaries = append(summaries, toSummary(d))
	}
	return nil, listDecisionsOutput{Decisions: summaries}, nil
}

func (s *Server) getDecision(_ context.Context, _ *mcpsdk.CallToolRequest, in getDecisionInput) (*mcpsdk.CallToolResult, getDecisionOutput, error) {
	d, err := decision.Find(s.cfg.ModelDir, in.ADRID)
	if err != nil {
		return nil, getDecisionOutput{}, err
	}
	return nil, getDecisionOutput{
		ADRID:  "ADR-" + d.ID,
		Title:  d.Title,
		Status: d.Status,
		Path:   d.Path,
		Body:   d.Body,
	}, nil
}
