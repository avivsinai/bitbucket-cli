// Package mcp wires the Model Context Protocol server into the CLI.
package mcp

import (
	"fmt"

	"github.com/spf13/cobra"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/avivsinai/bitbucket-cli/internal/mcpserver"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdMCP groups MCP-related commands.
func NewCmdMCP(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Model Context Protocol server for agents",
		Long:  "Serve Bitbucket tools to MCP clients such as Claude Code and Codex.",
	}

	cmd.AddCommand(newServeCmd(f))

	return cmd
}

func newServeCmd(f *cmdutil.Factory) *cobra.Command {
	return newServeCmdWithTransport(f, nil)
}

// newServeCmdWithTransport lets tests drive the full serve path (context
// resolution, banner, server run) over an injected transport; nil means the
// real stdio transport.
func newServeCmdWithTransport(f *cmdutil.Factory, transport sdk.Transport) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve Bitbucket tools over MCP stdio (read-only)",
		Long: `Start a Model Context Protocol server speaking JSON-RPC over stdio.

The server pins ONE bkt context, resolved once at startup: --context selects
a named context, otherwise the active context is used. The working directory
never influences the served target, and configuration changes require a
restart. Tool calls cannot switch hosts, contexts, or tokens.

v1 is read-only and registers tools only for capabilities the pinned platform
supports; call bkt_get_context to discover the target and capabilities.

stdout carries only MCP protocol messages; all diagnostics go to stderr.

Register with an MCP client, e.g. for Claude Code:
  claude mcp add bitbucket -- bkt mcp serve`,
		Example: `  # Serve the active context
  bkt mcp serve

  # Serve a specific named context
  bkt mcp serve --context work-dc`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd, f, transport)
		},
	}

	return cmd
}

func runServe(cmd *cobra.Command, f *cmdutil.Factory, transport sdk.Transport) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	snap, err := mcpserver.ResolveSnapshot(f, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}

	// Startup banner goes to stderr: stdout is reserved for the protocol.
	label := snap.ContextName
	if label == "" {
		label = "(env)"
	}
	fmt.Fprintf(ios.ErrOut, "bkt mcp serve: context %s, platform %s, host %s (read-only; Ctrl-C to stop)\n",
		label, snap.Platform, snap.HostLabel)

	server := mcpserver.New(snap, f.AppVersion)
	if transport == nil {
		transport = &sdk.StdioTransport{}
	}
	return server.Run(cmd.Context(), transport)
}
