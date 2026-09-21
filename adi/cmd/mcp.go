package cmd

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/boykush/adr/adi/internal/mcp"
	"github.com/spf13/cobra"
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
is what lets the decisions ship and change independently of the tool.`,
	Args: cobra.NoArgs,
	// A server failure is not a misuse of the command.
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		srv := mcp.NewServer(mcp.Config{ModelDir: mcpModel}, adiVersion())
		if mcpHTTP != "" {
			return srv.RunHTTP(cmd.Context(), mcpHTTP)
		}
		if err := srv.Run(cmd.Context()); !isCleanShutdown(err) {
			return err
		}
		return nil
	},
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
