package dockerhub

import (
	"fmt"
	"net/http"
)

func (me *dockerHubAPI) DoesImageExist(namespace, image string) bool {
	// A quick win is to check locally first, just in case we already found the image
	ns := hubNamespaces[namespace]

	if ns != nil && ns.hasLoaded {
		repo, _ := findFirstByName(&ns.allRepositories, image)

		if repo != nil {
			return true
		}
	}

	url := me.baseURL.JoinPath(
		fmt.Sprintf("namespaces/%s/repositories/%s", namespace, image),
	)

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
