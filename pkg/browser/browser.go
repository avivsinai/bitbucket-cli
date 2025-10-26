package browser

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
)

// Browser opens URLs using the host operating system facilities.
type Browser interface {
	Open(url string) error
}

type system struct{}

// NewSystem returns a Browser using the platform default opener.
func NewSystem() Browser {
	return &system{}
}

// Open launches the user's browser for the provided URL. Falls back to a
// descriptive error when the platform helper is unavailable.
func (s *system) Open(url string) error {
	if url == "" {
		return errors.New("url is required")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch browser: %w", err)
	}
	return cmd.Wait()
}
