package bbcloud

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUploadDownload(t *testing.T) {
	var gotName, gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/ws/repo/downloads" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}

		file, header, err := r.FormFile("files")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer func() { _ = file.Close() }()
		data, _ := io.ReadAll(file)
		gotName, gotBody = header.Filename, string(data)

		// Bitbucket answers 201 with an empty body.
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := client.UploadDownload(context.Background(), "ws", "repo", "shot.png", strings.NewReader("png-bytes")); err != nil {
		t.Fatalf("UploadDownload: %v", err)
	}
	if gotName != "shot.png" {
		t.Errorf("filename = %q, want shot.png", gotName)
	}
	if gotBody != "png-bytes" {
		t.Errorf("body = %q, want png-bytes", gotBody)
	}
}

func TestUploadDownloadValidation(t *testing.T) {
	client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	tests := []struct {
		name, workspace, repo, filename, wantErr string
	}{
		{"missing workspace", "", "repo", "a.png", "workspace and repository slug are required"},
		{"missing repo", "ws", "", "a.png", "workspace and repository slug are required"},
		{"missing filename", "ws", "repo", "", "filename is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.UploadDownload(context.Background(), tt.workspace, tt.repo, tt.filename, strings.NewReader("x"))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestListDownloadsPagination(t *testing.T) {
	var requestCount int
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		switch requestCount {
		case 1:
			_ = json.NewEncoder(w).Encode(downloadListPage{
				Values: []Download{{Name: "a.png", Size: 10}},
				Next:   serverURL + "/repositories/ws/repo/downloads?page=2",
			})
		case 2:
			_ = json.NewEncoder(w).Encode(downloadListPage{
				Values: []Download{{Name: "b.png", Size: 20}},
			})
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

	downloads, err := client.ListDownloads(context.Background(), "ws", "repo")
	if err != nil {
		t.Fatalf("ListDownloads: %v", err)
	}
	if len(downloads) != 2 || downloads[0].Name != "a.png" || downloads[1].Name != "b.png" {
		t.Errorf("unexpected downloads: %+v", downloads)
	}
}

func TestDownloadURL(t *testing.T) {
	got := DownloadURL("my-ws", "my-repo", "before shot.png")
	want := "https://bitbucket.org/my-ws/my-repo/downloads/before%20shot.png"
	if got != want {
		t.Errorf("DownloadURL = %q, want %q", got, want)
	}
}
