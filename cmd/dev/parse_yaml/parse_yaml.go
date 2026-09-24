package main

import (
	// "fmt"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

func main() {
	filepath := ".circleci/config.yml"
	// filepath := "examples/config1.yml"
	// filepath := "/home/adib/circleci/circle/.circleci/config.yml"

	schemaRef := flag.String("schema", "", "Location of the schema (optional, uses built-in schema if not provided)")

	flag.Parse()

	// If no schema is provided via flag or env, the embedded schema will be used.
	schema := *schemaRef
	if schema == "" {
		schema = os.Getenv("SCHEMA_LOCATION")
	}

	content, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Printf("Unable to read file \"%s\"", filepath)
		panic(err)
	}
	context := &session.Settings{
		Api: circleci.Config{
			Token:   "XXXXXXXXXXXX",
			HostUrl: "https://circleci.com",
		},
	}

	doc := yamlparser.ParseFile(content, context)
	doc.Close()

	c := cache.New()
	c.FileCache.SetFile(cache.File{
		TextDocument: protocol.TextDocumentItem{
			URI:  fileURI(filepath),
			Text: string(content),
		},
		Project:      circleci.Project{},
		EnvVariables: make([]string, 0),
	})

	if _, err := languageservice.DiagnosticFile(fileURI(filepath), c, context, schema); err != nil {
		panic(err)
	}

	// fmt.Printf("S-expression:\n%v\n\n", node.RootNode)
}

// fileURI is the URI of a file named by a path relative to the working
// directory, which a file URI cannot hold.
func fileURI(path string) uri.URI {
	if absolute, err := filepath.Abs(path); err == nil {
		path = absolute
	}
	return uri.File(path)
}
