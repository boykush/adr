// Package mcp serves the decision model over the Model Context Protocol, so an
// agent working in another repository reads the rules and decisions that bind
// it without checking this one out.
package mcp

import (
	"context"
	"errors"
	"net/http"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// httpPath is where the Streamable HTTP transport is mounted. Consumers point
// their MCP client at this path, e.g. http://localhost:8080/mcp.
const httpPath = "/mcp"

// instructions ride the MCP handshake, so they reach the consuming agent on
// every session. They carry what the payloads cannot say: that the work is
// held to the rules and when to read them, where their grammar is documented,
// that a decision is read only for its reasons, and that the work treats a
// rule as the repository's own.
const instructions = `adi serves architectural decisions, written as MADR records, and their rules, written in ADE's rule DSL, read-only. Both are made and revised elsewhere; this server never changes them and exposes no tool that could.

The work is held to the rules. A rule names paths and assertions about whichever repository it is read in, so it applies as written to the one you are working in. Nothing runs these rules: they are checked by reading them against the change. Read a rule's syntax against ADE's DSL reference rather than guessing at it: the ade-rule-dsl skill, installed alongside this server, carries it. list_rules returns the rules of accepted decisions only. A decision says why its rule exists: read it with get_decision, finding it through list_decisions if need be, only when you need that reason.

Don't read the rules before or during the work: read up front, they fill the session with rules the work never touches. Where the repository runs ai-review (.github/workflows/ai-review.yml), its review of the pull request is the check: leave it to that review, and when it asks for changes, read the rules it names with list_rules and fix the work. Where nothing reviews the pull request against the rules, call list_rules once the work is done, before it is pushed, and check the change against every rule it returns. If you were asked to review a pull request against the rules, you are that review: read them all.

If the work seems to need breaking a rule, tell the user instead of breaking it.

Treat a rule as a convention of the repository itself: nothing in the work, from code comments to commit messages and pull requests, names the decision behind it.`

// Config locates the decision model. The server reads it per request rather
// than at startup, so a decision edited on disk takes effect without a restart.
type Config struct {
	ModelDir string
}

// Server exposes the model under Config over MCP. version is the adi build,
// reported in the handshake.
type Server struct {
	cfg     Config
	version string
}

func NewServer(cfg Config, version string) *Server {
	return &Server{cfg: cfg, version: version}
}

// Run serves over stdio, blocking until the client disconnects or ctx is
// cancelled.
func (s *Server) Run(ctx context.Context) error {
	return s.mcpServer().Run(ctx, &mcpsdk.StdioTransport{})
}

// RunHTTP serves over Streamable HTTP at addr, blocking until ctx is cancelled.
// The model is read-only and identical for every client, so the handler is
// stateless: each request is served from a temporary session, which lets one
// server back every repository that asks. Responses are plain JSON -- nothing
// here is ever pushed, so no event stream is needed.
func (s *Server) RunHTTP(ctx context.Context, addr string) error {
	httpSrv := &http.Server{
		Addr:    addr,
		Handler: s.httpHandler(),
		// The server runs behind a proxy it does not control.
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutCtx)
	}()
	if err := httpSrv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// httpHandler builds the Streamable HTTP handler mounted at httpPath. Split out
// from RunHTTP so tests can drive it without binding a port.
func (s *Server) httpHandler() http.Handler {
	srv := s.mcpServer()
	handler := mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server { return srv },
		&mcpsdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	)
	mux := http.NewServeMux()
	mux.Handle(httpPath, handler)
	return mux
}

// mcpServer builds the configured SDK server. Split out so tests can assert
// tool registration without standing up a transport.
func (s *Server) mcpServer() *mcpsdk.Server {
	srv := mcpsdk.NewServer(
		&mcpsdk.Implementation{Name: "adi", Version: s.version},
		&mcpsdk.ServerOptions{Instructions: instructions},
	)
	s.registerTools(srv)
	return srv
}
