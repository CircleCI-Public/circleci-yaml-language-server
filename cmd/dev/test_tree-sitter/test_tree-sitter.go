package main

import (
	"context"
	"fmt"
	"os"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

func main() {
	ctx := context.Background()
	content, _ := os.ReadFile("examples/config1.yml")
	settings := &session.Settings{
		Api: circleci.Config{
			Token:   "XXXXXXXXXXXX",
			HostUrl: "https://circleci.com",
		},
	}
	rootNode := parser.ParseFile(ctx, []byte(content), settings)
	defer rootNode.Close()

	res, err := parser.FindDeepestNode(rootNode.RootNode, content, []string{"workflows", "test-build", "jobs", "0"})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(content[res.StartByte():res.EndByte()]))
}
