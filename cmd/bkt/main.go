package main

import (
	"os"

	"github.com/avivsinai/bitbucket-cli/internal/bktcmd"
)

func main() {
	os.Exit(bktcmd.Main())
}
