package parser

import (
	"regexp"

	"github.com/circleci/circleci-yaml-language-server/pkg/ast"
	"github.com/circleci/circleci-yaml-language-server/pkg/utils"
)

var dockerImageRegex = regexp.MustCompile(`^([a-z0-9\-_]+\/)?([a-z0-9\-_]+)(:(.*))?$`)
var aliasRemover = regexp.MustCompile(`^&[a-zA-Z0-9\-_]+\s*`)

func parseDockerImageValue(value string) ast.DockerImageInfo {
	value = aliasRemover.ReplaceAllString(value, "")
	imageName := dockerImageRegex.FindAllStringSubmatch(value, -1)

	if len(imageName) < 1 {
		return ast.DockerImageInfo{
			Namespace: "library",
			Name:      "",
			Tag:       "",
			FullPath:  value,
		}
	}

	namespace := imageName[0][1]
	repository := imageName[0][2]
	tag := imageName[0][3]

	if namespace == "" {
		namespace = "library"
	} else {
		// The regex includes the closing "/", just snip it
		namespace = namespace[:len(namespace)-1]
	}

	if tag != "" {
		// The regex includes the leading ":", just snip it
		tag = tag[1:]
	}

	// if the tag is a parameter, just consider that it's the "latest" tag for validation purposes
	if utils.CheckIfOnlyParamUsed(tag) {
		tag = "latest"
	}

	return ast.DockerImageInfo{
		Namespace: namespace,
		Name:      repository,
		Tag:       tag,
		FullPath:  value,
	}
}
