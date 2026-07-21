package docgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/avivsinai/bitbucket-cli/internal/mcpserver"
)

func appendMCPRegistry(path string) error {
	var content bytes.Buffer
	if err := writeMCPRegistry(&content, mcpserver.InventorySnapshot()); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("open %s for MCP registry: %w", path, err)
	}
	if _, err := file.Write(content.Bytes()); err != nil {
		_ = file.Close()
		return fmt.Errorf("append MCP registry to %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	return nil
}

func writeMCPRegistry(file io.Writer, inventory mcpserver.Inventory) error {
	fmt.Fprintln(file, "## MCP tool registry")
	fmt.Fprintln(file)
	fmt.Fprintln(file, "The read-only v1 server registers eight Bitbucket read tools plus `bkt_get_context`.")
	fmt.Fprintln(file, "Every tool below is available on Data Center and Cloud unless a capability note says otherwise.")
	fmt.Fprintln(file)

	fmt.Fprintln(file, "### Platform capabilities")
	fmt.Fprintln(file)
	fmt.Fprintln(file, "| Platform | Capabilities |")
	fmt.Fprintln(file, "|---|---|")
	for _, platform := range []mcpserver.PlatformInventoryItem{inventory.Platforms.DataCenter, inventory.Platforms.Cloud} {
		capabilities := "None"
		if len(platform.Capabilities) > 0 {
			quoted := make([]string, 0, len(platform.Capabilities))
			for _, capability := range platform.Capabilities {
				quoted = append(quoted, "`"+capability+"`")
			}
			capabilities = strings.Join(quoted, ", ")
		}
		fmt.Fprintf(file, "| %s | %s |\n", platform.Name, capabilities)
	}
	fmt.Fprintln(file)

	fmt.Fprintln(file, "### Frozen bounds")
	fmt.Fprintln(file)
	fmt.Fprintln(file, "| Bound | Value | Meaning |")
	fmt.Fprintln(file, "|---|---:|---|")
	for _, bound := range inventory.Bounds {
		fmt.Fprintf(file, "| `%s` | %s | %s |\n", bound.Name, formatBound(bound.Value, bound.Unit), bound.Description)
	}
	fmt.Fprintln(file)

	fmt.Fprintln(file, "### Structured error codes")
	fmt.Fprintln(file)
	fmt.Fprintln(file, "| Code | Meaning |")
	fmt.Fprintln(file, "|---|---|")
	for _, toolError := range inventory.Errors {
		fmt.Fprintf(file, "| `%s` | %s |\n", toolError.Code, toolError.Description)
	}
	fmt.Fprintln(file)

	fmt.Fprintln(file, "### Tools")
	fmt.Fprintln(file)
	for _, tool := range inventory.Tools {
		fmt.Fprintf(file, "#### `%s`\n\n", tool.Name)
		fmt.Fprintln(file, tool.Description)
		fmt.Fprintln(file)
		fmt.Fprintf(file, "- Availability: %s\n", formatPlatforms(tool.Platforms))
		fmt.Fprintf(file, "- Read-only: %t\n", tool.ReadOnly)
		fmt.Fprintf(file, "- Structured errors: %s\n", formatErrorCodes(tool.Errors))
		for _, note := range tool.Notes {
			fmt.Fprintf(file, "- Note: %s\n", note)
		}
		fmt.Fprintln(file)
		if err := writeSchema(file, "Input schema", tool.InputSchema); err != nil {
			return fmt.Errorf("render input schema for %s: %w", tool.Name, err)
		}
		if err := writeSchema(file, "Output schema", tool.OutputSchema); err != nil {
			return fmt.Errorf("render output schema for %s: %w", tool.Name, err)
		}
	}
	return nil
}

func formatBound(value int, unit string) string {
	if unit == "bytes" && value%(1024) == 0 {
		return fmt.Sprintf("%d KiB", value/1024)
	}
	return fmt.Sprintf("%d %s", value, unit)
}

func formatPlatforms(platforms []string) string {
	if len(platforms) == 2 && platforms[0] == "dc" && platforms[1] == "cloud" {
		return "Data Center and Cloud"
	}
	labels := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		switch platform {
		case "dc":
			labels = append(labels, "Data Center")
		case "cloud":
			labels = append(labels, "Cloud")
		default:
			labels = append(labels, platform)
		}
	}
	return strings.Join(labels, " and ")
}

func formatErrorCodes(codes []mcpserver.ErrorCode) string {
	if len(codes) == 0 {
		return "None"
	}
	formatted := make([]string, 0, len(codes))
	for _, code := range codes {
		formatted = append(formatted, "`"+string(code)+"`")
	}
	return strings.Join(formatted, ", ")
}

func writeSchema(file io.Writer, title string, schema json.RawMessage) error {
	formatted := make([]byte, 0, len(schema)+64)
	formatted = append(formatted, []byte("##### "+title+"\n\n```json\n")...)
	var indented bytes.Buffer
	if err := json.Indent(&indented, schema, "", "  "); err != nil {
		return err
	}
	formatted = append(formatted, indented.String()...)
	formatted = append(formatted, []byte("\n```\n\n")...)
	_, err := file.Write(formatted)
	return err
}
