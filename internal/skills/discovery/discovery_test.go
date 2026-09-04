package discovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/skills/sourcetest"
)

func TestMatchSkillConventions(t *testing.T) {
	tests := []struct {
		path       string
		name       string
		namespace  string
		convention string
	}{
		{path: "skills/code-review/SKILL.md", name: "code-review", convention: "skills"},
		{path: "skills/monalisa/issue-triage/SKILL.md", name: "issue-triage", namespace: "monalisa", convention: "skills-namespaced"},
		{path: "plugins/hubot/skills/pr-summary/SKILL.md", name: "pr-summary", namespace: "hubot", convention: "plugins"},
		{path: "code-review/SKILL.md", name: "code-review", convention: "root"},
		{path: "terraform/code-generation/skills/terraform-style-guide/SKILL.md", name: "terraform-style-guide", convention: "skills"},
		{path: "a/b/c/skills/my-skill/SKILL.md", name: "my-skill", convention: "skills"},
		{path: "terraform/code-generation/skills/hashicorp/terraform-style-guide/SKILL.md", name: "terraform-style-guide", namespace: "hashicorp", convention: "skills-namespaced"},
		{path: "packer/skills/packer-builder/SKILL.md", name: "packer-builder", convention: "skills"},
		{path: "skills/code-review/README.md"},
		{path: "skills/SKILL.md"},
		{path: "SKILL.md"},
		{path: ".hidden/SKILL.md"},
		{path: "terraform/skills/SKILL.md"},
		{path: ".claude/skills/code-review/SKILL.md"},
		{path: "vendor/plugins/hubot/skills/pr-summary/SKILL.md"},
		{path: "skills/bad name!/SKILL.md"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			m := matchSkillConventions(tt.path)
			if tt.name == "" {
				if m != nil {
					t.Fatalf("expected no match, got %+v", m)
				}
				return
			}
			if m == nil {
				t.Fatalf("expected match %q, got nil", tt.name)
			}
			if m.name != tt.name || m.namespace != tt.namespace || m.convention != tt.convention {
				t.Fatalf("got name=%q ns=%q conv=%q, want name=%q ns=%q conv=%q", m.name, m.namespace, m.convention, tt.name, tt.namespace, tt.convention)
			}
		})
	}
}

func TestMatchHiddenDirConventions(t *testing.T) {
	tests := []struct {
		path       string
		name       string
		namespace  string
		convention string
	}{
		{path: ".claude/skills/x/SKILL.md", name: "x", convention: "hidden-dir"},
		{path: ".agents/skills/x/SKILL.md", name: "x", convention: "hidden-dir"},
		{path: "foo/bar/.claude/skills/x/SKILL.md", name: "x", convention: "hidden-dir"},
		{path: ".claude/nested/skills/x/SKILL.md", name: "x", convention: "hidden-dir"},
		{path: ".claude/skills/monalisa/x/SKILL.md", name: "x", namespace: "monalisa", convention: "hidden-dir-namespaced"},
		{path: ".claude/SKILL.md"},
		{path: ".claude/code-review/SKILL.md"},
		{path: "visible/skills/x/SKILL.md"},
		{path: ".claude/skills/x/notes.md"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			m := matchHiddenDirConventions(tt.path)
			if tt.name == "" {
				if m != nil {
					t.Fatalf("expected no match, got %+v", m)
				}
				return
			}
			if m == nil {
				t.Fatalf("expected match %q, got nil", tt.name)
			}
			if m.name != tt.name || m.namespace != tt.namespace || m.convention != tt.convention {
				t.Fatalf("got name=%q ns=%q conv=%q, want name=%q ns=%q conv=%q", m.name, m.namespace, m.convention, tt.name, tt.namespace, tt.convention)
			}
		})
	}
}

func TestIsSkillPath(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"git-commit", false},
		{"SKILL.md", false},
		{"monalisa/code-review", false},
		{"myskills", false},
		{"", false},
		{"skills/code-review", true},
		{"skills/code-review/SKILL.md", true},
		{"plugins/hubot/skills/pr-summary", true},
		{"terraform/code-generation/skills/x", true},
		{"packages/agent-skills/x", true},
		{"skills-catalog/matlab-core/matlab-debugging/", true},
	}
	for _, tt := range tests {
		if got := IsSkillPath(tt.in); got != tt.want {
			t.Errorf("IsSkillPath(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestValidateNameAndSpecCompliance(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
		spec  bool
	}{
		{"code-review", true, true},
		{"Code Review", true, false},
		{"my_skill.v2", true, false},
		{"a", true, true},
		{"", false, false},
		{"-leading", false, false},
		{"double--hyphen", true, false},
		{"trailing-", true, false},
		{"has/slash", false, false},
		{"dot..dot", false, false},
		{strings.Repeat("a", 65), false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateName(tt.name); got != tt.valid {
				t.Errorf("validateName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
			if got := IsSpecCompliant(tt.name); got != tt.spec {
				t.Errorf("IsSpecCompliant(%q) = %v, want %v", tt.name, got, tt.spec)
			}
		})
	}
}

func TestDisplayAndInstallName(t *testing.T) {
	tests := []struct {
		skill       Skill
		wantDisplay string
		wantInstall string
	}{
		{Skill{Name: "x", Convention: "skills"}, "x", "x"},
		{Skill{Name: "x", Namespace: "ns", Convention: "skills-namespaced"}, "ns/x", "ns/x"},
		{Skill{Name: "x", Namespace: "hubot", Convention: "plugins"}, "[plugins] hubot/x", "hubot/x"},
		{Skill{Name: "x", Convention: "root"}, "[root] x", "x"},
		{Skill{Name: "x", Convention: "hidden-dir"}, "[hidden-dir] x", "x"},
		{Skill{Name: "x", Namespace: "ns", Convention: "hidden-dir-namespaced"}, "[hidden-dir] ns/x", "ns/x"},
	}
	for _, tt := range tests {
		if got := tt.skill.DisplayName(); got != tt.wantDisplay {
			t.Errorf("DisplayName(%+v) = %q, want %q", tt.skill, got, tt.wantDisplay)
		}
		if got := tt.skill.InstallName(); got != tt.wantInstall {
			t.Errorf("InstallName(%+v) = %q, want %q", tt.skill, got, tt.wantInstall)
		}
	}
}

func newFakeRepo() *sourcetest.Repo {
	return sourcetest.New("myteam/agent-skills", map[string]string{
		"README.md":                                       "# skills",
		"skills/zeta/SKILL.md":                            "---\nname: zeta\ndescription: Last alphabetically\n---\n",
		"skills/alpha/SKILL.md":                           "---\nname: alpha\ndescription: First skill\n---\n# Alpha\n",
		"skills/alpha/scripts/run.sh":                     "#!/bin/sh\n",
		"skills/alpha/reference/notes.md":                 "notes",
		"skills/monalisa/triage/SKILL.md":                 "---\nname: triage\ndescription: Namespaced\n---\n",
		"plugins/hubot/skills/pr-summary/SKILL.md":        "---\nname: pr-summary\n---\n",
		".claude/skills/hidden-one/SKILL.md":              "---\nname: hidden-one\n---\n",
		"docs/skills/alpha/README.md":                     "not a skill",
		"skills/broken/SKILL.md":                          "---\n: bad [[\n---\n",
		"skills/no-frontmatter/SKILL.md":                  "# just body\n",
		"a/b/c/skills/deep/SKILL.md":                      "---\ndescription: Deep\n---\n",
		"terraform/skills/hashicorp/style-guide/SKILL.md": "---\ndescription: Nested ns\n---\n",
	})
}

func TestDiscoverSkillsExcludesHiddenAndSorts(t *testing.T) {
	repo := newFakeRepo()
	skills, err := DiscoverSkills(context.Background(), repo, "sha")
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	var names []string
	for _, s := range skills {
		names = append(names, s.DisplayName())
	}
	want := []string{"[plugins] hubot/pr-summary", "alpha", "broken", "deep", "hashicorp/style-guide", "monalisa/triage", "no-frontmatter", "zeta"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("display names = %v, want %v", names, want)
	}
	if HasHiddenDirSkills(skills) {
		t.Fatal("hidden-dir skills should be excluded")
	}
	for _, s := range skills {
		if s.Description != "" {
			t.Fatalf("descriptions must not be fetched during discovery, got %q for %s", s.Description, s.Name)
		}
	}
}

func TestDiscoverAllSkillsIncludesHidden(t *testing.T) {
	repo := newFakeRepo()
	skills, err := DiscoverAllSkills(context.Background(), repo, "sha")
	if err != nil {
		t.Fatalf("DiscoverAllSkills: %v", err)
	}
	part := PartitionHiddenDirSkills(skills)
	if part.HiddenCount != 1 || len(part.Standard) != 8 {
		t.Fatalf("partition = hidden %d standard %d, want 1/8", part.HiddenCount, len(part.Standard))
	}
	if skills[0].DisplayName() != "[hidden-dir] hidden-one" {
		t.Fatalf("expected hidden skill sorted first by display name, got %q", skills[0].DisplayName())
	}
}

func TestDiscoverSkillsErrors(t *testing.T) {
	t.Run("no skills", func(t *testing.T) {
		repo := sourcetest.New("myteam/empty", map[string]string{"README.md": "x"})
		_, err := DiscoverSkills(context.Background(), repo, "sha")
		if err == nil || !strings.Contains(err.Error(), "no skills found in myteam/empty") {
			t.Fatalf("error = %v, want no-skills error", err)
		}
	})
	t.Run("only hidden skills", func(t *testing.T) {
		repo := sourcetest.New("myteam/hidden", map[string]string{".claude/skills/x/SKILL.md": "---\nname: x\n---\n"})
		_, err := DiscoverSkills(context.Background(), repo, "sha")
		if err == nil || !strings.Contains(err.Error(), "no skills found") {
			t.Fatalf("error = %v, want no-skills error", err)
		}
		all, err := DiscoverAllSkills(context.Background(), repo, "sha")
		if err != nil || len(all) != 1 {
			t.Fatalf("DiscoverAllSkills = %v, %v; want one hidden skill", all, err)
		}
	})
	t.Run("listing failure is wrapped", func(t *testing.T) {
		repo := newFakeRepo()
		repo.Err = errors.New("boom")
		_, err := DiscoverSkills(context.Background(), repo, "sha")
		if err == nil || err.Error() != "could not list repository files: boom" {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestFetchDescriptions(t *testing.T) {
	repo := newFakeRepo()
	skills, err := DiscoverSkills(context.Background(), repo, "sha")
	if err != nil {
		t.Fatal(err)
	}
	skills[0].Description = "preset"
	var progress []int
	FetchDescriptions(context.Background(), repo, "sha", skills, func(done, total int) {
		if total != len(skills)-1 {
			t.Errorf("total = %d, want %d", total, len(skills)-1)
		}
		progress = append(progress, done)
	})
	byName := map[string]string{}
	for _, s := range skills {
		byName[s.DisplayName()] = s.Description
	}
	if byName["alpha"] != "First skill" || byName["monalisa/triage"] != "Namespaced" {
		t.Fatalf("descriptions = %v", byName)
	}
	if byName["[plugins] hubot/pr-summary"] != "preset" {
		t.Fatalf("preset description must not be refetched, got %q", byName["[plugins] hubot/pr-summary"])
	}
	if byName["broken"] != "" || byName["no-frontmatter"] != "" {
		t.Fatalf("unparseable SKILL.md should yield empty description: %v", byName)
	}
	if len(progress) != len(skills)-1 {
		t.Fatalf("progress callbacks = %d, want %d", len(progress), len(skills)-1)
	}
}

func TestDiscoverSkillByPath(t *testing.T) {
	repo := newFakeRepo()
	tests := []struct {
		name      string
		path      string
		opts      DiscoverSkillByPathOptions
		want      Skill
		wantErr   string
		wantReads int
	}{
		{name: "standard path", path: "skills/alpha", want: Skill{Name: "alpha", Path: "skills/alpha", Description: "First skill"}, wantReads: 1},
		{name: "SKILL.md suffix and trailing slash trimmed", path: "skills/alpha/SKILL.md", want: Skill{Name: "alpha", Path: "skills/alpha", Description: "First skill"}, wantReads: 1},
		{name: "namespaced path infers namespace", path: "skills/monalisa/triage/", want: Skill{Name: "triage", Namespace: "monalisa", Path: "skills/monalisa/triage", Description: "Namespaced"}, wantReads: 1},
		{name: "plugins path infers convention", path: "plugins/hubot/skills/pr-summary", want: Skill{Name: "pr-summary", Namespace: "hubot", Convention: "plugins", Path: "plugins/hubot/skills/pr-summary"}, wantReads: 1},
		{name: "skip description", path: "skills/alpha", opts: DiscoverSkillByPathOptions{SkipDescription: true}, want: Skill{Name: "alpha", Path: "skills/alpha"}, wantReads: 0},
		{name: "non-standard nested path", path: "terraform/skills/hashicorp/style-guide", want: Skill{Name: "style-guide", Namespace: "hashicorp", Path: "terraform/skills/hashicorp/style-guide", Description: "Nested ns"}, wantReads: 1},
		{name: "missing directory", path: "skills/nope", wantErr: `skill directory "skills/nope" not found in myteam/agent-skills`},
		{name: "directory without SKILL.md", path: "docs/skills/alpha", wantErr: "no SKILL.md found in docs/skills/alpha"},
		{name: "invalid name", path: "skills/bad name!", wantErr: `invalid skill name "bad name!"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo.ReadFileCalls = 0
			got, err := DiscoverSkillByPath(context.Background(), repo, "sha", tt.path, tt.opts)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("DiscoverSkillByPath: %v", err)
			}
			if *got != tt.want {
				t.Fatalf("skill = %+v, want %+v", *got, tt.want)
			}
			if repo.ReadFileCalls != tt.wantReads {
				t.Fatalf("ReadFile calls = %d, want %d", repo.ReadFileCalls, tt.wantReads)
			}
		})
	}
}

func TestDiscoverSkillByPathInvalidNameRejectsTraversal(t *testing.T) {
	repo := newFakeRepo()
	_, err := DiscoverSkillByPath(context.Background(), repo, "sha", "skills/..", DiscoverSkillByPathOptions{})
	if err == nil {
		t.Fatal("expected error for traversal path")
	}
}

func TestSkillFiles(t *testing.T) {
	repo := newFakeRepo()
	files, err := SkillFiles(context.Background(), repo, "sha", "skills/alpha/")
	if err != nil {
		t.Fatalf("SkillFiles: %v", err)
	}
	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	want := []string{"SKILL.md", "reference/notes.md", "scripts/run.sh"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
	if files[0].Size == 0 {
		t.Fatal("expected file sizes to be populated")
	}

	if _, err := SkillFiles(context.Background(), repo, "sha", "skills/nope"); err == nil {
		t.Fatal("expected error for missing skill directory")
	}
}

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverLocalSkills(t *testing.T) {
	t.Run("repository layout", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "skills", "beta", "SKILL.md"), "---\nname: beta\ndescription: Beta skill\n---\n")
		writeFile(t, filepath.Join(dir, "skills", "acme", "alpha", "SKILL.md"), "---\ndescription: Namespaced\n---\n")
		writeFile(t, filepath.Join(dir, "plugins", "bot", "skills", "gamma", "SKILL.md"), "")
		writeFile(t, filepath.Join(dir, ".claude", "skills", "hidden", "SKILL.md"), "")
		writeFile(t, filepath.Join(dir, "skills", "Bad Name!", "SKILL.md"), "")
		writeFile(t, filepath.Join(dir, "README.md"), "x")

		skills, err := DiscoverLocalSkills(dir)
		if err != nil {
			t.Fatalf("DiscoverLocalSkills: %v", err)
		}
		var names []string
		for _, s := range skills {
			names = append(names, s.DisplayName()+"@"+s.Path)
		}
		want := []string{"[plugins] bot/gamma@plugins/bot/skills/gamma", "acme/alpha@skills/acme/alpha", "beta@skills/beta"}
		if !reflect.DeepEqual(names, want) {
			t.Fatalf("skills = %v, want %v", names, want)
		}
		if skills[2].Description != "Beta skill" {
			t.Fatalf("local discovery should read descriptions, got %q", skills[2].Description)
		}

		all, err := DiscoverAllLocalSkills(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(all) != 4 || !HasHiddenDirSkills(all) {
			t.Fatalf("DiscoverAllLocalSkills = %d skills (hidden=%v), want 4 incl. hidden", len(all), HasHiddenDirSkills(all))
		}
	})

	t.Run("single skill directory uses frontmatter name", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "SKILL.md"), "---\nname: renamed\ndescription: One\n---\n")
		skills, err := DiscoverLocalSkills(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(skills) != 1 || skills[0].Name != "renamed" || skills[0].Path != "." {
			t.Fatalf("skills = %+v", skills)
		}
	})

	t.Run("errors", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := DiscoverLocalSkills(dir); err == nil || !strings.Contains(err.Error(), "no skills found in") {
			t.Fatalf("empty dir error = %v", err)
		}
		if _, err := DiscoverLocalSkills(filepath.Join(dir, "missing")); err == nil || !strings.Contains(err.Error(), "could not access") {
			t.Fatalf("missing dir error = %v", err)
		}
		file := filepath.Join(dir, "file.txt")
		writeFile(t, file, "x")
		if _, err := DiscoverLocalSkills(file); err == nil || !strings.Contains(err.Error(), "is not a directory") {
			t.Fatalf("file error = %v", err)
		}
		writeFile(t, filepath.Join(dir, "SKILL.md"), "---\nname: bad/name\n---\n")
		if _, err := DiscoverLocalSkills(dir); err == nil || !strings.Contains(err.Error(), "invalid skill name") {
			t.Fatalf("invalid name error = %v", err)
		}
	})
}

func TestMatchSkillPath(t *testing.T) {
	name, ns := MatchSkillPath("skills/monalisa/triage/SKILL.md")
	if name != "triage" || ns != "monalisa" {
		t.Fatalf("got %q/%q", ns, name)
	}
	if name, ns := MatchSkillPath("docs/README.md"); name != "" || ns != "" {
		t.Fatalf("expected no match, got %q/%q", ns, name)
	}
}
