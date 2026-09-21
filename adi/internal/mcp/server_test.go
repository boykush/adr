package mcp

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"0001-first.md":    "---\nstatus: accepted\n---\n\n# First\n\n## Context and Problem Statement\n\nthe first one\n",
		"0001-first.rule":  "adr \"0001\" \"First\"\n\nfile \"x\" {\n  severity error\n}\n",
		"0002-second.md":   "---\nstatus: proposed\n---\n\n# Second\n\n## Context and Problem Statement\n\nstill arguing\n",
		"0002-second.rule": "adr \"0002\" \"Second\"\n\nfile \"y\" {\n  severity error\n}\n",
		"0003-third.md":    "---\nstatus: accepted\n---\n\n# Third\n\n## Context and Problem Statement\n\nno rule of its own\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return NewServer(Config{ModelDir: dir}, "test")
}

func connect(t *testing.T, s *Server) *mcpsdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	serverTransport, clientTransport := mcpsdk.NewInMemoryTransports()
	serverSession, err := s.mcpServer().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { serverSession.Close() })

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func contentText(res *mcpsdk.CallToolResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(*mcpsdk.TextContent); ok {
		return tc.Text
	}
	return ""
}

// TestEndToEnd drives the server through a real MCP session: the handshake, the
// three tools, and reading one decision by each spelling of its id.
func TestEndToEnd(t *testing.T) {
	ctx := context.Background()
	cs := connect(t, newTestServer(t))

	// The handshake is where the agent learns to follow the rules and where
	// their grammar is documented. The wording is free to change; that it names
	// the tool and the skill is not.
	if got := cs.InitializeResult().Instructions; !strings.Contains(got, "list_rules") || !strings.Contains(got, "ade-rule-dsl") {
		t.Errorf("instructions = %q, want them to name list_rules and the ade-rule-dsl skill", got)
	}

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	// Exactly three, all reads: the rules to follow and the decisions behind
	// them. A fourth tool would be a change of shape rather than an addition.
	if want := []string{"get_decision", "list_decisions", "list_rules"}; !slices.Equal(names, want) {
		t.Fatalf("tools = %v, want exactly %v", names, want)
	}

	res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "list_rules", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("call list_rules: %v", err)
	}
	var rules listRulesOutput
	if err := json.Unmarshal([]byte(contentText(res)), &rules); err != nil {
		t.Fatalf("decode list_rules: %v", err)
	}
	// Only the accepted decision's rule: the proposed one's is a draft, and the
	// other accepted decision has no rule.
	if len(rules.Rules) != 1 {
		t.Fatalf("rules = %+v, want ADR-0001's alone", rules.Rules)
	}
	rule := rules.Rules[0]
	if rule.ADRID != "ADR-0001" || rule.Title != "First" || rule.RulePath != "0001-first.rule" || !strings.Contains(rule.Rule, "adr \"0001\"") {
		t.Errorf("rule = %+v", rule)
	}

	res, err = cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "list_decisions", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("call list_decisions: %v", err)
	}
	var list listDecisionsOutput
	if err := json.Unmarshal([]byte(contentText(res)), &list); err != nil {
		t.Fatalf("decode list_decisions: %v", err)
	}
	if len(list.Decisions) != 3 {
		t.Fatalf("decisions = %+v, want 3", list.Decisions)
	}
	if list.Decisions[0].ADRID != "ADR-0001" || list.Decisions[0].Status != "accepted" {
		t.Errorf("first = %+v", list.Decisions[0])
	}
	// The listing carries status so an agent can tell a decision that binds from
	// one still being argued, without reading either in full.
	if list.Decisions[1].Status != "proposed" {
		t.Errorf("second status = %q, want proposed", list.Decisions[1].Status)
	}

	for _, id := range []string{"1", "0001", "ADR-0001"} {
		res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "get_decision", Arguments: map[string]any{"adr_id": id}})
		if err != nil {
			t.Fatalf("call get_decision(%q): %v", id, err)
		}
		var got getDecisionOutput
		if err := json.Unmarshal([]byte(contentText(res)), &got); err != nil {
			t.Fatalf("decode get_decision(%q): %v", id, err)
		}
		if got.ADRID != "ADR-0001" || !strings.Contains(got.Body, "the first one") {
			t.Errorf("get_decision(%q) = %+v", id, got)
		}
		// The decision comes without its rule: handing rules out is list_rules'
		// job, so there is one place a session learns what binds it.
		if text := contentText(res); strings.Contains(text, "severity error") {
			t.Errorf("get_decision(%q) carries the rule: %s", id, text)
		}
	}

	res, err = cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "get_decision", Arguments: map[string]any{"adr_id": "99"}})
	if err != nil {
		t.Fatalf("call get_decision(99): %v", err)
	}
	if !res.IsError {
		t.Error("get_decision of an absent id succeeded")
	}
}

// TestHTTPTransport drives the same surface over Streamable HTTP, the way the
// deployed server is reached. DisableStandaloneSSE mirrors a request/response
// client, since the stateless server pushes nothing.
func TestHTTPTransport(t *testing.T) {
	ctx := context.Background()
	s := newTestServer(t)

	httpServer := httptest.NewServer(s.httpHandler())
	defer httpServer.Close()

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, &mcpsdk.StreamableClientTransport{
		Endpoint:             httpServer.URL + httpPath,
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "list_decisions", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("call list_decisions: %v", err)
	}
	var list listDecisionsOutput
	if err := json.Unmarshal([]byte(contentText(res)), &list); err != nil {
		t.Fatalf("decode list_decisions: %v", err)
	}
	if len(list.Decisions) != 3 {
		t.Fatalf("decisions = %+v, want 3", list.Decisions)
	}
}
