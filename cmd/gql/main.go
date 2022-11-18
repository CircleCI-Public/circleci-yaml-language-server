package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/circleci/circleci-yaml-language-server/pkg/utils"
)

func main() {
	orbname := "circleci/go@1.7.1"

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          10,
			TLSHandshakeTimeout:   10 * time.Second,
		},
	}
	cl := utils.NewClient(httpClient, "https://circleci.com", "graphql-unstable", "", false)

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
