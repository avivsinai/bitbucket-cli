package main

import (
	"os"

	"github.com/example/bitbucket-cli/internal/bktcmd"
)

func main() {
	os.Exit(bktcmd.Main())
}
