// Package installer copies skills from a Bitbucket repository or a local
// directory into an agent's skills directory.
package installer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/avivsinai/bitbucket-cli/internal/skills/discovery"
	"github.com/avivsinai/bitbucket-cli/internal/skills/frontmatter"
	"github.com/avivsinai/bitbucket-cli/internal/skills/lockfile"
	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
)

// Options configures an installation from a Bitbucket repository.
type Options struct {
	Repo       source.Repository
	Ref        source.Ref // resolved ref and commit
	PinnedRef  string     // user-supplied --pin value (empty if unpinned)
	Skills     []discovery.Skill
	Dir        string // destination directory, resolved by the caller
	HomeDir    string // user home directory, for the skill lock file
	OnProgress func(done, total int)
}

// Result tracks what was installed.
type Result struct {
	Installed []string
	Dir       string
	Warnings  []string
}

// Install fetches and writes skills to the target directory. Skills are
// installed one after another: Bitbucket's request budget is per hour, so a
// worker pool would only trade throughput for rate-limit errors.
func Install(ctx context.Context, opts *Options) (*Result, error) {
	targetDir := opts.Dir
	if targetDir == "" {
		return nil, errors.New("destination directory is required")
	}

	total := len(opts.Skills)
	if opts.OnProgress != nil {
		opts.OnProgress(0, total)
	}

	result := &Result{Dir: targetDir}
	for i, skill := range opts.Skills {
		commit, err := installSkill(ctx, opts, skill, targetDir)
		if err != nil {
			return result, fmt.Errorf("failed to install skill %q: %w", skill.InstallName(), err)
		}
		result.Installed = append(result.Installed, skill.InstallName())

		entry := lockfile.Entry{
			Source:          opts.Repo.FullName(),
			SourceURL:       opts.Repo.CloneURL(),
			SkillPath:       skill.Path + "/SKILL.md",
			SkillFolderHash: commit,
			PinnedRef:       opts.PinnedRef,
		}
		if err := lockfile.RecordInstall(opts.HomeDir, skill.InstallName(), entry); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("could not record install for %s: %v", skill.InstallName(), err))
		}

		if opts.OnProgress != nil {
			opts.OnProgress(i+1, total)
		}
	}
	return result, nil
}

// installSkill writes one skill into baseDir and returns the commit recorded
// in its metadata.
func installSkill(ctx context.Context, opts *Options, skill discovery.Skill, baseDir string) (string, error) {
	// Use skill.Name (not InstallName) for a flat layout: most agents only
	// discover immediate subdirectories of their skills folder.
	skillDir := filepath.Join(baseDir, skill.Name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return "", fmt.Errorf("could not create directory %s: %w", skillDir, err)
	}

	commit, err := opts.Repo.LatestCommit(ctx, opts.Ref.SHA, skill.Path)
	if err != nil {
		return "", fmt.Errorf("could not determine latest commit for %s: %w", skill.Path, err)
	}

	files, err := discovery.SkillFiles(ctx, opts.Repo, opts.Ref.SHA, skill.Path)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		content, err := opts.Repo.ReadFile(ctx, opts.Ref.SHA, skill.Path+"/"+file.Path)
		if err != nil {
			return "", fmt.Errorf("could not fetch %s: %w", file.Path, err)
		}

		if filepath.Base(filepath.FromSlash(file.Path)) == "SKILL.md" {
			injected, err := frontmatter.InjectBitbucketMetadata(string(content), opts.Repo.WebURL(), opts.Ref.Ref, commit, opts.PinnedRef, skill.Path)
			if err != nil {
				return "", fmt.Errorf("could not inject metadata: %w", err)
			}
			content = []byte(injected)
		}

		if err := writeSkillFile(skillDir, file.Path, content); err != nil {
			return "", err
		}
	}

	return commit, nil
}

// LocalOptions configures an installation from a local directory.
type LocalOptions struct {
	SourceDir string
	Skills    []discovery.Skill
	Dir       string // destination directory, resolved by the caller
}

// InstallLocal copies skills from a local directory to the target location.
// Files are copied, never symlinked, and no lock file entry is written.
func InstallLocal(opts *LocalOptions) (*Result, error) {
	targetDir := opts.Dir
	if targetDir == "" {
		return nil, errors.New("destination directory is required")
	}

	result := &Result{Dir: targetDir}
	for _, skill := range opts.Skills {
		if err := installLocalSkill(opts.SourceDir, skill, targetDir); err != nil {
			return nil, fmt.Errorf("failed to install skill %q: %w", skill.InstallName(), err)
		}
		result.Installed = append(result.Installed, skill.InstallName())
	}
	return result, nil
}

func installLocalSkill(sourceRoot string, skill discovery.Skill, baseDir string) error {
	skillDir := filepath.Join(baseDir, skill.Name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", skillDir, err)
	}

	srcDir := filepath.Join(sourceRoot, filepath.FromSlash(skill.Path))
	absSource, err := filepath.Abs(srcDir)
	if err != nil {
		return fmt.Errorf("could not resolve source path: %w", err)
	}

	return filepath.WalkDir(srcDir, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 || d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, p)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("could not read %s: %w", p, err)
		}

		if filepath.Base(relPath) == "SKILL.md" {
			injected, err := frontmatter.InjectLocalMetadata(string(content), absSource)
			if err != nil {
				return fmt.Errorf("could not inject metadata: %w", err)
			}
			content = []byte(injected)
		}

		return writeSkillFile(skillDir, filepath.ToSlash(relPath), content)
	})
}

// writeSkillFile writes content to relPath (slash-separated) under skillDir,
// refusing paths that would escape the skill directory.
func writeSkillFile(skillDir, relPath string, content []byte) error {
	dest, err := safeJoin(skillDir, relPath)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(dest); dir != skillDir {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("could not create directory: %w", err)
		}
	}
	if err := os.WriteFile(dest, content, 0o644); err != nil {
		return fmt.Errorf("could not write %s: %w", dest, err)
	}
	return nil
}

// safeJoin joins a slash-separated relative path onto base and verifies the
// result stays inside base.
func safeJoin(base, relPath string) (string, error) {
	if relPath == "" || strings.HasPrefix(relPath, "/") || filepath.IsAbs(filepath.FromSlash(relPath)) {
		return "", fmt.Errorf("blocked path traversal in %q", relPath)
	}
	for seg := range strings.SplitSeq(relPath, "/") {
		if seg == ".." {
			return "", fmt.Errorf("blocked path traversal in %q", relPath)
		}
	}
	dest := filepath.Join(base, filepath.FromSlash(relPath))
	rel, err := filepath.Rel(base, dest)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("blocked path traversal in %q", relPath)
	}
	return dest, nil
}

// ResolveGitRoot returns the top-level directory of the current git
// repository, falling back to the working directory when not inside one.
func ResolveGitRoot(ctx context.Context) string {
	out, err := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		if root := strings.TrimSpace(string(out)); root != "" {
			return filepath.Clean(filepath.FromSlash(root))
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return ""
}

// ResolveHomeDir returns the user's home directory, or "" on error.
func ResolveHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
