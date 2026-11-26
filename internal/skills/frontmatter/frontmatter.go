// Package frontmatter parses and rewrites the YAML frontmatter of SKILL.md files.
package frontmatter

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
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
	var before, after string
	if strings.HasPrefix(rest, delimiter) {
		// Empty frontmatter: the closing delimiter directly follows the opening one.
		after = rest[len(delimiter):]
	} else {
		var ok bool
		before, after, ok = strings.Cut(rest, "\n"+delimiter)
		if !ok {
			return &ParseResult{Body: content}, nil
		}
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

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content[start:end]), &doc); err != nil {
		return "", err
	}
	entries, err := directMetadataEntries(&doc, metaIdx, len(lines), install)
	if err != nil {
		return "", err
	}

	comments, err := standaloneCommentsInInstallEntries(lines, entries, &doc)
	if err != nil {
		return "", err
	}

	remove := make([]bool, len(lines))
	removed := 0
	remaining := 0
	for _, entry := range entries {
		if !entry.install {
			remaining++
			continue
		}
		from, to := entry.start, entry.end
		// Keep separator lines at the end of the entry and any standalone
		// comments, without rescuing hash-prefixed text in a scalar value.
		for to > from && (strings.TrimSpace(lines[to-1]) == "" || comments[to-1]) {
			to--
		}
		for i := from; i < to; i++ {
			if !remove[i] && !comments[i] {
				remove[i] = true
				removed++
			}
		}
	}
	if removed == 0 {
		return content, nil
	}
	if remaining == 0 {
		remove[metaIdx] = true
	}

	kept := make([]string, 0, len(lines)-removed)
	for i, line := range lines {
		if !remove[i] {
			kept = append(kept, line)
		}
	}
	stripped := content[:start] + strings.Join(kept, "\n") + content[end:]
	if err := verifyOnlyInstallMetadataRemoved(content, stripped); err != nil {
		return "", err
	}
	return stripped, nil
}

type metadataEntry struct {
	start   int
	end     int
	install bool
}

// directMetadataEntries returns the line spans of the metadata mapping's direct
// children. yaml.v3 knows which keys are direct children even when they use
// explicit-key syntax or have multiline values; using its positions keeps the
// line rewrite from guessing at indentation inside those values.
func directMetadataEntries(doc *yaml.Node, metaIdx, lineCount int, install map[string]bool) ([]metadataEntry, error) {
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, nil
	}
	root := doc.Content[0]
	metaPair := -1
	var metaValue *yaml.Node
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i]
		if key.Value == "metadata" && key.Line-1 == metaIdx {
			metaPair = i
			metaValue = root.Content[i+1]
			break
		}
	}
	if metaPair == -1 || metaValue.Kind != yaml.MappingNode {
		return nil, nil
	}

	mappingEnd := lineCount
	if metaPair+2 < len(root.Content) {
		mappingEnd = root.Content[metaPair+2].Line - 1
	}
	entries := make([]metadataEntry, 0, len(metaValue.Content)/2)
	for i := 0; i+1 < len(metaValue.Content); i += 2 {
		key := metaValue.Content[i]
		start := key.Line - 1
		end := mappingEnd
		if i+2 < len(metaValue.Content) {
			end = metaValue.Content[i+2].Line - 1
		}
		if start < 0 || end < start || end > lineCount {
			return nil, fmt.Errorf("metadata entry positions are inconsistent; remove the install keys by hand")
		}
		entries = append(entries, metadataEntry{
			start:   start,
			end:     end,
			install: install[key.Value],
		})
	}
	return entries, nil
}

const commentProbePrefix = "#bkt_comment_probe_"

var commentProbeIndex = regexp.MustCompile(commentProbePrefix + `([0-9]+)_`)

// standaloneCommentsInInstallEntries asks yaml.v3 which hash-prefixed lines
// are comments instead of guessing from indentation. Insert line markers just
// after each hash in a temporary copy: the parser discards them from values
// for actual comments, but retains them in literal or quoted scalar content.
// The original lines, never this probe, are used for the rewrite.
func standaloneCommentsInInstallEntries(lines []string, entries []metadataEntry, doc *yaml.Node) (map[int]bool, error) {
	comments := make(map[int]bool)
	var probe []string
	for _, entry := range entries {
		if !entry.install {
			continue
		}
		for i := entry.start + 1; i < entry.end; i++ {
			text := strings.TrimLeft(lines[i], " \t")
			if !strings.HasPrefix(text, "#") {
				continue
			}
			if probe == nil {
				probe = append([]string(nil), lines...)
			}
			indent := len(lines[i]) - len(text)
			probe[i] = lines[i][:indent] + commentProbePrefix + strconv.Itoa(i) + "_" + text[1:]
			comments[i] = true
		}
	}
	if probe == nil {
		return comments, nil
	}

	// Never confuse author text (including decoded escape sequences) with a
	// marker. If the probe cannot establish the boundary, leave the file alone.
	if yamlValuesContain(doc, commentProbePrefix) {
		return nil, fmt.Errorf("could not distinguish metadata comments from scalar values; remove the install keys by hand")
	}
	var parsed yaml.Node
	if err := yaml.Unmarshal([]byte(strings.Join(probe, "\n")), &parsed); err != nil {
		return nil, fmt.Errorf("could not distinguish metadata comments from scalar values: %w", err)
	}
	visitYAMLValues(&parsed, func(value string) {
		for _, match := range commentProbeIndex.FindAllStringSubmatch(value, -1) {
			if line, err := strconv.Atoi(match[1]); err == nil {
				delete(comments, line)
			}
		}
	})
	return comments, nil
}

func yamlValuesContain(doc *yaml.Node, needle string) bool {
	found := false
	visitYAMLValues(doc, func(value string) { found = found || strings.Contains(value, needle) })
	return found
}

func visitYAMLValues(node *yaml.Node, visit func(string)) {
	visit(node.Value)
	for _, child := range node.Content {
		visitYAMLValues(child, visit)
	}
}

// verifyOnlyInstallMetadataRemoved proves that the semantic YAML document is
// unchanged except for direct install keys missing from metadata. The rewrite
// is intentionally line-oriented to preserve author formatting, but malformed
// or unusual YAML must fail closed rather than silently drop authored values.
func verifyOnlyInstallMetadataRemoved(original, stripped string) error {
	before, err := Parse(original)
	if err != nil {
		return err
	}
	after, err := Parse(stripped)
	if err != nil {
		return fmt.Errorf("stripped frontmatter does not parse: %w", err)
	}
	if before.Body != after.Body {
		return fmt.Errorf("stripping install metadata changed the document body; remove the install keys by hand")
	}

	expected := make(map[string]any, len(before.RawYAML))
	for key, value := range before.RawYAML {
		expected[key] = value
	}
	if meta, ok := before.RawYAML["metadata"].(map[string]any); ok {
		copyMeta := make(map[string]any, len(meta))
		for key, value := range meta {
			copyMeta[key] = value
		}
		for _, key := range InstallMetadataKeys {
			delete(copyMeta, key)
		}
		if len(copyMeta) == 0 {
			delete(expected, "metadata")
		} else {
			expected["metadata"] = copyMeta
		}
	}
	if !reflect.DeepEqual(expected, after.RawYAML) {
		return fmt.Errorf("stripping install metadata changed authored frontmatter values; remove the install keys by hand")
	}
	return nil
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
