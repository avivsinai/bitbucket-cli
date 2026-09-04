package source

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
)

// dcRepository adapts a Bitbucket Data Center client to Repository.
type dcRepository struct {
	client     *bbdc.Client
	baseURL    string
	projectKey string
	slug       string
}

// NewDCRepository returns a Repository backed by Bitbucket Data Center.
// baseURL is the host root, e.g. "https://bitbucket.example.com".
func NewDCRepository(client *bbdc.Client, baseURL, projectKey, slug string) Repository {
	return &dcRepository{
		client:     client,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		projectKey: projectKey,
		slug:       slug,
	}
}

func (r *dcRepository) FullName() string { return r.projectKey + "/" + r.slug }

func (r *dcRepository) WebURL() string {
	return fmt.Sprintf("%s/projects/%s/repos/%s", r.baseURL, r.projectKey, r.slug)
}

func (r *dcRepository) CloneURL() string {
	return fmt.Sprintf("%s/scm/%s/%s.git", r.baseURL, strings.ToLower(r.projectKey), r.slug)
}

func (r *dcRepository) Branch(ctx context.Context, name string) (string, error) {
	sha, err := r.client.GetBranch(ctx, r.projectKey, r.slug, name)
	return sha, translateDCErr(err)
}

func (r *dcRepository) Tag(ctx context.Context, name string) (string, error) {
	sha, err := r.client.GetTag(ctx, r.projectKey, r.slug, name)
	return sha, translateDCErr(err)
}

func (r *dcRepository) Commit(ctx context.Context, ref string) (string, error) {
	sha, err := r.client.GetCommit(ctx, r.projectKey, r.slug, ref)
	return sha, translateDCErr(err)
}

func (r *dcRepository) LatestTag(ctx context.Context) (string, string, error) {
	tags, err := r.client.ListTags(ctx, r.projectKey, r.slug, 1)
	if err != nil {
		return "", "", translateDCErr(err)
	}
	if len(tags) == 0 {
		return "", "", ErrNotFound
	}
	return tags[0].DisplayID, tags[0].Commit(), nil
}

func (r *dcRepository) DefaultBranch(ctx context.Context) (string, error) {
	branch, err := r.client.GetDefaultBranch(ctx, r.projectKey, r.slug)
	return branch, translateDCErr(err)
}

// ListFiles returns repository-root-relative paths. The Data Center endpoint
// reports paths relative to the requested directory, so the prefix is restored
// here. Sizes are not exposed by that endpoint and stay zero.
func (r *dcRepository) ListFiles(ctx context.Context, sha, dir string) ([]File, error) {
	paths, err := r.client.ListFiles(ctx, r.projectKey, r.slug, sha, dir)
	if err != nil {
		return nil, translateDCErr(err)
	}
	prefix := strings.Trim(dir, "/")
	if prefix != "" && len(paths) == 0 {
		return nil, ErrNotFound
	}
	files := make([]File, 0, len(paths))
	for _, p := range paths {
		if prefix != "" {
			p = prefix + "/" + p
		}
		files = append(files, File{Path: p})
	}
	return files, nil
}

func (r *dcRepository) ReadFile(ctx context.Context, sha, path string) ([]byte, error) {
	content, err := r.client.ReadFile(ctx, r.projectKey, r.slug, sha, path)
	return content, translateDCErr(err)
}

func (r *dcRepository) LatestCommit(ctx context.Context, sha, dir string) (string, error) {
	commit, err := r.client.LatestCommitForPath(ctx, r.projectKey, r.slug, sha, dir)
	return commit, translateDCErr(err)
}

// translateDCErr maps the client's not-found sentinel onto this package's,
// so callers only need to know about source.ErrNotFound.
func translateDCErr(err error) error {
	if err != nil && errors.Is(err, bbdc.ErrNotFound) {
		return fmt.Errorf("%w: %s", ErrNotFound, err.Error())
	}
	return err
}
