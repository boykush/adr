package mcp

import (
	"context"
	"os"
	"strings"

	"github.com/boykush/adr/adi/internal/decision"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// tagsHeader and tagsEnv carry the tags a repository declares, comma-separated.
// One HTTP server answers every repository, so each request brings its own in
// a header; a stdio server belongs to one repository and reads its environment.
const (
	tagsHeader = "Adi-Tags"
	tagsEnv    = "ADI_TAGS"
)

// registerTools wires the read surface. There is no writing tool. Only the
// decision listing narrows, by the tags the repository declares: a rule already
// confines itself to the paths it names, and a decision asked for by id is one
// the session wants whatever its tags.
func (s *Server) registerTools(srv *mcpsdk.Server) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "list_rules",
		Description: "List every rule that binds this session, in full: the rule files of the accepted decisions, written in ADE's rule DSL. The work is held to these. A rule names paths and assertions about whichever repository it is read in, so it applies as written to the one you are working in. Nothing runs the rules: they are checked by reading them against the change. Not at the start of a session: where the repository runs ai-review (.github/workflows/ai-review.yml), call this when its review asks for changes; elsewhere, once the work is done, before it is pushed.",
	}, s.listRules)
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "list_decisions",
		Description: "List the architectural decisions in the model as their id, title, status and tags, to find the one behind a rule. The work is held to the rules, not to the decisions, so call this only when you need a rule's reasons. A decision without tags bears on every repository; one with tags bears on the repositories that declare one of them, and when the repository you work in declares tags, the listing leaves the others out. Read any of them in full with get_decision.",
	}, s.listDecisions)
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "get_decision",
		Description: "Read one decision in full, addressed by the id list_rules or list_decisions gave: what was decided, the options weighed and the reasoning recorded alongside. Read it when you need to know why a rule exists. Check its status: only an accepted decision's rule binds.",
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

func (s *Server) listDecisions(_ context.Context, req *mcpsdk.CallToolRequest, _ listDecisionsInput) (*mcpsdk.CallToolResult, listDecisionsOutput, error) {
	decisions, err := decision.Load(s.cfg.ModelDir)
	if err != nil {
		return nil, listDecisionsOutput{}, err
	}
	tags := declaredTags(req)
	summaries := make([]decisionSummaryJSON, 0, len(decisions))
	for _, d := range decisions {
		// A repository that declares nothing has given no ground to leave a
		// decision out on.
		if len(tags) > 0 && !d.AppliesTo(tags) {
			continue
		}
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
		Tags:   d.Tags,
		Path:   d.Path,
		Body:   d.Body,
	}, nil
}

// declaredTags returns the tags the repository behind req declares. Over HTTP
// only the request's header counts: the server's own environment describes no
// repository of the many it answers.
func declaredTags(req *mcpsdk.CallToolRequest) []string {
	if req != nil && req.Extra != nil && req.Extra.Header != nil {
		return splitTags(req.Extra.Header.Get(tagsHeader))
	}
	return splitTags(os.Getenv(tagsEnv))
}

func splitTags(raw string) []string {
	var tags []string
	for _, tag := range strings.Split(raw, ",") {
		if tag = strings.TrimSpace(tag); tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}
