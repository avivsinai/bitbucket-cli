package frontmatter

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantName string
		wantDesc string
		wantBody string
		wantErr  bool
	}{
		{
			name:     "valid frontmatter",
			content:  "---\nname: test-skill\ndescription: A test skill\n---\n# Body\n",
			wantName: "test-skill",
			wantDesc: "A test skill",
			wantBody: "# Body\n",
		},
		{
			name:     "no frontmatter",
			content:  "# Just a markdown file\n",
			wantBody: "# Just a markdown file\n",
		},
		{
			name:    "invalid YAML",
			content: "---\n: invalid yaml [[\n---\n",
			wantErr: true,
		},
		{
			name:     "no closing delimiter is treated as body",
			content:  "---\nname: test\n",
			wantBody: "---\nname: test\n",
		},
		{
			name:     "windows line endings",
			content:  "---\r\nname: crlf\r\n---\r\n# Body\r\n",
			wantName: "crlf",
			wantBody: "# Body\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse returned error: %v", err)
			}
			if result.Metadata.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Metadata.Name, tt.wantName)
			}
			if result.Metadata.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", result.Metadata.Description, tt.wantDesc)
			}
			if result.Body != tt.wantBody {
				t.Errorf("Body = %q, want %q", result.Body, tt.wantBody)
			}
		})
	}
}

func TestInjectBitbucketMetadata(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		pinnedRef      string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:    "injects metadata without pin and preserves body",
			content: "---\nname: my-skill\ndescription: desc\nmetadata:\n    local-path: /old\n---\n# Body\n",
			wantContains: []string{
				"bitbucket-repo: https://bitbucket.org/myteam/agent-skills",
				"bitbucket-ref: refs/tags/v1.0.0",
				"bitbucket-commit: abc123",
				"bitbucket-path: skills/my-skill",
				"name: my-skill",
				"# Body",
			},
			wantNotContain: []string{"bitbucket-pinned", "local-path"},
		},
		{
			name:         "injects pinned ref",
			content:      "---\nname: my-skill\n---\n# Body\n",
			pinnedRef:    "v1.0.0",
			wantContains: []string{"bitbucket-pinned: v1.0.0"},
		},
		{
			name:    "injects into content with no frontmatter",
			content: "# Body only\n",
			wantContains: []string{
				"bitbucket-repo: https://bitbucket.org/myteam/agent-skills",
				"# Body only",
			},
		},
		{
			name:         "preserves unrelated metadata keys",
			content:      "---\nname: my-skill\nmetadata:\n    author: monalisa\n---\n",
			wantContains: []string{"author: monalisa", "bitbucket-path: skills/my-skill"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InjectBitbucketMetadata(tt.content, "https://bitbucket.org/myteam/agent-skills", "refs/tags/v1.0.0", "abc123", tt.pinnedRef, "skills/my-skill")
			if err != nil {
				t.Fatalf("InjectBitbucketMetadata returned error: %v", err)
			}
			for _, s := range tt.wantContains {
				if !strings.Contains(got, s) {
					t.Errorf("output missing %q:\n%s", s, got)
				}
			}
			for _, s := range tt.wantNotContain {
				if strings.Contains(got, s) {
					t.Errorf("output should not contain %q:\n%s", s, got)
				}
			}
		})
	}
}

func TestInjectBitbucketMetadataRejectsInvalidYAML(t *testing.T) {
	_, err := InjectBitbucketMetadata("---\n: bad [[\n---\n", "u", "r", "c", "", "p")
	if err == nil {
		t.Fatal("expected error for invalid frontmatter")
	}
}

func TestInjectLocalMetadata(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:           "strips bitbucket keys and injects local-path",
			content:        "---\nname: my-skill\nmetadata:\n    bitbucket-repo: old\n    bitbucket-ref: v1\n    bitbucket-commit: abc\n    bitbucket-pinned: v1\n    bitbucket-path: skills/my-skill\n---\n# Body\n",
			wantContains:   []string{"local-path: /home/monalisa/skills/my-skill"},
			wantNotContain: []string{"bitbucket-repo", "bitbucket-ref", "bitbucket-commit", "bitbucket-pinned", "bitbucket-path"},
		},
		{
			name:         "injects into content with no metadata",
			content:      "# Body only\n",
			wantContains: []string{"local-path: /home/monalisa/skills/my-skill"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InjectLocalMetadata(tt.content, "/home/monalisa/skills/my-skill")
			if err != nil {
				t.Fatalf("InjectLocalMetadata returned error: %v", err)
			}
			for _, s := range tt.wantContains {
				if !strings.Contains(got, s) {
					t.Errorf("output missing %q:\n%s", s, got)
				}
			}
			for _, s := range tt.wantNotContain {
				if strings.Contains(got, s) {
					t.Errorf("output should not contain %q:\n%s", s, got)
				}
			}
		})
	}
}

func TestSerialize(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		body        string
		wantPrefix  string
		wantSuffix  string
	}{
		{name: "with body", frontmatter: map[string]any{"name": "test"}, body: "# Body content\n", wantPrefix: "---\nname: test\n---\n", wantSuffix: "# Body content\n"},
		{name: "empty body ends after delimiter", frontmatter: map[string]any{"name": "test"}, body: "", wantSuffix: "---\n"},
		{name: "body without trailing newline gets one", frontmatter: map[string]any{"name": "test"}, body: "# No newline", wantSuffix: "# No newline\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Serialize(tt.frontmatter, tt.body)
			if err != nil {
				t.Fatalf("Serialize returned error: %v", err)
			}
			if tt.wantPrefix != "" && !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("output %q should start with %q", got, tt.wantPrefix)
			}
			if !strings.HasSuffix(got, tt.wantSuffix) {
				t.Errorf("output %q should end with %q", got, tt.wantSuffix)
			}
		})
	}
}

func TestRoundTripPreservesBody(t *testing.T) {
	content := "---\nname: rt\ndescription: round trip\n---\n# Title\n\nSome *markdown* with --- inside.\n"
	out, err := InjectBitbucketMetadata(content, "https://bitbucket.org/w/r", "refs/heads/main", "c", "", "skills/rt")
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	parsed, err := Parse(out)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Body != "# Title\n\nSome *markdown* with --- inside.\n" {
		t.Fatalf("body changed: %q", parsed.Body)
	}
	if parsed.Metadata.Meta[KeyRef] != "refs/heads/main" {
		t.Fatalf("metadata not readable back: %+v", parsed.Metadata.Meta)
	}
}

func TestFindInstallMetadata(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{name: "no frontmatter", content: "# body\n"},
		{name: "no metadata map", content: "---\nname: x\n---\n"},
		{name: "only the author's own keys", content: "---\nname: x\nmetadata:\n    author: monalisa\n---\n"},
		{
			name:    "bkt and gh keys are both reported, sorted",
			content: "---\nname: x\nmetadata:\n    local-path: /p\n    bitbucket-repo: r\n    github-ref: g\n    author: monalisa\n---\n",
			want:    []string{"bitbucket-repo", "github-ref", "local-path"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.content)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			got := FindInstallMetadata(result)
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Fatalf("FindInstallMetadata = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStripInstallMetadataPreservesAuthoredContent(t *testing.T) {
	// A published SKILL.md belongs to its author, so stripping must not
	// reformat it: comments, quoting, key order and indentation all survive.
	content := "---\n" +
		"# Owned by the platform team - keep name first\n" +
		"name: code-review\n" +
		"description: \"Reviews code carefully.\"\n" +
		"license: MIT\n" +
		"allowed-tools: Read Grep Bash\n" +
		"metadata:\n" +
		"  author: monalisa\n" +
		"  bitbucket-repo: https://bitbucket.org/other/skills\n" +
		"  bitbucket-commit: abc123\n" +
		"---\n" +
		"# Body\n"

	got, err := StripInstallMetadata(content)
	if err != nil {
		t.Fatalf("StripInstallMetadata: %v", err)
	}

	want := "---\n" +
		"# Owned by the platform team - keep name first\n" +
		"name: code-review\n" +
		"description: \"Reviews code carefully.\"\n" +
		"license: MIT\n" +
		"allowed-tools: Read Grep Bash\n" +
		"metadata:\n" +
		"  author: monalisa\n" +
		"---\n" +
		"# Body\n"
	if got != want {
		t.Fatalf("StripInstallMetadata rewrote authored content.\n got: %q\nwant: %q", got, want)
	}
}

func TestStripInstallMetadata(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr string
	}{
		{
			name:    "no frontmatter is untouched",
			content: "# body only\n",
			want:    "# body only\n",
		},
		{
			name:    "no metadata map is untouched",
			content: "---\nname: x\n---\n# body\n",
			want:    "---\nname: x\n---\n# body\n",
		},
		{
			name:    "no install keys is untouched",
			content: "---\nname: x\nmetadata:\n    author: monalisa\n---\n",
			want:    "---\nname: x\nmetadata:\n    author: monalisa\n---\n",
		},
		{
			name:    "an emptied metadata map is removed entirely",
			content: "---\nname: x\nmetadata:\n    bitbucket-repo: r\n    local-path: /p\n---\n# body\n",
			want:    "---\nname: x\n---\n# body\n",
		},
		{
			name:    "gh keys are stripped too",
			content: "---\nname: x\nmetadata:\n    github-repo: r\n    github-tree-sha: t\n    author: a\n---\n",
			want:    "---\nname: x\nmetadata:\n    author: a\n---\n",
		},
		{
			name:    "keys after the metadata block are kept",
			content: "---\nname: x\nmetadata:\n    bitbucket-repo: r\nlicense: MIT\n---\n",
			want:    "---\nname: x\nlicense: MIT\n---\n",
		},
		{
			name:    "nested install-looking keys outside the metadata map are kept",
			content: "---\nname: x\nmetadata:\n    bitbucket-repo: r\nlicense:\n    local-path: /licenses/MIT\n---\n",
			want:    "---\nname: x\nlicense:\n    local-path: /licenses/MIT\n---\n",
		},
		{
			name:    "nested install-looking keys inside another metadata value are kept",
			content: "---\nname: x\nmetadata:\n    bitbucket-repo: r\n    nested:\n        local-path: /nested/path\n---\n",
			want:    "---\nname: x\nmetadata:\n    nested:\n        local-path: /nested/path\n---\n",
		},
		{
			name:    "windows line endings survive",
			content: "---\r\nname: x\r\nmetadata:\r\n    bitbucket-repo: r\r\n    author: a\r\n---\r\n# body\r\n",
			want:    "---\r\nname: x\r\nmetadata:\r\n    author: a\r\n---\r\n# body\r\n",
		},
		{
			// A top-level comment does not end a YAML mapping, so the entries
			// after it are still metadata children: the install keys must be
			// stripped rather than surviving as orphans of a removed header.
			name:    "a comment before the first child does not end the metadata map",
			content: "---\nname: x\nmetadata:\n# written by the installer\n    bitbucket-repo: r\n    author: a\n---\n",
			want:    "---\nname: x\nmetadata:\n# written by the installer\n    author: a\n---\n",
		},
		{
			// The same comment after an authored child must not re-scope the
			// later entries either: the install keys below it are still inside
			// the metadata map and leave with the rest of the install metadata.
			name:    "a comment after an authored child does not re-scope later entries",
			content: "---\nname: x\nmetadata:\n    author: a\n# everything below came from the install\n    bitbucket-repo: r\n    local-path: /p\n---\n",
			want:    "---\nname: x\nmetadata:\n    author: a\n# everything below came from the install\n---\n",
		},
		{
			name:    "a comment is kept when an emptied metadata map is removed",
			content: "---\nname: x\nmetadata:\n# installed copy\n    bitbucket-repo: r\n---\n# body\n",
			want:    "---\nname: x\n# installed copy\n---\n# body\n",
		},
		{
			name:    "a comment after the metadata map is preserved with later top-level keys",
			content: "---\nname: x\nmetadata:\n    bitbucket-repo: r\n# license note\nlicense: MIT\n---\n",
			want:    "---\nname: x\n# license note\nlicense: MIT\n---\n",
		},
		{
			name:    "a comment and blank separator before a retained key are kept",
			content: "---\nname: x\nmetadata:\n  bitbucket-repo: source\n\n  # Note for owner\n  owner: Ada\n---\n# Body\n",
			want:    "---\nname: x\nmetadata:\n\n  # Note for owner\n  owner: Ada\n---\n# Body\n",
		},
		{
			name:    "indented standalone comments are kept without retaining hash-prefixed scalar data",
			content: "---\r\nname: x\r\nmetadata:\r\n  local-path:\r\n    origin: |\r\n      source\r\n      # scalar data, not a comment\r\n\r\n    # standalone at a deeper indentation\r\n  # Note for owner\r\n  owner: Ada\r\n---\r\n# Body\r\n",
			want:    "---\r\nname: x\r\nmetadata:\r\n\r\n    # standalone at a deeper indentation\r\n  # Note for owner\r\n  owner: Ada\r\n---\r\n# Body\r\n",
		},
		{
			name:    "hash-prefixed lines in multiline quoted install values are removed",
			content: "---\nname: x\nmetadata:\n  bitbucket-repo: 'source\n    # still part of the value'\n\n    # real comment\n  owner: Ada\n---\n",
			want:    "---\nname: x\nmetadata:\n\n    # real comment\n  owner: Ada\n---\n",
		},
		{
			name:    "a comment inside an install span is retained when its surrounding map is removed",
			content: "---\nname: x\nmetadata:\n  local-path:\n    first: a\n    # authored tracking explanation\n    next: b\nlicense: MIT\n---\n",
			want:    "---\nname: x\n    # authored tracking explanation\nlicense: MIT\n---\n",
		},
		{
			name:    "multiline install values do not leak into authored block scalars",
			content: "---\nname: x\nmetadata:\n  team: |\n    alpha\n\n    omega\n  bitbucket-repo: |\n    first\n\n    rest\n  author: a\n---\n",
			want:    "---\nname: x\nmetadata:\n  team: |\n    alpha\n\n    omega\n  author: a\n---\n",
		},
		{
			name:    "an explicit key with a nested value keeps its child indentation",
			content: "---\nname: x\nmetadata:\n  ? team\n  :\n    nested: original\n  bitbucket-repo: source\n---\n",
			want:    "---\nname: x\nmetadata:\n  ? team\n  :\n    nested: original\n---\n",
		},
		{
			name:    "an explicit key directly after an install key is preserved",
			content: "---\nname: x\nmetadata:\n  team: Ada\n  bitbucket-repo: source\n  ? owner\n  :\n    nested: original\n---\n",
			want:    "---\nname: x\nmetadata:\n  team: Ada\n  ? owner\n  :\n    nested: original\n---\n",
		},
		{
			name:    "flow-style metadata is reported, not mangled",
			content: "---\nname: x\nmetadata: {bitbucket-repo: r, author: a}\n---\n",
			wantErr: "metadata is not written as a block mapping",
		},
		{
			name:    "invalid frontmatter is reported",
			content: "---\n: bad [[\n---\n",
			wantErr: "invalid frontmatter YAML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StripInstallMetadata(tt.content)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("StripInstallMetadata: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got  %q\nwant %q", got, tt.want)
			}
			// The result must still parse and carry no install keys.
			result, parseErr := Parse(got)
			if parseErr != nil {
				t.Fatalf("stripped content no longer parses: %v", parseErr)
			}
			if keys := FindInstallMetadata(result); len(keys) != 0 {
				t.Fatalf("install keys survived: %v", keys)
			}
		})
	}
}

// Before the fix an empty frontmatter block was not recognised: the text up to
// the next markdown rule was parsed as YAML and dropped from the body on install.
func TestParseEmptyFrontmatter(t *testing.T) {
	res, err := Parse("---\n---\n# Title\n\n---\n\ntext\n")
	if err != nil {
		t.Fatal(err)
	}
	if res.Metadata.Name != "" || len(res.RawYAML) != 0 {
		t.Fatalf("metadata = %+v, raw = %v; want empty", res.Metadata, res.RawYAML)
	}
	if !strings.Contains(res.Body, "# Title") || !strings.Contains(res.Body, "\n---\n") {
		t.Fatalf("body lost content: %q", res.Body)
	}
}
