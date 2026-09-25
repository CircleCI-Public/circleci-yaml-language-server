package parser

import (
	"context"
	"fmt"
	"log/slog"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"golang.org/x/mod/semver"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

// OrbQuery is a resolved orb version: the version itself, its YAML source, and
// the sibling versions its orb has published.
type OrbQuery struct {
	Id      string
	Version string
	Orb     struct {
		Id       string
		Versions []struct {
			Version string
		}
	}
	Source string
}

// GetOrbInfo returns the remote orb a reference such as "circleci/go@1.7.1"
// names, resolving it only when the cache has no answer for it. Concurrent
// callers for the same reference share one resolution.
func GetOrbInfo(orbVersionCode string, cache *cache.Cache, context *session.Settings) (*ast.OrbInfo, error) {
	orb, err := cache.OrbCache.Load(orbVersionCode, func() (*ast.OrbInfo, error) {
		return fetchOrbInfo(orbVersionCode, cache, context)
	})
	if err != nil {
		return &ast.OrbInfo{}, err
	}

	return orb, nil
}

// ParseRemoteOrbs resolves the remote orbs a config names, so that they are in
// hand by the time the config is validated.
func ParseRemoteOrbs(orbs map[string]ast.Orb, cache *cache.Cache, context *session.Settings) {
	for _, orb := range orbs {
		if orb.Url.IsLocal || orb.Url.IsURL {
			continue
		}

		if _, err := GetOrbInfo(orb.Url.GetOrbID(), cache, context); err != nil {
			slog.Warn("fetching remote orb", "orb", orb.Url.GetOrbID(), "err", err)
		}
	}
}

func fetchOrbInfo(orbVersionCode string, cache *cache.Cache, context *session.Settings) (*ast.OrbInfo, error) {
	orbQuery, err := GetRemoteOrb(orbVersionCode, context.Api.Token, context.Api.HostUrl, context.UserIdForTelemetry)
	if err != nil {
		return nil, err
	}

	parsedOrbSource, err := ParseFromContent([]byte(orbQuery.Source), context, uri.File(""), protocol.Position{})
	if err != nil {
		return nil, err
	}
	defer parsedOrbSource.Close()

	// Go-to-definition into the orb needs a file to open.
	filePath, err := cache.WriteOrbSource(orbVersionCode, orbQuery.Source)
	if err != nil {
		return nil, err
	}

	latest, latestMinor, latestPatch := GetVersionInfo(
		orbQuery.Orb.Versions,
		"v"+orbQuery.Version,
	)

	return &ast.OrbInfo{
		OrbParsedAttributes: parsedOrbSource.ToOrbParsedAttributes(),
		Description:         parsedOrbSource.Description,
		Source:              orbQuery.Source,
		IsLocal:             false,

		RemoteInfo: ast.RemoteOrbInfo{
			ID:                 orbQuery.Id,
			FilePath:           filePath,
			Version:            orbQuery.Version,
			LatestVersion:      latest[1:],
			LatestMinorVersion: latestMinor[1:],
			LatestPatchVersion: latestPatch[1:],
		},
	}, nil
}

/**
 * List all versions provided and return the latest minor, patch and major corresponding
 * to the current version.
 *
 * Notice: all versions number should have the format "v1.2.3"
 *
 * Return format: latest, latestMinor, latestMajor
 *
 * Where:
 *   - latest: Latest version published
 *   - latestMinor: Latest minor version published with the same major version
 *   - latestPatch: Latest patch published with the same major.minor version
 */
func GetVersionInfo(
	versions []struct{ Version string },
	initialVersion string,
) (string, string, string) {
	latest := initialVersion
	latestMinor := latest
	latestPatch := latest

	major := semver.Major(latest)
	minor := semver.MajorMinor(latest)

	for _, version := range versions {
		current := "v" + version.Version

		if semver.Compare(current, latest) == 1 {
			latest = current
		}

		if semver.Major(current) != major {
			continue
		}

		if semver.Compare(current, latestMinor) == 1 {
			latestMinor = current
		}

		if semver.MajorMinor(current) != minor {
			continue
		}

		if semver.Compare(current, latestPatch) == 1 {
			latestPatch = current
		}
	}

	return latest, latestMinor, latestPatch
}

// GetRemoteOrb resolves an orb reference such as "circleci/go@1.7.1" and
// returns the version it names, that version's YAML source, and the versions
// its orb has published.
//
// Exact versions, partial versions ("circleci/go@1.7"), "volatile" and
// development tags all resolve.
func GetRemoteOrb(orbId string, token string, hostUrl, userId string) (OrbQuery, error) {
	registry := circleci.NewOrbRegistry(hostUrl, token, userId, false)

	resolved, err := registry.ResolveVersion(context.Background(), orbId)
	if err != nil {
		if circleci.IsNotFound(err) {
			// validateSingleOrb keys off this prefix to report an unknown
			// version rather than an unknown orb.
			return OrbQuery{}, fmt.Errorf("could not find orb %s", orbId)
		}

		return OrbQuery{}, err
	}

	orbQuery := OrbQuery{
		Id:      resolved.ID,
		Version: resolved.Version,
		Source:  resolved.Source,
	}
	orbQuery.Orb.Id = resolved.OrbPackageID
	orbQuery.Orb.Versions = toVersionList(resolved.Versions)

	return orbQuery, nil
}

// toVersionList adapts the API's versions to the shape GetVersionInfo takes.
func toVersionList(versions []circleci.OrbPackageVersion) []struct{ Version string } {
	list := make([]struct{ Version string }, 0, len(versions))
	for _, version := range versions {
		list = append(list, struct{ Version string }{Version: version.Version})
	}

	return list
}
