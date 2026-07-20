package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// TestServeCommandStdoutCarriesOnlyJSONRPC drives the real `mcp serve`
// command path — Cobra execution, context resolution, startup banner, server
// run — over an injected transport standing in for stdio. Every protocol
// frame must be pure JSON-RPC, and the banner must appear only on stderr.
func TestServeCommandStdoutCarriesOnlyJSONRPC(t *testing.T) {
	cfg := &config.Config{
		ActiveContext: "work",
		Contexts: map[string]*config.Context{
			"work": {Host: "dc-host", ProjectKey: "PROJ"},
		},
		Hosts: map[string]*config.Host{
			"dc-host": {Kind: "dc", BaseURL: "https://bitbucket.example.com", Username: "u", Token: "t"},
		},
	}

	var stderr bytes.Buffer
	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: &failIfWritten{t: t}, ErrOut: &stderr},
		Config:         func() (*config.Config, error) { return cfg, nil },
	}

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	cmd := newServeCmdWithTransport(f, &sdk.IOTransport{Reader: stdinR, Writer: stdoutW})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	runDone := make(chan error, 1)
	go func() { runDone <- cmd.ExecuteContext(ctx) }()

	scanner := bufio.NewScanner(stdoutR)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)

	send := func(frame string) {
		t.Helper()
		if _, err := io.WriteString(stdinW, frame+"\n"); err != nil {
			t.Fatalf("write frame: %v", err)
		}
	}
	recv := func() string {
		t.Helper()
		lineCh := make(chan string, 1)
		errCh := make(chan error, 1)
		go func() {
			if scanner.Scan() {
				lineCh <- scanner.Text()
				return
			}
			errCh <- scanner.Err()
		}()
		select {
		case line := <-lineCh:
			return line
		case err := <-errCh:
			t.Fatalf("protocol stream closed early: %v", err)
		case <-ctx.Done():
			t.Fatal("timeout waiting for a response frame")
		}
		return ""
	}

	send(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"purity-test","version":"0"}}}`)
	initResp := recv()
	send(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	send(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	listResp := recv()
	send(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"bkt_get_context"}}`)
	callResp := recv()

	_ = stdinW.Close()
	select {
	case <-runDone:
	case <-ctx.Done():
		t.Fatal("serve command did not stop after stdin close")
	}

	for i, line := range []string{initResp, listResp, callResp} {
		var msg struct {
			JSONRPC string          `json:"jsonrpc"`
			Result  json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			t.Fatalf("protocol frame %d is not JSON-RPC (stdout impurity): %v\n%q", i+1, err, line)
		}
		if msg.JSONRPC != "2.0" {
			t.Fatalf("protocol frame %d lacks jsonrpc 2.0 envelope: %q", i+1, line)
		}
	}
	if !strings.Contains(listResp, "bkt_get_context") {
		t.Fatalf("tools/list response missing bkt_get_context: %q", listResp)
	}
	if !strings.Contains(listResp, `"readOnlyHint":true`) {
		t.Fatalf("tools/list must advertise readOnlyHint: %q", listResp)
	}
	if !strings.Contains(callResp, "structuredContent") {
		t.Fatalf("tools/call response missing structured content: %q", callResp)
	}
	if strings.Contains(callResp, `"token"`) {
		t.Fatalf("tools/call response contains credential-looking field: %q", callResp)
	}

	banner := stderr.String()
	if !strings.Contains(banner, "bkt mcp serve: context work, platform dc, host dc-host") {
		t.Fatalf("startup banner missing or wrong on stderr: %q", banner)
	}
}

// failIfWritten fails the test on any write: the factory's stdout stream must
// never be used by the serve path — protocol frames flow only through the
// transport, and diagnostics only through stderr.
type failIfWritten struct{ t *testing.T }

func (f *failIfWritten) Write(p []byte) (int, error) {
	f.t.Errorf("serve path wrote %q to the factory stdout stream; stdout is reserved for the protocol", p)
	return len(p), nil
}
