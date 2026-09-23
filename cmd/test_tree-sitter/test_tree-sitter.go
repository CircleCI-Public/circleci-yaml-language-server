package main

import (
	"fmt"
	"os"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
)

func main() {
	content, _ := os.ReadFile("examples/config1.yml")
	context := &session.Settings{
		Api: circleci.Config{
			Token:   "XXXXXXXXXXXX",
			HostUrl: "https://circleci.com",
		},
	}
	rootNode := yamlparser.ParseFile([]byte(content), context)
	defer rootNode.Close()

	res, err := yamlparser.FindDeepestNode(rootNode.RootNode, content, []string{"workflows", "test-build", "jobs", "0"})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(content[res.StartByte():res.EndByte()]))
}
