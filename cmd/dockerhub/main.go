package main

import (
	"fmt"

	"github.com/circleci/circleci-yaml-language-server/pkg/dockerhub"
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

	fmt.Println(dockerhub.DoesImageExist("cimg", "node1", "18.9.0"))

}
