package mcpserver

import (
	"bytes"
	"encoding/json"
	"os"
	"slices"
	"testing"
)

func TestInventorySnapshotMatchesGolden(t *testing.T) {
	inventory := InventorySnapshot()

	wantNames := []string{
		"bkt_get_context",
		"bkt_get_pull_request",
		"bkt_get_pull_request_checks",
		"bkt_get_pull_request_diff",
		"bkt_get_repository",
		"bkt_list_my_pull_requests",
		"bkt_list_pull_request_comments",
		"bkt_list_pull_requests",
		"bkt_list_repositories",
	}
	if len(inventory.Tools) != len(wantNames) {
		t.Fatalf("registered tool count = %d, want %d", len(inventory.Tools), len(wantNames))
	}

	gotNames := make([]string, 0, len(inventory.Tools))
	for _, tool := range inventory.Tools {
		gotNames = append(gotNames, tool.Name)
		if !tool.ReadOnly {
			t.Errorf("tool %q readOnly = false, want true", tool.Name)
		}
		if len(tool.InputSchema) == 0 || len(tool.OutputSchema) == 0 {
			t.Errorf("tool %q must freeze both input and output schemas", tool.Name)
		}
	}
	if !slices.Equal(gotNames, wantNames) {
		t.Fatalf("registered tool names = %v, want %v", gotNames, wantNames)
	}

	data, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		t.Fatalf("marshal inventory: %v", err)
	}
	data = append(data, '\n')
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("create testdata: %v", err)
		}
		if err := os.WriteFile("testdata/tool_registry.golden.json", data, 0o644); err != nil {
			t.Fatalf("update registry golden: %v", err)
		}
	}
	want, err := os.ReadFile("testdata/tool_registry.golden.json")
	if err != nil {
		t.Fatalf("read registry golden: %v", err)
	}
	if !bytes.Equal(data, want) {
		t.Fatalf("tool registry drifted; review the schema change and update testdata/tool_registry.golden.json\n--- got ---\n%s\n--- want ---\n%s", data, want)
	}
}

func TestInventoryKeepsSDKValidatedRoleOptional(t *testing.T) {
	inventory := InventorySnapshot()
	for _, tool := range inventory.Tools {
		if tool.Name != "bkt_list_my_pull_requests" {
			continue
		}
		var schema struct {
			Required []string `json:"required"`
		}
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
			t.Fatalf("decode input schema: %v", err)
		}
		if slices.Contains(schema.Required, "role") {
			t.Fatalf("role must stay optional in the SDK input schema; the handler returns structured invalid_input when it is omitted")
		}
		return
	}
	t.Fatal("bkt_list_my_pull_requests missing from inventory")
}

func TestInventoryPinsPlatformCapabilities(t *testing.T) {
	inventory := InventorySnapshot()
	if !slices.Equal(inventory.Platforms.DataCenter.Capabilities, []string{"my_prs.role.reviewer"}) {
		t.Fatalf("Data Center capabilities = %v", inventory.Platforms.DataCenter.Capabilities)
	}
	if len(inventory.Platforms.Cloud.Capabilities) != 0 {
		t.Fatalf("Cloud capabilities = %v, want none", inventory.Platforms.Cloud.Capabilities)
	}
}
