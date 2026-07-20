package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/avivsinai/bitbucket-cli/internal/mcpserver"
)

// Stdout must carry only JSON-RPC protocol frames: any stray print corrupts
// every MCP client. Drive the server over raw stdio frames in lockstep and
// verify every stdout line parses as a JSON-RPC message.
func TestServeStdoutCarriesOnlyJSONRPC(t *testing.T) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	snap := &mcpserver.Snapshot{ContextName: "work", Platform: "dc", HostLabel: "h", DefaultScope: "PROJ"}
	server := mcpserver.New(snap, "test")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	runDone := make(chan error, 1)
	go func() {
		runDone <- server.Run(ctx, &sdk.IOTransport{Reader: stdinR, Writer: stdoutW})
	}()

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
			t.Fatalf("stdout closed early: %v", err)
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
		t.Fatal("server did not stop after stdin close")
	}

	for i, line := range []string{initResp, listResp, callResp} {
		var msg struct {
			JSONRPC string          `json:"jsonrpc"`
			Result  json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			t.Fatalf("stdout frame %d is not JSON-RPC (stdout impurity): %v\n%q", i+1, err, line)
		}
		if msg.JSONRPC != "2.0" {
			t.Fatalf("stdout frame %d lacks jsonrpc 2.0 envelope: %q", i+1, line)
		}
	}
	if !strings.Contains(listResp, "bkt_get_context") {
		t.Fatalf("tools/list response missing bkt_get_context: %q", listResp)
	}
	if !strings.Contains(callResp, "structuredContent") {
		t.Fatalf("tools/call response missing structured content: %q", callResp)
	}
	if strings.Contains(callResp, `"token"`) {
		t.Fatalf("tools/call response contains credential-looking field: %q", callResp)
	}
}
