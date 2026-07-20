package pipeline

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func TestDefaultPipelineLogStepIDUsesLastReturnedStep(t *testing.T) {
	steps := []bbcloud.PipelineStep{
		{UUID: "{11111111-1111-4111-8111-111111111111}", Name: "Build"},
		{UUID: "{22222222-2222-4222-8222-222222222222}", Name: "Deploy"},
	}

	got, ok := defaultPipelineLogStepID(steps)
	if !ok {
		t.Fatal("expected a default step")
	}
	if got != "{22222222-2222-4222-8222-222222222222}" {
		t.Fatalf("default step = %q, want deploy step", got)
	}
}

func TestDefaultPipelineLogStepIDEmpty(t *testing.T) {
	got, ok := defaultPipelineLogStepID(nil)
	if ok {
		t.Fatal("expected no default step")
	}
	if got != "" {
		t.Fatalf("default step = %q, want empty string", got)
	}
}

func waitOptsForTest() *waitOptions {
	return &waitOptions{
		Wait:        true,
		Interval:    time.Millisecond,
		MaxInterval: time.Millisecond,
		waitForPoll: func(context.Context, time.Duration) error { return nil },
	}
}

func pipelineInState(state, result, stage string) *bbcloud.Pipeline {
	p := &bbcloud.Pipeline{UUID: "{u}", BuildNumber: 7}
	p.State.Name = state
	p.State.Result.Name = result
	p.State.Stage.Name = stage
	p.Target.Ref.Name = "main"
	return p
}

func TestPipelineStatusHelpers(t *testing.T) {
	tests := []struct {
		name      string
		p         *bbcloud.Pipeline
		status    string
		completed bool
		succeeded bool
	}{
		{"in progress running", pipelineInState("IN_PROGRESS", "", "RUNNING"), "IN_PROGRESS (RUNNING)", false, false},
		{"pending", pipelineInState("PENDING", "", ""), "PENDING", false, false},
		{"completed successful", pipelineInState("COMPLETED", "SUCCESSFUL", ""), "COMPLETED SUCCESSFUL", true, true},
		{"completed failed", pipelineInState("COMPLETED", "FAILED", ""), "COMPLETED FAILED", true, false},
		{"completed stopped", pipelineInState("COMPLETED", "STOPPED", ""), "COMPLETED STOPPED", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pipelineStatus(tt.p); got != tt.status {
				t.Errorf("pipelineStatus = %q, want %q", got, tt.status)
			}
			if got := pipelineCompleted(tt.p); got != tt.completed {
				t.Errorf("pipelineCompleted = %v, want %v", got, tt.completed)
			}
			if got := pipelineSucceeded(tt.p); got != tt.succeeded {
				t.Errorf("pipelineSucceeded = %v, want %v", got, tt.succeeded)
			}
		})
	}
}

func TestWaitExitError(t *testing.T) {
	if err := waitExitError(pipelineInState("IN_PROGRESS", "", "RUNNING"), true); !errors.Is(err, cmdutil.ErrPending) {
		t.Errorf("timed out: got %v, want ErrPending", err)
	}
	if err := waitExitError(pipelineInState("COMPLETED", "FAILED", ""), false); !errors.Is(err, cmdutil.ErrSilent) {
		t.Errorf("failed: got %v, want ErrSilent", err)
	}
	if err := waitExitError(pipelineInState("COMPLETED", "SUCCESSFUL", ""), false); err != nil {
		t.Errorf("succeeded: got %v, want nil", err)
	}
}

func TestPollPipelineUntilDoneTransitions(t *testing.T) {
	var out, errOut bytes.Buffer
	ios := &iostreams.IOStreams{Out: &out, ErrOut: &errOut}

	states := []*bbcloud.Pipeline{
		pipelineInState("IN_PROGRESS", "", "RUNNING"),
		pipelineInState("COMPLETED", "SUCCESSFUL", ""),
	}
	calls := 0
	fetch := func(context.Context) (*bbcloud.Pipeline, error) {
		p := states[min(calls, len(states)-1)]
		calls++
		return p, nil
	}

	got, err := pollPipelineUntilDone(context.Background(), ios, waitOptsForTest(), false, pipelineInState("PENDING", "", ""), fetch)
	if err != nil {
		t.Fatalf("pollPipelineUntilDone: %v", err)
	}
	if !pipelineSucceeded(got) {
		t.Fatalf("final state = %s, want COMPLETED SUCCESSFUL", pipelineStatus(got))
	}
	if !strings.Contains(out.String(), "Pipeline #7 (main): PENDING") {
		t.Errorf("missing initial status in output: %q", out.String())
	}
	if !strings.Contains(out.String(), "COMPLETED SUCCESSFUL") {
		t.Errorf("missing final status in output: %q", out.String())
	}
}

func TestPollPipelineUntilDoneQuietProducesNoOutput(t *testing.T) {
	var out, errOut bytes.Buffer
	ios := &iostreams.IOStreams{Out: &out, ErrOut: &errOut}

	fetch := func(context.Context) (*bbcloud.Pipeline, error) {
		return pipelineInState("COMPLETED", "SUCCESSFUL", ""), nil
	}
	if _, err := pollPipelineUntilDone(context.Background(), ios, waitOptsForTest(), true, pipelineInState("PENDING", "", ""), fetch); err != nil {
		t.Fatalf("pollPipelineUntilDone: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("quiet poll wrote to stdout: %q", out.String())
	}
}

func TestPollPipelineUntilDoneFetchErrorsExhaust(t *testing.T) {
	var out, errOut bytes.Buffer
	ios := &iostreams.IOStreams{Out: &out, ErrOut: &errOut}

	fetch := func(context.Context) (*bbcloud.Pipeline, error) {
		return nil, errors.New("boom")
	}
	last, err := pollPipelineUntilDone(context.Background(), ios, waitOptsForTest(), true, pipelineInState("PENDING", "", ""), fetch)
	if err == nil || !strings.Contains(err.Error(), "after 3 attempts") {
		t.Fatalf("err = %v, want exhaustion after 3 attempts", err)
	}
	if last == nil || !strings.EqualFold(last.State.Name, "PENDING") {
		t.Fatalf("last = %+v, want the initial pipeline", last)
	}
	if got := strings.Count(errOut.String(), "Warning: error fetching pipeline"); got != 2 {
		t.Errorf("warnings printed = %d, want 2 (third attempt errors out)", got)
	}
}

func TestPollPipelineUntilDoneCancelledDuringWait(t *testing.T) {
	var out, errOut bytes.Buffer
	ios := &iostreams.IOStreams{Out: &out, ErrOut: &errOut}

	w := waitOptsForTest()
	w.waitForPoll = func(context.Context, time.Duration) error { return context.Canceled }

	fetch := func(context.Context) (*bbcloud.Pipeline, error) {
		t.Fatal("fetch should not be called after cancelled wait")
		return nil, nil
	}
	last, err := pollPipelineUntilDone(context.Background(), ios, w, true, pipelineInState("PENDING", "", ""), fetch)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if last == nil {
		t.Fatal("last should be the initial pipeline")
	}
}

func TestValidateWaitFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"no wait no flags", nil, ""},
		{"wait with defaults", []string{"--wait"}, ""},
		{"interval requires wait", []string{"--interval", "5s"}, "--interval requires --wait"},
		{"max-interval requires wait", []string{"--max-interval", "1m"}, "--max-interval requires --wait"},
		{"timeout requires wait", []string{"--timeout", "1m"}, "--timeout requires --wait"},
		{"max below interval", []string{"--wait", "--interval", "1m", "--max-interval", "5s"}, "--max-interval must be >= --interval"},
		{"negative interval", []string{"--wait", "--interval", "-5s"}, "--interval must be positive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "x"}
			var w waitOptions
			addWaitFlags(cmd, &w, "wait")
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("ParseFlags: %v", err)
			}
			err := validateWaitFlags(cmd, &w)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateWaitFlags: %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("validateWaitFlags = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

// Regression test for the steps-fetch context wiring: cancelling the command
// context while the steps request is in flight must abort it immediately
// (a detached context would hang until the 10s fetch deadline instead).
func TestPipelineViewStepsFetchHonorsCommandCancellation(t *testing.T) {
	pipelineJSON := `{"uuid":"{a1b2c3d4-e5f6-4890-abcd-ef1234567890}","build_number":1,"state":{"name":"COMPLETED","result":{"name":"SUCCESSFUL"}},"target":{"ref":{"name":"main"}}}`
	stepsReached := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/steps") {
			select {
			case stepsReached <- struct{}{}:
			default:
			}
			<-r.Context().Done() // hold the request open until the client aborts
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(pipelineJSON))
	}))
	defer srv.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "cloud", Workspace: "ws", DefaultRepo: "repo"},
		},
		Hosts: map[string]*config.Host{
			"cloud": {Kind: "cloud", BaseURL: srv.URL, Username: "u", Token: "t"},
		},
	}

	var out, errOut bytes.Buffer
	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: &out, ErrOut: &errOut},
		Config:         func() (*config.Config, error) { return cfg, nil },
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-stepsReached
		cancel()
	}()

	cmd := newViewCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"1"})

	start := time.Now()
	err := cmd.ExecuteContext(ctx)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("view took %v; steps fetch ignored command cancellation", elapsed)
	}
}
