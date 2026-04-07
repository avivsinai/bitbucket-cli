// Command docgen generates skill rule files from the bkt Cobra command tree.
//
// Usage:
//
//	go run ./cmd/docgen [-o skills/bkt/rules]
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/avivsinai/bitbucket-cli/internal/docgen"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/root"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func main() {
	outDir := flag.String("o", "skills/bkt/rules", "Output directory for generated rule files")
	flag.Parse()

	f := &cmdutil.Factory{
		ExecutableName: "bkt",
		IOStreams: &iostreams.IOStreams{
			In:     io.NopCloser(strings.NewReader("")),
			Out:    io.Discard,
			ErrOut: io.Discard,
		},
	}

	rootCmd, err := root.NewCmdRoot(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build command tree: %v\n", err)
		os.Exit(1)
	}

	if err := docgen.GenerateAll(rootCmd, "bkt", *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "generate: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Generated skill rules in %s\n", *outDir)
}
