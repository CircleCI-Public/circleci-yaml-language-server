package parser

import (
	"regexp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
)

var dockerImageRegex = regexp.MustCompile(`^([a-z0-9\-_]+\/)?([a-z0-9\-_]+)(:([^@]*))?(@(.+))?$`)
var aliasRemover = regexp.MustCompile(`^&[a-zA-Z0-9\-_]+\s*`)

func ParseDockerImageValue(value string) ast.DockerImageInfo {
	value = aliasRemover.ReplaceAllString(value, "")
	imageName := dockerImageRegex.FindAllStringSubmatch(value, -1)

	if len(imageName) < 1 {
		return ast.DockerImageInfo{
			Namespace: "library",
			Name:      "",
			Tag:       "",
			Digest:    "",
			FullPath:  value,
		}
	}

	namespace := imageName[0][1]
	repository := imageName[0][2]
	tag := imageName[0][4]
	digest := imageName[0][6]

	if namespace == "" {
		namespace = "library"
	} else {
		// The regex includes the closing "/", just snip it
		namespace = namespace[:len(namespace)-1]
	}

	return ast.DockerImageInfo{
		Namespace: namespace,
		Name:      repository,
		Tag:       tag,
		Digest:    digest,
		FullPath:  value,
	}
}
