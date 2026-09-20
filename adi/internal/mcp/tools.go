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
		Name:        "list_decisions",
		Description: "List every architectural decision in the model as its id, title and status. Call this at the start of a session: it is small enough to read whole, and it is what says which decisions exist to ask about. Read any of them in full with get_decision.",
	}, s.listDecisions)
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "get_decision",
		Description: "Read one decision in full, addressed by the id list_decisions gave. Returns what was decided together with the reasoning recorded alongside it. Check the status before following it: a decision that is still open binds nothing.",
	}, s.getDecision)
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
