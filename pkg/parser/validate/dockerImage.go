package validate

import (
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/dockerhub"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/Masterminds/semver"
	"go.lsp.dev/protocol"
)

func DoesDockerImageExists(img *ast.DockerImage, cache *utils.DockerCache, api dockerhub.DockerHubAPI) bool {
	cachedDockerImage := cache.Get(img.Image.FullPath)

	if !isDockerImageCheckable(img) {
		// When a Docker image can't be checked, return true (consider it valid)
		return true
	}

	if cachedDockerImage == nil {
		cache.Add(
			img.Image.FullPath,
			api.DoesImageExist(img.Image.Namespace, img.Image.Name),
		)

		cachedDockerImage = cache.Get(img.Image.FullPath)
	}

	return cachedDockerImage.Exists
}

/*
Not all Docker image syntaxes are supported
Unsupported syntaxes:
  - Direct URL, not on Docker HUB (Example: 183081753049.dkr.ecr.us-east-1.amazonaws.com/circleci/ecs-test-kms:0.1)
  - Using aliases (Example: image: *my_alias)
  - When authentication is required
  - When tag uses CircleCI parameter syntax
*/
func isDockerImageCheckable(img *ast.DockerImage) bool {
	// For now, just make the name & version mandatory
	hasParamInTag, _ := utils.CheckIfParamIsPartiallyReferenced(img.Image.Tag)
	return img.Image.Name != "" && img.Auth == ast.DockerImageAuth{} && img.AwsAuth == ast.DockerImageAWSAuth{} && !hasParamInTag
}

func DoesTagExist(img *ast.DockerImage, searchedTag string, cache *utils.DockerTagsCache, api dockerhub.DockerHubAPI) bool {
	tagInfo := GetImageTagInfo(img, cache, api)

	if tagInfo == nil {
		return true
	}

	tagExists, ok := tagInfo.CheckedTags[searchedTag]
	if !ok {
		tagExists = api.ImageHasTag(img.Image.Namespace, img.Image.Name, searchedTag)
		tagInfo.CheckedTags[searchedTag] = tagExists
		cache.Add(img.Image.Namespace, img.Image.Name, *tagInfo)
	}

	return tagExists
}

func GetImageTagActions(doc *parser.YamlDocument, img *ast.DockerImage, cache *utils.DockerTagsCache, api dockerhub.DockerHubAPI) []protocol.CodeAction {
	tagInfo := GetImageTagInfo(img, cache, api)
	actions := []protocol.CodeAction{}

	if tagInfo == nil {
		return actions
	}

	if tagInfo.Recommended != "" {
		actions = append(actions, utils.CreateCodeActionTextEdit(
			"Use last tag",
			doc.URI,
			[]protocol.TextEdit{
				createTagTextEdit(img, tagInfo.Recommended),
			},
			true,
		))
	}

	if DoesTagExist(img, "latest", cache, api) {
		actions = append(actions, utils.CreateCodeActionTextEdit(
			"Use 'latest'",
			doc.URI,
			[]protocol.TextEdit{
				createTagTextEdit(img, "latest"),
			},
			false,
		))
	}

	return actions
}

// Get the image tag info and fill the image info if it is not present in the cache
func GetImageTagInfo(img *ast.DockerImage, cache *utils.DockerTagsCache, api dockerhub.DockerHubAPI) *utils.CachedDockerTags {
	tagInfo := cache.Get(img.Image.Namespace, img.Image.Name)

	if tagInfo != nil {
		return tagInfo
	}
	tags, err := api.GetImageTags(img.Image.Namespace, img.Image.Name)
	if err != nil {
		return nil
	}

	tagsForCache := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tagsForCache[tag] = true
	}

	recommended := chooseTagToRecommend(tags)

	tagInfo = &utils.CachedDockerTags{
		CheckedTags: tagsForCache,
		Recommended: recommended,
	}
	cache.Add(img.Image.Namespace, img.Image.Name, *tagInfo)
	return tagInfo
}

func chooseTagToRecommend(allTags []string) string {
	// Filter 'latest' tags out

	tags := []string{}
	for _, tag := range allTags {
		if tag != "latest" {
			tags = append(tags, tag)
		}
	}

	if len(tags) == 0 {
		return ""
	}

	// Find a tag that matches semantic versioning
	//
	// We use the Masterminds semver package and not the golang.org package because it is less strict
	// about which version is accepts. It accepts versions without the preceding 'v', and it accepts
	// the 'X', 'X.Y' as well as the 'X.Y.Z' versions

	for _, tag := range tags {
		if _, err := semver.NewVersion(tag); err == nil {
			return tag
		}
	}

	// If we have not found any tag matching our needs, return the first tag

	return tags[0]
}

func createTagTextEdit(img *ast.DockerImage, tag string) protocol.TextEdit {
	imgArray := strings.Split(img.Image.FullPath, ":")
	version := imgArray[len(imgArray)-1]
	versionLength := uint32(len(version)) + 1
	return protocol.TextEdit{
		NewText: ":" + tag,
		Range: protocol.Range{
			Start: protocol.Position{
				Character: img.ImageRange.End.Character - versionLength,
				Line:      img.ImageRange.End.Line,
			},
			End: protocol.Position{
				Character: img.ImageRange.End.Character + 1 + uint32(len(tag)) - versionLength,
				Line:      img.ImageRange.End.Line,
			},
		},
	}
}
