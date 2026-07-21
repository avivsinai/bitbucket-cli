package docgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestGenerateAllAddsSelfContainedMCPRegistryReference(t *testing.T) {
	root := &cobra.Command{Use: "bkt"}
	mcpCommand := &cobra.Command{Use: "mcp", Short: "Serve Bitbucket tools to MCP clients"}
	mcpCommand.AddCommand(&cobra.Command{Use: "serve", Short: "Serve MCP over stdio"})
	root.AddCommand(mcpCommand)

	skillDir := t.TempDir()
	outDir := filepath.Join(skillDir, "rules")
	writeTestFile(t, filepath.Join(skillDir, "SKILL.md"), "# Test\n")
	if err := GenerateAll(root, "bkt", outDir); err != nil {
		t.Fatalf("GenerateAll: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outDir, "mcp.md"))
	if err != nil {
		t.Fatalf("read mcp.md: %v", err)
	}
	got := string(content)
	for _, want := range []string{
		"## MCP tool registry",
		"eight Bitbucket read tools plus `bkt_get_context`",
		"`bkt_get_context`",
		"`bkt_get_pull_request_checks`",
		"`bkt_list_repositories`",
		"Data Center and Cloud",
		"`my_prs.role.reviewer`",
		"`unsupported_on_platform`",
		"256 KiB",
		"Input schema",
		"Output schema",
		"role",
		"restart the MCP server",
		"bearer-only",
		"Content-Length",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated mcp.md missing %q", want)
		}
	}
}
