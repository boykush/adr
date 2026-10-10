package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/caarlos0/env/v11"
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
		"0004-fourth.md":   "---\nstatus: accepted\ntags: [go]\n---\n\n# Fourth\n\n## Context and Problem Statement\n\nfor go\n",
		"0004-fourth.rule": "adr \"0004\" \"Fourth\"\n\nfile \"z\" {\n  severity error\n}\n",
		"0005-fifth.md":    "---\nstatus: accepted\ntags: [product]\n---\n\n# Fifth\n\n## Context and Problem Statement\n\nfor products\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return NewServer(Config{ModelDir: dir}, "test")
}

// declareInEnv has s read tags the way a stdio server does, from ADI_TAGS in
// its environment. A map stands in for that environment, so the one running
// the tests is neither read nor changed.
func declareInEnv(t *testing.T, s *Server, tags string) {
	t.Helper()
	if err := s.readEnv(env.Options{Environment: map[string]string{"ADI_TAGS": tags}}); err != nil {
		t.Fatalf("read the environment: %v", err)
	}
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

// headerTransport stands in for an MCP client configured with headers for this
// server, the way a repository declares its tags.
type headerTransport struct{ header http.Header }

func (h headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	for k, v := range h.header {
		r.Header[k] = v
	}
	return http.DefaultTransport.RoundTrip(r)
}

// connectHTTP reaches s over Streamable HTTP, the way the deployed server is
// reached. DisableStandaloneSSE mirrors a request/response client, since the
// stateless server pushes nothing.
func connectHTTP(t *testing.T, s *Server, header http.Header) *mcpsdk.ClientSession {
	t.Helper()
	httpServer := httptest.NewServer(s.httpHandler())
	t.Cleanup(httpServer.Close)

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(context.Background(), &mcpsdk.StreamableClientTransport{
		Endpoint:             httpServer.URL + httpPath,
		DisableStandaloneSSE: true,
		HTTPClient:           &http.Client{Transport: headerTransport{header}},
	}, nil)
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

func listDecisionIDs(t *testing.T, cs *mcpsdk.ClientSession) []string {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{Name: "list_decisions", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("call list_decisions: %v", err)
	}
	var list listDecisionsOutput
	if err := json.Unmarshal([]byte(contentText(res)), &list); err != nil {
		t.Fatalf("decode list_decisions: %v", err)
	}
	ids := make([]string, 0, len(list.Decisions))
	for _, d := range list.Decisions {
		ids = append(ids, d.ADRID)
	}
	return ids
}

func listRuleIDs(t *testing.T, cs *mcpsdk.ClientSession) []string {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{Name: "list_rules", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("call list_rules: %v", err)
	}
	var rules listRulesOutput
	if err := json.Unmarshal([]byte(contentText(res)), &rules); err != nil {
		t.Fatalf("decode list_rules: %v", err)
	}
	ids := make([]string, 0, len(rules.Rules))
	for _, r := range rules.Rules {
		ids = append(ids, r.ADRID)
	}
	return ids
}

// TestEndToEnd drives the server through a real MCP session: the handshake, the
// three tools, and reading one decision by each spelling of its id.
func TestEndToEnd(t *testing.T) {
	ctx := context.Background()
	cs := connect(t, newTestServer(t))

	// The handshake is where the agent learns when the rules are checked and
	// which skill says how. The wording is free to change; that it names the
	// tool and the skills is not.
	got := cs.InitializeResult().Instructions
	for _, name := range []string{"list_rules", "adr-check", "adr-review", "ade-rule-dsl"} {
		if !strings.Contains(got, name) {
			t.Errorf("instructions = %q, want them to name %s", got, name)
		}
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
	// Only the accepted decision's rule: the proposed one's is a draft, the
	// other untagged accepted decision has no rule, and the tagged one's reaches
	// only a repository declaring its tag.
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
	// A repository that declares nothing is listed the untagged decisions alone.
	if list.Decisions[2].ADRID != "ADR-0003" || list.Decisions[2].Tags != nil {
		t.Errorf("third = %+v, want ADR-0003 without tags", list.Decisions[2])
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
		if got.ADRID != "ADR-0001" || !strings.Contains(got.Body, "the first one") || got.Tags != nil {
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

// TestDeclaredTagsNarrowDecisionsAndRules serves one repository over stdio, as
// a consumer running adi itself would, declaring its tags in the environment.
func TestDeclaredTagsNarrowDecisionsAndRules(t *testing.T) {
	ctx := context.Background()
	s := newTestServer(t)
	declareInEnv(t, s, " Go ")
	cs := connect(t, s)

	// ADR-0005 carries product alone, so it drops out. The untagged decisions
	// bear on every repository, and ADR-0004 is added for declaring go.
	if got, want := listDecisionIDs(t, cs), []string{"ADR-0001", "ADR-0002", "ADR-0003", "ADR-0004"}; !slices.Equal(got, want) {
		t.Errorf("decisions = %v, want %v", got, want)
	}
	if got, want := listRuleIDs(t, cs), []string{"ADR-0001", "ADR-0004"}; !slices.Equal(got, want) {
		t.Errorf("rules = %v, want %v", got, want)
	}

	res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "get_decision", Arguments: map[string]any{"adr_id": "5"}})
	if err != nil {
		t.Fatalf("call get_decision: %v", err)
	}
	if res.IsError {
		t.Errorf("get_decision of a decision left out of the listing failed: %s", contentText(res))
	}
}

// TestHTTPTransport drives the same surface over Streamable HTTP, where each
// request declares its repository's tags in a header.
func TestHTTPTransport(t *testing.T) {
	s := newTestServer(t)
	// The server answers every repository, so its own environment declares
	// nothing on their behalf.
	declareInEnv(t, s, "go")

	undeclared := connectHTTP(t, s, nil)
	if got, want := listDecisionIDs(t, undeclared), []string{"ADR-0001", "ADR-0002", "ADR-0003"}; !slices.Equal(got, want) {
		t.Errorf("declaring nothing: decisions = %v, want %v", got, want)
	}
	if got, want := listRuleIDs(t, undeclared), []string{"ADR-0001"}; !slices.Equal(got, want) {
		t.Errorf("declaring nothing: rules = %v, want %v", got, want)
	}
	declared := connectHTTP(t, s, http.Header{"Adi-Tags": {"go, rust"}})
	if got, want := listDecisionIDs(t, declared), []string{"ADR-0001", "ADR-0002", "ADR-0003", "ADR-0004"}; !slices.Equal(got, want) {
		t.Errorf("declaring go, rust: decisions = %v, want %v", got, want)
	}
	if got, want := listRuleIDs(t, declared), []string{"ADR-0001", "ADR-0004"}; !slices.Equal(got, want) {
		t.Errorf("declaring go, rust: rules = %v, want %v", got, want)
	}
}
