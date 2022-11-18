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
	tag := getTag()
	fmt.Println(tag)
}

func getTag() string {
	// CIRCLE_TAG is a built-in environment variable documented here: https://circleci.com/docs/variables/
	tag, hasTag := os.LookupEnv("CIRCLE_TAG")
	if hasTag {
		return tag
	}
	return getNextPrereleaseTag()
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
	resp, err := http.Get("https://api.github.com/repos/CircleCI-Public/circleci-yaml-language-server/releases")
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	versions := []GithubRelease{}
	err = json.Unmarshal(body, &versions)
	if err != nil {
		panic(err)
	}
	if len(versions) == 0 {
		panic(fmt.Errorf("Did not find previous versions"))
	}

	return versions[0].TagName
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
