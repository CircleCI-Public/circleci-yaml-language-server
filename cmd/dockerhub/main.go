package main

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/dockerhub"
)

func main() {

	results := dockerhub.Search("cimg/")
	if !results.HasNext() {
		fmt.Println("No images found")
		panic("No images found")
	}

	for results.HasNext() {
		fmt.Println(results.Next())
	}

	fmt.Println(dockerhub.NewAPI().DoesImageExist("cimg", "node1"))

}
