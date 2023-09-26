package main

import (
	"fmt"
	"os"

	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func main() {
	content, _ := os.ReadFile("examples/config1.yml")
	context := &utils.LsContext{
		Api: utils.ApiContext{
			Token:   "XXXXXXXXXXXX",
			HostUrl: "https://circleci.com",
		},
	}
	rootNode := yamlparser.ParseFile([]byte(content), context)

	res, err := yamlparser.FindDeepestNode(rootNode.RootNode, content, []string{"workflows", "test-build", "jobs", "0"})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(content[res.StartByte():res.EndByte()]))
}
