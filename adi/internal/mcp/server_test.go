package mcp

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"AD0001-first.md":  "---\nadr_id: \"0001\"\nstatus: decided\ntitle: First\n---\n\n## Decision\n\nthe first one\n",
		"AD0002-second.md": "---\nadr_id: \"0002\"\nstatus: open\ntitle: Second\n---\n\n## Question\n\nstill arguing\n",
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
// two tools, and reading one decision by each spelling of its id.
func TestEndToEnd(t *testing.T) {
	ctx := context.Background()
	cs := connect(t, newTestServer(t))

	// The handshake is where the citation practice reaches the agent. The
	// wording is free to change; that it arrives and names an id is not.
	if got := cs.InitializeResult().Instructions; got == "" || !strings.Contains(got, "AD0001") {
		t.Errorf("instructions = %q, want non-empty and showing a cited id", got)
	}

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	// Exactly two: the surface is read-only, and a third tool would be a change
	// of shape rather than an addition.
	if len(names) != 2 || !slicesContains(names, "list_decisions") || !slicesContains(names, "get_decision") {
		t.Fatalf("tools = %v, want exactly list_decisions and get_decision", names)
	}

	res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "list_decisions", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("call list_decisions: %v", err)
	}
	var list listDecisionsOutput
	if err := json.Unmarshal([]byte(contentText(res)), &list); err != nil {
		t.Fatalf("decode list_decisions: %v", err)
	}
	if len(list.Decisions) != 2 {
		t.Fatalf("decisions = %+v, want 2", list.Decisions)
	}
	if list.Decisions[0].ADRID != "AD0001" || list.Decisions[0].Status != "decided" {
		t.Errorf("first = %+v", list.Decisions[0])
	}
	// The listing carries status so an agent can tell a decision that binds from
	// one still being argued, without reading either in full.
	if list.Decisions[1].Status != "open" {
		t.Errorf("second status = %q, want open", list.Decisions[1].Status)
	}

	for _, id := range []string{"1", "0001", "AD0001"} {
		res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "get_decision", Arguments: map[string]any{"adr_id": id}})
		if err != nil {
			t.Fatalf("call get_decision(%q): %v", id, err)
		}
		var got getDecisionOutput
		if err := json.Unmarshal([]byte(contentText(res)), &got); err != nil {
			t.Fatalf("decode get_decision(%q): %v", id, err)
		}
		if got.ADRID != "AD0001" || !strings.Contains(got.Body, "the first one") {
			t.Errorf("get_decision(%q) = %+v", id, got)
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
	if len(list.Decisions) != 2 {
		t.Fatalf("decisions = %+v, want 2", list.Decisions)
	}
}

func slicesContains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
