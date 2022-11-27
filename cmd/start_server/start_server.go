package main

import (
	"flag"
	"fmt"
	"os"

	lsp "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/server"
)

func main() {
	versionRef := flag.Bool("version", false, "display version")
	flag.Parse()

	// Parameter: version
	version := *versionRef
	if version {
		fmt.Println(lsp.GetServerVersion())
		return
	}

	// Command: stdio
	if len(os.Args) > 1 && os.Args[1] == "stdio" {
		lsp.StartServerStdio()
		return
	}

	lsp.StartServer()
}
