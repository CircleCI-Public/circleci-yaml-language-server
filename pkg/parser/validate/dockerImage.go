package validate

import (
	"fmt"

	"github.com/circleci/circleci-yaml-language-server/pkg/ast"
	"github.com/circleci/circleci-yaml-language-server/pkg/utils"
)

func ValidateDockerImage(img *ast.DockerImage, cache *utils.DockerCache) (bool, string) {
	cachedDockerImage := cache.Get(img.Image.FullPath)

	if img.Image.Name != "" && img.Image.Tag == "" {
		return false, "Missing image tag"
	}

	if !isDockerImageCheckable(img) {
		// When a Docker image can't be checked, return true (consider it valid)
		return true, ""
	}

	if cachedDockerImage == nil {
		cache.Add(
			img.Image.FullPath,
			utils.DoesDockerImageExist(img.Image.Namespace, img.Image.Name, img.Image.Tag),
		)

		cachedDockerImage = cache.Get(img.Image.FullPath)
	}

	return cachedDockerImage.Exists, fmt.Sprintf("Docker image not found %s", img.Image.FullPath)
}

/*
Not all Docker image syntaxes are supported
Unsuported syntaxes:
  - Direct URL, not on Docker HUB (Example: 183081753049.dkr.ecr.us-east-1.amazonaws.com/circleci/ecs-test-kms:0.1)
  - Using aliases (Example: image: *my_alias)
  - When authentication is required
*/
func isDockerImageCheckable(img *ast.DockerImage) bool {
	// For now, just make the name & version mandatory
	return img.Image.Name != "" && img.Auth == ast.DockerImageAuth{} && img.AwsAuth == ast.DockerImageAWSAuth{}
}
