package mcpserver

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sort"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var allPlatforms = []string{"dc", "cloud"}

type toolDocumentation struct {
	Platforms []string
	Errors    []ErrorCode
	Notes     []string
}

type registeredTool struct {
	tool          *mcp.Tool
	documentation toolDocumentation
}

type toolRegistry struct {
	server *mcp.Server
	tools  []registeredTool
}

func newToolRegistry(server *mcp.Server) *toolRegistry {
	return &toolRegistry{server: server}
}

func standardReadErrors() []ErrorCode {
	return []ErrorCode{
		ErrorInvalidInput,
		ErrorNotFound,
		ErrorAuthFailed,
		ErrorRateLimited,
		ErrorUpstream,
	}
}

func schemaFor[T any]() *jsonschema.Schema {
	typeOf := reflect.TypeFor[T]()
	if typeOf.Kind() == reflect.Pointer {
		typeOf = typeOf.Elem()
	}
	schema, err := jsonschema.ForType(typeOf, &jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("infer MCP schema for %s: %v", typeOf, err))
	}
	return schema
}

// addReadOnlyTool is the single registration path for v1 tools. It materializes
// the inferred schemas before handing the tool to the SDK so runtime serving,
// schema goldens, and generated documentation share one typed registry.
func addReadOnlyTool[In, Out any](registry *toolRegistry, tool *mcp.Tool, documentation toolDocumentation, handler mcp.ToolHandlerFor[In, Out]) {
	if tool.Annotations == nil {
		tool.Annotations = &mcp.ToolAnnotations{}
	}
	tool.Annotations.ReadOnlyHint = true
	if tool.InputSchema == nil {
		tool.InputSchema = schemaFor[In]()
	}
	if tool.OutputSchema == nil {
		tool.OutputSchema = schemaFor[Out]()
	}
	if len(documentation.Platforms) == 0 {
		documentation.Platforms = slices.Clone(allPlatforms)
	}
	if documentation.Errors == nil {
		documentation.Errors = []ErrorCode{}
	}
	if documentation.Notes == nil {
		documentation.Notes = []string{}
	}
	registry.tools = append(registry.tools, registeredTool{tool: tool, documentation: documentation})
	if registry.server != nil {
		mcp.AddTool(registry.server, tool, handler)
	}
}

// Inventory is a serializable snapshot of the complete MCP registry. It is the
// source for both schema drift tests and the generated standalone skill rule.
type Inventory struct {
	Platforms PlatformInventory   `json:"platforms"`
	Bounds    []BoundInventory    `json:"bounds"`
	Errors    []ErrorInventory    `json:"errors"`
	Tools     []ToolInventoryItem `json:"tools"`
}

type PlatformInventory struct {
	DataCenter PlatformInventoryItem `json:"data_center"`
	Cloud      PlatformInventoryItem `json:"cloud"`
}

type PlatformInventoryItem struct {
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities"`
}

type BoundInventory struct {
	Name        string `json:"name"`
	Value       int    `json:"value"`
	Unit        string `json:"unit"`
	Description string `json:"description"`
}

type ErrorInventory struct {
	Code        ErrorCode `json:"code"`
	Description string    `json:"description"`
}

type ToolInventoryItem struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Platforms    []string        `json:"platforms"`
	ReadOnly     bool            `json:"read_only"`
	Errors       []ErrorCode     `json:"errors"`
	Notes        []string        `json:"notes"`
	InputSchema  json.RawMessage `json:"input_schema"`
	OutputSchema json.RawMessage `json:"output_schema"`
}

// InventorySnapshot returns the nine-tool v1 registry without constructing a
// platform client or reading user configuration.
func InventorySnapshot() Inventory {
	registry := newToolRegistry(nil)
	snapshot := &Snapshot{Platform: "dc"}
	var backend fullPlatformBackend
	registerFullTools(registry, snapshot, backend)

	tools := make([]ToolInventoryItem, 0, len(registry.tools))
	for _, registered := range registry.tools {
		inputSchema, err := json.Marshal(registered.tool.InputSchema)
		if err != nil {
			panic(fmt.Sprintf("marshal input schema for %s: %v", registered.tool.Name, err))
		}
		outputSchema, err := json.Marshal(registered.tool.OutputSchema)
		if err != nil {
			panic(fmt.Sprintf("marshal output schema for %s: %v", registered.tool.Name, err))
		}
		tools = append(tools, ToolInventoryItem{
			Name:         registered.tool.Name,
			Description:  registered.tool.Description,
			Platforms:    slices.Clone(registered.documentation.Platforms),
			ReadOnly:     registered.tool.Annotations != nil && registered.tool.Annotations.ReadOnlyHint,
			Errors:       slices.Clone(registered.documentation.Errors),
			Notes:        slices.Clone(registered.documentation.Notes),
			InputSchema:  inputSchema,
			OutputSchema: outputSchema,
		})
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })

	return Inventory{
		Platforms: PlatformInventory{
			DataCenter: PlatformInventoryItem{Name: "Data Center", Capabilities: capabilities(&Snapshot{Platform: "dc"})},
			Cloud:      PlatformInventoryItem{Name: "Cloud", Capabilities: capabilities(&Snapshot{Platform: "cloud"})},
		},
		Bounds: []BoundInventory{
			{Name: "default_list_limit", Value: DefaultListLimit, Unit: "items", Description: "default page size for bounded list tools"},
			{Name: "max_list_limit", Value: MaxListLimit, Unit: "items", Description: "maximum returned items for bounded list and check tools"},
			{Name: "comment_body_limit", Value: CommentBodyLimit, Unit: "bytes", Description: "maximum retained comment body"},
			{Name: "pull_request_description_limit", Value: PullRequestDescriptionLimit, Unit: "bytes", Description: "maximum retained pull request description"},
			{Name: "diff_content_limit", Value: DiffContentLimit, Unit: "bytes", Description: "maximum retained unified diff content"},
		},
		Errors: []ErrorInventory{
			{Code: ErrorInvalidInput, Description: "the tool arguments or frozen context are incomplete or invalid"},
			{Code: ErrorNotFound, Description: "the requested Bitbucket resource was not found"},
			{Code: ErrorAuthFailed, Description: "Bitbucket rejected the frozen credential"},
			{Code: ErrorUnsupportedOnPlatform, Description: "the requested operation is unavailable on the pinned platform"},
			{Code: ErrorRateLimited, Description: "Bitbucket rate-limited the request; retryable is true"},
			{Code: ErrorUpstream, Description: "Bitbucket or the transport failed; retryable reflects the failure class"},
		},
		Tools: tools,
	}
}
