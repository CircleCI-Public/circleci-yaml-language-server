package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/Masterminds/semver"
)

func main() {
	tag := getNextPrereleaseTag()
	fmt.Println(tag)
}

func getNextPrereleaseTag() string {
	latestVersion := getLatestVersion()
	newVersion := incrementVersion(latestVersion)
	return newVersion
}

type GithubRelease struct {
	TagName string `json:"tag_name"`
}

func getLatestVersion() string {
	req, err := http.NewRequest("GET", "https://api.github.com/repos/CircleCI-Public/circleci-yaml-language-server/releases/latest", nil)
	if err != nil {
		panic(err)
	}

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		panic(fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body)))
	}

	var release GithubRelease
	err = json.Unmarshal(body, &release)
	if err != nil {
		panic(err)
	}
	if release.TagName == "" {
		panic(fmt.Errorf("Did not find previous versions"))
	}

	return release.TagName
}

func incrementVersion(tagName string) string {
	v, err := semver.NewVersion(tagName)
	if err != nil {
		panic(err)
	}
	prerelease := v.Prerelease()
	if prerelease == "" {
		newVersion, err := v.IncPatch().SetPrerelease("pre.1")
		if err != nil {
			panic(err)
		}
		return newVersion.String()
	}

	prereleaseReg := regexp.MustCompile("^pre\\.(\\d+)")
	subMatches := prereleaseReg.FindAllStringSubmatch(prerelease, 1)
	if len(subMatches) != 1 {
		panic(fmt.Errorf("Invalid metadata format"))
	}

	prereleaseNumber, err := strconv.Atoi(subMatches[0][1])
	if err != nil {
		panic(err)
	}

	newPrerelease := fmt.Sprintf("pre.%d", prereleaseNumber+1)
	newVersion, err := v.SetPrerelease(newPrerelease)
	if err != nil {
		panic(err)
	}

	return newVersion.String()
}
