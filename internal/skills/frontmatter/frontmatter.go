// Package frontmatter parses and rewrites the YAML frontmatter of SKILL.md files.
package frontmatter

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const delimiter = "---"

// Metadata keys written under the spec-defined "metadata" map to track where
// an installed skill came from. The "bitbucket-" prefix avoids collisions with
// other tools' keys (gh writes "github-*", local installs write "local-path").
const (
	KeyRepo   = "bitbucket-repo"   // canonical web URL of the source repository
	KeyRef    = "bitbucket-ref"    // resolved ref: refs/tags/*, refs/heads/*, or a commit SHA
	KeyCommit = "bitbucket-commit" // latest commit that touched the skill directory
	KeyPath   = "bitbucket-path"   // skill directory within the repository
	KeyPinned = "bitbucket-pinned" // user-supplied pin, present only when pinned
	KeyLocal  = "local-path"       // absolute source directory for --from-local installs
)

// InstallMetadataKeys lists every key that marks a SKILL.md as installed by a
// tool (bkt, gh, or a local copy) rather than authored in place.
var InstallMetadataKeys = []string{
	KeyRepo, KeyRef, KeyCommit, KeyPath, KeyPinned, KeyLocal,
	"github-repo", "github-ref", "github-tree-sha", "github-path", "github-pinned",
}

// Metadata represents the parsed YAML frontmatter of a SKILL.md file.
type Metadata struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Meta        map[string]any `yaml:"metadata,omitempty"`
}

// ParseResult contains the parsed frontmatter and remaining body.
type ParseResult struct {
	Metadata Metadata
	Body     string
	RawYAML  map[string]any
}

// Parse extracts YAML frontmatter delimited by "---" lines. Content without
// frontmatter is returned unchanged as Body.
func Parse(content string) (*ParseResult, error) {
	trimmed := strings.TrimLeft(content, "\r\n")
	if !strings.HasPrefix(trimmed, delimiter) {
		return &ParseResult{Body: content}, nil
	}

	rest := strings.TrimLeft(trimmed[len(delimiter):], "\r\n")
	before, after, ok := strings.Cut(rest, "\n"+delimiter)
	if !ok {
		return &ParseResult{Body: content}, nil
	}

	var rawYAML map[string]any
	if err := yaml.Unmarshal([]byte(before), &rawYAML); err != nil {
		return nil, fmt.Errorf("invalid frontmatter YAML: %w", err)
	}
	var meta Metadata
	if err := yaml.Unmarshal([]byte(before), &meta); err != nil {
		return nil, fmt.Errorf("invalid frontmatter YAML: %w", err)
	}

	return &ParseResult{
		Metadata: meta,
		Body:     strings.TrimLeft(after, "\r\n"),
		RawYAML:  rawYAML,
	}, nil
}

// InjectBitbucketMetadata records the Bitbucket source of an installed skill in
// its frontmatter. pinnedRef is the user's explicit pin (empty when unpinned);
// skillPath is the skill directory within the repository (e.g. "skills/my-skill").
func InjectBitbucketMetadata(content, repoURL, ref, commit, pinnedRef, skillPath string) (string, error) {
	result, err := Parse(content)
	if err != nil {
		return "", err
	}

	meta := metadataMap(result)
	meta[KeyRepo] = repoURL
	meta[KeyRef] = ref
	meta[KeyCommit] = commit
	meta[KeyPath] = skillPath
	if pinnedRef != "" {
		meta[KeyPinned] = pinnedRef
	} else {
		delete(meta, KeyPinned)
	}
	delete(meta, KeyLocal)
	result.RawYAML["metadata"] = meta

	return Serialize(result.RawYAML, result.Body)
}

// InjectLocalMetadata records the absolute source directory of a skill copied
// from the local filesystem, replacing any Bitbucket source metadata.
func InjectLocalMetadata(content, sourcePath string) (string, error) {
	result, err := Parse(content)
	if err != nil {
		return "", err
	}

	meta := metadataMap(result)
	for _, key := range []string{KeyRepo, KeyRef, KeyCommit, KeyPath, KeyPinned} {
		delete(meta, key)
	}
	meta[KeyLocal] = sourcePath
	result.RawYAML["metadata"] = meta

	return Serialize(result.RawYAML, result.Body)
}

func metadataMap(result *ParseResult) map[string]any {
	if result.RawYAML == nil {
		result.RawYAML = make(map[string]any)
	}
	meta, _ := result.RawYAML["metadata"].(map[string]any)
	if meta == nil {
		meta = make(map[string]any)
	}
	return meta
}

// Serialize writes a frontmatter map and body back to a SKILL.md string.
func Serialize(frontmatter map[string]any, body string) (string, error) {
	yamlBytes, err := yaml.Marshal(frontmatter)
	if err != nil {
		return "", fmt.Errorf("failed to serialize frontmatter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(delimiter + "\n")
	buf.Write(yamlBytes)
	buf.WriteString(delimiter + "\n")
	if body != "" {
		buf.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			buf.WriteString("\n")
		}
	}
	return buf.String(), nil
}
