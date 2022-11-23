package main

import (
	"fmt"
	"os"

	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
)

func main() {
	content, _ := os.ReadFile("examples/config1.yml")
	rootNode := yamlparser.ParseFile([]byte(content))

	res, err := yamlparser.FindDeepestNode(rootNode.RootNode, content, []string{"workflows", "test-build", "jobs", "0"})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(content[res.StartByte():res.EndByte()]))
}
