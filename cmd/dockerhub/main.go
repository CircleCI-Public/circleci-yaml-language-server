package main

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func main() {

	results := utils.SearchDockerHUB("cimg/")
	if !results.HasNext() {
		fmt.Println("No images found")
		panic("No images found")
	}

	for results.HasNext() {
		fmt.Println(results.Next())
	}

	fmt.Println(utils.DoesDockerImageExist("cimg", "node1", "18.9.0"))

}
