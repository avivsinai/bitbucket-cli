package installer

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/skills/discovery"
	"github.com/avivsinai/bitbucket-cli/internal/skills/frontmatter"
	"github.com/avivsinai/bitbucket-cli/internal/skills/lockfile"
	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
	"github.com/avivsinai/bitbucket-cli/internal/skills/sourcetest"
)

func newRepo() *sourcetest.Repo {
	r := sourcetest.New("myteam/agent-skills", map[string]string{
		"skills/alpha/SKILL.md":           "---\nname: alpha\ndescription: Alpha\n---\n# Alpha\n",
		"skills/alpha/scripts/run.sh":     "#!/bin/sh\necho hi\n",
		"skills/alpha/reference/notes.md": "notes",
		"skills/acme/beta/SKILL.md":       "---\nname: beta\n---\n",
	})
	r.Commits = map[string]string{"skills/alpha": "alpha-commit", "skills/acme/beta": "beta-commit"}
	return r
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return string(data)
}

func TestInstallWritesFilesMetadataAndLockfile(t *testing.T) {
	repo := newRepo()
	home := t.TempDir()
	target := filepath.Join(t.TempDir(), "skills")

	var progress [][2]int
	result, err := Install(context.Background(), &Options{
		Repo:      repo,
		Ref:       source.Ref{Ref: "refs/tags/v1.0.0", SHA: "sha1"},
		PinnedRef: "v1.0.0",
		Skills: []discovery.Skill{
			{Name: "alpha", Path: "skills/alpha", Convention: "skills"},
			{Name: "beta", Namespace: "acme", Path: "skills/acme/beta", Convention: "skills-namespaced"},
		},
		Dir:        target,
		HomeDir:    home,
		OnProgress: func(done, total int) { progress = append(progress, [2]int{done, total}) },
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !reflect.DeepEqual(result.Installed, []string{"alpha", "acme/beta"}) || result.Dir != target || len(result.Warnings) != 0 {
		t.Fatalf("result = %+v", result)
	}
	if !reflect.DeepEqual(progress, [][2]int{{0, 2}, {1, 2}, {2, 2}}) {
		t.Fatalf("progress = %v", progress)
	}

	// Flat layout: namespaced skill lands in {target}/beta, not {target}/acme/beta.
	if _, err := os.Stat(filepath.Join(target, "beta", "SKILL.md")); err != nil {
		t.Fatalf("namespaced skill not installed flat: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "acme")); err == nil {
		t.Fatal("namespace directory must not be created")
	}

	if got := readFile(t, filepath.Join(target, "alpha", "scripts", "run.sh")); got != "#!/bin/sh\necho hi\n" {
		t.Fatalf("nested file content = %q", got)
	}

	skillMD := readFile(t, filepath.Join(target, "alpha", "SKILL.md"))
	parsed, err := frontmatter.Parse(skillMD)
	if err != nil {
		t.Fatalf("parse installed SKILL.md: %v", err)
	}
	wantMeta := map[string]string{
		frontmatter.KeyRepo:   "https://bitbucket.org/myteam/agent-skills",
		frontmatter.KeyRef:    "refs/tags/v1.0.0",
		frontmatter.KeyCommit: "alpha-commit",
		frontmatter.KeyPath:   "skills/alpha",
		frontmatter.KeyPinned: "v1.0.0",
	}
	for k, v := range wantMeta {
		if parsed.Metadata.Meta[k] != v {
			t.Errorf("metadata %s = %v, want %q", k, parsed.Metadata.Meta[k], v)
		}
	}
	if parsed.Body != "# Alpha\n" {
		t.Errorf("body = %q", parsed.Body)
	}

	lock := readFile(t, lockfile.Path(home))
	for _, want := range []string{`"version": 3`, `"acme/beta"`, `"sourceType": "bitbucket"`, `"skillFolderHash": "alpha-commit"`, `"skillPath": "skills/alpha/SKILL.md"`, `"pinnedRef": "v1.0.0"`, `"sourceUrl": "https://bitbucket.org/myteam/agent-skills.git"`} {
		if !strings.Contains(lock, want) {
			t.Errorf("lock file missing %s:\n%s", want, lock)
		}
	}
}

func TestInstallUnpinnedRecordsBranchRef(t *testing.T) {
	repo := newRepo()
	target := t.TempDir()

	result, err := Install(context.Background(), &Options{
		Repo:    repo,
		Ref:     source.Ref{Ref: "refs/heads/main", SHA: "sha-main"},
		Skills:  []discovery.Skill{{Name: "alpha", Path: "skills/alpha"}},
		Dir:     target,
		HomeDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if result.Dir != target {
		t.Fatalf("Dir = %q, want %q", result.Dir, target)
	}
	skillMD := readFile(t, filepath.Join(target, "alpha", "SKILL.md"))
	if !strings.Contains(skillMD, "bitbucket-ref: refs/heads/main") {
		t.Error("unpinned install should record the branch ref")
	}
	if strings.Contains(skillMD, "bitbucket-pinned") {
		t.Error("unpinned install must not write bitbucket-pinned")
	}
}

func TestInstallErrors(t *testing.T) {
	t.Run("missing destination directory", func(t *testing.T) {
		_, err := Install(context.Background(), &Options{Repo: newRepo()})
		if err == nil || err.Error() != "destination directory is required" {
			t.Fatalf("error = %v", err)
		}
		if _, err := InstallLocal(&LocalOptions{SourceDir: t.TempDir()}); err == nil || err.Error() != "destination directory is required" {
			t.Fatalf("InstallLocal error = %v", err)
		}
	})

	t.Run("blocks path traversal from repository listing", func(t *testing.T) {
		repo := newRepo()
		repo.Files["skills/alpha/../../evil.txt"] = "pwned"
		target := t.TempDir()
		_, err := Install(context.Background(), &Options{
			Repo:    repo,
			Ref:     source.Ref{Ref: "refs/heads/main", SHA: "sha"},
			Skills:  []discovery.Skill{{Name: "alpha", Path: "skills/alpha"}},
			Dir:     target,
			HomeDir: t.TempDir(),
		})
		if err == nil || !strings.Contains(err.Error(), "blocked path traversal") {
			t.Fatalf("error = %v, want path traversal block", err)
		}
		if _, statErr := os.Stat(filepath.Join(filepath.Dir(target), "evil.txt")); statErr == nil {
			t.Fatal("file escaped the skill directory")
		}
	})

	t.Run("partial result on failure keeps earlier installs", func(t *testing.T) {
		repo := newRepo()
		repo.Files["skills/gamma/SKILL.md"] = "---\n: bad [[\n---\n"
		target := t.TempDir()
		result, err := Install(context.Background(), &Options{
			Repo:    repo,
			Ref:     source.Ref{Ref: "refs/heads/main", SHA: "sha"},
			Skills:  []discovery.Skill{{Name: "alpha", Path: "skills/alpha"}, {Name: "gamma", Path: "skills/gamma"}},
			Dir:     target,
			HomeDir: t.TempDir(),
		})
		if err == nil || !strings.Contains(err.Error(), `failed to install skill "gamma": could not inject metadata`) {
			t.Fatalf("error = %v", err)
		}
		if !reflect.DeepEqual(result.Installed, []string{"alpha"}) {
			t.Fatalf("Installed = %v", result.Installed)
		}
	})

	t.Run("lock file failure is a warning, not an error", func(t *testing.T) {
		repo := newRepo()
		result, err := Install(context.Background(), &Options{
			Repo:   repo,
			Ref:    source.Ref{Ref: "refs/heads/main", SHA: "sha"},
			Skills: []discovery.Skill{{Name: "alpha", Path: "skills/alpha"}},
			Dir:    t.TempDir(),
		})
		if err != nil {
			t.Fatalf("Install: %v", err)
		}
		if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "could not record install for alpha") {
			t.Fatalf("warnings = %v", result.Warnings)
		}
	})
}

func TestInstallLocalCopiesAndInjectsLocalPath(t *testing.T) {
	src := t.TempDir()
	skillSrc := filepath.Join(src, "skills", "alpha")
	if err := os.MkdirAll(filepath.Join(skillSrc, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte("---\nname: alpha\nmetadata:\n    bitbucket-repo: stale\n---\n# Alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillSrc, "scripts", "run.sh"), []byte("echo"), 0o644); err != nil {
		t.Fatal(err)
	}

	target := t.TempDir()
	result, err := InstallLocal(&LocalOptions{
		SourceDir: src,
		Skills:    []discovery.Skill{{Name: "alpha", Path: "skills/alpha"}},
		Dir:       target,
	})
	if err != nil {
		t.Fatalf("InstallLocal: %v", err)
	}
	if !reflect.DeepEqual(result.Installed, []string{"alpha"}) {
		t.Fatalf("Installed = %v", result.Installed)
	}
	skillMD := readFile(t, filepath.Join(target, "alpha", "SKILL.md"))
	absSrc, _ := filepath.Abs(skillSrc)
	if !strings.Contains(skillMD, "local-path: "+absSrc) && !strings.Contains(skillMD, "local-path: '"+absSrc+"'") {
		t.Fatalf("SKILL.md missing local-path %q:\n%s", absSrc, skillMD)
	}
	if strings.Contains(skillMD, "bitbucket-repo") {
		t.Fatal("stale bitbucket metadata must be stripped on local install")
	}
	if got := readFile(t, filepath.Join(target, "alpha", "scripts", "run.sh")); got != "echo" {
		t.Fatalf("nested file = %q", got)
	}
}

func TestInstallReplacesDestinationWithoutFollowingSymlinks(t *testing.T) {
	install := func(t *testing.T, target string) {
		t.Helper()
		_, err := Install(context.Background(), &Options{
			Repo:    newRepo(),
			Ref:     source.Ref{Ref: "refs/heads/main", SHA: "sha"},
			Skills:  []discovery.Skill{{Name: "alpha", Path: "skills/alpha"}},
			Dir:     target,
			HomeDir: t.TempDir(),
		})
		if err != nil {
			t.Fatalf("Install: %v", err)
		}
	}

	t.Run("skill directory symlink", func(t *testing.T) {
		target, outside := t.TempDir(), t.TempDir()
		if err := os.Symlink(outside, filepath.Join(target, "alpha")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}

		install(t, target)

		if _, err := os.Stat(filepath.Join(outside, "SKILL.md")); !os.IsNotExist(err) {
			t.Fatalf("outside destination was modified: %v", err)
		}
		if info, err := os.Lstat(filepath.Join(target, "alpha")); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("installed skill is not a real directory: info=%v err=%v", info, err)
		}
	})

	t.Run("nested symlink and stale file", func(t *testing.T) {
		target, outside := t.TempDir(), t.TempDir()
		skillDir := filepath.Join(target, "alpha")
		if err := os.Mkdir(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(skillDir, "scripts")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "stale.txt"), []byte("stale"), 0o644); err != nil {
			t.Fatal(err)
		}

		install(t, target)

		if _, err := os.Stat(filepath.Join(outside, "run.sh")); !os.IsNotExist(err) {
			t.Fatalf("outside destination was modified: %v", err)
		}
		if _, err := os.Stat(filepath.Join(skillDir, "stale.txt")); !os.IsNotExist(err) {
			t.Fatalf("stale file remains: %v", err)
		}
		if got := readFile(t, filepath.Join(skillDir, "scripts", "run.sh")); got != "#!/bin/sh\necho hi\n" {
			t.Fatalf("nested file = %q", got)
		}
	})
}

func TestInstallFailureLeavesExistingDestinationIntact(t *testing.T) {
	target := t.TempDir()
	skillDir := filepath.Join(target, "alpha")
	if err := os.Mkdir(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("previous"), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := newRepo()
	repo.Files["skills/alpha/SKILL.md"] = "---\n: invalid [[\n---\n"
	_, err := Install(context.Background(), &Options{
		Repo:    repo,
		Ref:     source.Ref{Ref: "refs/heads/main", SHA: "sha"},
		Skills:  []discovery.Skill{{Name: "alpha", Path: "skills/alpha"}},
		Dir:     target,
		HomeDir: t.TempDir(),
	})
	if err == nil || !strings.Contains(err.Error(), "could not inject metadata") {
		t.Fatalf("error = %v, want metadata failure", err)
	}
	if got := readFile(t, filepath.Join(skillDir, "SKILL.md")); got != "previous" {
		t.Fatalf("existing destination changed to %q", got)
	}
}

func TestInstallLocalReplacesDestinationWithoutFollowingSymlinks(t *testing.T) {
	sourceRoot := t.TempDir()
	skillSource := filepath.Join(sourceRoot, "skills", "alpha")
	if err := os.MkdirAll(filepath.Join(skillSource, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillSource, "SKILL.md"), []byte("---\nname: alpha\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillSource, "scripts", "run.sh"), []byte("echo"), 0o644); err != nil {
		t.Fatal(err)
	}

	install := func(t *testing.T, target string) {
		t.Helper()
		if _, err := InstallLocal(&LocalOptions{
			SourceDir: sourceRoot,
			Skills:    []discovery.Skill{{Name: "alpha", Path: "skills/alpha"}},
			Dir:       target,
		}); err != nil {
			t.Fatalf("InstallLocal: %v", err)
		}
	}

	t.Run("skill directory symlink", func(t *testing.T) {
		target, outside := t.TempDir(), t.TempDir()
		if err := os.Symlink(outside, filepath.Join(target, "alpha")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}

		install(t, target)

		if _, err := os.Stat(filepath.Join(outside, "SKILL.md")); !os.IsNotExist(err) {
			t.Fatalf("outside destination was modified: %v", err)
		}
		if _, err := os.Stat(filepath.Join(target, "alpha", "SKILL.md")); err != nil {
			t.Fatalf("local skill not installed: %v", err)
		}
	})

	t.Run("nested symlink", func(t *testing.T) {
		target, outside := t.TempDir(), t.TempDir()
		skillDir := filepath.Join(target, "alpha")
		if err := os.Mkdir(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(skillDir, "scripts")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}

		install(t, target)

		if _, err := os.Stat(filepath.Join(outside, "run.sh")); !os.IsNotExist(err) {
			t.Fatalf("outside destination was modified: %v", err)
		}
		if got := readFile(t, filepath.Join(skillDir, "scripts", "run.sh")); got != "echo" {
			t.Fatalf("nested file = %q", got)
		}
	})
}

func TestSafeJoin(t *testing.T) {
	base := filepath.Join("base", "skill")
	tests := []struct {
		rel     string
		wantErr bool
	}{
		{"SKILL.md", false},
		{"scripts/run.sh", false},
		{"../evil", true},
		{"a/../../evil", true},
		{"/abs", true},
		{"", true},
		// A backslash is a path separator on Windows, so these must be blocked
		// even though they are a single slash-separated segment.
		{`..\..\evil.txt`, true},
		{`a\..\..\evil`, true},
	}
	for _, tt := range tests {
		t.Run(tt.rel, func(t *testing.T) {
			got, err := safeJoin(base, tt.rel)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("safeJoin: %v", err)
			}
			if !strings.HasPrefix(got, base+string(filepath.Separator)) {
				t.Fatalf("result %q not under %q", got, base)
			}
		})
	}
}

func TestResolveHelpers(t *testing.T) {
	if ResolveHomeDir() == "" {
		t.Fatal("ResolveHomeDir returned empty")
	}
	root := ResolveGitRoot(context.Background())
	if root == "" {
		t.Fatal("ResolveGitRoot returned empty")
	}
	if !filepath.IsAbs(root) {
		t.Fatalf("ResolveGitRoot = %q, want absolute path", root)
	}
}
