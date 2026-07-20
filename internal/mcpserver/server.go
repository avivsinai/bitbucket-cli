// Package mcpserver implements bkt's Model Context Protocol server: a typed
// tool registry over the bbdc/bbcloud clients, serving one context frozen at
// startup. v1 is read-only and stdio-only; see docs/plans for the full spec.
package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// Snapshot is the frozen effective target resolved once at server startup.
// It is a deep copy: later config edits require a server restart, and the
// working directory never influences it (resolution skips git-remote
// default detection).
type Snapshot struct {
	ContextName  string
	Platform     string // "dc" or "cloud"
	HostLabel    string // key of the host entry in config.yml
	Host         config.Host
	DefaultScope string // DC project key or Cloud workspace
	DefaultRepo  string
}

// ResolveSnapshot resolves and freezes the served context. contextOverride
// selects a named context; empty means the active context (or env synthesis).
func ResolveSnapshot(f *cmdutil.Factory, contextOverride string) (*Snapshot, error) {
	name, ctx, host, err := cmdutil.ResolveContextStatic(f, contextOverride)
	if err != nil {
		return nil, err
	}
	if host.Kind != "dc" && host.Kind != "cloud" {
		return nil, fmt.Errorf("context host kind %q is not supported", host.Kind)
	}

	scope := ctx.ProjectKey
	if host.Kind == "cloud" {
		scope = ctx.Workspace
	}

	return &Snapshot{
		ContextName:  name,
		Platform:     host.Kind,
		HostLabel:    ctx.Host,
		Host:         *host,
		DefaultScope: scope,
		DefaultRepo:  ctx.DefaultRepo,
	}, nil
}

// ContextInfo is the bkt_get_context result DTO. Field shapes are part of
// the frozen v1 contract; never include credentials.
type ContextInfo struct {
	Platform     string   `json:"platform" jsonschema:"the pinned Bitbucket platform: dc (Data Center) or cloud"`
	HostLabel    string   `json:"host_label" jsonschema:"the bkt config host entry this server is pinned to"`
	DefaultScope string   `json:"default_scope,omitempty" jsonschema:"default scope (DC project key or Cloud workspace) used when a tool call omits the repository locator"`
	DefaultRepo  string   `json:"default_repo,omitempty" jsonschema:"default repository slug used when a tool call omits the repository locator"`
	Capabilities []string `json:"capabilities" jsonschema:"capability identifiers for the tools and roles this server supports on the pinned platform"`
}

// New builds the MCP server for the given frozen snapshot. All registered
// tools are read-only in v1; registration is the single enforcement point.
func New(snap *Snapshot, version string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "bkt", Version: version}, nil)
	registerGetContext(server, snap)
	return server
}

// capabilities lists what the pinned platform supports. Wave C extends this
// alongside the tool registry; identifiers are part of the public contract.
func capabilities(snap *Snapshot) []string {
	caps := []string{"context"}
	_ = snap // platform-dependent capabilities arrive with the read tools
	return caps
}

type getContextArgs struct{}

func registerGetContext(server *mcp.Server, snap *Snapshot) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "bkt_get_context",
		Description: "Describe the Bitbucket target this server is pinned to: " +
			"platform (dc or cloud), host label, default repository scope/slug, " +
			"and the capabilities available here. Never returns credentials.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args getContextArgs) (*mcp.CallToolResult, ContextInfo, error) {
		return nil, ContextInfo{
			Platform:     snap.Platform,
			HostLabel:    snap.HostLabel,
			DefaultScope: snap.DefaultScope,
			DefaultRepo:  snap.DefaultRepo,
			Capabilities: capabilities(snap),
		}, nil
	})
}
