package bbcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListRepositoryVariablesValidation(t *testing.T) {
	client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name          string
		workspace     string
		repoSlug      string
		errorContains string
	}{
		{
			name:          "missing workspace",
			workspace:     "",
			repoSlug:      "repo",
			errorContains: "workspace and repository slug are required",
		},
		{
			name:          "missing repo slug",
			workspace:     "workspace",
			repoSlug:      "",
			errorContains: "workspace and repository slug are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.ListRepositoryVariables(ctx, tt.workspace, tt.repoSlug, VariableListOptions{})
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errorContains)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}

func TestListRepositoryVariablesPagination(t *testing.T) {
	var requestCount int
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		switch requestCount {
		case 1:
			resp := variableListPage{
				Values: []PipelineVariable{
					{UUID: "{uuid-1}", Key: "VAR1", Value: "value1"},
					{UUID: "{uuid-2}", Key: "VAR2", Value: "value2"},
				},
				Next: serverURL + "/repositories/ws/repo/pipelines_config/variables?page=2",
			}
			_ = json.NewEncoder(w).Encode(resp)
		case 2:
			resp := variableListPage{
				Values: []PipelineVariable{
					{UUID: "{uuid-3}", Key: "VAR3", Value: "value3"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			t.Fatalf("unexpected request %d", requestCount)
		}
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	variables, err := client.ListRepositoryVariables(ctx, "ws", "repo", VariableListOptions{})
	if err != nil {
		t.Fatalf("ListRepositoryVariables: %v", err)
	}

	if len(variables) != 3 {
		t.Errorf("expected 3 variables, got %d", len(variables))
	}
	if requestCount != 2 {
		t.Errorf("expected 2 requests for pagination, got %d", requestCount)
	}
}

func TestListRepositoryVariablesRespectsLimit(t *testing.T) {
	var requestCount int
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		resp := variableListPage{
			Values: []PipelineVariable{
				{UUID: "{uuid-1}", Key: "VAR1"},
				{UUID: "{uuid-2}", Key: "VAR2"},
				{UUID: "{uuid-3}", Key: "VAR3"},
			},
			Next: serverURL + "/repositories/ws/repo/pipelines_config/variables?page=2",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	variables, err := client.ListRepositoryVariables(ctx, "ws", "repo", VariableListOptions{Limit: 2})
	if err != nil {
		t.Fatalf("ListRepositoryVariables: %v", err)
	}

	if len(variables) != 2 {
		t.Errorf("expected 2 variables (limit), got %d", len(variables))
	}
	if requestCount != 1 {
		t.Errorf("expected 1 request (limit satisfied), got %d", requestCount)
	}
}

func TestCreateRepositoryVariableValidation(t *testing.T) {
	client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name          string
		workspace     string
		repoSlug      string
		input         CreateRepositoryVariableInput
		errorContains string
	}{
		{
			name:          "missing workspace",
			workspace:     "",
			repoSlug:      "repo",
			input:         CreateRepositoryVariableInput{Key: "VAR1", Value: "value"},
			errorContains: "workspace and repository slug are required",
		},
		{
			name:          "missing repo slug",
			workspace:     "workspace",
			repoSlug:      "",
			input:         CreateRepositoryVariableInput{Key: "VAR1", Value: "value"},
			errorContains: "workspace and repository slug are required",
		},
		{
			name:          "missing key",
			workspace:     "workspace",
			repoSlug:      "repo",
			input:         CreateRepositoryVariableInput{Key: "", Value: "value"},
			errorContains: "variable key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateRepositoryVariable(ctx, tt.workspace, tt.repoSlug, tt.input)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errorContains)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}

func TestCreateRepositoryVariable(t *testing.T) {
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		_ = json.NewDecoder(r.Body).Decode(&capturedBody)

		w.Header().Set("Content-Type", "application/json")
		resp := PipelineVariable{
			UUID:    "{new-uuid}",
			Key:     capturedBody["key"].(string),
			Secured: capturedBody["secured"].(bool),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	variable, err := client.CreateRepositoryVariable(ctx, "ws", "repo", CreateRepositoryVariableInput{
		Key:     "MY_VAR",
		Value:   "secret",
		Secured: true,
	})
	if err != nil {
		t.Fatalf("CreateRepositoryVariable: %v", err)
	}

	if variable.Key != "MY_VAR" {
		t.Errorf("expected key MY_VAR, got %s", variable.Key)
	}
	if !variable.Secured {
		t.Error("expected variable to be secured")
	}

	if capturedBody["key"] != "MY_VAR" {
		t.Errorf("expected request body key=MY_VAR, got %v", capturedBody["key"])
	}
	if capturedBody["value"] != "secret" {
		t.Errorf("expected request body value=secret, got %v", capturedBody["value"])
	}
	if capturedBody["secured"] != true {
		t.Errorf("expected request body secured=true, got %v", capturedBody["secured"])
	}
}

func TestUpdateRepositoryVariableValidation(t *testing.T) {
	client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name          string
		workspace     string
		repoSlug      string
		uuid          string
		input         UpdateRepositoryVariableInput
		errorContains string
	}{
		{
			name:          "missing workspace",
			workspace:     "",
			repoSlug:      "repo",
			uuid:          "{uuid}",
			input:         UpdateRepositoryVariableInput{Key: "VAR1", Value: "value"},
			errorContains: "workspace and repository slug are required",
		},
		{
			name:          "missing uuid",
			workspace:     "workspace",
			repoSlug:      "repo",
			uuid:          "",
			input:         UpdateRepositoryVariableInput{Key: "VAR1", Value: "value"},
			errorContains: "variable UUID is required",
		},
		{
			name:          "missing key",
			workspace:     "workspace",
			repoSlug:      "repo",
			uuid:          "{uuid}",
			input:         UpdateRepositoryVariableInput{Key: "", Value: "value"},
			errorContains: "variable key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.UpdateRepositoryVariable(ctx, tt.workspace, tt.repoSlug, tt.uuid, tt.input)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errorContains)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}

func TestDeleteRepositoryVariableValidation(t *testing.T) {
	client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name          string
		workspace     string
		repoSlug      string
		uuid          string
		errorContains string
	}{
		{
			name:          "missing workspace",
			workspace:     "",
			repoSlug:      "repo",
			uuid:          "{uuid}",
			errorContains: "workspace and repository slug are required",
		},
		{
			name:          "missing uuid",
			workspace:     "workspace",
			repoSlug:      "repo",
			uuid:          "",
			errorContains: "variable UUID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.DeleteRepositoryVariable(ctx, tt.workspace, tt.repoSlug, tt.uuid)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errorContains)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}

func TestListWorkspaceVariablesValidation(t *testing.T) {
	client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()

	_, err = client.ListWorkspaceVariables(ctx, "", VariableListOptions{})
	if err == nil {
		t.Fatal("expected error for missing workspace, got nil")
	}
	if !strings.Contains(err.Error(), "workspace is required") {
		t.Errorf("expected error about workspace, got %q", err.Error())
	}
}

func TestListDeploymentEnvironments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/environments") {
			t.Errorf("expected path to contain /environments, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := deploymentEnvironmentListPage{
			Values: []DeploymentEnvironment{
				{UUID: "{env-1}", Name: "production", Slug: "production"},
				{UUID: "{env-2}", Name: "staging", Slug: "staging"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	environments, err := client.ListDeploymentEnvironments(ctx, "ws", "repo")
	if err != nil {
		t.Fatalf("ListDeploymentEnvironments: %v", err)
	}

	if len(environments) != 2 {
		t.Errorf("expected 2 environments, got %d", len(environments))
	}
	if environments[0].Name != "production" {
		t.Errorf("expected first environment to be production, got %s", environments[0].Name)
	}
}

func TestListDeploymentVariables(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/deployments_config/environments/") {
			t.Errorf("expected path to contain /deployments_config/environments/, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := variableListPage{
			Values: []PipelineVariable{
				{UUID: "{var-1}", Key: "DEPLOY_VAR", Value: "value1"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	variables, err := client.ListDeploymentVariables(ctx, "ws", "repo", "{env-uuid}", VariableListOptions{})
	if err != nil {
		t.Fatalf("ListDeploymentVariables: %v", err)
	}

	if len(variables) != 1 {
		t.Errorf("expected 1 variable, got %d", len(variables))
	}
	if variables[0].Key != "DEPLOY_VAR" {
		t.Errorf("expected key DEPLOY_VAR, got %s", variables[0].Key)
	}
}
