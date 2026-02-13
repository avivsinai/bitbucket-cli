package types

// CommitStatus describes build status for a commit.
// This type is shared between Bitbucket Data Center and Cloud APIs.
type CommitStatus struct {
	State       string `json:"state"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
}
