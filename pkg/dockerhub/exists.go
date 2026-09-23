package dockerhub

import (
	"fmt"
)

func (me *dockerHubAPI) DoesImageExist(namespace, image string) (bool, error) {
	// A quick win is to check locally first, just in case we already found the image
	if ns := me.knownNamespace(namespace); ns != nil && ns.hasRepository(image) {
		return true, nil
	}

	url := me.baseURL.JoinPath(
		fmt.Sprintf("namespaces/%s/repositories/%s", namespace, image),
	)

	return me.exists(url.String())
}
