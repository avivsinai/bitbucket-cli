package pipeline

import (
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
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
