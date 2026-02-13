package bbcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListIssueAttachments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/issues/42/attachments") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := issueAttachmentListPage{
			Values: []IssueAttachment{
				{Name: "screenshot.png"},
				{Name: "log.txt"},
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
	attachments, err := client.ListIssueAttachments(ctx, "workspace", "repo", 42)
	if err != nil {
		t.Fatalf("ListIssueAttachments: %v", err)
	}

	if len(attachments) != 2 {
		t.Errorf("expected 2 attachments, got %d", len(attachments))
	}
	if attachments[0].Name != "screenshot.png" {
		t.Errorf("expected first attachment to be screenshot.png, got %s", attachments[0].Name)
	}
}

func TestListIssueAttachmentsPagination(t *testing.T) {
	var requestCount int
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		switch requestCount {
		case 1:
			resp := issueAttachmentListPage{
				Values: []IssueAttachment{{Name: "file1.txt"}},
				Next:   serverURL + "/repositories/ws/repo/issues/42/attachments?page=2",
			}
			_ = json.NewEncoder(w).Encode(resp)
		case 2:
			resp := issueAttachmentListPage{
				Values: []IssueAttachment{{Name: "file2.txt"}},
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
	attachments, err := client.ListIssueAttachments(ctx, "ws", "repo", 42)
	if err != nil {
		t.Fatalf("ListIssueAttachments: %v", err)
	}

	if len(attachments) != 2 {
		t.Errorf("expected 2 attachments, got %d", len(attachments))
	}
	if requestCount != 2 {
		t.Errorf("expected 2 requests, got %d", requestCount)
	}
}

func TestListIssueAttachmentsValidation(t *testing.T) {
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
			_, err := client.ListIssueAttachments(ctx, tt.workspace, tt.repoSlug, 42)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errorContains)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}

func TestUploadIssueAttachment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/issues/42/attachments") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		contentType := r.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "multipart/form-data") {
			t.Errorf("expected multipart/form-data content type, got %s", contentType)
		}

		// Parse multipart form to verify file was sent
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}

		file, header, err := r.FormFile("files")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer func() { _ = file.Close() }()

		if header.Filename != "test.txt" {
			t.Errorf("expected filename test.txt, got %s", header.Filename)
		}

		content, _ := io.ReadAll(file)
		if string(content) != "hello world" {
			t.Errorf("expected content 'hello world', got %q", string(content))
		}

		w.Header().Set("Content-Type", "application/json")
		resp := []IssueAttachment{{Name: "test.txt"}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	attachment, err := client.UploadIssueAttachment(ctx, "workspace", "repo", 42, "test.txt", strings.NewReader("hello world"))
	if err != nil {
		t.Fatalf("UploadIssueAttachment: %v", err)
	}

	if attachment.Name != "test.txt" {
		t.Errorf("expected attachment name test.txt, got %s", attachment.Name)
	}
}

func TestUploadIssueAttachmentValidation(t *testing.T) {
	client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name          string
		workspace     string
		repoSlug      string
		filename      string
		errorContains string
	}{
		{
			name:          "missing workspace",
			workspace:     "",
			repoSlug:      "repo",
			filename:      "file.txt",
			errorContains: "workspace and repository slug are required",
		},
		{
			name:          "missing repo slug",
			workspace:     "workspace",
			repoSlug:      "",
			filename:      "file.txt",
			errorContains: "workspace and repository slug are required",
		},
		{
			name:          "missing filename",
			workspace:     "workspace",
			repoSlug:      "repo",
			filename:      "",
			errorContains: "filename is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.UploadIssueAttachment(ctx, tt.workspace, tt.repoSlug, 42, tt.filename, strings.NewReader("test"))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errorContains)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}

func TestDownloadIssueAttachment(t *testing.T) {
	expectedContent := []byte("file content here")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/issues/42/attachments/screenshot.png") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(expectedContent)
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	var buf bytes.Buffer
	err = client.DownloadIssueAttachment(ctx, "workspace", "repo", 42, "screenshot.png", &buf)
	if err != nil {
		t.Fatalf("DownloadIssueAttachment: %v", err)
	}

	if !bytes.Equal(buf.Bytes(), expectedContent) {
		t.Errorf("expected %q, got %q", expectedContent, buf.Bytes())
	}
}

func TestDownloadIssueAttachmentValidation(t *testing.T) {
	client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name          string
		workspace     string
		repoSlug      string
		filename      string
		errorContains string
	}{
		{
			name:          "missing workspace",
			workspace:     "",
			repoSlug:      "repo",
			filename:      "file.txt",
			errorContains: "workspace and repository slug are required",
		},
		{
			name:          "missing repo slug",
			workspace:     "workspace",
			repoSlug:      "",
			filename:      "file.txt",
			errorContains: "workspace and repository slug are required",
		},
		{
			name:          "missing filename",
			workspace:     "workspace",
			repoSlug:      "repo",
			filename:      "",
			errorContains: "filename is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := client.DownloadIssueAttachment(ctx, tt.workspace, tt.repoSlug, 42, tt.filename, &buf)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errorContains)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}

func TestDeleteIssueAttachment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/issues/42/attachments/old-file.txt") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	err = client.DeleteIssueAttachment(ctx, "workspace", "repo", 42, "old-file.txt")
	if err != nil {
		t.Fatalf("DeleteIssueAttachment: %v", err)
	}
}

func TestDeleteIssueAttachmentValidation(t *testing.T) {
	client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name          string
		workspace     string
		repoSlug      string
		filename      string
		errorContains string
	}{
		{
			name:          "missing workspace",
			workspace:     "",
			repoSlug:      "repo",
			filename:      "file.txt",
			errorContains: "workspace and repository slug are required",
		},
		{
			name:          "missing repo slug",
			workspace:     "workspace",
			repoSlug:      "",
			filename:      "file.txt",
			errorContains: "workspace and repository slug are required",
		},
		{
			name:          "missing filename",
			workspace:     "workspace",
			repoSlug:      "repo",
			filename:      "",
			errorContains: "filename is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.DeleteIssueAttachment(ctx, tt.workspace, tt.repoSlug, 42, tt.filename)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errorContains)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}

// --- API Error Response Tests ---

func TestListIssueAttachmentsAPIError(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		body          string
		errorContains string
	}{
		{
			name:          "not found",
			status:        http.StatusNotFound,
			body:          `{"error": {"message": "Issue not found"}}`,
			errorContains: "404",
		},
		{
			name:          "forbidden",
			status:        http.StatusForbidden,
			body:          `{"error": {"message": "Access denied"}}`,
			errorContains: "403",
		},
		{
			name:          "structured error",
			status:        http.StatusBadRequest,
			body:          `{"errors": [{"message": "Invalid issue ID"}]}`,
			errorContains: "Invalid issue ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			client, err := New(Options{BaseURL: server.URL})
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			ctx := context.Background()
			_, err = client.ListIssueAttachments(ctx, "workspace", "repo", 42)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}

func TestUploadIssueAttachmentAPIError(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		body          string
		errorContains string
	}{
		{
			name:          "file too large",
			status:        http.StatusRequestEntityTooLarge,
			body:          `{"errors": [{"message": "File exceeds maximum size of 10MB"}]}`,
			errorContains: "File exceeds maximum size",
		},
		{
			name:          "unsupported media type",
			status:        http.StatusUnsupportedMediaType,
			body:          `{"errors": [{"message": "File type not allowed"}]}`,
			errorContains: "File type not allowed",
		},
		{
			name:          "issue not found",
			status:        http.StatusNotFound,
			body:          `{"errors": [{"message": "Issue does not exist"}]}`,
			errorContains: "Issue does not exist",
		},
		{
			name:          "permission denied",
			status:        http.StatusForbidden,
			body:          `{"errors": [{"message": "You don't have permission to add attachments"}]}`,
			errorContains: "permission",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			client, err := New(Options{BaseURL: server.URL})
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			ctx := context.Background()
			_, err = client.UploadIssueAttachment(ctx, "workspace", "repo", 42, "test.txt", strings.NewReader("content"))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}

func TestDownloadIssueAttachmentAPIError(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		body          string
		errorContains string
	}{
		{
			name:          "attachment not found",
			status:        http.StatusNotFound,
			body:          `{"errors": [{"message": "Attachment not found"}]}`,
			errorContains: "Attachment not found",
		},
		{
			name:          "forbidden",
			status:        http.StatusForbidden,
			body:          `{"errors": [{"message": "Access denied"}]}`,
			errorContains: "Access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			client, err := New(Options{BaseURL: server.URL})
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			ctx := context.Background()
			var buf bytes.Buffer
			err = client.DownloadIssueAttachment(ctx, "workspace", "repo", 42, "missing.txt", &buf)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}

func TestDeleteIssueAttachmentAPIError(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		body          string
		errorContains string
	}{
		{
			name:          "attachment not found",
			status:        http.StatusNotFound,
			body:          `{"errors": [{"message": "Attachment not found"}]}`,
			errorContains: "Attachment not found",
		},
		{
			name:          "forbidden",
			status:        http.StatusForbidden,
			body:          `{"errors": [{"message": "You don't have permission to delete attachments"}]}`,
			errorContains: "permission",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			client, err := New(Options{BaseURL: server.URL})
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			ctx := context.Background()
			err = client.DeleteIssueAttachment(ctx, "workspace", "repo", 42, "file.txt")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}
