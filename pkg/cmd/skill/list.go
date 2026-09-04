package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/skills/frontmatter"
	"github.com/avivsinai/bitbucket-cli/internal/skills/installer"
	"github.com/avivsinai/bitbucket-cli/internal/skills/registry"
	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

const (
	agentHostPublished        = "published"
	agentHostPublishedDisplay = "n/a (published)"
	scopeCustom               = "custom"
)

type scanFilter int

const (
	scanAllSkills scanFilter = iota
	scanInstalledOnly
	scanPublishedOnly
)

type listOptions struct {
	Agent string
	Scope string
	Dir   string
}

type scanTarget struct {
	dir          string
	agentHostIDs []string
	scope        string
	filter       scanFilter
}

// listedSkill is one row of "bkt skill list" and its --json shape.
type listedSkill struct {
	SkillName    string   `json:"skillName"`
	AgentHostIDs []string `json:"agentHosts"`
	Scope        string   `json:"scope"`
	SourceURL    string   `json:"sourceURL"`
	Version      string   `json:"version"`
	Pinned       bool     `json:"pinned"`
	Path         string   `json:"path"`

	source string // human-readable source ("workspace/repo" or a local path)
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List installed skills",
		Long: `List installed agent skills across known agent host directories.

By default, scans all supported agent hosts in both project and user scope.
Use --agent to scan one host, --scope to scan only project or user scope, or
--dir to scan a custom skills directory.

Project-scope skills are discovered relative to the current git repository
root. User-scope skills are discovered relative to your home directory. Skills
authored in the current repository's skills/ directory are listed as
published. Skills installed by other tools (for example gh) are listed with
the source they record.`,
		Example: `  # List all installed skills
  bkt skill list

  # List skills installed for Claude Code
  bkt skill list --agent claude-code

  # List user-scope skills
  bkt skill list --scope user

  # List skills as JSON
  bkt skill list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Dir != "" && opts.Agent != "" {
				return fmt.Errorf("--dir and --agent cannot be used together")
			}
			if opts.Dir != "" && opts.Scope != "" {
				return fmt.Errorf("--dir and --scope cannot be used together")
			}
			if opts.Scope != "" && opts.Scope != string(registry.ScopeProject) && opts.Scope != string(registry.ScopeUser) {
				return fmt.Errorf("--scope must be %q or %q", registry.ScopeProject, registry.ScopeUser)
			}
			return runList(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Agent, "agent", "", "Filter by target agent (see `bkt skill install --help` for values)")
	cmd.Flags().StringVar(&opts.Scope, "scope", "", "Filter by installation scope: project or user")
	cmd.Flags().StringVar(&opts.Dir, "dir", "", "Scan a custom directory for installed skills")

	return cmd
}

func runList(cmd *cobra.Command, f *cmdutil.Factory, opts *listOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	targets, err := buildScanTargets(ctx, opts)
	if err != nil {
		return err
	}

	skills := []listedSkill{}
	for _, target := range targets {
		found, scanErr := scanInstalledSkills(target.dir, target.agentHostIDs, target.scope, target.filter)
		if scanErr != nil {
			if opts.Dir != "" {
				return fmt.Errorf("could not scan directory: %w", scanErr)
			}
			continue
		}
		skills = append(skills, found...)
	}
	sortListedSkills(skills)

	return cmdutil.WriteOutput(cmd, ios.Out, skills, func() error {
		if len(skills) == 0 {
			_, err := fmt.Fprintln(ios.Out, "No skills installed. Use `bkt skill install <repository>` to add one.")
			return err
		}
		if !ios.IsStdoutTTY() {
			for _, s := range skills {
				if _, err := fmt.Fprintf(ios.Out, "%s\t%s\t%s\t%s\n", sanitizeForTerminal(s.SkillName), formatAgentHosts(s.AgentHostIDs), displayOrDash(s.Scope), displayOrDash(sanitizeForTerminal(s.source))); err != nil {
					return err
				}
			}
			return nil
		}
		tw := tabwriter.NewWriter(ios.Out, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "NAME\tAGENT\tSCOPE\tSOURCE")
		for _, s := range skills {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", sanitizeForTerminal(s.SkillName), formatAgentHosts(s.AgentHostIDs), displayOrDash(s.Scope), displayOrDash(sanitizeForTerminal(s.source)))
		}
		return tw.Flush()
	})
}

func buildScanTargets(ctx context.Context, opts *listOptions) ([]scanTarget, error) {
	if opts.Dir != "" {
		dir, err := filepath.Abs(opts.Dir)
		if err != nil {
			return nil, fmt.Errorf("could not resolve path: %w", err)
		}
		if _, err := os.Stat(dir); err != nil {
			return nil, fmt.Errorf("could not access directory: %w", err)
		}
		return []scanTarget{{dir: dir, scope: scopeCustom}}, nil
	}

	gitRoot := installer.ResolveGitRoot(ctx)
	homeDir := installer.ResolveHomeDir()

	agentHosts, err := selectedAgentHosts(opts.Agent)
	if err != nil {
		return nil, err
	}
	scopes := selectedScopes(opts.Scope)

	byDir := map[string]int{}
	var targets []scanTarget
	for _, host := range agentHosts {
		for _, scope := range scopes {
			dir, installErr := host.InstallDir(scope, gitRoot, homeDir)
			if installErr != nil {
				continue
			}
			if idx, ok := byDir[dir]; ok {
				if !slices.Contains(targets[idx].agentHostIDs, host.ID) {
					targets[idx].agentHostIDs = append(targets[idx].agentHostIDs, host.ID)
				}
				if targets[idx].filter != scanFilterForAgentHost(host, scope) {
					targets[idx].filter = scanAllSkills
				}
				continue
			}
			byDir[dir] = len(targets)
			targets = append(targets, scanTarget{
				dir:          dir,
				agentHostIDs: []string{host.ID},
				scope:        string(scope),
				filter:       scanFilterForAgentHost(host, scope),
			})
		}
	}

	// Skills authored in this repository (skills/*) show up as published.
	if opts.Agent == "" && gitRoot != "" && slices.Contains(scopes, registry.ScopeProject) {
		targets = append(targets, scanTarget{
			dir:          filepath.Join(gitRoot, "skills"),
			agentHostIDs: []string{agentHostPublished},
			scope:        string(registry.ScopeProject),
			filter:       scanPublishedOnly,
		})
	}
	return targets, nil
}

func selectedAgentHosts(agentID string) ([]*registry.AgentHost, error) {
	if agentID != "" {
		host, err := registry.FindByID(agentID)
		if err != nil {
			return nil, err
		}
		return []*registry.AgentHost{host}, nil
	}
	hosts := make([]*registry.AgentHost, len(registry.Agents))
	for i := range registry.Agents {
		hosts[i] = &registry.Agents[i]
	}
	return hosts, nil
}

func selectedScopes(scope string) []registry.Scope {
	if scope != "" {
		return []registry.Scope{registry.Scope(scope)}
	}
	return []registry.Scope{registry.ScopeProject, registry.ScopeUser}
}

// scanFilterForAgentHost keeps OpenClaw's non-hidden "skills" project dir
// from listing authored skills as installed.
func scanFilterForAgentHost(host *registry.AgentHost, scope registry.Scope) scanFilter {
	if scope == registry.ScopeProject && host.ProjectDir == "skills" {
		return scanInstalledOnly
	}
	return scanAllSkills
}

// scanInstalledSkills reads every SKILL.md directly under skillsDir (flat
// layout) or one level deeper (namespaced layout written by other tools).
func scanInstalledSkills(skillsDir string, agentHostIDs []string, scope string, filter scanFilter) ([]listedSkill, error) {
	entries, err := os.ReadDir(skillsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read skills directory: %w", err)
	}

	var skills []listedSkill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillDir := filepath.Join(skillsDir, e.Name())
		if data, readErr := readSkillFile(filepath.Join(skillDir, "SKILL.md")); readErr == nil {
			skill, hasMeta := parseListedSkill(data, e.Name(), skillDir, agentHostIDs, scope)
			if shouldIncludeSkill(filter, hasMeta) {
				skills = append(skills, skill)
			}
			continue
		}

		subEntries, subErr := os.ReadDir(skillDir)
		if subErr != nil {
			continue
		}
		for _, sub := range subEntries {
			if !sub.IsDir() {
				continue
			}
			subDir := filepath.Join(skillDir, sub.Name())
			if data, readErr := readSkillFile(filepath.Join(subDir, "SKILL.md")); readErr == nil {
				skill, hasMeta := parseListedSkill(data, e.Name()+"/"+sub.Name(), subDir, agentHostIDs, scope)
				if shouldIncludeSkill(filter, hasMeta) {
					skills = append(skills, skill)
				}
			}
		}
	}
	return skills, nil
}

// readSkillFile reads a SKILL.md only if it is a regular file.
func readSkillFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("SKILL.md is not a regular file: %s", path)
	}
	return os.ReadFile(path)
}

func shouldIncludeSkill(filter scanFilter, hasInstallMetadata bool) bool {
	switch filter {
	case scanInstalledOnly:
		return hasInstallMetadata
	case scanPublishedOnly:
		return !hasInstallMetadata
	default:
		return true
	}
}

// parseListedSkill derives the list row from SKILL.md frontmatter. It reads
// bkt's bitbucket-* keys, gh's github-* keys, and local-path so skills
// installed by any tool are described.
func parseListedSkill(data []byte, name, dir string, agentHostIDs []string, scope string) (listedSkill, bool) {
	s := listedSkill{
		SkillName:    name,
		AgentHostIDs: agentHostIDs,
		Scope:        scope,
		Path:         dir,
	}

	result, err := frontmatter.Parse(string(data))
	if err != nil || result.Metadata.Meta == nil {
		return s, false
	}
	meta := result.Metadata.Meta
	hasMeta := hasInstallMetadata(meta)

	str := func(key string) string {
		v, _ := meta[key].(string)
		return v
	}

	if sourcePath := cmdutil.FirstNonEmpty(str(frontmatter.KeyPath), str("github-path")); sourcePath != "" {
		if skillName := skillNameFromSourcePath(sourcePath); skillName != "" {
			s.SkillName = skillName
		}
	}

	if repoURL := str(frontmatter.KeyRepo); repoURL != "" {
		s.SourceURL = repoURL
		s.source = repoURL
		if ref, parseErr := source.ParseRepoArg(repoURL); parseErr == nil {
			s.source = ref.Owner + "/" + ref.Slug
		}
	} else if repoURL := str("github-repo"); repoURL != "" {
		s.SourceURL = repoURL
		s.source = strings.TrimPrefix(strings.TrimPrefix(repoURL, "https://"), "http://")
	} else if localPath := str(frontmatter.KeyLocal); localPath != "" {
		s.SourceURL = localPath
		s.source = localPath
	}

	if ref := cmdutil.FirstNonEmpty(str(frontmatter.KeyRef), str("github-ref")); ref != "" {
		s.Version = source.ShortRef(ref)
	}
	if pinned := cmdutil.FirstNonEmpty(str(frontmatter.KeyPinned), str("github-pinned")); pinned != "" {
		s.Pinned = true
		if s.Version == "" {
			s.Version = pinned
		}
	}
	return s, hasMeta
}

func hasInstallMetadata(meta map[string]any) bool {
	for _, key := range frontmatter.InstallMetadataKeys {
		value, ok := meta[key]
		if !ok {
			continue
		}
		if str, ok := value.(string); !ok || strings.TrimSpace(str) != "" {
			return true
		}
	}
	return false
}

// skillNameFromSourcePath recovers "namespace/name" from the recorded source
// path, since namespaced skills are installed flat on disk.
func skillNameFromSourcePath(sourcePath string) string {
	sourcePath = strings.TrimSuffix(sourcePath, "/SKILL.md")
	sourcePath = strings.Trim(sourcePath, "/")
	if sourcePath == "" {
		return ""
	}
	parts := strings.Split(sourcePath, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "skills" {
			continue
		}
		if i >= 2 && parts[i-2] == "plugins" && i+1 < len(parts) {
			return parts[i-1] + "/" + parts[len(parts)-1]
		}
		switch afterSkills := len(parts) - i - 1; afterSkills {
		case 0:
			return ""
		case 1:
			return parts[i+1]
		default:
			return parts[i+1] + "/" + parts[len(parts)-1]
		}
	}
	return parts[len(parts)-1]
}

func sortListedSkills(skills []listedSkill) {
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].SkillName != skills[j].SkillName {
			return skills[i].SkillName < skills[j].SkillName
		}
		if skills[i].Scope != skills[j].Scope {
			return skills[i].Scope < skills[j].Scope
		}
		if a, b := formatAgentHosts(skills[i].AgentHostIDs), formatAgentHosts(skills[j].AgentHostIDs); a != b {
			return a < b
		}
		return skills[i].Path < skills[j].Path
	})
}

func displayOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func formatAgentHosts(agentHostIDs []string) string {
	if len(agentHostIDs) == 0 {
		return "-"
	}
	if len(agentHostIDs) == 1 && agentHostIDs[0] == agentHostPublished {
		return agentHostPublishedDisplay
	}
	return strings.Join(agentHostIDs, ", ")
}
