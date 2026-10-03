package mcp

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

// tracerName is the instrumentation scope the spans are reported under.
const tracerName = "github.com/boykush/adr/adi/internal/mcp"

// tagsKey holds the tags the repository behind a request declares. It is adi's
// own: no convention names what a caller says of itself this way. A span keeps
// them without the opt-in the content of a call waits for: they say which
// repository called, not what it sent.
const tagsKey = attribute.Key("adi.tags")

// toolError is the error.type OpenTelemetry's MCP conventions give a tool call
// that was answered, but with isError set.
const toolError = "tool_error"

// What a span keeps of a request is bounded. Spans wait in memory to be sent,
// and a caller can make a tool's name, its arguments or a header as long as a
// request may be.
const (
	maxFromCaller = 1024
	maxTags       = 32
)

// callerFaults are the JSON-RPC codes that turn away a request the server
// could not serve as sent. The conventions keep them out of a server's errors,
// the fault being the caller's. The last is MCP's own, for a missing resource.
var callerFaults = []int64{
	jsonrpc.CodeParseError,
	jsonrpc.CodeInvalidRequest,
	jsonrpc.CodeMethodNotFound,
	jsonrpc.CodeInvalidParams,
	-32002,
}

// traceContext is how a caller hands over the trace its request belongs to.
var traceContext = propagation.TraceContext{}

// traceRequests reports every message the server receives as a server span,
// named and attributed after OpenTelemetry's semantic conventions for MCP.
// Over HTTP every message is the same POST, so only here can a span tell the
// method and the tool apart.
func (s *Server) traceRequests(tp trace.TracerProvider) mcpsdk.Middleware {
	tracer := tp.Tracer(tracerName, trace.WithSchemaURL(semconv.SchemaURL))
	return func(next mcpsdk.MethodHandler) mcpsdk.MethodHandler {
		return func(ctx context.Context, method string, req mcpsdk.Request) (mcpsdk.Result, error) {
			attrs := append(transport(req), semconv.McpMethodNameKey.String(method))
			if tags := s.declaredTags(req.GetExtra()); len(tags) > 0 {
				attrs = append(attrs, tagsKey.StringSlice(recordedTags(tags)))
			}
			var tool string
			if params, ok := req.GetParams().(*mcpsdk.CallToolParamsRaw); ok && params != nil {
				tool = params.Name
				attrs = append(attrs, semconv.GenAIOperationNameExecuteTool, semconv.GenAIToolName(fromCaller(tool)))
				// A call to a tool that takes nothing sends an empty object,
				// which says nothing worth keeping.
				if arguments := string(params.Arguments); s.cfg.CaptureContent && arguments != "" && arguments != "{}" {
					attrs = append(attrs, semconv.GenAIToolCallArgumentsKey.String(fromCaller(arguments)))
				}
			}
			ctx, span := tracer.Start(withCallerTrace(ctx, req), method,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(attrs...),
			)
			defer span.End()

			res, err := next(ctx, method, req)

			if version := protocolVersion(req, res); version != "" {
				span.SetAttributes(semconv.McpProtocolVersion(version))
			}
			if err != nil {
				s.recordError(span, err)
				return res, err
			}
			// The tool joins the name only once the call has reached it. Until
			// then the name is the caller's to make up, and the conventions keep
			// values without bound out of span names.
			if tool != "" {
				span.SetName(method + " " + tool)
			}
			if result, ok := res.(*mcpsdk.CallToolResult); ok && result != nil && result.IsError {
				span.SetAttributes(semconv.ErrorTypeKey.String(toolError))
				span.SetStatus(codes.Error, s.reason(result.GetError()))
			}
			return res, nil
		}
	}
}

// withCallerTrace returns ctx carrying the trace the caller sent req under, if
// it sent one, so the span continues that trace. MCP carries it in
// params._meta: one HTTP request's headers cannot speak for each message.
func withCallerTrace(ctx context.Context, req mcpsdk.Request) context.Context {
	meta := requestMeta(req)
	carrier := propagation.MapCarrier{}
	for _, field := range traceContext.Fields() {
		if value, ok := meta[field].(string); ok {
			carrier[field] = value
		}
	}
	return traceContext.Extract(ctx, carrier)
}

// requestMeta returns params._meta. A request sent without params reaches the
// middleware as a nil pointer behind the interface, which GetMeta would
// dereference.
func requestMeta(req mcpsdk.Request) map[string]any {
	params := req.GetParams()
	if params == nil {
		return nil
	}
	if v := reflect.ValueOf(params); v.Kind() == reflect.Pointer && v.IsNil() {
		return nil
	}
	return params.GetMeta()
}

// transport says which of the two ways adi is reached req came by. The header
// of an HTTP request reaches the middleware only over Streamable HTTP.
func transport(req mcpsdk.Request) []attribute.KeyValue {
	if extra := req.GetExtra(); extra != nil && extra.Header != nil {
		return []attribute.KeyValue{semconv.NetworkTransportTCP, semconv.NetworkProtocolName("http")}
	}
	return []attribute.KeyValue{semconv.NetworkTransportPipe}
}

// protocolVersion returns the MCP version req was served under. The handshake
// settles it in its result; a later request carries what was settled.
func protocolVersion(req mcpsdk.Request, res mcpsdk.Result) string {
	if init, ok := res.(*mcpsdk.InitializeResult); ok && init != nil {
		return init.ProtocolVersion
	}
	if versioned, ok := req.(interface{ ProtocolVersion() string }); ok {
		return versioned.ProtocolVersion()
	}
	return ""
}

// recordError marks span for a request that ended in err. A JSON-RPC error
// carries a code, and the conventions say which codes fail the server; an
// error without one is the server's own failure.
func (s *Server) recordError(span trace.Span, err error) {
	var rpcErr *jsonrpc.Error
	if errors.As(err, &rpcErr) {
		code := strconv.FormatInt(rpcErr.Code, 10)
		span.SetAttributes(semconv.RPCResponseStatusCode(code))
		if slices.Contains(callerFaults, rpcErr.Code) {
			return
		}
		span.SetAttributes(semconv.ErrorTypeKey.String(code))
	} else {
		span.SetAttributes(semconv.ErrorType(err))
	}
	span.SetStatus(codes.Error, s.reason(err))
}

// reason returns what a span's status says of the failure err: nothing, unless
// the server was asked to keep what callers send. An error quotes its request,
// be it the id of a decision that is not there or arguments that do not fit.
// adi logs nothing, so without the opt-in no place says why a model fails to
// load.
func (s *Server) reason(err error) string {
	if err == nil || !s.cfg.CaptureContent {
		return ""
	}
	return fromCaller(err.Error())
}

// recordedTags returns the declared tags as a span keeps them.
func recordedTags(tags []string) []string {
	tags = tags[:min(len(tags), maxTags)]
	recorded := make([]string, len(tags))
	for i, tag := range tags {
		recorded[i] = fromCaller(tag)
	}
	return recorded
}

// fromCaller makes a string the caller had a hand in fit to keep on a span:
// bounded, and valid UTF-8. A single byte that is not fails the encoding of
// the whole batch a span is sent in, taking other requests' spans with it.
func fromCaller(s string) string {
	if len(s) > maxFromCaller {
		s = s[:maxFromCaller]
	}
	return strings.ToValidUTF8(s, "�")
}
