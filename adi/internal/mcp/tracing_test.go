package mcp

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// newTracedServer is newTestServer with a tracer provider that keeps the spans
// where the test can read them, in place of one that exports them.
func newTracedServer(t *testing.T) (*Server, *tracetest.SpanRecorder) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	// A sampler named in the environment running the tests would decide which
	// spans there are to read.
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()), sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	s := newTestServer(t)
	s.cfg.TracerProvider = tp
	return s, recorder
}

func spanAttributes(span sdktrace.ReadOnlySpan) map[attribute.Key]string {
	attrs := make(map[attribute.Key]string)
	for _, kv := range span.Attributes() {
		attrs[kv.Key] = kv.Value.String()
	}
	return attrs
}

// lastSpan returns the span of the request the test has just made.
func lastSpan(t *testing.T, recorder *tracetest.SpanRecorder) sdktrace.ReadOnlySpan {
	t.Helper()
	spans := recorder.Ended()
	if len(spans) == 0 {
		t.Fatal("no span was reported")
	}
	return spans[len(spans)-1]
}

// postMessage sends one JSON-RPC message the way a client that holds no session
// does, each in a POST of its own. It takes the message as bytes rather than
// as params to marshal, so a test can send what no well-behaved client would.
func postMessage(t *testing.T, s *Server, header http.Header, message string) {
	t.Helper()
	httpServer := httptest.NewServer(s.httpHandler())
	defer httpServer.Close()
	req, err := http.NewRequest(http.MethodPost, httpServer.URL+httpPath, strings.NewReader(message))
	if err != nil {
		t.Fatal(err)
	}
	maps.Copy(req.Header, header)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= http.StatusBadRequest {
		t.Fatalf("post: %s", res.Status)
	}
}

// TestRequestsAreTraced reaches a traced server over HTTP, where every call is
// the same POST, and reads back the spans that tell the calls apart.
func TestRequestsAreTraced(t *testing.T) {
	ctx := context.Background()
	s, recorder := newTracedServer(t)
	cs := connectHTTP(t, s, nil)
	// The handshake is the SDK's to shape, so the spans read here start after it.
	handshake := len(recorder.Ended())

	if _, err := cs.ListTools(ctx, nil); err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if _, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "list_rules", Arguments: map[string]any{}}); err != nil {
		t.Fatalf("call list_rules: %v", err)
	}
	if res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "get_decision", Arguments: map[string]any{"adr_id": "99"}}); err != nil || !res.IsError {
		t.Fatalf("get_decision of an absent id: err = %v, want a result that is an error", err)
	}
	if _, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "get_rule", Arguments: map[string]any{}}); err == nil {
		t.Fatal("call of a tool the server does not have succeeded")
	}

	toolCall := func(tool string, more map[attribute.Key]string) map[attribute.Key]string {
		attrs := map[attribute.Key]string{"mcp.method.name": "tools/call", "gen_ai.operation.name": "execute_tool", "gen_ai.tool.name": tool}
		maps.Copy(attrs, more)
		return attrs
	}
	want := []struct {
		name   string
		attrs  map[attribute.Key]string
		status codes.Code
		// reason is what the status has to say of the failure.
		reason string
	}{
		{name: "tools/list", attrs: map[attribute.Key]string{"mcp.method.name": "tools/list"}},
		// The tool takes no arguments, so the span keeps none.
		{name: "tools/call list_rules", attrs: toolCall("list_rules", nil)},
		// The tool answered, and its answer was the failure. The span is the
		// only place that says why: adi keeps no log.
		{
			name:   "tools/call get_decision",
			attrs:  toolCall("get_decision", map[attribute.Key]string{"gen_ai.tool.call.arguments": `{"adr_id":"99"}`, "error.type": "tool_error"}),
			status: codes.Error,
			reason: "no decision ADR-0099",
		},
		// No tool was reached, so none is in the name: a caller can ask for any
		// name it likes. Asking for one that is not there is its fault, not the
		// server's failure.
		{name: "tools/call", attrs: toolCall("get_rule", map[attribute.Key]string{"rpc.response.status_code": "-32602"})},
	}

	spans := recorder.Ended()[handshake:]
	// One span a request, and none besides.
	if len(spans) != len(want) {
		t.Fatalf("got %d spans, want %d", len(spans), len(want))
	}
	for i, w := range want {
		span := spans[i]
		if span.Name() != w.name {
			t.Errorf("span %d is named %q, want %q", i, span.Name(), w.name)
		}
		if span.SpanKind() != trace.SpanKindServer {
			t.Errorf("%s: kind = %v, want server", w.name, span.SpanKind())
		}
		// Nothing the caller sent places the request in a trace, so each starts
		// its own.
		if span.Parent().IsValid() {
			t.Errorf("%s: has a parent, %v", w.name, span.Parent())
		}
		if got := span.Status(); got.Code != w.status || !strings.Contains(got.Description, w.reason) {
			t.Errorf("%s: status = %v %q, want %v %q", w.name, got.Code, got.Description, w.status, w.reason)
		}
		w.attrs["network.transport"] = "tcp"
		w.attrs["network.protocol.name"] = "http"
		w.attrs["mcp.protocol.version"] = cs.InitializeResult().ProtocolVersion
		// Compared whole: an attribute the conventions keep off a span, like the
		// tool operation on a call to no tool, would be an extra here.
		if got := spanAttributes(span); !maps.Equal(got, w.attrs) {
			t.Errorf("%s: attributes = %v, want %v", w.name, got, w.attrs)
		}
	}
}

// TestHandshakeIsTraced sends what a client on the protocol that opens with
// initialize sends. The listing comes without params, which the tracing has to
// read all the same.
func TestHandshakeIsTraced(t *testing.T) {
	s, recorder := newTracedServer(t)

	postMessage(t, s, nil, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test-client","version":"0"}}}`)
	postMessage(t, s, nil, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)

	spans := recorder.Ended()
	want := []string{"initialize", "tools/list"}
	if len(spans) != len(want) {
		t.Fatalf("got %d spans, want %d", len(spans), len(want))
	}
	for i, name := range want {
		if spans[i].Name() != name {
			t.Errorf("span %d is named %q, want %q", i, spans[i].Name(), name)
		}
	}
	// The version is the one the handshake settled, which only its result says.
	if got := spanAttributes(spans[0])["mcp.protocol.version"]; got != "2025-06-18" {
		t.Errorf("initialize: mcp.protocol.version = %q, want 2025-06-18", got)
	}
}

// TestSpansSayHowAndByWhomTheServerWasReached covers the two transports: over
// each, a repository declares its tags its own way.
func TestSpansSayHowAndByWhomTheServerWasReached(t *testing.T) {
	ctx := context.Background()

	s, recorder := newTracedServer(t)
	if _, err := connectHTTP(t, s, http.Header{"Adi-Tags": {"product, rust"}}).ListTools(ctx, nil); err != nil {
		t.Fatalf("list tools over HTTP: %v", err)
	}
	attrs := spanAttributes(lastSpan(t, recorder))
	if attrs["network.transport"] != "tcp" || attrs["network.protocol.name"] != "http" {
		t.Errorf("over HTTP: transport = %q, protocol = %q, want tcp and http", attrs["network.transport"], attrs["network.protocol.name"])
	}
	if got, want := attrs["adi.tags"], `["product","rust"]`; got != want {
		t.Errorf("over HTTP: adi.tags = %s, want %s", got, want)
	}

	s, recorder = newTracedServer(t)
	t.Setenv(tagsEnv, "go")
	if _, err := connect(t, s).ListTools(ctx, nil); err != nil {
		t.Fatalf("list tools over stdio: %v", err)
	}
	attrs = spanAttributes(lastSpan(t, recorder))
	if attrs["network.transport"] != "pipe" {
		t.Errorf("over stdio: transport = %q, want pipe", attrs["network.transport"])
	}
	if name, ok := attrs["network.protocol.name"]; ok {
		t.Errorf("over stdio: network.protocol.name = %q, want none without HTTP", name)
	}
	if got, want := attrs["adi.tags"], `["go"]`; got != want {
		t.Errorf("over stdio: adi.tags = %s, want %s", got, want)
	}
}

// TestSpanContinuesTheCallersTrace sends the trace context where MCP carries
// it, in the request's _meta.
func TestSpanContinuesTheCallersTrace(t *testing.T) {
	const (
		traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
		spanID  = "00f067aa0ba902b7"
	)
	s, recorder := newTracedServer(t)
	cs := connectHTTP(t, s, nil)
	if _, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Meta:      mcpsdk.Meta{"traceparent": "00-" + traceID + "-" + spanID + "-01"},
		Name:      "list_rules",
		Arguments: map[string]any{},
	}); err != nil {
		t.Fatalf("call list_rules: %v", err)
	}

	span := lastSpan(t, recorder)
	if got := span.SpanContext().TraceID().String(); got != traceID {
		t.Errorf("trace id = %s, want the caller's %s", got, traceID)
	}
	if parent := span.Parent(); parent.SpanID().String() != spanID || !parent.IsRemote() {
		t.Errorf("parent = %v, want the caller's span %s", parent, spanID)
	}
}

// TestWhatTheCallerSendsIsKeptInBounds sends, in every place a span reads the
// request, more than a span should hold and a byte that is not UTF-8. One such
// byte in a span fails the export of every span batched with it.
func TestWhatTheCallerSendsIsKeptInBounds(t *testing.T) {
	const notUTF8 = "\xff"
	long := strings.Repeat("9", 4*maxFromCaller)
	tags := strings.Repeat("go"+notUTF8+",", 2*maxTags)

	for _, message := range []string{
		// The name of a tool the server does not have, with arguments.
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + long + `","arguments":{"note":"` + notUTF8 + long + `"}}}`,
		// An id that the tool's error quotes back.
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_decision","arguments":{"adr_id":"` + long + `"}}}`,
	} {
		s, recorder := newTracedServer(t)
		postMessage(t, s, http.Header{"Adi-Tags": {tags}}, message)
		span := lastSpan(t, recorder)

		kept := []string{span.Status().Description}
		for _, kv := range span.Attributes() {
			if kv.Key != tagsKey {
				kept = append(kept, kv.Value.String())
				continue
			}
			declared := kv.Value.AsStringSlice()
			if len(declared) != maxTags {
				t.Errorf("%s: the span keeps %d tags, want %d", span.Name(), len(declared), maxTags)
			}
			kept = append(kept, declared...)
		}
		for _, value := range kept {
			// What stands in for the byte that is not UTF-8 is longer than it.
			if len(value) > maxFromCaller+utf8.UTFMax {
				t.Errorf("%s: the span keeps %d bytes of one value, want about %d", span.Name(), len(value), maxFromCaller)
			}
			if !utf8.ValidString(value) {
				t.Errorf("%s: the span keeps a value that is not UTF-8: %q", span.Name(), value)
			}
		}
	}
}
