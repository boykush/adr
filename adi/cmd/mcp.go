package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/boykush/adr/adi/internal/mcp"
	"github.com/boykush/adr/adi/internal/telemetry"
	"github.com/spf13/cobra"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	mcpModel string
	mcpHTTP  string
)

func init() {
	mcpCmd.Flags().StringVar(&mcpModel, "model", "decisions", "path to the decision model directory")
	mcpCmd.Flags().StringVar(&mcpHTTP, "http", "", "serve over Streamable HTTP at this address (e.g. 0.0.0.0:8080) instead of stdio; the MCP endpoint is <addr>/mcp")
	rootCmd.AddCommand(mcpCmd)
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run an MCP server exposing the decision model over stdio or HTTP",
	Long: `Run a Model Context Protocol server that exposes the architectural
decisions and their rule files, so an agent working in another repository can
read what binds it without checking this one out.

By default it serves over stdio, spawned per consumer. Pass --http to instead
serve over Streamable HTTP from one long-running process, so every repository
can share a single server by pointing its MCP client at <addr>/mcp.

The model is never embedded in the binary: --model says where to read it, which
is what lets the decisions ship and change independently of the tool.

A repository narrows the decisions it lists by declaring tags, comma-separated:
in the Adi-Tags header over HTTP, or in ADI_TAGS over stdio. Untagged decisions
are listed whatever it declares.

Requests are traced when OpenTelemetry's standard variables name a collector:
with OTEL_EXPORTER_OTLP_ENDPOINT or OTEL_EXPORTER_OTLP_TRACES_ENDPOINT set, each
MCP request is exported as a span over OTLP/HTTP, under the service name in
OTEL_SERVICE_NAME. A span carries the arguments of the tool call it reports and
the tags the repository declared. With neither variable set, nothing is
exported.`,
	Args: cobra.NoArgs,
	// A server failure is not a misuse of the command.
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		cfg := mcp.Config{ModelDir: mcpModel}

		tp, tracingErr := telemetry.NewTracerProvider(ctx, adiVersion())
		if tracingErr != nil {
			// Tracing serves whoever runs the server. A fault in setting it up
			// is theirs to read, and no reason to turn the agents away.
			fmt.Fprintf(cmd.ErrOrStderr(), "adi: serving without tracing: %v\n", tracingErr)
		}
		if tp != nil {
			cfg.TracerProvider = tp
			// Spans leave in batches, so the signal that ends the process has
			// to end the server instead, leaving time to send the last ones.
			var stop context.CancelFunc
			ctx, stop = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
			defer stop()
			defer func() { err = errors.Join(err, flush(tp)) }()
		}

		srv := mcp.NewServer(cfg, adiVersion())
		if mcpHTTP != "" {
			return srv.RunHTTP(ctx, mcpHTTP)
		}
		if err := srv.Run(ctx); !isCleanShutdown(err) {
			return err
		}
		return nil
	},
}

// flush sends the spans still queued and stops the exporter. The server has
// stopped by now, so the wait is bounded: a collector that does not answer
// must not hold the process past its termination.
func flush(tp *sdktrace.TracerProvider) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := tp.Shutdown(ctx); err != nil {
		return fmt.Errorf("send the last traces: %w", err)
	}
	return nil
}

// isCleanShutdown reports whether err is the normal end of an stdio session:
// the client closed the stream (EOF) or cancelled the context. The SDK wraps
// these in an internal error that is not comparable with errors.Is, so the
// message is matched as a fallback.
func isCleanShutdown(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "EOF") || strings.Contains(msg, "server is closing")
}
