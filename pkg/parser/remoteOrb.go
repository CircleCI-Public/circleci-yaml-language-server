package parser

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"golang.org/x/mod/semver"
)

type OrbResponse struct {
	OrbVersion OrbQuery
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

var baseUrl = "https://circleci.com"

func GetOrbInfo(orbVersionCode string, cache *utils.Cache) (*ast.CachedOrb, error) {
	// Returning cache if exists
	if !cache.OrbCache.HasOrb(orbVersionCode) {

		orb, err := fetchOrbInfo(orbVersionCode, cache)
		return orb, err
	}

	return cache.OrbCache.GetOrb(orbVersionCode), nil
}

func ParseRemoteOrbs(orbs map[string]ast.Orb, cache *utils.Cache) {
	for _, orb := range orbs {
		if orb.Url.Version != "volatile" && checkIfRemoteOrbAlreadyExistsInFSCache(orb.Url.GetOrbID()) {
			err := addAlreadyExistingRemoteOrbsToFSCache(orb, cache)

			// If no error, we continue
			// Otherwise, we fetch again orb info
			if err == nil {
				continue
			}
		}

		fetchOrbInfo(orb.Url.GetOrbID(), cache)
	}
}

func fetchOrbInfo(orbVersionCode string, cache *utils.Cache) (*ast.CachedOrb, error) {
	orbQuery, err := GetRemoteOrb(orbVersionCode, cache.TokenCache.GetToken(), cache.SelfHostedUrl.GetSelfHostedUrl())

	if err != nil {
		return &ast.CachedOrb{}, err
	}

	parsedOrbSource, err := ParseContent([]byte(orbQuery.Source))

	if err != nil {
		return &ast.CachedOrb{}, err
	}

	filePath, err := writeRemoteOrbSourceInFSCache(orbVersionCode, orbQuery.Source)

	if err != nil {
		return &ast.CachedOrb{}, err
	}

	latest, latestMinor, latestPatch := GetVersionInfo(
		orbQuery.Orb.Versions,
		"v"+orbQuery.Version,
	)

	orb := &ast.CachedOrb{
		ID:                 orbQuery.Id,
		Version:            orbQuery.Version,
		Source:             orbQuery.Source,
		Commands:           parsedOrbSource.Commands,
		Jobs:               parsedOrbSource.Jobs,
		Executors:          parsedOrbSource.Executors,
		Description:        parsedOrbSource.Description,
		FilePath:           filePath,
		LatestVersion:      latest[1:],
		LatestMinorVersion: latestMinor[1:],
		LatestPatchVersion: latestPatch[1:],
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

func GetRemoteOrb(orbId string, token string, selfHostedUrl string) (OrbQuery, error) {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          10,
			TLSHandshakeTimeout:   10 * time.Second,
		},
	}
	url := baseUrl
	if selfHostedUrl != "" {
		url = selfHostedUrl
	}
	client := utils.NewClient(httpClient, url, "graphql-unstable", token, false)
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
	request.Var("orbVersionRef", orbId)

	var response OrbResponse
	err := client.Run(request, &response)

	if response.OrbVersion.Id == "" {
		return response.OrbVersion, fmt.Errorf("could not find orb %s", orbId)
	}

	return response.OrbVersion, err
}

func GetOrbVersions(orbId string, token string, selfHostedUrl string) ([]struct{ Version string }, error) {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          10,
			TLSHandshakeTimeout:   10 * time.Second,
		},
	}

	url := baseUrl
	if selfHostedUrl != "" {
		url = selfHostedUrl
	}

	client := utils.NewClient(httpClient, url, "graphql-unstable", token, false)
	query := `query($orbVersionRef: String!) {
		orbVersion(orbVersionRef: $orbVersionRef) {
			version
		}
	}`

	request := utils.NewRequest(query)
	request.SetToken(client.Token)
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
		fmt.Println("Writing remote orb source in cache:", filePath)

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

func addAlreadyExistingRemoteOrbsToFSCache(orb ast.Orb, cache *utils.Cache) error {
	filePath := utils.GetOrbCacheFSPath(orb.Url.GetOrbID())

	source, err := os.ReadFile(filePath)

	if err != nil {
		return err
	}

	parsedOrbSource, err := ParseContent(source)

	if err != nil {
		return err
	}

	versions, err := GetOrbVersions(orb.Url.GetOrbID(), cache.TokenCache.GetToken(), cache.SelfHostedUrl.GetSelfHostedUrl())

	if err != nil {
		return nil
	}

	latest, latestMinor, latestPatch := GetVersionInfo(versions, "v"+orb.Url.Version)

	cache.OrbCache.SetOrb(&ast.CachedOrb{
		ID:                 orb.Url.GetOrbID(),
		Version:            orb.Url.Version,
		Source:             string(source),
		Commands:           parsedOrbSource.Commands,
		Jobs:               parsedOrbSource.Jobs,
		Executors:          parsedOrbSource.Executors,
		Description:        parsedOrbSource.Description,
		FilePath:           filePath,
		LatestVersion:      latest[1:],
		LatestMinorVersion: latestMinor[1:],
		LatestPatchVersion: latestPatch[1:],
	}, orb.Url.GetOrbID())

	return nil
}
