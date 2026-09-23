package dockerhub

import (
	"fmt"
	"net/http"
)

func (me *dockerHubAPI) DoesImageExist(namespace, image string) bool {
	// A quick win is to check locally first, just in case we already found the image
	ns := me.namespaces[namespace]

	if ns != nil && ns.hasLoaded {
		repo, _ := findFirstByName(&ns.allRepositories, image)

		if repo != nil {
			return true
		}
	}

	url := me.baseURL.JoinPath(
		fmt.Sprintf("namespaces/%s/repositories/%s", namespace, image),
	)

	status, err := me.get(url.String(), nil)

	return err == nil && status == http.StatusOK
}
