package main

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func main() {
	orbname := "circleci/go@1.7.1"

	cl := utils.NewClient("https://circleci.com", "graphql-unstable", "", false)

	query := `query($orbVersionRef: String!) {
		orbVersion(orbVersionRef: $orbVersionRef) {
			id
						version
						orb { id }
						source
		}
	  }`
	request := utils.NewRequest(query)
	request.Var("orbVersionRef", orbname)
	var response struct {
		OrbVersion OrbVersion
	}
	err := cl.Run(request, &response)

	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	fmt.Println(response.OrbVersion)
}

type OrbVersion struct {
	ID        string
	Version   string
	Source    string
	CreatedAt string
}
