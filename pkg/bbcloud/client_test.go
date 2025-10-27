package bbcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestListPipelinesPaginates(t *testing.T) {
	var hits int32
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")

		switch count {
		case 1:
			if r.URL.Query().Get("pagelen") == "" {
				t.Fatalf("expected pagelen query in first request")
			}
			payload := PipelinePage{
				Values: []Pipeline{{UUID: "1"}, {UUID: "2"}},
				Next:   serverURL + "/repositories/work/repo/pipelines/?pagelen=20&page=2",
			}
			_ = json.NewEncoder(w).Encode(payload)
		case 2:
			payload := PipelinePage{
				Values: []Pipeline{{UUID: "3"}},
			}
			_ = json.NewEncoder(w).Encode(payload)
		default:
			t.Fatalf("unexpected extra request %d", count)
		}
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	pipelines, err := client.ListPipelines(ctx, "work", "repo", 0)
	if err != nil {
		t.Fatalf("ListPipelines: %v", err)
	}

	if len(pipelines) != 3 {
		t.Fatalf("expected 3 pipelines, got %d", len(pipelines))
	}
	if hits != 2 {
		t.Fatalf("expected 2 requests, got %d", hits)
	}
}

func TestListPipelinesRespectsLimit(t *testing.T) {
	var hits int32
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")

		if count == 1 {
			payload := PipelinePage{
				Values: []Pipeline{{UUID: "1"}, {UUID: "2"}},
				Next:   serverURL + "/repositories/work/repo/pipelines/?pagelen=20&page=2",
			}
			_ = json.NewEncoder(w).Encode(payload)
			return
		}

		t.Fatalf("unexpected second request when limit satisfied")
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	pipelines, err := client.ListPipelines(ctx, "work", "repo", 1)
	if err != nil {
		t.Fatalf("ListPipelines: %v", err)
	}

	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}
	if hits != 1 {
		t.Fatalf("expected 1 request, got %d", hits)
	}
}
