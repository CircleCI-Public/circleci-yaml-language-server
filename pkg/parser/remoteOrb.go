package parser

import (
	"errors"
	"fmt"
	"os"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"golang.org/x/mod/semver"
)

type OrbResponse struct {
	OrbVersion OrbQuery
}

type OrbGQLData struct {
	ID   string
	Name string
}

type OrbByNameResponse struct {
	Orb OrbGQLData
}

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

func GetOrbInfo(orbVersionCode string, cache *utils.Cache, context *utils.LsContext) (*ast.OrbInfo, error) {
	// Returning cache if exists
	if !cache.OrbCache.HasOrb(orbVersionCode) {

		orb, err := fetchOrbInfo(orbVersionCode, cache, context)
		return orb, err
	}

	return cache.OrbCache.GetOrb(orbVersionCode), nil
}

func GetOrbByName(orbName string, context *utils.LsContext) (OrbGQLData, error) {
	if context.Api.HostUrl == "" {
		return OrbGQLData{}, errors.New("host URL not defined")
	}

	client := utils.NewClient(context.Api.HostUrl, "graphql-unstable", context.Api.Token, false)
	query := `
		query($orbName: String!) {
			orb(name: $orbName) {
				id
				name
			}
		}
	`

	request := utils.NewRequest(query)
	request.SetToken(client.Token)
	request.SetUserId(context.UserIdForTelemetry)
	request.Var("orbName", orbName)

	var response OrbByNameResponse
	err := client.Run(request, &response)

	if err != nil {
		return OrbGQLData{}, err
	}

	if response.Orb.Name == "" {
		return OrbGQLData{}, fmt.Errorf("Orb does not exists")
	}

	return response.Orb, nil
}

func ParseRemoteOrbs(orbs map[string]ast.Orb, cache *utils.Cache, context *utils.LsContext) {
	for _, orb := range orbs {
		if orb.Url.IsLocal {
			continue
		}

		if orb.Url.Version != "volatile" && checkIfRemoteOrbAlreadyExistsInFSCache(orb.Url.GetOrbID()) {
			err := addAlreadyExistingRemoteOrbsToFSCache(orb, cache, context)

			// If no error, we continue
			// Otherwise, we fetch again orb info
			if err == nil {
				continue
			}
		}

		fetchOrbInfo(orb.Url.GetOrbID(), cache, context)
	}
}

func fetchOrbInfo(orbVersionCode string, cache *utils.Cache, context *utils.LsContext) (*ast.OrbInfo, error) {
	orbQuery, err := GetRemoteOrb(orbVersionCode, context.Api.Token, context.Api.HostUrl, context.UserIdForTelemetry)

	if err != nil {
		return &ast.OrbInfo{}, err
	}

	parsedOrbSource, err := ParseFromContent([]byte(orbQuery.Source), context, uri.File(""), protocol.Position{})

	if err != nil {
		return &ast.OrbInfo{}, err
	}

	filePath, err := writeRemoteOrbSourceInFSCache(orbVersionCode, orbQuery.Source)

	if err != nil {
		return &ast.OrbInfo{}, err
	}

	latest, latestMinor, latestPatch := GetVersionInfo(
		orbQuery.Orb.Versions,
		"v"+orbQuery.Version,
	)

	orb := &ast.OrbInfo{
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
	}

	cache.OrbCache.SetOrb(orb, orbVersionCode)

	return orb, nil
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

func GetRemoteOrb(orbId string, token string, hostUrl, userId string) (OrbQuery, error) {
	if hostUrl == "" {
		return OrbQuery{}, errors.New("host URL not defined")
	}

	client := utils.NewClient(hostUrl, "graphql-unstable", token, false)
	query := `query($orbVersionRef: String!) {
		orbVersion(orbVersionRef: $orbVersionRef) {
			id
			version
			orb {
				id
				versions(count: 100) {
					version
				}
			}
			source
		}
	}`

	request := utils.NewRequest(query)
	request.SetToken(client.Token)
	request.SetUserId(userId)
	request.Var("orbVersionRef", orbId)

	var response OrbResponse
	err := client.Run(request, &response)

	if response.OrbVersion.Id == "" {
		return response.OrbVersion, fmt.Errorf("could not find orb %s", orbId)
	}

	return response.OrbVersion, err
}

func GetOrbVersions(orbId string, token string, hostUrl, userId string) ([]struct{ Version string }, error) {
	if hostUrl == "" {
		emptyList := make([]struct{ Version string }, 0)
		return emptyList, fmt.Errorf("host URL not defined")
	}

	client := utils.NewClient(hostUrl, "graphql-unstable", token, false)
	query := `query($orbVersionRef: String!) {
		orbVersion(orbVersionRef: $orbVersionRef) {
			version
		}
	}`

	request := utils.NewRequest(query)
	request.SetToken(client.Token)
	request.SetUserId(userId)
	request.Var("orbVersionRef", orbId)

	var response OrbResponse
	err := client.Run(request, &response)

	if response.OrbVersion.Id == "" {
		emptyList := make([]struct{ Version string }, 0)
		return emptyList, fmt.Errorf("could not find orb %s", orbId)
	}

	return response.OrbVersion.Orb.Versions, err
}

func writeRemoteOrbSourceInFSCache(orbYaml string, source string) (string, error) {
	filePath := utils.GetOrbCacheFSPath(orbYaml)
	_, err := os.Stat(filePath)

	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(os.Stderr, "Writing remote orb source in cache:", filePath)

		err = os.WriteFile(filePath, []byte(source), 0644)
		return filePath, err
	}

	return filePath, err
}

func checkIfRemoteOrbAlreadyExistsInFSCache(orbYaml string) bool {
	filePath := utils.GetOrbCacheFSPath(orbYaml)

	// Err == nil means the file exists
	_, err := os.Stat(filePath)
	return err == nil
}

func addAlreadyExistingRemoteOrbsToFSCache(orb ast.Orb, cache *utils.Cache, context *utils.LsContext) error {
	filePath := utils.GetOrbCacheFSPath(orb.Url.GetOrbID())

	content, err := os.ReadFile(filePath)

	AddOrbToCacheWithContent(orb, uri.File(filePath), content, context, cache)

	if err != nil {
		return err
	}

	return nil
}

func AddOrbToCacheWithContent(orb ast.Orb, uri protocol.URI, content []byte, context *utils.LsContext, cache *utils.Cache) error {
	parsedOrbSource, err := ParseFromContent(content, context, uri, protocol.Position{})

	if err != nil {
		return err
	}

	versions, err := GetOrbVersions(orb.Url.GetOrbID(), context.Api.Token, context.Api.HostUrl, context.UserIdForTelemetry)

	if err != nil {
		return nil
	}

	latest, latestMinor, latestPatch := GetVersionInfo(versions, "v"+orb.Url.Version)

	cache.OrbCache.SetOrb(&ast.OrbInfo{

		Description:         parsedOrbSource.Description,
		Source:              string(content),
		IsLocal:             false,
		OrbParsedAttributes: parsedOrbSource.ToOrbParsedAttributes(),

		RemoteInfo: ast.RemoteOrbInfo{
			ID:                 orb.Url.GetOrbID(),
			FilePath:           uri.Filename(),
			Version:            orb.Url.Version,
			LatestVersion:      latest[1:],
			LatestMinorVersion: latestMinor[1:],
			LatestPatchVersion: latestPatch[1:],
		},
	}, orb.Url.GetOrbID())

	return nil
}
