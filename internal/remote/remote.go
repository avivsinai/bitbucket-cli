package remote

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
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
	// RemoteHost is the host as written in the git remote when SSH alias
	// resolution replaced it in Host; empty otherwise.
	RemoteHost string
}

// Detect attempts to infer the locator from git remotes.
func Detect(repoPath string) (Locator, error) {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		repoPath = "."
	}

	remotes, err := ListRemotes(repoPath)
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
		loc, err := ParseRemoteLocator(raw)
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

// ListRemotes returns a map of remote names to their URLs by parsing `git remote -v`.
func ListRemotes(repoPath string) (map[string][]string, error) {
	args := []string{"remote", "-v"}
	if repoPath != "." && repoPath != "" {
		args = append([]string{"-C", repoPath}, args...)
	} else {
		args = append([]string{"-C", "."}, args...)
	}

	// Add timeout to prevent hanging on corrupted repos or network mounts
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("git executable not found: %w", err)
		}
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("git remote -v timed out after 30s")
		}
		if message := strings.TrimSpace(string(out)); message != "" {
			return nil, fmt.Errorf("git remote -v: %s", message)
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

// ParseLocator parses a git remote URL into a Locator containing the host,
// kind, and repository identifiers.
func ParseLocator(raw string) (Locator, error) {
	return parseLocator(raw, nil)
}

// ParseRemoteLocator is like ParseLocator but first resolves SSH host aliases
// through the user's OpenSSH configuration (`ssh -G`), so a remote such as
// git@work.bitbucket.org:team/repo.git maps to the host git actually reaches.
// Use it for URLs read from git remotes; arguments typed by the user should
// keep going through ParseLocator.
func ParseRemoteLocator(raw string) (Locator, error) {
	return parseLocator(raw, resolveSSHHost)
}

func parseLocator(raw string, resolveSSH func(string) string) (Locator, error) {
	host, segments, isSSH, err := dissectRemote(raw)
	if err != nil {
		return Locator{}, err
	}
	remoteHost := ""
	if isSSH && resolveSSH != nil {
		if resolved := resolveSSH(host); resolved != host {
			remoteHost, host = host, resolved
		}
	}
	// Bitbucket Cloud serves SSH from ssh.bitbucket.org as of 12 Nov 2026.
	if host == "ssh.bitbucket.org" {
		host = "bitbucket.org"
	}
	if len(segments) < 2 {
		return Locator{}, fmt.Errorf("remote %q missing repository segments", raw)
	}

	loc := Locator{
		Host:       host,
		RemoteHost: remoteHost,
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

// dissectRemote splits a remote URL into host and path segments and reports
// whether git reaches it over SSH (scp-like syntax or an ssh:// scheme).
func dissectRemote(raw string) (string, []string, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil, false, fmt.Errorf("empty remote URL")
	}

	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", nil, false, fmt.Errorf("parse remote: %w", err)
		}
		host := hostWithoutPort(u.Host)
		path := cleanPath(u.Path)
		segments := splitSegments(path)
		isSSH := false
		switch strings.ToLower(u.Scheme) {
		case "ssh", "git+ssh", "ssh+git":
			isSSH = true
		}
		return host, segments, isSSH, nil
	}

	colon := strings.Index(raw, ":")
	if colon == -1 {
		return "", nil, false, fmt.Errorf("invalid remote URL %q", raw)
	}
	hostPart := raw[:colon]
	pathPart := raw[colon+1:]

	if at := strings.LastIndex(hostPart, "@"); at != -1 {
		hostPart = hostPart[at+1:]
	}

	host := hostWithoutPort(hostPart)
	segments := splitSegments(pathPart)
	return host, segments, true, nil
}

// sshHostnameLookup returns the HostName OpenSSH would connect to for host.
// It is a variable so tests can avoid depending on the local ssh setup.
var sshHostnameLookup = lookupSSHHostname

var (
	sshHostCache   sync.Map
	sshHostPattern = regexp.MustCompile(`^[a-z0-9_][a-z0-9._-]*$`)
)

// resolveSSHHost maps an SSH host alias to its configured HostName. Lookups
// are cached per process; on any failure the host is returned unchanged.
func resolveSSHHost(host string) string {
	if host == "" || host == "bitbucket.org" || !sshHostPattern.MatchString(host) {
		return host
	}
	if cached, ok := sshHostCache.Load(host); ok {
		return cached.(string)
	}
	resolved := host
	if name, err := sshHostnameLookup(host); err == nil {
		if name = hostWithoutPort(name); name != "" {
			resolved = name
		}
	}
	sshHostCache.Store(host, resolved)
	return resolved
}

func lookupSSHHostname(host string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ssh", "-G", "--", host).Output()
	if err != nil {
		return "", fmt.Errorf("ssh -G %s: %w", host, err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if name, ok := strings.CutPrefix(strings.TrimSpace(line), "hostname "); ok {
			return strings.TrimSpace(name), nil
		}
	}
	return "", fmt.Errorf("ssh -G %s: no hostname in output", host)
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
