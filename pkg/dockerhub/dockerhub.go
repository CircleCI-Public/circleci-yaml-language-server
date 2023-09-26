package dockerhub

import "net/url"

type DockerHubAPI interface {
	DoesImageExist(namespace, image string) bool
	GetImageTags(namespace, image string) ([]string, error)
	ImageHasTag(namespace, image, tag string) bool
}

type dockerHubAPI struct {
	baseURL url.URL
}

func NewAPI() DockerHubAPI {
	return &dockerHubAPI{baseURL: baseURL}
}
