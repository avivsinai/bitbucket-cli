package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// repositoryBackend is the normalized, fakeable seam used by repository
// handlers. Platform client types never cross into tool handlers.
type repositoryBackend interface {
	listRepositories(context.Context, string, int) ([]Repository, bool, error)
	getRepository(context.Context, RepositoryRef) (Repository, error)
}

type platformBackend interface {
	repositoryBackend
	listPullRequests(context.Context, RepositoryRef, string, string, int) ([]PullRequest, bool, error)
	listMyPullRequests(context.Context, string, string, string, int) ([]PullRequest, bool, error)
}

// pullRequestReadBackend is the normalized seam for the C.2b detail tools.
// Tool handlers never receive raw platform client types.
type pullRequestReadBackend interface {
	getPullRequest(context.Context, RepositoryRef, int) (PullRequest, error)
	getPullRequestDiff(context.Context, RepositoryRef, int) (Diff, error)
	listPullRequestComments(context.Context, RepositoryRef, int, int) ([]Comment, bool, error)
	getPullRequestChecks(context.Context, RepositoryRef, int) ([]Check, bool, error)
}

type fullPlatformBackend interface {
	platformBackend
	pullRequestReadBackend
}

type dcBackend struct {
	client   *bbdc.Client
	username string
}

type cloudBackend struct {
	client *bbcloud.Client
}

var (
	_ pullRequestReadBackend = (*dcBackend)(nil)
	_ pullRequestReadBackend = (*cloudBackend)(nil)
)

func newPlatformBackend(snap *Snapshot) (fullPlatformBackend, error) {
	if snap == nil {
		return nil, fmt.Errorf("MCP snapshot is required")
	}
	host := snap.Host
	switch snap.Platform {
	case "dc":
		client, err := cmdutil.NewDCClient(&host)
		if err != nil {
			return nil, fmt.Errorf("create Bitbucket Data Center client: %w", err)
		}
		return &dcBackend{client: client, username: host.Username}, nil
	case "cloud":
		client, err := cmdutil.NewFrozenCloudClient(&host)
		if err != nil {
			return nil, fmt.Errorf("create Bitbucket Cloud client: %w", err)
		}
		return &cloudBackend{client: client}, nil
	default:
		return nil, fmt.Errorf("unsupported MCP platform %q", snap.Platform)
	}
}

func (b *dcBackend) listRepositories(ctx context.Context, scope string, limit int) ([]Repository, bool, error) {
	page, err := b.client.ListRepositoriesPage(ctx, scope, limit, 0)
	if err != nil {
		return nil, false, err
	}
	items := make([]Repository, 0, len(page.Values))
	for _, raw := range page.Values {
		items = append(items, adaptDCRepository(raw))
	}
	return items, !page.IsLast, nil
}

func (b *dcBackend) getRepository(ctx context.Context, locator RepositoryRef) (Repository, error) {
	raw, err := b.client.GetRepository(ctx, locator.Scope, locator.Slug)
	if err != nil {
		return Repository{}, err
	}
	return adaptDCRepository(*raw), nil
}

func (b *cloudBackend) listRepositories(ctx context.Context, scope string, limit int) ([]Repository, bool, error) {
	page, err := b.client.ListRepositoriesPage(ctx, scope, limit, "")
	if err != nil {
		return nil, false, err
	}
	items := make([]Repository, 0, len(page.Values))
	for _, raw := range page.Values {
		items = append(items, adaptCloudRepository(raw))
	}
	return items, page.Next != "", nil
}

func (b *cloudBackend) getRepository(ctx context.Context, locator RepositoryRef) (Repository, error) {
	raw, err := b.client.GetRepository(ctx, locator.Scope, locator.Slug)
	if err != nil {
		return Repository{}, err
	}
	return adaptCloudRepository(*raw), nil
}

func (b *dcBackend) getPullRequest(ctx context.Context, locator RepositoryRef, id int) (PullRequest, error) {
	raw, err := b.client.GetPullRequest(ctx, locator.Scope, locator.Slug, id)
	if err != nil {
		return PullRequest{}, err
	}
	return adaptDCPullRequest(*raw, true)
}

func (b *cloudBackend) getPullRequest(ctx context.Context, locator RepositoryRef, id int) (PullRequest, error) {
	raw, err := b.client.GetPullRequest(ctx, locator.Scope, locator.Slug, id)
	if err != nil {
		return PullRequest{}, err
	}
	return adaptCloudPullRequest(*raw, true)
}

func (b *dcBackend) getPullRequestDiff(ctx context.Context, locator RepositoryRef, id int) (Diff, error) {
	pr, err := b.client.GetPullRequest(ctx, locator.Scope, locator.Slug, id)
	if err != nil {
		return Diff{}, err
	}
	if pr.FromRef.LatestCommit == "" || pr.ToRef.LatestCommit == "" {
		return Diff{}, fmt.Errorf("pull request source and target commits are required")
	}
	sink := newBoundedTextSink(DiffContentLimit)
	if err := b.client.PullRequestDiff(ctx, locator.Scope, locator.Slug, id, sink); err != nil {
		return Diff{}, err
	}
	return Diff{
		Content:      sink.boundedText(),
		SourceCommit: pr.FromRef.LatestCommit,
		TargetCommit: pr.ToRef.LatestCommit,
	}, nil
}

func (b *cloudBackend) getPullRequestDiff(ctx context.Context, locator RepositoryRef, id int) (Diff, error) {
	pr, err := b.client.GetPullRequest(ctx, locator.Scope, locator.Slug, id)
	if err != nil {
		return Diff{}, err
	}
	if pr.Source.Commit.Hash == "" || pr.Destination.Commit.Hash == "" {
		return Diff{}, fmt.Errorf("pull request source and target commits are required")
	}
	sink := newBoundedTextSink(DiffContentLimit)
	if err := b.client.PullRequestDiff(ctx, locator.Scope, locator.Slug, id, sink); err != nil {
		return Diff{}, err
	}
	return Diff{
		Content:      sink.boundedText(),
		SourceCommit: pr.Source.Commit.Hash,
		TargetCommit: pr.Destination.Commit.Hash,
	}, nil
}

func (b *dcBackend) listPullRequestComments(ctx context.Context, locator RepositoryRef, id, limit int) ([]Comment, bool, error) {
	limit = normalizedListLimit(limit)
	pageSize := min(limit+1, MaxListLimit)
	start := 0
	items := make([]Comment, 0, limit)
	for {
		page, err := b.client.ListPullRequestCommentsPage(ctx, locator.Scope, locator.Slug, id, pageSize, start)
		if err != nil {
			return nil, false, err
		}
		for _, raw := range page.Values {
			if len(items) == limit {
				return items, true, nil
			}
			item, err := adaptDCComment(raw)
			if err != nil {
				return nil, false, err
			}
			items = append(items, item)
		}
		if page.IsLast {
			return items, false, nil
		}
		if page.NextStart <= start {
			return nil, false, fmt.Errorf("bitbucket Data Center activity pagination did not advance")
		}
		start = page.NextStart
	}
}

func (b *cloudBackend) listPullRequestComments(ctx context.Context, locator RepositoryRef, id, limit int) ([]Comment, bool, error) {
	limit = normalizedListLimit(limit)
	page, err := b.client.ListPullRequestCommentsPage(ctx, locator.Scope, locator.Slug, id, limit, "")
	if err != nil {
		return nil, false, err
	}
	items := make([]Comment, 0, len(page.Values))
	for _, raw := range page.Values {
		item, err := adaptCloudComment(raw)
		if err != nil {
			return nil, false, err
		}
		items = append(items, item)
	}
	return items, page.Next != "" || len(items) > limit, nil
}

func (b *dcBackend) getPullRequestChecks(ctx context.Context, locator RepositoryRef, id int) ([]Check, bool, error) {
	pr, err := b.client.GetPullRequest(ctx, locator.Scope, locator.Slug, id)
	if err != nil {
		return nil, false, err
	}
	if pr.FromRef.LatestCommit == "" {
		return nil, false, fmt.Errorf("pull request source commit is required")
	}
	page, err := b.client.CommitStatusesPage(ctx, pr.FromRef.LatestCommit, MaxListLimit, 0)
	if err != nil {
		return nil, false, err
	}
	items := make([]Check, 0, len(page.Values))
	for _, raw := range page.Values {
		items = append(items, adaptDCCheck(raw))
	}
	return items, !page.IsLast || len(items) >= MaxListLimit, nil
}

func (b *cloudBackend) getPullRequestChecks(ctx context.Context, locator RepositoryRef, id int) ([]Check, bool, error) {
	pr, err := b.client.GetPullRequest(ctx, locator.Scope, locator.Slug, id)
	if err != nil {
		return nil, false, err
	}
	if pr.Source.Commit.Hash == "" {
		return nil, false, fmt.Errorf("pull request source commit is required")
	}
	sourceScope, sourceSlug, ok := strings.Cut(strings.TrimSpace(pr.Source.Repository.FullName), "/")
	sourceScope = strings.TrimSpace(sourceScope)
	sourceSlug = strings.TrimSpace(sourceSlug)
	if !ok || sourceScope == "" || sourceSlug == "" {
		return nil, false, fmt.Errorf("pull request source repository identity is incomplete")
	}
	if slug := strings.TrimSpace(pr.Source.Repository.Slug); slug != "" {
		sourceSlug = slug
	}
	page, err := b.client.CommitStatusesPage(ctx, sourceScope, sourceSlug, pr.Source.Commit.Hash, MaxListLimit, "")
	if err != nil {
		return nil, false, err
	}
	items := make([]Check, 0, len(page.Values))
	for _, raw := range page.Values {
		items = append(items, adaptCloudCheck(raw))
	}
	return items, page.Next != "", nil
}

type boundedTextSink struct {
	limit int
	text  strings.Builder
	size  int
}

func newBoundedTextSink(limit int) *boundedTextSink {
	return &boundedTextSink{limit: max(limit, 0)}
}

func (w *boundedTextSink) Write(p []byte) (int, error) {
	w.size += len(p)
	retain := w.limit + utf8.UTFMax
	if remaining := retain - w.text.Len(); remaining > 0 {
		_, _ = w.text.Write(p[:min(len(p), remaining)])
	}
	return len(p), nil
}

func (w *boundedTextSink) boundedText() BoundedText {
	bounded := boundBitbucketText(w.text.String(), w.limit)
	if bounded.Truncated || w.size > w.limit {
		originalSize := w.size
		bounded.Truncated = true
		bounded.OriginalSize = &originalSize
	}
	return bounded
}

func (b *dcBackend) listPullRequests(ctx context.Context, locator RepositoryRef, state, role string, limit int) ([]PullRequest, bool, error) {
	if role != "ALL" && strings.TrimSpace(b.username) == "" {
		return nil, false, newToolError(
			ErrorInvalidInput,
			"repo-scoped role filtering needs a username in the frozen Data Center context; re-authenticate with a username, use role=all, or use bkt_list_my_pull_requests",
			false,
		)
	}
	opts := bbdc.RepoPullRequestsOptions{State: state, Limit: limit}
	if role != "ALL" {
		opts.Role = role
		opts.Username = b.username
	}
	page, err := b.client.ListRepoPullRequestsPage(ctx, locator.Scope, locator.Slug, opts)
	if err != nil {
		return nil, false, err
	}
	items, err := adaptDCPullRequests(page.Values)
	if err != nil {
		return nil, false, err
	}
	return items, !page.IsLast, nil
}

func (b *dcBackend) listMyPullRequests(ctx context.Context, _ string, state, role string, limit int) ([]PullRequest, bool, error) {
	page, err := b.client.ListDashboardPullRequestsPage(ctx, bbdc.DashboardPullRequestsOptions{
		State: state,
		Role:  role,
		Limit: limit,
	}, 0)
	if err != nil {
		return nil, false, err
	}
	items, err := adaptDCPullRequests(page.Values)
	if err != nil {
		return nil, false, err
	}
	return items, !page.IsLast, nil
}

func (b *cloudBackend) listPullRequests(ctx context.Context, locator RepositoryRef, state, role string, limit int) ([]PullRequest, bool, error) {
	opts := bbcloud.PullRequestListOptions{State: state, Limit: limit}
	if role != "ALL" {
		identity, err := b.repositoryUserIdentity(ctx)
		if err != nil {
			return nil, false, err
		}
		if role == "AUTHOR" {
			opts.Mine = identity
		} else {
			opts.Reviewer = identity
		}
	}
	page, err := b.client.ListRepoPullRequestsPage(ctx, locator.Scope, locator.Slug, opts, "")
	if err != nil {
		return nil, false, err
	}
	items, err := adaptCloudPullRequests(page.Values)
	if err != nil {
		return nil, false, err
	}
	return items, page.Next != "", nil
}

func (b *cloudBackend) listMyPullRequests(ctx context.Context, scope, state, role string, limit int) ([]PullRequest, bool, error) {
	if role != "AUTHOR" {
		return nil, false, fmt.Errorf("workspace pull requests on Bitbucket Cloud support only author role")
	}
	identity, err := b.workspaceUserIdentity(ctx)
	if err != nil {
		return nil, false, err
	}
	page, err := b.client.ListWorkspacePullRequestsPage(ctx, scope, identity, bbcloud.WorkspacePullRequestsOptions{
		State: state,
		Limit: limit,
	}, "")
	if err != nil {
		return nil, false, err
	}
	items, err := adaptCloudPullRequests(page.Values)
	if err != nil {
		return nil, false, err
	}
	return items, page.Next != "", nil
}

func (b *cloudBackend) repositoryUserIdentity(ctx context.Context) (string, error) {
	user, err := b.client.CurrentUser(ctx)
	if err != nil {
		return "", err
	}
	if identity := bbcloud.NormalizeUUID(user.UUID); identity != "" {
		return identity, nil
	}
	if user.AccountID != "" {
		return user.AccountID, nil
	}
	return "", fmt.Errorf("authenticated Bitbucket Cloud user has no stable repository-filter identity")
}

func (b *cloudBackend) workspaceUserIdentity(ctx context.Context) (string, error) {
	user, err := b.client.CurrentUser(ctx)
	if err != nil {
		return "", err
	}
	if identity := bbcloud.NormalizeUUID(user.UUID); identity != "" {
		return identity, nil
	}
	if user.AccountID != "" {
		return user.AccountID, nil
	}
	return "", fmt.Errorf("authenticated Bitbucket Cloud user has no stable workspace-filter identity")
}

func adaptDCPullRequests(raw []bbdc.PullRequest) ([]PullRequest, error) {
	items := make([]PullRequest, 0, len(raw))
	for _, item := range raw {
		adapted, err := adaptDCPullRequest(item, false)
		if err != nil {
			return nil, err
		}
		items = append(items, adapted)
	}
	return items, nil
}

func adaptCloudPullRequests(raw []bbcloud.PullRequest) ([]PullRequest, error) {
	items := make([]PullRequest, 0, len(raw))
	for _, item := range raw {
		adapted, err := adaptCloudPullRequest(item, false)
		if err != nil {
			return nil, err
		}
		items = append(items, adapted)
	}
	return items, nil
}
