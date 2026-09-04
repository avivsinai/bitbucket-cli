// Package discovery finds skills (directories containing SKILL.md) in Bitbucket
// repositories and local directories using the Agent Skills layout conventions.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/avivsinai/bitbucket-cli/internal/skills/frontmatter"
	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
)

// specNamePattern matches the strict agentskills.io name spec:
// 1-64 chars, lowercase alphanumeric + hyphens, no leading/trailing/consecutive hyphens.
var specNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// safeNamePattern matches names that are safe for filesystem use during discovery.
// Allows letters (any case), numbers, hyphens, underscores, dots, and spaces.
// Must start with a letter or number.
var safeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._\- ]*$`)

// maxDescriptionWorkers bounds concurrent SKILL.md fetches so a large skills
// repository does not exhaust Bitbucket's per-hour request budget in one burst.
const maxDescriptionWorkers = 5

// Skill represents a discovered skill in a repository.
type Skill struct {
	Name        string
	Namespace   string // author/scope prefix for namespaced skills
	Description string
	Path        string // path within the repo, e.g. "skills/git-commit"
	Convention  string // which directory convention matched
}

// DisplayName returns the skill name, prefixed with namespace if present
// to disambiguate skills from different authors in the same repository.
// Skills discovered via non-standard conventions (plugins, root, hidden dirs)
// carry a convention tag to distinguish them from identically-named skills in
// the standard skills/ directory.
func (s Skill) DisplayName() string {
	name := s.Name
	if s.Namespace != "" {
		name = s.Namespace + "/" + name
	}
	switch s.Convention {
	case "plugins":
		return "[plugins] " + name
	case "root":
		return "[root] " + name
	case "hidden-dir", "hidden-dir-namespaced":
		return "[hidden-dir] " + name
	default:
		return name
	}
}

// InstallName returns "namespace/name" for namespaced skills and the plain name
// otherwise. It identifies a skill in output and the lock file; the on-disk
// directory is always the plain Name.
func (s Skill) InstallName() string {
	if s.Namespace != "" {
		return s.Namespace + "/" + s.Name
	}
	return s.Name
}

// IsHiddenDirConvention returns true if the skill was discovered in a hidden
// (dot-prefixed) directory such as .claude/skills/ or .agents/skills/.
func (s Skill) IsHiddenDirConvention() bool {
	return s.Convention == "hidden-dir" || s.Convention == "hidden-dir-namespaced"
}

// HasHiddenDirSkills returns true if any of the given skills were discovered
// in hidden directories.
func HasHiddenDirSkills(skills []Skill) bool {
	return slices.ContainsFunc(skills, Skill.IsHiddenDirConvention)
}

// HiddenDirFilterResult holds the outcome of partitioning skills into standard
// and hidden-dir buckets.
type HiddenDirFilterResult struct {
	Standard    []Skill
	HiddenCount int
}

// PartitionHiddenDirSkills splits skills into standard and hidden-dir groups.
func PartitionHiddenDirSkills(skills []Skill) HiddenDirFilterResult {
	var r HiddenDirFilterResult
	for _, s := range skills {
		if s.IsHiddenDirConvention() {
			r.HiddenCount++
		} else {
			r.Standard = append(r.Standard, s)
		}
	}
	return r
}

// skillMatch represents a matched SKILL.md file and its convention.
type skillMatch struct {
	name       string
	namespace  string
	skillDir   string
	convention string
}

// IsSkillPath reports whether a skill selector looks like a repo-relative path
// rather than a simple skill name.
func IsSkillPath(name string) bool {
	name = strings.TrimSuffix(name, "/")
	if name == "" {
		return false
	}
	if strings.HasSuffix(name, "/SKILL.md") {
		return true
	}
	if strings.HasPrefix(name, "skills/") || strings.HasPrefix(name, "plugins/") {
		return true
	}
	if strings.Contains(name, "/skills/") || strings.Contains(name, "/plugins/") {
		return true
	}
	return strings.Count(name, "/") >= 2
}

// matchSkillConventions checks if a slash-separated file path matches any
// standard skill convention.
func matchSkillConventions(p string) *skillMatch {
	if path.Base(p) != "SKILL.md" {
		return nil
	}

	dir := path.Dir(p)
	parentDir := path.Dir(dir)
	skillName := path.Base(dir)

	if !validateName(skillName) {
		return nil
	}

	// skills/<name>/SKILL.md
	if parentDir == "skills" {
		return &skillMatch{name: skillName, skillDir: dir, convention: "skills"}
	}

	// skills/<namespace>/<name>/SKILL.md
	grandparentDir := path.Dir(parentDir)
	if grandparentDir == "skills" {
		namespace := path.Base(parentDir)
		if !validateName(namespace) {
			return nil
		}
		return &skillMatch{name: skillName, namespace: namespace, skillDir: dir, convention: "skills-namespaced"}
	}

	// plugins/<namespace>/skills/<name>/SKILL.md
	if path.Base(parentDir) == "skills" && path.Dir(grandparentDir) == "plugins" {
		namespace := path.Base(grandparentDir)
		if !validateName(namespace) {
			return nil
		}
		return &skillMatch{name: skillName, namespace: namespace, skillDir: dir, convention: "plugins"}
	}

	// <prefix>/skills/<name>/SKILL.md at any depth. Dot-prefixed segments are
	// handled by matchHiddenDirConventions and plugins/ ancestors by the
	// plugins convention above.
	if path.Base(parentDir) == "skills" && !hasHiddenSegment(p) && !hasPluginsAncestor(p) {
		return &skillMatch{name: skillName, skillDir: dir, convention: "skills"}
	}

	// <prefix>/skills/<namespace>/<name>/SKILL.md
	if path.Base(grandparentDir) == "skills" && !hasHiddenSegment(p) && !hasPluginsAncestor(p) {
		namespace := path.Base(parentDir)
		if !validateName(namespace) {
			return nil
		}
		return &skillMatch{name: skillName, namespace: namespace, skillDir: dir, convention: "skills-namespaced"}
	}

	// <name>/SKILL.md at the repository root
	if parentDir == "." && skillName != "skills" && skillName != "plugins" && !strings.HasPrefix(skillName, ".") {
		return &skillMatch{name: skillName, skillDir: dir, convention: "root"}
	}

	return nil
}

// matchHiddenDirConventions checks if a file path matches a skill convention
// under a hidden (dot-prefixed) directory:
//
//   - {prefix}/.{host}/{suffix}/skills/*/SKILL.md         -> "hidden-dir"
//   - {prefix}/.{host}/{suffix}/skills/{scope}/*/SKILL.md -> "hidden-dir-namespaced"
func matchHiddenDirConventions(p string) *skillMatch {
	if path.Base(p) != "SKILL.md" {
		return nil
	}
	if !hasHiddenSegment(p) {
		return nil
	}

	dir := path.Dir(p)
	skillName := path.Base(dir)
	if !validateName(skillName) {
		return nil
	}

	parentDir := path.Dir(dir)
	if path.Base(parentDir) == "skills" {
		return &skillMatch{name: skillName, skillDir: dir, convention: "hidden-dir"}
	}

	grandparentDir := path.Dir(parentDir)
	if path.Base(grandparentDir) == "skills" {
		namespace := path.Base(parentDir)
		if !validateName(namespace) {
			return nil
		}
		return &skillMatch{name: skillName, namespace: namespace, skillDir: dir, convention: "hidden-dir-namespaced"}
	}

	return nil
}

// DiscoverAllSkills finds every skill in a repository at the given commit,
// including skills in hidden directories, sorted by display name.
func DiscoverAllSkills(ctx context.Context, repo source.Repository, sha string) ([]Skill, error) {
	files, err := repo.ListFiles(ctx, sha, "")
	if err != nil {
		return nil, fmt.Errorf("could not list repository files: %w", err)
	}

	seen := make(map[string]bool)
	var matches []skillMatch
	for _, f := range files {
		m := matchSkillConventions(f.Path)
		if m == nil {
			m = matchHiddenDirConventions(f.Path)
		}
		if m == nil || seen[m.skillDir] {
			continue
		}
		seen[m.skillDir] = true
		matches = append(matches, *m)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf(
			"no skills found in %s\n"+
				"  Expected skills in skills/*/SKILL.md, skills/{scope}/*/SKILL.md,\n"+
				"  {prefix}/skills/*/SKILL.md, {prefix}/skills/{scope}/*/SKILL.md,\n"+
				"  */SKILL.md, or plugins/*/skills/*/SKILL.md\n"+
				"  This repository may be a curated list rather than a skills publisher",
			repo.FullName(),
		)
	}

	skills := make([]Skill, 0, len(matches))
	for _, m := range matches {
		skills = append(skills, Skill{
			Name:       m.name,
			Namespace:  m.namespace,
			Path:       m.skillDir,
			Convention: m.convention,
		})
	}
	sort.SliceStable(skills, func(i, j int) bool {
		return skills[i].DisplayName() < skills[j].DisplayName()
	})
	return skills, nil
}

// fetchDescription reads a skill's SKILL.md and returns its frontmatter
// description, or "" when it cannot be read or parsed.
func fetchDescription(ctx context.Context, repo source.Repository, sha string, skill *Skill) string {
	content, err := repo.ReadFile(ctx, sha, skill.Path+"/SKILL.md")
	if err != nil {
		return ""
	}
	result, err := frontmatter.Parse(string(content))
	if err != nil {
		return ""
	}
	return result.Metadata.Description
}

// FetchDescriptions fills in the Description of every skill that lacks one,
// fetching SKILL.md files with bounded concurrency.
func FetchDescriptions(ctx context.Context, repo source.Repository, sha string, skills []Skill, onProgress func(done, total int)) {
	total := 0
	for _, s := range skills {
		if s.Description == "" {
			total++
		}
	}
	if total == 0 {
		return
	}

	var wg sync.WaitGroup
	var done atomic.Int32
	jobs := make(chan *Skill)

	for range min(maxDescriptionWorkers, total) {
		wg.Go(func() {
			for s := range jobs {
				s.Description = fetchDescription(ctx, repo, sha, s)
				if onProgress != nil {
					onProgress(int(done.Add(1)), total)
				}
			}
		})
	}

	for i := range skills {
		if skills[i].Description == "" {
			jobs <- &skills[i]
		}
	}
	close(jobs)
	wg.Wait()
}

// DiscoverSkillByPathOptions controls DiscoverSkillByPath.
type DiscoverSkillByPathOptions struct {
	SkipDescription bool
}

// DiscoverSkillByPath looks up a single skill by its exact directory path in
// the repository without walking the whole tree.
func DiscoverSkillByPath(ctx context.Context, repo source.Repository, sha, skillPath string, opts DiscoverSkillByPathOptions) (*Skill, error) {
	skillPath = strings.TrimSuffix(skillPath, "/SKILL.md")
	skillPath = strings.Trim(skillPath, "/")

	skillName := path.Base(skillPath)
	if !validateName(skillName) {
		return nil, fmt.Errorf("invalid skill name %q", skillName)
	}

	files, err := repo.ListFiles(ctx, sha, skillPath)
	if errors.Is(err, source.ErrNotFound) {
		return nil, fmt.Errorf("skill directory %q not found in %s", skillPath, repo.FullName())
	}
	if err != nil {
		return nil, fmt.Errorf("could not read skill directory %q in %s: %w", skillPath, repo.FullName(), err)
	}
	if !slices.ContainsFunc(files, func(f source.File) bool { return f.Path == skillPath+"/SKILL.md" }) {
		return nil, fmt.Errorf("no SKILL.md found in %s", skillPath)
	}

	var namespace, convention string
	parts := strings.Split(skillPath, "/")
	for i, p := range parts {
		if p != "skills" {
			continue
		}
		// Plugin convention: .../plugins/<ns>/skills/<name>
		if i >= 2 && parts[i-2] == "plugins" {
			namespace = parts[i-1]
			convention = "plugins"
			break
		}
		// Namespaced skill convention: .../skills/<ns>/<name>
		if afterSkills := parts[i+1:]; len(afterSkills) >= 2 {
			namespace = afterSkills[0]
		}
		break
	}

	skill := &Skill{
		Name:       skillName,
		Namespace:  namespace,
		Convention: convention,
		Path:       skillPath,
	}
	if !opts.SkipDescription {
		skill.Description = fetchDescription(ctx, repo, sha, skill)
	}
	return skill, nil
}

// SkillFiles returns every file in a skill directory with paths relative to the
// skill root, sorted.
func SkillFiles(ctx context.Context, repo source.Repository, sha, skillPath string) ([]source.File, error) {
	skillPath = strings.Trim(skillPath, "/")
	files, err := repo.ListFiles(ctx, sha, skillPath)
	if err != nil {
		return nil, fmt.Errorf("could not list skill files: %w", err)
	}
	prefix := skillPath + "/"
	relative := make([]source.File, 0, len(files))
	for _, f := range files {
		rel := strings.TrimPrefix(f.Path, prefix)
		if rel == "" || rel == f.Path {
			continue
		}
		relative = append(relative, source.File{Path: rel, Size: f.Size})
	}
	sort.Slice(relative, func(i, j int) bool { return relative[i].Path < relative[j].Path })
	return relative, nil
}

// DiscoverAllLocalSkills finds every skill in a local directory, including
// skills in hidden directories. A directory that itself contains SKILL.md is
// treated as a single skill with Path ".".
func DiscoverAllLocalSkills(dir string) ([]Skill, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("could not resolve path: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return nil, fmt.Errorf("could not access %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	if _, err := os.Stat(filepath.Join(absDir, "SKILL.md")); err == nil {
		skill, err := localSkillFromDir(absDir)
		if err != nil {
			return nil, err
		}
		skill.Path = "."
		return []Skill{*skill}, nil
	}

	var skills []Skill
	seen := make(map[string]bool)

	err = filepath.Walk(absDir, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// Git metadata is not skill content; pruning it also avoids races with
		// concurrent maintenance. Other hidden directories remain eligible.
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		// Skip symlinks to avoid following links outside the source tree.
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if info.IsDir() || info.Name() != "SKILL.md" {
			return nil
		}

		relPath, relErr := filepath.Rel(absDir, p)
		if relErr != nil {
			return relErr
		}
		relPath = filepath.ToSlash(relPath)

		m := matchSkillConventions(relPath)
		if m == nil {
			m = matchHiddenDirConventions(relPath)
		}
		if m == nil || seen[m.skillDir] {
			return nil
		}
		seen[m.skillDir] = true

		skill, skillErr := localSkillFromDir(filepath.Join(absDir, filepath.FromSlash(m.skillDir)))
		if skillErr != nil {
			return nil //nolint:nilerr // intentionally skip directories that are not valid skills
		}
		skill.Path = m.skillDir
		skill.Namespace = m.namespace
		skill.Convention = m.convention
		skills = append(skills, *skill)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not walk directory: %w", err)
	}

	if len(skills) == 0 {
		return nil, fmt.Errorf(
			"no skills found in %s\n"+
				"  Expected SKILL.md in the directory, or skills in skills/*/SKILL.md,\n"+
				"  skills/{scope}/*/SKILL.md, {prefix}/skills/*/SKILL.md,\n"+
				"  {prefix}/skills/{scope}/*/SKILL.md, */SKILL.md, or\n"+
				"  plugins/*/skills/*/SKILL.md",
			dir,
		)
	}

	return skills, nil
}

func localSkillFromDir(dir string) (*Skill, error) {
	skillFile := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", skillFile, err)
	}

	name := filepath.Base(dir)
	var description string
	if result, parseErr := frontmatter.Parse(string(data)); parseErr == nil {
		if result.Metadata.Name != "" {
			name = result.Metadata.Name
		}
		description = result.Metadata.Description
	}

	if !validateName(name) {
		return nil, fmt.Errorf("invalid skill name %q in %s", name, dir)
	}

	return &Skill{
		Name:        name,
		Description: description,
		Path:        filepath.Base(dir),
	}, nil
}

// validateName checks if a skill name is safe for use as a directory name.
func validateName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	if strings.Contains(name, "/") || strings.Contains(name, "..") {
		return false
	}
	return safeNamePattern.MatchString(name)
}

// hasHiddenSegment reports whether any path component starts with a dot.
func hasHiddenSegment(p string) bool {
	for seg := range strings.SplitSeq(p, "/") {
		if strings.HasPrefix(seg, ".") {
			return true
		}
	}
	return false
}

// hasPluginsAncestor reports whether any path component is "plugins".
func hasPluginsAncestor(p string) bool {
	return slices.Contains(strings.Split(p, "/"), "plugins")
}

// IsSpecCompliant checks if a skill name matches the strict agentskills.io spec.
func IsSpecCompliant(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	if strings.Contains(name, "--") {
		return false
	}
	return specNamePattern.MatchString(name)
}

// DiscoverLocalSkills finds skills in a local directory, excluding those in
// hidden directories, which are installed copies rather than the repository's
// own work.
func DiscoverLocalSkills(dir string) ([]Skill, error) {
	all, err := DiscoverAllLocalSkills(dir)
	if err != nil {
		return nil, err
	}
	skills := PartitionHiddenDirSkills(all).Standard
	if len(skills) == 0 {
		return nil, fmt.Errorf(
			"no skills found in %s\n"+
				"  Expected SKILL.md in the directory, or skills in skills/*/SKILL.md,\n"+
				"  skills/{scope}/*/SKILL.md, */SKILL.md, or plugins/*/skills/*/SKILL.md",
			dir,
		)
	}
	return skills, nil
}
