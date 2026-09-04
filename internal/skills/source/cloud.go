package source

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
)

// cloudWebHost is the browser host for Bitbucket Cloud repositories.
const cloudWebHost = "https://bitbucket.org"

// cloudRepository adapts a Bitbucket Cloud client to Repository.
type cloudRepository struct {
	client    *bbcloud.Client
	workspace string
	slug      string
}

// NewCloudRepository returns a Repository backed by Bitbucket Cloud.
func NewCloudRepository(client *bbcloud.Client, workspace, slug string) Repository {
	return &cloudRepository{client: client, workspace: workspace, slug: slug}
}

func (r *cloudRepository) FullName() string { return r.workspace + "/" + r.slug }
func (r *cloudRepository) WebURL() string {
	return fmt.Sprintf("%s/%s/%s", cloudWebHost, r.workspace, r.slug)
}
func (r *cloudRepository) CloneURL() string { return r.WebURL() + ".git" }

func (r *cloudRepository) Branch(ctx context.Context, name string) (string, error) {
	sha, err := r.client.GetBranch(ctx, r.workspace, r.slug, name)
	return sha, translateCloudErr(err)
}

func (r *cloudRepository) Tag(ctx context.Context, name string) (string, error) {
	sha, err := r.client.GetTag(ctx, r.workspace, r.slug, name)
	return sha, translateCloudErr(err)
}

func (r *cloudRepository) Commit(ctx context.Context, ref string) (string, error) {
	sha, err := r.client.GetCommit(ctx, r.workspace, r.slug, ref)
	return sha, translateCloudErr(err)
}

func (r *cloudRepository) LatestTag(ctx context.Context) (string, string, error) {
	tags, err := r.client.ListTags(ctx, r.workspace, r.slug, 1)
	if err != nil {
		return "", "", translateCloudErr(err)
	}
	if len(tags) == 0 {
		return "", "", ErrNotFound
	}
	return tags[0].Name, tags[0].Target.Hash, nil
}

func (r *cloudRepository) DefaultBranch(ctx context.Context) (string, error) {
	repo, err := r.client.GetRepository(ctx, r.workspace, r.slug)
	if err != nil {
		return "", translateCloudErr(err)
	}
	return repo.MainBranch.Name, nil
}

func (r *cloudRepository) ListFiles(ctx context.Context, sha, dir string) ([]File, error) {
	entries, err := r.client.ListSource(ctx, r.workspace, r.slug, sha, dir)
	if err != nil {
		return nil, translateCloudErr(err)
	}
	if len(entries) == 0 && strings.Trim(dir, "/") != "" {
		return nil, ErrNotFound
	}
	files := make([]File, 0, len(entries))
	for _, e := range entries {
		files = append(files, File{Path: e.Path, Size: e.Size})
	}
	return files, nil
}

func (r *cloudRepository) ReadFile(ctx context.Context, sha, path string) ([]byte, error) {
	content, err := r.client.ReadSource(ctx, r.workspace, r.slug, sha, path)
	return content, translateCloudErr(err)
}

func (r *cloudRepository) LatestCommit(ctx context.Context, sha, dir string) (string, error) {
	commit, err := r.client.LatestCommitForPath(ctx, r.workspace, r.slug, sha, dir)
	return commit, translateCloudErr(err)
}

// translateCloudErr maps the client's not-found sentinel onto this package's,
// so callers only need to know about source.ErrNotFound.
func translateCloudErr(err error) error {
	if err != nil && errors.Is(err, bbcloud.ErrNotFound) {
		return fmt.Errorf("%w: %s", ErrNotFound, err.Error())
	}
	return err
}

// CreateTag creates a tag pointing at a commit.
func (r *cloudRepository) CreateTag(ctx context.Context, name, commit, message string) error {
	err := r.client.CreateTag(ctx, r.workspace, r.slug, bbcloud.CreateTagInput{
		Name:    name,
		Commit:  commit,
		Message: message,
	})
	return translateCloudErr(err)
}
