package remote

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// ErrNoGitRemote indicates that the repository does not contain a Bitbucket
// remote we can infer defaults from.
var ErrNoGitRemote = errors.New("no Bitbucket git remote found")

// Locator represents a repository identifier derived from a git remote.
type Locator struct {
	Host       string
	Kind       string // dc | cloud
	Workspace  string
	ProjectKey string
	RepoSlug   string
}

// Detect attempts to infer the locator from git remotes.
func Detect(repoPath string) (Locator, error) {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		repoPath = "."
	}

	remotes, err := listGitRemotes(repoPath)
	if err != nil {
		return Locator{}, err
	}
	if len(remotes) == 0 {
		return Locator{}, ErrNoGitRemote
	}

	var tried []string
	appendIfMissing := func(values []string, candidate string) []string {
		for _, v := range values {
			if v == candidate {
				return values
			}
		}
		return append(values, candidate)
	}

	var urls []string
	for _, name := range []string{"origin", "upstream"} {
		if candidates, ok := remotes[name]; ok {
			for _, candidate := range candidates {
				urls = appendIfMissing(urls, candidate)
			}
			tried = append(tried, name)
		}
	}

	for name, candidates := range remotes {
		skipped := false
		for _, t := range tried {
			if t == name {
				skipped = true
				break
			}
		}
		if skipped {
			continue
		}
		for _, candidate := range candidates {
			urls = appendIfMissing(urls, candidate)
		}
	}

	for _, raw := range urls {
		loc, err := parseLocator(raw)
		if err != nil {
			continue
		}
		if loc.RepoSlug == "" {
			continue
		}
		return loc, nil
	}

	return Locator{}, ErrNoGitRemote
}

func listGitRemotes(repoPath string) (map[string][]string, error) {
	args := []string{"remote", "-v"}
	if repoPath != "." && repoPath != "" {
		args = append([]string{"-C", repoPath}, args...)
	} else {
		args = append([]string{"-C", "."}, args...)
	}

	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, ErrNoGitRemote
		}
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("git executable not found: %w", err)
		}
		return nil, fmt.Errorf("git remote -v: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	result := make(map[string][]string)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		name := fields[0]
		u := fields[1]

		existing := result[name]
		already := false
		for _, v := range existing {
			if v == u {
				already = true
				break
			}
		}
		if !already {
			result[name] = append(result[name], u)
		}
	}

	return result, nil
}

func parseLocator(raw string) (Locator, error) {
	host, segments, err := dissectRemote(raw)
	if err != nil {
		return Locator{}, err
	}
	if len(segments) < 2 {
		return Locator{}, fmt.Errorf("remote %q missing repository segments", raw)
	}

	loc := Locator{
		Host: host,
	}

	if host == "bitbucket.org" {
		loc.Kind = "cloud"
		loc.Workspace = segments[0]
		loc.RepoSlug = segments[1]
		return loc, nil
	}

	project, repo := extractDCProjectRepo(segments)
	if project == "" || repo == "" {
		return Locator{}, fmt.Errorf("unable to parse Bitbucket Data Center remote %q", raw)
	}

	loc.Kind = "dc"
	loc.ProjectKey = strings.ToUpper(project)
	loc.RepoSlug = repo
	return loc, nil
}

func dissectRemote(raw string) (string, []string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil, fmt.Errorf("empty remote URL")
	}

	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", nil, fmt.Errorf("parse remote: %w", err)
		}
		host := hostWithoutPort(u.Host)
		path := cleanPath(u.Path)
		segments := splitSegments(path)
		return host, segments, nil
	}

	colon := strings.Index(raw, ":")
	if colon == -1 {
		return "", nil, fmt.Errorf("invalid remote URL %q", raw)
	}
	hostPart := raw[:colon]
	pathPart := raw[colon+1:]

	if at := strings.LastIndex(hostPart, "@"); at != -1 {
		hostPart = hostPart[at+1:]
	}

	host := hostWithoutPort(hostPart)
	segments := splitSegments(pathPart)
	return host, segments, nil
}

func cleanPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/")
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}
	if idx := strings.Index(path, "#"); idx != -1 {
		path = path[:idx]
	}
	path = strings.TrimSuffix(path, "/")
	return path
}

func splitSegments(path string) []string {
	rawSegments := strings.FieldsFunc(path, func(r rune) bool { return r == '/' })
	var segments []string
	for _, seg := range rawSegments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		segments = append(segments, seg)
	}
	if len(segments) == 0 {
		return segments
	}

	last := segments[len(segments)-1]
	last = strings.TrimSuffix(last, ".git")
	segments[len(segments)-1] = last
	return segments
}

func extractDCProjectRepo(segments []string) (string, string) {
	if len(segments) >= 4 && strings.EqualFold(segments[0], "projects") && strings.EqualFold(segments[2], "repos") {
		return segments[1], segments[3]
	}
	if len(segments) >= 3 && strings.EqualFold(segments[0], "scm") {
		return segments[1], segments[2]
	}
	if len(segments) >= 2 {
		return segments[0], segments[1]
	}
	return "", ""
}

func hostWithoutPort(host string) string {
	host = strings.TrimSpace(host)
	host = strings.Trim(host, "[]")
	if host == "" {
		return host
	}
	if strings.Contains(host, ":") {
		if parsed, _, err := net.SplitHostPort(host); err == nil {
			host = parsed
		}
	}
	return strings.ToLower(host)
}
