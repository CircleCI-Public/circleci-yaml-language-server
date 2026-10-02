package languageservice

import (
	"context"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

// DocumentLinks link each orb the config declares to its page in the orb
// registry, or to its source when it is declared by URL, and each Docker Hub
// image to its page there.
func DocumentLinks(
	ctx context.Context, params protocol.DocumentLinkParams, cache *cache.Cache, context *session.Settings,
) ([]protocol.DocumentLink, error) {
	doc, err := yamlparser.ParseFromUriWithCache(ctx, params.TextDocument.URI, cache, context)
	if err != nil {
		return nil, err
	}
	defer doc.Close()

	links := []protocol.DocumentLink{}
	add := func(rng protocol.Range, target, tooltip string) {
		link := uri.URI(target)
		links = append(links, protocol.DocumentLink{Range: rng, Target: &link, Tooltip: &tooltip})
	}

	for _, orb := range doc.Orbs {
		switch {
		case orb.Url.IsLocal:
		case orb.Url.IsURL:
			if rng, ok := nameWithin(doc.Content, orb.ValueRange, orb.Url.Name); ok {
				add(rng, orb.Url.Name, "Open the orb's source")
			}
		// The registry's pages are circleci.com's, and so are only of its
		// orbs.
		case context.Api.UseDefaultInstance():
			if rng, ok := nameWithin(doc.Content, orb.ValueRange, orbReference(orb.Url)); ok {
				add(rng, orbRegistryPage(orb.Url), "Open "+orb.Url.Name+" in the orb registry")
			}
		}
	}

	for _, image := range dockerImages(doc) {
		if page, ok := dockerHubPage(image.Image); ok {
			add(image.ImageValueRange, page, "Open "+image.Image.Namespace+"/"+image.Image.Name+" on Docker Hub")
		}
	}

	slices.SortFunc(links, func(a, b protocol.DocumentLink) int { return position.Compare(a.Range.Start, b.Range.Start) })
	return links, nil
}

// orbReference is the orb as a config declares it, such as
// circleci/node@5.0.0.
func orbReference(orb ast2.OrbURL) string {
	if orb.Version == "" {
		return orb.Name
	}
	return orb.Name + "@" + orb.Version
}

var fullVersion = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// orbRegistryPage is the orb's page in the orb registry, at its version when
// that is a whole one rather than a range such as 5 or volatile.
func orbRegistryPage(orb ast2.OrbURL) string {
	page := url.URL{Scheme: "https", Host: "circleci.com", Path: "/developer/orbs/orb/" + orb.Name}
	if fullVersion.MatchString(orb.Version) {
		page.RawQuery = url.Values{"version": {orb.Version}}.Encode()
	}
	return page.String()
}

// dockerImages are the images of every executor and job.
func dockerImages(doc yamlparser.YamlDocument) []ast2.DockerImage {
	var images []ast2.DockerImage
	for _, executor := range doc.Executors {
		if docker, ok := executor.(ast2.DockerExecutor); ok {
			images = append(images, docker.Image...)
		}
	}
	for _, job := range doc.Jobs {
		images = append(images, job.Docker.Image...)
	}
	return images
}

// dockerHubPage is the image's page on Docker Hub. An image whose name holds
// anything that isn't plain, such as a registry's host or a parameter, has
// none.
func dockerHubPage(image ast2.DockerImageInfo) (string, bool) {
	if image.Name == "" || strings.Contains(image.FullPath, "<<") {
		return "", false
	}
	if image.Namespace == "library" {
		return "https://hub.docker.com/_/" + image.Name, true
	}
	return "https://hub.docker.com/r/" + image.Namespace + "/" + image.Name, true
}
