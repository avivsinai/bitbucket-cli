package skill

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
	"github.com/avivsinai/bitbucket-cli/internal/skills/sourcetest"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// newRepublisherRepos returns a repository whose skill was copied from another
// repository, plus that upstream repository.
func newRepublisherRepos() (republisher, upstream *sourcetest.Repo) {
	republisher = sourcetest.New("myteam/mirror", map[string]string{
		"skills/alpha/SKILL.md": "---\nname: alpha\nmetadata:\n    bitbucket-repo: https://bitbucket.org/upstream/agent-skills\n---\n# Mirror copy\n",
	})
	upstream = sourcetest.New("upstream/agent-skills", map[string]string{
		"skills/alpha/SKILL.md": "---\nname: alpha\n---\n# Upstream copy\n",
	})
	return republisher, upstream
}

func TestInstallUpstreamRedirect(t *testing.T) {
	republisher, upstream := newRepublisherRepos()

	var args []string
	original := openRepositoryFunc
	openRepositoryFunc = func(_ *cobra.Command, _ *cmdutil.Factory, arg string) (source.Repository, error) {
		args = append(args, arg)
		if strings.Contains(arg, "upstream") {
			return upstream, nil
		}
		return republisher, nil
	}
	t.Cleanup(func() { openRepositoryFunc = original })

	f, stdout, stderr := newTestFactory(t)
	target := t.TempDir()
	if err := runSkill(t, f, stdout, stderr, "install", "myteam/mirror", "alpha", "--dir", target, "--upstream"); err != nil {
		t.Fatalf("install --upstream: %v (stderr=%s)", err, stderr)
	}

	// The second open is the redirect into the upstream repository.
	if len(args) != 2 || args[0] != "myteam/mirror" || args[1] != "https://bitbucket.org/upstream/agent-skills" {
		t.Fatalf("repository arguments = %v, want the re-publisher then the upstream", args)
	}
	for _, want := range []string{"originally published in upstream/agent-skills", "Redirecting install to upstream/agent-skills"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr missing %q:\n%s", want, stderr)
		}
	}

	installed := readFile(t, filepath.Join(target, "alpha", "SKILL.md"))
	if !strings.Contains(installed, "# Upstream copy") {
		t.Fatalf("installed the re-publisher's copy instead of the upstream:\n%s", installed)
	}
	if !strings.Contains(installed, "bitbucket-repo: https://bitbucket.org/upstream/agent-skills") {
		t.Errorf("metadata should point at the upstream:\n%s", installed)
	}
}

func TestInstallWithoutUpstreamKeepsRepublisher(t *testing.T) {
	republisher, _ := newRepublisherRepos()
	args := stubRepository(t, republisher)

	f, stdout, stderr := newTestFactory(t)
	target := t.TempDir()
	if err := runSkill(t, f, stdout, stderr, "install", "myteam/mirror", "alpha", "--dir", target); err != nil {
		t.Fatalf("install: %v (stderr=%s)", err, stderr)
	}

	if len(*args) != 1 {
		t.Fatalf("repository opened %d times, want 1 without --upstream", len(*args))
	}
	if !strings.Contains(stderr.String(), "use --upstream or interactive mode to choose upstream") {
		t.Errorf("stderr should say how to switch to the upstream:\n%s", stderr)
	}
	if !strings.Contains(readFile(t, filepath.Join(target, "alpha", "SKILL.md")), "# Mirror copy") {
		t.Error("without --upstream the re-publisher's copy is installed")
	}
}

func TestInstallIgnoresSelfReferentialProvenance(t *testing.T) {
	// A skill installed from this same repository is not a re-publication.
	repo := sourcetest.New("myteam/agent-skills", map[string]string{
		"skills/alpha/SKILL.md": "---\nname: alpha\nmetadata:\n    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n---\n# Own copy\n",
	})
	args := stubRepository(t, repo)

	f, stdout, stderr := newTestFactory(t)
	if err := runSkill(t, f, stdout, stderr, "install", "myteam/agent-skills", "alpha", "--dir", t.TempDir(), "--upstream"); err != nil {
		t.Fatalf("install: %v (stderr=%s)", err, stderr)
	}
	if len(*args) != 1 {
		t.Fatalf("repository opened %d times, want 1", len(*args))
	}
	if strings.Contains(stderr.String(), "originally published in") {
		t.Errorf("a self-referential source must not trigger a redirect:\n%s", stderr)
	}
}
