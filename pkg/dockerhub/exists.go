package dockerhub

import (
	"fmt"
	"net/http"
)

func DoesImageExist(namespace, image, tag string) bool {
	// A quick win is to check locally first, just in case we already found the image
	ns := hubNamespaces[namespace]

	if ns != nil && ns.hasLoaded {
		repo, _ := findFirstByName(&ns.allRepositories, image)

		if repo != nil {
			return true
		}
	}

	url := baseURL.JoinPath(
		fmt.Sprintf("namespaces/%s/repositories/%s", namespace, image),
	)

	if tag != "" && tag != "latest" {
		// "Latest" is a keyword and is not fetchable via API.
		// in that case, the image is valid as long as a repo with the name exists.

		url = url.JoinPath(
			fmt.Sprintf("/tags/%s", tag),
		)
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return false
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}

	return res.StatusCode == 200
}
