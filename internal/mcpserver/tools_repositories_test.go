package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

type fakeRepositoryBackend struct {
	listScope string
	listLimit int
	listItems []Repository
	listMore  bool
	listErr   error

	getLocator RepositoryRef
	getResult  Repository
	getErr     error
	getCalls   int

	listEntered  chan struct{}
	listCanceled chan struct{}
}

func (f *fakeRepositoryBackend) listRepositories(ctx context.Context, scope string, limit int) ([]Repository, bool, error) {
	f.listScope = scope
	f.listLimit = limit
	if f.listEntered != nil {
		close(f.listEntered)
		<-ctx.Done()
		close(f.listCanceled)
		return nil, false, ctx.Err()
	}
	return f.listItems, f.listMore, f.listErr
}

func (f *fakeRepositoryBackend) getRepository(_ context.Context, locator RepositoryRef) (Repository, error) {
	f.getCalls++
	f.getLocator = locator
	return f.getResult, f.getErr
}

func (f *fakeRepositoryBackend) listPullRequests(context.Context, RepositoryRef, string, string, int) ([]PullRequest, bool, error) {
	return []PullRequest{}, false, nil
}

func (f *fakeRepositoryBackend) listMyPullRequests(context.Context, string, string, string, int) ([]PullRequest, bool, error) {
	return []PullRequest{}, false, nil
}

func TestListRepositoriesToolUsesFrozenScopeAndBoundedPage(t *testing.T) {
	backend := &fakeRepositoryBackend{
		listItems: []Repository{
			{Scope: "PROJ", Slug: "one", Name: "One"},
			{Scope: "PROJ", Slug: "two", Name: "Two"},
		},
		listMore: true,
	}
	snap := &Snapshot{Platform: "dc", HostLabel: "dc", DefaultScope: "PROJ"}
	session := connectPair(t, newServer(snap, "test", backend))

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "bkt_list_repositories",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool error: %+v", res.Content)
	}
	if backend.listScope != "PROJ" || backend.listLimit != DefaultListLimit {
		t.Fatalf("backend args = scope %q limit %d", backend.listScope, backend.listLimit)
	}

	var got ListEnvelope[Repository]
	decodeStructuredContent(t, res, &got)
	if got.Count != 2 || got.Limit != DefaultListLimit || !got.Truncated || len(got.Items) != 2 {
		t.Fatalf("result = %+v", got)
	}
}

func TestListRepositoriesToolPreservesEmptyUpstreamContinuation(t *testing.T) {
	backend := &fakeRepositoryBackend{listMore: true}
	session := connectPair(t, newServer(
		&Snapshot{Platform: "dc", HostLabel: "dc", DefaultScope: "PROJ"},
		"test",
		backend,
	))

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "bkt_list_repositories",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	var got ListEnvelope[Repository]
	decodeStructuredContent(t, res, &got)
	if got.Items == nil || got.Count != 0 || !got.Truncated {
		t.Fatalf("result = %#v, want non-nil empty items with continuation", got)
	}
}

func TestListRepositoriesToolRejectsMissingScope(t *testing.T) {
	backend := &fakeRepositoryBackend{}
	session := connectPair(t, newServer(&Snapshot{Platform: "cloud", HostLabel: "cloud"}, "test", backend))

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "bkt_list_repositories",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	got := decodeStructuredToolError(t, res)
	if got.Code != ErrorInvalidInput || got.Retryable {
		t.Fatalf("error = %+v", got)
	}
}

func TestListRepositoriesToolPropagatesCancellation(t *testing.T) {
	backend := &fakeRepositoryBackend{
		listEntered:  make(chan struct{}),
		listCanceled: make(chan struct{}),
	}
	session := connectPair(t, newServer(&Snapshot{Platform: "dc", HostLabel: "dc", DefaultScope: "PROJ"}, "test", backend))
	ctx, cancel := context.WithCancel(context.Background())
	callErr := make(chan error, 1)
	go func() {
		_, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "bkt_list_repositories", Arguments: map[string]any{}})
		callErr <- err
	}()

	select {
	case <-backend.listEntered:
	case <-time.After(time.Second):
		t.Fatal("backend did not receive tool call")
	}
	cancel()
	select {
	case <-backend.listCanceled:
	case <-time.After(time.Second):
		t.Fatal("backend context was not canceled")
	}
	select {
	case err := <-callErr:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("CallTool error = %T %v, want context.Canceled", err, err)
		}
	case <-time.After(time.Second):
		t.Fatal("tool call did not return after cancellation")
	}
}

func TestGetRepositoryToolResolvesDefaultAndRejectsPartialLocator(t *testing.T) {
	backend := &fakeRepositoryBackend{
		getResult: Repository{Scope: "PROJ", Slug: "api", Name: "API"},
	}
	snap := &Snapshot{Platform: "dc", HostLabel: "dc", DefaultScope: "PROJ", DefaultRepo: "api"}
	session := connectPair(t, newServer(snap, "test", backend))

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "bkt_get_repository",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("default CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("default tool error: %+v", res.Content)
	}
	if backend.getLocator != (RepositoryRef{Scope: "PROJ", Slug: "api"}) {
		t.Fatalf("default locator = %+v", backend.getLocator)
	}
	var got Repository
	decodeStructuredContent(t, res, &got)
	if got.Slug != "api" {
		t.Fatalf("repository = %+v", got)
	}

	res, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "bkt_get_repository",
		Arguments: map[string]any{
			"locator": map[string]any{"scope": "OTHER"},
		},
	})
	if err != nil {
		t.Fatalf("partial locator CallTool: %v", err)
	}
	toolErr := decodeStructuredToolError(t, res)
	if toolErr.Code != ErrorInvalidInput || backend.getCalls != 1 {
		t.Fatalf("partial locator error = %+v, backend calls = %d", toolErr, backend.getCalls)
	}
}

func TestMapToolErrorUsesTypedHTTPStatusAndRedactsUpstreamText(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		wantCode      ErrorCode
		wantRetryable bool
	}{
		{name: "unauthorized", err: &httpx.HTTPError{StatusCode: http.StatusUnauthorized}, wantCode: ErrorAuthFailed},
		{name: "forbidden", err: &httpx.HTTPError{StatusCode: http.StatusForbidden}, wantCode: ErrorAuthFailed},
		{name: "not found", err: &httpx.HTTPError{StatusCode: http.StatusNotFound}, wantCode: ErrorNotFound},
		{name: "rate limited", err: &httpx.HTTPError{StatusCode: http.StatusTooManyRequests}, wantCode: ErrorRateLimited, wantRetryable: true},
		{name: "server error", err: &httpx.HTTPError{StatusCode: http.StatusBadGateway}, wantCode: ErrorUpstream, wantRetryable: true},
		{name: "other client error", err: &httpx.HTTPError{StatusCode: http.StatusBadRequest}, wantCode: ErrorUpstream},
		{name: "transport error", err: &url.Error{Op: "GET", URL: "https://example.test?token=secret", Err: &net.DNSError{IsTimeout: true}}, wantCode: ErrorUpstream, wantRetryable: true},
		{name: "permanent network error", err: &url.Error{Op: "GET", URL: "https://example.test?token=secret", Err: errors.New("x509: certificate signed by unknown authority")}, wantCode: ErrorUpstream},
		{name: "local adaptation error", err: errors.New("request https://example.test?token=secret failed"), wantCode: ErrorUpstream},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapped := mapToolError(tt.err)
			var got Error
			if err := json.Unmarshal([]byte(mapped.Error()), &got); err != nil {
				t.Fatalf("mapped error is not JSON: %v", err)
			}
			if got.Code != tt.wantCode || got.Retryable != tt.wantRetryable {
				t.Fatalf("mapped = %+v, want code=%q retryable=%v", got, tt.wantCode, tt.wantRetryable)
			}
			if strings.Contains(mapped.Error(), "secret") || strings.Contains(mapped.Error(), "example.test") {
				t.Fatalf("mapped error leaked upstream details: %s", mapped)
			}
		})
	}
}

func TestMapToolErrorPreservesStructuredPayloadWithoutWrapperText(t *testing.T) {
	want := newToolError(ErrorInvalidInput, "frozen context is missing required identity", false)
	mapped := mapToolError(fmt.Errorf("debug secret wrapper: %w", want))
	var got *structuredToolError
	if !errors.As(mapped, &got) {
		t.Fatalf("mapped error = %T %v, want structuredToolError", mapped, mapped)
	}
	if got.payload.Code != ErrorInvalidInput || got.payload.Retryable {
		t.Fatalf("mapped payload = %+v", got.payload)
	}
	if strings.Contains(mapped.Error(), "secret") {
		t.Fatalf("mapped error leaked wrapper text: %s", mapped)
	}
}

func TestMapToolErrorPreservesContextCancellation(t *testing.T) {
	for _, want := range []error{context.Canceled, context.DeadlineExceeded} {
		if got := mapToolError(want); !errors.Is(got, want) {
			t.Fatalf("mapToolError(%v) = %T %v, want original context error", want, got, got)
		}
	}
}

func TestGetRepositoryToolEmitsOnlyRedactedErrorDTOText(t *testing.T) {
	backend := &fakeRepositoryBackend{
		getErr: fmt.Errorf("GET https://example.test/repo?token=secret: %w", &httpx.HTTPError{StatusCode: http.StatusNotFound}),
	}
	snap := &Snapshot{Platform: "dc", HostLabel: "dc", DefaultScope: "PROJ", DefaultRepo: "api"}
	session := connectPair(t, newServer(snap, "test", backend))

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "bkt_get_repository",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	got := decodeStructuredToolError(t, res)
	if got.Code != ErrorNotFound || got.Retryable {
		t.Fatalf("error = %+v", got)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	want, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if text != string(want) {
		t.Fatalf("error text = %q, want only Error DTO JSON %q", text, want)
	}
	if strings.Contains(text, "secret") || strings.Contains(text, "example.test") {
		t.Fatalf("tool error leaked upstream details: %s", text)
	}
}

func decodeStructuredContent(t *testing.T, res *mcp.CallToolResult, out any) {
	t.Helper()
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("decode structured content: %v\n%s", err, raw)
	}
}

func decodeStructuredToolError(t *testing.T, res *mcp.CallToolResult) Error {
	t.Helper()
	if !res.IsError {
		t.Fatalf("result IsError=false: %+v", res)
	}
	if len(res.Content) != 1 {
		t.Fatalf("error content = %+v", res.Content)
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("error content type = %T", res.Content[0])
	}
	var got Error
	if err := json.Unmarshal([]byte(text.Text), &got); err != nil {
		t.Fatalf("error is not Error JSON: %v\n%s", err, text.Text)
	}
	return got
}
