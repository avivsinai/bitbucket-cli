package skill

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// startProgress shows a spinner while a long operation runs. The spinner is
// only used when stderr is a terminal so piped output stays clean; the
// returned function stops it.
func startProgress(f *cmdutil.Factory, ios *iostreams.IOStreams, label string) func() {
	if !ios.IsStderrTTY() {
		return func() {}
	}
	spinner := f.ProgressSpinner()
	spinner.Start(label)
	return func() { spinner.Stop("") }
}

// shortSHA abbreviates a commit hash for display.
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func pluralize(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// sanitizeForTerminal replaces control characters in frontmatter-provided
// strings so they cannot inject terminal escape sequences.
func sanitizeForTerminal(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// friendlyDir shortens an absolute directory for display: relative to the
// working directory when inside it, "~/..." when inside the home directory.
func friendlyDir(dir string) string {
	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, dir); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			if rel == "." {
				return filepath.Base(dir)
			}
			return rel
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		if rel, err := filepath.Rel(home, dir); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "~/" + filepath.ToSlash(rel)
		}
	}
	return dir
}

// printInstalledTree renders the on-disk contents of each installed skill.
func printInstalledTree(w io.Writer, dir string, skillNames []string) {
	if len(skillNames) == 0 {
		return
	}
	fmt.Fprintln(w)
	for _, name := range skillNames {
		// Skills are installed flat by their base name even when namespaced.
		base := name
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			base = name[idx+1:]
		}
		fmt.Fprintf(w, "  %s/\n", sanitizeForTerminal(name))
		printTreeDir(w, filepath.Join(dir, base), "  ")
	}
}

func printTreeDir(w io.Writer, dir, indent string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(w, "%s(could not read directory)\n", indent)
		return
	}
	for i, entry := range entries {
		connector, childIndent := "├── ", "│   "
		if i == len(entries)-1 {
			connector, childIndent = "└── ", "    "
		}
		// Names come from a repository, so they are neutralised before printing.
		if entry.IsDir() {
			fmt.Fprintf(w, "%s%s%s/\n", indent, connector, sanitizeForTerminal(entry.Name()))
			printTreeDir(w, filepath.Join(dir, entry.Name()), indent+childIndent)
		} else {
			fmt.Fprintf(w, "%s%s%s\n", indent, connector, sanitizeForTerminal(entry.Name()))
		}
	}
}

// treeNode represents a file or directory when rendering a remote file list.
type treeNode struct {
	name     string
	children []*treeNode
	isDir    bool
}

// printFileTree renders slash-separated file paths as a tree.
func printFileTree(w io.Writer, files []source.File) {
	root := &treeNode{isDir: true}
	for _, f := range files {
		parts := strings.Split(f.Path, "/")
		current := root
		for i, part := range parts {
			var next *treeNode
			for _, child := range current.children {
				if child.name == part {
					next = child
					break
				}
			}
			if next == nil {
				next = &treeNode{name: part, isDir: i < len(parts)-1}
				current.children = append(current.children, next)
			}
			current = next
		}
	}
	sortTree(root)
	printTreeNodes(w, root.children, "")
}

func sortTree(node *treeNode) {
	sort.Slice(node.children, func(i, j int) bool {
		if node.children[i].isDir != node.children[j].isDir {
			return node.children[i].isDir
		}
		return node.children[i].name < node.children[j].name
	})
	for _, child := range node.children {
		if child.isDir {
			sortTree(child)
		}
	}
}

func printTreeNodes(w io.Writer, nodes []*treeNode, indent string) {
	for i, node := range nodes {
		connector, childIndent := "├── ", "│   "
		if i == len(nodes)-1 {
			connector, childIndent = "└── ", "    "
		}
		// Names come from a repository, so they are neutralised before printing.
		if node.isDir {
			fmt.Fprintf(w, "%s%s%s/\n", indent, connector, sanitizeForTerminal(node.name))
			printTreeNodes(w, node.children, indent+childIndent)
		} else {
			fmt.Fprintf(w, "%s%s%s\n", indent, connector, sanitizeForTerminal(node.name))
		}
	}
}
