package bbcloud

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

// IssueAttachment represents an attachment on a Bitbucket Cloud issue.
type IssueAttachment struct {
	Name  string `json:"name"`
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

type issueAttachmentListPage struct {
	Values []IssueAttachment `json:"values"`
	Next   string            `json:"next"`
}

// ListIssueAttachments lists attachments for an issue.
func (c *Client) ListIssueAttachments(ctx context.Context, workspace, repoSlug string, issueID int) ([]IssueAttachment, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/issues/%d/attachments",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		issueID,
	)

	var attachments []IssueAttachment
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page issueAttachmentListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		attachments = append(attachments, page.Values...)

		if page.Next == "" {
			break
		}

		nextURL, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		path = nextURL.RequestURI()
	}

	return attachments, nil
}

// UploadIssueAttachment uploads a file attachment to an issue.
func (c *Client) UploadIssueAttachment(ctx context.Context, workspace, repoSlug string, issueID int, filename string, r io.Reader) (*IssueAttachment, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if filename == "" {
		return nil, fmt.Errorf("filename is required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/issues/%d/attachments",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		issueID,
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
		return nil, err
	}

	// The API returns an array of attachments on upload
	var attachments []IssueAttachment
	if err := c.http.Do(req, &attachments); err != nil {
		return nil, err
	}

	if len(attachments) == 0 {
		return nil, fmt.Errorf("upload succeeded but no attachment returned")
	}

	return &attachments[0], nil
}

// DownloadIssueAttachment downloads an attachment from an issue to the provided writer.
// The API returns a 302 redirect which the http.Client follows automatically.
func (c *Client) DownloadIssueAttachment(ctx context.Context, workspace, repoSlug string, issueID int, filename string, w io.Writer) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}
	if filename == "" {
		return fmt.Errorf("filename is required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/issues/%d/attachments/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		issueID,
		url.PathEscape(filename),
	)

	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	// Override Accept header for binary content
	req.Header.Set("Accept", "*/*")

	return c.http.Do(req, w)
}

// DeleteIssueAttachment deletes an attachment from an issue.
func (c *Client) DeleteIssueAttachment(ctx context.Context, workspace, repoSlug string, issueID int, filename string) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}
	if filename == "" {
		return fmt.Errorf("filename is required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/issues/%d/attachments/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		issueID,
		url.PathEscape(filename),
	)

	req, err := c.http.NewRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}
