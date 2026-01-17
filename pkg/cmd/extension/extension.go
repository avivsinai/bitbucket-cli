package extension

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/example/bitbucket-cli/pkg/cmdutil"
)

// NewCmdExtension manages external bkt extensions.
func NewCmdExtension(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extension",
		Short: "Manage bkt CLI extensions",
	}

	cmd.AddCommand(newInstallCmd(f))
	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newRemoveCmd(f))
	cmd.AddCommand(newExecCmd(f))

	return cmd
}

func newInstallCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <repository>",
		Short: "Install an extension from a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExtensionInstall(cmd, f, args[0])
		},
	}
	return cmd
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List installed extensions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExtensionList(cmd, f)
		},
	}
	return cmd
}

func newRemoveCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Short:   "Remove an installed extension",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExtensionRemove(cmd, f, args[0])
		},
	}
	return cmd
}

func newExecCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec <name> [args...]",
		Short: "Execute an installed extension",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExtensionExec(cmd, f, args[0], args[1:])
		},
	}
	return cmd
}

func runExtensionInstall(cmd *cobra.Command, f *cmdutil.Factory, repo string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	root, err := ensureExtensionRoot(f)
	if err != nil {
		return err
	}

	name := inferExtensionName(repo)
	if name == "" {
		return fmt.Errorf("unable to infer extension name from %q", repo)
	}

	destination := filepath.Join(root, name)
	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("extension %q is already installed", name)
	}

	args := []string{"clone", repo, destination}
	gitCmd := exec.CommandContext(cmd.Context(), "git", args...)
	gitCmd.Stdout = ios.Out
	gitCmd.Stderr = ios.ErrOut
	gitCmd.Stdin = ios.In

	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}
	execPath, err := findExtensionExecutable(destination, name)
	if err != nil {
		if _, warnErr := fmt.Fprintf(ios.ErrOut, "warning: %v\n", err); warnErr != nil {
			return warnErr
		}
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Installed extension %s\n", name); err != nil {
		return err
	}
	if execPath != "" {
		rel, _ := filepath.Rel(root, execPath)
		if _, err := fmt.Fprintf(ios.Out, "  binary: %s\n", rel); err != nil {
			return err
		}
	}
	return nil
}

func runExtensionList(cmd *cobra.Command, f *cmdutil.Factory) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	root, err := extensionRoot(f)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		entries = nil
	} else if err != nil {
		return err
	}

	type extensionSummary struct {
		Name       string `json:"name"`
		Path       string `json:"path"`
		Executable string `json:"executable,omitempty"`
	}

	var summaries []extensionSummary
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		dir := filepath.Join(root, name)
		execPath, _ := findExtensionExecutable(dir, name)
		rel := ""
		if execPath != "" {
			rel, _ = filepath.Rel(root, execPath)
		}
		summaries = append(summaries, extensionSummary{
			Name:       name,
			Path:       dir,
			Executable: rel,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	data := struct {
		Extensions []extensionSummary `json:"extensions"`
	}{Extensions: summaries}

	return cmdutil.WriteOutput(cmd, ios.Out, data, func() error {
		if len(summaries) == 0 {
			_, err := fmt.Fprintln(ios.Out, "No extensions installed. Use `bkt extension install <repository>` to add one.")
			return err
		}
		for _, ext := range summaries {
			line := ext.Name
			if ext.Executable != "" {
				line = fmt.Sprintf("%s\t%s", ext.Name, ext.Executable)
			}
			if _, err := fmt.Fprintln(ios.Out, line); err != nil {
				return err
			}
		}
		return nil
	})
}

func runExtensionRemove(cmd *cobra.Command, f *cmdutil.Factory, name string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	root, err := extensionRoot(f)
	if err != nil {
		return err
	}

	dir := filepath.Join(root, name)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("extension %q is not installed", name)
	} else if err != nil {
		return err
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove extension: %w", err)
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Removed extension %s\n", name); err != nil {
		return err
	}
	return nil
}

func runExtensionExec(cmd *cobra.Command, f *cmdutil.Factory, name string, args []string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	root, err := extensionRoot(f)
	if err != nil {
		return err
	}

	dir := filepath.Join(root, name)
	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("extension %q is not installed", name)
		}
		return err
	}

	execPath, err := findExtensionExecutable(dir, name)
	if err != nil {
		return err
	}

	cmdExec := exec.CommandContext(cmd.Context(), execPath, args...)
	cmdExec.Stdout = ios.Out
	cmdExec.Stderr = ios.ErrOut
	cmdExec.Stdin = ios.In
	cmdExec.Dir = dir
	cmdExec.Env = append(os.Environ(),
		fmt.Sprintf("BKT_EXTENSION_DIR=%s", dir),
		fmt.Sprintf("BKT_EXTENSION_NAME=%s", name),
	)

	return cmdExec.Run()
}

func extensionRoot(f *cmdutil.Factory) (string, error) {
	cfg, err := f.ResolveConfig()
	if err != nil {
		return "", err
	}
	path := cfg.Path()
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("configuration path unknown; run `bkt auth login` first")
	}
	return filepath.Join(filepath.Dir(path), "extensions"), nil
}

func ensureExtensionRoot(f *cmdutil.Factory) (string, error) {
	root, err := extensionRoot(f)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	return root, nil
}

func inferExtensionName(repo string) string {
	trimmed := strings.TrimSpace(repo)
	trimmed = strings.TrimSuffix(trimmed, ".git")
	trimmed = strings.TrimSuffix(trimmed, "/")

	delim := strings.LastIndexAny(trimmed, "/:")
	if delim != -1 {
		trimmed = trimmed[delim+1:]
	}

	trimmed = strings.TrimPrefix(trimmed, "bkt-")
	return strings.TrimSpace(trimmed)
}

func findExtensionExecutable(dir, name string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var candidates []string
	prefix := fmt.Sprintf("bkt-%s", name)
	for _, entry := range entries {
		if entry.IsDir() {
			// consider bin/ subdirectory
			if entry.Name() == "bin" {
				subEntries, err := os.ReadDir(filepath.Join(dir, "bin"))
				if err != nil {
					continue
				}
				for _, sub := range subEntries {
					if sub.IsDir() {
						continue
					}
					if strings.HasPrefix(sub.Name(), prefix) && isExecutable(sub) {
						candidates = append(candidates, filepath.Join(dir, "bin", sub.Name()))
					}
				}
			}
			continue
		}
		if strings.HasPrefix(entry.Name(), prefix) && isExecutable(entry) {
			candidates = append(candidates, filepath.Join(dir, entry.Name()))
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no executable matching %q found in %s", prefix, dir)
	}

	sort.Strings(candidates)
	return candidates[0], nil
}

func isExecutable(entry os.DirEntry) bool {
	info, err := entry.Info()
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}
	mode := info.Mode()
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(info.Name()))
		return ext == ".exe" || ext == ".bat" || ext == ".cmd" || ext == ".ps1"
	}
	return mode&0o111 != 0
}
