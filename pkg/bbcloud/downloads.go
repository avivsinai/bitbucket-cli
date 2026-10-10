package bbcloud

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

// Download represents a file in a repository's Downloads section.
type Download struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedOn string `json:"created_on"`
	Links     struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

type downloadListPage struct {
	Values []Download `json:"values"`
	Next   string     `json:"next"`
}

// ListDownloads lists files in a repository's Downloads section.
func (c *Client) ListDownloads(ctx context.Context, workspace, repoSlug string) ([]Download, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/downloads",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)

	var downloads []Download
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page downloadListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		downloads = append(downloads, page.Values...)

		if page.Next == "" {
			break
		}

		nextURL, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		path = nextURL.RequestURI()
	}

	return downloads, nil
}

// UploadDownload uploads a file to a repository's Downloads section. An existing
// file with the same name is replaced. The API answers 201 with an empty body.
func (c *Client) UploadDownload(ctx context.Context, workspace, repoSlug, filename string, r io.Reader) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}
	if filename == "" {
		return fmt.Errorf("filename is required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/downloads",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)

	files := []httpx.MultipartFile{
		{
			FieldName: "files",
			FileName:  filename,
			Reader:    r,
		},
	}

	req, err := c.http.NewMultipartRequest(ctx, "POST", path, files)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// DownloadURL returns the browser URL for a file in a repository's Downloads section.
func DownloadURL(workspace, repoSlug, filename string) string {
	return fmt.Sprintf("https://bitbucket.org/%s/%s/downloads/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(filename),
	)
}
