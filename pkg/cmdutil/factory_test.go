package cmdutil

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/browser"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
	"github.com/avivsinai/bitbucket-cli/pkg/pager"
	"github.com/avivsinai/bitbucket-cli/pkg/progress"
	"github.com/avivsinai/bitbucket-cli/pkg/prompter"
)

type stubBrowser struct{}

func (stubBrowser) Open(string) error { return nil }

type stubPager struct{}

func (stubPager) Enabled() bool                  { return true }
func (stubPager) Start() (io.WriteCloser, error) { return nopCloser{Writer: io.Discard}, nil }
func (stubPager) Stop() error                    { return nil }

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

type stubPrompter struct{}

func (stubPrompter) Input(string, string) (string, error) { return "", nil }
func (stubPrompter) Password(string) (string, error)      { return "", nil }
func (stubPrompter) Confirm(string, bool) (bool, error)   { return false, nil }

type stubSpinner struct{}

func (stubSpinner) Start(string) {}
func (stubSpinner) Stop(string)  {}
func (stubSpinner) Fail(string)  {}

func testIOStreams() (*iostreams.IOStreams, *bytes.Buffer, *bytes.Buffer) {
	var out, errOut bytes.Buffer
	return &iostreams.IOStreams{
		In:     io.NopCloser(strings.NewReader("")),
		Out:    &out,
		ErrOut: &errOut,
	}, &out, &errOut
}

func TestFactoryResolveConfigCachesSuccess(t *testing.T) {
	want := &config.Config{ActiveContext: "dev"}
	var calls int
	f := &Factory{
		Config: func() (*config.Config, error) {
			calls++
			return want, nil
		},
	}

	got1, err := f.ResolveConfig()
	if err != nil {
		t.Fatalf("ResolveConfig first call: %v", err)
	}
	got2, err := f.ResolveConfig()
	if err != nil {
		t.Fatalf("ResolveConfig second call: %v", err)
	}
	if calls != 1 {
		t.Fatalf("Config called %d times, want 1", calls)
	}
	if got1 != want || got2 != want {
		t.Fatalf("ResolveConfig did not return cached config pointer")
	}
}

func TestFactoryResolveConfigFallsBackToLoad(t *testing.T) {
	t.Setenv("BKT_CONFIG_DIR", t.TempDir())

	f := &Factory{}
	got, err := f.ResolveConfig()
	if err != nil {
		t.Fatalf("ResolveConfig fallback: %v", err)
	}
	if got == nil {
		t.Fatal("ResolveConfig fallback returned nil config")
	}
	if got.Path() == "" {
		t.Fatal("ResolveConfig fallback did not set config path")
	}
}

func TestFactoryResolveConfigCachesError(t *testing.T) {
	wantErr := errors.New("config boom")
	var calls int
	f := &Factory{
		Config: func() (*config.Config, error) {
			calls++
			return nil, wantErr
		},
	}

	_, err1 := f.ResolveConfig()
	_, err2 := f.ResolveConfig()
	if !errors.Is(err1, wantErr) || !errors.Is(err2, wantErr) {
		t.Fatalf("ResolveConfig errors = %v / %v, want %v", err1, err2, wantErr)
	}
	if calls != 1 {
		t.Fatalf("Config called %d times, want 1", calls)
	}
}

func TestFactoryStreamsCachesProvidedIOStreams(t *testing.T) {
	ios, _, _ := testIOStreams()
	f := &Factory{IOStreams: ios}

	got1, err := f.Streams()
	if err != nil {
		t.Fatalf("Streams first call: %v", err)
	}
	got2, err := f.Streams()
	if err != nil {
		t.Fatalf("Streams second call: %v", err)
	}
	if got1 != ios || got2 != ios {
		t.Fatalf("Streams did not return provided IOStreams pointer")
	}
}

func TestFactoryStreamsFallsBackToSystem(t *testing.T) {
	f := &Factory{}

	got1, err := f.Streams()
	if err != nil {
		t.Fatalf("Streams first fallback call: %v", err)
	}
	got2, err := f.Streams()
	if err != nil {
		t.Fatalf("Streams second fallback call: %v", err)
	}
	if got1 == nil || got2 == nil {
		t.Fatal("Streams fallback returned nil")
	}
	if got1 != got2 {
		t.Fatal("Streams fallback did not cache system streams")
	}
	if got1.In == nil || got1.Out == nil || got1.ErrOut == nil {
		t.Fatal("Streams fallback returned incomplete system streams")
	}
}

func TestFactoryCloudClientUsesOverride(t *testing.T) {
	want, err := bbcloud.New(bbcloud.Options{BaseURL: "https://api.bitbucket.org/2.0", Token: "tok"})
	if err != nil {
		t.Fatalf("bbcloud.New: %v", err)
	}

	f := &Factory{
		NewCloudClientFunc: func(host *config.Host) (*bbcloud.Client, error) {
			if host == nil || host.BaseURL != "https://api.bitbucket.org/2.0" {
				t.Fatalf("unexpected host: %#v", host)
			}
			return want, nil
		},
	}

	got, err := f.CloudClient(&config.Host{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("CloudClient: %v", err)
	}
	if got != want {
		t.Fatal("CloudClient did not return override client")
	}
}

func TestFactoryCloudClientFallsBackWithNilReceiver(t *testing.T) {
	var f *Factory
	got, err := f.CloudClient(&config.Host{BaseURL: "https://api.bitbucket.org/2.0", Token: "tok"})
	if err != nil {
		t.Fatalf("CloudClient fallback: %v", err)
	}
	if got == nil {
		t.Fatal("CloudClient fallback returned nil client")
	}
}

func TestFactoryDCClientUsesOverride(t *testing.T) {
	want, err := bbdc.New(bbdc.Options{BaseURL: "https://bitbucket.example.com", Token: "tok"})
	if err != nil {
		t.Fatalf("bbdc.New: %v", err)
	}

	f := &Factory{
		NewDCClientFunc: func(host *config.Host) (*bbdc.Client, error) {
			if host == nil || host.BaseURL != "https://bitbucket.example.com" {
				t.Fatalf("unexpected host: %#v", host)
			}
			return want, nil
		},
	}

	got, err := f.DCClient(&config.Host{BaseURL: "https://bitbucket.example.com"})
	if err != nil {
		t.Fatalf("DCClient: %v", err)
	}
	if got != want {
		t.Fatal("DCClient did not return override client")
	}
}

func TestFactoryDCClientFallsBackWithNilReceiver(t *testing.T) {
	var f *Factory
	got, err := f.DCClient(&config.Host{BaseURL: "https://bitbucket.example.com", Token: "tok"})
	if err != nil {
		t.Fatalf("DCClient fallback: %v", err)
	}
	if got == nil {
		t.Fatal("DCClient fallback returned nil client")
	}
}

func TestFactoryBrowserOpenerCachesDefaultAndHonorsOverride(t *testing.T) {
	f := &Factory{}

	got1 := f.BrowserOpener()
	got2 := f.BrowserOpener()
	if got1 == nil || got2 == nil {
		t.Fatal("BrowserOpener returned nil")
	}
	if got1 != got2 {
		t.Fatal("BrowserOpener did not cache default browser")
	}

	override := stubBrowser{}
	f.Browser = override
	if got := f.BrowserOpener(); got != override {
		t.Fatal("BrowserOpener did not return configured browser")
	}
}

func TestFactoryPagerManagerCachesDefaultAndHonorsOverride(t *testing.T) {
	ios, _, _ := testIOStreams()
	f := &Factory{IOStreams: ios}

	got1 := f.PagerManager()
	got2 := f.PagerManager()
	if got1 == nil || got2 == nil {
		t.Fatal("PagerManager returned nil")
	}
	if got1 != got2 {
		t.Fatal("PagerManager did not cache default pager")
	}
	if got1.Enabled() {
		t.Fatal("PagerManager default should be noop for non-TTY streams")
	}

	override := stubPager{}
	f.Pager = override
	if got := f.PagerManager(); got != override {
		t.Fatal("PagerManager did not return configured pager")
	}
}

func TestFactoryPromptCachesDefaultAndHonorsOverride(t *testing.T) {
	ios, _, _ := testIOStreams()
	f := &Factory{IOStreams: ios}

	got1 := f.Prompt()
	got2 := f.Prompt()
	if got1 == nil || got2 == nil {
		t.Fatal("Prompt returned nil")
	}
	if got1 != got2 {
		t.Fatal("Prompt did not cache default prompter")
	}

	override := stubPrompter{}
	f.Prompter = override
	if got := f.Prompt(); got != override {
		t.Fatal("Prompt did not return configured prompter")
	}
}

func TestFactoryProgressSpinnerCachesDefaultAndHonorsOverride(t *testing.T) {
	ios, _, errOut := testIOStreams()
	f := &Factory{IOStreams: ios}

	got1 := f.ProgressSpinner()
	got2 := f.ProgressSpinner()
	if got1 == nil || got2 == nil {
		t.Fatal("ProgressSpinner returned nil")
	}
	if got1 != got2 {
		t.Fatal("ProgressSpinner did not cache default spinner")
	}

	got1.Start("starting")
	got1.Stop("done")
	if !strings.Contains(errOut.String(), "starting") || !strings.Contains(errOut.String(), "done") {
		t.Fatalf("default spinner did not write expected messages, got %q", errOut.String())
	}

	override := stubSpinner{}
	f.Spinner = override
	if got := f.ProgressSpinner(); got != override {
		t.Fatal("ProgressSpinner did not return configured spinner")
	}
}

var (
	_ browser.Browser    = stubBrowser{}
	_ pager.Manager      = stubPager{}
	_ prompter.Interface = stubPrompter{}
	_ progress.Spinner   = stubSpinner{}
)
