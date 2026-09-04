// Package frontmatter parses and rewrites the YAML frontmatter of SKILL.md files.
package frontmatter

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
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

// FindInstallMetadata returns the install-tracking keys present in a parsed
// SKILL.md, sorted. A skill being published should carry none of them: they
// describe where a copy was installed from, not what the repository authored.
func FindInstallMetadata(result *ParseResult) []string {
	meta, _ := result.RawYAML["metadata"].(map[string]any)
	if meta == nil {
		return nil
	}
	var found []string
	for _, key := range InstallMetadataKeys {
		if _, ok := meta[key]; ok {
			found = append(found, key)
		}
	}
	sort.Strings(found)
	return found
}

// metadataKeyLine matches a block-style "metadata:" key at the top level of the
// frontmatter, allowing a trailing comment.
var metadataKeyLine = regexp.MustCompile(`^metadata:[ \t]*(#.*)?$`)

// entryKeyLine captures the indentation and key of a mapping entry.
var entryKeyLine = regexp.MustCompile(`^([ \t]+)([^\s:#][^:]*):`)

// StripInstallMetadata removes the install-tracking keys from a SKILL.md by
// editing the frontmatter line by line. Everything else the author wrote,
// including comments, quoting, key order and indentation, is preserved byte
// for byte: this rewrites a file the author owns, unlike the frontmatter of an
// installed copy, which bkt is free to re-serialise.
//
// A "metadata" value written in flow style ({a: b}) is reported rather than
// edited, since removing one key from it cannot be done safely line by line.
func StripInstallMetadata(content string) (string, error) {
	// Validate first so an unparseable file is reported rather than edited.
	if _, err := Parse(content); err != nil {
		return "", err
	}

	start, end, ok := frontmatterBounds(content)
	if !ok {
		return content, nil
	}
	lines := strings.Split(content[start:end], "\n")

	metaIdx := -1
	for i, line := range lines {
		if !strings.HasPrefix(line, "metadata:") {
			continue
		}
		if !metadataKeyLine.MatchString(strings.TrimRight(line, "\r")) {
			return "", fmt.Errorf("metadata is not written as a block mapping; remove the install keys by hand: %s", strings.TrimSpace(line))
		}
		metaIdx = i
		break
	}
	if metaIdx == -1 {
		return content, nil
	}

	install := make(map[string]bool, len(InstallMetadataKeys))
	for _, key := range InstallMetadataKeys {
		install[key] = true
	}

	kept := append([]string(nil), lines[:metaIdx+1]...)
	remaining := 0
	drop := false
	for _, line := range lines[metaIdx+1:] {
		trimmed := strings.TrimRight(line, "\r")
		m := entryKeyLine.FindStringSubmatch(trimmed)
		switch {
		case m != nil:
			// A new entry decides whether it and its continuation lines stay.
			key := strings.TrimSpace(strings.Trim(m[2], `"'`))
			drop = install[key]
			if !drop {
				remaining++
			}
		case strings.TrimSpace(trimmed) == "":
			drop = false
		case !strings.HasPrefix(trimmed, " ") && !strings.HasPrefix(trimmed, "\t"):
			// Dedented back to the top level: the metadata block has ended.
			drop = false
		}
		if !drop {
			kept = append(kept, line)
		}
	}

	if remaining == 0 {
		kept = append(kept[:metaIdx], kept[metaIdx+1:]...)
	}

	return content[:start] + strings.Join(kept, "\n") + content[end:], nil
}

// frontmatterBounds returns the byte range of the YAML between the delimiters.
func frontmatterBounds(content string) (start, end int, ok bool) {
	lead := len(content) - len(strings.TrimLeft(content, "\r\n"))
	rest := content[lead:]
	if !strings.HasPrefix(rest, delimiter) {
		return 0, 0, false
	}
	afterOpen := lead + len(delimiter)
	afterOpen += len(rest[len(delimiter):]) - len(strings.TrimLeft(rest[len(delimiter):], "\r\n"))

	closeIdx := strings.Index(content[afterOpen:], "\n"+delimiter)
	if closeIdx < 0 {
		return 0, 0, false
	}
	return afterOpen, afterOpen + closeIdx, true
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
