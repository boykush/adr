// Package mcp serves the decision model over the Model Context Protocol, so an
// agent working in another repository reads the decisions that bind it without
// checking this one out.
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
// every session. They carry the two things the payloads cannot say on their
// own: how to read a status, and how to cite a decision in work that outlives
// the connection.
const instructions = `adi serves a model of architectural decisions, read-only. The decisions are made and revised elsewhere; this server never changes them and exposes no tool that could.

Call list_decisions at the start of a session, then get_decision for the ones that bear on the work at hand. Read the status before following a decision: a decided one binds the work, an open one is a decision point still being argued and binds nothing, and a superseded one names what replaced it.

Cite a decision by its id wherever a constraint it imposes shows up outside this model -- a commit message, a pull request body, a code comment -- so a reader can find the reasoning behind the rule.

    good: // AD0001: the instructions live in AGENTS.md, so no CLAUDE.md here
    bad:  // project convention`

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
