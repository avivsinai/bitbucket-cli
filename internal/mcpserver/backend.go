package mcpserver

import (
	"context"
	"fmt"
	"strings"

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

type dcBackend struct {
	client   *bbdc.Client
	username string
}

type cloudBackend struct {
	client *bbcloud.Client
}

func newPlatformBackend(snap *Snapshot) (platformBackend, error) {
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
