package main

import (
	"fmt"
	"os"

	lsp "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/server"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version":
			fmt.Println(lsp.GetServerVersion())
			return
		case "--stdio":
			lsp.StartServerStdio()
			return
		}
	}

	lsp.StartServer()
}
