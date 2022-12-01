package main

import (
	// "fmt"

	"os"

	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func main() {
	filepath := ".circleci/config.yml"
	// filepath := "examples/config1.yml"
	// filepath := "/home/adib/circleci/circle/.circleci/config.yml"

	content, err := os.ReadFile(filepath)
	if err != nil {
		panic(err)
	}
	yamlparser.ParseFile(content)

	cache := utils.CreateCache()
	cache.FileCache.SetFile(&protocol.TextDocumentItem{
		URI:  uri.File(filepath),
		Text: string(content),
	})

	param := protocol.PublishDiagnosticsParams{
		URI: uri.File(filepath),
	}

	languageservice.Diagnostic(param, cache)

	// fmt.Printf("S-expression:\n%v\n\n", node.RootNode)
}
